package dto

import "time"

type UploadDocumentResponse struct {
	ID            uint64      `json:"id"`
	Title         string      `json:"title"`
	Status        string      `json:"status"`
	QualityScore  int         `json:"qualityScore"`
	QualityResult interface{} `json:"qualityResult,omitempty"`
}

type DocumentListItem struct {
	ID            uint64    `json:"id"`
	Title         string    `json:"title"`
	SystemName    string    `json:"systemName"`
	ComponentName string    `json:"componentName"`
	DocType       string    `json:"docType"`
	Status        string    `json:"status"`
	QualityScore  int       `json:"qualityScore"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

type DocumentListResponse struct {
	Items []DocumentListItem `json:"items"`
	Total int64              `json:"total"`
}

type DocumentDetailResponse struct {
	ID            uint64      `json:"id"`
	Title         string      `json:"title"`
	FileName      string      `json:"fileName"`
	SystemName    string      `json:"systemName"`
	ComponentName string      `json:"componentName"`
	DocType       string      `json:"docType"`
	Tags          string      `json:"tags"`
	Summary       string      `json:"summary"`
	Status        string      `json:"status"`
	QualityScore  int         `json:"qualityScore"`
	QualityResult interface{} `json:"qualityResult"`
	UpdatedAt     time.Time   `json:"updatedAt"`
}
