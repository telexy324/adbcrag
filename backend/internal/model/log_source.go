package model

import (
	"time"

	"gorm.io/datatypes"
)

const (
	LogSourceTypeElasticsearch = "elasticsearch"
	LogSourceTypeServerFile    = "server_file"

	LogAuthTypePassword   = "password"
	LogAuthTypePrivateKey = "private_key"
)

type LogSource struct {
	ID             uint64         `gorm:"primaryKey" json:"id"`
	Name           string         `gorm:"size:120;not null" json:"name"`
	SourceType     string         `gorm:"size:50;not null" json:"sourceType"`
	SystemName     string         `gorm:"size:100" json:"systemName"`
	ComponentName  string         `gorm:"size:100" json:"componentName"`
	Environment    string         `gorm:"size:50" json:"environment"`
	Endpoint       string         `gorm:"type:text" json:"endpoint"`
	Username       string         `gorm:"size:255" json:"username"`
	CredentialRef  string         `gorm:"type:text" json:"-"`
	ESIndexPattern string         `gorm:"size:255" json:"esIndexPattern"`
	ESTimeField    string         `gorm:"size:100" json:"esTimeField"`
	ServerHost     string         `gorm:"size:255" json:"serverHost"`
	ServerPort     int            `gorm:"default:22" json:"serverPort"`
	AuthType       string         `gorm:"size:50" json:"authType"`
	LogPath        string         `gorm:"type:text" json:"logPath"`
	PathAllowlist  datatypes.JSON `gorm:"type:jsonb" json:"pathAllowlist"`
	Enabled        bool           `gorm:"default:true" json:"enabled"`
	CreatedBy      string         `gorm:"size:100" json:"createdBy"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

func (LogSource) TableName() string {
	return "log_source"
}
