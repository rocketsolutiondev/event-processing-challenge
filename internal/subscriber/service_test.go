package subscriber

import (
    "context"
    "encoding/json"
    "testing"
    "time"
    "github.com/nats-io/nats.go"
    "github.com/Bitstarz-eng/event-processing-challenge/internal/casino"
)

// mockEnricher implements Enricher interface for testing
type mockEnricher struct {
    enrichFunc func(context.Context, *casino.Event) error
}

func (m *mockEnricher) Enrich(ctx context.Context, event *casino.Event) error {
    if m.enrichFunc != nil {
        return m.enrichFunc(ctx, event)
    }
    return nil
}

func waitForNATS(t *testing.T) *nats.Conn {
    var nc *nats.Conn
    var err error
    
    for i := 0; i < 3; i++ {
        nc, err = nats.Connect(nats.DefaultURL)
        if err == nil && nc.IsConnected() {
            return nc
        }
        time.Sleep(time.Second)
    }
    t.Fatalf("Failed to connect to NATS after retries: %v", err)
    return nil
}

func TestSubscriber(t *testing.T) {
    nc := waitForNATS(t)
    defer nc.Close()

    // Create mock enricher
    enriched := make(chan struct{})
    mockEnricher := &mockEnricher{
        enrichFunc: func(ctx context.Context, event *casino.Event) error {
            event.Description = "enriched"
            enriched <- struct{}{}
            return nil
        },
    }

    // Create subscriber
    sub, err := New(nats.DefaultURL, mockEnricher)
    if err != nil {
        t.Fatalf("Failed to create subscriber: %v", err)
    }
    defer sub.Close()

    // Start subscriber in background
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    go func() {
        if err := sub.Start(ctx); err != nil {
            t.Errorf("Subscriber stopped with error: %v", err)
        }
    }()

    // Wait for subscription to be ready
    time.Sleep(100 * time.Millisecond)

    // Publish test event
    event := casino.Event{
        ID:   1,
        Type: "test",
    }
    data, _ := json.Marshal(event)
    if err := nc.Publish(EventsTopic, data); err != nil {
        t.Fatalf("Failed to publish event: %v", err)
    }
    nc.Flush() // Ensure the message is sent

    // Wait for enrichment
    select {
    case <-enriched:
        // Success
    case <-time.After(time.Second):
        t.Fatal("Timeout waiting for event enrichment")
    }
} 