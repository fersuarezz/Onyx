package core

type DiaryEntry struct {
	ID   int      `json:"id"`
	Date Date     `json:"date"`
	Text string   `json:"text"`
	Keys []string `json:"keys"` // Rutas en el servidor de los adjuntos (sin el sufijo _full, _thumbnail)
}
