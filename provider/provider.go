package provider

import (
	"context"
	"errors"
	"fmt"
	"llm-language-server/lsp"
)

type Provider interface {
	Initialize(any) error
	Generate(context.Context, lsp.InlineCompletionParams) ([]lsp.CompletionItem, error)
}

var CurrentProvider Provider = nil

func Initialize(options lsp.InitializationOptions, params any) error {
	switch options.Provider {
	case "codestral":
		CurrentProvider = &CodestralProvider{}
		return CurrentProvider.Initialize(params)
	case "ollama":
		CurrentProvider = &OllamaProvider{}
		return CurrentProvider.Initialize(params)
	}

	return errors.New(fmt.Sprintf("invalid provider %s", options.Provider))
}
