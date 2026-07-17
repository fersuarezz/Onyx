package db

import (
	"context"
	"onyx/core"
)

type DiaryDataBase interface {
	Insert(ctx context.Context, entry core.DiaryEntry) (int, error)
	// Devuelve las entradas de una fecha concreta. Si year = 0, devuelve de todos los años
	GetByDate(ctx context.Context, day int, month int, year int) ([]core.DiaryEntry, error)
	Search(ctx context.Context, text string, limi int) ([]core.DiaryEntry, error)
	Delete(ctx context.Context, id int) error
	Edit(ctx context.Context, entry core.DiaryEntry) error
	AllAttachments(ctx context.Context) ([]string, error)
}
