package lsp

type MessageType int

const (
	Error   MessageType = 1
	Warning             = 2
	Info                = 3
	Log                 = 4
	Debug               = 5
)

type LogMessageParams struct {
	Type    MessageType `json:"type"`
	Message string      `json:"message"`
}

func NewLogMesssage(message string, _type MessageType) NotificationMessage {
	return NotificationMessage{
		Notification: Notification{
			JsonRPC: "2.0",
			Method:  "window/logMessage",
		},
		Params: LogMessageParams{
			Type:    1,
			Message: "[llm-language-server] " + message,
		},
	}
}
