package lsp

import (
	"fmt"
)

type DocumentUri string

type TextDocumentIdentifier struct {
	Uri DocumentUri `json:"uri"`
}

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type TextDocumentItem struct {
	Uri        DocumentUri `json:"uri"`
	LanguageId string      `json:"languageId"`
	Version    int         `json:"version"`
	Text       string      `json:"text"`
}

type TextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type TextDocument struct {
	Version int
	Text    string
}

var State = make(map[string]TextDocument)

func SetState(params TextDocumentParams) {
	State[string(params.TextDocument.Uri)] = TextDocument{
		Version: params.TextDocument.Version,
		Text:    params.TextDocument.Text,
	}
}

func DeleteState(params TextDocumentParams) {
	delete(State, string(params.TextDocument.Uri))
}

func FindIndex(text string, line int, col int) int {
	l := 0
	for i, char := range text {
		if l == line {
			return i + col
		}
		if char == '\n' {
			l++
		}
	}

	return -1
}

func UpdateState(params DidChangeTextDocumentParams) error {
	current, exists := State[string(params.TextDocument.Uri)]
	if !exists {
		return fmt.Errorf("Document not found: %s", params.TextDocument.Uri)
	}
	if current.Version > params.TextDocument.Version {
		return fmt.Errorf("version mismatch: current (%d) > new (%d)", current.Version, params.TextDocument.Version)
	}

	for _, change := range params.ContentChanges {
		startIndex := FindIndex(current.Text, change.Range.Start.Line, change.Range.Start.Character)
		endIndex := FindIndex(current.Text, change.Range.End.Line, change.Range.End.Character)
		current.Text = current.Text[:startIndex] + change.Text + current.Text[endIndex:]
	}

	// NOTE: dont like it but whatever
	SetState(TextDocumentParams{TextDocument: TextDocumentItem{Uri: params.TextDocument.Uri, Text: current.Text, Version: params.TextDocument.Version}})

	return nil
}
