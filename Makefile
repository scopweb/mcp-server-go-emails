.PHONY: build run test clean

build:
	go mod tidy
	go build -o email-mcp-server.exe main.go

run:
	go run main.go

test:
	go test -v ./...

clean:
	del email-mcp-server.exe

install:
	go mod download
	go build -o email-mcp-server.exe main.go
	@echo Built email-mcp-server.exe successfully
	@echo Configure Claude Desktop with the JSON config

# Test connection
test-imap:
	@echo Testing IMAP connection...
	@go run test_connection.go