package core

import "time"

type DiaryEntry struct {
	Id          int       `json:"id"`
	Date        time.Time `json:"date"`
	Text        string    `json:"text"`
	Attachments []string  `json:"attachments"` // Rutas del servidor de los adjuntos
}
