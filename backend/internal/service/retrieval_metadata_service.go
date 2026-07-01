package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ops-kb-rag/backend/internal/client"
)

type ChunkMetadata struct {
	Summary           string   `json:"summary"`
	Keywords          []string `json:"keywords"`
	PossibleQuestions []string `json:"possibleQuestions"`
}

type RetrievalMetadataService struct {
	llm client.DeepSeekClient
}

func NewRetrievalMetadataService(llm client.DeepSeekClient) *RetrievalMetadataService {
	return &RetrievalMetadataService{llm: llm}
}

func (s *RetrievalMetadataService) Extract(ctx context.Context, title, section, content string) (*ChunkMetadata, []byte, []byte, string) {
	prompt := fmt.Sprintf(`请为运维知识库片段生成检索增强信息。

要求：
1. 只输出 JSON。
2. keywords 包含系统、组件、告警、命令、指标、错误码、操作动作等关键词。
3. possibleQuestions 写出用户可能提出的问题。

输出格式：
{
  "summary": "",
  "keywords": [],
  "possibleQuestions": []
}

文档标题：%s
章节：%s
片段内容：
%s`, title, section, truncate(content, 3000))
	resp, err := s.llm.Chat(ctx, []client.ChatMessage{{Role: "user", Content: prompt}})
	if err == nil {
		var meta ChunkMetadata
		if unmarshalJSON(resp.Content, &meta) == nil {
			return normalizeMetadata(&meta, title, section, content)
		}
	}
	return normalizeMetadata(fallbackMetadata(title, section, content), title, section, content)
}

func normalizeMetadata(meta *ChunkMetadata, title, section, content string) (*ChunkMetadata, []byte, []byte, string) {
	if meta.Summary == "" {
		meta.Summary = firstRunes(content, 160)
	}
	meta.Keywords = uniqueNonEmpty(append(meta.Keywords, title, section))
	meta.PossibleQuestions = uniqueNonEmpty(meta.PossibleQuestions)
	keywordsJSON, _ := json.Marshal(meta.Keywords)
	questionsJSON, _ := json.Marshal(meta.PossibleQuestions)
	searchText := strings.Join(append([]string{title, section, meta.Summary, content}, append(meta.Keywords, meta.PossibleQuestions...)...), "\n")
	return meta, keywordsJSON, questionsJSON, searchText
}

func fallbackMetadata(title, section, content string) *ChunkMetadata {
	keywords := []string{title, section}
	for _, token := range strings.Fields(strings.NewReplacer("，", " ", "。", " ", "：", " ", "；", " ", "\n", " ").Replace(content)) {
		if len([]rune(token)) >= 2 && len([]rune(token)) <= 32 {
			keywords = append(keywords, token)
		}
		if len(keywords) >= 20 {
			break
		}
	}
	return &ChunkMetadata{
		Summary:           firstRunes(content, 160),
		Keywords:          keywords,
		PossibleQuestions: []string{fmt.Sprintf("%s 怎么处理？", section), fmt.Sprintf("%s 有哪些排查步骤？", title)},
	}
}

func unmarshalJSON(text string, target interface{}) error {
	return json.Unmarshal([]byte(extractJSON(text)), target)
}

func uniqueNonEmpty(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func firstRunes(text string, limit int) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit])
}
