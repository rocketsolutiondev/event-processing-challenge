package player

import (
    "context"
    "database/sql"
    "errors"
    "testing"
    "time"
    "github.com/Bitstarz-eng/event-processing-challenge/internal/casino"
)

func TestPlayerEnricher(t *testing.T) {
    now := time.Now().UTC().Truncate(time.Second)

    tests := []struct {
        name      string
        playerID  int
        mockRow   *mockRow
        wantMail  string
        wantTime  time.Time
        wantErr   bool
    }{
        {
            name:     "existing player",
            playerID: 999,
            mockRow:  &mockRow{email: "test@example.com", signedIn: now},
            wantMail: "test@example.com",
            wantTime: now,
        },
        {
            name:     "non-existent player",
            playerID: 9999,
            mockRow:  &mockRow{email: "", signedIn: time.Time{}, err: sql.ErrNoRows},
            wantMail: "",
            wantTime: time.Time{},
        },
        {
            name:     "database error",
            playerID: 1,
            mockRow:  &mockRow{email: "", signedIn: time.Time{}, err: errors.New("database connection lost")},
            wantErr: true,
        },
        {
            name:     "context cancelled",
            playerID: 1,
            mockRow:  &mockRow{email: "", signedIn: time.Time{}, err: context.Canceled},
            wantErr: true,
        },
        {
            name:     "zero player ID",
            playerID: 0,
            mockRow:  &mockRow{email: "", signedIn: time.Time{}, err: sql.ErrNoRows},
            wantMail: "",
            wantTime: time.Time{},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mock := &mockDB{
                queryFunc: func(ctx context.Context, query string, args ...interface{}) Scanner {
                    return tt.mockRow
                },
            }
            svc := &Service{db: mock}

            // Create context
            ctx := context.Background()
            if tt.name == "context cancelled" {
                var cancel context.CancelFunc
                ctx, cancel = context.WithCancel(ctx)
                cancel()
            }

            // Create test event
            event := casino.Event{PlayerID: tt.playerID}

            // Test enrichment
            err := svc.Enrich(ctx, &event)
            if (err != nil) != tt.wantErr {
                t.Errorf("Enrich() error = %v, wantErr %v", err, tt.wantErr)
                return
            }

            if !tt.wantErr {
                if event.Player.Email != tt.wantMail {
                    t.Errorf("Player.Email = %v, want %v", event.Player.Email, tt.wantMail)
                }

                if !event.Player.LastSignedInAt.Equal(tt.wantTime) {
                    t.Errorf("Player.LastSignedInAt = %v, want %v", 
                        event.Player.LastSignedInAt, tt.wantTime)
                }
            }
        })
    }
}

// Integration test with real database
func TestPlayerEnricherIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    // Connect to real database
    db, err := sql.Open("postgres", "postgres://casino:casino@localhost:5432/casino?sslmode=disable")
    if err != nil {
        t.Skip("Skipping test, no database connection:", err)
    }
    defer db.Close()

    if err := db.Ping(); err != nil {
        t.Skip("Skipping test, database not available:", err)
    }

    // Insert test data
    now := time.Now().UTC().Truncate(time.Second)
    _, err = db.Exec(`
        INSERT INTO players (id, email, last_signed_in_at)
        VALUES ($1, $2, $3)
        ON CONFLICT (id) DO UPDATE 
        SET email = EXCLUDED.email, 
            last_signed_in_at = EXCLUDED.last_signed_in_at
    `, 999, "test@example.com", now)
    if err != nil {
        t.Fatalf("Failed to insert test data: %v", err)
    }

    svc := &Service{db: &sqlDB{DB: db}}

    // Test with real data
    event := casino.Event{PlayerID: 999}
    if err := svc.Enrich(context.Background(), &event); err != nil {
        t.Fatalf("Failed to enrich event: %v", err)
    }

    if event.Player.Email != "test@example.com" {
        t.Errorf("Player.Email = %v, want test@example.com", event.Player.Email)
    }

    if !event.Player.LastSignedInAt.Equal(now) {
        t.Errorf("Player.LastSignedInAt = %v, want %v", 
            event.Player.LastSignedInAt, now)
    }
} 