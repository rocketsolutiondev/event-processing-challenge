package player

import (
	"context"
	"time"
)

// mockDB implements DB interface for testing
type mockDB struct {
	queryFunc func(ctx context.Context, query string, args ...interface{}) Scanner
}

func (m *mockDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) Scanner {
	return m.queryFunc(ctx, query, args...)
}

func (m *mockDB) Close() error {
	return nil
}

type mockRow struct {
	email    string
	signedIn time.Time
	err      error
}

func (m *mockRow) Scan(dest ...interface{}) error {
	if m.err != nil {
		return m.err
	}
	emailPtr := dest[0].(*string)
	timePtr := dest[1].(*time.Time)
	*emailPtr = m.email
	*timePtr = m.signedIn
	return nil
} 