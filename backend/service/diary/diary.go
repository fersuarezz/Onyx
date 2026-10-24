package diary

import (
	"context"
	"errors"
	"fmt"
	"io"
	"onyx/domain"

	"github.com/google/uuid"
)

type Service struct {
	db      Repository
	storage Storage
}

func NewService(db Repository, storage Storage) *Service {
	return &Service{db: db, storage: storage}
}

func (d *Service) Insert(ctx context.Context, date domain.Date, text string, srcs []io.ReadSeeker) (int, error) {
	if text == "" && len(srcs) == 0 {
		return -1, fmt.Errorf("%w: entry inválido", domain.ErrInvalidInput)
	}

	datePointer := &date
	if date.IsZero() {
		datePointer = nil
	}

	keys, err := d.storage.SaveAttachments(srcs)
	if err != nil {
		return -1, err
	}

	entry := Entry{
		Date: datePointer,
		Text: text,
		Keys: keys,
	}

	id, err := d.db.Insert(ctx, entry)
	if err != nil {
		d.storage.DeleteAttachmentFolders(keys)
		return -1, fmt.Errorf("%w: %w", domain.ErrDatabase, err)
	}

	return id, nil
}

func (d *Service) GetByDate(ctx context.Context, date domain.Date, ignoreYear bool) ([]Entry, error) {
	if date.IsZero() {
		return nil, fmt.Errorf("%w: fecha inválida", domain.ErrInvalidInput)
	}

	entries, err := d.db.GetByDate(ctx, date, ignoreYear)

	if err != nil {
		return nil, fmt.Errorf("%w: %w", domain.ErrDatabase, err)
	}

	return entries, nil
}

func (d *Service) Search(ctx context.Context, text string, limit int, offset int) ([]Entry, bool, error) {
	if limit < 1 || limit > 100 {
		return nil, false, fmt.Errorf("%w: limit debe estar entre 1 y 100", domain.ErrInvalidInput)
	}
	if offset < 0 {
		return nil, false, fmt.Errorf("%w: offset debe ser positivo", domain.ErrInvalidInput)
	}

	entries, hasMore, err := d.db.Search(ctx, text, limit, offset)
	if err != nil {
		return nil, false, fmt.Errorf("%w: %w", domain.ErrDatabase, err)
	}

	return entries, hasMore, nil
}

func (d *Service) Delete(ctx context.Context, id int) error {
	keys, err := d.db.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}

		return fmt.Errorf("%w: %w", domain.ErrDatabase, err)
	}

	d.storage.DeleteAttachmentFolders(keys)

	return nil
}

func (d *Service) Edit(ctx context.Context, id int, date *domain.Date, text *string, srcs []io.ReadSeeker) error {
	if text == nil && len(srcs) == 0 && date == nil {
		return fmt.Errorf("%w: entry inválido, nada que editar", domain.ErrInvalidInput)
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

		if errors.Is(err, domain.ErrNotFound) {
			return domain.ErrNotFound
		}

		return fmt.Errorf("%w: %w", domain.ErrDatabase, err)
	}

	d.storage.DeleteAttachmentFolders(orphaned)

	return nil
}

func (d *Service) AllAttachments(ctx context.Context, limit int, offset int) ([]string, bool, error) {
	if limit < 1 || limit > 100 {
		return nil, false, fmt.Errorf("%w: limit debe estar entre 1 y 100", domain.ErrInvalidInput)
	}
	if offset < 0 {
		return nil, false, fmt.Errorf("%w: offset debe ser positivo", domain.ErrInvalidInput)
	}

	keys, hasMore, err := d.db.AllAttachments(ctx, limit, offset)
	if err != nil {
		return nil, false, fmt.Errorf("%w: %w", domain.ErrDatabase, err)
	}

	return keys, hasMore, nil
}

func (d *Service) GetAtemporal(ctx context.Context, limit int, offset int) ([]Entry, bool, error) {
	if limit < 1 || limit > 100 {
		return nil, false, fmt.Errorf("%w: limit debe estar entre 1 y 100", domain.ErrInvalidInput)
	}
	if offset < 0 {
		return nil, false, fmt.Errorf("%w: offset debe ser positivo", domain.ErrInvalidInput)
	}

	entries, hasMore, err := d.db.GetAtemporal(ctx, limit, offset)
	if err != nil {
		return nil, false, fmt.Errorf("%w: %w", domain.ErrDatabase, err)
	}

	return entries, hasMore, nil
}

func (d *Service) GetById(ctx context.Context, id int) (Entry, error) {
	entry, err := d.db.GetById(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return Entry{}, domain.ErrNotFound
		}

		return Entry{}, fmt.Errorf("%w: %w", domain.ErrDatabase, err)
	}

	return entry, nil
}

func (d *Service) GetAttachment(key string, size string) (io.ReadSeekCloser, error) {
	k, err := uuid.Parse(key)
	if err != nil {
		return nil, fmt.Errorf("%w: key inválida", domain.ErrInvalidInput)
	}

	file, err := d.storage.Open(k.String(), size)
	if err != nil {
		return nil, err
	}

	return file, nil
}
