package erp

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"machine-simulator-2/internal/event"
	"machine-simulator-2/internal/storage"
)

// ProductRegistration ties a product to a line instance.
type ProductRegistration struct {
	LineInstanceID string
	LineTemplate   string
	ProductID      string
	PlannedQtyMin  int
	PlannedQtyMax  int
}

// Config holds ERP simulator settings.
type Config struct {
	Mode                string
	AutoOrderIntervalMS int
	MaxBacklog          int
	TimeScale           float64
}

// ERP simulates enterprise resource planning - order generation and lifecycle.
type ERP struct {
	mu sync.RWMutex

	cfg      Config
	orders   *storage.OrderRepository
	bus      *event.Bus
	products []ProductRegistration
	rng      *rand.Rand
}

// New creates a new ERP simulator.
func New(cfg Config, orders *storage.OrderRepository, bus *event.Bus) *ERP {
	return &ERP{
		cfg:    cfg,
		orders: orders,
		bus:    bus,
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// RegisterProduct adds a product that ERP can generate orders for.
func (e *ERP) RegisterProduct(reg ProductRegistration) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.products = append(e.products, reg)
}

// SetTimeScale updates the time scale for interval adjustment.
func (e *ERP) SetTimeScale(scale float64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cfg.TimeScale = scale
}

// Start begins the ERP auto-order loop if in auto mode.
func (e *ERP) Start(ctx context.Context) {
	if e.cfg.Mode != "auto" {
		log.Info().Msg("ERP in manual mode — no auto-order generation")
		return
	}

	go e.autoOrderLoop(ctx)
}

func (e *ERP) autoOrderLoop(ctx context.Context) {
	for {
		e.mu.RLock()
		intervalMS := e.cfg.AutoOrderIntervalMS
		scale := e.cfg.TimeScale
		e.mu.RUnlock()

		if scale <= 0 {
			scale = 1.0
		}
		interval := time.Duration(float64(intervalMS)/scale) * time.Millisecond

		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
			e.generateAutoOrder()
		}
	}
}

func (e *ERP) generateAutoOrder() {
	e.mu.RLock()
	if len(e.products) == 0 {
		e.mu.RUnlock()
		return
	}

	// Shuffle to pick random line/product
	idx := e.rng.Intn(len(e.products))
	prod := e.products[idx]
	maxBacklog := e.cfg.MaxBacklog
	e.mu.RUnlock()

	// Check backlog
	if maxBacklog > 0 {
		pending, err := e.orders.CountByStatus(prod.LineInstanceID, "CREATED")
		if err != nil {
			log.Error().Err(err).Msg("ERP: failed to count pending orders")
			return
		}
		released, _ := e.orders.CountByStatus(prod.LineInstanceID, "RELEASED")
		inProgress, _ := e.orders.CountByStatus(prod.LineInstanceID, "IN_PROGRESS")
		if pending+released+inProgress >= maxBacklog {
			log.Debug().Str("line", prod.LineInstanceID).Msg("ERP: backlog full, skipping")
			return
		}
	}

	// Generate order
	qty := prod.PlannedQtyMin
	if prod.PlannedQtyMax > prod.PlannedQtyMin {
		qty += e.rng.Intn(prod.PlannedQtyMax - prod.PlannedQtyMin + 1)
	}

	order := &storage.Order{
		ID:             uuid.New().String(),
		ProductID:      prod.ProductID,
		LineInstanceID: prod.LineInstanceID,
		LineTemplate:   prod.LineTemplate,
		Status:         "CREATED",
		PlannedQty:     qty,
		Priority:       50 + e.rng.Intn(50),
		CreatedAt:      time.Now(),
	}

	if err := e.orders.Create(order); err != nil {
		log.Error().Err(err).Msg("ERP: failed to create order")
		return
	}

	log.Info().
		Str("order_id", order.ID).
		Str("product", order.ProductID).
		Str("line", order.LineInstanceID).
		Int("qty", order.PlannedQty).
		Msg("ERP: order created")

	if e.bus != nil {
		e.bus.Publish(event.Event{
			Type:   event.OrderCreated,
			LineID: order.LineInstanceID,
			Data:   order,
		})
	}
}

// CreateManualOrder creates an order from an API request.
func (e *ERP) CreateManualOrder(lineInstanceID, lineTemplate, productID string, plannedQty, priority int) (*storage.Order, error) {
	order := &storage.Order{
		ID:             uuid.New().String(),
		ProductID:      productID,
		LineInstanceID: lineInstanceID,
		LineTemplate:   lineTemplate,
		Status:         "CREATED",
		PlannedQty:     plannedQty,
		Priority:       priority,
		CreatedAt:      time.Now(),
	}

	if err := e.orders.Create(order); err != nil {
		return nil, err
	}

	if e.bus != nil {
		e.bus.Publish(event.Event{
			Type:   event.OrderCreated,
			LineID: order.LineInstanceID,
			Data:   order,
		})
	}

	return order, nil
}

// ReleaseOrder transitions an order from CREATED to RELEASED.
func (e *ERP) ReleaseOrder(orderID string) error {
	if err := e.orders.UpdateStatus(orderID, "RELEASED"); err != nil {
		return err
	}

	order, err := e.orders.GetByID(orderID)
	if err != nil {
		return err
	}

	if e.bus != nil {
		e.bus.Publish(event.Event{
			Type:   event.OrderReleased,
			LineID: order.LineInstanceID,
			Data:   order,
		})
	}

	return nil
}

// CloseOrder transitions an order to CLOSED.
func (e *ERP) CloseOrder(orderID string) error {
	if err := e.orders.UpdateStatus(orderID, "CLOSED"); err != nil {
		return err
	}

	order, _ := e.orders.GetByID(orderID)
	if e.bus != nil && order != nil {
		e.bus.Publish(event.Event{
			Type:   event.OrderClosed,
			LineID: order.LineInstanceID,
			Data:   order,
		})
	}

	return nil
}

// CancelOrder transitions an order to CANCELLED.
func (e *ERP) CancelOrder(orderID string) error {
	return e.orders.UpdateStatus(orderID, "CANCELLED")
}

// GetProducts returns all registered products.
func (e *ERP) GetProducts() []ProductRegistration {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]ProductRegistration, len(e.products))
	copy(result, e.products)
	return result
}
