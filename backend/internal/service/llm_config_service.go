package service

import (
	"context"
	"fmt"
	"strings"

	"ops-kb-rag/backend/internal/client"
	"ops-kb-rag/backend/internal/config"
	"ops-kb-rag/backend/internal/dto"
	"ops-kb-rag/backend/internal/model"
	"ops-kb-rag/backend/internal/repository"
	"ops-kb-rag/backend/internal/security"

	"gorm.io/gorm"
)

type LLMConfigService struct {
	cfg    *config.Config
	repo   *repository.LLMConfigRepository
	crypto *security.CredentialCrypto
}

func NewLLMConfigService(cfg *config.Config, repo *repository.LLMConfigRepository, crypto *security.CredentialCrypto) *LLMConfigService {
	return &LLMConfigService{cfg: cfg, repo: repo, crypto: crypto}
}

func (s *LLMConfigService) List(ctx context.Context) ([]model.LLMConfig, error) {
	return s.repo.List(ctx)
}

func (s *LLMConfigService) Create(ctx context.Context, req dto.SaveLLMConfigRequest, createdBy string) (*model.LLMConfig, error) {
	item, err := s.buildConfig(req, nil)
	if err != nil {
		return nil, err
	}
	item.CreatedBy = createdBy
	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *LLMConfigService) Update(ctx context.Context, id uint64, req dto.SaveLLMConfigRequest) (*model.LLMConfig, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	next, err := s.buildConfig(req, existing)
	if err != nil {
		return nil, err
	}
	next.ID = existing.ID
	next.CreatedAt = existing.CreatedAt
	next.CreatedBy = existing.CreatedBy
	if err := s.repo.Update(ctx, next); err != nil {
		return nil, err
	}
	return next, nil
}

func (s *LLMConfigService) Delete(ctx context.Context, id uint64) error {
	return s.repo.Delete(ctx, id)
}

func (s *LLMConfigService) SetDefault(ctx context.Context, id uint64) (*model.LLMConfig, error) {
	return s.repo.SetDefault(ctx, id)
}

func (s *LLMConfigService) Active(ctx context.Context) (*dto.ActiveLLMConfigResponse, error) {
	item, err := s.repo.GetDefault(ctx)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &dto.ActiveLLMConfigResponse{
				UsingFallback: true, Name: "环境变量 DeepSeek", Provider: model.LLMProviderDeepSeek,
				BaseURL: s.cfg.DeepSeekBaseURL, Model: s.cfg.DeepSeekModel, Enabled: true,
				Message: "未配置默认模型接口，当前回退使用 DEEPSEEK_* 环境变量",
			}, nil
		}
		return nil, err
	}
	message := "当前使用数据库中设置的默认模型接口"
	if !item.Enabled {
		message = "当前默认模型接口已禁用，调用时会报错，不会静默回退 DeepSeek"
	}
	return &dto.ActiveLLMConfigResponse{
		UsingFallback: false, Name: item.Name, Provider: item.Provider, BaseURL: item.BaseURL,
		Model: item.Model, Enabled: item.Enabled, Message: message,
	}, nil
}

func (s *LLMConfigService) Test(ctx context.Context, id uint64, prompt string) (*dto.TestLLMConfigResponse, error) {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	llm, err := s.clientForConfig(item)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(prompt) == "" {
		prompt = "请回复：连接成功"
	}
	resp, err := llm.Chat(ctx, []client.ChatMessage{{Role: "user", Content: prompt}})
	if err != nil {
		return &dto.TestLLMConfigResponse{OK: false, Message: err.Error()}, nil
	}
	return &dto.TestLLMConfigResponse{OK: true, Message: "连接成功", Content: resp.Content}, nil
}

func (s *LLMConfigService) DefaultClient(ctx context.Context) (client.DeepSeekClient, string, error) {
	item, err := s.repo.GetDefault(ctx)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return client.NewOpenAICompatibleLLMClient("deepseek", s.cfg.DeepSeekBaseURL, s.cfg.DeepSeekAPIKey, s.cfg.DeepSeekModel), s.cfg.DeepSeekModel, nil
		}
		return nil, "", fmt.Errorf("读取默认模型配置失败: %w", err)
	}
	if !item.Enabled {
		return nil, "", fmt.Errorf("默认模型接口「%s」已禁用，请启用后再使用", item.Name)
	}
	llm, err := s.clientForConfig(item)
	if err != nil {
		return nil, "", fmt.Errorf("默认模型接口「%s」不可用: %w", item.Name, err)
	}
	return llm, item.Model, nil
}

func (s *LLMConfigService) clientForConfig(item *model.LLMConfig) (client.DeepSeekClient, error) {
	apiKey, err := s.crypto.Decrypt(item.APIKeyRef)
	if err != nil {
		return nil, err
	}
	apiSecret, err := s.crypto.Decrypt(item.APISecretRef)
	if err != nil {
		return nil, err
	}
	provider := item.Provider
	if provider == "" {
		provider = model.LLMProviderOpenAICompatible
	}
	return client.NewOpenAICompatibleLLMClientWithSecret(provider, item.BaseURL, apiKey, apiSecret, item.Model), nil
}

func (s *LLMConfigService) buildConfig(req dto.SaveLLMConfigRequest, existing *model.LLMConfig) (*model.LLMConfig, error) {
	name := strings.TrimSpace(req.Name)
	provider := strings.TrimSpace(req.Provider)
	baseURL := strings.TrimRight(strings.TrimSpace(req.BaseURL), "/")
	modelName := strings.TrimSpace(req.Model)
	if name == "" || provider == "" || baseURL == "" || modelName == "" {
		return nil, fmt.Errorf("name, provider, baseUrl and model are required")
	}
	if provider != model.LLMProviderDeepSeek && provider != model.LLMProviderQwen3 && provider != model.LLMProviderOpenAICompatible {
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
	temperature := 0.2
	if req.Temperature != nil {
		temperature = *req.Temperature
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	} else if existing != nil {
		enabled = existing.Enabled
	}
	isDefault := false
	if req.IsDefault != nil {
		isDefault = *req.IsDefault
	} else if existing != nil {
		isDefault = existing.IsDefault
	}
	apiKeyRef := ""
	apiSecretRef := ""
	if existing != nil {
		apiKeyRef = existing.APIKeyRef
		apiSecretRef = existing.APISecretRef
	}
	if req.APIKey != "" {
		encrypted, err := s.crypto.Encrypt(req.APIKey)
		if err != nil {
			return nil, err
		}
		apiKeyRef = encrypted
	}
	if req.APISecret != "" {
		encrypted, err := s.crypto.Encrypt(req.APISecret)
		if err != nil {
			return nil, err
		}
		apiSecretRef = encrypted
	}
	return &model.LLMConfig{
		Name: name, Provider: provider, BaseURL: baseURL, Model: modelName,
		APIKeyRef: apiKeyRef, APISecretRef: apiSecretRef, Temperature: temperature, Enabled: enabled, IsDefault: isDefault,
	}, nil
}

type DynamicLLMClient struct {
	service *LLMConfigService
}

func NewDynamicLLMClient(service *LLMConfigService) *DynamicLLMClient {
	return &DynamicLLMClient{service: service}
}

func (c *DynamicLLMClient) Chat(ctx context.Context, messages []client.ChatMessage) (*client.ChatResponse, error) {
	llm, modelName, err := c.service.DefaultClient(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := llm.Chat(ctx, messages)
	if err != nil {
		return nil, err
	}
	if resp.Model == "" {
		resp.Model = modelName
	}
	return resp, nil
}
