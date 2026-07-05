package model

import "time"

type QualityCriteria struct {
	ID        uint64    `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:120;not null" json:"name"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	IsDefault bool      `gorm:"default:false" json:"isDefault"`
	CreatedBy string    `gorm:"size:100" json:"createdBy"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (QualityCriteria) TableName() string {
	return "quality_criteria"
}
