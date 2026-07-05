package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ops-kb-rag/backend/internal/client"
	"ops-kb-rag/backend/internal/model"
)

type QualityResult struct {
	Score              int      `json:"score"`
	Level              string   `json:"level"`
	Summary            string   `json:"summary"`
	Criteria           string   `json:"criteria,omitempty"`
	Problems           []Issue  `json:"problems"`
	MissingFields      []string `json:"missingFields"`
	RiskPoints         []string `json:"riskPoints"`
	RewriteSuggestions []string `json:"rewriteSuggestions"`
}

type Issue struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
}

type QualityService struct {
	llm client.DeepSeekClient
}

func NewQualityService(llm client.DeepSeekClient) *QualityService {
	return &QualityService{llm: llm}
}

const defaultQualityCriteria = "完整性、准确性、可操作性、可验证性、可追溯性"

func (s *QualityService) Check(ctx context.Context, content, criteria string) (*QualityResult, []byte, error) {
	criteria = normalizeCriteria(criteria)
	prompt := fmt.Sprintf(`你是一个银行生产运维文档审核专家。

请检查下面的运维手册是否适合进入生产知识库。

请严格依据以下评分标准评分，总分 100 分：
%s

请输出 JSON，不要输出多余解释。

输出格式：
{
  "score": 0,
  "level": "pass | warning | reject",
  "summary": "",
  "criteria": "",
  "problems": [{"type": "", "description": "", "suggestion": ""}],
  "missingFields": [],
  "riskPoints": [],
  "rewriteSuggestions": []
}

手册内容：
%s`, criteria, truncate(content, 12000))
	resp, err := s.llm.Chat(ctx, []client.ChatMessage{{Role: "user", Content: prompt}})
	if err != nil {
		result := fallbackQuality(content, criteria)
		return result, mustJSON(result), nil
	}
	raw := extractJSON(resp.Content)
	var result QualityResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		result := fallbackQuality(content, criteria)
		return result, mustJSON(result), nil
	}
	if result.Score == 0 {
		result.Score = 70
	}
	result.Criteria = criteria
	return &result, mustJSON(result), nil
}

func StatusByQuality(score int) string {
	if score < 70 {
		return model.DocumentStatusDraft
	}
	return model.DocumentStatusReviewing
}

func fallbackQuality(content, criteria string) *QualityResult {
	score := 75
	level := "warning"
	if strings.Contains(content, "回滚") && strings.Contains(content, "验证") && strings.Contains(content, "风险") {
		score = 86
	}
	return &QualityResult{
		Score:    score,
		Level:    level,
		Summary:  "已完成基础质量检查；LLM 质检不可用时使用本地启发式结果。",
		Criteria: criteria,
		Problems: []Issue{{
			Type:        "llm_unavailable",
			Description: "未能调用或解析 LLM 质检结果。",
			Suggestion:  "请确认 DeepSeek v4 接口可用后重新入库或复核。",
		}},
	}
}

func normalizeCriteria(criteria string) string {
	criteria = strings.TrimSpace(criteria)
	if criteria == "" {
		return defaultQualityCriteria
	}
	return truncate(criteria, 2000)
}

func extractJSON(text string) string {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start >= 0 && end > start {
		return text[start : end+1]
	}
	return text
}

func truncate(text string, max int) string {
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	return string(runes[:max])
}

func mustJSON(value interface{}) []byte {
	data, _ := json.Marshal(value)
	return data
}
