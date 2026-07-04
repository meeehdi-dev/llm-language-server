package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"llm-language-server/cache"
	"llm-language-server/lsp"
)

type DeepSeekModelParams struct {
	LogProbs    *uint
	MaxTokens   *uint
	Temperature *float64
	TopP        *float64
	Stop        *[]string
}

type DeepSeekProvider struct {
	Model       string
	ApiKey      string
	ModelParams DeepSeekModelParams
}

type DeepSeekModelDetails struct {
	LogProbs    *uint     `json:"logprobs"`
	MaxTokens   *uint     `json:"max_tokens"`
	Temperature *float64  `json:"temperature"`
	TopP        *float64  `json:"top_p"`
	Stop        *[]string `json:"stop"`
}

type DeepSeekInitializationParams struct {
	Model       string               `json:"model"`
	ApiKey      string               `json:"api_key"`
	ModelParams DeepSeekModelDetails `json:"model_params"`
}

type DeepSeekChoice struct {
	Index int    `json:"index"`
	Text  string `json:"text"`
}

type DeepSeekResponse struct {
	Choices []DeepSeekChoice `json:"choices"`
}

func (p *DeepSeekProvider) Initialize(params any) error {
	jsonParams, err := json.Marshal(params)
	if err != nil {
		return err
	}
	var deepSeekParams DeepSeekInitializationParams
	err = json.Unmarshal(jsonParams, &deepSeekParams)
	if err != nil {
		return err
	}

	p.Model = deepSeekParams.Model
	p.ApiKey = deepSeekParams.ApiKey
	p.ModelParams = deepSeekParams.ModelParams

	return nil
}

func (p *DeepSeekProvider) Generate(
	ctx context.Context,
	params lsp.InlineCompletionParams,
) ([]lsp.CompletionItem, error) {
	items := make([]lsp.CompletionItem, 0)

	if p.ApiKey == "" {
		return items, fmt.Errorf("api key not set")
	}

	document, exists := lsp.State[string(params.TextDocument.Uri)]
	if !exists {
		return items, fmt.Errorf("document not found %s", params.TextDocument.Uri)
	}

	index := lsp.FindIndex(document.Text, params.Position.Line, params.Position.Character)
	// DeepSeek FIM formatting is limited to a context window of 4k tokens for now
	prompt, suffix := truncate(document.Text, index, 4000)

	cacheValue, exists := cache.Get(prompt, suffix)
	if exists {
		return cacheValue, nil
	}

	data := map[string]any{
		"model":       p.Model,
		"prompt":      prompt,
		"suffix":      suffix,
		"logprobs":    p.ModelParams.LogProbs,
		"max_tokens":  p.ModelParams.MaxTokens,
		"temperature": p.ModelParams.Temperature,
		"top_p":       p.ModelParams.TopP,
		"stop":        p.ModelParams.Stop,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return items, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		"https://api.deepseek.com/beta/completions",
		bytes.NewBuffer(jsonData),
	)
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

	var response DeepSeekResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return items, err
	}

	for _, choice := range response.Choices {
		items = append(
			items,
			lsp.CompletionItem{
				Label:      strconv.Itoa(choice.Index),
				InsertText: choice.Text,
			},
		)
	}

	cache.Set(prompt, suffix, items)

	return items, nil
}
