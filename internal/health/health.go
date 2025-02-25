package health

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/nats-io/nats.go"
	"time"
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

func New(nc *nats.Conn, db *sql.DB) *Health {
	return &Health{
		nats: nc,
		db:   db,
	}
}

func (h *Health) Check(ctx context.Context) Status {
	status := Status{
		Healthy:    true,
		Components: make(map[string]string),
		Timestamp:  time.Now(),
	}

	// Check NATS
	if h.nats.IsConnected() {
		status.Components["nats"] = "connected"
	} else {
		status.Components["nats"] = "disconnected"
		status.Healthy = false
	}

	// Check DB
	if h.db == nil {
		status.Components["database"] = "not configured"
		status.Healthy = false
	} else {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()

		if err := h.db.PingContext(ctx); err != nil {
			status.Components["database"] = fmt.Sprintf("error: %v", err)
			status.Healthy = false
		} else {
			status.Components["database"] = "connected"
		}
	}

	return status
} 