package storage

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"onyx/core"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

type FileSystemStorage struct {
	mediaRoot string
}

func NewFileSystemStorage(mediaRoot string) *FileSystemStorage {
	return &FileSystemStorage{mediaRoot: mediaRoot}
}

func (s *FileSystemStorage) SaveAttachments(srcs []io.ReadSeeker) ([]string, error) {
	var keys []string

	for _, src := range srcs {
		key, err := s.saveAttachment(src)
		if err != nil {
			s.DeleteAttachmentFolders(keys)

			return nil, err
		}

		keys = append(keys, key)
	}

	return keys, nil
}

func (s *FileSystemStorage) deleteAttachmentFolder(key string) error {
	folder := filepath.Join(s.mediaRoot, key)

	err := os.RemoveAll(folder)
	if err != nil {
		return fmt.Errorf("%w: %w", core.ErrFileSystem, err)
	}
	return nil
}

func (s *FileSystemStorage) DeleteAttachmentFolders(keys []string) {
	for _, v := range keys {
		err := s.deleteAttachmentFolder(v)
		if err != nil {
			slog.Error("no se pudo borrar el adjunto", "key", v, "err", err)
		}
	}
}

func (s *FileSystemStorage) saveAttachment(src io.ReadSeeker) (string, error) {
	key, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("generando uui7: %w", err)
	}

	tempFolder := filepath.Join(s.mediaRoot, tempDir, key.String())
	full := filepath.Join(tempFolder, imgFull)
	thumbnail := filepath.Join(tempFolder, imgThumb)

	err = os.MkdirAll(tempFolder, 0o755)
	if err != nil {
		return "", fmt.Errorf("%w: %w", core.ErrFileSystem, err)
	}

	defer os.RemoveAll(tempFolder) // Rollback

	// Guardar imagen full
	dst, err := os.Create(full)
	if err != nil {
		return "", fmt.Errorf("%w: %w", core.ErrFileSystem, err)
	}

	defer dst.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		return "", fmt.Errorf("%w: %w", core.ErrFileSystem, err)
	}

	// Guardar thummbnail
	dst2, err := os.Create(thumbnail)
	if err != nil {
		return "", fmt.Errorf("%w: %w", core.ErrFileSystem, err)
	}

	_, err = src.Seek(0, io.SeekStart)
	if err != nil {
		return "", fmt.Errorf("%w: %w", core.ErrFileSystem, err)
	}
	th, err := Thumbnail(src)
	if err != nil {
		return "", err
	}

	defer dst2.Close()

	_, err = io.Copy(dst2, th)
	if err != nil {
		return "", fmt.Errorf("%w: %w", core.ErrFileSystem, err)
	}

	// Rename folder
	err = dst.Close()
	if err != nil {
		return "", fmt.Errorf("%w: %w", core.ErrFileSystem, err)
	}

	err = dst2.Close()
	if err != nil {
		return "", fmt.Errorf("%w: %w", core.ErrFileSystem, err)
	}

	finalFolder := filepath.Join(s.mediaRoot, key.String())

	err = os.Rename(tempFolder, finalFolder)
	if err != nil {
		return "", fmt.Errorf("%w: %w", core.ErrFileSystem, err)
	}

	return key.String(), nil
}

func (s *FileSystemStorage) GetFilePath(key string, size string) (string, error) {
	var file string
	switch size {
	case SizeFull:
		file = imgFull
	case SizeThumb:
		file = imgThumb
	default:
		return "", fmt.Errorf("%w: size inválido", core.ErrInvalidInput)
	}

	path := filepath.Join(s.mediaRoot, key, file)

	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("%w: el archivo no existe", core.ErrNotFound)
		}
		return "", fmt.Errorf("%w: %w", core.ErrFileSystem, err)
	}

	return path, nil
}
