package core

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type DiaryEntry struct {
	ID   int      `json:"id"`
	Date Date     `json:"date"`
	Text string   `json:"text"`
	Keys []string `json:"keys"` // Rutas en el servidor de los adjuntos (sin el sufijo _full, _thumbnail)
}

type Querier interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}
