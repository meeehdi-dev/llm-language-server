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
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type GeminiProvider struct {
	ApiKey string
}

type GeminiInitializationParams struct {
	ApiKey string `json:"api_key"`
}

func (p *GeminiProvider) Initialize(params any) error {
	jsonParams, err := json.Marshal(params)
	if err != nil {
		return err
	}
	var geminiParams GeminiInitializationParams
	err = json.Unmarshal(jsonParams, &geminiParams)
	if err != nil {
		return err
	}

	p.ApiKey = geminiParams.ApiKey

	return nil
}

type GeminiPart struct {
	Text string `json:"text"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type GeminiGenerationConfig struct {
	Temperature     float32 `json:"temperature"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
}

type GeminiRequest struct {
	Contents         []GeminiContent        `json:"contents"`
	GenerationConfig GeminiGenerationConfig `json:"generationConfig"`
}

type GeminiCandidate struct {
	Content      GeminiContent `json:"content"`
	FinishReason string        `json:"finishReason"`
	Index        int           `json:"index"`
}

type GeminiResponse struct {
	Candidates []GeminiCandidate `json:"candidates"`
}

func (p *GeminiProvider) Generate(ctx context.Context, params lsp.InlineCompletionParams) ([]lsp.CompletionItem, error) {
	items := make([]lsp.CompletionItem, 0)

	if p.ApiKey == "" {
		return items, fmt.Errorf("api key not set")
	}

	document, exists := lsp.State[string(params.TextDocument.Uri)]
	if !exists {
		return items, fmt.Errorf("document not found %s", params.TextDocument.Uri)
	}

	index := lsp.FindIndex(document.Text, params.Position.Line, params.Position.Character)
	promptText := document.Text[:index]
	suffixText := document.Text[index:]

	cacheValue, exists := cache.Get(promptText, suffixText)
	if exists {
		return cacheValue, nil
	}

	var contextBlock strings.Builder

	// 1. Project Metadata
	if lsp.ServerWorkspaceFolder != "" {
		var rootPath string
		if strings.HasPrefix(lsp.ServerWorkspaceFolder, "file://") {
			parsedUrl, err := url.Parse(lsp.ServerWorkspaceFolder)
			if err == nil {
				rootPath = parsedUrl.Path
			}
		} else {
			rootPath = lsp.ServerWorkspaceFolder
		}

		if rootPath != "" {
			metaFiles := []string{"package.json", "go.mod", "requirements.txt", "Cargo.toml"}
			for _, file := range metaFiles {
				filePath := filepath.Join(rootPath, file)
				content, err := os.ReadFile(filePath)
				if err == nil {
					contextBlock.WriteString(fmt.Sprintf("\n--- PROJECT METADATA: %s ---\n%s\n", file, string(content)))
				}
			}
		}
	}

	// 2. Open Files
	for uri, doc := range lsp.State {
		if uri != string(params.TextDocument.Uri) {
			contextBlock.WriteString(fmt.Sprintf("\n--- OPEN FILE: %s ---\n%s\n", uri, doc.Text))
		}
	}

	langContext := ""
	if document.LanguageId != "" {
		langContext = fmt.Sprintf("You are completing code for a %s file.\n", document.LanguageId)
	}

	prompt := fmt.Sprintf(`You are an expert software engineer providing inline code completion. 
%sBelow is a file currently being edited. The cursor position is marked with <CURSOR>.
Please provide the code that should replace the <CURSOR> marker to complete the code accurately.

CRITICAL INSTRUCTIONS:
- Return ONLY the exact code completion that goes in place of <CURSOR>.
- Do NOT include any conversational text or explanations.
- Do NOT wrap the code in markdown code blocks (e.g., no %s).
- Do NOT repeat the code before or after the <CURSOR>.
%s
FILE CONTENT:
%s<CURSOR>%s`, langContext, "``` ... ```", contextBlock.String(), promptText, suffixText)

	reqData := GeminiRequest{
		Contents: []GeminiContent{
			{
				Parts: []GeminiPart{
					{Text: prompt},
				},
			},
		},
		GenerationConfig: GeminiGenerationConfig{
			Temperature:     0.0,
			MaxOutputTokens: 64,
		},
	}

	jsonData, err := json.Marshal(reqData)
	if err != nil {
		return items, err
	}

	endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-3.1-flash-lite-preview:generateContent?key=%s", p.ApiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return items, err
	}

	req.Header.Set("Content-Type", "application/json")

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

	if resp.StatusCode != http.StatusOK {
		return items, fmt.Errorf("gemini api error: %s", string(body))
	}

	var response GeminiResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return items, err
	}

	for _, candidate := range response.Candidates {
		if len(candidate.Content.Parts) > 0 {
			text := candidate.Content.Parts[0].Text

			// Clean up potential markdown formatting that the model might incorrectly add
			if strings.HasPrefix(text, "```") {
				if idx := strings.Index(text, "\n"); idx != -1 {
					text = text[idx+1:]
				} else {
					text = ""
				}
			}
			text = strings.TrimSuffix(text, "\n```")
			text = strings.TrimSuffix(text, "```")

			items = append(items, lsp.CompletionItem{
				Label:      strconv.Itoa(candidate.Index),
				InsertText: text,
			})
		}
	}

	cache.Set(promptText, suffixText, items)

	return items, nil
}
