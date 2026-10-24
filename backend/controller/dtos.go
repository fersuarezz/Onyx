package controller

import (
	"onyx/domain"
	"onyx/service/diary"
)

type InsertRequest struct {
	Date domain.Date `form:"date"`
	Text string      `form:"text"`
}

type InsertResponse struct {
	CreatedId int `json:"created_id"`
}

type GetByDateRequest struct {
	Date       domain.Date `query:"date"`
	IgnoreYear bool        `query:"ignore_year"`
}

type GetByDateResponse struct {
	Entries []diary.Entry `json:"entries"`
}

type SearchRequest struct {
	Text   string `query:"text"`
	Limit  int    `query:"limit"`
	Offset int    `query:"offset"`
}

type SearchResponse struct {
	Entries []diary.Entry `json:"entries"`
	HasMore bool          `json:"has_more"`
}

type EditRequest struct {
	Date *domain.Date `form:"date"`
	Text *string      `form:"text"`
}

type AllAttachmentsRequest struct {
	Limit  int `query:"limit"`
	Offset int `query:"offset"`
}

type AllAttachmentsResponse struct {
	Keys    []string `json:"keys"`
	HasMore bool     `json:"has_more"`
}

type GetAttachmentRequest struct {
	Key  string `params:"key"`
	Size string `params:"size"`
}

type GetAtemporalEntriesRequest struct {
	Limit  int `query:"limit"`
	Offset int `query:"offset"`
}

type GetAtemporalEntriesResponse struct {
	Entries []diary.Entry `json:"entries"`
	HasMore bool          `json:"has_more"`
}

type GetByIdResponse struct {
	Entry diary.Entry `json:"entry"`
}
