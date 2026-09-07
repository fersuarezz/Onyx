package main

import (
	"context"
	"fmt"
	"log/slog"
	"onyx/api"
	"onyx/core"
	"onyx/core/db"
	"onyx/core/storage"
	"onyx/migrations"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err := run(ctx)
	if err != nil {
		slog.Error("in main function: ", "err", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	DATABASE_URL := os.Getenv("DATABASE_URL")
	if DATABASE_URL == "" {
		return fmt.Errorf("DATABASE_URL not set")
	}

	MEDIA_ROOT := getenv("MEDIA_ROOT", "./media")
	PORT := getenv("PORT", "8080")

	err := migrations.Run(DATABASE_URL)
	if err != nil {
		return fmt.Errorf("aplicando migraciones iniciales: %w", err)
	}

	pool, err := pgxpool.New(ctx, DATABASE_URL)
	if err != nil {
		return fmt.Errorf("creando el pool inicial: %w", err)
	}
	defer pool.Close()

	ctxPing, cancelPing := context.WithTimeout(ctx, time.Second*5)
	defer cancelPing()
	err = pool.Ping(ctxPing) // pool.New() es lazy
	if err != nil {
		return fmt.Errorf("conectandose al pool: %w", err)
	}

	fileStorage := composeStorage(MEDIA_ROOT)

	diaryDB := composeDiaryDatabase(pool)
	diaryHandler := api.NewDiaryHandler(diaryDB, fileStorage)

	app := api.CreateApp(diaryHandler)

	errCh := make(chan error, 1) // Size 1 para evitar bloqueo
	go func() {
		errCh <- app.Listen(":" + PORT)
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("servidor: %w", err)
	case <-ctx.Done():
		slog.Info("apagando servidor...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		return app.ShutdownWithContext(shutdownCtx)
	}
}

func getenv(key string, def string) string {
	env := os.Getenv(key)
	if env == "" {
		return def
	}

	return env
}

// COMPOSERS: Añadir variable en .env en un futuro para elegir
func composeStorage(mediaRoot string) storage.Storage {
	return storage.NewFileSystemStorage(mediaRoot)
}

func composeDiaryDatabase(pool core.Querier) db.DiaryDataBase {
	return db.NewPostgresDiary(pool)
}
