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
)

type CodestralProvider struct {
	ApiKey string
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

	cacheKey := cache.GetKey(prompt, suffix)
	cacheValue, exists := cache.Get(cacheKey)
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

	req, err := http.NewRequestWithContext(ctx, "POST", "https://codestral.mistral.ai/v1/fim/completions", bytes.NewBuffer(jsonData))
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

	cache.Set(cacheKey, items)

	return items, nil
}
