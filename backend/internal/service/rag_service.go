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
	cfg       *config.Config
	embedding *EmbeddingService
	chunks    *repository.ChunkRepository
	qa        *repository.QARepository
	llm       client.DeepSeekClient
}

func NewRAGService(cfg *config.Config, embedding *EmbeddingService, chunks *repository.ChunkRepository, qa *repository.QARepository, llm client.DeepSeekClient) *RAGService {
	return &RAGService{cfg: cfg, embedding: embedding, chunks: chunks, qa: qa, llm: llm}
}

func (s *RAGService) Ask(ctx context.Context, req dto.AskQuestionRequest) (*dto.AskQuestionResponse, error) {
	if req.TopK <= 0 {
		req.TopK = s.cfg.RAGTopK
	}
	vector, err := s.embedding.Embed(ctx, req.Question)
	if err != nil {
		return nil, err
	}
	results, err := s.chunks.Search(ctx, vector, repository.SearchFilter{
		SystemName: req.SystemName, ComponentName: req.ComponentName, DocType: req.DocType,
		TopK: req.TopK, MinScore: s.cfg.RAGMinScore,
	})
	if err != nil {
		return nil, err
	}
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
