package db

import (
	"context"
	"onyx/core"
)

type DiaryDataBase interface {
	Insert(ctx context.Context, entry core.DiaryEntry) (int, error)
	GetByDate(ctx context.Context, date core.Date, ignoreYear bool) ([]core.DiaryEntry, error)
	Search(ctx context.Context, text string, limit int) ([]core.DiaryEntry, error)
	Delete(ctx context.Context, id int) ([]string, error)
	Edit(ctx context.Context, entry core.DiaryEntry) error
	AllAttachments(ctx context.Context) ([]string, error)
}
