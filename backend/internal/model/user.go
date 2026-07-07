package model

import "time"

const (
	UserRoleAdmin = "admin"
	UserRoleUser  = "user"
)

type AppUser struct {
	ID                uint64     `gorm:"primaryKey" json:"id"`
	Username          string     `gorm:"size:100;not null;uniqueIndex" json:"username"`
	DisplayName       string     `gorm:"size:120" json:"displayName"`
	PasswordHash      string     `gorm:"type:text;not null" json:"-"`
	Role              string     `gorm:"size:30;not null;default:user" json:"role"`
	Enabled           bool       `gorm:"not null;default:true" json:"enabled"`
	LastLoginAt       *time.Time `json:"lastLoginAt"`
	PasswordUpdatedAt *time.Time `json:"passwordUpdatedAt"`
	CreatedBy         uint64     `json:"createdBy"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

func (AppUser) TableName() string {
	return "app_user"
}

type LoginAudit struct {
	ID            uint64    `gorm:"primaryKey" json:"id"`
	UserID        *uint64   `json:"userId"`
	Username      string    `gorm:"size:100" json:"username"`
	Success       bool      `gorm:"not null" json:"success"`
	FailureReason string    `gorm:"type:text" json:"failureReason"`
	ClientIP      string    `gorm:"size:100" json:"clientIp"`
	UserAgent     string    `gorm:"type:text" json:"userAgent"`
	CreatedAt     time.Time `json:"createdAt"`
}

func (LoginAudit) TableName() string {
	return "login_audit"
}
