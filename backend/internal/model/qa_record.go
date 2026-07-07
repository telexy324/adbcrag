package model

import (
	"time"

	"gorm.io/datatypes"
)

type QARecord struct {
	ID              uint64         `gorm:"primaryKey" json:"id"`
	Question        string         `gorm:"type:text;not null" json:"question"`
	Answer          string         `gorm:"type:text;not null" json:"answer"`
	RetrievedChunks datatypes.JSON `gorm:"type:jsonb" json:"retrievedChunks"`
	ModelName       string         `gorm:"size:100" json:"modelName"`
	CreatedBy       string         `gorm:"size:100" json:"createdBy"`
	UserID          uint64         `gorm:"index" json:"userId"`
	ConversationID  uint64         `gorm:"index" json:"conversationId"`
	CreatedAt       time.Time      `json:"createdAt"`
}

func (QARecord) TableName() string {
	return "qa_record"
}
