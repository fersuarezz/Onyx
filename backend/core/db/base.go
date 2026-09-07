package db

import (
	"context"
	"onyx/core"
)

type DiaryDataBase interface {
	Insert(ctx context.Context, entry core.DiaryEntry) (int, error)
	GetByDate(ctx context.Context, date core.Date, ignoreYear bool) ([]core.DiaryEntry, error)
	Search(ctx context.Context, text string, limit int, offset int) ([]core.DiaryEntry, bool, error)
	Delete(ctx context.Context, id int) ([]string, error)
	Edit(ctx context.Context, entry core.DiaryEntry) ([]string, error)
	AllAttachments(ctx context.Context, limit int, offset int) ([]string, bool, error)
	GetAtemporal(ctx context.Context, limit int, offset int) ([]core.DiaryEntry, bool, error)
}
