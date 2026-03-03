# Agent Instructions for LLM Language Server

This document outlines the guidelines, commands, and conventions for AI coding agents operating within the `llm-language-server` repository. Adhering to these instructions ensures consistency, readability, and maintainability across the codebase.

## 1. Build, Test, and Lint Commands

This project is written in Go (Go 1.24.2). The standard Go toolchain is used for all operations. There are no Makefiles, so standard go commands are the source of truth.

### Build
To compile the project and build the language server binary:
```bash
go build -o llm-language-server .
```
Always verify that the project builds successfully after making structural changes or modifying types.

### Testing
Tests are written using the standard Go `testing` package.
- **Run all tests in the repository**:
  ```bash
  go test ./...
  ```
- **Run a single test suite / package**:
  ```bash
  go test ./jsonrpc
  ```
- **Run a specific single test by name**:
  ```bash
  # Example: running TestDecodeMessage in the jsonrpc package
  go test -run ^TestDecodeMessage$ ./jsonrpc
  ```
- **Run tests with verbose output**:
  ```bash
  go test -v ./...
  ```
- **Check for race conditions**:
  ```bash
  go test -race ./...
  ```

### Linting and Formatting
Before committing, ensure that the code is correctly formatted and free of common issues.
- **Format code** (apply to all modified files):
  ```bash
  gofmt -s -w .
  # or if goimports is installed:
  goimports -w .
  ```
- **Run standard Go vet** (to catch suspicious constructs):
  ```bash
  go vet ./...
  ```

## 2. Code Style Guidelines

### 2.1 Formatting & Structure
- **Gofmt**: All code must be formatted using standard `gofmt` (tabs for indentation, standard spacing). Do not introduce custom formatting styles.
- **File Length**: Keep files focused. If a file grows beyond 500 lines, consider if it can be broken down logically.
- **Package Layout**: 
  - `main.go`: Entry point and basic JSON-RPC message routing via `stdin`/`stdout`.
  - `lsp/`: Language Server Protocol types, structures, constants, and message builders.
  - `jsonrpc/`: JSON-RPC encoding, decoding, header parsing, and byte stream splitting logic.
  - `handler/`: Handles specific LSP requests and notifications (e.g., completions).
  - `provider/`: Interfaces and implementations for LLM providers (Codestral, Ollama).
  - `cache/`: Caching logic for completion requests to minimize redundant LLM calls.

### 2.2 Imports
- **Grouping**: Group imports into standard library packages and internal/external packages. Separate them with a blank line.
- **Sorting**: Imports should be sorted alphabetically within their groups (standard behavior of `gofmt`/`goimports`).
- **Example**:
  ```go
  import (
  	"context"
  	"encoding/json"
  	"fmt"
  	"os"

  	"llm-language-server/handler"
  	"llm-language-server/lsp"
  )
  ```

### 2.3 Naming Conventions
- Follow standard Go naming conventions as detailed in *Effective Go*.
- **Exported Identifiers**: Use `PascalCase` for variables, types, and functions that are exported (e.g., `HandleMessage`, `Provider`).
- **Unexported Identifiers**: Use `camelCase` for variables, types, and functions internal to a package (e.g., `scanner`, `writer`).
- **Acronyms**: Keep acronyms in all caps (e.g., `URL`, `ID`, `JSON`, `RPC`, `HTTP` not `Url`, `Id`, `Json`).
- **Interfaces**: Interface names often end in `-er` (if applicable), though functional names like `Provider` are perfectly fine.

### 2.4 Types and Interfaces
- Keep interfaces small and focused. For instance, the `Provider` interface:
  ```go
  type Provider interface {
  	Initialize(any) error
  	Generate(context.Context, lsp.InlineCompletionParams) ([]lsp.CompletionItem, error)
  }
  ```
- Prefer strongly typed structs over maps or `any` where the schema is known.
- Map the Language Server Protocol objects exactly as they are defined in the official LSP specification using `json` tags.

### 2.5 Error Handling & Logging
- **Return Errors**: Functions should return errors as their last return value if they can fail (`func doSomething() (Result, error)`).
- **Check Errors**: Always explicitly check for errors (`if err != nil { ... }`). Do not use `_` to discard errors blindly unless strictly necessary and commented.
- **Wrapping Errors**: Use `fmt.Errorf("doing something: %w", err)` to wrap errors to preserve context.
- **LSP Communication for Errors**: Since this is a language server communicating via `stdin`/`stdout`, **do not use** `log.Fatal`, `log.Print`, or `fmt.Println` to stdout for general logging. This will corrupt the JSON-RPC byte stream.
- **Error Responses**: When an error occurs during message handling, write it back to the client as an LSP Log Message Notification using the `writer`:
  ```go
  if err != nil {
  	writer.Write(lsp.NewNotificationMessage(lsp.NewLogMesssage(err.Error(), lsp.Error)))
  	return
  }
  ```

### 2.6 Debugging and Standard Output
- Debug logs can be introduced if the server is started with the `-debug` flag.
- Never write debug logs directly to standard out unless formatted strictly as an LSP message.
- Use the LSP logging mechanism with severity `lsp.Debug` to push debug information to the client if the debug flag is enabled.

### 2.7 Context and Concurrency
- **Contexts**: Always pass `context.Context` as the first argument to functions that perform network I/O, file I/O, or might take a long time (like the `Generate` function in LLM providers).
- **Cancellation**: Respect context cancellation. The LSP client may send cancellation requests for in-flight completions, so ensure HTTP requests to LLMs use the provided context.
- **Goroutines**: If spawning goroutines, ensure they are managed correctly and cannot leak. Always handle panics within goroutines if they are interacting with the main event loop.

### 2.8 Adding New LLM Providers
- To add a new provider, implement the `Provider` interface located in `provider/provider.go`.
- Add the corresponding structure (e.g., `NewProvider{}`) in a new file inside `provider/` (e.g., `provider/newprovider.go`).
- Register the new provider within the `Initialize` function in `provider/provider.go`.
- Ensure provider payloads conform to the specific REST APIs of the target LLM.

---
**Note to AI Agents**: Always run `go build .` and `go test ./...` after performing any structural changes or adding features to ensure no compilation errors or test regressions were introduced. You must strictly abide by these rules during autonomous operation.