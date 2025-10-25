package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"llm-language-server/lsp"
	"llm-language-server/provider"
	"os"
)

var reqCancel = make(map[int]func())

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
		var params lsp.TextDocumentParams
		err = json.Unmarshal(jsonParams, &params)
		if err != nil {
			return err
		}
		lsp.SetState(params)
		output := lsp.NewLogMesssage("textDocument/didOpen - OK", lsp.Info)
		writer.Write(lsp.NewNotificationMessage(output))
		return nil
	case "textDocument/didFocus":
		jsonParams, err := json.Marshal(request.Params)
		if err != nil {
			return err
		}
		var params lsp.TextDocumentParams
		err = json.Unmarshal(jsonParams, &params)
		if err != nil {
			return err
		}
		lsp.SetState(params)
		output := lsp.NewLogMesssage("textDocument/didFocus - OK", lsp.Info)
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
	case "textDocument/didClose":
		jsonParams, err := json.Marshal(request.Params)
		if err != nil {
			return err
		}
		var params lsp.TextDocumentParams
		err = json.Unmarshal(jsonParams, &params)
		if err != nil {
			return err
		}
		lsp.DeleteState(params)
		output := lsp.NewLogMesssage("textDocument/didClose - OK", lsp.Info)
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
		items, err := provider.CurrentProvider.Generate(ctx, params)
		if err != nil {
			output := lsp.NewLogMesssage(fmt.Sprintf("textDocument/inlineCompletion - ERROR: %s", err.Error()), lsp.Error)
			writer.Write(lsp.NewNotificationMessage(output))
		}
		output := lsp.NewInlineCompletionResponse(request.ID, lsp.InlineCompletionResult{Items: items})
		if len(items) == 0 {
			output := lsp.NewLogMesssage("textDocument/inlineCompletion - DEBUG: no items", lsp.Debug)
			writer.Write(lsp.NewNotificationMessage(output))
		} else {
			output := lsp.NewLogMesssage(fmt.Sprintf("textDocument/inlineCompletion - DEBUG: %s", items[0].InsertText), lsp.Debug)
			writer.Write(lsp.NewNotificationMessage(output))
		}
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
		c, exists := reqCancel[params.ID]
		if !exists {
			return fmt.Errorf("$/cancelRequest - Request %d not found", params.ID)
		}
		c()
		delete(reqCancel, params.ID)
		output := lsp.NewLogMesssage("$/cancelRequest - OK", lsp.Info)
		writer.Write(lsp.NewNotificationMessage(output))
		return nil
	}

	return fmt.Errorf("handler not found: %s", request.Method)
}
