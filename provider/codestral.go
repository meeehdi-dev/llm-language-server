package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"llm-language-server/cache"
	"llm-language-server/lsp"
	"net/http"
	"strconv"
	"sync"
)

type CodestralProvider struct {
	ApiKey   string
	endpoint string
	mutex    sync.RWMutex
}

type CodestralInitializationParams struct {
	ApiKey string `json:"api_key"`
}

func (p *CodestralProvider) Initialize(params any) error {
	jsonParams, err := json.Marshal(params)
	if err != nil {
		return err
	}
	var codestralParams CodestralInitializationParams
	err = json.Unmarshal(jsonParams, &codestralParams)
	if err != nil {
		return err
	}

	p.ApiKey = codestralParams.ApiKey

	return nil
}

type CodestralMessage struct {
	Content   string `json:"content"`
	Role      string `json:"role"`
	ToolCalls any    `json:"tool_calls"`
}

type CodestralChoice struct {
	FinishReason string           `json:"finish_reason"`
	Index        int              `json:"index"`
	Message      CodestralMessage `json:"message"`
}

type CodestralUsage struct {
	CompletionTokens int `json:"completion_tokens"`
	PromptTokens     int `json:"prompt_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type CodestralResponse struct {
	Choices []CodestralChoice `json:"choices"`
	Created int               `json:"created"`
	ID      string            `json:"id"`
	Model   string            `json:"model"`
	Object  string            `json:"object"`
	Usage   CodestralUsage    `json:"usage"`
}

func (p *CodestralProvider) getEndpoint(ctx context.Context) (string, error) {
	p.mutex.RLock()
	if p.endpoint != "" {
		endpoint := p.endpoint
		p.mutex.RUnlock()
		return endpoint, nil
	}
	p.mutex.RUnlock()

	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Double-check after acquiring the write lock
	if p.endpoint != "" {
		return p.endpoint, nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.mistral.ai/v1/models", nil)
	if err != nil {
		return "", fmt.Errorf("creating check request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.ApiKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("checking api key type: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		p.endpoint = "https://codestral.mistral.ai/v1/fim/completions"
	} else {
		p.endpoint = "https://api.mistral.ai/v1/fim/completions"
	}

	return p.endpoint, nil
}

func (p *CodestralProvider) Generate(ctx context.Context, params lsp.InlineCompletionParams) ([]lsp.CompletionItem, error) {
	items := make([]lsp.CompletionItem, 0)

	if p.ApiKey == "" {
		return items, fmt.Errorf("api key not set")
	}

	document, exists := lsp.State[string(params.TextDocument.Uri)]
	if !exists {
		return items, fmt.Errorf("document not found %s", params.TextDocument.Uri)
	}

	index := lsp.FindIndex(document.Text, params.Position.Line, params.Position.Character)
	prompt := document.Text[:index]
	suffix := document.Text[index:]

	cacheValue, exists := cache.Get(prompt, suffix)
	if exists {
		return cacheValue, nil
	}

	data := map[string]any{
		"model":       "codestral-latest",
		"prompt":      document.Text[:index],
		"suffix":      document.Text[index:],
		"stop":        []string{"\n\n"},
		"max_tokens":  64,
		"temperature": 0,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return items, err
	}

	endpoint, err := p.getEndpoint(ctx)
	if err != nil {
		return items, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return items, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.ApiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		if err == context.Canceled {
			return items, nil
		}
		return items, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return items, err
	}

	var response CodestralResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return items, err
	}

	for _, choice := range response.Choices {
		items = append(items, lsp.CompletionItem{Label: strconv.Itoa(choice.Index), InsertText: choice.Message.Content})
	}

	cache.Set(prompt, suffix, items)

	return items, nil
}
