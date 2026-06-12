# 前后端对接规范

> 本文件是前端开发的详细参考文档。
> 所有标记为"已就绪"的端点均已在后端实现并测试通过。

## 1. 铁律

1. **零假数据原则**：前端所有展示的数字、状态、列表必须来自后端 API 真实响应。
2. **接口未就绪时前端行为**：显示 skeleton 加载态或明确的 `接口未就绪` 提示，不渲染任何具体数值。
3. **数据来源标记**：每个组件文件头部注释标明数据来源端点，例如 `// Data: GET /health`。

---

## 2. 后端 API 就绪状态

### 已就绪（真实数据，直接对接）

| 端点 | 方法 | 用途 | 认证 |
|------|------|------|------|
| `/health` | GET | 健康检查 + 子组件状态 | 否 |
| `/health/deep` | GET | 深度检查（含 LLM 连通性） | 否 |
| `/version` | GET | 版本信息 | 否 |
| `/api/v1/auth/login` | POST | JWT 登录 | 否 |
| `/api/v1/auth/lockouts` | GET | 查看被锁定的 IP | 否 |
| `/api/v1/auth/lockout/{ip}` | DELETE | 解锁 IP（仅 localhost） | 否 |
| `/api/v1/chat/stream` | POST | SSE 流式对话 | JWT |
| `/api/v1/chat` | POST | 同步对话 | JWT |
| `/api/v1/tools` | GET | 工具列表（22个） | JWT |
| `/api/v1/tools/status` | GET | 工具启用状态 | JWT |
| `/api/v1/tools/toggle` | POST | 切换工具状态 | JWT |
| `/api/v1/safety/scan` | GET | 命令安全扫描 | JWT |
| `/api/v1/safety/preview` | POST | 风险预演 | JWT |
| `/api/v1/safety/confirm` | POST | 确认执行 | JWT |
| `/api/v1/sessions` | GET | 会话列表 | JWT |
| `/api/v1/sessions/{id}` | GET | 会话详情 + 消息 | JWT |
| `/api/v1/sessions/{id}` | DELETE | 删除会话 | JWT |
| `/api/v1/audit/logs` | GET | 审计日志列表 | JWT |
| `/api/v1/audit/{traceID}` | GET | 单次请求完整链路 | JWT |
| `/api/v1/desktop/probe/{name}` | POST | 桌面模式直调探针 | JWT |
| `/api/v1/desktop/action/{name}` | POST | 桌面模式触发动作 | JWT |
| `/api/v1/configs` | GET | 批量获取配置项 | JWT |
| `/api/v1/configs` | PUT | 批量更新配置项 | JWT |
| `/api/v1/permission/mode` | GET | 获取权限模式 | JWT |
| `/api/v1/permission/mode` | PUT | 设置权限模式 | JWT |
| `/api/v1/permission/respond` | POST | 响应权限请求 | JWT |
| `/api/v1/plan/approve` | POST | 批准计划 | JWT |
| `/api/v1/plan/reject` | POST | 拒绝计划 | JWT |
| `/api/v1/models/pool` | GET | 获取模型池 | JWT |
| `/api/v1/models/pool` | PUT | 保存模型池 | JWT |
| `/api/v1/models/switch` | POST | 切换活跃模型 | JWT |
| `/api/v1/models/test` | POST | 测试模型连通性 | JWT |
| `/api/v1/models/active` | GET | 获取当前活跃模型 | JWT |
| `/api/v1/mcp/servers` | GET | 获取 MCP 服务器列表 | JWT |
| `/api/v1/mcp/servers` | POST | 添加 MCP 服务器 | JWT |
| `/api/v1/mcp/servers/{id}` | DELETE | 删除 MCP 服务器 | JWT |
| `/api/v1/fs/list` | GET | 列出目录内容 | JWT |
| `/api/v1/fs/stat` | GET | 获取文件状态 | JWT |
| `/api/v1/fs/mkdir` | POST | 创建目录 | JWT |
| `/api/v1/fs/rename` | POST | 重命名 | JWT |
| `/api/v1/fs/copy` | POST | 复制文件 | JWT |
| `/api/v1/fs/move` | POST | 移动文件 | JWT |
| `/api/v1/fs/delete` | POST | 删除文件 | JWT |
| `/api/v1/terminal/exec` | POST | 执行终端命令 | JWT |

---

## 3. SSE 事件协议

`POST /api/v1/chat/stream` 返回 `text/event-stream`。

### 请求体

```json
{
  "session_id": "可选，不传则自动生成",
  "message": "用户输入文本"
}
```

### 请求头

```
Authorization: Bearer <jwt_token>
Content-Type: application/json
```

### 事件类型及顺序

```
start -> sense -> mode_decision -> [analyze -> plan -> execute -> execute_done]* -> output -> done
```

| event | data 结构 | 前端行为 |
|-------|----------|---------|
| `start` | `{trace_id, session_id, mode}` | 开始渲染，显示 thinking 状态 |
| `sense` | `{status: "ok"|"blocked", reason?}` | blocked -> 显示红色拦截卡片 |
| `mode_decision` | `{mode: "single"|"multi", reason}` | 更新模式指示器 |
| `analyze` | `{round, has_tool_calls, finish_reason, reply_preview}` | 推理链路面板更新 ANALYZE 阶段 |
| `plan` | `{round, tools: [{name, args}]}` | 推理链路面板更新 PLAN，显示将调用的工具 |
| `execute` | `{tool, args, security_check}` | 推理链路面板更新 EXECUTE，显示"正在执行..." |
| `execute_done` | `{tool, status, result_preview, elapsed_ms}` | 更新工具执行结果，耗时 |
| `output` | `{reply, tokens_used, elapsed_ms, mode}` | 渲染最终回复（Markdown） |
| `error` | `{error_code, message, recoverable}` | 显示错误卡片，按 error_code 展示 |
| `done` | `{trace_id, session_id, status}` | 结束 thinking 状态，断开 SSE |

### 注入被拦截时的事件流（短路）

```
start -> sense(blocked) -> error(SAFETY_INJECT_001) -> done(error)
```

---

## 4. 认证流程

### 登录

```
POST /api/v1/auth/login
Content-Type: application/json

{"username": "admin", "password": "admin123"}
```

响应：
```json
{
  "code": 0,
  "data": {
    "token": "eyJhbGci...",
    "expires_at": "2026-06-04T12:00:00Z",
    "user": {"username": "admin", "role": "admin"}
  }
}
```

**注意**: token 在 `data.token`，不是 `token`。

### 使用 Token

所有需要认证的请求在 Header 中携带：
```
Authorization: Bearer <token>
```

### Token 过期处理

- Token 有效期 24 小时
- 收到 401 时，前端清除 token 并跳转到登录页
- 使用 `authFetch`（`web/src/lib/auth.ts`）自动处理认证头和 401 退避

### 登录锁定

- 同一 IP 5 次失败后锁定 3 分钟
- 锁定期间返回 HTTP 429，body 中含 `remaining_seconds`
- 解锁：`curl -X DELETE http://localhost:8080/api/v1/auth/lockout/<IP>`（仅 localhost）

---

## 5. 资源带数据来源

顶部资源带数据通过探针 API 直接获取，不走 LLM：

| 指标 | 数据源 | 刷新频率 |
|------|--------|---------|
| 磁盘使用率 | `POST /api/v1/desktop/probe/disk` | 30s |
| 系统负载 | `POST /api/v1/desktop/probe/top` | 30s |
| 内存 | `POST /api/v1/desktop/probe/memory` | 30s |
| 进程数 | `POST /api/v1/desktop/probe/process` | 30s |
| 端口数 | `POST /api/v1/desktop/probe/network_connections` | 30s |

**注意**: 资源轮询只在用户已登录时运行（`useResourcePolling` 检查 `authToken`）。

---

## 6. 右侧面板数据来源

### Tab 1: 推理链路

数据源：SSE 事件流实时推送。每个事件对应时间线一个节点。

```
SENSE    -> event:sense
ANALYZE  -> event:analyze
PLAN     -> event:plan
EXECUTE  -> event:execute + event:execute_done
OUTPUT   -> event:output
```

### Tab 2: 实时数据

| 图表 | 数据来源 |
|------|---------|
| 磁盘柱状图 | `POST /api/v1/desktop/probe/disk` |
| CPU/内存环形图 | `POST /api/v1/desktop/probe/memory` + `probe/top` |
| 负载条形图 | `POST /api/v1/desktop/probe/top` |
| Top CPU 进程 | `POST /api/v1/desktop/probe/process` |

### Tab 3: 健康状态

数据源：`GET /health`，每 30s 轮询。

---

## 7. CORS 配置

开发模式下后端已内置 CORS 中间件，允许 `localhost` 跨域。前端 dev server 也可用 Vite proxy：

```typescript
// vite.config.ts
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
| 时间 `*_at` | ISO8601 (`2026-05-23T14:32:01.234Z`) | 格式化为 `14:32:01` 或 `5分钟前` |
| 时长 `*_ms` | 整数毫秒 | `<1s` 显示 ms，`>=1s` 显示 `1.2s` |
| ID `sess_xxx` / `trc_xxx` | 前缀+随机 | 前端不显示完整 ID，仅在 debug 面板展示 |
| 布尔 `is_*` / `has_*` | true/false | -- |
| 枚举 | 英文小写 | 前端维护枚举到中文的映射表 |
