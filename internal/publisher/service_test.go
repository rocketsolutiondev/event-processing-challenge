package publisher

import (
	"context"
	"encoding/json"
	"github.com/Bitstarz-eng/event-processing-challenge/internal/casino"
	"github.com/nats-io/nats.go"
	"testing"
	"time"
)

func TestPublisher(t *testing.T) {
	// Create mock generator
	mockGen := &mockGenerator{
		events: make(chan casino.Event, 10),
	}
	// Pre-fill some test events
	go func() {
		mockGen.events <- casino.Event{ID: 1, Type: "test"}
		mockGen.events <- casino.Event{ID: 2, Type: "test"}
		mockGen.events <- casino.Event{ID: 3, Type: "test"}
		mockGen.events <- casino.Event{ID: 4, Type: "test"}
		mockGen.events <- casino.Event{ID: 5, Type: "test"}
		close(mockGen.events)
	}()

	// Connect to NATS
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	// Create a subscriber to verify published messages
	receivedEvents := make(chan casino.Event, 10)
	sub, err := nc.Subscribe(EventsTopic, func(msg *nats.Msg) {
		var event casino.Event
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			t.Errorf("Failed to unmarshal event: %v", err)
			return
		}
		receivedEvents <- event
	})
	if err != nil {
		t.Fatalf("Failed to subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	// Create publisher service
	pub, err := New(nats.DefaultURL, mockGen)
	if err != nil {
		t.Fatalf("Failed to create publisher: %v", err)
	}
	defer pub.Close()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start publisher in a goroutine
	go func() {
		if err := pub.Start(ctx); err != nil {
			t.Errorf("Publisher stopped with error: %v", err)
		}
	}()

	// Wait for some events
	eventCount := 0
	timeout := time.After(1 * time.Second)

	for eventCount < 5 {
		select {
		case event := <-receivedEvents:
			t.Logf("Received event: %+v", event)
			eventCount++
		case <-timeout:
			t.Fatal("Timeout waiting for events")
		}
	}

	if eventCount < 5 {
		t.Errorf("Expected at least 5 events, got %d", eventCount)
	}
}

// mockGenerator implements casino.Generator interface for testing
type mockGenerator struct {
	events chan casino.Event
}

func (m *mockGenerator) Generate(ctx context.Context) <-chan casino.Event {
	return m.events
} 