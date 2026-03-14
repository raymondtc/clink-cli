.PHONY: all build test clean install

VERSION := 0.1.0
BINARY := clink
MCP_BINARY := clink-mcp

all: build

build:
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(BINARY) ./cmd/clink
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(MCP_BINARY) ./cmd/clink-mcp

test:
	go test -v -race -cover ./...

clean:
	rm -f $(BINARY) $(MCP_BINARY)

install: build
	cp $(BINARY) /usr/local/bin/
	cp $(MCP_BINARY) /usr/local/bin/

# Cross compilation
build-all:
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/$(BINARY)-linux-amd64 ./cmd/clink
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o dist/$(BINARY)-linux-arm64 ./cmd/clink
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o dist/$(BINARY)-darwin-amd64 ./cmd/clink
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o dist/$(BINARY)-darwin-arm64 ./cmd/clink
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/$(BINARY)-windows-amd64.exe ./cmd/clink
	
	GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/$(MCP_BINARY)-linux-amd64 ./cmd/clink-mcp
	GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o dist/$(MCP_BINARY)-linux-arm64 ./cmd/clink-mcp
	GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o dist/$(MCP_BINARY)-darwin-amd64 ./cmd/clink-mcp
	GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o dist/$(MCP_BINARY)-darwin-arm64 ./cmd/clink-mcp
	GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/$(MCP_BINARY)-windows-amd64.exe ./cmd/clink-mcp
