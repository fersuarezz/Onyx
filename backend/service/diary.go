package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"onyx/core"
	"onyx/core/db"
	"onyx/core/storage"

	"github.com/google/uuid"
)

type DiaryService struct {
	db      db.DiaryDataBase
	storage storage.Storage
}

func NewDiaryService(db db.DiaryDataBase, storage storage.Storage) *DiaryService {
	return &DiaryService{db: db, storage: storage}
}

func (d *DiaryService) Insert(ctx context.Context, date core.Date, text string, srcs []io.ReadSeeker) (int, error) {
	if text == "" && len(srcs) == 0 {
		return -1, fmt.Errorf("%w: entry inválido", core.ErrInvalidInput)
	}

	datePointer := &date
	if date.IsZero() {
		datePointer = nil
	}

	keys, err := d.storage.SaveAttachments(srcs)
	if err != nil {
		return -1, err
	}

	entry := core.DiaryEntry{
		Date: datePointer,
		Text: text,
		Keys: keys,
	}

	id, err := d.db.Insert(ctx, entry)
	if err != nil {
		d.storage.DeleteAttachmentFolders(keys)
		return -1, fmt.Errorf("%w: %w", core.ErrDatabase, err)
	}

	return id, nil
}

func (d *DiaryService) GetByDate(ctx context.Context, date core.Date, ignoreYear bool) ([]core.DiaryEntry, error) {
	if date.IsZero() {
		return nil, fmt.Errorf("%w: fecha inválida", core.ErrInvalidInput)
	}

	entries, err := d.db.GetByDate(ctx, date, ignoreYear)

	if err != nil {
		return nil, fmt.Errorf("%w: %w", core.ErrDatabase, err)
	}

	return entries, nil
}

func (d *DiaryService) Search(ctx context.Context, text string, limit int, offset int) ([]core.DiaryEntry, bool, error) {
	if limit < 1 || limit > 100 {
		return nil, false, fmt.Errorf("%w: limit debe estar entre 1 y 100", core.ErrInvalidInput)
	}
	if offset < 0 {
		return nil, false, fmt.Errorf("%w: offset debe ser positivo", core.ErrInvalidInput)
	}

	entries, hasMore, err := d.db.Search(ctx, text, limit, offset)
	if err != nil {
		return nil, false, fmt.Errorf("%w: %w", core.ErrDatabase, err)
	}

	return entries, hasMore, nil
}

func (d *DiaryService) Delete(ctx context.Context, id int) error {
	keys, err := d.db.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return core.ErrNotFound
		}

		return fmt.Errorf("%w: %w", core.ErrDatabase, err)
	}

	d.storage.DeleteAttachmentFolders(keys)

	return nil
}

func (d *DiaryService) Edit(ctx context.Context, id int, date *core.Date, text *string, srcs []io.ReadSeeker) error {
	if text == nil && len(srcs) == 0 && date == nil {
		return fmt.Errorf("%w: entry inválido, nada que editar", core.ErrInvalidInput)
	}

	var keys []string
	var err error
	if len(srcs) != 0 {
		keys, err = d.storage.SaveAttachments(srcs)
		if err != nil {
			return err
		}
	}

	orphaned, err := d.db.Edit(ctx, id, date, text, keys)
	if err != nil {
		d.storage.DeleteAttachmentFolders(keys)

		if errors.Is(err, core.ErrNotFound) {
			return core.ErrNotFound
		}

		return fmt.Errorf("%w: %w", core.ErrDatabase, err)
	}

	d.storage.DeleteAttachmentFolders(orphaned)

	return nil
}

func (d *DiaryService) AllAttachments(ctx context.Context, limit int, offset int) ([]string, bool, error) {
	if limit < 1 || limit > 100 {
		return nil, false, fmt.Errorf("%w: limit debe estar entre 1 y 100", core.ErrInvalidInput)
	}
	if offset < 0 {
		return nil, false, fmt.Errorf("%w: offset debe ser positivo", core.ErrInvalidInput)
	}

	keys, hasMore, err := d.db.AllAttachments(ctx, limit, offset)
	if err != nil {
		return nil, false, fmt.Errorf("%w: %w", core.ErrDatabase, err)
	}

	return keys, hasMore, nil
}

func (d *DiaryService) GetAtemporal(ctx context.Context, limit int, offset int) ([]core.DiaryEntry, bool, error) {
	if limit < 1 || limit > 100 {
		return nil, false, fmt.Errorf("%w: limit debe estar entre 1 y 100", core.ErrInvalidInput)
	}
	if offset < 0 {
		return nil, false, fmt.Errorf("%w: offset debe ser positivo", core.ErrInvalidInput)
	}

	entries, hasMore, err := d.db.GetAtemporal(ctx, limit, offset)
	if err != nil {
		return nil, false, fmt.Errorf("%w: %w", core.ErrDatabase, err)
	}

	return entries, hasMore, nil
}

func (d *DiaryService) GetById(ctx context.Context, id int) (core.DiaryEntry, error) {
	entry, err := d.db.GetById(ctx, id)
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return core.DiaryEntry{}, core.ErrNotFound
		}

		return core.DiaryEntry{}, fmt.Errorf("%w: %w", core.ErrDatabase, err)
	}

	return entry, nil
}

func (d *DiaryService) GetAttachmentPath(key string, size string) (string, error) {
	k, err := uuid.Parse(key)
	if err != nil {
		return "", fmt.Errorf("%w: key inválida", core.ErrInvalidInput)
	}

	path, err := d.storage.GetFilePath(k.String(), size)
	if err != nil {
		return "", err
	}

	return path, nil
}
