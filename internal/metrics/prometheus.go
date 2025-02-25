package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	EventsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "casino_events_processed_total",
		Help: "The total number of processed events",
	})

	EventsEnriched = promauto.NewCounter(prometheus.CounterOpts{
		Name: "casino_events_enriched_total",
		Help: "The total number of enriched events",
	})

	EnrichmentErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "casino_enrichment_errors_total",
		Help: "The total number of enrichment errors",
	})

	ProcessingTime = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "casino_event_processing_duration_seconds",
		Help:    "Time spent processing events",
		Buckets: prometheus.DefBuckets,
	})

	// Materializer metrics
	TopPlayerBets = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "casino_top_player_bets",
		Help: "Top player by number of bets",
	}, []string{"player_id"})

	TopPlayerWins = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "casino_top_player_wins",
		Help: "Top player by number of wins",
	}, []string{"player_id"})

	TopPlayerDeposits = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "casino_top_player_deposits_eur",
		Help: "Top player by deposits in EUR",
	}, []string{"player_id"})

	EventsPerSecond = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "casino_events_per_second",
		Help: "Events per second (moving average)",
	})

	ServiceUp = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "casino_service_up",
		Help: "Service health status (1 = healthy, 0 = unhealthy)",
	})
) 