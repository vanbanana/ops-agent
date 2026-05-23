#!/bin/bash
# 一键运行全部测试并保存日志
# 用法: ./run_tests.sh [unit|integration|all]
set -e

TIMESTAMP=$(date +%Y-%m-%d_%H%M%S)
MODE=${1:-all}

echo "═══════════════════════════════════════"
echo " OPS-AGENT 测试运行器"
echo " 时间: $(date '+%Y-%m-%d %H:%M:%S')"
echo " 模式: $MODE"
echo "═══════════════════════════════════════"
echo ""

# Ensure log directories exist
mkdir -p test-logs/unit test-logs/integration test-logs/e2e

if [ "$MODE" = "unit" ] || [ "$MODE" = "all" ]; then
    UNIT_LOG="test-logs/unit/${TIMESTAMP}.log"
    echo "🧪 运行单元测试..."
    echo "   日志 → $UNIT_LOG"
    go test ./internal/... ./cmd/... -v -count=1 2>&1 | tee "$UNIT_LOG"
    UNIT_EXIT=${PIPESTATUS[0]}
    echo ""
    
    PASS_COUNT=$(grep -c "^--- PASS" "$UNIT_LOG" 2>/dev/null || echo "0")
    FAIL_COUNT=$(grep -c "^--- FAIL" "$UNIT_LOG" 2>/dev/null || echo "0")
    echo "   结果: ✅ $PASS_COUNT 通过, ❌ $FAIL_COUNT 失败"
    echo ""
fi

if [ "$MODE" = "integration" ] || [ "$MODE" = "all" ]; then
    INT_LOG="test-logs/integration/${TIMESTAMP}.log"
    echo "🔗 运行集成测试 (真实 LLM)..."
    echo "   日志 → $INT_LOG"
    
    if [ -z "$LLM_API_KEY" ] && [ -f .env ]; then
        export $(grep -v '^#' .env | xargs)
    fi
    
    if [ -z "$LLM_API_KEY" ]; then
        echo "   ⚠️  LLM_API_KEY 未设置，跳过集成测试"
    else
        go test ./tests/... -tags=integration -v -count=1 -timeout=180s 2>&1 | tee "$INT_LOG"
        INT_EXIT=${PIPESTATUS[0]}
        
        PASS_COUNT=$(grep -c "^--- PASS" "$INT_LOG" 2>/dev/null || echo "0")
        FAIL_COUNT=$(grep -c "^--- FAIL" "$INT_LOG" 2>/dev/null || echo "0")
        echo ""
        echo "   结果: ✅ $PASS_COUNT 通过, ❌ $FAIL_COUNT 失败"
    fi
    echo ""
fi

echo "═══════════════════════════════════════"
echo " 完成。日志已保存到 test-logs/"
echo "═══════════════════════════════════════"

# Exit with failure if any test failed
if [ "${UNIT_EXIT:-0}" -ne 0 ] || [ "${INT_EXIT:-0}" -ne 0 ]; then
    exit 1
fi
