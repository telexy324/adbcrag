package model

import "time"

const (
	LLMProviderDeepSeek         = "deepseek"
	LLMProviderQwen3            = "qwen3"
	LLMProviderOpenAICompatible = "openai_compatible"
)

type LLMConfig struct {
	ID           uint64    `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"size:120;not null" json:"name"`
	Provider     string    `gorm:"size:50;not null" json:"provider"`
	BaseURL      string    `gorm:"type:text;not null" json:"baseUrl"`
	Model        string    `gorm:"size:120;not null" json:"model"`
	APIKeyRef    string    `gorm:"type:text" json:"-"`
	APISecretRef string    `gorm:"type:text" json:"-"`
	Temperature  float64   `gorm:"default:0.2" json:"temperature"`
	IsDefault    bool      `gorm:"default:false" json:"isDefault"`
	Enabled      bool      `gorm:"default:true" json:"enabled"`
	CreatedBy    string    `gorm:"size:100" json:"createdBy"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func (LLMConfig) TableName() string {
	return "llm_config"
}
