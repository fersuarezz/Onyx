package diary

import "onyx/domain"

type Entry struct {
	ID   int          `json:"id"`
	Date *domain.Date `json:"date"` // NULL: atemporal
	Text string       `json:"text"`
	Keys []string     `json:"keys"` // Rutas en el servidor de los adjuntos (sin el sufijo _full, _thumbnail)
}
