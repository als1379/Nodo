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

type ConfigurationError struct{ message string }

func (e *ConfigurationError) Error() string              { return e.message }
func (e *ConfigurationError) IsConfigurationError() bool { return true }

type ProviderError struct {
	Status int
	Body   string
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("OpenRouter returned status %d: %s", e.Status, e.Body)
}
func (e *ProviderError) StatusCode() int { return e.Status }

func New(apiKey, model string, httpClient *http.Client) *Client {
	return &Client{apiKey: apiKey, model: model, http: httpClient}
}

func (c *Client) CompleteJSON(ctx context.Context, systemPrompt, userPrompt string) ([]byte, error) {
	return c.completeJSON(ctx, systemPrompt, userPrompt, map[string]any{"type": "json_object"})
}

func (c *Client) CompleteJSONSchema(ctx context.Context, systemPrompt, userPrompt, name string, schema map[string]any) ([]byte, error) {
	return c.completeJSON(ctx, systemPrompt, userPrompt, map[string]any{
		"type":        "json_schema",
		"json_schema": map[string]any{"name": name, "strict": true, "schema": schema},
	})
}

func (c *Client) completeJSON(ctx context.Context, systemPrompt, userPrompt string, responseFormat map[string]any) ([]byte, error) {
	if c.apiKey == "" {
		return nil, &ConfigurationError{message: "OPENROUTER_API_KEY is not configured"}
	}
	var lastErr error
	for _, model := range strings.Split(c.model, ",") {
		content, err := c.completeModel(ctx, strings.TrimSpace(model), systemPrompt, userPrompt, responseFormat)
		if err == nil {
			return content, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func (c *Client) completeModel(ctx context.Context, model, systemPrompt, userPrompt string, responseFormat map[string]any) ([]byte, error) {
	payload := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"response_format": responseFormat,
		"provider":        map[string]any{"require_parameters": true, "allow_fallbacks": true},
		"reasoning":       map[string]string{"effort": "minimal"},
		"temperature":     0.2,
		"max_tokens":      4000,
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
		return nil, &ProviderError{Status: resp.StatusCode, Body: strings.TrimSpace(string(responseBody))}
	}
	var completion struct {
		Choices []struct {
			FinishReason string `json:"finish_reason"`
			Message      struct {
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
	if reason := completion.Choices[0].FinishReason; reason != "" && reason != "stop" {
		return nil, fmt.Errorf("OpenRouter completion ended with finish_reason %q", reason)
	}
	content := strings.TrimSpace(completion.Choices[0].Message.Content)
	if start, end := strings.Index(content, "{"), strings.LastIndex(content, "}"); start >= 0 && end > start {
		content = content[start : end+1]
	}
	return []byte(content), nil
}
