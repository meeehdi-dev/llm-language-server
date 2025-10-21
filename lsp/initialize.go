package lsp

type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type InitializationOptions struct {
	Provider string `json:"provider"`
	Params   any    `json:"params"`
}

type InitializeParams struct {
	ClientInfo            ClientInfo            `json:"clientInfo"`
	InitializationOptions InitializationOptions `json:"initializationOptions"`
}

type TextDocumentSyncKind int

const (
	None        TextDocumentSyncKind = 0
	Full        TextDocumentSyncKind = 1
	Incremental TextDocumentSyncKind = 2
)

type ServerCapabilities struct {
	TextDocumentSync TextDocumentSyncKind `json:"textDocumentSync"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   ServerInfo         `json:"serverInfo"`
}

func NewInitializeResponse(id int) ResponseMessage {
	return ResponseMessage{
		Message: Message{
			JsonRPC: "2.0",
		},
		ID: id,
		Result: InitializeResult{
			Capabilities: ServerCapabilities{TextDocumentSync: Incremental},
			ServerInfo: ServerInfo{
				Name:    "llm-language-server",
				Version: "1.0.0-0",
			},
		},
	}
}
