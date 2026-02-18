package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"

	"machine-simulator-2/internal/event"
)

// Config holds webhook publisher settings.
type Config struct {
	Enabled       bool     `yaml:"enabled"`
	TargetURL     string   `yaml:"target_url"`
	EventTypes    []string `yaml:"event_types"`
	TimeoutMS     int      `yaml:"timeout_ms"`
	RetryAttempts int      `yaml:"retry_attempts"`
	RetryDelayMS  int      `yaml:"retry_delay_ms"`
}

// Payload is the JSON body sent to the webhook target.
type Payload struct {
	EventType string      `json:"event_type"`
	Timestamp int64       `json:"timestamp"`
	MachineID string      `json:"machine_id,omitempty"`
	LineID    string      `json:"line_id,omitempty"`
	Data      interface{} `json:"data"`
}

// Publisher subscribes to events and POSTs them to a target URL.
type Publisher struct {
	cfg    Config
	bus    *event.Bus
	client *http.Client
}

// New creates a new webhook publisher.
func New(cfg Config, bus *event.Bus) *Publisher {
	timeout := time.Duration(cfg.TimeoutMS) * time.Millisecond
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	return &Publisher{
		cfg: cfg,
		bus: bus,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Start subscribes to configured event types and begins publishing.
func (p *Publisher) Start(ctx context.Context) {
	if !p.cfg.Enabled || p.cfg.TargetURL == "" {
		return
	}

	for _, eventType := range p.cfg.EventTypes {
		et := eventType
		p.bus.Subscribe(et, func(e event.Event) {
			go p.send(ctx, e)
		})
	}

	log.Info().
		Str("target_url", p.cfg.TargetURL).
		Int("event_types", len(p.cfg.EventTypes)).
		Msg("Webhook publisher started")
}

func (p *Publisher) send(ctx context.Context, e event.Event) {
	payload := Payload{
		EventType: e.Type,
		Timestamp: time.Now().UnixMilli(),
		MachineID: e.MachineID,
		LineID:    e.LineID,
		Data:      e.Data,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Error().Err(err).Str("event_type", e.Type).Msg("Webhook: failed to marshal payload")
		return
	}

	retryDelay := time.Duration(p.cfg.RetryDelayMS) * time.Millisecond
	if retryDelay == 0 {
		retryDelay = time.Second
	}

	attempts := p.cfg.RetryAttempts
	if attempts <= 0 {
		attempts = 1
	}

	for attempt := 0; attempt < attempts; attempt++ {
		if ctx.Err() != nil {
			return
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.TargetURL, bytes.NewReader(body))
		if err != nil {
			log.Error().Err(err).Msg("Webhook: failed to create request")
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := p.client.Do(req)
		if err != nil {
			log.Warn().Err(err).Int("attempt", attempt+1).Msg("Webhook: request failed")
			if attempt < attempts-1 {
				delay := retryDelay * time.Duration(1<<uint(attempt))
				time.Sleep(delay)
			}
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return
		}

		log.Warn().Int("status", resp.StatusCode).Int("attempt", attempt+1).Msg("Webhook: non-2xx response")
		if attempt < attempts-1 {
			delay := retryDelay * time.Duration(1<<uint(attempt))
			time.Sleep(delay)
		}
	}
}
