package diary

import (
	"context"
	"io"
	"onyx/domain"
)

type Repository interface {
	Insert(ctx context.Context, entry Entry) (int, error)
	GetByDate(ctx context.Context, date domain.Date, ignoreYear bool) ([]Entry, error)
	Search(ctx context.Context, text string, limit int, offset int) ([]Entry, bool, error)
	Delete(ctx context.Context, id int) ([]string, error)
	Edit(ctx context.Context, id int, date *domain.Date, text *string, keys []string) ([]string, error)
	AllAttachments(ctx context.Context, limit int, offset int) ([]string, bool, error)
	GetAtemporal(ctx context.Context, limit int, offset int) ([]Entry, bool, error)
	GetById(ctx context.Context, id int) (Entry, error)
}

type Storage interface {
	SaveAttachments(srcs []io.ReadSeeker) ([]string, error)
	DeleteAttachmentFolders(keys []string)
	Open(key string, size string) (io.ReadSeekCloser, error)
}
