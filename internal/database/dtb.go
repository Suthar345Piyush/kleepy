// make a new db connection, close the connection and health check  for db

// using go sqlite driver with cgo free - modernc.org/sqlite

package database

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

// new connection to db

func NewConn(connectionString string) (*DB, error) {

	db, err := sql.Open("sqlite", connectionString)

	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	// connections metrics

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Second)

	// ping the database

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to db: %w", err)
	}

	fmt.Println("database connection established")

	return &DB{db}, nil

}

// connection close func

func (db *DB) Close() error {
	return db.Close()
}

// db health check

func (db *DB) Health() error {
	return db.Ping()
}
