package web

import (
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"

	"machine-simulator-2/internal/erp"
	"machine-simulator-2/internal/line"
	"machine-simulator-2/internal/machine"
	"machine-simulator-2/internal/recipe"
	"machine-simulator-2/internal/simulation"
	"machine-simulator-2/internal/storage"
)

//go:embed templates/*.html static/*
var content embed.FS

// PortLookup returns the OPC-UA port for a machine (0 if not found).
type PortLookup func(machineID string) int

type Web struct {
	templates  *template.Template
	engine     *simulation.Engine
	lineMgr    *line.Manager
	orders     *storage.OrderRepository
	erp        *erp.ERP
	catalog    *recipe.Catalog
	opcuaPort  PortLookup
}

func New(engine *simulation.Engine, lineMgr *line.Manager, orders *storage.OrderRepository, erp *erp.ERP, catalog *recipe.Catalog) *Web {
	tmpl := template.Must(template.ParseFS(content, "templates/*.html"))
	return &Web{
		templates: tmpl,
		engine:    engine,
		lineMgr:   lineMgr,
		orders:    orders,
		erp:       erp,
		catalog:   catalog,
	}
}

// SetOPCUAPortLookup sets the function to resolve OPC-UA ports for machines.
func (w *Web) SetOPCUAPortLookup(fn PortLookup) {
	w.opcuaPort = fn
}

func (w *Web) Routes() chi.Router {
	r := chi.NewRouter()

	// Static files
	r.Handle("/static/*", http.FileServer(http.FS(content)))

	// Pages
	r.Get("/", w.handleDashboard)
	r.Get("/lines/{id}", w.handleLineDetail)
	r.Get("/machines/{id}", w.handleMachineDetail)
	r.Get("/orders", w.handleOrders)
	r.Get("/simulation", w.handleSimulation)

	return r
}

func (w *Web) handleDashboard(wr http.ResponseWriter, r *http.Request) {
	lines := w.lineMgr.GetAllLines()
	machines := w.engine.GetAllMachines()

	data := map[string]interface{}{
		"Title":    "Dashboard",
		"Lines":    lines,
		"Machines": machines,
		"Stats": map[string]int{
			"LineCount":    len(lines),
			"MachineCount": len(machines),
		},
	}
	w.templates.ExecuteTemplate(wr, "layout.html", data)
}

func (w *Web) handleLineDetail(wr http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	l := w.lineMgr.GetLine(id)
	if l == nil {
		http.Error(wr, "line not found", http.StatusNotFound)
		return
	}

	machineStates := make([]map[string]interface{}, 0)
	for _, mid := range l.MachineIDs {
		m := w.engine.GetMachine(mid)
		if m != nil {
			orderID, _ := m.GetCurrentOrder()
			port := 0
			if w.opcuaPort != nil {
				port = w.opcuaPort(mid)
			}
			machineStates = append(machineStates, map[string]interface{}{
				"ID":        m.ID(),
				"Type":      m.Type(),
				"Role":      m.Role(),
				"State":     string(m.GetState()),
				"Order":     orderID,
				"OPCUAPort": port,
			})
		}
	}

	buffers := make([]map[string]interface{}, 0)
	for _, b := range l.OrderedBuffers() {
		count := b.Count()
		cap := b.MaxCapacity
		pct := 0
		if cap > 0 {
			pct = count * 100 / cap
		}
		level := "normal"
		if cap > 0 && count*100 > cap*90 {
			level = "critical"
		} else if cap > 0 && count*100 > cap*70 {
			level = "warning"
		}
		buffers = append(buffers, map[string]interface{}{
			"ID":       b.ID,
			"Count":    count,
			"Capacity": cap,
			"Pct":      pct,
			"Level":    level,
		})
	}

	data := map[string]interface{}{
		"Title":    "Line: " + id,
		"Line":     l,
		"Machines": machineStates,
		"Buffers":  buffers,
		"Page":     "line-detail",
	}
	w.templates.ExecuteTemplate(wr, "layout.html", data)
}

func (w *Web) handleMachineDetail(wr http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	m := w.engine.GetMachine(id)
	if m == nil {
		http.Error(wr, "machine not found", http.StatusNotFound)
		return
	}

	orderID, opID := m.GetCurrentOrder()

	// Build sorted tag list with definitions
	tags := m.GetTags()
	defs := machine.AllTagDefinitions(m)
	tagRows := buildTagRows(defs, tags)

	port := 0
	if w.opcuaPort != nil {
		port = w.opcuaPort(id)
	}

	data := map[string]interface{}{
		"Title":       "Machine: " + id,
		"Machine":     m,
		"MachineID":   m.ID(),
		"MachineType": m.Type(),
		"Role":        m.Role(),
		"State":       string(m.GetState()),
		"TagRows":     tagRows,
		"OPCUAPort":   port,
		"OrderID":     orderID,
		"OperationID": opID,
		"Page":        "machine-detail",
	}
	w.templates.ExecuteTemplate(wr, "layout.html", data)
}

func (w *Web) handleOrders(wr http.ResponseWriter, r *http.Request) {
	orders, _ := w.orders.List(storage.OrderFilter{Limit: 100})
	lines := w.lineMgr.GetAllLines()
	products := w.catalog.GetAllProducts()

	data := map[string]interface{}{
		"Title":    "Orders",
		"Orders":   orders,
		"Lines":    lines,
		"Products": products,
		"Page":     "orders",
	}
	w.templates.ExecuteTemplate(wr, "layout.html", data)
}

func (w *Web) handleSimulation(wr http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":        "Simulation",
		"TimeScale":    w.engine.GetTimeScale(),
		"RandomFactor": w.engine.GetRandomFactor(),
		"MachineCount": w.engine.MachineCount(),
		"LineCount":    len(w.lineMgr.GetAllLines()),
		"Page":         "simulation",
	}
	w.templates.ExecuteTemplate(wr, "layout.html", data)
}

// TagRow is a single row in the machine detail tags table.
type TagRow struct {
	NodeID      int
	NodeIDStr   string // OPC-UA format: ns=1;i=N
	Name        string
	DisplayName string
	Value       string
	Unit        string
}

// buildTagRows creates sorted tag rows from tag definitions and values.
func buildTagRows(defs []machine.TagDefinition, tags map[string]interface{}) []TagRow {
	defMap := make(map[string]machine.TagDefinition, len(defs))
	for _, d := range defs {
		defMap[d.Name] = d
	}

	rows := make([]TagRow, 0, len(tags))
	for name, val := range tags {
		d := defMap[name]
		row := TagRow{
			NodeID:      d.NodeID,
			NodeIDStr:   fmt.Sprintf("ns=1;i=%d", d.NodeID),
			Name:        name,
			DisplayName: d.DisplayName,
			Unit:        d.Unit,
		}
		if row.DisplayName == "" {
			row.DisplayName = name
		}
		switch v := val.(type) {
		case float64:
			row.Value = fmt.Sprintf("%.2f", v)
		case float32:
			row.Value = fmt.Sprintf("%.2f", v)
		default:
			row.Value = fmt.Sprintf("%v", val)
		}
		rows = append(rows, row)
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].NodeID < rows[j].NodeID
	})
	return rows
}
