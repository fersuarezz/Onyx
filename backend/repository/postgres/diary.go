package postgres

import (
	"context"
	"fmt"
	"onyx/domain"
	"onyx/service/diary"
	"strings"

	"github.com/jackc/pgx/v5"
)

type Querier interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func NewDiaryRepository(pool Querier) *DiaryRepository {
	return &DiaryRepository{pool: pool}
}

type DiaryRepository struct {
	pool Querier
}

var _ diary.Repository = (*DiaryRepository)(nil)

func (db *DiaryRepository) Insert(ctx context.Context, entry diary.Entry) (int, error) {
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

func (db *DiaryRepository) GetByDate(ctx context.Context, date domain.Date, ignoreYear bool) ([]diary.Entry, error) {
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

	entries, err := pgx.CollectRows(rows, pgx.RowToStructByName[diary.Entry])
	if err != nil {
		return nil, fmt.Errorf("parsing rows to struct: %w", err)
	}

	return entries, nil
}

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func (db *DiaryRepository) Search(ctx context.Context, text string, limit int, offset int) ([]diary.Entry, bool, error) {
	query := `	SELECT d.id, d.date, d.text, COALESCE(array_agg(a.key ORDER BY a.key) FILTER (WHERE a.key is not NULL), '{}') AS keys
				FROM diary_entries AS d 
				LEFT JOIN attachments AS a ON d.id = a.entry_id 
				WHERE d.text ILIKE $1
				GROUP BY d.id, d.date, d.text
				ORDER BY COALESCE(d.date, d.created_at::date) DESC, d.id DESC
				LIMIT $2 OFFSET $3
			`

	rows, err := db.pool.Query(ctx, query, "%"+likeEscaper.Replace(text)+"%", limit+1, offset)
	if err != nil {
		return nil, false, fmt.Errorf("querying diary entries: %w", err)
	}

	entries, err := pgx.CollectRows(rows, pgx.RowToStructByName[diary.Entry])
	if err != nil {
		return nil, false, fmt.Errorf("parsing rows to struct: %w", err)
	}

	hasMore := len(entries) > limit
	if hasMore {
		entries = entries[:limit]
	}

	return entries, hasMore, nil
}

func (db *DiaryRepository) Delete(ctx context.Context, id int) ([]string, error) {
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
		return nil, domain.ErrNotFound
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, fmt.Errorf("commiteando tx: %w", err)
	}

	return keys, nil
}

func (db *DiaryRepository) Edit(ctx context.Context, id int, date *domain.Date, text *string, keys []string) ([]string, error) {
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("pool.Begin: %w", err)
	}
	defer tx.Rollback(ctx)

	// Update date and text
	query := `UPDATE diary_entries SET date = COALESCE($1, date), text = COALESCE($2, text) WHERE id = $3`

	tag, err := tx.Exec(ctx, query, date, text, id)
	if err != nil {
		return nil, fmt.Errorf("update diary entry: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, domain.ErrNotFound
	}

	// Update attachments
	if keys == nil {
		return nil, tx.Commit(ctx)
	}

	query = `DELETE FROM attachments WHERE entry_id = $1 RETURNING key`

	rows, err := tx.Query(ctx, query, id)
	if err != nil {
		return nil, fmt.Errorf("delete entry attachments: %w", err)
	}

	orphans, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("parsing orphans: %w", err)
	}

	err = insertAttachments(ctx, tx, id, keys)
	if err != nil {
		return nil, err
	}

	return orphans, tx.Commit(ctx)
}

func (db *DiaryRepository) AllAttachments(ctx context.Context, limit int, offset int) ([]string, bool, error) {
	query := `SELECT a.key FROM attachments AS a
			JOIN diary_entries AS d ON a.entry_id = d.id
			ORDER BY COALESCE(d.date, d.created_at::date) DESC, d.id DESC, a.key
			LIMIT $1 OFFSET $2`

	rows, err := db.pool.Query(ctx, query, limit+1, offset)

	if err != nil {
		return nil, false, fmt.Errorf("getting attachments: %w", err)
	}

	atts, err := pgx.CollectRows(rows, pgx.RowTo[string])

	if err != nil {
		return nil, false, fmt.Errorf("parsing rows to []string: %w", err)
	}

	hasMore := len(atts) > limit
	if hasMore {
		atts = atts[:limit]
	}

	return atts, hasMore, nil
}

func (db *DiaryRepository) GetAtemporal(ctx context.Context, limit int, offset int) ([]diary.Entry, bool, error) {
	query := `	SELECT d.id, d.date, d.text, COALESCE(array_agg(a.key ORDER BY a.key) FILTER (WHERE a.key is not NULL), '{}') AS keys
				FROM diary_entries AS d 
				LEFT JOIN attachments AS a ON d.id = a.entry_id 
				WHERE d.date IS NULL
				GROUP BY d.id, d.date, d.text
				ORDER BY d.created_at DESC, d.id DESC
				LIMIT $1 OFFSET $2`

	rows, err := db.pool.Query(ctx, query, limit+1, offset)
	if err != nil {
		return nil, false, fmt.Errorf("querying diary entries: %w", err)
	}

	entries, err := pgx.CollectRows(rows, pgx.RowToStructByName[diary.Entry])
	if err != nil {
		return nil, false, fmt.Errorf("parsing rows to struct: %w", err)
	}

	hasMore := len(entries) > limit
	if hasMore {
		entries = entries[:limit]
	}

	return entries, hasMore, nil
}

func (db *DiaryRepository) GetById(ctx context.Context, id int) (diary.Entry, error) {
	query := `	SELECT d.id, d.date, d.text, COALESCE(array_agg(a.key ORDER BY a.key) FILTER (WHERE a.key is not NULL), '{}') AS keys
				FROM diary_entries AS d 
				LEFT JOIN attachments AS a ON d.id = a.entry_id 
				WHERE d.id = $1
				GROUP BY d.id, d.date, d.text`

	rows, err := db.pool.Query(ctx, query, id)
	if err != nil {
		return diary.Entry{}, fmt.Errorf("querying diary entries: %w", err)
	}

	entry, err := pgx.CollectRows(rows, pgx.RowToStructByName[diary.Entry])
	if err != nil {
		return diary.Entry{}, fmt.Errorf("parsing rows to struct: %w", err)
	}

	if len(entry) == 0 {
		return diary.Entry{}, domain.ErrNotFound
	}

	return entry[0], nil
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
