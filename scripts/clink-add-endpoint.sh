#!/bin/bash
#
# clink-add-endpoint - 一键添加新接口到 clink-cli
# 
# Usage: ./scripts/clink-add-endpoint.sh <operation-id> [options]
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
CONFIG_FILE="$PROJECT_ROOT/config/generator.yaml"
OPENAPI_FILE="$PROJECT_ROOT/api/openapi.yaml"
OUTPUT_DIR="$PROJECT_ROOT/cmd/clink"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 显示帮助
show_help() {
    cat << EOF
Clink CLI 接口添加工具

Usage: $0 <operation-id> [options]

Arguments:
  operation-id    OpenAPI 中的 operationId (如: listCdrIbs, webcall)

Options:
  -c, --command   命令路径 (如: "records inbound", "agent status")
  -d, --desc      命令描述
  -f, --flags     自定义 flag 映射 (格式: "param:flag:shorthand:description")
  -h, --help      显示帮助

Examples:
  # 添加 listCdrIbs 接口，使用默认配置
  $0 listCdrIbs

  # 添加新接口，指定命令路径
  $0 newEndpoint -c "mycommand sub" -d "My description"

  # 添加接口，自定义 flag 映射
  $0 webcall -f "customerNumber:phone:p:客户号码" -f "clid:display::外显号码"

EOF
}

# 解析参数
OPERATION_ID=""
COMMAND=""
DESCRIPTION=""
FLAGS=()

while [[ $# -gt 0 ]]; do
    case $1 in
        -c|--command)
            COMMAND="$2"
            shift 2
            ;;
        -d|--desc)
            DESCRIPTION="$2"
            shift 2
            ;;
        -f|--flags)
            FLAGS+=("$2")
            shift 2
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        -*)
            log_error "Unknown option: $1"
            show_help
            exit 1
            ;;
        *)
            if [[ -z "$OPERATION_ID" ]]; then
                OPERATION_ID="$1"
            else
                log_error "Unknown argument: $1"
                exit 1
            fi
            shift
            ;;
    esac
done

if [[ -z "$OPERATION_ID" ]]; then
    log_error "Operation ID is required"
    show_help
    exit 1
fi

log_info "Adding endpoint: $OPERATION_ID"

# 检查 OpenAPI 中是否存在该 operation
if ! grep -q "operationId: $OPERATION_ID" "$OPENAPI_FILE"; then
    log_error "Operation '$OPERATION_ID' not found in $OPENAPI_FILE"
    log_info "Available operations:"
    grep "operationId:" "$OPENAPI_FILE" | sed 's/.*operationId: /  - /'
    exit 1
fi

log_info "Found operation in OpenAPI spec"

# 检查是否已在配置中
if grep -q "^  $OPERATION_ID:" "$CONFIG_FILE" 2>/dev/null; then
    log_warning "Operation '$OPERATION_ID' already exists in config"
    read -p "Do you want to update it? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Aborted"
        exit 0
    fi
    # 删除旧配置
    sed -i '' "/^  $OPERATION_ID:/,/^  [a-z]/d" "$CONFIG_FILE" 2>/dev/null || true
fi

# 提取 OpenAPI 中的信息
log_info "Extracting API information..."

# 获取路径和 HTTP 方法
ENDPOINT_INFO=$(grep -B 20 "operationId: $OPERATION_ID" "$OPENAPI_FILE" | grep -E "^(  /|    (get|post):)" | tail -2)
API_PATH=$(echo "$ENDPOINT_INFO" | grep "^  /" | sed 's/:.*//' | tr -d ' ')
HTTP_METHOD=$(echo "$ENDPOINT_INFO" | grep -E "(get|post):" | sed 's/.*: //' | tr -d ' ')

log_info "API Path: $API_PATH"
log_info "HTTP Method: $HTTP_METHOD"

# 如果没有提供命令，自动生成
if [[ -z "$COMMAND" ]]; then
    # 根据 operationId 推断
    case "$OPERATION_ID" in
        list*)
            COMMAND="list"
            ;;
        get*|describe*)
            COMMAND="get"
            ;;
        create*)
            COMMAND="create"
            ;;
        update*)
            COMMAND="update"
            ;;
        delete*)
            COMMAND="delete"
            ;;
        *)
            COMMAND="$OPERATION_ID"
            ;;
    esac
    log_info "Auto-generated command: $COMMAND"
fi

# 如果没有提供描述，自动生成
if [[ -z "$DESCRIPTION" ]]; then
    # 从 OpenAPI 中提取 summary
    DESCRIPTION=$(grep -A 5 "operationId: $OPERATION_ID" "$OPENAPI_FILE" | grep "summary:" | sed 's/.*summary: //')
    if [[ -z "$DESCRIPTION" ]]; then
        DESCRIPTION="$OPERATION_ID operation"
    fi
    log_info "Auto-generated description: $DESCRIPTION"
fi

# 生成配置片段
log_info "Generating config entry..."

CONFIG_ENTRY="
  $OPERATION_ID:
    command: [$(echo "$COMMAND" | sed 's/ /", "/g' | sed 's/^/"/' | sed 's/$/"/' | sed 's/, /", "/g')]
    description: \"$DESCRIPTION\"
"

# 提取参数并生成 flags
log_info "Extracting parameters..."

PARAMS_SECTION=$(grep -A 100 "operationId: $OPERATION_ID" "$OPENAPI_FILE" | sed -n '/parameters:/,/responses:/p' | head -20)

if echo "$PARAMS_SECTION" | grep -q "parameters:"; then
    CONFIG_ENTRY="${CONFIG_ENTRY}    flags:\n"
    
    # 提取参数名
    PARAM_NAMES=$(echo "$PARAMS_SECTION" | grep "^- name:" | sed 's/.*- name: //')
    
    while IFS= read -r param; do
        [[ -z "$param" ]] && continue
        
        # 检查是否为自定义 flag
        FLAG_DEF=""
        for f in "${FLAGS[@]}"; do
            if [[ "$f" == "$param":* ]]; then
                FLAG_DEF="$f"
                break
            fi
        done
        
        if [[ -n "$FLAG_DEF" ]]; then
            # 解析自定义 flag 格式: param:flag:shorthand:description
            IFS=':' read -r p flag shorthand desc <<< "$FLAG_DEF"
            CONFIG_ENTRY="${CONFIG_ENTRY}      - param: $param\n"
            CONFIG_ENTRY="${CONFIG_ENTRY}        flag: $flag\n"
            [[ -n "$shorthand" ]] && CONFIG_ENTRY="${CONFIG_ENTRY}        shorthand: $shorthand\n"
            [[ -n "$desc" ]] && CONFIG_ENTRY="${CONFIG_ENTRY}        description: \"$desc\"\n"
        else
            # 自动生成 flag
            FLAG_NAME=$(echo "$param" | sed 's/\([A-Z]\)/-\1/g' | tr '[:upper:]' '[:lower:]')
            CONFIG_ENTRY="${CONFIG_ENTRY}      - param: $param\n"
            CONFIG_ENTRY="${CONFIG_ENTRY}        flag: $FLAG_NAME\n"
            CONFIG_ENTRY="${CONFIG_ENTRY}        description: \"$param parameter\"\n"
        fi
    done <<< "$PARAM_NAMES"
fi

# 添加配置到文件
log_info "Updating $CONFIG_FILE..."
echo -e "$CONFIG_ENTRY" >> "$CONFIG_FILE"

log_success "Configuration updated"

# 重新生成代码
log_info "Regenerating code..."

cd "$PROJECT_ROOT"

# 检查生成器
if [[ ! -f "scripts/clink-generator/main.go" ]]; then
    log_error "Generator not found: scripts/clink-generator/main.go"
    exit 1
fi

# 运行生成器
go run scripts/clink-generator/main.go \
    "$CONFIG_FILE" \
    "$OPENAPI_FILE" \
    "$OUTPUT_DIR"

log_success "Code regenerated"

# 可选：运行代码格式化
if command -v gofmt &> /dev/null; then
    log_info "Formatting generated code..."
    gofmt -w "$OUTPUT_DIR"/*_gen.go
fi

echo ""
log_success "Endpoint '$OPERATION_ID' added successfully!"
echo ""
echo "Next steps:"
echo "  1. Review generated code in $OUTPUT_DIR"
echo "  2. Build and test: make build"
echo "  3. Run: ./bin/clink $COMMAND --help"
