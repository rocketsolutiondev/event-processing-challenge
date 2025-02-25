package player

import (
    "context"
    "database/sql"
    "fmt"
    "log"
    _ "github.com/lib/pq"
    "github.com/Bitstarz-eng/event-processing-challenge/internal/casino"
    "github.com/Bitstarz-eng/event-processing-challenge/internal/enricher/exchange"
)

type Service struct {
    db *sql.DB
    exchange *exchange.Service
}

func New(dbURL string) (*Service, error) {
    db, err := sql.Open("postgres", dbURL)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }

    // Test the connection
    if err := db.Ping(); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }

    exchange, err := exchange.New(db)
    if err != nil {
        return nil, fmt.Errorf("failed to create exchange service: %w", err)
    }

    return &Service{
        db: db,
        exchange: exchange,
    }, nil
}

func (s *Service) Enrich(ctx context.Context, event *casino.Event) error {
    // First convert amount to EUR regardless of player data
    if event.Currency != "EUR" {
        rate, err := s.exchange.GetRate(event.Currency)
        if err != nil {
            return fmt.Errorf("failed to get rate: %w", err)
        }

        // Simple conversion for all currencies
        event.AmountEUR = float64(event.Amount) / rate

        log.Printf("Converting %d %s to EUR: amount=%.2f, rate=%.10f", 
            event.Amount, event.Currency, event.AmountEUR, rate)
    } else {
        event.AmountEUR = float64(event.Amount)
    }

    // Then try to get player data
    var player casino.Player
    err := s.db.QueryRowContext(ctx, 
        `SELECT email, last_signed_in_at 
         FROM players 
         WHERE id = $1`, 
        event.PlayerID,
    ).Scan(&player.Email, &player.LastSignedInAt)

    if err == sql.ErrNoRows {
        log.Printf("No player data found for ID: %d", event.PlayerID)
        return nil
    }
    if err != nil {
        return fmt.Errorf("failed to query player data: %w", err)
    }

    event.Player = player

    return nil
}

func (s *Service) Close() error {
    return s.db.Close()
}

func (s *Service) DB() *sql.DB {
    return s.db
}

func (s *Service) GetExchangeService() *exchange.Service {
    return s.exchange
} 