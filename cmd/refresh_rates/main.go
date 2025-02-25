package main

import (
    "database/sql"
    "flag"
    "fmt"
    "log"
    "os"
    "strings"
    "time"
    _ "github.com/lib/pq"
    "github.com/joho/godotenv"
    "github.com/Bitstarz-eng/event-processing-challenge/internal/enricher/exchange"
)

func main() {
    refresh := flag.Bool("refresh", false, "Refresh rates from API")
    flag.Parse()

    // Load .env file
    if err := godotenv.Load(); err != nil {
        log.Printf("Warning: Error loading .env file: %v", err)
    }

    // Connect to database
    dbURL := fmt.Sprintf(
        "host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
        os.Getenv("DB_HOST"),
        os.Getenv("DB_PORT"),
        os.Getenv("DB_USER"),
        os.Getenv("DB_PASSWORD"),
        os.Getenv("DB_NAME"),
        os.Getenv("DB_SSL_MODE"),
    )

    db, err := sql.Open("postgres", dbURL)
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    defer db.Close()

    // Create exchange service
    svc, err := exchange.New(db)
    if err != nil {
        log.Fatalf("Failed to create exchange service: %v", err)
    }

    // Refresh rates if requested
    if *refresh {
        if err := svc.RefreshRates(); err != nil {
            log.Fatalf("Failed to refresh rates: %v", err)
        }
    }

    // Display current rates
    rows, err := db.Query(`
        SELECT currency, rate_to_eur, updated_at 
        FROM exchange_rates 
        ORDER BY currency
    `)
    if err != nil {
        log.Fatalf("Failed to query rates: %v", err)
    }
    defer rows.Close()

    fmt.Println("\nCurrent exchange rates:")
    fmt.Printf("%-10s %-15s %-25s\n", "Currency", "Rate to EUR", "Last Updated")
    fmt.Println(strings.Repeat("-", 50))

    for rows.Next() {
        var currency string
        var rate float64
        var updatedAt time.Time
        if err := rows.Scan(&currency, &rate, &updatedAt); err != nil {
            log.Printf("Error scanning row: %v", err)
            continue
        }
        fmt.Printf("%-10s %-15.6f %-25s\n", currency, rate, updatedAt.Format(time.RFC3339))
    }
} 