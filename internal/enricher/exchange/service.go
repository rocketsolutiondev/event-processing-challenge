package exchange

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Service struct {
	db            *sql.DB
	apiKey        string
	apiURL        string
	memoryCacheDuration time.Duration
	sourceCurrency string
	rates         map[string]float64
	lastUpdate    time.Time
	mu            sync.RWMutex
}

type APIResponse struct {
	Success bool               `json:"success"`
	Source  string            `json:"source"`
	Quotes  map[string]float64 `json:"quotes"`
}

func New(db *sql.DB) (*Service, error) {
	memoryCacheDuration, err := time.ParseDuration(os.Getenv("EXCHANGE_RATE_MEMORY_CACHE_DURATION"))
	if err != nil {
		memoryCacheDuration = time.Minute // default 1m
	}

	log.Printf("Exchange service initialized with memory cache: %v", memoryCacheDuration)

	return &Service{
		db:            db,
		apiKey:        os.Getenv("EXCHANGE_RATE_API_KEY"),
		apiURL:        os.Getenv("EXCHANGE_RATE_API_URL"),
		memoryCacheDuration: memoryCacheDuration,
		sourceCurrency: os.Getenv("EXCHANGE_RATE_SOURCE_CURRENCY"),
		rates:         make(map[string]float64),
	}, nil
}

// RefreshRates forces an update of rates from the API
func (s *Service) RefreshRates() error {
	log.Println("Refreshing exchange rates from API...")
	return s.updateRates()
}

func (s *Service) GetRate(currency string) (float64, error) {
	if currency == s.sourceCurrency {
		return 1.0, nil
	}

	// Try memory cache first
	s.mu.RLock()
	if time.Since(s.lastUpdate) < s.memoryCacheDuration {
		if rate, ok := s.rates[currency]; ok {
			s.mu.RUnlock()
			return rate, nil
		}
	}
	s.mu.RUnlock()

	// Try database
	var rate float64
	var updatedAt time.Time
	err := s.db.QueryRow(
		`SELECT rate_to_eur, updated_at 
		 FROM exchange_rates 
		 WHERE currency = $1`, 
		currency,
	).Scan(&rate, &updatedAt)

	// Log rate from database
	log.Printf("Got rate for %s: %.10f (updated: %s)", 
		currency, rate, updatedAt.Format(time.RFC3339))

	if err == nil {
		// Store in memory cache
		s.mu.Lock()
		s.rates[currency] = rate
		s.lastUpdate = time.Now()
		s.mu.Unlock()
		return rate, nil
	}

	// Rate not found, try API once
	if err := s.RefreshRates(); err != nil {
		return 0, fmt.Errorf("no rate found for currency %s and API refresh failed: %w", currency, err)
	}

	s.mu.RLock()
	rate, ok := s.rates[currency]
	s.mu.RUnlock()
	if !ok {
		return 0, fmt.Errorf("no rate found for currency %s even after API refresh", currency)
	}

	return rate, nil
}

func (s *Service) updateRates() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	url := fmt.Sprintf("%s?access_key=%s&source=%s", s.apiURL, s.apiKey, s.sourceCurrency)
	log.Printf("Fetching rates from: %s", url)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to get rates: %w", err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if !apiResp.Success {
		return fmt.Errorf("API request failed")
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update database and memory cache
	for key, value := range apiResp.Quotes {
		currency := strings.TrimPrefix(key, s.sourceCurrency)
		s.rates[currency] = value

		_, err = tx.Exec(
			`INSERT INTO exchange_rates (currency, rate_to_eur, updated_at) 
			 VALUES ($1, $2, NOW())
			 ON CONFLICT (currency) 
			 DO UPDATE SET rate_to_eur = $2, updated_at = NOW()`,
			currency, value,
		)
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.lastUpdate = time.Now()
	return nil
} 