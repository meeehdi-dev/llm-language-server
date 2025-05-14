package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"llm-language-server/lsp"
	"llm-language-server/provider"
	"os"
)

var generateRequest = -1
var generateCancel context.CancelFunc = nil

func HandleRequestMessage(writer *os.File, message []byte) error {
	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	var request lsp.RequestMessage
	err := json.Unmarshal(message, &request)
	if err != nil {
		return err
	}

	switch request.Method {
	case "initialize":
		jsonParams, err := json.Marshal(request.Params)
		if err != nil {
			return err
		}
		var params lsp.InitializeParams
		err = json.Unmarshal(jsonParams, &params)
		if err != nil {
			return err
		}
		err = provider.Initialize(params.InitializationOptions, params.InitializationOptions.Params)
		if err != nil {
			return err
		}
		output := lsp.NewInitializeResponse(request.ID)
		writer.Write(lsp.NewResponseMessage(output))
		return nil
	case "initialized":
		output := lsp.NewLogMesssage("initialized - OK", lsp.Info)
		writer.Write(lsp.NewNotificationMessage(output))
		return nil
	case "textDocument/didOpen":
		jsonParams, err := json.Marshal(request.Params)
		if err != nil {
			return err
		}
		var params lsp.DidOpenTextDocumentParams
		err = json.Unmarshal(jsonParams, &params)
		if err != nil {
			return err
		}
		lsp.SetState(params)
		output := lsp.NewLogMesssage("textDocument/didOpen - OK", lsp.Info)
		writer.Write(lsp.NewNotificationMessage(output))
		return nil
	case "textDocument/didChange":
		jsonParams, err := json.Marshal(request.Params)
		if err != nil {
			return err
		}
		var params lsp.DidChangeTextDocumentParams
		err = json.Unmarshal(jsonParams, &params)
		if err != nil {
			return err
		}
		err = lsp.UpdateState(params)
		if err != nil {
			return err
		}
		output := lsp.NewLogMesssage("textDocument/didChange - OK", lsp.Info)
		writer.Write(lsp.NewNotificationMessage(output))
		return nil
	case "textDocument/didSave":
		output := lsp.NewLogMesssage("textDocument/didSave - OK", lsp.Info)
		writer.Write(lsp.NewNotificationMessage(output))
		return nil
	case "textDocument/inlineCompletion":
		jsonParams, err := json.Marshal(request.Params)
		if err != nil {
			return err
		}
		var params lsp.InlineCompletionParams
		err = json.Unmarshal(jsonParams, &params)
		if err != nil {
			return err
		}
		if generateRequest != -1 {
			generateCancel()
			generateRequest = -1
		}
		generateRequest = request.ID
		generateCancel = cancel
		items, err := provider.CurrentProvider.Generate(ctx, params)
		if err != nil {
			output := lsp.NewLogMesssage(fmt.Sprintf("textDocument/inlineCompletion - ERROR: %s", err.Error()), lsp.Error)
			writer.Write(lsp.NewNotificationMessage(output))
		}
		output := lsp.NewInlineCompletionResponse(request.ID, lsp.InlineCompletionResult{Items: items})
		writer.Write(lsp.NewResponseMessage(output))
		return nil
	case "$/cancelRequest":
		jsonParams, err := json.Marshal(request.Params)
		if err != nil {
			return err
		}
		var params lsp.CancelRequestParams
		err = json.Unmarshal(jsonParams, &params)
		if err != nil {
			return err
		}
		if generateRequest == params.ID {
			generateCancel()
			generateRequest = -1
		}
		output := lsp.NewLogMesssage("$/cancelRequest - OK", lsp.Info)
		writer.Write(lsp.NewNotificationMessage(output))
		return nil
	}

	return errors.New(fmt.Sprintf("handler not found: %s", request.Method))
}
