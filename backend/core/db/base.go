package db

import (
	"onyx/core"
)

type DiaryDataBase interface {
	Insert(entry core.DiaryEntry) error
	// Devuelve las entradas de una fecha concreta. Si year = 0, devuelve de todos los años
	GetByDate(day int, month int, year int) ([]core.DiaryEntry, error)
	GetByMonth(month int, year int) ([]core.DiaryEntry, error)
	Search(text string) ([]core.DiaryEntry, error)
	Delete(id int) error
	Edit(entry core.DiaryEntry) error
	AllAttachments() ([]string, error)
}
