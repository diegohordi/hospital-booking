// Package mock contains utilities for tests.
package mock

import (
	"context"
	"database/sql"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// Connection is the mock version for database.Connection.
type Connection struct {
	db      *sql.DB
	SQLMock sqlmock.Sqlmock
}

func (m Connection) CreateContext(ctx context.Context) (context.Context, context.CancelFunc) {
	timeout := 5 * time.Second
	return context.WithTimeout(ctx, timeout)
}

func (m Connection) DB() *sql.DB {
	return m.db
}

func (m Connection) Close() {
	_ = m.DB().Close()
}

func MustCreateConnectionMock() Connection {
	db, mock, err := sqlmock.New()
	if err != nil {
		panic(err)
	}
	return Connection{
		db:      db,
		SQLMock: mock,
	}
}

type DBResultOption func(dbConn Connection)

func MockDBResults(dbConn Connection, opts ...DBResultOption) {
	for _, opt := range opts {
		opt(dbConn)
	}
}
