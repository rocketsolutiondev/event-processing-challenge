package player

import (
    "context"
    "database/sql"
)

// DB interface allows us to mock the database for testing
type DB interface {
    QueryRowContext(ctx context.Context, query string, args ...interface{}) Scanner
    Close() error
}

// Scanner interface matches sql.Row's Scan method
type Scanner interface {
    Scan(dest ...interface{}) error
}

// sqlDB adapts sql.DB to our DB interface
type sqlDB struct {
    *sql.DB
}

func (db *sqlDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) Scanner {
    return db.DB.QueryRowContext(ctx, query, args...)
}

func (db *sqlDB) Close() error {
    return db.DB.Close()
} 