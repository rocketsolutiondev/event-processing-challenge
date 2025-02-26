package health

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/nats-io/nats.go"
	"net/http"
	"time"
	"github.com/Bitstarz-eng/event-processing-challenge/internal/metrics"
)

type Health struct {
	nats *nats.Conn
	db   *sql.DB
}

type Status struct {
	Healthy    bool              `json:"healthy"`
	Components map[string]string `json:"components"`
	Timestamp  time.Time         `json:"timestamp"`
}

type HealthResponse struct {
	Status string `json:"status"`
}

func New(nc *nats.Conn, db *sql.DB) *Health {
	return &Health{
		nats: nc,
		db:   db,
	}
}

func (h *Health) Check(ctx context.Context) Status {
	now := time.Now()
	start := time.Now()
	defer func() {
		metrics.HealthCheckDuration.Observe(time.Since(start).Seconds())
	}()

	status := Status{
		Healthy:    true,
		Components: make(map[string]string),
		Timestamp:  now,
	}

	// Update health check timestamp using the same time
	metrics.HealthCheckTimestamp.Set(float64(now.Unix()))

	// Check components and update metrics
	dbConnected := h.db != nil && h.db.PingContext(ctx) == nil
	natsConnected := h.nats.IsConnected()
	metrics.UpdateHealthMetrics(dbConnected, natsConnected)

	// Check NATS
	if h.nats.IsConnected() {
		status.Components["nats"] = "connected"
		metrics.ComponentStatus.WithLabelValues("nats", "connected").Set(1)
		metrics.ComponentStatus.WithLabelValues("nats", "error").Set(0)
	} else {
		status.Components["nats"] = "disconnected"
		status.Healthy = false
		metrics.ComponentStatus.WithLabelValues("nats", "connected").Set(0)
		metrics.ComponentStatus.WithLabelValues("nats", "error").Set(1)
	}

	// Check DB
	if h.db == nil {
		status.Components["database"] = "not configured"
		status.Healthy = false
		metrics.ComponentStatus.WithLabelValues("database", "connected").Set(0)
		metrics.ComponentStatus.WithLabelValues("database", "error").Set(1)
	} else {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		if err := h.db.PingContext(ctx); err != nil {
			status.Components["database"] = fmt.Sprintf("error: %v", err)
			status.Healthy = false
			metrics.ComponentStatus.WithLabelValues("database", "connected").Set(0)
			metrics.ComponentStatus.WithLabelValues("database", "error").Set(1)
		} else {
			status.Components["database"] = "connected"
			metrics.ComponentStatus.WithLabelValues("database", "connected").Set(1)
			metrics.ComponentStatus.WithLabelValues("database", "error").Set(0)
		}
	}

	return status
}

func Handler(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status: "ok",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
} 