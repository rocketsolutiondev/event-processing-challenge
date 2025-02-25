package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/nats-io/nats.go"
	"github.com/Bitstarz-eng/event-processing-challenge/internal/casino"
	"github.com/Bitstarz-eng/event-processing-challenge/internal/casino/generator"
	"log"
)

const (
	EventsTopic = "casino.events"
)

type Service struct {
	nc *nats.Conn
	gen generator.Generator
}

func New(natsURL string, gen generator.Generator) (*Service, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return &Service{
		nc: nc,
		gen: gen,
	}, nil
}

func (s *Service) Start(ctx context.Context) error {
	events := s.gen.Generate(ctx)
	
	for event := range events {
		if err := s.PublishEvent(ctx, event); err != nil {
			log.Printf("Failed to publish event: %v", err)
			continue
		}
	}
	
	return nil
}

func (s *Service) PublishEvent(ctx context.Context, event casino.Event) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	if err := s.nc.Publish(EventsTopic, data); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	return nil
}

func (s *Service) Close() error {
	s.nc.Close()
	return nil
} 