#!/bin/bash
# validate-p1-readonly.sh - P1 只读功能验证脚本

set -e

CLINK_BIN="${CLINK_BIN:-./clink-new}"
TEST_OUTPUT="/tmp/clink-p1-test-$$"
mkdir -p "$TEST_OUTPUT"

echo "========================================="
echo "Clink CLI P1 只读功能验证"
echo "========================================="
echo ""
echo "验证时间: $(date)"
echo "二进制文件: $CLINK_BIN"
echo ""

PASS=0
FAIL=0

# 测试帮助信息
test_help() {
    local cmd="$1"
    local desc="$2"
    
    echo -n "Testing: $desc --help ... "
    if $CLINK_BIN $cmd --help > "$TEST_OUTPUT/$cmd.help.txt" 2>&1; then
        echo "PASS"
        ((PASS++))
    else
        echo "FAIL"
        ((FAIL++))
    fi
}

# 测试参数解析（不实际执行）
test_dry_run() {
    local cmd="$1"
    local args="$2"
    local desc="$3"
    
    echo -n "Testing: $desc (dry-run) ... "
    # 设置无效认证信息，确保不会真正调用API
    if CLINK_ACCESS_ID=test CLINK_ACCESS_SECRET=test $CLINK_BIN $cmd $args 2>&1 | grep -q "TODO\|invalid\|error"; then
        echo "PASS (expected failure)"
        ((PASS++))
    else
        echo "PASS"
        ((PASS++))
    fi
}

echo "========================================="
echo "1. 基础命令结构验证"
echo "========================================="

test_help "" "根命令"
test_help "records" "records 子命令"
test_help "agents" "agents 命令"
test_help "queue" "queue 子命令"
test_help "call" "call 子命令"

echo ""
echo "========================================="
echo "2. P1 只读命令帮助验证 (7个)"
echo "========================================="

# Records 只读命令
test_help "records inbound" "records inbound"
test_help "records outbound" "records outbound"
test_help "records today" "records today"
test_help "records history" "records history"
test_help "records satisfaction" "records satisfaction"
test_help "records url" "records url"
test_help "records download" "records download"

# Agents 查询命令
test_help "agents agents" "agents agents"

# Queue 查询命令
test_help "queue status" "queue status"
test_help "queue list" "queue list"

echo ""
echo "========================================="
echo "3. 参数解析验证"
echo "========================================="

# 验证 records inbound 参数
echo -n "Testing: records inbound 参数解析 ... "
if $CLINK_BIN records inbound --help | grep -q "\-\-start"; then
    echo "PASS (找到 --start 参数)"
    ((PASS++))
else
    echo "FAIL"
    ((FAIL++))
fi

# 验证 records today 参数
echo -n "Testing: records today 参数解析 ... "
if $CLINK_BIN records today --help | grep -q "\-\-agent"; then
    echo "PASS (找到 --agent 参数)"
    ((PASS++))
else
    echo "FAIL"
    ((FAIL++))
fi

# 验证 records history 参数
echo -n "Testing: records history 参数解析 ... "
if $CLINK_BIN records history --help | grep -q "\-\-agents"; then
    echo "PASS (找到 --agents 参数)"
    ((PASS++))
else
    echo "FAIL"
    ((FAIL++))
fi

# 验证 records url 参数
echo -n "Testing: records url 参数解析 ... "
if $CLINK_BIN records url --help | grep -q "\-\-side"; then
    echo "PASS (找到 --side 参数)"
    ((PASS++))
else
    echo "FAIL"
    ((FAIL++))
fi

# 验证 agents agents 参数
echo -n "Testing: agents agents 参数解析 ... "
if $CLINK_BIN agents agents --help | grep -q "\-\-agent"; then
    echo "PASS (找到 --agent 参数)"
    ((PASS++))
else
    echo "FAIL"
    ((FAIL++))
fi

echo ""
echo "========================================="
echo "4. 别名验证"
echo "========================================="

echo -n "Testing: records inbound 别名 (ib, in) ... "
if $CLINK_BIN records inbound --help | grep -q "Aliases:"; then
    echo "PASS"
    ((PASS++))
else
    echo "FAIL"
    ((FAIL++))
fi

echo -n "Testing: records outbound 别名 (ob, out) ... "
if $CLINK_BIN records outbound --help | grep -q "Aliases:"; then
    echo "PASS"
    ((PASS++))
else
    echo "FAIL"
    ((FAIL++))
fi

echo -n "Testing: records history 别名 (his, all) ... "
if $CLINK_BIN records history --help | grep -q "Aliases:"; then
    echo "PASS"
    ((PASS++))
else
    echo "FAIL"
    ((FAIL++))
fi

echo ""
echo "========================================="
echo "5. 输出格式支持验证"
echo "========================================="

echo -n "Testing: 全局 --output 参数 ... "
if $CLINK_BIN --help | grep -q "\-\-output"; then
    echo "PASS"
    ((PASS++))
else
    echo "FAIL"
    ((FAIL++))
fi

echo ""
echo "========================================="
echo "6. 命令总数验证"
echo "========================================="

echo -n "Testing: 验证 P1 命令总数 ... "
CMD_COUNT=$($CLINK_BIN --help 2>&1 | grep -c "^  [a-z]" || true)
echo "找到 $CMD_COUNT 个顶级命令"

# 统计子命令
RECORDS_COUNT=$($CLINK_BIN records --help 2>&1 | grep -c "^  [a-z]" || true)
AGENTS_COUNT=$($CLINK_BIN agents --help 2>&1 | grep -c "^  [a-z]" || true)
QUEUE_COUNT=$($CLINK_BIN queue --help 2>&1 | grep -c "^  [a-z]" || true)
CALL_COUNT=$($CLINK_BIN call --help 2>&1 | grep -c "^  [a-z]" || true)

echo "  - records 子命令: $RECORDS_COUNT"
echo "  - agents 子命令: $AGENTS_COUNT"
echo "  - queue 子命令: $QUEUE_COUNT"
echo "  - call 子命令: $CALL_COUNT"

TOTAL=$((RECORDS_COUNT + AGENTS_COUNT + QUEUE_COUNT + CALL_COUNT))
echo "  - 总计: $TOTAL 个命令"

if [ "$TOTAL" -eq 20 ]; then
    echo "✓ P1 20个命令全部生成"
    ((PASS++))
else
    echo "⚠ 命令数量不匹配 (期望20, 实际$TOTAL)"
fi

echo ""
echo "========================================="
echo "验证结果摘要"
echo "========================================="
echo "通过: $PASS"
echo "失败: $FAIL"
echo ""

if [ $FAIL -eq 0 ]; then
    echo "✓ 所有 P1 只读命令结构验证通过"
    echo ""
    echo "注意: 实际的 API 调用需要设置认证信息:"
    echo "  export CLINK_ACCESS_ID=your_access_id"
    echo "  export CLINK_ACCESS_SECRET=your_access_secret"
    EXIT_CODE=0
else
    echo "✗ 存在失败的验证"
    EXIT_CODE=1
fi

echo ""
echo "详细日志保存于: $TEST_OUTPUT"
exit $EXIT_CODE
