package db

import (
	"database/sql"
	_ "embed"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed schema.sql
var schemaSQL string

// Open connects to PostgreSQL, sets pool limits, and runs the schema.
func Open(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}