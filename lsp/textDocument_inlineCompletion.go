package lsp

type TriggerKind int

const (
	Invoked   = 1
	Automatic = 2
)

type CompletionContext struct {
	TriggerKind      TriggerKind `json:"triggerKind"`
	TriggerCharacter string      `json:"triggerCharacter"`
}

type InlineCompletionParams struct {
	TextDocumentPositionParams `json:",inline"`
	Context                    CompletionContext `json:"context"`
}

type CompletionItem struct {
	Label      string `json:"label"`
	InsertText string `json:"insertText"`
}

type InlineCompletionResult struct {
	Items []CompletionItem `json:"items"`
}

func NewInlineCompletionResponse(id int, result InlineCompletionResult) ResponseMessage {
	return ResponseMessage{
		Message: Message{
			JsonRPC: "2.0",
		},
		ID:     id,
		Result: result,
	}
}
