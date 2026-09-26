package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// TelemetryEvent captures runtime performance and financial savings metrics.
type TelemetryEvent struct {
	QueryID      string    `json:"query_id"`
	Prompt       string    `json:"prompt"`
	CacheHit     bool      `json:"cache_hit"`
	Similarity   float32   `json:"similarity"`
	Threshold    float32   `json:"threshold"`
	LatencyMs    int64     `json:"latency_ms"`
	BaselineMs   int64     `json:"baseline_ms"`
	TokensInput  int       `json:"tokens_input"`
	TokensOutput int       `json:"tokens_output"`
	TokensSaved  int       `json:"tokens_saved"`
	CostSaved    float64   `json:"cost_saved"`
	Model        string    `json:"model"`
	Timestamp    time.Time `json:"timestamp"`
}

// Dispatcher manages an asynchronous, non-blocking telemetry event queue.
type Dispatcher struct {
	queue        chan TelemetryEvent
	endpoint     string
	client       *http.Client
	eventsSent   atomic.Uint64
	eventsDrop   atomic.Uint64
	eventsQueued atomic.Uint64
	stopCh       chan struct{}
	wg           sync.WaitGroup
}

// NewDispatcher creates a telemetry dispatcher with a 2,000 capacity buffered channel.
func NewDispatcher(endpoint string) *Dispatcher {
	d := &Dispatcher{
		queue:    make(chan TelemetryEvent, 2000),
		endpoint: endpoint,
		client: &http.Client{
			Timeout: 2 * time.Second,
		},
		stopCh: make(chan struct{}),
	}

	d.wg.Add(1)
	go d.worker()

	return d
}

// Dispatch non-blockingly enqueues a telemetry event.
// If the buffer is full, it drops the event rather than slowing down the proxy data plane.
func (d *Dispatcher) Dispatch(event TelemetryEvent) {
	d.eventsQueued.Add(1)
	select {
	case d.queue <- event:
	default:
		d.eventsDrop.Add(1)
	}
}

func (d *Dispatcher) worker() {
	defer d.wg.Done()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	batch := make([]TelemetryEvent, 0, 100)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		toSend := make([]TelemetryEvent, len(batch))
		copy(toSend, batch)
		batch = batch[:0]

		go d.sendBatch(toSend)
	}

	for {
		select {
		case <-d.stopCh:
			// Drain remaining
			for {
				select {
				case ev := <-d.queue:
					batch = append(batch, ev)
					if len(batch) >= 100 {
						flush()
					}
				default:
					flush()
					return
				}
			}
		case ev := <-d.queue:
			batch = append(batch, ev)
			if len(batch) >= 50 {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (d *Dispatcher) sendBatch(batch []TelemetryEvent) {
	if d.endpoint == "" {
		return
	}

	data, err := json.Marshal(batch)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.endpoint, bytes.NewBuffer(data))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.client.Do(req)
	if err != nil {
		// Control plane might be restarting or not yet running; proxy remains 100% resilient
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusAccepted {
		d.eventsSent.Add(uint64(len(batch)))
	}
}

// Stats returns current telemetry queue statistics.
func (d *Dispatcher) Stats() map[string]uint64 {
	return map[string]uint64{
		"queued":  d.eventsQueued.Load(),
		"sent":    d.eventsSent.Load(),
		"dropped": d.eventsDrop.Load(),
	}
}

// Stop shuts down the dispatcher and flushes remaining queue items.
func (d *Dispatcher) Stop() {
	close(d.stopCh)
	d.wg.Wait()
}
