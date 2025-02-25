package materializer

import (
    "sync"
    "time"
    "fmt"
    "github.com/Bitstarz-eng/event-processing-challenge/internal/casino"
    "github.com/Bitstarz-eng/event-processing-challenge/internal/metrics"
)

type TopPlayer struct {
    ID    int   `json:"id"`
    Count int64 `json:"count"`
}

type MaterializedData struct {
    EventsTotal                  int64     `json:"events_total"`
    EventsPerMinute             float64   `json:"events_per_minute"`
    EventsPerSecondMovingAverage float64   `json:"events_per_second_moving_average"`
    TopPlayerBets               TopPlayer `json:"top_player_bets"`
    TopPlayerWins              TopPlayer `json:"top_player_wins"`
    TopPlayerDeposits          TopPlayer `json:"top_player_deposits"`
}

type Service struct {
    data          *MaterializedData
    playerStats   map[int]*PlayerStats
    eventTimes    []time.Time // Ring buffer for moving average
    currentIndex  int
    startTime     time.Time
    mu            sync.RWMutex
}

type PlayerStats struct {
    BetTotal     float64  // Track total bet amount in EUR
    WinCount     int64
    WinTotal     float64  // Track total win amount in EUR
    DepositTotal int64
}

func New() *Service {
    return &Service{
        data: &MaterializedData{},
        playerStats: make(map[int]*PlayerStats),
        eventTimes: make([]time.Time, 60), // 60 second buffer
        startTime: time.Now(),
    }
}

func (s *Service) Process(event casino.Event) {
    s.mu.Lock()
    defer s.mu.Unlock()

    // Update total events
    s.data.EventsTotal++

    // Update moving average buffer
    now := time.Now()
    s.eventTimes[s.currentIndex] = now
    s.currentIndex = (s.currentIndex + 1) % len(s.eventTimes)

    // Get or create player stats
    stats, ok := s.playerStats[event.PlayerID]
    if !ok {
        stats = &PlayerStats{}
        s.playerStats[event.PlayerID] = stats
    }

    // Update player stats
    switch event.Type {
    case "bet":
        stats.BetTotal += event.AmountEUR  // Track bet amount
        if event.HasWon {
            stats.WinCount++
            stats.WinTotal += event.AmountEUR  // Track win amount
        }
    case "deposit":
        stats.DepositTotal += int64(event.AmountEUR)
    }

    // Update top players
    s.updateTopPlayers()

    // Calculate events per minute
    duration := now.Sub(s.startTime).Minutes()
    if duration > 0 {
        s.data.EventsPerMinute = float64(s.data.EventsTotal) / duration
    }

    // Calculate moving average
    count := 0
    oneMinuteAgo := now.Add(-time.Minute)
    for _, t := range s.eventTimes {
        if t.After(oneMinuteAgo) {
            count++
        }
    }
    s.data.EventsPerSecondMovingAverage = float64(count) / 60.0
}

func (s *Service) updateTopPlayers() {
    var topBets, topWins, topDeposits TopPlayer

    for playerID, stats := range s.playerStats {
        // Compare bet amounts
        if int64(stats.BetTotal) > topBets.Count {
            topBets.ID = playerID
            topBets.Count = int64(stats.BetTotal)
        }

        // Compare win amounts
        if int64(stats.WinTotal) > topWins.Count {
            topWins.ID = playerID
            topWins.Count = int64(stats.WinTotal)
        }

        // Compare deposit totals
        if stats.DepositTotal > topDeposits.Count {
            topDeposits.ID = playerID
            topDeposits.Count = stats.DepositTotal
        }
    }

    s.data.TopPlayerBets = topBets
    metrics.TopPlayerBets.WithLabelValues(fmt.Sprintf("%d", topBets.ID)).Set(float64(topBets.Count))

    s.data.TopPlayerWins = topWins
    metrics.TopPlayerWins.WithLabelValues(fmt.Sprintf("%d", topWins.ID)).Set(float64(topWins.Count))

    s.data.TopPlayerDeposits = topDeposits
    metrics.TopPlayerDeposits.WithLabelValues(fmt.Sprintf("%d", topDeposits.ID)).Set(float64(topDeposits.Count))

    metrics.EventsPerSecond.Set(s.data.EventsPerSecondMovingAverage)
}

func (s *Service) GetData() MaterializedData {
    s.mu.RLock()
    defer s.mu.RUnlock()
    return *s.data
} 