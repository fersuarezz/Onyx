package api

import (
	"onyx/core"
)

type InsertRequest struct {
	Date core.Date `form:"date" json:"date"`
	Text string    `form:"text" json:"text"`
}

type InsertResponse struct {
	CreatedId int `json:"created_id"`
}

type GetByDateRequest struct {
	Date       core.Date `query:"date"`
	IgnoreYear bool      `query:"ignore_year"`
}

type GetByDateResponse struct {
	Entries []core.DiaryEntry `json:"entries"`
}

type SearchRequest struct {
	Text  string `query:"text"`
	Limit int    `query:"limit"`
}

type SearchResponse struct {
	Entries []core.DiaryEntry `json:"entries"`
}

type DeleteRequest struct {
	Id int `query:"id" json:"id"`
}

type EditRequest struct {
	Id   int       `form:"id" json:"id"`
	Date core.Date `form:"date" json:"date"`
	Text string    `form:"text" json:"text"`
}
