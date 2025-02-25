package subscriber

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "testing"
    "time"
    "github.com/nats-io/nats.go"
    "github.com/Bitstarz-eng/event-processing-challenge/internal/casino"
    "github.com/Bitstarz-eng/event-processing-challenge/internal/enricher/currency"
    "github.com/Bitstarz-eng/event-processing-challenge/internal/enricher/player"
    "github.com/Bitstarz-eng/event-processing-challenge/internal/enricher/description"
)

func TestIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    // Connect to NATS
    nc := waitForNATS(t)
    defer nc.Close()

    // Create enrichers
    currencyEnricher := currency.NewMock()
    playerEnricher, err := player.New("postgres://casino:casino@localhost:5432/casino?sslmode=disable")
    if err != nil {
        t.Fatalf("Failed to create player enricher: %v", err)
    }
    defer playerEnricher.Close()

    gameTitles := map[int]string{
        100: "Book of Dead",
    }
    descriptionEnricher := description.New(gameTitles)

    // Create subscriber
    sub, err := New(nats.DefaultURL, currencyEnricher, playerEnricher, descriptionEnricher)
    if err != nil {
        t.Fatalf("Failed to create subscriber: %v", err)
    }
    defer sub.Close()

    // Setup test data in database
    db, err := sql.Open("postgres", "postgres://casino:casino@localhost:5432/casino?sslmode=disable")
    if err != nil {
        t.Fatalf("Failed to connect to database: %v", err)
    }
    defer db.Close()

    now := time.Now().UTC().Truncate(time.Second)
    _, err = db.Exec(`
        INSERT INTO players (id, email, last_signed_in_at)
        VALUES ($1, $2, $3)
        ON CONFLICT (id) DO UPDATE 
        SET email = EXCLUDED.email, 
            last_signed_in_at = EXCLUDED.last_signed_in_at
    `, 123, "player123@example.com", now)
    if err != nil {
        t.Fatalf("Failed to insert test data: %v", err)
    }

    // Create channel to receive enriched events
    enriched := make(chan casino.Event)
    enrichedSub, err := nc.Subscribe("casino.events.enriched", func(msg *nats.Msg) {
        var event casino.Event
        if err := json.Unmarshal(msg.Data, &event); err != nil {
            t.Errorf("Failed to unmarshal enriched event: %v", err)
            return
        }
        enriched <- event
    })
    if err != nil {
        t.Fatalf("Failed to subscribe to enriched events: %v", err)
    }
    defer enrichedSub.Unsubscribe()

    // Start subscriber in background
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    go func() {
        if err := sub.Start(ctx); err != nil {
            t.Errorf("Subscriber stopped with error: %v", err)
        }
    }()

    // Wait for NATS subscriptions to be ready
    time.Sleep(100 * time.Millisecond)
    nc.Flush() // Ensure NATS connection is ready

    // Test cases
    tests := []struct {
        name     string
        event    casino.Event
        validate func(*testing.T, casino.Event)
    }{
        {
            name: "bet event with all enrichments",
            event: casino.Event{
                ID:       1,
                Type:     "bet",
                PlayerID: 123,
                GameID:   100,
                Amount:   1000,
                Currency: "USD",
                HasWon:   true,
                CreatedAt: now,
            },
            validate: func(t *testing.T, e casino.Event) {
                if e.AmountEUR != 910 {
                    t.Error("AmountEUR not enriched correctly")
                }
                if e.Player.Email != "player123@example.com" {
                    t.Errorf("Player email = %q, want player123@example.com", e.Player.Email)
                }
                if !e.Player.LastSignedInAt.Equal(now) {
                    t.Errorf("Player LastSignedInAt = %v, want %v", e.Player.LastSignedInAt, now)
                }
                wantDesc := "Player 123 won USD 10.00 in Book of Dead"
                if e.Description != wantDesc {
                    t.Errorf("Description = %q, want %q", e.Description, wantDesc)
                }
            },
        },
        {
            name: "deposit event with enrichments",
            event: casino.Event{
                ID:       2,
                Type:     "deposit",
                PlayerID: 123,
                Amount:   1,    // 0.001 BTC (smallest unit)
                Currency: "BTC",
                CreatedAt: now,
            },
            validate: func(t *testing.T, e casino.Event) {
                if e.AmountEUR != 35 { // 0.001 BTC = 35 EUR
                    t.Error("AmountEUR not enriched correctly")
                }
                if e.Player.Email != "player123@example.com" {
                    t.Errorf("Player email = %q, want player123@example.com", e.Player.Email)
                }
                wantDesc := fmt.Sprintf("Player 123 deposited BTC 0.001 at %s", now.Format(time.RFC3339))
                if e.Description != wantDesc {
                    t.Errorf("Description = %q, want %q", e.Description, wantDesc)
                }
            },
        },
        {
            name: "game start event with enrichments",
            event: casino.Event{
                ID:       3,
                Type:     "game_start",
                PlayerID: 123,
                GameID:   100,
                CreatedAt: now,
            },
            validate: func(t *testing.T, e casino.Event) {
                if e.Player.Email != "player123@example.com" {
                    t.Errorf("Player email = %q, want player123@example.com", e.Player.Email)
                }
                wantDesc := "Player 123 started playing Book of Dead"
                if e.Description != wantDesc {
                    t.Errorf("Description = %q, want %q", e.Description, wantDesc)
                }
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Publish test event
            data, _ := json.Marshal(tt.event)
            if err := nc.Publish("casino.events", data); err != nil {
                t.Fatalf("Failed to publish event: %v", err)
            }
            nc.Flush()

            // Wait for enriched event
            select {
            case event := <-enriched:
                tt.validate(t, event)
            case <-time.After(time.Second):
                t.Fatal("Timeout waiting for enriched event")
            }
        })
    }
} 