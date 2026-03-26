package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Provider identifies which LLM backend to use
type Provider string

const (
	ProviderOpenAI   Provider = "openai"
	ProviderDeepSeek Provider = "deepseek"
	ProviderClaude   Provider = "claude"
)

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LLMClient provides a unified interface across AI providers
type LLMClient struct {
	Provider   Provider
	APIKey     string
	Model      string
	BaseURL    string
	HTTPClient *http.Client
}

// LLMResponse is the parsed response from any provider
type LLMResponse struct {
	Content string
}

// NewLLMClient creates a client for the specified provider
func NewLLMClient(provider Provider, apiKey, model string) *LLMClient {
	c := &LLMClient{
		Provider: provider,
		APIKey:   apiKey,
		Model:    model,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}

	switch provider {
	case ProviderOpenAI:
		c.BaseURL = "https://api.openai.com"
		if c.Model == "" {
			c.Model = "gpt-4o-mini"
		}
	case ProviderDeepSeek:
		c.BaseURL = "https://api.deepseek.com"
		if c.Model == "" {
			c.Model = "deepseek-chat"
		}
	case ProviderClaude:
		c.BaseURL = "https://api.anthropic.com"
		if c.Model == "" {
			c.Model = "claude-3-5-haiku-20241022"
		}
	}

	return c
}

// Chat sends a prompt and returns the model's response
func (c *LLMClient) Chat(ctx context.Context, systemPrompt string, messages []Message) (*LLMResponse, error) {
	if c.Provider == ProviderClaude {
		return c.chatClaude(ctx, systemPrompt, messages)
	}
	return c.chatOpenAICompatible(ctx, systemPrompt, messages)
}

// chatOpenAICompatible handles OpenAI and DeepSeek (same API format)
func (c *LLMClient) chatOpenAICompatible(ctx context.Context, systemPrompt string, messages []Message) (*LLMResponse, error) {
	allMessages := make([]Message, 0, len(messages)+1)
	if systemPrompt != "" {
		allMessages = append(allMessages, Message{Role: "system", Content: systemPrompt})
	}
	allMessages = append(allMessages, messages...)

	body := map[string]interface{}{
		"model":       c.Model,
		"messages":    allMessages,
		"temperature": 0.3,
		"max_tokens":  1024,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("ai: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("ai: create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ai: send request to %s: %w", c.Provider, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ai: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ai: %s returned status %d: %s", c.Provider, resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("ai: unmarshal response: %w", err)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("ai: %s returned no choices", c.Provider)
	}

	return &LLMResponse{Content: result.Choices[0].Message.Content}, nil
}

// chatClaude handles Anthropic's Messages API
func (c *LLMClient) chatClaude(ctx context.Context, systemPrompt string, messages []Message) (*LLMResponse, error) {
	claudeMessages := make([]map[string]string, 0, len(messages))
	for _, m := range messages {
		claudeMessages = append(claudeMessages, map[string]string{
			"role":    m.Role,
			"content": m.Content,
		})
	}

	body := map[string]interface{}{
		"model":      c.Model,
		"messages":   claudeMessages,
		"max_tokens": 1024,
	}
	if systemPrompt != "" {
		body["system"] = systemPrompt
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("ai: marshal claude request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("ai: create claude request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ai: send claude request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ai: read claude response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ai: claude returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("ai: unmarshal claude response: %w", err)
	}
	if len(result.Content) == 0 {
		return nil, fmt.Errorf("ai: claude returned no content")
	}

	return &LLMResponse{Content: result.Content[0].Text}, nil
}
