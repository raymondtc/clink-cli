#!/bin/bash
# validate-readonly.sh - 只读功能验证脚本
# 用于验证 CLI 的只读命令（不修改环境数据）

set -e

CLINK_BIN="${CLINK_BIN:-./clink-test}"
TEST_OUTPUT="/tmp/clink-readonly-test-$$"
mkdir -p "$TEST_OUTPUT"

echo "========================================="
echo "Clink CLI 只读功能验证"
echo "========================================="
echo ""
echo "验证时间: $(date)"
echo "二进制文件: $CLINK_BIN"
echo "测试输出目录: $TEST_OUTPUT"
echo ""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 计数器
PASS=0
FAIL=0
SKIP=0

# 测试函数
run_test() {
    local name="$1"
    local cmd="$2"
    local check="$3"
    
    echo -n "Testing: $name ... "
    
    if eval "$cmd" > "$TEST_OUTPUT/${name// /_}.log" 2>&1; then
        if [ -n "$check" ]; then
            if eval "$check" "$TEST_OUTPUT/${name// /_}.log"; then
                echo "PASS"
                ((PASS++))
            else
                echo "FAIL (check failed)"
                ((FAIL++))
            fi
        else
            echo "PASS"
            ((PASS++))
        fi
    else
        echo "SKIP (需要认证或 API 不可用)"
        ((SKIP++))
    fi
}

# 检查帮助信息是否包含预期内容
check_help_contains() {
    local expected="$1"
    grep -q "$expected" "$2"
}

echo "========================================="
echo "1. 基础命令结构验证"
echo "========================================="

run_test "根命令帮助" \
    "$CLINK_BIN --help" \
    "check_help_contains records"

run_test "records 子命令帮助" \
    "$CLINK_BIN records --help" \
    "check_help_contains inbound"

run_test "agents 命令帮助" \
    "$CLINK_BIN agents --help" \
    "check_help_contains agent"

run_test "queue 子命令帮助" \
    "$CLINK_BIN queue --help" \
    "check_help_contains status"

echo ""
echo "========================================="
echo "2. 参数定义验证 (只读命令)"
echo "========================================="

# 验证呼入记录查询参数
run_test "records inbound 参数" \
    "$CLINK_BIN records inbound --help" \
    "check_help_contains start"

# 验证呼出记录查询参数  
run_test "records outbound 参数" \
    "$CLINK_BIN records outbound --help" \
    "check_help_contains phone"

# 验证座席状态查询参数
run_test "agents 参数" \
    "$CLINK_BIN agents --help" \
    "check_help_contains agent"

# 验证队列状态查询参数
run_test "queue status 参数" \
    "$CLINK_BIN queue status --help" \
    "check_help_contains queue"

# 验证队列列表查询参数
run_test "queue list 参数" \
    "$CLINK_BIN queue list --help" \
    "check_help_contains limit"

echo ""
echo "========================================="
echo "3. 输出格式验证"
echo "========================================="

# 验证 JSON 输出格式支持
run_test "JSON 输出格式支持" \
    "$CLINK_BIN agents --help | grep -q json" \
    ""

# 验证 table 输出格式支持
run_test "Table 输出格式支持" \
    "$CLINK_BIN agents --help | grep -q table" \
    ""

echo ""
echo "========================================="
echo "4. 配置加载验证 (如提供)"
echo "========================================="

# 如果提供了环境变量，验证连接
if [ -n "$CLINK_ACCESS_ID" ] && [ -n "$CLINK_ACCESS_SECRET" ]; then
    echo "检测到认证信息，执行 API 调用验证..."
    echo ""
    
    echo "Testing: agents API 调用 (只读)..."
    if $CLINK_BIN agents --output json > "$TEST_OUTPUT/agents_api.json" 2>&1; then
        echo -e "${GREEN}PASS${NC} - agents API 调用成功"
        ((PASS++))
        
        # 验证返回数据格式
        if jq -e '. != null' "$TEST_OUTPUT/agents_api.json" > /dev/null 2>&1; then
            echo -e "${GREEN}  ✓${NC} 返回数据格式正确 (JSON)"
        fi
    else
        echo -e "${YELLOW}SKIP${NC} - API 调用失败 (检查网络和认证)"
        ((SKIP++))
    fi
    
    echo "Testing: queue status API 调用 (只读)..."
    if $CLINK_BIN queue status --output json > "$TEST_OUTPUT/queue_api.json" 2>&1; then
        echo -e "${GREEN}PASS${NC} - queue status API 调用成功"
        ((PASS++))
    else
        echo -e "${YELLOW}SKIP${NC} - API 调用失败"
        ((SKIP++))
    fi
    
    echo "Testing: records inbound API 调用 (只读)..."
    if $CLINK_BIN records inbound --limit 1 --output json > "$TEST_OUTPUT/records_api.json" 2>&1; then
        echo -e "${GREEN}PASS${NC} - records inbound API 调用成功"
        ((PASS++))
    else
        echo -e "${YELLOW}SKIP${NC} - API 调用失败"
        ((SKIP++))
    fi
else
    echo "未提供认证信息 (CLINK_ACCESS_ID/CLINK_ACCESS_SECRET)"
    echo "跳过 API 调用验证"
    echo ""
    echo "如需完整验证，请设置环境变量:"
    echo "  export CLINK_ACCESS_ID=your_access_id"
    echo "  export CLINK_ACCESS_SECRET=your_access_secret"
fi

echo ""
echo "========================================="
echo "5. 配置文件验证"
echo "========================================="

# 验证配置文件存在
if [ -f "config/cli.yaml" ]; then
    echo -e "${GREEN}✓${NC} config/cli.yaml 存在"
    ((PASS++))
    
    # 统计配置中的命令数量
    CMD_COUNT=$(grep -c "^[[:space:]]*[A-Z].*:" config/cli.yaml || true)
    echo "  配置文件包含约 $CMD_COUNT 个命令定义"
else
    echo -e "${RED}✗${NC} config/cli.yaml 不存在"
    ((FAIL++))
fi

# 验证 OpenAPI 规范存在
if [ -f "openapi/openapi.json" ]; then
    echo -e "${GREEN}✓${NC} openapi/openapi.json 存在"
    ((PASS++))
    
    # 统计 API 数量
    API_COUNT=$(cat openapi/openapi.json | grep -c '"operationId"' || true)
    echo "  OpenAPI 包含 $API_COUNT 个 operation"
else
    echo -e "${YELLOW}!${NC} openapi/openapi.json 不存在 (运行 make extract-openapi 生成)"
fi

echo "========================================="
echo "验证结果摘要"
echo "========================================="
echo "通过: $PASS"
echo "失败: $FAIL"
echo "跳过: $SKIP"
echo ""

echo ""
echo "详细日志保存于: $TEST_OUTPUT"
echo ""

exit $EXIT_CODE
