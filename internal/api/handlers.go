package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"machine-simulator-2/internal/erp"
	"machine-simulator-2/internal/line"
	"machine-simulator-2/internal/machine"
	"machine-simulator-2/internal/mes"
	"machine-simulator-2/internal/recipe"
	"machine-simulator-2/internal/simulation"
	"machine-simulator-2/internal/storage"
)

// Server holds all dependencies for the REST API.
type Server struct {
	engine  *simulation.Engine
	lineMgr *line.Manager
	orders  *storage.OrderRepository
	ops     *storage.OperationRepository
	erp     *erp.ERP
	mes     *mes.MES
	catalog *recipe.Catalog
	wsHub   *Hub

	modbusRegMapFn func() interface{}
}

// SetModbusRegisterMapFn sets the function to retrieve the Modbus register map.
func (s *Server) SetModbusRegisterMapFn(fn func() interface{}) {
	s.modbusRegMapFn = fn
}

// NewServer creates a new API server.
func NewServer(engine *simulation.Engine, lineMgr *line.Manager, orders *storage.OrderRepository,
	ops *storage.OperationRepository, erp *erp.ERP, mes *mes.MES, catalog *recipe.Catalog, wsHub *Hub) *Server {
	return &Server{
		engine:  engine,
		lineMgr: lineMgr,
		orders:  orders,
		ops:     ops,
		erp:     erp,
		mes:     mes,
		catalog: catalog,
		wsHub:   wsHub,
	}
}

// Router returns the configured chi router.
func (s *Server) Router() http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Health
	r.Get("/health", s.handleHealth)

	// API routes
	r.Route("/api", func(r chi.Router) {
		// Lines
		r.Get("/lines", s.handleGetLines)
		r.Get("/lines/{id}", s.handleGetLine)
		r.Get("/lines/{id}/buffers", s.handleGetLineBuffers)
		r.Post("/lines/{id}/start", s.handleStartLine)
		r.Post("/lines/{id}/stop", s.handleStopLine)

		// Machines
		r.Get("/machines", s.handleGetMachines)
		r.Get("/machines/{id}", s.handleGetMachine)
		r.Get("/machines/{id}/tags", s.handleGetMachineTags)
		r.Post("/machines/{id}/command", s.handleMachineCommand)

		// Orders
		r.Get("/orders", s.handleGetOrders)
		r.Get("/orders/{id}", s.handleGetOrder)
		r.Post("/orders", s.handleCreateOrder)
		r.Put("/orders/{id}/status", s.handleUpdateOrderStatus)
		r.Delete("/orders/{id}", s.handleCancelOrder)

		// Recipes
		r.Get("/recipes", s.handleGetRecipes)
		r.Get("/lines/{id}/recipes", s.handleGetLineRecipes)

		// Simulation
		r.Get("/simulation", s.handleGetSimulation)
		r.Put("/simulation/timescale", s.handleSetTimescale)
		r.Put("/simulation/random-factor", s.handleSetRandomFactor)

		// Modbus
		r.Get("/modbus/register-map", s.handleGetModbusRegisterMap)

		// Settings
		r.Get("/settings/machines", s.handleGetMachineSettings)
		r.Put("/settings/machines/{id}", s.handleUpdateMachineSettings)
		r.Get("/settings/buffers", s.handleGetBufferSettings)
		r.Put("/settings/buffers/{id}", s.handleUpdateBufferSettings)
	})

	// WebSocket
	if s.wsHub != nil {
		r.Get("/ws", s.wsHub.HandleWebSocket)
	}

	return r
}

// Health
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Lines
func (s *Server) handleGetLines(w http.ResponseWriter, r *http.Request) {
	lines := s.lineMgr.GetAllLines()
	result := make([]lineResponse, 0, len(lines))
	for _, l := range lines {
		result = append(result, toLineResponse(l))
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleGetLine(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	l := s.lineMgr.GetLine(id)
	if l == nil {
		writeError(w, http.StatusNotFound, "line not found")
		return
	}
	writeJSON(w, http.StatusOK, toLineResponse(l))
}

func (s *Server) handleGetLineBuffers(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	l := s.lineMgr.GetLine(id)
	if l == nil {
		writeError(w, http.StatusNotFound, "line not found")
		return
	}
	buffers := make([]bufferResponse, 0, len(l.Buffers))
	for _, b := range l.Buffers {
		buffers = append(buffers, toBufferResponse(b))
	}
	writeJSON(w, http.StatusOK, buffers)
}

func (s *Server) handleStartLine(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	l := s.lineMgr.GetLine(id)
	if l == nil {
		writeError(w, http.StatusNotFound, "line not found")
		return
	}
	for _, machineID := range l.MachineIDs {
		m := s.engine.GetMachine(machineID)
		if m != nil && m.GetState() == machine.StateIdle {
			m.SendCommand(machine.CommandStart)
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "started"})
}

func (s *Server) handleStopLine(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	l := s.lineMgr.GetLine(id)
	if l == nil {
		writeError(w, http.StatusNotFound, "line not found")
		return
	}
	for _, machineID := range l.MachineIDs {
		m := s.engine.GetMachine(machineID)
		if m != nil && m.GetState() == machine.StateRunning {
			m.SendCommand(machine.CommandStop)
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

// Machines
func (s *Server) handleGetMachines(w http.ResponseWriter, r *http.Request) {
	machines := s.engine.GetAllMachines()
	result := make([]machineResponse, 0, len(machines))
	for _, m := range machines {
		result = append(result, toMachineResponse(m))
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleGetMachine(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	m := s.engine.GetMachine(id)
	if m == nil {
		writeError(w, http.StatusNotFound, "machine not found")
		return
	}
	writeJSON(w, http.StatusOK, toMachineResponse(m))
}

func (s *Server) handleGetMachineTags(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	m := s.engine.GetMachine(id)
	if m == nil {
		writeError(w, http.StatusNotFound, "machine not found")
		return
	}
	writeJSON(w, http.StatusOK, m.GetTags())
}

func (s *Server) handleMachineCommand(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	m := s.engine.GetMachine(id)
	if m == nil {
		writeError(w, http.StatusNotFound, "machine not found")
		return
	}

	var req struct {
		Command string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cmd := machine.Command(req.Command)
	if err := m.SendCommand(cmd); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Orders
func (s *Server) handleGetOrders(w http.ResponseWriter, r *http.Request) {
	filter := storage.OrderFilter{
		Status:         r.URL.Query().Get("status"),
		LineInstanceID: r.URL.Query().Get("line_instance_id"),
		ProductID:      r.URL.Query().Get("product_id"),
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		filter.Limit, _ = strconv.Atoi(l)
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		filter.Offset, _ = strconv.Atoi(o)
	}
	orders, err := s.orders.List(filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, orders)
}

func (s *Server) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	order, err := s.orders.GetByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, order)
}

func (s *Server) handleCreateOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		LineInstanceID string `json:"line_instance_id"`
		LineTemplate   string `json:"line_template"`
		ProductID      string `json:"product_id"`
		PlannedQty     int    `json:"planned_qty"`
		Priority       int    `json:"priority"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Priority == 0 {
		req.Priority = 50
	}

	order, err := s.erp.CreateManualOrder(req.LineInstanceID, req.LineTemplate, req.ProductID, req.PlannedQty, req.Priority)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, order)
}

func (s *Server) handleUpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	switch req.Status {
	case "RELEASED":
		if err := s.erp.ReleaseOrder(id); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
	case "CLOSED":
		if err := s.erp.CloseOrder(id); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
	case "CANCELLED":
		if err := s.mes.CancelOrder(id); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
	default:
		writeError(w, http.StatusBadRequest, "invalid status: "+req.Status)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": req.Status})
}

func (s *Server) handleCancelOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.mes.CancelOrder(id); err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// Recipes
func (s *Server) handleGetRecipes(w http.ResponseWriter, r *http.Request) {
	products := s.catalog.GetAllProducts()
	writeJSON(w, http.StatusOK, products)
}

func (s *Server) handleGetLineRecipes(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	l := s.lineMgr.GetLine(id)
	if l == nil {
		writeError(w, http.StatusNotFound, "line not found")
		return
	}
	recipes := s.catalog.GetRecipes(l.TemplateName)
	writeJSON(w, http.StatusOK, recipes)
}

// Simulation
func (s *Server) handleGetSimulation(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"time_scale":    s.engine.GetTimeScale(),
		"random_factor": s.engine.GetRandomFactor(),
		"machine_count": s.engine.MachineCount(),
		"line_count":    len(s.lineMgr.GetAllLines()),
	})
}

func (s *Server) handleSetTimescale(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TimeScale float64 `json:"time_scale"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.TimeScale < 0.1 || req.TimeScale > 100 {
		writeError(w, http.StatusBadRequest, "time_scale must be between 0.1 and 100")
		return
	}
	s.engine.SetTimeScale(req.TimeScale)
	s.erp.SetTimeScale(req.TimeScale)
	s.mes.SetTimeScale(req.TimeScale)
	writeJSON(w, http.StatusOK, map[string]float64{"time_scale": req.TimeScale})
}

func (s *Server) handleSetRandomFactor(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RandomFactor float64 `json:"random_factor"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.RandomFactor < 0 || req.RandomFactor > 1 {
		writeError(w, http.StatusBadRequest, "random_factor must be between 0 and 1")
		return
	}
	s.engine.SetRandomFactor(req.RandomFactor)
	writeJSON(w, http.StatusOK, map[string]float64{"random_factor": req.RandomFactor})
}

// Modbus
func (s *Server) handleGetModbusRegisterMap(w http.ResponseWriter, r *http.Request) {
	if s.modbusRegMapFn == nil {
		writeError(w, http.StatusNotFound, "modbus adapter not enabled")
		return
	}
	writeJSON(w, http.StatusOK, s.modbusRegMapFn())
}

// Settings
func (s *Server) handleGetMachineSettings(w http.ResponseWriter, r *http.Request) {
	machines := s.engine.GetAllMachines()
	result := make([]map[string]interface{}, 0, len(machines))
	for _, m := range machines {
		settings := m.GetSettings()
		result = append(result, map[string]interface{}{
			"id":                    m.ID(),
			"type":                  m.Type(),
			"line_instance_id":      m.LineInstanceID(),
			"cycle_time_ms":         settings.CycleTimeMS,
			"scrap_rate":            settings.ScrapRate,
			"breakdown_probability": settings.BreakdownProbability,
		})
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleUpdateMachineSettings(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	m := s.engine.GetMachine(id)
	if m == nil {
		writeError(w, http.StatusNotFound, "machine not found")
		return
	}

	var req struct {
		CycleTimeMS          *int     `json:"cycle_time_ms"`
		ScrapRate            *float64 `json:"scrap_rate"`
		BreakdownProbability *float64 `json:"breakdown_probability"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cfg := m.GetSettings()
	if req.CycleTimeMS != nil {
		cfg.CycleTimeMS = *req.CycleTimeMS
	}
	if req.ScrapRate != nil {
		cfg.ScrapRate = *req.ScrapRate
	}
	if req.BreakdownProbability != nil {
		cfg.BreakdownProbability = *req.BreakdownProbability
	}
	m.UpdateConfig(cfg)
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleGetBufferSettings(w http.ResponseWriter, r *http.Request) {
	buffers := s.lineMgr.GetAllBuffers()
	result := make([]map[string]interface{}, 0, len(buffers))
	for _, b := range buffers {
		result = append(result, map[string]interface{}{
			"id":                    b.ID,
			"line":                  b.LineName,
			"upstream_machine_id":   b.UpstreamMachineID,
			"downstream_machine_id": b.DownstreamMachineID,
			"capacity":              b.MaxCapacity,
			"count":                 b.Count(),
		})
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleUpdateBufferSettings(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	buf := s.lineMgr.GetBuffer(id)
	if buf == nil {
		writeError(w, http.StatusNotFound, "buffer not found")
		return
	}

	var req struct {
		Capacity int `json:"capacity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Capacity < 1 {
		writeError(w, http.StatusBadRequest, "capacity must be >= 1")
		return
	}
	buf.SetCapacity(req.Capacity)
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

// Response types
type lineResponse struct {
	InstanceID   string       `json:"instance_id"`
	TemplateName string       `json:"template_name"`
	Domain       string       `json:"domain"`
	MachineIDs   []string     `json:"machine_ids"`
	Recipes      []recipeRef  `json:"recipes"`
}

type recipeRef struct {
	ProductID   string `json:"product_id"`
	Description string `json:"description"`
}

type bufferResponse struct {
	ID                  string `json:"id"`
	UpstreamMachineID   string `json:"upstream_machine_id"`
	DownstreamMachineID string `json:"downstream_machine_id"`
	Capacity            int    `json:"capacity"`
	Count               int    `json:"count"`
	TotalThroughput     int64  `json:"total_throughput"`
}

type machineResponse struct {
	ID             string                 `json:"id"`
	Type           string                 `json:"type"`
	Role           string                 `json:"role"`
	LineInstanceID string                 `json:"line_instance_id"`
	State          string                 `json:"state"`
	OrderID        string                 `json:"order_id,omitempty"`
	OperationID    string                 `json:"operation_id,omitempty"`
	OPCUAPort      int                    `json:"opcua_port,omitempty"`
	Tags           map[string]interface{} `json:"tags"`
}

// OPCUAPortLookup is a function that returns the OPC-UA port for a machine ID.
type OPCUAPortLookup func(machineID string) int

var opcuaPortLookup OPCUAPortLookup

// SetOPCUAPortLookup sets the function used to resolve OPC-UA ports.
func SetOPCUAPortLookup(fn OPCUAPortLookup) {
	opcuaPortLookup = fn
}

func toLineResponse(l *line.Line) lineResponse {
	recipes := make([]recipeRef, 0, len(l.Recipes))
	for _, r := range l.Recipes {
		recipes = append(recipes, recipeRef{ProductID: r.ProductID, Description: r.Description})
	}
	return lineResponse{
		InstanceID:   l.InstanceID,
		TemplateName: l.TemplateName,
		Domain:       l.Domain,
		MachineIDs:   l.MachineIDs,
		Recipes:      recipes,
	}
}

func toBufferResponse(b *line.Buffer) bufferResponse {
	return bufferResponse{
		ID:                  b.ID,
		UpstreamMachineID:   b.UpstreamMachineID,
		DownstreamMachineID: b.DownstreamMachineID,
		Capacity:            b.MaxCapacity,
		Count:               b.Count(),
		TotalThroughput:     b.TotalThroughput(),
	}
}

func toMachineResponse(m machine.Machine) machineResponse {
	orderID, opID := m.GetCurrentOrder()
	resp := machineResponse{
		ID:             m.ID(),
		Type:           m.Type(),
		Role:           m.Role(),
		LineInstanceID: m.LineInstanceID(),
		State:          string(m.GetState()),
		OrderID:        orderID,
		OperationID:    opID,
		Tags:           m.GetTags(),
	}
	if opcuaPortLookup != nil {
		resp.OPCUAPort = opcuaPortLookup(m.ID())
	}
	return resp
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
