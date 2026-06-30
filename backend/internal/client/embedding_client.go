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

type EmbeddingClient interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

type OpenAICompatibleEmbeddingClient struct {
	baseURL string
	apiKey  string
	model   string
	dim     int
	http    *http.Client
}

func NewEmbeddingClient(baseURL, apiKey, model string, dim int) EmbeddingClient {
	return &OpenAICompatibleEmbeddingClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
		dim:     dim,
		http:    &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *OpenAICompatibleEmbeddingClient) Embed(ctx context.Context, text string) ([]float32, error) {
	body, _ := json.Marshal(map[string]interface{}{"model": c.model, "input": text})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("embedding request failed: status=%d body=%s", resp.StatusCode, string(respBody))
	}
	var parsed struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("parse embedding response: %w", err)
	}
	if len(parsed.Data) == 0 {
		return nil, fmt.Errorf("embedding response has no data")
	}
	if c.dim > 0 && len(parsed.Data[0].Embedding) != c.dim {
		return nil, fmt.Errorf("embedding dim mismatch: got %d want %d", len(parsed.Data[0].Embedding), c.dim)
	}
	return parsed.Data[0].Embedding, nil
}
