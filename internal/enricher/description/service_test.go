package description

import (
	"context"
	"testing"
	"time"
	"github.com/Bitstarz-eng/event-processing-challenge/internal/casino"
)

func TestDescriptionEnricher(t *testing.T) {
	gameTitles := map[int]string{
		100: "Book of Dead",
		101: "Starburst",
	}

	tests := []struct {
		name        string
		event       casino.Event
		wantDesc    string
	}{
		{
			name: "game start with known game",
			event: casino.Event{
				Type:     "game_start",
				PlayerID: 123,
				GameID:   100,
			},
			wantDesc: "Player 123 started playing Book of Dead",
		},
		{
			name: "game start with unknown game",
			event: casino.Event{
				Type:     "game_start",
				PlayerID: 123,
				GameID:   999,
			},
			wantDesc: "Player 123 started playing Game 999",
		},
		{
			name: "game stop",
			event: casino.Event{
				Type:     "game_stop",
				PlayerID: 123,
				GameID:   101,
			},
			wantDesc: "Player 123 stopped playing Starburst",
		},
		{
			name: "winning bet",
			event: casino.Event{
				Type:     "bet",
				PlayerID: 123,
				GameID:   100,
				Amount:   1000, // 10.00 EUR
				Currency: "EUR",
				HasWon:   true,
			},
			wantDesc: "Player 123 won EUR 10.00 in Book of Dead",
		},
		{
			name: "losing bet",
			event: casino.Event{
				Type:     "bet",
				PlayerID: 123,
				GameID:   100,
				Amount:   1000,
				Currency: "EUR",
				HasWon:   false,
			},
			wantDesc: "Player 123 lost EUR 10.00 in Book of Dead",
		},
		{
			name: "deposit",
			event: casino.Event{
				Type:      "deposit",
				PlayerID:  123,
				Amount:    10000, // 100.00 BTC
				Currency:  "BTC",
				CreatedAt: time.Date(2024, 2, 24, 10, 30, 0, 0, time.UTC),
			},
			wantDesc: "Player 123 deposited BTC 100.00 at 2024-02-24T10:30:00Z",
		},
	}

	svc := New(gameTitles)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.Enrich(context.Background(), &tt.event)
			if err != nil {
				t.Errorf("Enrich() error = %v", err)
				return
			}

			if tt.event.Description != tt.wantDesc {
				t.Errorf("Description = %q, want %q", tt.event.Description, tt.wantDesc)
			}
		})
	}
} 