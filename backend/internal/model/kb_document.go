package model

import (
	"time"

	"gorm.io/datatypes"
)

const (
	DocumentStatusDraft      = "draft"
	DocumentStatusReviewing  = "reviewing"
	DocumentStatusPublished  = "published"
	DocumentStatusArchived   = "archived"
	DocumentStatusDeprecated = "deprecated"
	DocumentStatusRejected   = "rejected"
)

type KBDocument struct {
	ID            uint64         `gorm:"primaryKey" json:"id"`
	Title         string         `gorm:"size:255;not null" json:"title"`
	FileName      string         `gorm:"size:255;not null" json:"fileName"`
	FilePath      string         `gorm:"type:text;not null" json:"filePath"`
	FileType      string         `gorm:"size:50;not null" json:"fileType"`
	SystemName    string         `gorm:"size:100" json:"systemName"`
	ComponentName string         `gorm:"size:100" json:"componentName"`
	DocType       string         `gorm:"size:100" json:"docType"`
	Version       string         `gorm:"size:50;default:v1.0" json:"version"`
	Status        string         `gorm:"size:50;default:draft" json:"status"`
	Tags          string         `gorm:"type:text" json:"tags"`
	Summary       string         `gorm:"type:text" json:"summary"`
	QualityScore  int            `gorm:"default:0" json:"qualityScore"`
	QualityResult datatypes.JSON `gorm:"type:jsonb" json:"qualityResult"`
	CreatedBy     string         `gorm:"size:100" json:"createdBy"`
	ReviewedBy    string         `gorm:"size:100" json:"reviewedBy"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	ReviewedAt    *time.Time     `json:"reviewedAt"`
}

func (KBDocument) TableName() string {
	return "kb_document"
}
