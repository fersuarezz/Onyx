package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"onyx/core"
	"onyx/core/db"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

const imgFull = "full.jpg"
const imgThumb = "thumbnail.jpg"
const tempDir = ".temp"

type DiaryService struct {
	Diary     db.DiaryDataBase
	MediaRoot string
}

func (d *DiaryService) saveAttachments(srcs []io.ReadSeeker) ([]string, error) {
	var keys []string

	for _, src := range srcs {
		key, err := d.saveAttachment(src)
		if err != nil {
			for _, v := range keys {
				err2 := d.deleteAttachmentFolder(v)
				if err2 != nil {
					slog.Error("no se pudo borrar el adjunto", "key", v, "err", err2)
				}
			}

			return nil, err
		}

		keys = append(keys, key)
	}

	return keys, nil
}

func (d *DiaryService) deleteAttachmentFolder(key string) error {
	folder := filepath.Join(d.MediaRoot, key)

	err := os.RemoveAll(folder)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrFileSystem, err)
	}
	return nil
}

func (d *DiaryService) saveAttachment(src io.ReadSeeker) (string, error) {
	key, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("generando uui7: %w", err)
	}

	tempFolder := filepath.Join(d.MediaRoot, tempDir, key.String())
	full := filepath.Join(tempFolder, imgFull)
	thumbnail := filepath.Join(tempFolder, imgThumb)

	err = os.MkdirAll(tempFolder, 0o755)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrFileSystem, err)
	}

	defer os.RemoveAll(tempFolder) // Rollback

	// Guardar imagen full
	dst, err := os.Create(full)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrFileSystem, err)
	}

	defer dst.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrFileSystem, err)
	}

	// Guardar thummbnail
	dst2, err := os.Create(thumbnail)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrFileSystem, err)
	}

	_, err = src.Seek(0, io.SeekStart)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrFileSystem, err)
	}
	th, err := Thumbnail(src)
	if err != nil {
		return "", err
	}

	defer dst2.Close()

	_, err = io.Copy(dst2, th)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrFileSystem, err)
	}

	// Rename folder
	err = dst.Close()
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrFileSystem, err)
	}

	err = dst2.Close()
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrFileSystem, err)
	}

	finalFolder := filepath.Join(d.MediaRoot, key.String())

	err = os.Rename(tempFolder, finalFolder)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrFileSystem, err)
	}

	return key.String(), nil
}

func (d *DiaryService) Insert(ctx context.Context, date core.Date, text string, srcs []io.ReadSeeker) (int, error) {
	if date.IsZero() || (text == "" && len(srcs) == 0) {
		return -1, fmt.Errorf("%w: entry inválido", ErrInvalidInput)
	}

	keys, err := d.saveAttachments(srcs)
	if err != nil {
		return -1, err
	}

	entry := core.DiaryEntry{
		Date: date,
		Text: text,
		Keys: keys,
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

func (d *DiaryService) Search(ctx context.Context, text string, limit int) ([]core.DiaryEntry, error) {
	if limit < 1 || limit > 100 {
		return nil, fmt.Errorf("%w: limit debe estar entre 1 y 100", ErrInvalidInput)
	}

	entries, err := d.Diary.Search(ctx, text, limit)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrDatabase, err)
	}

	return entries, nil
}

func (d *DiaryService) Delete(ctx context.Context, id int) error {
	keys, err := d.Diary.Delete(ctx, id)

	if errors.Is(err, db.ErrNotFound) {
		return err
	}
	if err != nil {
		return fmt.Errorf("%w: %w", ErrDatabase, err)
	}

	for _, key := range keys {
		err = d.deleteAttachmentFolder(key)
		if err != nil {
			slog.Error("no se pudo borrar el adjunto", "key", key, "err", err)
		}
	}
	return nil
}
