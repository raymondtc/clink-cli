# Generate code from OpenAPI spec
generate:
	@echo "Generating code from OpenAPI spec..."
	@go run scripts/generate.go api/openapi.yaml
	@echo "Code generation complete!"

# Build all binaries
build:
	go build -o bin/clink ./cmd/clink
	go build -o bin/clink-mcp ./cmd/clink-mcp

# Run tests
test:
	go test -v ./...

# Clean generated files and binaries
clean:
	rm -rf bin/
	rm -f pkg/generated/*.go

# Full rebuild: generate + build
rebuild: clean generate build

.PHONY: all build test clean generate rebuild
