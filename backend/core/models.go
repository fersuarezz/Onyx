package core

import "time"

type DiaryEntry struct {
	ID          int       `json:"id"`
	Date        time.Time `json:"date"`
	Text        string    `json:"text"`
	Attachments []string  `json:"attachments"` // Rutas en el servidor de los adjuntos
}
