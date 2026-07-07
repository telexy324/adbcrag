package model

import (
	"time"

	"gorm.io/datatypes"
)

const (
	ConversationStatusActive   = "active"
	ConversationStatusArchived = "archived"
	ConversationTypeQA         = "qa"
)

type Conversation struct {
	ID               uint64    `gorm:"primaryKey" json:"id"`
	UserID           uint64    `gorm:"not null;index" json:"userId"`
	Title            string    `gorm:"size:255" json:"title"`
	ConversationType string    `gorm:"size:50;default:qa" json:"conversationType"`
	Status           string    `gorm:"size:50;default:active" json:"status"`
	LastMessageAt    time.Time `json:"lastMessageAt"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

func (Conversation) TableName() string {
	return "conversation"
}

type ConversationMessage struct {
	ID             uint64         `gorm:"primaryKey" json:"id"`
	ConversationID uint64         `gorm:"not null;index" json:"conversationId"`
	UserID         uint64         `gorm:"index" json:"userId"`
	Role           string         `gorm:"size:30;not null" json:"role"`
	Content        string         `gorm:"type:text;not null" json:"content"`
	MessageType    string         `gorm:"size:50;default:text" json:"messageType"`
	Metadata       datatypes.JSON `gorm:"type:jsonb" json:"metadata"`
	CreatedAt      time.Time      `json:"createdAt"`
}

func (ConversationMessage) TableName() string {
	return "conversation_message"
}

type ConversationSummary struct {
	ID             uint64    `gorm:"primaryKey" json:"id"`
	ConversationID uint64    `gorm:"not null;uniqueIndex" json:"conversationId"`
	Summary        string    `gorm:"type:text" json:"summary"`
	MessageCount   int       `gorm:"default:0" json:"messageCount"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

func (ConversationSummary) TableName() string {
	return "conversation_summary"
}

type TaskState struct {
	ID             uint64         `gorm:"primaryKey" json:"id"`
	UserID         uint64         `gorm:"not null;index" json:"userId"`
	ConversationID uint64         `gorm:"not null;index" json:"conversationId"`
	TaskType       string         `gorm:"size:50;not null" json:"taskType"`
	TaskStatus     string         `gorm:"size:50;default:running" json:"taskStatus"`
	StateData      datatypes.JSON `gorm:"type:jsonb" json:"stateData"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

func (TaskState) TableName() string {
	return "task_state"
}

type ToolCallRecord struct {
	ID             uint64         `gorm:"primaryKey" json:"id"`
	UserID         uint64         `gorm:"index" json:"userId"`
	ConversationID uint64         `gorm:"index" json:"conversationId"`
	TaskID         uint64         `gorm:"index" json:"taskId"`
	ToolName       string         `gorm:"size:120" json:"toolName"`
	Request        datatypes.JSON `gorm:"type:jsonb" json:"request"`
	Response       datatypes.JSON `gorm:"type:jsonb" json:"response"`
	CreatedAt      time.Time      `json:"createdAt"`
}

func (ToolCallRecord) TableName() string {
	return "tool_call_record"
}

type ContextSnapshot struct {
	ID             uint64         `gorm:"primaryKey" json:"id"`
	UserID         uint64         `gorm:"index" json:"userId"`
	ConversationID uint64         `gorm:"index" json:"conversationId"`
	TaskID         uint64         `gorm:"index" json:"taskId"`
	SnapshotType   string         `gorm:"size:50" json:"snapshotType"`
	Content        datatypes.JSON `gorm:"type:jsonb" json:"content"`
	CreatedAt      time.Time      `json:"createdAt"`
}

func (ContextSnapshot) TableName() string {
	return "context_snapshot"
}
