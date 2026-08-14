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

	err = insertAttachments(ctx, tx, id, entry.Keys)

	if err != nil {
		return 0, err
	}

	return id, tx.Commit(ctx)
}

func (db *PostgresDiary) GetByDate(ctx context.Context, date core.Date, ignoreYear bool) ([]core.DiaryEntry, error) {
	query := ``
	args := []any{}

	if ignoreYear {
		query = `	SELECT d.id, d.date, d.text, COALESCE(array_agg(a.key ORDER BY a.key) FILTER (WHERE a.key is not NULL), '{}') AS keys
				FROM diary_entries AS d 
				LEFT JOIN attachments AS a ON d.id = a.entry_id 
				WHERE EXTRACT(DAY FROM d.date) = $1
				AND EXTRACT(MONTH FROM d.date) = $2
				GROUP BY d.id, d.date, d.text
				ORDER BY d.date DESC, d.id DESC`
		args = []any{date.Day, date.Month}
	} else {
		query = `	SELECT d.id, d.date, d.text, COALESCE(array_agg(a.key ORDER BY a.key) FILTER (WHERE a.key is not NULL), '{}') AS keys
				FROM diary_entries AS d 
				LEFT JOIN attachments AS a ON d.id = a.entry_id 
				WHERE d.date = $1
				GROUP BY d.id, d.date, d.text
				ORDER BY d.date DESC, d.id DESC`
		args = []any{date}
	}

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

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func (db *PostgresDiary) Search(ctx context.Context, text string, limit int) ([]core.DiaryEntry, error) {
	query := `	SELECT d.id, d.date, d.text, COALESCE(array_agg(a.key ORDER BY a.key) FILTER (WHERE a.key is not NULL), '{}') AS keys
				FROM diary_entries AS d 
				LEFT JOIN attachments AS a ON d.id = a.entry_id 
				WHERE d.text ILIKE $1
				GROUP BY d.id, d.date, d.text
				ORDER BY d.date DESC, d.id DESC
				LIMIT $2
			`

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

func (db *PostgresDiary) Delete(ctx context.Context, id int) ([]string, error) {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("pool.Begin: %w", err)
	}

	defer tx.Rollback(ctx)

	query := `SELECT key
			FROM attachments AS a
			WHERE a.entry_id = $1`

	rows, err := tx.Query(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("recuperando las keys de las filas para borrar: %w", err)
	}

	keys, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("parseando las keys de las filas para borrar: %w", err)
	}

	query = `DELETE FROM diary_entries as d WHERE d.id = $1`

	tag, err := tx.Exec(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("borrando las entradas: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, fmt.Errorf("commiteando tx: %w", err)
	}

	return keys, nil
}

func (db *PostgresDiary) Edit(ctx context.Context, entry core.DiaryEntry) ([]string, error) {
	tx, err := db.pool.Begin(ctx)

	if err != nil {
		return nil, fmt.Errorf("pool.Begin: %w", err)
	}

	id := entry.ID

	defer tx.Rollback(ctx)

	query := `UPDATE diary_entries SET date = $1, text = $2 WHERE id = $3`

	tag, err := tx.Exec(ctx, query, entry.Date, entry.Text, id)
	if err != nil {
		return nil, fmt.Errorf("update diary entry: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return nil, ErrNotFound
	}

	// Update attachments
	query = `DELETE FROM attachments WHERE entry_id = $1 RETURNING key`

	rows, err := tx.Query(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("delete entry attachments: %w", err)
	}

	orphans, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("parsing orphans: %w", err)
	}

	err = insertAttachments(ctx, tx, id, entry.Keys)
	if err != nil {
		return nil, err
	}

	return orphans, tx.Commit(ctx)
}

func (db *PostgresDiary) AllAttachments(ctx context.Context) ([]string, error) {
	query := `SELECT key FROM attachments`

	rows, err := db.pool.Query(ctx, query)

	if err != nil {
		return nil, fmt.Errorf("getting attachments: %w", err)
	}

	atts, err := pgx.CollectRows(rows, pgx.RowTo[string])

	if err != nil {
		return nil, fmt.Errorf("parsing rows to []string: %w", err)
	}

	return atts, nil
}

func insertAttachments(ctx context.Context, tx pgx.Tx, id int, attachments []string) error {
	query := `INSERT INTO attachments (entry_id, key) values ($1, $2)`
	for _, att := range attachments {
		_, err := tx.Exec(ctx, query, id, att)
		if err != nil {
			return fmt.Errorf("insert attachment: %w", err)
		}
	}

	return nil
}
