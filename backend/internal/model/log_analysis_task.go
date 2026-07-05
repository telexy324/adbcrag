package model

import (
	"time"

	"gorm.io/datatypes"
)

const (
	LogAnalysisStatusPending = "pending"
	LogAnalysisStatusRunning = "running"
	LogAnalysisStatusSuccess = "success"
	LogAnalysisStatusFailed  = "failed"
)

type LogAnalysisTask struct {
	ID              uint64         `gorm:"primaryKey" json:"id"`
	SourceID        uint64         `json:"sourceId"`
	Question        string         `gorm:"type:text" json:"question"`
	SystemName      string         `gorm:"size:100" json:"systemName"`
	ComponentName   string         `gorm:"size:100" json:"componentName"`
	TimeStart       *time.Time     `json:"timeStart"`
	TimeEnd         *time.Time     `json:"timeEnd"`
	Keyword         string         `gorm:"type:text" json:"keyword"`
	LogLevel        string         `gorm:"size:50" json:"logLevel"`
	Status          string         `gorm:"size:50;default:pending" json:"status"`
	SampleCount     int            `gorm:"default:0" json:"sampleCount"`
	ErrorMessage    string         `gorm:"type:text" json:"errorMessage"`
	Result          datatypes.JSON `gorm:"type:jsonb" json:"result"`
	RetrievedChunks datatypes.JSON `gorm:"type:jsonb" json:"retrievedChunks"`
	CreatedBy       string         `gorm:"size:100" json:"createdBy"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

func (LogAnalysisTask) TableName() string {
	return "log_analysis_task"
}
