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

	"ops-kb-rag/backend/internal/dto"
)

type ESConfig struct {
	Endpoint string
	Username string
	Password string
	Timeout  time.Duration
}

type ESLogQuery struct {
	ESConfig
	IndexPattern string
	TimeField    string
	TimeStart    *time.Time
	TimeEnd      *time.Time
	Keyword      string
	LogLevel     string
	Limit        int
}

type ElasticsearchClient interface {
	Test(ctx context.Context, cfg ESConfig) error
	QueryLogs(ctx context.Context, query ESLogQuery) ([]dto.LogItem, error)
}

type HTTPElasticsearchClient struct{}

func NewElasticsearchClient() ElasticsearchClient {
	return &HTTPElasticsearchClient{}
}

func (c *HTTPElasticsearchClient) Test(ctx context.Context, cfg ESConfig) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(cfg.Endpoint, "/"), nil)
	if err != nil {
		return err
	}
	if cfg.Username != "" {
		req.SetBasicAuth(cfg.Username, cfg.Password)
	}
	resp, err := (&http.Client{Timeout: cfg.Timeout}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("elasticsearch returned %s", resp.Status)
	}
	return nil
}

func (c *HTTPElasticsearchClient) QueryLogs(ctx context.Context, query ESLogQuery) ([]dto.LogItem, error) {
	if query.Limit <= 0 {
		query.Limit = 100
	}
	timeField := query.TimeField
	if timeField == "" {
		timeField = "@timestamp"
	}
	must := []map[string]interface{}{}
	if query.Keyword != "" {
		must = append(must, map[string]interface{}{"query_string": map[string]interface{}{"query": query.Keyword}})
	}
	if query.LogLevel != "" {
		must = append(must, map[string]interface{}{"query_string": map[string]interface{}{"query": query.LogLevel, "fields": []string{"level", "log.level", "severity", "message"}}})
	}
	filter := []map[string]interface{}{}
	if query.TimeStart != nil || query.TimeEnd != nil {
		rng := map[string]interface{}{}
		if query.TimeStart != nil {
			rng["gte"] = query.TimeStart.Format(time.RFC3339)
		}
		if query.TimeEnd != nil {
			rng["lte"] = query.TimeEnd.Format(time.RFC3339)
		}
		filter = append(filter, map[string]interface{}{"range": map[string]interface{}{timeField: rng}})
	}
	body := map[string]interface{}{
		"size":  query.Limit,
		"sort":  []map[string]interface{}{{timeField: map[string]string{"order": "desc"}}},
		"query": map[string]interface{}{"bool": map[string]interface{}{"must": must, "filter": filter}},
	}
	if len(must) == 0 && len(filter) == 0 {
		body["query"] = map[string]interface{}{"match_all": map[string]interface{}{}}
	}
	payload, _ := json.Marshal(body)
	url := fmt.Sprintf("%s/%s/_search", strings.TrimRight(query.Endpoint, "/"), query.IndexPattern)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if query.Username != "" {
		req.SetBasicAuth(query.Username, query.Password)
	}
	resp, err := (&http.Client{Timeout: query.Timeout}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("elasticsearch search returned %s: %s", resp.Status, truncateForError(string(data)))
	}
	var parsed struct {
		Hits struct {
			Hits []struct {
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, err
	}
	items := make([]dto.LogItem, 0, len(parsed.Hits.Hits))
	for _, hit := range parsed.Hits.Hits {
		items = append(items, mapESLogItem(hit.Source, timeField))
	}
	return items, nil
}

func mapESLogItem(source map[string]interface{}, timeField string) dto.LogItem {
	raw, _ := json.Marshal(source)
	item := dto.LogItem{Raw: string(raw)}
	for _, key := range []string{"message", "msg", "log", "error.message"} {
		if value, ok := source[key].(string); ok && value != "" {
			item.Message = value
			break
		}
	}
	if item.Message == "" {
		item.Message = item.Raw
	}
	for _, key := range []string{"level", "log.level", "severity"} {
		if value, ok := source[key].(string); ok {
			item.Level = value
			break
		}
	}
	for _, key := range []string{"service.name", "service", "host.name", "host"} {
		if value, ok := source[key].(string); ok {
			item.Source = value
			break
		}
	}
	if value, ok := source[timeField].(string); ok {
		if ts, err := time.Parse(time.RFC3339, value); err == nil {
			item.Timestamp = &ts
		}
	}
	return item
}

func truncateForError(text string) string {
	if len(text) <= 300 {
		return text
	}
	return text[:300]
}
