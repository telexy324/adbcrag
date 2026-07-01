package model

import (
	"time"

	"gorm.io/datatypes"
)

type KBChunk struct {
	ID                uint64         `gorm:"primaryKey" json:"id"`
	DocumentID        uint64         `gorm:"not null;index" json:"documentId"`
	ChunkIndex        int            `gorm:"not null" json:"chunkIndex"`
	Content           string         `gorm:"type:text;not null" json:"content"`
	SourceTitle       string         `gorm:"size:255" json:"sourceTitle"`
	SourceSection     string         `gorm:"size:255" json:"sourceSection"`
	SourcePage        *int           `json:"sourcePage"`
	TokenCount        int            `gorm:"default:0" json:"tokenCount"`
	SearchText        string         `gorm:"type:text" json:"searchText"`
	Keywords          datatypes.JSON `gorm:"type:jsonb" json:"keywords"`
	PossibleQuestions datatypes.JSON `gorm:"type:jsonb" json:"possibleQuestions"`
	CreatedAt         time.Time      `json:"createdAt"`
	Document          KBDocument     `gorm:"foreignKey:DocumentID" json:"document,omitempty"`
}

func (KBChunk) TableName() string {
	return "kb_chunk"
}
