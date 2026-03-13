package lsp

import (
	"testing"
)

func TestFindIndex(t *testing.T) {
	text := "abc\ndef\n"
	
	tests := []struct {
		line     int
		col      int
		expected int
	}{
		{0, 0, 0},
		{0, 2, 2},
		{1, 0, 4},
		{1, 3, 7},
		{2, 0, 8}, // after the last newline
		{3, 0, -1}, // out of bounds line
	}

	for _, tc := range tests {
		actual := FindIndex(text, tc.line, tc.col)
		if actual != tc.expected {
			t.Errorf("FindIndex(%q, %d, %d) = %d, expected %d", text, tc.line, tc.col, actual, tc.expected)
		}
	}
}

func TestUpdateState_FullSync(t *testing.T) {
	State = make(map[string]TextDocument)
	uri := DocumentUri("file://test.txt")
	SetState(TextDocumentParams{
		TextDocument: TextDocumentItem{
			Uri:        uri,
			LanguageId: "go",
			Version:    1,
			Text:       "hello world",
		},
	})

	err := UpdateState(DidChangeTextDocumentParams{
		TextDocument: VersionedTextDocumentIdentifier{
			Uri:     uri,
			Version: 2,
		},
		ContentChanges: []TextDocumentContentChangeEvent{
			{
				Text: "new text",
			},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	doc := State[string(uri)]
	if doc.Text != "new text" {
		t.Errorf("expected text 'new text', got %q", doc.Text)
	}
}

func TestUpdateState_Incremental(t *testing.T) {
	State = make(map[string]TextDocument)
	uri := DocumentUri("file://test.txt")
	SetState(TextDocumentParams{
		TextDocument: TextDocumentItem{
			Uri:        uri,
			LanguageId: "go",
			Version:    1,
			Text:       "hello\nworld",
		},
	})

	err := UpdateState(DidChangeTextDocumentParams{
		TextDocument: VersionedTextDocumentIdentifier{
			Uri:     uri,
			Version: 2,
		},
		ContentChanges: []TextDocumentContentChangeEvent{
			{
				Range: &Range{
					Start: Position{Line: 1, Character: 0},
					End:   Position{Line: 1, Character: 5},
				},
				Text: "gophers",
			},
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	doc := State[string(uri)]
	if doc.Text != "hello\ngophers" {
		t.Errorf("expected text 'hello\\ngophers', got %q", doc.Text)
	}
}
