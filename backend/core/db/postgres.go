package db

import (
	"context"
	"fmt"
	"onyx/core"
	"strings"

	"github.com/jackc/pgx/v5"
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

func (db *PostgresDiary) GetByDate(ctx context.Context, day int, month int, year int) ([]core.DiaryEntry, error) {
	query := `	SELECT d.id, d.date, d.text, COALESCE(array_agg(a.dir) FILTER (WHERE a.dir is not NULL), '{}') AS attachments
				FROM diary_entries AS d 
				LEFT JOIN attachments AS a ON d.id = a.entry_id 
				WHERE EXTRACT(DAY FROM d.date) = $1
				AND EXTRACT(MONTH FROM d.date) = $2`

	args := []any{day, month}
	if year != 0 {
		query += ` AND EXTRACT(YEAR FROM d.date) = $3`
		args = append(args, year)
	}

	query += ` GROUP BY d.id, d.date, d.text`

	rows, err := db.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying diary entries: %w", err)
	}

	entries, err := pgx.CollectRows(rows, pgx.RowToStructByName[core.DiaryEntry])
	if err != nil {
		return nil, fmt.Errorf("parsing rows to struct: %w", err)
	}

	return entries, nil
}

func (db *PostgresDiary) Search(ctx context.Context, text string, limit int) ([]core.DiaryEntry, error) {
	query := `	SELECT d.id, d.date, d.text, COALESCE(array_agg(a.dir) FILTER (WHERE a.dir is not NULL), '{}') AS attachments
				FROM diary_entries AS d 
				LEFT JOIN attachments AS a ON d.id = a.entry_id 
				WHERE d.text ILIKE $1
				GROUP BY d.id, d.date, d.text
				ORDER BY d.date DESC
				LIMIT $2
			`

	var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

	rows, err := db.pool.Query(ctx, query, "%"+likeEscaper.Replace(text)+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("querying diary entries: %w", err)
	}

	entries, err := pgx.CollectRows(rows, pgx.RowToStructByName[core.DiaryEntry])
	if err != nil {
		return nil, fmt.Errorf("parsing rows to struct: %w", err)
	}

	return entries, nil
}
