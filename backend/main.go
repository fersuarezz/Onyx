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
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	err := run()
	if err != nil {
		slog.Error("Error en el main", "err", err)
		os.Exit(1)
	}
}

func run() error {
	DATABASE_URL := os.Getenv("DATABASE_URL")
	if DATABASE_URL == "" {
		return fmt.Errorf("DATABASE_URL not set")
	}

	MEDIA_ROOT := getenv("MEDIA_ROOT", "./media")
	PORT := getenv("PORT", "8000")

	err := migrations.Run(DATABASE_URL)
	if err != nil {
		return fmt.Errorf("aplicando migraciones iniciales: %w", err)
	}

	ctx := context.Background()
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

	return nil
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
