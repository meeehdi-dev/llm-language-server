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
	"strings"
)

type OllamaModelParams struct {
	NumCtx      int      `json:"num_ctx"`
	NumPredict  int      `json:"num_predict"`
	Temperature int      `json:"temperature"`
	TopP        float32  `json:"top_p"`
	Stop        []string `json:"stop"`
}

type OllamaProvider struct {
	URL         string
	Model       string
	ModelParams OllamaModelParams
}

type OllamaInitializationParams struct {
	URL         string            `json:"url"`
	Model       string            `json:"model"`
	ModelParams OllamaModelParams `json:"model_params"`
}

type OllamaModelDetails struct {
	ParentModel       string   `json:"parent_model"`
	Format            string   `json:"format"`
	Family            string   `json:"family"`
	Families          []string `json:"families"`
	ParameterSize     string   `json:"parameter_size"`
	QuantizationLevel string   `json:"quantization_level"`
}

type OllamaModel struct {
	Name       string             `json:"name"`
	Model      string             `json:"model"`
	ModifiedAt string             `json:"modified_at"`
	Size       int                `json:"size"`
	Digest     string             `json:"digest"`
	Details    OllamaModelDetails `json:"details"`
}

type OllamaTagsResponse struct {
	Models []OllamaModel `json:"models"`
}

func (p *OllamaProvider) getModels() ([]OllamaModel, error) {
	models := make([]OllamaModel, 0)

	req, err := http.NewRequest("GET", p.URL+"/tags", nil)
	if err != nil {
		return models, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return models, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return models, err
	}

	var response OllamaTagsResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return models, err
	}

	return response.Models, nil
}

type OllamaPullResponse struct {
}

func (p *OllamaProvider) pullModel(name string) error {
	data := map[string]string{
		"name": name,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", p.URL+"/pull", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var response OllamaPullResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}

	return nil
}

func (p *OllamaProvider) preloadModel() error {
	data := map[string]any{
		"model":  p.Model,
		"stream": false,
		"options": map[string]any{
			"num_ctx":     p.ModelParams.NumCtx,
			"num_predict": p.ModelParams.NumPredict,
			"temperature": p.ModelParams.Temperature,
			"top_p":       p.ModelParams.TopP,
			"stop":        p.ModelParams.Stop,
		},
		"keep_alive": "1h",
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", p.URL+"/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var response OllamaGenerateResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return err
	}

	// NOTE: should we check if response.Done is true?

	return nil
}

func (p *OllamaProvider) Initialize(params any) error {
	jsonParams, err := json.Marshal(params)
	if err != nil {
		return err
	}
	var ollamaParams OllamaInitializationParams
	err = json.Unmarshal(jsonParams, &ollamaParams)
	if err != nil {
		return err
	}

	p.URL = ollamaParams.URL
	p.Model = ollamaParams.Model
	p.ModelParams = ollamaParams.ModelParams

	models, err := p.getModels()
	if err != nil {
		return err
	}

	found := false
	for _, model := range models {
		if model.Name == p.Model {
			found = true
		}
	}

	if !found {
		err := p.pullModel(p.Model)
		if err != nil {
			return err
		}
	}

	err = p.preloadModel()
	if err != nil {
		return err
	}

	return nil
}

type OllamaGenerateResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
}

func truncate(text string, index int, ctx int) (string, string) {
	promptLines := strings.Split(text[:index], "\n")
	suffixLines := strings.Split(text[index:], "\n")

	prompt := promptLines[len(promptLines)-1]
	suffix := suffixLines[len(suffixLines)-1]

	ctxSize := len(prompt+suffix) / 4

	promptLinesCount := 1
	suffixLinesCount := 1

	ctxInc := true
	for ctxInc {
		ctxInc = false

		if suffixLinesCount < len(suffixLines) {
			suffix_line := suffixLines[suffixLinesCount]
			if ctxSize+len(suffix_line) < ctx {
				suffix = suffix + "\n" + suffix_line
				suffixLinesCount++
				ctxInc = true
			}
		}

		if promptLinesCount < len(promptLines) {
			promptLine := promptLines[len(promptLines)-promptLinesCount-1]
			if ctxSize+len(promptLine) < ctx {
				prompt = promptLine + "\n" + prompt
				promptLinesCount++
				ctxInc = true
			}
		}
	}

	return prompt, suffix
}

func (p *OllamaProvider) Generate(ctx context.Context, params lsp.InlineCompletionParams) ([]lsp.CompletionItem, error) {
	items := make([]lsp.CompletionItem, 0)

	document, exists := lsp.State[string(params.TextDocument.Uri)]
	if !exists {
		return items, fmt.Errorf("document not found %s", params.TextDocument.Uri)
	}

	index := lsp.FindIndex(document.Text, params.Position.Line, params.Position.Character)
	prompt, suffix := truncate(document.Text, index, p.ModelParams.NumCtx)

	cacheKey := cache.GetKey(prompt, suffix)
	cacheValue, exists := cache.Get(cacheKey)
	if exists {
		return cacheValue, nil
	}

	// TODO: handle stream true + lsp progress?
	data := map[string]any{
		"model":  p.Model,
		"prompt": prompt,
		"suffix": suffix,
		"stream": false,
		"options": map[string]any{
			"num_ctx":     p.ModelParams.NumCtx,
			"num_predict": p.ModelParams.NumPredict,
			"temperature": p.ModelParams.Temperature,
			"top_p":       p.ModelParams.TopP,
			"stop":        p.ModelParams.Stop,
		},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return items, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.URL+"/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return items, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

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

	var response OllamaGenerateResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return items, err
	}

	if response.Done {
		items = append(items, lsp.CompletionItem{Label: "0", InsertText: response.Response})
	}

	cache.Set(cacheKey, items)

	return items, nil
}
