package jsonrpc_test

import (
	"encoding/json"
	"llm-language-server/jsonrpc"
	"llm-language-server/lsp"
	"testing"
)

type EncodingExample struct {
	Method string `json:"method"`
}

func TestEncodeMessage(t *testing.T) {
	expected := "Content-Length: 36\r\n\r\n{\"method\":\"textDocument/completion\"}"
	actual := jsonrpc.EncodeMessage(EncodingExample{Method: "textDocument/completion"})
	if actual != expected {
		t.Errorf("Expected %s, got %s", expected, actual)
	}
}

func TestDecodeMessage(t *testing.T) {
	incoming := []byte("Content-Length: 36\r\n\r\n{\"method\":\"textDocument/completion\"}")
	content, err := jsonrpc.DecodeMessage(incoming)
	contentLength := len(content)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	expectedLength := 36
	if contentLength != expectedLength {
		t.Errorf("Expected content length %d, got %d", expectedLength, contentLength)
	}

	var request lsp.Request
	err = json.Unmarshal(content, &request)
	if err != nil {
		t.Errorf("%s", err.Error())
	}

	expectedMethod := "textDocument/completion"
	if request.Method != expectedMethod {
		t.Errorf("Expected method %s, got %s", expectedMethod, request.Method)
	}
}
