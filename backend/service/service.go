package service

import (
	"context"
	"fmt"
	"onyx/core"
	"onyx/core/db"
)

type DiaryService struct {
	Diary db.DiaryDataBase
}

func (d *DiaryService) Insert(ctx context.Context, entry core.DiaryEntry) (int, error) {
	if entry.Date.IsZero() || (entry.Text == "" && len(entry.Attachments) == 0) {
		return -1, fmt.Errorf("%w: entry inválido", ErrInvalidInput)
	}

	id, err := d.Diary.Insert(ctx, entry)

	if err != nil {
		return -1, fmt.Errorf("%w: %w", ErrDatabase, err)
	}

	return id, nil
}

func (d *DiaryService) GetByDate(ctx context.Context, date core.Date, ignoreYear bool) ([]core.DiaryEntry, error) {
	if date.IsZero() {
		return nil, fmt.Errorf("%w: fecha inválida", ErrInvalidInput)
	}

	entries, err := d.Diary.GetByDate(ctx, date, ignoreYear)

	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDatabase, err)
	}

	return entries, nil
}
