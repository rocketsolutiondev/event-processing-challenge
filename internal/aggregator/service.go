package aggregator

import (
    "sync"
    "time"
    "github.com/Bitstarz-eng/event-processing-challenge/internal/casino"
)

type Aggregates struct {
    TotalBetsEUR     int64
    TotalDepositsEUR int64
    TotalWinsEUR     int64
    UniqueUsers      map[int]bool
    ActiveGames      map[int]int // gameID -> active players
    mu              sync.RWMutex
}

type Service struct {
    aggregates *Aggregates
    window     time.Duration
}

type Aggregate struct {
    Count     int       `json:"count"`
    TotalEUR  float64   `json:"total_eur"`
    StartTime time.Time `json:"start_time"`
    EndTime   time.Time `json:"end_time"`
}

func New(window time.Duration) *Service {
    return &Service{
        aggregates: &Aggregates{
            UniqueUsers: make(map[int]bool),
            ActiveGames: make(map[int]int),
        },
        window: window,
    }
}

func (s *Service) Process(event casino.Event) {
    s.aggregates.mu.Lock()
    defer s.aggregates.mu.Unlock()

    // Track unique users
    s.aggregates.UniqueUsers[event.PlayerID] = true

    switch event.Type {
    case "bet":
        s.aggregates.TotalBetsEUR = s.aggregates.TotalBetsEUR + int64(event.AmountEUR)
        if event.HasWon {
            s.aggregates.TotalWinsEUR = s.aggregates.TotalWinsEUR + int64(event.AmountEUR)
        }
    case "deposit":
        s.aggregates.TotalDepositsEUR = s.aggregates.TotalDepositsEUR + int64(event.AmountEUR)
    case "game_start":
        s.aggregates.ActiveGames[event.GameID]++
    case "game_stop":
        if s.aggregates.ActiveGames[event.GameID] > 0 {
            s.aggregates.ActiveGames[event.GameID]--
        }
    }
}

func (s *Service) GetAggregates() Aggregates {
    s.aggregates.mu.RLock()
    defer s.aggregates.mu.RUnlock()

    // Return a copy to avoid race conditions
    return Aggregates{
        TotalBetsEUR:     s.aggregates.TotalBetsEUR,
        TotalDepositsEUR: s.aggregates.TotalDepositsEUR,
        TotalWinsEUR:     s.aggregates.TotalWinsEUR,
        UniqueUsers:      make(map[int]bool),
        ActiveGames:      make(map[int]int),
    }
} 