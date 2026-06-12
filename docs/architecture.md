# 系统架构

## 整体架构

OPS-Agent 采用单体架构，单二进制部署，前端静态文件内嵌服务。

```
浏览器 <--HTTP/SSE--> Go HTTP Server <--OpenAI API--> LLM
                          |
                    +-----+-----+
                    |           |
               Agent Loop    Tool Registry
                    |           |
               +----+----+  +--+--+
               |         |  |     |
          Single Agent  Multi  22 Tools
          (Tool-Use)   Agent  (Probe/Write)
```

## 模块划分

### cmd/server/main.go -- 入口与路由

所有 HTTP 路由在此注册。启动流程：

1. `config.LoadDotEnv(".env")` -- 加载 .env（剥离引号，已有环境变量不覆盖）
2. `config.Load()` -- 读取配置项
3. 初始化 SQLite 数据库
4. 初始化 LLM 客户端 + 模型池
5. 初始化 MCP 客户端
6. 初始化工具注册表
7. 注册中间件（CORS、限速、JWT）
8. 注册路由
9. 启动 HTTP 服务

### internal/agent/ -- Agent 核心

| 文件 | 职责 |
|------|------|
| `loop.go` | 单 Agent Tool-Use 循环：调用 LLM -> 解析 tool_calls -> 执行工具 -> 回传结果 -> 最多 10 轮 |
| `multi.go` | 多 Agent 协作：Planner 拆解 -> Executor 并行执行 -> Verifier 验证 -> Coordinator 汇总 |
| `plan_mode.go` | 计划模式：先生成计划，用户确认后执行 |
| `compaction.go` | 上下文压缩：token 超过阈值时，LLM 生成摘要替换旧消息 |
| `circuit_breaker.go` | 熔断器：LLM 连续失败时暂停请求 |
| `output_storage.go` | 工具输出持久化：超 4KB 的输出存文件，消息中保留摘要 |
| `complexity.go` | 任务复杂度评估：决定用单 Agent 还是多 Agent |
| `prompt/base.go` | System prompt 构建 |
| `prompt/prompt.go` | Skill/Agent prompt 注入 |

### internal/api/ -- HTTP 处理

| 文件 | 职责 |
|------|------|
| `auth.go` | JWT 认证、登录、IP 锁定/解锁 |
| `ratelimit.go` | 限速中间件（200 req/min） |
| `models.go` | 模型池管理、热切换、连通性测试 |
| `terminal.go` | 终端命令执行 |
| `fs.go` | 文件系统操作 |

### internal/safety/ -- 安全护栏

五层校验流水线，每层独立测试：

| 层 | 文件 | 检查内容 |
|----|------|---------|
| 1 | `rules.go` | 命令白名单（24 条） |
| 2 | `injection.go` | 注入检测（13 条规则，中英双语） |
| 3 | `validator.go` | Shell 解析验证（防绕过） |
| 4 | `flags.go` | 危险参数检测（rm -rf, chmod 777 等） |
| 5 | `warning.go` | 风险等级评估（低/中/高/致命） |

### internal/tools/ -- 工具注册与实现

| 文件 | 职责 |
|------|------|
| `registry.go` | ToolRegistry 接口：Register/Dispatch/List |
| `register.go` | 注册所有内置工具 |
| `tool.go` | Tool 接口定义 |
| `probe_*.go` | 只读探针（disk/process/memory/top/network/logs/service/system） |
| `bash.go` | 受限 Shell 执行 |
| `write_tools.go` | 写操作工具（service_control/truncate_log/delete_file/vacuum_journal/logrotate/kill_process） |
| `multi_agent_tool.go` | 多 Agent 分析工具 |
| `read_output.go` | 读取持久化的工具输出 |
| `file_view.go` | 文件内容查看 |

### internal/llm/ -- LLM 客户端

| 文件 | 职责 |
|------|------|
| `client.go` | OpenAI 兼容协议客户端 |
| `pool.go` | 模型池管理（多供应商配置） |
| `hotreload.go` | 运行时切换活跃模型（无需重启） |
| `models.go` | 模型元数据定义 |
| `types.go` | LLM 请求/响应类型 |
| `errors.go` | LLM 错误处理 |

### internal/store/ -- 数据持久化

| 文件 | 职责 |
|------|------|
| `db.go` | SQLite 连接管理 |
| `schema.sql` | 表结构定义（sessions/messages/audit_logs/configs/mcp_servers） |
| `session.go` | 会话 CRUD |
| `sqlite_session.go` | SQLite 会话实现 |

### internal/config/ -- 配置管理

| 文件 | 职责 |
|------|------|
| `config.go` | 环境变量读取、.env 解析（自动剥离引号） |

### internal/mcp/ -- MCP 协议

| 文件 | 职责 |
|------|------|
| `client.go` | MCP Client 管理器，支持 stdio/SSE/streamable HTTP |

### internal/permission/ -- 权限管理

| 文件 | 职责 |
|------|------|
| `service.go` | 权限模式管理（default/auto_approve/plan），Go channel 阻塞等待用户确认 |

### internal/audit/ -- 审计日志

| 文件 | 职责 |
|------|------|
| `writer.go` | 审计日志写入，按 trace_id 关联完整链路 |

## 数据流

### 对话流程

```
用户输入
  -> POST /api/v1/chat/stream
  -> Agent Loop
     -> SENSE: 安全校验（五层流水线）
     -> ANALYZE: LLM 推理（决定调用哪些工具）
     -> PLAN: 列出工具调用计划
     -> EXECUTE: 执行工具（需权限确认时阻塞等待）
     -> 循环直到 LLM 不再调用工具
     -> OUTPUT: 最终回复
  -> SSE 事件流推送到前端
```

### 认证流程

```
登录请求 -> 校验密码 -> 生成 JWT (HS256, 24h TTL)
  -> 后续请求携带 Authorization: Bearer <token>
  -> JWT 中间件验证
  -> 5 次失败锁定 IP 3 分钟
```

## 数据库表

| 表 | 用途 |
|----|------|
| `sessions` | 对话会话元数据 |
| `messages` | 会话消息（用户/助手/工具） |
| `audit_logs` | 审计日志（按 trace_id 关联） |
| `configs` | 动态配置项 |
| `mcp_servers` | MCP 服务器配置 |
