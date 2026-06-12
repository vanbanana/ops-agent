# API 对接文档

所有需要认证的接口需在请求头携带 `Authorization: Bearer <token>`。

统一响应格式：
```json
{"code": 0, "data": {...}}
```
错误响应：
```json
{"code": 401, "error": "unauthorized"}
```

---

## 认证

### POST /api/v1/auth/login

登录获取 JWT Token。

**请求：**
```json
{"username": "admin", "password": "admin123"}
```

**响应：**
```json
{
  "code": 0,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "expires_at": "2026-06-04T12:00:00Z",
    "user": {"username": "admin", "role": "admin"}
  }
}
```

**注意：** token 在 `data.token`，不是顶层 `token`。

### GET /api/v1/auth/lockouts

查看当前被锁定的 IP 列表（公开接口，无需认证）。

**响应：**
```json
{"code": 0, "data": {"lockouts": [{"ip": "192.168.1.100", "attempts": 5, "remaining_seconds": 120}]}}
```

### DELETE /api/v1/auth/lockout/{ip}

解锁指定 IP（仅允许从 localhost 访问）。

**响应：**
```json
{"code": 0, "message": "IP unlocked"}
```

---

## 健康检查

### GET /health

快速健康检查，不检测 LLM 连通性。

**响应：**
```json
{"status": "healthy", "components": {"llm": {"status": "configured", "vendor": "deepseek-v4-flash"}}}
```

### GET /health/deep

深度检查，实际 ping LLM（3 秒超时）。

**响应：**
```json
{"status": "healthy", "components": {"llm": {"status": "ok", "vendor": "deepseek-v4-flash", "latency_ms": 450}}}
```

### GET /version

版本信息。

**响应：**
```json
{"version": "1.0.0", "git_commit": "abc1234", "build_time": "2026-06-03T12:00:00Z", "go_version": "go1.25.5", "platform": "linux/loong64"}
```

---

## 对话

### POST /api/v1/chat/stream

SSE 流式对话。20 秒心跳保活。

**请求：**
```json
{"session_id": "可选", "message": "看下磁盘使用情况"}
```

**响应：** `text/event-stream`

事件类型按顺序：`start -> sense -> mode_decision -> [analyze -> plan -> execute -> execute_done]* -> output -> done`

详见 [frontend-integration.md](frontend-integration.md) 第 3 节。

### POST /api/v1/chat

同步对话（非流式）。

**请求：**
```json
{"session_id": "可选", "message": "看下磁盘"}
```

**响应：**
```json
{"code": 0, "data": {"reply": "...", "session_id": "sess_xxx"}}
```

---

## 工具管理

### GET /api/v1/tools

获取所有工具定义。

**响应：**
```json
{"code": 0, "data": {"tools": [{"name": "probe_disk", "description": "查看磁盘使用情况", "type": "readonly", "schema": {...}}], "count": 22}}
```

### GET /api/v1/tools/status

获取工具启用/禁用状态。

**响应：**
```json
{"code": 0, "data": {"probe_disk": true, "bash": true, "multi_agent_analyze": true}}
```

### POST /api/v1/tools/toggle

切换工具状态。

**请求：**
```json
{"name": "multi_agent_analyze", "enabled": false}
```

---

## 安全

### GET /api/v1/safety/scan?cmd=xxx

扫描单条命令安全性。

**响应：**
```json
{"code": 0, "data": {"status": "allowed", "reason": "", "detail": ""}}
```

`status` 取值：`allowed` / `blocked` / `warning`

### POST /api/v1/safety/preview

创建风险预览。

**请求：**
```json
{"command": "systemctl restart nginx", "description": "重启 Nginx 服务"}
```

**响应：**
```json
{"code": 0, "data": {"preview_id": "pv_xxx", "risk_level": "medium", "description": "重启服务会导致短暂不可用"}}
```

### POST /api/v1/safety/confirm

确认或拒绝风险预览。

**请求：**
```json
{"preview_id": "pv_xxx", "confirmed": true}
```

---

## 会话

### GET /api/v1/sessions

列出所有会话。

**响应：**
```json
{"code": 0, "data": {"sessions": [{"id": "sess_xxx", "created_at": "...", "message_count": 5, "last_message": "..."}]}}
```

### GET /api/v1/sessions/{id}

获取会话详情（元数据 + 消息）。

### GET /api/v1/sessions/{id}/messages

获取会话消息列表（最近 100 条）。

### DELETE /api/v1/sessions/{id}

删除会话。

---

## 权限

### GET /api/v1/permission/mode

获取当前权限模式。

**响应：**
```json
{"code": 0, "data": {"mode": "default"}}
```

`mode` 取值：`default`（逐条确认）/ `auto_approve`（自动批准）/ `plan`（计划模式）

### PUT /api/v1/permission/mode

设置权限模式。

**请求：**
```json
{"mode": "plan"}
```

### POST /api/v1/permission/respond

响应权限请求。

**请求：**
```json
{"request_id": "perm_xxx", "action": "approve"}
```

`action` 取值：`approve` / `reject`

---

## 计划模式

### POST /api/v1/plan/approve

批准计划。

**请求：**
```json
{"plan_id": "plan_xxx"}
```

### POST /api/v1/plan/reject

拒绝计划。

**请求：**
```json
{"plan_id": "plan_xxx"}
```

---

## 模型管理

### GET /api/v1/models/pool

获取模型池所有供应商。

### PUT /api/v1/models/pool

保存模型池配置。

**请求：**
```json
{"providers": [{"id": "deepseek-flash", "name": "DeepSeek V4 Flash", "provider": "deepseek", "base_url": "https://api.deepseek.com/v1", "api_key": "sk-xxx", "model_id": "deepseek-v4-flash", "context_window": 128000, "max_output": 8192, "is_active": true, "can_reason": true}]}
```

### POST /api/v1/models/switch

切换活跃模型供应商（运行时生效，无需重启）。

**请求：**
```json
{"provider_id": "deepseek-flash"}
```

### POST /api/v1/models/test

测试模型连通性。

**请求：**
```json
{"provider_id": "deepseek-flash"}
```

**响应：**
```json
{"code": 0, "data": {"ok": true, "latency_ms": 450, "model": "deepseek-v4-flash"}}
```

### GET /api/v1/models/active

获取当前活跃模型信息（轻量，用于前端显示）。

---

## MCP 服务器

### GET /api/v1/mcp/servers

获取所有 MCP 服务器配置。

### POST /api/v1/mcp/servers

添加 MCP 服务器。

**请求：**
```json
{"id": "context7", "name": "Context7", "transport": "sse", "url": "https://mcp.context7.com/mcp", "is_active": true}
```

`transport` 取值：`stdio` / `sse` / `streamable-http`

### DELETE /api/v1/mcp/servers/{id}

删除 MCP 服务器。

---

## 桌面操作

### POST /api/v1/desktop/probe/{name}

直接调用探针工具（只读，不经过 LLM）。

`name` 取值：`disk` / `top` / `memory` / `process` / `network_connections` / `network_interfaces` / `logs_journal` / `logs_file` / `service_status` / `file_holders` / `system_info` / `large_files`

**请求：**
```json
{}
```

部分探针支持参数，如 `probe_disk` 可传 `{"path": "/var"}`。

**响应：**
```json
{"code": 0, "data": {"result": "Filesystem      Size  Used Avail Use% ...\n/dev/vda1       100G   45G   55G  45% /"}}
```

### POST /api/v1/desktop/action/{name}

触发写操作工具（需风险预览确认）。

`name` 取值：`bash` / `service_control` / `truncate_log_file` / `delete_file` / `vacuum_journal` / `logrotate_now` / `kill_process`

---

## 文件系统

### GET /api/v1/fs/list?path=/var/log

列出目录内容。

### GET /api/v1/fs/stat?path=/var/log/syslog

获取文件/目录状态。

### POST /api/v1/fs/mkdir

**请求：** `{"path": "/tmp/test"}`

### POST /api/v1/fs/rename

**请求：** `{"old_path": "/tmp/a", "new_path": "/tmp/b"}`

### POST /api/v1/fs/copy

**请求：** `{"src": "/tmp/a", "dst": "/tmp/b"}`

### POST /api/v1/fs/move

**请求：** `{"src": "/tmp/a", "dst": "/tmp/b"}`

### POST /api/v1/fs/delete

**请求：** `{"path": "/tmp/a"}`

---

## 终端

### POST /api/v1/terminal/exec

执行终端命令（经过安全校验）。

**请求：**
```json
{"command": "df -h", "timeout": 30}
```

**响应：**
```json
{"code": 0, "data": {"stdout": "...", "stderr": "", "exit_code": 0, "elapsed_ms": 150}}
```

---

## 审计

### GET /api/v1/audit/logs

获取审计日志列表（最近 100 条，按时间倒序）。

### GET /api/v1/audit/{traceID}

按 trace_id 获取完整请求链路（SENSE -> ANALYZE -> PLAN -> EXECUTE -> OUTPUT）。

---

## 配置

### GET /api/v1/configs?keys=key1,key2

批量获取配置项。

### PUT /api/v1/configs

批量更新配置项。

**请求：**
```json
{"configs": {"permission_mode": "auto_approve", "max_history_messages": 30}}
```
