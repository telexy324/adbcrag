package dto

type AskQuestionRequest struct {
	Question       string `json:"question" binding:"required"`
	SystemName     string `json:"systemName"`
	ComponentName  string `json:"componentName"`
	DocType        string `json:"docType"`
	TopK           int    `json:"topK"`
	ConversationID uint64 `json:"conversationId"`
}

type Citation struct {
	DocumentID    uint64  `json:"documentId"`
	DocumentTitle string  `json:"documentTitle"`
	ChunkID       uint64  `json:"chunkId"`
	SourceSection string  `json:"sourceSection"`
	Content       string  `json:"content"`
	Score         float64 `json:"score"`
}

type AskQuestionResponse struct {
	Answer         string     `json:"answer"`
	Citations      []Citation `json:"citations"`
	ConversationID uint64     `json:"conversationId"`
	MessageID      uint64     `json:"messageId"`
}
