package dto

import "time"

type SaveLogSourceRequest struct {
	Name                 string   `json:"name" binding:"required"`
	SourceType           string   `json:"sourceType" binding:"required"`
	SystemName           string   `json:"systemName"`
	ComponentName        string   `json:"componentName"`
	Environment          string   `json:"environment"`
	Endpoint             string   `json:"endpoint"`
	Username             string   `json:"username"`
	Password             string   `json:"password"`
	ESIndexPattern       string   `json:"esIndexPattern"`
	ESTimeField          string   `json:"esTimeField"`
	ServerHost           string   `json:"serverHost"`
	ServerPort           int      `json:"serverPort"`
	AuthType             string   `json:"authType"`
	PrivateKey           string   `json:"privateKey"`
	PrivateKeyPassphrase string   `json:"privateKeyPassphrase"`
	LogPath              string   `json:"logPath"`
	PathAllowlist        []string `json:"pathAllowlist"`
	Enabled              *bool    `json:"enabled"`
}

type TestLogSourceResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

type LogPreviewRequest struct {
	SourceID  uint64 `json:"sourceId" binding:"required"`
	TimeStart string `json:"timeStart"`
	TimeEnd   string `json:"timeEnd"`
	Keyword   string `json:"keyword"`
	LogLevel  string `json:"logLevel"`
	LogPath   string `json:"logPath"`
	Limit     int    `json:"limit"`
}

type LogItem struct {
	Timestamp *time.Time `json:"timestamp,omitempty"`
	Level     string     `json:"level"`
	Message   string     `json:"message"`
	Source    string     `json:"source"`
	Raw       string     `json:"raw"`
}

type LogPreviewResponse struct {
	Items []LogItem `json:"items"`
	Total int       `json:"total"`
}

type LogAnalysisRequest struct {
	SourceID      uint64 `json:"sourceId" binding:"required"`
	Question      string `json:"question" binding:"required"`
	SystemName    string `json:"systemName"`
	ComponentName string `json:"componentName"`
	TimeStart     string `json:"timeStart"`
	TimeEnd       string `json:"timeEnd"`
	Keyword       string `json:"keyword"`
	LogLevel      string `json:"logLevel"`
	LogPath       string `json:"logPath"`
	TopK          int    `json:"topK"`
}

type LogAnalysisResponse struct {
	TaskID         uint64     `json:"taskId"`
	Status         string     `json:"status"`
	Summary        string     `json:"summary"`
	PossibleCauses []string   `json:"possibleCauses"`
	Evidence       []string   `json:"evidence"`
	Suggestions    []string   `json:"suggestions"`
	RiskTips       []string   `json:"riskTips"`
	Citations      []Citation `json:"citations"`
}
