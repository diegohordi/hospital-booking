// Package database contains useful functions to handle database operations, as create connections,
// close resources and also helpers to parse result into structs.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"hospital-booking/internal/configs"
	"log"
	"reflect"
	"time"

	_ "github.com/lib/pq"
)

type defaultConnection struct {
	db *sql.DB
}

// Connection holds a DB instance.
type Connection interface {
	DB() *sql.DB
	CreateContext(ctx context.Context) (context.Context, context.CancelFunc)
	Close()
}

// DB gets the DB instance associated to the connection.
func (d *defaultConnection) DB() *sql.DB {
	return d.db
}

// CreateContext creates a new context based on the given one, with a default timeout.
func (d *defaultConnection) CreateContext(ctx context.Context) (context.Context, context.CancelFunc) {
	timeout := 5 * time.Second
	return context.WithTimeout(ctx, timeout)
}

// NewConnection creates a new DB instance based on the given configurations.
func NewConnection(config configs.Config) (Connection, error) {
	db, err := sql.Open(config.DatabaseDriver(), config.DatabaseDSN())
	if err != nil {
		return nil, fmt.Errorf("could not create a connection: %w", err)
	}
	db.SetConnMaxLifetime(time.Minute * 3)
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("database is not reachable: %w", err)
	}
	return &defaultConnection{db: db}, nil
}

// Close closes the DB connection.
func (d *defaultConnection) Close() {
	if err := d.DB().Close(); err != nil {
		log.Printf("could not close the database connection %v\n", err)
		return
	}
	log.Printf("database connection released succesfully")
}

// CloseRows closes the given rows.
func CloseRows(rows *sql.Rows) {
	if err := rows.Close(); err != nil {
		log.Printf("could not close the given rows %v\n", err)
	}
}

// TransformRow transforms the current row given by the into the given struct.
// The transformation is performed by reflection, using a field tag called dbfield for that.
func TransformRow(rows *sql.Rows, model interface{}) error {
	modelType := reflect.TypeOf(model).Elem()
	modelValue := reflect.ValueOf(model)
	columns, err := rows.Columns()
	values := make([]interface{}, 0)
	if err != nil {
		return err
	}
	for _, column := range columns {
		for i := 0; i < modelType.NumField(); i++ {
			field := modelType.Field(i)
			dbfield := field.Tag.Get("dbfield")
			if dbfield != column {
				continue
			}
			values = append(values, modelValue.Elem().Field(i).Addr().Interface())
		}
	}
	if err = rows.Scan(values...); err != nil {
		return err
	}
	return nil
}
