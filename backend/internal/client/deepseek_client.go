package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
}

type ChatResponse struct {
	Content string
	Model   string
}

type DeepSeekClient interface {
	Chat(ctx context.Context, messages []ChatMessage) (*ChatResponse, error)
}

type OpenAICompatibleDeepSeekClient struct {
	provider  string
	baseURL   string
	apiKey    string
	apiSecret string
	model     string
	http      *http.Client
}

func NewDeepSeekClient(baseURL, apiKey, model string) DeepSeekClient {
	return NewOpenAICompatibleLLMClient("deepseek", baseURL, apiKey, model)
}

func NewOpenAICompatibleLLMClient(provider, baseURL, apiKey, model string) DeepSeekClient {
	return NewOpenAICompatibleLLMClientWithSecret(provider, baseURL, apiKey, "", model)
}

func NewOpenAICompatibleLLMClientWithSecret(provider, baseURL, apiKey, apiSecret, model string) DeepSeekClient {
	return &OpenAICompatibleDeepSeekClient{
		provider:  provider,
		baseURL:   strings.TrimRight(baseURL, "/"),
		apiKey:    apiKey,
		apiSecret: apiSecret,
		model:     model,
		http:      &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *OpenAICompatibleDeepSeekClient) Chat(ctx context.Context, messages []ChatMessage) (*ChatResponse, error) {
	body, _ := json.Marshal(ChatRequest{Model: c.model, Messages: messages, Temperature: 0.2})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("X-API-Key", c.apiKey)
	}
	if c.apiSecret != "" {
		req.Header.Set("X-API-Secret", c.apiSecret)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("%s chat failed: status=%d body=%s", c.provider, resp.StatusCode, string(respBody))
	}
	var parsed struct {
		Choices []struct {
			Message ChatMessage `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("parse %s response: %w", c.provider, err)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("%s response has no choices", c.provider)
	}
	return &ChatResponse{Content: parsed.Choices[0].Message.Content, Model: c.model}, nil
}
