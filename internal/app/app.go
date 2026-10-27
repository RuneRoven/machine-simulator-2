package app

import (
	"context"
	"database/sql"
	"fmt"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
	"machine-simulator-2/internal/webhook"
)

// App is the composition root for the simulator.
type App struct {
	cfg         *config.Config
	bus         *event.Bus
	db          *sql.DB
	engine      *simulation.Engine
	lineMgr     *line.Manager
	catalog     *recipe.Catalog
	orders      *storage.OrderRepository
	ops         *storage.OperationRepository
	material    *storage.LineMaterialRepository
	erp         *erp.ERP
	mes         *mes.MES
	adapters    []protocol.Adapter
	lineServers *opcuaAdapter.LineServers
	webhookPub  *webhook.Publisher
	httpSrv     *http.Server
	wsHub       *api.Hub
}

// New creates and wires all components.
func New(cfg *config.Config) (*App, error) {
	app := &App{cfg: cfg}

	// Event bus
	app.bus = event.New()

	// Database
	dbPath := os.Getenv("SIMULATOR_DB_PATH")
	if dbPath == "" {
		dbPath = filepath.Join("data", "simulator.db")
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("creating database directory: %w", err)
	}
	db, err := storage.NewDB(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	app.db = db
	app.orders = storage.NewOrderRepository(db)
	app.ops = storage.NewOperationRepository(db)
	app.material = storage.NewLineMaterialRepository(db)

	// Recipe catalog
	app.catalog = recipe.NewCatalog()
	app.catalog.PopulateFromConfig(cfg.Templates)

	// Simulation engine
	app.engine = simulation.New(app.bus, cfg.Global.TimeScale, cfg.Global.RandomFactor)

	// Line manager
	app.lineMgr = line.NewManager(app.bus)
	app.lineMgr.SetRecipeSource(app.catalog)
	// Book parts before anyone else (MES, webhook) sees the cycle.
	app.lineMgr.SubscribeCycleEvents()

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
	if err := app.restoreMaterial(); err != nil {
		return nil, err
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

	// Line job servers ("line PLCs") for an external MES
	if cfg.Adapters.OPCUA.Enabled && cfg.Global.Orders.External() {
		app.lineServers = opcuaAdapter.NewLineServers(cfg.Adapters.OPCUA, app.lineMgr)
		for _, l := range app.lineMgr.GetAllLines() {
			app.lineServers.Register(l.InstanceID)
		}
	}

	// Webhook publisher. With an external order source the simulator has no
	// orders of its own, so order events are not sent.
	if cfg.Global.Webhook.Enabled {
		eventTypes := cfg.Global.Webhook.EventTypes
		if cfg.Global.Orders.External() {
			eventTypes = withoutOrderEvents(eventTypes)
		}
		whCfg := webhook.Config{
			Enabled:       cfg.Global.Webhook.Enabled,
			TargetURL:     cfg.Global.Webhook.TargetURL,
			EventTypes:    eventTypes,
			TimeoutMS:     cfg.Global.Webhook.TimeoutMS,
			RetryAttempts: cfg.Global.Webhook.RetryAttempts,
			RetryDelayMS:  cfg.Global.Webhook.RetryDelayMS,
		}
		app.webhookPub = webhook.New(whCfg, app.bus)
	}

	// WebSocket hub
	app.wsHub = api.NewHub(app.bus)

	// HTTP server (API + Web)
	apiServer := api.NewServer(app.engine, app.lineMgr, app.orders, app.ops, app.erp, app.mes, app.catalog, app.wsHub)
	apiServer.SetExternalOrders(cfg.Global.Orders.External())

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
					api.SetOPCUAPortLookup(oa.GetPort)
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

	go a.watchStalls(ctx)
	go a.keepMaterial(ctx)

	if a.lineServers != nil {
		a.lineServers.Start(ctx)
	}

	if a.cfg.Global.Orders.External() {
		log.Info().Msg("orders.source=external: built-in ERP and MES are off, lines wait for jobs")
	} else {
		a.erp.Start(ctx)
		a.mes.Start(ctx)
	}

	// Start webhook publisher
	if a.webhookPub != nil {
		a.webhookPub.Start(ctx)
	}

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

	// Keep the line-side material, then close the database
	a.saveMaterial(nil)
	a.db.Close()

	log.Info().Msg("Simulator stopped")
	return nil
}

// watchStalls restarts lines whose job stopped moving parts for longer than
// the deadlock threshold (in simulated time) while no machine is faulted.
func (a *App) watchStalls(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			scale := a.engine.GetTimeScale()
			if scale <= 0 {
				scale = 1
			}
			threshold := time.Duration(float64(a.cfg.Global.DeadlockThresholdMS)/scale) * time.Millisecond
			for _, id := range a.lineMgr.CheckStalls(now, threshold) {
				log.Warn().Str("line", id).Msg("line stalled without a fault; restarted")
			}
		}
	}
}

// restoreMaterial loads the line-side material saved by the last run. Lines
// that no longer exist are skipped.
func (a *App) restoreMaterial() error {
	saved, err := a.material.Load()
	if err != nil {
		return err
	}
	for lineID, stock := range saved {
		if err := a.lineMgr.RestoreMaterial(lineID, stock); err != nil {
			log.Warn().Err(err).Str("line", lineID).Msg("saved line material not restored")
			continue
		}
		log.Info().Str("line", lineID).Int("materials", len(stock)).Msg("line material restored")
	}
	return nil
}

// keepMaterial saves the line-side material every few seconds, so a restart
// loses at most the last few parts' consumption.
func (a *App) keepMaterial(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	last := map[string]map[string]float64{}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.saveMaterial(last)
		}
	}
}

// saveMaterial writes each line's material that changed since last (all when
// last is nil) and records what it wrote in last.
func (a *App) saveMaterial(last map[string]map[string]float64) {
	for _, l := range a.lineMgr.GetAllLines() {
		stock, err := a.lineMgr.Material(l.InstanceID)
		if err != nil {
			continue
		}
		if last != nil && maps.Equal(last[l.InstanceID], stock) {
			continue
		}
		if err := a.material.Save(l.InstanceID, stock); err != nil {
			log.Error().Err(err).Str("line", l.InstanceID).Msg("saving line material")
			continue
		}
		if last != nil {
			last[l.InstanceID] = stock
		}
	}
}

func withoutOrderEvents(types []string) []string {
	out := make([]string, 0, len(types))
	for _, t := range types {
		if !strings.HasPrefix(t, "order.") {
			out = append(out, t)
		}
	}
	return out
}
