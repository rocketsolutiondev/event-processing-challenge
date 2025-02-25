package metrics

import (
	"sync/atomic"
	"time"
)

type Metrics struct {
	EventsProcessed   uint64
	EventsEnriched    uint64
	EnrichmentErrors  uint64
	ProcessingTimeMs  uint64
}

var m = &Metrics{}

func IncrementEventsProcessed() {
	atomic.AddUint64(&m.EventsProcessed, 1)
}

func IncrementEventsEnriched() {
	atomic.AddUint64(&m.EventsEnriched, 1)
}

func IncrementEnrichmentErrors() {
	atomic.AddUint64(&m.EnrichmentErrors, 1)
}

func AddProcessingTime(d time.Duration) {
	atomic.AddUint64(&m.ProcessingTimeMs, uint64(d.Milliseconds()))
}

func GetMetrics() *Metrics {
	return &Metrics{
		EventsProcessed:   atomic.LoadUint64(&m.EventsProcessed),
		EventsEnriched:    atomic.LoadUint64(&m.EventsEnriched),
		EnrichmentErrors:  atomic.LoadUint64(&m.EnrichmentErrors),
		ProcessingTimeMs:  atomic.LoadUint64(&m.ProcessingTimeMs),
	}
} 