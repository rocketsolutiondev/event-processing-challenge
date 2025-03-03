package description

import (
    "context"
    "fmt"
    "time"
    "github.com/Bitstarz-eng/event-processing-challenge/internal/casino"
)

type Service struct {
}

func New() *Service {
    return &Service{}
}

func (s *Service) Enrich(ctx context.Context, event *casino.Event) error {
    // Get game title
    game, ok := casino.Games[event.GameID]
    if !ok {
        game = casino.Game{Title: fmt.Sprintf("Game %d", event.GameID)}
    }

    amount := float64(event.Amount) // Use actual amount for all currencies

    switch event.Type {
    case "bet":
        if event.HasWon {
            event.Description = fmt.Sprintf("Player %d won %s %.2f in %s", 
                event.PlayerID, event.Currency, amount, game.Title)
        } else {
            event.Description = fmt.Sprintf("Player %d lost %s %.2f in %s",
                event.PlayerID, event.Currency, amount, game.Title)
        }
    case "deposit":
        event.Description = fmt.Sprintf("Player %d deposited %s %.2f at %s",
            event.PlayerID, event.Currency, amount, event.CreatedAt.Format(time.RFC3339))
    case "game_start":
        event.Description = fmt.Sprintf("Player %d started playing %s", 
            event.PlayerID, game.Title)
    case "game_stop":
        event.Description = fmt.Sprintf("Player %d stopped playing %s", 
            event.PlayerID, game.Title)
    }

    return nil
}

func (s *Service) getGameTitle(id int) string {
    if title, ok := casino.Games[id]; ok {
        return title.Title
    }
    return fmt.Sprintf("Game %d", id)
} 