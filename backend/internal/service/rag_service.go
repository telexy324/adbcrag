package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ops-kb-rag/backend/internal/client"
	"ops-kb-rag/backend/internal/config"
	"ops-kb-rag/backend/internal/dto"
	"ops-kb-rag/backend/internal/model"
	"ops-kb-rag/backend/internal/repository"

	"gorm.io/datatypes"
)

type RAGService struct {
	cfg    *config.Config
	chunks *repository.ChunkRepository
	qa     *repository.QARepository
	llm    client.DeepSeekClient
}

func NewRAGService(cfg *config.Config, chunks *repository.ChunkRepository, qa *repository.QARepository, llm client.DeepSeekClient) *RAGService {
	return &RAGService{cfg: cfg, chunks: chunks, qa: qa, llm: llm}
}

func (s *RAGService) Ask(ctx context.Context, req dto.AskQuestionRequest) (*dto.AskQuestionResponse, error) {
	if req.TopK <= 0 {
		req.TopK = s.cfg.RAGTopK
	}
	analysis := s.rewriteQuery(ctx, req)
	results, err := s.chunks.KeywordSearch(ctx, repository.SearchFilter{
		SystemName: req.SystemName, ComponentName: req.ComponentName, DocType: req.DocType,
		TopK: s.cfg.RAGRecallK, Query: analysis.Query, Keywords: analysis.Keywords,
	})
	if err != nil {
		return nil, err
	}
	results = s.rerank(ctx, req.Question, results, req.TopK)
	citations := make([]dto.Citation, 0, len(results))
	for _, item := range results {
		citations = append(citations, dto.Citation{
			DocumentID: item.DocumentID, DocumentTitle: item.DocumentTitle, ChunkID: item.ID,
			SourceSection: item.SourceSection, Content: item.Content, Score: item.Score,
		})
	}
	if len(results) == 0 {
		answer := "知识库中未找到明确依据。AI 回答仅供运维排查参考，生产操作请遵守变更审批流程。"
		_ = s.saveRecord(ctx, req.Question, answer, citations)
		return &dto.AskQuestionResponse{Answer: answer, Citations: citations}, nil
	}
	prompt := buildRAGPrompt(req.Question, results)
	resp, err := s.llm.Chat(ctx, []client.ChatMessage{{Role: "user", Content: prompt}})
	if err != nil {
		return nil, err
	}
	answer := ensureSafety(resp.Content)
	if err := s.saveRecord(ctx, req.Question, answer, citations); err != nil {
		return nil, err
	}
	return &dto.AskQuestionResponse{Answer: answer, Citations: citations}, nil
}

type QueryAnalysis struct {
	Query    string   `json:"query"`
	Keywords []string `json:"keywords"`
	Intent   string   `json:"intent"`
}

func (s *RAGService) rewriteQuery(ctx context.Context, req dto.AskQuestionRequest) QueryAnalysis {
	prompt := fmt.Sprintf(`请将用户运维问题改写为知识库检索条件，只输出 JSON。

输出格式：
{
  "query": "适合直接检索的短查询",
  "keywords": ["组件", "告警", "命令", "指标", "故障现象"],
  "intent": "告警处置|应急预案|变更回滚|启停手册|其他"
}

用户问题：%s
系统过滤：%s
组件过滤：%s
文档类型过滤：%s`, req.Question, req.SystemName, req.ComponentName, req.DocType)
	resp, err := s.llm.Chat(ctx, []client.ChatMessage{{Role: "user", Content: prompt}})
	if err == nil {
		var analysis QueryAnalysis
		if unmarshalJSON(resp.Content, &analysis) == nil {
			if analysis.Query == "" {
				analysis.Query = req.Question
			}
			analysis.Keywords = uniqueNonEmpty(append(analysis.Keywords, req.Question, req.SystemName, req.ComponentName, req.DocType))
			return analysis
		}
	}
	return QueryAnalysis{Query: req.Question, Keywords: uniqueNonEmpty([]string{req.Question, req.SystemName, req.ComponentName, req.DocType})}
}

func (s *RAGService) rerank(ctx context.Context, question string, candidates []repository.SearchResult, topK int) []repository.SearchResult {
	if topK <= 0 || topK > len(candidates) {
		topK = len(candidates)
	}
	if len(candidates) <= topK {
		return candidates
	}
	var b strings.Builder
	for i, item := range candidates {
		fmt.Fprintf(&b, "[%d] 文档《%s》章节「%s」\n%s\n\n", i, item.DocumentTitle, item.SourceSection, truncate(item.Content, 800))
	}
	prompt := fmt.Sprintf(`请基于用户问题，从候选知识片段中选出最相关的 %d 个片段。

只输出 JSON：
{"indexes":[0,1,2]}

用户问题：
%s

候选片段：
%s`, topK, question, b.String())
	resp, err := s.llm.Chat(ctx, []client.ChatMessage{{Role: "user", Content: prompt}})
	if err != nil {
		return candidates[:topK]
	}
	var parsed struct {
		Indexes []int `json:"indexes"`
	}
	if unmarshalJSON(resp.Content, &parsed) != nil || len(parsed.Indexes) == 0 {
		return candidates[:topK]
	}
	reranked := make([]repository.SearchResult, 0, topK)
	used := map[int]bool{}
	for _, idx := range parsed.Indexes {
		if idx >= 0 && idx < len(candidates) && !used[idx] {
			reranked = append(reranked, candidates[idx])
			used[idx] = true
		}
		if len(reranked) == topK {
			return reranked
		}
	}
	for i := range candidates {
		if !used[i] {
			reranked = append(reranked, candidates[i])
		}
		if len(reranked) == topK {
			break
		}
	}
	return reranked
}

func (s *RAGService) saveRecord(ctx context.Context, question, answer string, citations []dto.Citation) error {
	data, _ := json.Marshal(citations)
	return s.qa.Create(ctx, &model.QARecord{
		Question: question, Answer: answer, RetrievedChunks: datatypes.JSON(data), ModelName: s.cfg.DeepSeekModel,
	})
}

func buildRAGPrompt(question string, chunks []repository.SearchResult) string {
	var b strings.Builder
	for i, chunk := range chunks {
		fmt.Fprintf(&b, "引用 %d：文档《%s》章节「%s」\n%s\n\n", i+1, chunk.DocumentTitle, chunk.SourceSection, chunk.Content)
	}
	return fmt.Sprintf(`你是一个资深银行生产运维专家。

请严格基于【知识库内容】回答用户问题。

要求：
1. 不要编造知识库中不存在的信息。
2. 如果知识库没有相关依据，请明确说明：“知识库中未找到明确依据”。
3. 涉及生产命令时，只能作为排查建议展示，不允许描述为可以直接执行。
4. 涉及重启、删除、清理、扩容、切换、回滚等高风险操作时，必须提示需要按生产变更流程审批。
5. 回答要结构清晰。
6. 最后列出引用来源。

用户问题：
%s

知识库内容：
%s

请按以下格式回答：

## 结论

## 依据

## 排查步骤

## 建议命令

## 风险提示

## 引用来源`, question, b.String())
}

func ensureSafety(answer string) string {
	if !strings.Contains(answer, "生产操作请遵守变更审批流程") {
		answer += "\n\n风险提示：AI 回答仅供运维排查参考，生产操作请遵守变更审批流程。"
	}
	return answer
}
