.PHONY: help sync-sdk extract-openapi generate openapi clean validate-config

# Default target
help:
	@echo "Clink CLI - Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  sync-sdk         - Update SDK submodule to latest"
	@echo "  extract-openapi  - Extract OpenAPI spec from SDK"
	@echo "  validate-config  - Validate CLI config against OpenAPI"
	@echo "  generate         - Generate Go code from OpenAPI"
	@echo "  openapi          - Full pipeline: sync + extract"
	@echo "  clean            - Remove generated files"

# Update SDK submodule
sync-sdk:
	@echo "=== Syncing SDK submodule ==="
	git submodule update --init --recursive
	git submodule update --remote
	@echo "✓ SDK updated"

# Extract OpenAPI from SDK
extract-openapi:
	@echo "=== Extracting OpenAPI from SDK ==="
	@mkdir -p openapi
	go run scripts/extract-openapi.go \
		-sdk=./sdk/clink-sdk/clink-serversdk/src/main/java/com/tinet/clink \
		-out=./openapi/openapi.json

# Validate CLI config against OpenAPI
validate-config:
	@echo "=== Validating CLI config ==="
	go run scripts/validate-config/main.go \
		-openapi=./openapi/openapi.json \
		-config=./config/cli.yaml

# Generate code from OpenAPI
generate:
	@echo "=== Generating API code ==="
	go run scripts/api-generator-v2/main.go config/generator.v2.yaml api/openapi.yaml pkg/api/auto_generated.go
	@echo "=== Generating CLI code ==="
	go run scripts/clink-generator/main.go config/generator.yaml api/openapi.yaml cmd/clink

# Full pipeline: sync SDK and extract OpenAPI
openapi: sync-sdk extract-openapi
	@echo "=== OpenAPI generation complete ==="
	@echo "Output: ./openapi/openapi.json"

# Clean generated files
clean:
	@echo "=== Cleaning generated files ==="
	rm -f openapi/openapi.json
	@echo "✓ Cleaned"

# Development helpers
dev-test:
	@echo "=== Running tests ==="
	go test ./...

dev-build:
	@echo "=== Building binaries ==="
	go build -o bin/clink ./cmd/clink
	go build -o bin/clink-mcp ./cmd/clink-mcp
