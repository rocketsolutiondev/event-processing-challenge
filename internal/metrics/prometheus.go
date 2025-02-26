package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Database metrics
	DatabaseConnected = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "casino_database_connected",
		Help: "Database connection status (1 = connected, 0 = disconnected)",
	})

	// NATS metrics
	NatsConnected = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "casino_nats_connected",
		Help: "NATS connection status (1 = connected, 0 = disconnected)",
	})

	// Health check metrics
	HealthCheckDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "casino_health_check_duration_seconds",
		Help: "Duration of health check in seconds",
		Buckets: prometheus.LinearBuckets(0.001, 0.001, 10),
	})

	// Event processing metrics
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

	EventsByType = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "casino_events_by_type_total",
		Help: "The total number of events by type",
	}, []string{"type"})

	EventsByPlayer = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "casino_events_by_player_total",
		Help: "The total number of events by player",
	}, []string{"player_id"})

	EventsByGame = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "casino_events_by_game_total",
		Help: "The total number of events by game",
	}, []string{"game_id", "game_title"})

	ComponentStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "casino_component_status",
		Help: "Status of each component (1 = healthy, 0 = unhealthy)",
	}, []string{"component", "status"})

	HealthCheckTimestamp = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "casino_health_check_timestamp_seconds",
		Help: "Timestamp of last successful health check",
	})
)

// Helper functions
func UpdateHealthMetrics(dbConnected, natsConnected bool) {
	if dbConnected {
		DatabaseConnected.Set(1)
		ComponentStatus.WithLabelValues("database", "connected").Set(1)
		ComponentStatus.WithLabelValues("database", "error").Set(0)
	} else {
		DatabaseConnected.Set(0)
		ComponentStatus.WithLabelValues("database", "connected").Set(0)
		ComponentStatus.WithLabelValues("database", "error").Set(1)
	}

	if natsConnected {
		NatsConnected.Set(1)
		ComponentStatus.WithLabelValues("nats", "connected").Set(1)
		ComponentStatus.WithLabelValues("nats", "error").Set(0)
	} else {
		NatsConnected.Set(0)
		ComponentStatus.WithLabelValues("nats", "connected").Set(0)
		ComponentStatus.WithLabelValues("nats", "error").Set(1)
	}
} 