package main

import (
	"bufio"
	"flag"
	"fmt"
	"llm-language-server/handler"
	"llm-language-server/jsonrpc"
	"llm-language-server/lsp"
	"os"
)

func HandleMessage(writer *os.File, message []byte) {
	err := handler.HandleRequestMessage(writer, message)
	if err != nil {
		writer.Write(lsp.NewNotificationMessage(lsp.NewLogMesssage(err.Error(), lsp.Error)))
	}
}

func main() {
	debug := flag.Bool("debug", false, "enable debug log messages")
	flag.Parse()

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Split(jsonrpc.Split)

	writer := os.Stdout
	writer.Write(lsp.NewNotificationMessage(lsp.NewLogMesssage(fmt.Sprintf("Successfully started! (debug: %t)", *debug), lsp.Info)))

	for scanner.Scan() {
		bytes := scanner.Bytes()
		message, err := jsonrpc.DecodeMessage(bytes)
		if err != nil {
			writer.Write(lsp.NewNotificationMessage(lsp.NewLogMesssage(err.Error(), lsp.Error)))
			continue
		}
		if *debug {
			writer.Write(lsp.NewNotificationMessage(lsp.NewLogMesssage(fmt.Sprintf("Successfully received message: %s", string(message)), lsp.Debug)))
		}
		HandleMessage(writer, message)
	}
}
