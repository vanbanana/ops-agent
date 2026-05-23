# 前后端对接规范

> 本文件是前端开发的详细参考文档。
> **精简版持久契约**: `.kiro/steering/frontend-api-contract.md`（Kiro steering 自动加载）
> 两者有冲突时以 steering 为准（steering 更新更频繁）。
> 禁止使用假数据上线。

## 1. 铁律

1. **零假数据原则**：前端所有展示的数字、状态、列表必须来自后端 API 真实响应。开发期间如果后端 API 未就绪，用 `<!-- MOCK -->` 注释包裹并显示灰色占位符 `[等待接口]`，**绝不填写看起来像真的假数字**。
2. **接口未就绪时前端行为**：显示 skeleton 加载态或明确的 `接口未就绪` 提示，不渲染任何具体数值。
3. **数据来源标记**：每个组件文件头部注释标明数据来源端点，例如 `// Data: GET /health`。

---

## 2. 后端 API 就绪状态

### ✅ 已就绪（真实数据，直接对接）

| 端点 | 方法 | 用途 | 响应格式 |
|------|------|------|----------|
| `/health` | GET | 健康检查 + 子组件状态 | `{status, components: {llm, ...}}` |
| `/api/v1/tools` | GET | 工具列表（12个探针） | `{code:0, data: {tools:[], count:12}}` |
| `/api/v1/chat/stream` | POST | SSE 流式对话 | `text/event-stream`，见下方事件协议 |
| `/api/v1/chat` | POST | 同步对话 | `{code:0, data: {reply, session_id}}` |
| `/api/v1/safety/scan?cmd=xxx` | GET | 命令安全扫描 | `{code:0, data: {status, reason, detail}}` |

### ⏳ 未就绪（后端还没做，前端先占位）

| 端点 | 方法 | 用途 | 预计完成 Task |
|------|------|------|--------------|
| `/api/v1/auth/login` | POST | JWT 登录 | Task 10（完善版） |
| `/api/v1/sessions` | GET | 会话列表 | Task 10 |
| `/api/v1/sessions/{id}` | GET | 会话详情+消息 | Task 10 |
| `/api/v1/sessions/{id}` | DELETE | 删除会话 | Task 10 |
| `/api/v1/audit` | GET | 审计日志列表 | Task 10 |
| `/api/v1/audit/{trace_id}` | GET | 单次请求完整链路 | Task 10 |
| `/api/v1/safety/preview` | POST | 风险预演 | Task 12 |
| `/api/v1/safety/confirm` | POST | 确认执行 | Task 12 |
| `/api/v1/mcp/servers` | GET/POST/DELETE | MCP 配置 | Task 4 |
| `/api/v1/desktop/probe/{name}` | POST | 桌面模式直调探针 | Task 14 |
| `/api/v1/desktop/action/{name}` | POST | 桌面模式触发动作 | Task 14 |
| `/api/v1/configs` | GET/PUT | 动态配置 | Task 10 |
| `/version` | GET | 版本信息 | Task 10 |
| `/metrics` | GET | Prometheus 指标 | Task 17 |

---

## 3. SSE 事件协议（已就绪，直接对接）

`POST /api/v1/chat/stream` 返回 `text/event-stream`。

### 请求体

```json
{
  "session_id": "可选，不传则自动生成",
  "message": "用户输入文本"
}
```

### 事件类型及顺序

```
start → sense → mode_decision → [analyze → plan → execute → execute_done]* → output → done
```

| event | data 结构 | 前端行为 |
|-------|----------|---------|
| `start` | `{trace_id, session_id, mode}` | 开始渲染，显示 thinking 状态 |
| `sense` | `{status: "ok"\|"blocked", reason?}` | blocked → 显示红色拦截卡片 |
| `mode_decision` | `{mode: "single"\|"multi", reason}` | 更新模式指示器 |
| `analyze` | `{round, has_tool_calls, finish_reason, reply_preview}` | 推理链路面板更新 ANALYZE 阶段 |
| `plan` | `{round, tools: [{name, args}]}` | 推理链路面板更新 PLAN，显示将调用的工具 |
| `execute` | `{tool, args, security_check}` | 推理链路面板更新 EXECUTE，显示"正在执行..." |
| `execute_done` | `{tool, status, result_preview, elapsed_ms}` | 更新工具执行结果，耗时 |
| `output` | `{reply, tokens_used, elapsed_ms, mode}` | 渲染最终回复（Markdown） |
| `error` | `{error_code, message, recoverable}` | 显示错误卡片，按 error_code 展示 |
| `done` | `{trace_id, session_id, status}` | 结束 thinking 状态，断开 SSE |

### 注入被拦截时的事件流（短路）

```
start → sense(blocked) → error(SAFETY_INJECT_001) → done(error)
```

前端收到 `sense.status=blocked` 后直接显示红色拦截卡片，不等后续事件。

---

## 4. 资源带数据来源

顶部资源带（32px）的数据**不走 LLM**，直接调探针 API：

| 指标 | 数据源 | 刷新频率 |
|------|--------|---------|
| 磁盘使用率 | `POST /api/v1/desktop/probe/disk`（⏳未就绪）<br>临时方案：从 chat 回复中解析 | 30s |
| 系统负载 | `POST /api/v1/desktop/probe/top`（⏳未就绪） | 30s |
| 内存 | `POST /api/v1/desktop/probe/memory`（⏳未就绪） | 30s |
| 进程数 | `POST /api/v1/desktop/probe/process`（⏳未就绪） | 60s |
| 端口数 | `POST /api/v1/desktop/probe/network_connections`（⏳未就绪） | 60s |

**临时方案**（Task 14 完成前）：资源带显示 skeleton 占位或 `--` 表示无数据。在第一次对话返回探针数据后，从 SSE `execute_done` 事件的 `result_preview` 中提取数据填入（这是真实数据，只是来源是对话触发而非定时轮询）。

---

## 5. 右侧面板数据来源

### Tab 1: 推理链路（✅ 已就绪）

数据源：SSE 事件流实时推送。每个事件对应时间线一个节点。

```
SENSE    → 来自 event:sense
ANALYZE  → 来自 event:analyze
PLAN     → 来自 event:plan
EXECUTE  → 来自 event:execute + event:execute_done
OUTPUT   → 来自 event:output
```

### Tab 2: 实时数据（部分就绪）

| 图表 | 数据来源 | 状态 |
|------|---------|------|
| 磁盘柱状图 | `execute_done` 事件中 `tool=probe_disk` 的 `result_preview` | ✅ 有真实数据 |
| CPU/内存环形图 | `execute_done` 事件中 `tool=probe_memory/probe_top` | ✅ 有真实数据 |
| 负载条形图 | `execute_done` 事件中 `tool=probe_top` | ✅ 有真实数据 |
| Top CPU 进程 | `execute_done` 事件中 `tool=probe_process` | ✅ 有真实数据 |

**规则**：这些图表在 Agent 未调用过对应探针时显示空态（`暂无数据，发起对话后自动填充`），调用过之后用最近一次的真实结果渲染。**不允许预填假数据。**

### Tab 3: 健康状态（✅ 已就绪）

数据源：`GET /health`，每 30s 轮询。

---

## 6. 前端开发检查清单

每个组件提交前必须确认：

```
□ 数据来自真实 API 调用（不是硬编码数字）
□ API 未就绪时显示加载态或"接口未就绪"，不显示假数据
□ 文件头注释标明数据端点
□ SSE 事件解析按本文档 §3 协议
□ 错误码按 design.md §4.6 展示对应中文提示
□ 所有数值用等宽字体
□ 颜色阈值判断在前端做（磁盘>80%黄，>95%红）
```

---

## 7. CORS 配置

开发模式下后端已默认同源。如果前端 dev server 跑在不同端口：

```bash
# 启动后端时设置（后续会加到代码里）
CORS_ORIGINS=http://localhost:5173 go run ./cmd/server/
```

当前版本暂未实现 CORS 中间件，前端开发时用 Vite proxy 代理到 `http://localhost:8080`：

```js
// vite.config.js
export default {
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
      '/health': 'http://localhost:8080',
    }
  }
}
```

---

## 8. 命名约定

| 后端字段 | 格式 | 前端展示 |
|---------|------|---------|
| 时间 `*_at` | ISO8601 (`2026-05-23T14:32:01.234Z`) | 用 dayjs 格式化为 `14:32:01` 或 `5分钟前` |
| 时长 `*_ms` | 整数毫秒 | `<1s` 显示 ms，`>=1s` 显示 `1.2s` |
| ID `sess_xxx` / `trc_xxx` | 前缀+随机 | 前端不显示完整 ID，仅在 debug 面板展示 |
| 布尔 `is_*` / `has_*` | true/false | — |
| 枚举 | 英文小写 | 前端维护枚举到中文的映射表 |
