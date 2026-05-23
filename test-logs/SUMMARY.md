# 测试总结日志

> 每次测试运行后追加记录。最新在最上面。

---

## 2026-05-23 15:29 — 10.12 + 10.14 集成测试（真实LLM + SQLite）

**运行命令:** `go test ./tests/... -tags=integration -v -run "Audit|Health" -count=1 -timeout=60s`
**日志文件:** `test-logs/integration/2026-05-23_152905.jsonl`

| 指标 | 值 |
|------|-----|
| 新增测试 | 2 |
| 通过 | 2 |
| 失败 | 0 |
| 总耗时 | 15s |

| 测试 | 耗时 | 验证点 |
|------|------|--------|
| TestIntegration_AuditWritesAfterChat | 12.5s | 真实LLM对话后，audit 表有 SENSE+OUTPUT 两条记录 |
| TestIntegration_HealthDegradedWhenLLMDown | 2.0s | 连接不可达LLM → 状态返回 degraded |

**Task 打勾:** 10.12 ✅, 10.14 ✅, Task 10 整体 ✅

---

## 2026-05-23 15:15 — 全量单元测试（无跳过，含真实LLM）

**运行命令:** `go test ./internal/... ./cmd/... -v -count=1 -timeout=120s`
**日志文件:** `test-logs/unit/2026-05-23_151500_full.log`

| 指标 | 值 |
|------|-----|
| 总测试数 | 73 |
| 通过 | 73 |
| 失败 | 0 |
| 跳过 | 0 |
| 总耗时 | ~13s |

### 真实 LLM 调用测试

| 测试 | 耗时 | LLM 行为 |
|------|------|---------|
| TestLLMChatWithToolCalls | 2.94s | 调用 DeepSeek，返回 probe_disk tool_call |
| TestLLMInvalidAPIKey | 0.29s | 真实 API 返回 401 → LLM_AUTH_001 |
| TestLLMNetworkTimeout | 3.00s | 连接 192.0.2.1 超时 → LLM_NETWORK_001 |

### 各包结果

| 包 | 测试数 | 耗时 | 状态 |
|----|--------|------|------|
| internal/agent | 12 | 6.66s | ✅ |
| internal/api | 16 | 1.70s | ✅ |
| internal/safety | 28 | 0.71s | ✅ |
| internal/store | 5 | 1.33s | ✅ |
| internal/tools | 14 | 1.17s | ✅ |
| cmd/server | 3 | 1.78s | ✅ |

---

## 2026-05-23 14:57 — 集成测试（真实LLM端到端）

**运行命令:** `go test ./tests/... -tags=integration -v -count=1 -timeout=180s`
**日志文件:** `test-logs/integration/2026-05-23_145743.jsonl`

| 指标 | 值 |
|------|-----|
| 总测试数 | 5 |
| 通过 | 5 |
| 失败 | 0 |
| 跳过 | 0 |
| 总耗时 | 128s |
| LLM 模型 | mimo-v2.5-pro |

### 各场景结果

| 场景 | 耗时 | 关键验证点 |
|------|------|-----------|
| BasicChat "看磁盘" | 12s | LLM调用probe_disk → 返回含%数字的磁盘表格 |
| MultiTurnContext | 28s | 第2句"那内存呢"成功引用第1轮上下文 |
| InjectionBlocked | 0ms | "忽略之前所有指令" → sense(blocked) → error → done |
| SafetyGuard | 0ms | rm -rf / 等4个危险命令拦截，df -h 等4个安全命令放行 |
| MultiAgentMode | 88s | 强制multi模式，40个SSE事件，agent_role(planner/executor/verifier)全可见 |

---

## 2026-05-23 14:30 — 首次全量单元测试基线

**日志文件:** `test-logs/unit/2026-05-23_143000.log`

| 指标 | 值 |
|------|-----|
| 总测试数 | 66 |
| 通过 | 66 |
| 失败 | 0 |

*注: 此时还没有 auth_test.go / multi_test.go / desktop_test.go*

---

## 变更记录

| 日期 | 变更 | 测试数变化 |
|------|------|-----------|
| 2026-05-23 14:30 | 初始测试套件 | 42 → 66 (+24) |
| 2026-05-23 14:40 | 添加 loop_test + preview_test + session_test + fs_test | 66 → 73 (+7) |
| 2026-05-23 15:08 | 添加 auth_test + multi_test + desktop_test + write_tools_test | 66 → 73 (+7 实际新增) |
| 2026-05-23 15:15 | 去掉所有 short skip，全量含真实LLM | 73 (零跳过) |
| 2026-05-23 14:57 | 创建集成测试 (tests/integration_test.go) | +5 集成 |

---

## 当前总计

| 类型 | 数量 | 跳过 | 状态 |
|------|------|------|------|
| 单元测试 | 73 | 0 | ✅ 全绿 |
| 集成测试 | 7 | 0 | ✅ 全绿 |
| **合计** | **80** | **0** | ✅ |
