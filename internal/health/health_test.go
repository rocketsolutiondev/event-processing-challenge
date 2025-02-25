package health

import (
	"context"
	"testing"
	"time"
)

func TestHealth(t *testing.T) {
	h := New(nil, nil)
	status := h.Check(context.Background())

	if status.Healthy {
		t.Error("Expected unhealthy status with nil components")
	}

	if status.Timestamp.IsZero() {
		t.Error("Expected non-zero timestamp")
	}

	if len(status.Components) != 2 {
		t.Errorf("Expected 2 components, got %d", len(status.Components))
	}
} 