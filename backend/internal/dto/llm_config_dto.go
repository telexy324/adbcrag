package dto

type SaveLLMConfigRequest struct {
	Name        string   `json:"name" binding:"required"`
	Provider    string   `json:"provider" binding:"required"`
	BaseURL     string   `json:"baseUrl" binding:"required"`
	Model       string   `json:"model" binding:"required"`
	APIKey      string   `json:"apiKey"`
	APISecret   string   `json:"apiSecret"`
	Temperature *float64 `json:"temperature"`
	IsDefault   *bool    `json:"isDefault"`
	Enabled     *bool    `json:"enabled"`
}

type TestLLMConfigRequest struct {
	Prompt string `json:"prompt"`
}

type TestLLMConfigResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	Content string `json:"content,omitempty"`
}
