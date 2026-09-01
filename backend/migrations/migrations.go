package migrations

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed *.sql
var migrationFiles embed.FS

func Run(dbURL string) error {
	source, err := iofs.New(migrationFiles, ".")
	if err != nil {
		return fmt.Errorf("iofs.New: %w", err)
	}

	// Creates db handle
	db, err := sql.Open("pgx/v5", dbURL)
	if err != nil {
		return fmt.Errorf("sql.Open: %w", err)
	}
	defer db.Close()

	// Interface wrapper
	driver, err := migratepgx.WithInstance(db, &migratepgx.Config{})
	if err != nil {
		return fmt.Errorf("migratepgx.WithInstance: %w", err)
	}

	// Orchestrator
	m, err := migrate.NewWithInstance("iofs", source, "pgx5", driver)
	if err != nil {
		return fmt.Errorf("migrate.NewWithInstance: %w", err)
	}

	// Apply migrations
	err = m.Up()
	if err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			return fmt.Errorf("migrate.Up: %w", err)
		}
	}

	return nil
}
