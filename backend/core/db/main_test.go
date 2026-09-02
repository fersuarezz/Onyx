package db

import (
	"context"
	"fmt"
	"onyx/migrations"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	os.Exit(Run(m))
}

func Run(m *testing.M) int {
	ctx := context.Background()

	// Create container
	ctr, err := postgres.Run(ctx,
		"postgres:17-alpine",
		postgres.WithDatabase("onyx_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		postgres.BasicWaitStrategies(),
	)

	defer func() {
		err := testcontainers.TerminateContainer(ctr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "terminando contenedor: %v\n", err)
		}
	}()

	if err != nil {
		fmt.Fprintf(os.Stderr, "arrancando postgres: %v\n", err)
		return 1
	}

	connStr, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Fprintf(os.Stderr, "connection string: %v\n", err)
		return 1
	}

	fmt.Println("CONNECTION: ", connStr)

	// Migrations
	err = migrations.Run(connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "aplicando migraciones: %v\n", err)
		return 1
	}

	// Create pool
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "creando el pool: %v\n", err)
		return 1
	}

	defer pool.Close()

	testPool = pool

	// Exec tests
	return m.Run()
}
