package dto

type ReviewDocumentRequest struct {
	Action  string `json:"action" binding:"required"`
	Comment string `json:"comment"`
}
