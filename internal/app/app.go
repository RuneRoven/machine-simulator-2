package app

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"machine-simulator-2/internal/api"
	"machine-simulator-2/internal/config"
	"machine-simulator-2/internal/erp"
	"machine-simulator-2/internal/event"
	"machine-simulator-2/internal/line"
	"machine-simulator-2/internal/machine"
	"machine-simulator-2/internal/mes"
	"machine-simulator-2/internal/protocol"
	"machine-simulator-2/internal/protocol/modbus"
	opcuaAdapter "machine-simulator-2/internal/protocol/opcua"
	"machine-simulator-2/internal/recipe"
	"machine-simulator-2/internal/simulation"
	"machine-simulator-2/internal/storage"
	"machine-simulator-2/internal/web"
)

// App is the composition root for the simulator.
type App struct {
	cfg     *config.Config
	bus     *event.Bus
	db      *sql.DB
	engine  *simulation.Engine
	lineMgr *line.Manager
	catalog *recipe.Catalog
	orders  *storage.OrderRepository
	ops     *storage.OperationRepository
	erp     *erp.ERP
	mes     *mes.MES
	adapters []protocol.Adapter
	httpSrv  *http.Server
	wsHub    *api.Hub
}

// New creates and wires all components.
func New(cfg *config.Config) (*App, error) {
	app := &App{cfg: cfg}

	// Event bus
	app.bus = event.New()

	// Database
	db, err := storage.NewDB("simulator.db")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	app.db = db
	app.orders = storage.NewOrderRepository(db)
	app.ops = storage.NewOperationRepository(db)

	// Recipe catalog
	app.catalog = recipe.NewCatalog()
	app.catalog.PopulateFromConfig(cfg.Templates)

	// Simulation engine
	app.engine = simulation.New(app.bus, cfg.Global.TimeScale, cfg.Global.RandomFactor)

	// Line manager
	app.lineMgr = line.NewManager(app.bus)

	// Instantiate lines from config
	for _, li := range cfg.Lines {
		// Create machines for this line instance
		lineMachines := make(map[string]machine.Machine)
		for _, mc := range li.Machines {
			m, err := machine.Create(mc.Type, mc)
			if err != nil {
				log.Warn().Err(err).Str("type", mc.Type).Str("id", mc.ID).Msg("Failed to create machine, using base type")
				m = machine.NewBaseMachine(mc)
			}
			app.engine.RegisterMachine(m)
			lineMachines[mc.ID] = m
		}

		// Create line
		l := line.InstantiateFromConfig(li)
		app.lineMgr.AddLine(l, lineMachines)
	}

	// ERP
	erpCfg := erp.Config{
		Mode:                cfg.Global.ERP.Mode,
		AutoOrderIntervalMS: cfg.Global.ERP.AutoOrderIntervalMS,
		MaxBacklog:          cfg.Global.ERP.MaxBacklog,
		TimeScale:           cfg.Global.TimeScale,
	}
	app.erp = erp.New(erpCfg, app.orders, app.bus)

	// Register products from all active lines
	for _, li := range cfg.Lines {
		recipes := app.catalog.GetRecipes(li.TemplateName)
		for _, r := range recipes {
			app.erp.RegisterProduct(erp.ProductRegistration{
				LineInstanceID: li.InstanceID,
				LineTemplate:   li.TemplateName,
				ProductID:      r.ProductID,
				PlannedQtyMin:  r.PlannedQtyRange[0],
				PlannedQtyMax:  r.PlannedQtyRange[1],
			})
		}
	}

	// MES
	mesCfg := mes.Config{
		DispatchIntervalMS: cfg.Global.MES.DispatchIntervalMS,
		DispatchStrategy:   cfg.Global.MES.DispatchStrategy,
		TimeScale:          cfg.Global.TimeScale,
	}
	app.mes = mes.New(mesCfg, app.orders, app.ops, app.bus, app.catalog, app.lineMgr, app.engine)

	// Register lines with MES
	for _, li := range cfg.Lines {
		app.mes.RegisterLine(li)
	}

	// Protocol adapters
	if cfg.Adapters.OPCUA.Enabled {
		opcua := opcuaAdapter.New(cfg.Adapters.OPCUA)
		app.adapters = append(app.adapters, opcua)
	}
	if cfg.Adapters.Modbus.Enabled {
		mb := modbus.New(cfg.Adapters.Modbus)
		app.adapters = append(app.adapters, mb)
	}

	// Register machines with protocol adapters
	for _, adapter := range app.adapters {
		for _, m := range app.engine.GetAllMachines() {
			adapter.RegisterMachine(m, protocol.AdapterMachineConfig{})
		}
	}

	// WebSocket hub
	app.wsHub = api.NewHub(app.bus)

	// HTTP server (API + Web)
	apiServer := api.NewServer(app.engine, app.lineMgr, app.orders, app.ops, app.erp, app.mes, app.catalog, app.wsHub)

	// Wire Modbus register map endpoint if enabled
	if cfg.Adapters.Modbus.Enabled {
		for _, adapter := range app.adapters {
			if adapter.Name() == "modbus" {
				if mb, ok := adapter.(*modbus.Adapter); ok {
					apiServer.SetModbusRegisterMapFn(func() interface{} {
						return mb.GetRegisterMap()
					})
				}
			}
		}
	}

	webUI := web.New(app.engine, app.lineMgr, app.orders, app.erp, app.catalog)

	// Wire OPC-UA port lookup if enabled
	if cfg.Adapters.OPCUA.Enabled {
		for _, adapter := range app.adapters {
			if adapter.Name() == "opcua" {
				if oa, ok := adapter.(*opcuaAdapter.Adapter); ok {
					webUI.SetOPCUAPortLookup(oa.GetPort)
				}
			}
		}
	}

	// Compose routes: API router handles /api, /ws, /health; web UI handles everything else
	apiRouter := apiServer.Router()
	mux := http.NewServeMux()
	mux.Handle("/api/", apiRouter)
	mux.Handle("/ws", apiRouter)
	mux.Handle("/health", apiRouter)
	mux.Handle("/", webUI.Routes())

	addr := fmt.Sprintf("%s:%d", cfg.Global.API.Host, cfg.Global.API.Port)
	app.httpSrv = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return app, nil
}

// Run starts all components and blocks until ctx is cancelled.
func (a *App) Run(ctx context.Context) error {
	log.Info().
		Int("lines", len(a.cfg.Lines)).
		Int("machines", a.engine.MachineCount()).
		Int("adapters", len(a.adapters)).
		Msg("Starting simulator")

	// Start engine (machines)
	a.engine.Start(ctx)

	// Start protocol adapters
	for _, adapter := range a.adapters {
		if err := adapter.Start(ctx); err != nil {
			log.Error().Err(err).Str("adapter", adapter.Name()).Msg("Failed to start adapter")
		}
	}

	// Start ERP
	a.erp.Start(ctx)

	// Start MES
	a.mes.Start(ctx)

	// Start HTTP server
	go func() {
		log.Info().Str("addr", a.httpSrv.Addr).Msg("HTTP server listening")
		if err := a.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("HTTP server error")
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()
	log.Info().Msg("Shutting down...")

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop HTTP server
	if err := a.httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("HTTP server shutdown error")
	}

	// Stop protocol adapters
	for _, adapter := range a.adapters {
		if err := adapter.Stop(); err != nil {
			log.Error().Err(err).Str("adapter", adapter.Name()).Msg("Adapter stop error")
		}
	}

	// Stop engine (machines)
	a.engine.Stop()

	// Close database
	a.db.Close()

	log.Info().Msg("Simulator stopped")
	return nil
}
