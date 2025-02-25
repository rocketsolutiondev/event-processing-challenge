package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
	"github.com/Bitstarz-eng/event-processing-challenge/internal/generator"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	// Get delay from env
	delayMs, err := strconv.Atoi(os.Getenv("EVENT_DELAY_MS"))
	if err != nil || delayMs < 0 {
		delayMs = 0 // default to no delay
	}

	// Connect to NATS
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	// Handle graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Generate and publish events
	log.Printf("Starting publisher with NATS URL: %s and delay: %dms", natsURL, delayMs)
	events := generator.Generate(ctx)
	for event := range events {
		data, err := json.Marshal(event)
		if err != nil {
			log.Printf("Failed to marshal event: %v", err)
			continue
		}

		if err := nc.Publish("casino.events", data); err != nil {
			log.Printf("Failed to publish event: %v", err)
			continue
		}

		// Apply configured delay
		if delayMs > 0 {
			time.Sleep(time.Duration(delayMs) * time.Millisecond)
		}
	}
} 