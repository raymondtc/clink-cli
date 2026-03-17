# Generate code from OpenAPI spec using oapi-codegen
OAPI_CODEGEN := $(shell go env GOPATH)/bin/oapi-codegen

# Generator scripts
CLI_GENERATOR := scripts/clink-generator/main.go
API_GENERATOR := scripts/api-generator/main.go

# Config files
CONFIG_FILE := config/generator.yaml
OPENAPI_FILE := api/openapi.yaml

# Output directories
CMD_OUTPUT := cmd/clink
API_OUTPUT := pkg/api

.PHONY: all generate generate-cli generate-api build test clean help

# Default: show help
help:
	@echo "Clink CLI - Available targets:"
	@echo ""
	@echo "  make generate        - Generate all code (CLI + API)"
	@echo "  make generate-cli    - Generate CLI commands from config"
	@echo "  make generate-api    - Generate API methods from config"
	@echo "  make generate-types  - Generate types from OpenAPI using oapi-codegen"
	@echo "  make build           - Build the CLI binary"
	@echo "  make test            - Run tests"
	@echo "  make clean           - Clean generated files"
	@echo "  make add-endpoint    - Interactive endpoint addition"
	@echo ""

# Generate all code
all: generate build

# Generate all code (types + CLI + API)
generate: generate-types generate-cli generate-api
	@echo "✓ All code generated"

# Generate types from OpenAPI using oapi-codegen
generate-types:
	@echo "Generating types from OpenAPI..."
	@test -f $(OAPI_CODEGEN) || go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	@mkdir -p pkg/generated
	@$(OAPI_CODEGEN) -generate types,client -package generated $(OPENAPI_FILE) > pkg/generated/clink.gen.go
	@echo "✓ Generated pkg/generated/clink.gen.go"

# Generate CLI commands from config
generate-cli:
	@echo "Generating CLI commands..."
	@go run $(CLI_GENERATOR) $(CONFIG_FILE) $(OPENAPI_FILE) $(CMD_OUTPUT)
	@echo "✓ CLI commands generated"

# Generate API methods from config  
generate-api:
	@echo "Generating API methods..."
	@go run $(API_GENERATOR) $(CONFIG_FILE) $(OPENAPI_FILE) $(API_OUTPUT)/auto_generated.go
	@echo "✓ API methods generated"

# Build CLI binary
build:
	@echo "Building clink CLI..."
	@go build -o bin/clink ./cmd/clink
	@echo "✓ Built bin/clink"

# Run tests
test:
	@go test -v ./...

# Clean generated files
clean:
	@echo "Cleaning generated files..."
	@rm -f pkg/generated/*.go
	@rm -f $(CMD_OUTPUT)/*_gen.go
	@rm -f $(API_OUTPUT)/auto_generated.go
	@rm -f bin/clink
	@echo "✓ Cleaned"

# Interactive endpoint addition
add-endpoint:
	@./scripts/clink-add-endpoint.sh

# Full rebuild: clean + generate + build
rebuild: clean generate build
