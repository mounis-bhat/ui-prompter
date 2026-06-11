package db

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"

	"ui-prompter/internal/db/queries"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

type Database struct {
	DB      *sql.DB
	Queries *queries.Queries
}

func NewDatabase(ctx context.Context, dsn string) (*Database, error) {
	// For modernc.org/sqlite, the driver name is "sqlite"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return nil, fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return nil, fmt.Errorf("failed to run goose migrations: %w", err)
	}

	q := queries.New(db)

	return &Database{
		DB:      db,
		Queries: q,
	}, nil
}
