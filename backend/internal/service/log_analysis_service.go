package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"ops-kb-rag/backend/internal/client"
	"ops-kb-rag/backend/internal/config"
	"ops-kb-rag/backend/internal/dto"
	"ops-kb-rag/backend/internal/model"
	"ops-kb-rag/backend/internal/repository"

	"gorm.io/datatypes"
)

type LogAnalysisService struct {
	cfg        *config.Config
	sources    *LogSourceService
	tasks      *repository.LogAnalysisRepository
	chunks     *repository.ChunkRepository
	es         client.ElasticsearchClient
	ssh        client.SSHLogClient
	llm        client.DeepSeekClient
	maskRegexp []*regexp.Regexp
}

func NewLogAnalysisService(cfg *config.Config, sources *LogSourceService, tasks *repository.LogAnalysisRepository, chunks *repository.ChunkRepository, es client.ElasticsearchClient, ssh client.SSHLogClient, llm client.DeepSeekClient) *LogAnalysisService {
	return &LogAnalysisService{
		cfg: cfg, sources: sources, tasks: tasks, chunks: chunks, es: es, ssh: ssh, llm: llm,
		maskRegexp: []*regexp.Regexp{
			regexp.MustCompile(`(?i)(password|passwd|pwd|secret|token|access[_-]?key|session)[=:]\s*[^,\s]+`),
			regexp.MustCompile(`\b1[3-9]\d{9}\b`),
			regexp.MustCompile(`\b\d{16,19}\b`),
			regexp.MustCompile(`\b\d{17}[\dXx]\b`),
		},
	}
}

func (s *LogAnalysisService) Preview(ctx context.Context, req dto.LogPreviewRequest) (*dto.LogPreviewResponse, error) {
	items, err := s.readLogs(ctx, req.SourceID, req.TimeStart, req.TimeEnd, req.Keyword, req.LogLevel, req.LogPath, req.Limit)
	if err != nil {
		return nil, err
	}
	items = s.sanitizeAndLimit(items)
	return &dto.LogPreviewResponse{Items: items, Total: len(items)}, nil
}

func (s *LogAnalysisService) Analyze(ctx context.Context, req dto.LogAnalysisRequest, createdBy string) (*dto.LogAnalysisResponse, error) {
	task := &model.LogAnalysisTask{
		SourceID: req.SourceID, Question: req.Question, SystemName: req.SystemName, ComponentName: req.ComponentName,
		Keyword: req.Keyword, LogLevel: req.LogLevel, Status: model.LogAnalysisStatusRunning, CreatedBy: createdBy,
	}
	task.TimeStart = parseOptionalTimePtr(req.TimeStart)
	task.TimeEnd = parseOptionalTimePtr(req.TimeEnd)
	if err := s.tasks.Create(ctx, task); err != nil {
		return nil, err
	}
	items, err := s.readLogs(ctx, req.SourceID, req.TimeStart, req.TimeEnd, req.Keyword, req.LogLevel, req.LogPath, s.cfg.LogMaxLines)
	if err != nil {
		task.Status = model.LogAnalysisStatusFailed
		task.ErrorMessage = err.Error()
		_ = s.tasks.Update(ctx, task)
		return nil, err
	}
	items = s.sanitizeAndLimit(items)
	task.SampleCount = len(items)
	query := strings.TrimSpace(req.Question + " " + req.Keyword + " " + extractLogKeywords(items))
	results, err := s.chunks.KeywordSearch(ctx, repository.SearchFilter{
		SystemName: req.SystemName, ComponentName: req.ComponentName, Query: query,
		Keywords: uniqueNonEmpty([]string{req.Question, req.Keyword, req.SystemName, req.ComponentName}),
		TopK:     choosePositive(req.TopK, s.cfg.RAGTopK),
	})
	if err != nil {
		task.Status = model.LogAnalysisStatusFailed
		task.ErrorMessage = err.Error()
		_ = s.tasks.Update(ctx, task)
		return nil, err
	}
	citations := searchResultsToCitations(results)
	response := s.analyzeWithLLM(ctx, req, items, results, citations)
	response.TaskID = task.ID
	response.Status = model.LogAnalysisStatusSuccess
	resultJSON, _ := json.Marshal(response)
	citationJSON, _ := json.Marshal(citations)
	task.Status = model.LogAnalysisStatusSuccess
	task.Result = datatypes.JSON(resultJSON)
	task.RetrievedChunks = datatypes.JSON(citationJSON)
	if err := s.tasks.Update(ctx, task); err != nil {
		return nil, err
	}
	return response, nil
}

func (s *LogAnalysisService) readLogs(ctx context.Context, sourceID uint64, start, end, keyword, level, overridePath string, limit int) ([]dto.LogItem, error) {
	source, credentials, err := s.sources.GetWithCredentials(ctx, sourceID)
	if err != nil {
		return nil, err
	}
	if !source.Enabled {
		return nil, fmt.Errorf("log source is disabled")
	}
	if limit <= 0 || limit > s.cfg.LogMaxLines {
		limit = s.cfg.LogMaxLines
	}
	if source.SourceType == model.LogSourceTypeElasticsearch {
		return s.es.QueryLogs(ctx, client.ESLogQuery{
			ESConfig:     client.ESConfig{Endpoint: source.Endpoint, Username: source.Username, Password: credentials.Password, Timeout: time.Duration(s.cfg.ESTimeoutSec) * time.Second},
			IndexPattern: source.ESIndexPattern, TimeField: source.ESTimeField, TimeStart: parseOptionalTimePtr(start), TimeEnd: parseOptionalTimePtr(end),
			Keyword: keyword, LogLevel: level, Limit: limit,
		})
	}
	path := source.LogPath
	if overridePath != "" {
		path = overridePath
	}
	return s.ssh.ReadLogs(ctx, client.SSHLogQuery{
		SSHConfig: client.SSHConfig{
			Host: source.ServerHost, Port: source.ServerPort, Username: source.Username, Password: credentials.Password,
			PrivateKey: credentials.PrivateKey, Passphrase: credentials.PrivateKeyPassphrase, AuthType: source.AuthType,
			Timeout: time.Duration(s.cfg.SSHTimeoutSec) * time.Second,
		},
		LogPath: path, PathAllowlist: decodeStringArray(source.PathAllowlist), Keyword: keyword, LogLevel: level, Limit: limit,
	})
}

func (s *LogAnalysisService) sanitizeAndLimit(items []dto.LogItem) []dto.LogItem {
	seen := map[string]bool{}
	limited := []dto.LogItem{}
	bytesUsed := 0
	for _, item := range items {
		item.Message = s.mask(item.Message)
		item.Raw = s.mask(item.Raw)
		key := item.Message
		if key == "" {
			key = item.Raw
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		bytesUsed += len(item.Raw) + len(item.Message)
		if bytesUsed > s.cfg.LogMaxBytes || len(limited) >= s.cfg.LogMaxLines {
			break
		}
		limited = append(limited, item)
	}
	return limited
}

func (s *LogAnalysisService) mask(text string) string {
	for _, re := range s.maskRegexp {
		text = re.ReplaceAllString(text, "[MASKED]")
	}
	return text
}

func (s *LogAnalysisService) analyzeWithLLM(ctx context.Context, req dto.LogAnalysisRequest, logs []dto.LogItem, chunks []repository.SearchResult, citations []dto.Citation) *dto.LogAnalysisResponse {
	if len(logs) == 0 {
		return &dto.LogAnalysisResponse{
			Summary:   "未读取到符合条件的日志样本。",
			RiskTips:  []string{"AI 日志分析仅供运维排查参考，生产操作请遵守变更审批流程。"},
			Citations: citations,
		}
	}
	prompt := buildLogAnalysisPrompt(req, logs, chunks)
	resp, err := s.llm.Chat(ctx, []client.ChatMessage{{Role: "user", Content: prompt}})
	if err != nil {
		return fallbackLogAnalysis(logs, citations)
	}
	var parsed dto.LogAnalysisResponse
	if unmarshalJSON(resp.Content, &parsed) != nil {
		parsed = *fallbackLogAnalysis(logs, citations)
	} else {
		parsed.Citations = citations
	}
	if len(parsed.RiskTips) == 0 {
		parsed.RiskTips = []string{"AI 日志分析仅供运维排查参考，生产操作请遵守变更审批流程。"}
	}
	return &parsed
}

func buildLogAnalysisPrompt(req dto.LogAnalysisRequest, logs []dto.LogItem, chunks []repository.SearchResult) string {
	var lb strings.Builder
	for i, item := range logs {
		fmt.Fprintf(&lb, "[%d] %s %s %s\n", i+1, item.Level, item.Source, chooseNonEmpty(item.Message, item.Raw))
	}
	var cb strings.Builder
	for i, chunk := range chunks {
		fmt.Fprintf(&cb, "引用 %d：文档《%s》章节「%s」\n%s\n\n", i+1, chunk.DocumentTitle, chunk.SourceSection, truncate(chunk.Content, 1000))
	}
	return fmt.Sprintf(`你是一个资深银行生产运维日志分析专家。

请基于【日志样本】和【知识库内容】分析用户问题，只输出 JSON。

要求：
1. 区分日志事实、知识库依据和推测原因。
2. 不要编造日志中不存在的错误、时间点、接口、主机或指标。
3. 涉及生产命令时只能作为排查建议展示。
4. 涉及重启、删除、清理、扩容、切换、回滚等高风险操作时，必须提示需要按生产变更流程审批。

输出格式：
{"summary":"","possibleCauses":[],"evidence":[],"suggestions":[],"riskTips":[]}

用户问题：
%s

日志样本：
%s

知识库内容：
%s`, req.Question, lb.String(), cb.String())
}

func fallbackLogAnalysis(logs []dto.LogItem, citations []dto.Citation) *dto.LogAnalysisResponse {
	evidence := []string{}
	for i, item := range logs {
		if i >= 5 {
			break
		}
		evidence = append(evidence, chooseNonEmpty(item.Message, item.Raw))
	}
	return &dto.LogAnalysisResponse{
		Summary:     "已读取日志样本，但 LLM 分析不可用或返回格式异常；请结合日志证据和引用文档人工复核。",
		Evidence:    evidence,
		Suggestions: []string{"优先核对异常时间窗口内的发布、配置变更、依赖服务状态和资源指标。"},
		RiskTips:    []string{"AI 日志分析仅供运维排查参考，生产操作请遵守变更审批流程。"},
		Citations:   citations,
	}
}

func searchResultsToCitations(results []repository.SearchResult) []dto.Citation {
	citations := make([]dto.Citation, 0, len(results))
	for _, item := range results {
		citations = append(citations, dto.Citation{DocumentID: item.DocumentID, DocumentTitle: item.DocumentTitle, ChunkID: item.ID, SourceSection: item.SourceSection, Content: item.Content, Score: item.Score})
	}
	return citations
}

func extractLogKeywords(items []dto.LogItem) string {
	words := []string{}
	re := regexp.MustCompile(`[A-Za-z][A-Za-z0-9_.-]{3,}`)
	for _, item := range items {
		words = append(words, re.FindAllString(chooseNonEmpty(item.Message, item.Raw), 10)...)
		if len(words) > 30 {
			break
		}
	}
	return strings.Join(uniqueNonEmpty(words), " ")
}

func decodeStringArray(data []byte) []string {
	var values []string
	_ = json.Unmarshal(data, &values)
	return values
}

func parseOptionalTimePtr(value string) *time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return &parsed
		}
	}
	return nil
}

func choosePositive(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func chooseNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
