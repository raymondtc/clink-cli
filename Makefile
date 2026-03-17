# Generate code from OpenAPI spec
OAPI_CODEGEN := $(shell go env GOPATH)/bin/oapi-codegen

generate:
	@echo "Generating code from OpenAPI spec..."
	@test -f $(OAPI_CODEGEN) || go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	@mkdir -p pkg/generated
	@$(OAPI_CODEGEN) -generate types,client -package generated api/openapi.yaml > pkg/generated/clink.gen.go
	@echo "Code generation complete! pkg/generated/clink.gen.go"

# Build CLI binary
build:
	go build -o bin/clink ./cmd/clink

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
