#!/bin/bash
#
# clink-generate - 统一的代码生成脚本
#
# Usage:
#   ./scripts/clink-generate.sh all          # 生成所有代码
#   ./scripts/clink-generate.sh types        # 只生成类型
#   ./scripts/clink-generate.sh cli          # 只生成 CLI
#   ./scripts/clink-generate.sh api          # 只生成 API
#   ./scripts/clink-generate.sh check        # 检查配置一致性
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

CONFIG_FILE="$PROJECT_ROOT/config/generator.yaml"
OPENAPI_FILE="$PROJECT_ROOT/api/openapi.yaml"

CMD_OUTPUT="$PROJECT_ROOT/cmd/clink"
API_OUTPUT="$PROJECT_ROOT/pkg/api"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[!]${NC} $1"
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
}

log_section() {
    echo ""
    echo -e "${CYAN}══════════════════════════════════════════════════════════════${NC}"
    echo -e "${CYAN}  $1${NC}"
    echo -e "${CYAN}══════════════════════════════════════════════════════════════${NC}"
}

# 检查依赖
check_deps() {
    log_info "Checking dependencies..."
    
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        exit 1
    fi
    
    if [[ ! -f "$CONFIG_FILE" ]]; then
        log_error "Config file not found: $CONFIG_FILE"
        exit 1
    fi
    
    if [[ ! -f "$OPENAPI_FILE" ]]; then
        log_error "OpenAPI spec not found: $OPENAPI_FILE"
        exit 1
    fi
    
    log_success "Dependencies OK"
}

# 检查 oapi-codegen
check_oapi_codegen() {
    OAPI_CODEGEN=$(go env GOPATH)/bin/oapi-codegen
    if [[ ! -f "$OAPI_CODEGEN" ]]; then
        log_info "Installing oapi-codegen..."
        go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
    fi
    echo "$OAPI_CODEGEN"
}

# 生成类型
generate_types() {
    log_section "Generating Types"
    
    OAPI_CODEGEN=$(check_oapi_codegen)
    
    mkdir -p "$PROJECT_ROOT/pkg/generated"
    
    log_info "Running oapi-codegen..."
    "$OAPI_CODEGEN" -generate types,client -package generated \
        "$OPENAPI_FILE" > "$PROJECT_ROOT/pkg/generated/clink.gen.go"
    
    log_success "Generated pkg/generated/clink.gen.go"
}

# 生成 CLI
generate_cli() {
    log_section "Generating CLI Commands"
    
    log_info "Running CLI generator..."
    go run "$PROJECT_ROOT/scripts/clink-generator/main.go" \
        "$CONFIG_FILE" \
        "$OPENAPI_FILE" \
        "$CMD_OUTPUT"
    
    log_success "Generated CLI commands"
    
    # 格式化代码
    if command -v gofmt &> /dev/null; then
        log_info "Formatting generated code..."
        gofmt -w "$CMD_OUTPUT"/*_gen.go 2>/dev/null || true
    fi
}

# 生成 API
generate_api() {
    log_section "Generating API Methods"
    
    log_info "Running API generator..."
    go run "$PROJECT_ROOT/scripts/api-generator/main.go" \
        "$CONFIG_FILE" \
        "$OPENAPI_FILE" \
        "$API_OUTPUT/auto_generated.go"
    
    log_success "Generated API methods"
}

# 检查配置一致性
check_config() {
    log_section "Configuration Check"
    
    log_info "Checking config vs OpenAPI spec..."
    
    # 提取配置中的 operation IDs
    CONFIG_OPS=$(grep "^  [a-zA-Z]" "$CONFIG_FILE" | grep -v "^  command:\|^  description:\|^  flags:\|^  - \|^  use:\|^  args:" | sed 's/:.*//' | sed 's/^  //')
    
    # 提取 OpenAPI 中的 operation IDs
    SPEC_OPS=$(grep "operationId:" "$OPENAPI_FILE" | sed 's/.*operationId: //')
    
    echo ""
    echo "Configured endpoints:"
    echo "$CONFIG_OPS" | while read -r op; do
        if echo "$SPEC_OPS" | grep -q "^${op}$"; then
            echo -e "  ${GREEN}✓${NC} $op"
        else
            echo -e "  ${RED}✗${NC} $op ${YELLOW}(not in OpenAPI spec)${NC}"
        fi
    done
    
    echo ""
    echo "Available in OpenAPI but not configured:"
    echo "$SPEC_OPS" | while read -r op; do
        if ! echo "$CONFIG_OPS" | grep -q "^${op}$"; then
            echo -e "  ${YELLOW}•${NC} $op"
        fi
    done
}

# 显示统计信息
show_stats() {
    log_section "Generation Statistics"
    
    # 统计配置中的接口数
    ENDPOINT_COUNT=$(grep -c "^  [a-zA-Z].*:" "$CONFIG_FILE" 2>/dev/null || echo "0")
    
    # 统计生成的文件数
    CLI_FILES=$(ls "$CMD_OUTPUT"/*_gen.go 2>/dev/null | wc -l)
    API_FILES=$(ls "$API_OUTPUT"/auto_generated.go 2>/dev/null | wc -l)
    
    echo "  Configured endpoints: $ENDPOINT_COUNT"
    echo "  Generated CLI files:  $CLI_FILES"
    echo "  Generated API files:  $API_FILES"
    
    # 列出可用的命令
    echo ""
    echo "Available commands:"
    grep "command:" "$CONFIG_FILE" | sed 's/.*command: \[//' | sed 's/\].*//' | sed 's/", "/ /g' | sed 's/"//g' | sort -u | sed 's/^/  - /'
}

# 主函数
main() {
    cd "$PROJECT_ROOT"
    
    case "${1:-all}" in
        all)
            log_section "Clink CLI Code Generator"
            check_deps
            generate_types
            generate_cli
            generate_api
            show_stats
            echo ""
            log_success "All code generated successfully!"
            echo ""
            echo "Next steps:"
            echo "  make build    # Build the CLI"
            echo "  make test     # Run tests"
            ;;
        types)
            check_deps
            generate_types
            ;;
        cli)
            check_deps
            generate_cli
            ;;
        api)
            check_deps
            generate_api
            ;;
        check)
            check_config
            ;;
        stats)
            show_stats
            ;;
        *)
            echo "Usage: $0 {all|types|cli|api|check|stats}"
            echo ""
            echo "Commands:"
            echo "  all    - Generate all code (default)"
            echo "  types  - Generate types from OpenAPI"
            echo "  cli    - Generate CLI commands"
            echo "  api    - Generate API methods"
            echo "  check  - Check config consistency"
            echo "  stats  - Show generation statistics"
            exit 1
            ;;
    esac
}

main "$@"
