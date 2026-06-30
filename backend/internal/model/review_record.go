package model

import "time"

type KBReviewRecord struct {
	ID         uint64     `gorm:"primaryKey" json:"id"`
	DocumentID uint64     `gorm:"not null;index" json:"documentId"`
	FromStatus string     `gorm:"size:50" json:"fromStatus"`
	ToStatus   string     `gorm:"size:50" json:"toStatus"`
	Reviewer   string     `gorm:"size:100" json:"reviewer"`
	Comment    string     `gorm:"type:text" json:"comment"`
	CreatedAt  time.Time  `json:"createdAt"`
	Document   KBDocument `gorm:"foreignKey:DocumentID" json:"document,omitempty"`
}

func (KBReviewRecord) TableName() string {
	return "kb_review_record"
}
