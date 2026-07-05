package dto

type SaveQualityCriteriaRequest struct {
	Name      string `json:"name" binding:"required"`
	Content   string `json:"content" binding:"required"`
	IsDefault *bool  `json:"isDefault"`
}
