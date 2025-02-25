package subscriber

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "time"
    "github.com/nats-io/nats.go"
    "github.com/Bitstarz-eng/event-processing-challenge/internal/casino"
    "github.com/Bitstarz-eng/event-processing-challenge/internal/metrics"
    "github.com/Bitstarz-eng/event-processing-challenge/internal/health"
    "github.com/Bitstarz-eng/event-processing-challenge/internal/enricher/player"
    "github.com/Bitstarz-eng/event-processing-challenge/internal/enricher/exchange"
    "github.com/Bitstarz-eng/event-processing-challenge/internal/aggregator"
    "github.com/Bitstarz-eng/event-processing-challenge/internal/materializer"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "github.com/Bitstarz-eng/event-processing-challenge/internal/config"
)

const (
    EventsTopic = "casino.events"  // Match the topic name from publisher
)

var (
    databaseConnected = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "casino_database_connected",
        Help: "Database connection status (1 = connected, 0 = disconnected)",
    })
    natsConnected = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "casino_nats_connected",
        Help: "NATS connection status (1 = connected, 0 = disconnected)",
    })
    healthCheckTimestamp = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "casino_health_check_timestamp_seconds",
        Help: "Timestamp of last successful health check",
    })
    healthCheckDuration = promauto.NewHistogram(prometheus.HistogramOpts{
        Name: "casino_health_check_duration_seconds",
        Help: "Duration of health check in seconds",
        Buckets: prometheus.LinearBuckets(0.001, 0.001, 10), // 1ms to 10ms
    })
    componentStatus = promauto.NewGaugeVec(prometheus.GaugeOpts{
        Name: "casino_component_status",
        Help: "Status of each component (1 = healthy, 0 = unhealthy)",
    }, []string{"component", "status"})
)

type Service struct {
    nc *nats.Conn
    enrichers []Enricher
    health *health.Health
    db *sql.DB
    aggregator *aggregator.Service
    materializer *materializer.Service
}

type Enricher interface {
    Enrich(context.Context, *casino.Event) error
}

func New(natsURL string, enrichers ...Enricher) (*Service, error) {
    nc, err := nats.Connect(natsURL)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to NATS: %w", err)
    }

    // Get the database connection from the player enricher
    var db *sql.DB
    for _, e := range enrichers {
        if pe, ok := e.(*player.Service); ok {
            db = pe.DB()
        }
    }

    h := health.New(nc, db)

    // Create aggregator with 1-minute window
    agg := aggregator.New(time.Minute)
    mat := materializer.New()

    return &Service{
        nc: nc,
        enrichers: enrichers,
        health: h,
        db: db,
        aggregator: agg,
        materializer: mat,
    }, nil
}

func (s *Service) Start(ctx context.Context) error {
    // Set initial connection status
    log.Println("Setting initial metrics")
    databaseConnected.Set(1)
    natsConnected.Set(1)
    metrics.ServiceUp.Set(1)
    log.Println("Initial metrics set")

    go s.startHTTP()
    go s.startRateRefresh(ctx)

    sub, err := s.nc.Subscribe(EventsTopic, func(msg *nats.Msg) {
        start := time.Now()
        metrics.IncrementEventsProcessed()

        var event casino.Event
        if err := json.Unmarshal(msg.Data, &event); err != nil {
            log.Printf("Failed to unmarshal event: %v", err)
            metrics.IncrementEnrichmentErrors()
            return
        }

        log.Printf("Processing event: %+v", event)

        // First enrich with player data and currency conversion
        if err := s.enrichers[0].Enrich(ctx, &event); err != nil {
            log.Printf("Player enricher failed: %v", err)
            log.Printf("Failed to enrich event: %v", err)
            metrics.IncrementEnrichmentErrors()
            return  // Stop if currency conversion fails
        }

        // Then enrich with description
        if err := s.enrichers[1].Enrich(ctx, &event); err != nil {
            log.Printf("Description enricher failed: %v", err)
            metrics.IncrementEnrichmentErrors()
        }

        // Output the enriched event
        data, _ := json.Marshal(event)
        if err := s.nc.Publish("casino.events.enriched", data); err != nil {
            log.Printf("Failed to publish enriched event: %v", err)
            metrics.IncrementEnrichmentErrors()
            return
        }

        // Process aggregates with EUR amounts
        s.aggregator.Process(event)
        s.materializer.Process(event)

        metrics.IncrementEventsEnriched()
        metrics.AddProcessingTime(time.Since(start))
        log.Println(string(data))
    })
    if err != nil {
        return fmt.Errorf("failed to subscribe: %w", err)
    }
    defer sub.Unsubscribe()

    <-ctx.Done()
    return nil
}

func (s *Service) Close() error {
    s.nc.Close()
    return nil
}

func (s *Service) startHTTP() {
    mux := http.NewServeMux()

    // Custom metrics handler that includes health data
    mux.HandleFunc("/metrics", s.metricsHandler)

    // Health check ticker
    go func() {
        ticker := time.NewTicker(15 * time.Second)
        for range ticker.C {
            s.updateHealthMetrics(context.Background())
        }
    }()

    mux.HandleFunc("/health", s.healthHandler)

    mux.HandleFunc("/aggregates", func(w http.ResponseWriter, r *http.Request) {
        agg := s.aggregator.GetAggregates()
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(agg)
    })

    mux.HandleFunc("/materialized", func(w http.ResponseWriter, r *http.Request) {
        data := s.materializer.GetData()
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(data)
    })

    srv := &http.Server{
        Addr:    ":8080",
        Handler: mux,
    }

    if err := srv.ListenAndServe(); err != http.ErrServerClosed {
        log.Printf("HTTP server error: %v", err)
    }
}

func (s *Service) metricsHandler(w http.ResponseWriter, r *http.Request) {
    log.Printf("Metrics endpoint called")
    // First update health metrics
    s.updateHealthMetrics(r.Context())

    log.Printf("Health metrics updated")
    // Then serve all Prometheus metrics
    promhttp.Handler().ServeHTTP(w, r)
    log.Printf("Metrics served")
}

func (s *Service) updateHealthMetrics(ctx context.Context) {
    start := time.Now()
    defer func() {
        healthCheckDuration.Observe(time.Since(start).Seconds())
    }()

    // Get health status
    status := s.health.Check(ctx)

    // Update component status metrics
    if status.Healthy {
        metrics.ServiceUp.Set(1)
    } else {
        metrics.ServiceUp.Set(0)
    }

    // Update database status
    if status.Components["database"] == "connected" {
        databaseConnected.Set(1)
        componentStatus.WithLabelValues("database", "connected").Set(1)
        componentStatus.WithLabelValues("database", "error").Set(0)
    } else {
        databaseConnected.Set(0)
        componentStatus.WithLabelValues("database", "connected").Set(0)
        componentStatus.WithLabelValues("database", "error").Set(1)
    }

    // Update NATS status
    if status.Components["nats"] == "connected" {
        natsConnected.Set(1)
        componentStatus.WithLabelValues("nats", "connected").Set(1)
        componentStatus.WithLabelValues("nats", "error").Set(0)
    } else {
        natsConnected.Set(0)
        componentStatus.WithLabelValues("nats", "connected").Set(0)
        componentStatus.WithLabelValues("nats", "error").Set(1)
    }

    healthCheckTimestamp.Set(float64(status.Timestamp.Unix()))
}

func (s *Service) healthHandler(w http.ResponseWriter, r *http.Request) {
    status := health.Status{
        Healthy:   true,
        Components: make(map[string]string),
        Timestamp: time.Now().UTC(),
    }

    // Check database
    if err := s.db.PingContext(r.Context()); err != nil {
        status.Components["database"] = fmt.Sprintf("error: %v", err)
        status.Healthy = false
    } else {
        status.Components["database"] = "connected"
    }

    // Check NATS
    if !s.nc.IsConnected() {
        status.Components["nats"] = "disconnected"
        status.Healthy = false
    } else {
        status.Components["nats"] = "connected"
    }

    // Update metrics
    s.updateHealthMetrics(r.Context())

    w.Header().Set("Content-Type", "application/json")
    if !status.Healthy {
        w.WriteHeader(http.StatusServiceUnavailable)
    }
    json.NewEncoder(w).Encode(status)
}

// startRateRefresh periodically checks and refreshes exchange rates
func (s *Service) startRateRefresh(ctx context.Context) {
    var exchange *exchange.Service
    for _, e := range s.enrichers {
        if pe, ok := e.(*player.Service); ok {
            exchange = pe.GetExchangeService()
            break
        }
    }
    if exchange == nil {
        log.Printf("Warning: Exchange service not found, automatic rate refresh disabled")
        return
    }

    cfg, err := config.Load()
    if err != nil {
        log.Printf("Failed to load config, using default refresh interval: %v", err)
        return
    }

    // Get refresh interval from env
    refreshInterval, err := time.ParseDuration(cfg.ExchangeRateRefreshInterval)
    if err != nil {
        refreshInterval = time.Hour // default to 1 hour
        log.Printf("Invalid refresh interval, using default: %v", refreshInterval)
    }
    
    ticker := time.NewTicker(refreshInterval)
    defer ticker.Stop()
    log.Printf("Starting rate refresh with interval: %v", refreshInterval)

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            log.Printf("Checking exchange rates for refresh...")
            if err := exchange.RefreshRates(); err != nil {
                log.Printf("Failed to refresh exchange rates: %v", err)
            } else {
                log.Printf("Exchange rates refreshed successfully")
            }
        }
    }
} 