# Casino Event Processing System Documentation

## Architecture Overview

```
 ┌─────────────┐    ┌─────────────┐
 │  Generator  │───►│  Publisher  │
 └─────────────┘    └──────┬──────┘
                           │
                           ▼
 ┌─────────────┐    ┌──────────────┐
 │   Postgres  │◄───┤  Subscriber  │
 └─────────────┘    └──────┬───────┘
                           │
        ┌─────────────┬─────┴────┬─────────────┬──────────────┐
        ▼             ▼          ▼             ▼              ▼
 ┌──────────┐  ┌─────────┐ ┌─────────┐  ┌─────────┐   ┌──────────┐
 │ Currency │  │ Player  │ │  Game   │  │  Desc   │   │ Metrics  │
 │ Enricher │  │Enricher │ │Enricher │  │Enricher │   │ Enricher │
 └──────────┘  └─────────┘ └─────────┘  └─────────┘   └────┬─────┘
                                                           │
                                                           ▼
                                                    ┌──────────┐
                                                    │ Grafana  │
                                                    └──────────┘
```

## Database Migrations

The system uses automatic database migrations that run when the database container starts up. The migrations are handled through Docker Compose's initialization system.

### Migration Structure
```
db/
└── migrations/
    ├── 00-init.sh           # Shell script that executes migrations in order
    ├── 00001.create_base.sql    # Creates initial tables (players, etc.)
    └── 00002.exchange_rates.sql # Creates exchange rates table
```

### How It Works
1. The `database` service in docker-compose.yml mounts the migrations directory:
   ```yaml
   volumes:
     - "./db/migrations:/docker-entrypoint-initdb.d"
   ```

2. PostgreSQL's Docker image automatically executes files in `/docker-entrypoint-initdb.d` in alphabetical order

3. The `00-init.sh` script runs first and executes the SQL migrations in sequence:
   ```bash
   cd /docker-entrypoint-initdb.d
   psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" -f 00001.create_base.sql
   psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" -f 00002.exchange_rates.sql
   ```

### Migration Files
- `00001.create_base.sql`: Creates initial tables for player data
- `00002.exchange_rates.sql`: Creates exchange rates table with initial currency data

### Execution
Migrations run automatically when:
- First time database container starts
- After volume is removed (`docker-compose down -v`)
- When database container is recreated

## Components

### Publisher
- Receives events from the generator
- Publishes to NATS topic "casino.events"
- Handles graceful shutdown

### Subscriber
- Subscribes to "casino.events"
- Runs enrichment pipeline
- Publishes enriched events to "casino.events.enriched"
- Collects metrics

### Enrichers

#### Currency Enricher
- Converts amounts to EUR using exchangerate.host API
- Handles currency conversion with high precision
- Caches rates for 1 minute
- Rate limited to 1 request/minute
- Graceful error handling

#### Player Enricher
- Looks up player data from Postgres
- No caching (per requirements)
- Handles missing players gracefully
- Adds email and last_signed_in_at

#### Description Enricher
- Generates human-friendly descriptions
- Currency-specific formatting:
  - All currencies: 10 decimal places for maximum precision
- Uses game title mapping

## Quick Start

1. Clone the repository:
```bash
git clone https://github.com/Bitstarz-eng/event-processing-challenge.git
cd event-processing-challenge
```

2. Build and start all services:
```bash
# First build all services
docker-compose build --no-cache

# Then start everything (includes migrations, subscriber, and publisher)
docker-compose up -d
```

This will:
- Build all required containers
- Start infrastructure (NATS, PostgreSQL, Prometheus, Grafana)
- Run database migrations automatically
- Start both subscriber and publisher services
- Set up metrics collection

## Manual Setup (Alternative)

For development or debugging, you can run components separately. This is useful when you want to run the subscriber or publisher with local modifications:

## Setup Instructions

1. Start required infrastructure:
```bash
# Start only NATS and Postgres (not the app containers)
docker-compose up -d nats postgres prometheus grafana
```

2. Run migrations:
```bash
# Only needed if you want to run migrations manually
make migrate
```

3. Start the subscriber:
```bash
# Run subscriber locally for development
# Make sure the app container is not running to avoid port conflicts
go run cmd/subscriber/main.go
```

4. Start the publisher:
```bash
# Run publisher locally for development
# Make sure the app container is not running to avoid port conflicts
go run cmd/publisher/main.go
```

Notes: 
- Manual setup is primarily for development purposes
- For production use, prefer `docker-compose up -d` which handles everything automatically
- Cannot run containerized and local versions simultaneously due to port conflicts
- When switching between containerized and local versions, use `docker-compose down` first

## Testing

The project includes both unit and integration tests:

```bash
# Run unit tests
# These test core logic without external dependencies
go test -v -short ./...

# Run integration tests
# These test full system with actual database and NATS connections
go test -v ./...
```

Key test areas:
- Currency conversion logic
- Event enrichment pipeline
- Data aggregation
- Metrics collection

#### Currency Enricher
- Converts amounts to EUR using exchangerate.host API
- Caches rates for 1 minute
- Rate limited to 1 request/minute
- Graceful error handling

#### Player Enricher
- Looks up player data from Postgres
- No caching (per requirements)
- Handles missing players gracefully
- Adds email and last_signed_in_at

#### Description Enricher
- Generates human-friendly descriptions
- Currency-specific formatting:
  - USD/EUR: 2 decimal places
  - BTC: 3 decimal places
- Uses game title mapping

## Metrics

The system collects the following metrics:
- Events Processed: Total events received
- Events Enriched: Successfully enriched events
- Enrichment Errors: Failed enrichment attempts
- Processing Time: Average processing time per event

## Example Events

### Input Event
```json
{
  "id": 1,
  "player_id": 123,
  "game_id": 100,
  "type": "bet",
  "amount": 1000,
  "currency": "USD",
  "has_won": true,
  "created_at": "2024-02-24T10:48:10Z"
}
```

### Enriched Event
```json
{
  "id": 1,
  "player_id": 123,
  "game_id": 100,
  "type": "bet",
  "amount": 1000,
  "currency": "USD",
  "has_won": true,
  "created_at": "2024-02-24T10:48:10Z",
  "amount_eur": 910,
  "player": {
    "email": "player123@example.com",
    "last_signed_in_at": "2024-02-24T10:48:10Z"
  },
  "description": "Player 123 won USD 10.00 in Book of Dead"
}
```

## Error Handling

The system implements graceful error handling:
1. Individual enricher failures don't stop processing
2. Failed enrichments are logged and counted
3. Missing data is handled gracefully:
   - Missing player data: logged, continues processing
   - Missing exchange rates: retries with backoff
   - Missing game titles: uses default format

## Performance Considerations

1. Currency Conversion
   - Caches rates for 1 minute
   - Rate limited to avoid API throttling
   - Uses atomic operations for thread safety

2. Player Data
   - No caching per requirements
   - Uses prepared statements
   - Connection pooling via sql.DB

3. Description Generation
   - In-memory game title mapping
   - Efficient string formatting
   - No external dependencies

## Monitoring

The system provides metrics for monitoring:
```go
type Metrics struct {
    EventsProcessed   uint64
    EventsEnriched    uint64
    EnrichmentErrors  uint64
    ProcessingTimeMs  uint64
}
```

These metrics can be used to:
1. Monitor system health
2. Track error rates
3. Measure performance
4. Alert on issues

### Health Checks

The system provides HTTP endpoints for monitoring:

```bash
# Check system health
curl http://localhost:8080/health

# Get metrics
curl http://localhost:8080/metrics
```

Example health check response:
```json
{
  "healthy": true,
  "components": {
    "nats": "connected",
    "database": "connected"
  },
  "timestamp": "2024-02-24T12:34:56Z"
}
```

### Metrics

The system collects the following metrics:
- Events Processed: Total events received
- Events Enriched: Successfully enriched events
- Enrichment Errors: Failed enrichment attempts
- Processing Time: Average processing time per event

### Aggregates

The system maintains real-time aggregates:
- Total bets in EUR
- Total deposits in EUR
- Total wins in EUR
- Unique active users
- Active games and players

Access aggregates via HTTP:
```bash
curl http://localhost:8080/aggregates
```

Example response:
```json
{
  "TotalBetsEUR": 15000,
  "TotalDepositsEUR": 50000,
  "TotalWinsEUR": 12000,
  "UniqueUsers": 42,
  "ActiveGames": {
    "100": 5,
    "101": 3
  }
}
```

### Materialized Data

The system materializes real-time statistics:
- Total number of events
- Events per minute
- Events per second (moving average)
- Top players by:
  - Number of bets
  - Number of wins
  - Total deposits in EUR

Access via HTTP:
```bash
curl http://localhost:8080/materialized
```

Example response:
```json
{
  "events_total": 12345,
  "events_per_minute": 123.45,
  "events_per_second_moving_average": 3.12,
  "top_player_bets": {
    "id": 10,
    "count": 150
  },
  "top_player_wins": {
    "id": 11,
    "count": 50
  },
  "top_player_deposits": {
    "id": 12,
    "count": 15000
  }
}
```

### Metrics Visualization

The system provides metrics visualization through Grafana:

1. Access Grafana UI: http://localhost:3000
   - Username: admin
   - Password: admin

2. Available dashboards:
   - Casino Events
     - Events per Second
     - Total Events Processed
     - Top Players
     - Error Rates

3. Metrics are collected via Prometheus:
   - Event processing metrics
   - Player statistics
   - System health
   - Performance metrics

4. Raw metrics are available at:
   - http://localhost:8080/metrics
   - This endpoint is scraped by Prometheus and used by Grafana
   - You can inspect raw metrics to understand what data is available 