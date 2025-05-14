build:
	go build -o llm-language-server ./main.go

test:
	go test ./...
