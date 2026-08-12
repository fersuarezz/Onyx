package api

import (
	"onyx/core"
)

type InsertRequest struct {
	Date core.Date `form:"date" json:"date"`
	Text string    `form:"text" json:"text"`
}

type InsertResponse struct {
	CreatedId int    `json:"created_id"`
	Response  string `json:"response"`
}

type GetByDateRequest struct {
	Date       core.Date `query:"date"`
	IgnoreYear bool      `query:"ignore_year"`
}

type GetByDateResponse struct {
	Entries  []core.DiaryEntry `json:"entries"`
	Response string            `json:"response"`
}
