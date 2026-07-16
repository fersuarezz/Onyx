package db

import (
	"context"
	"fmt"
	"onyx/core"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresDiary struct {
	pool *pgxpool.Pool
}

func (db *PostgresDiary) Insert(ctx context.Context, entry core.DiaryEntry) (int, error) {
	tx, err := db.pool.Begin(ctx)

	if err != nil {
		return 0, fmt.Errorf("pool.Begin: %w", err)
	}

	defer tx.Rollback(ctx)

	query := `INSERT INTO diary_entries (date, text) VALUES ($1, $2) RETURNING id`

	var id int
	err = tx.QueryRow(ctx, query, entry.Date, entry.Text).Scan(&id)

	if err != nil {
		return 0, fmt.Errorf("insert diary entry: %w", err)
	}

	query = `INSERT INTO attachments (entry_id, dir) values ($1, $2)`
	for _, att := range entry.Attachments {
		_, err := tx.Exec(ctx, query, id, att)
		if err != nil {
			return 0, fmt.Errorf("insert attachment: %w", err)
		}
	}

	return id, tx.Commit(ctx)
}
