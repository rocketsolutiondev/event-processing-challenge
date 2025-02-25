package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "github.com/Bitstarz-eng/event-processing-challenge/internal/config"
    "github.com/Bitstarz-eng/event-processing-challenge/internal/subscriber"
    "github.com/Bitstarz-eng/event-processing-challenge/internal/enricher/player"
    "github.com/Bitstarz-eng/event-processing-challenge/internal/enricher/description"
)

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    log.Println("Starting subscriber...")
    log.Printf("NATS URL: %s", cfg.NATSURL)
    log.Printf("DB URL: %s", cfg.GetDBURL())

    // Create enrichers
    playerEnricher, err := player.New(cfg.GetDBURL())
    if err != nil {
        log.Fatalf("Failed to create player enricher: %v", err)
    }
    defer playerEnricher.Close()

    // Game titles mapping
    gameTitles := map[int]string{
        100: "Book of Dead",
        101: "Starburst",
        102: "Gonzo's Quest",
        103: "Dead or Alive",
        104: "Fire Joker",
    }
    descriptionEnricher := description.New(gameTitles)

    // Create and start subscriber
    sub, err := subscriber.New(cfg.NATSURL, playerEnricher, descriptionEnricher)
    if err != nil {
        log.Fatalf("Failed to create subscriber: %v", err)
    }
    defer sub.Close()

    // Handle graceful shutdown
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    log.Println("Starting subscriber...")
    if err := sub.Start(ctx); err != nil {
        log.Printf("Subscriber stopped with error: %v", err)
    }
} 