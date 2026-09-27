package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Client struct {
	apiKey string
	model  string
	http   *http.Client
}

func New(apiKey, model string, httpClient *http.Client) *Client {
	return &Client{apiKey: apiKey, model: model, http: httpClient}
}

func (c *Client) CompleteJSON(ctx context.Context, systemPrompt, userPrompt string) ([]byte, error) {
	if c.apiKey == "" {
		return nil, errors.New("OPENROUTER_API_KEY is not configured")
	}
	payload := map[string]any{
		"model": c.model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"response_format": map[string]string{"type": "json_object"},
		"temperature":     0.8,
		"max_tokens":      800,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://openrouter.ai/api/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "http://localhost")
	req.Header.Set("X-OpenRouter-Title", "Nodo")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call OpenRouter: %w", err)
	}
	defer resp.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenRouter returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}
	var completion struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(responseBody, &completion); err != nil {
		return nil, fmt.Errorf("decode OpenRouter response: %w", err)
	}
	if len(completion.Choices) == 0 {
		return nil, errors.New("OpenRouter returned no choices")
	}
	content := strings.TrimSpace(completion.Choices[0].Message.Content)
	if start, end := strings.Index(content, "{"), strings.LastIndex(content, "}"); start >= 0 && end > start {
		content = content[start : end+1]
	}
	return []byte(content), nil
}
