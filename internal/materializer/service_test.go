package materializer

import (
    "testing"
    "github.com/Bitstarz-eng/event-processing-challenge/internal/casino"
)

func TestMaterializer(t *testing.T) {
    s := New()

    // Process some test events
    events := []casino.Event{
        {ID: 1, PlayerID: 1, Type: "bet", AmountEUR: 100, HasWon: true},
        {ID: 2, PlayerID: 1, Type: "bet", AmountEUR: 100, HasWon: false},
        {ID: 3, PlayerID: 2, Type: "deposit", AmountEUR: 1000},
        {ID: 4, PlayerID: 2, Type: "bet", AmountEUR: 100, HasWon: true},
    }

    for _, e := range events {
        s.Process(e)
    }

    // Get materialized data
    data := s.GetData()

    // Check total events
    if data.EventsTotal != 4 {
        t.Errorf("Expected 4 total events, got %d", data.EventsTotal)
    }

    // Check top bettor
    if data.TopPlayerBets.ID != 1 || data.TopPlayerBets.Count != 2 {
        t.Errorf("Expected player 1 with 2 bets, got player %d with %d bets", 
            data.TopPlayerBets.ID, data.TopPlayerBets.Count)
    }

    // Check top winner
    if data.TopPlayerWins.ID != 1 || data.TopPlayerWins.Count != 1 {
        t.Errorf("Expected player 1 with 1 win, got player %d with %d wins",
            data.TopPlayerWins.ID, data.TopPlayerWins.Count)
    }

    // Check top depositor
    if data.TopPlayerDeposits.ID != 2 || data.TopPlayerDeposits.Count != 1000 {
        t.Errorf("Expected player 2 with 1000 deposits, got player %d with %d", 
            data.TopPlayerDeposits.ID, data.TopPlayerDeposits.Count)
    }
} 