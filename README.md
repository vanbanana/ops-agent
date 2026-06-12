# OPS-Agent

AI 驱动的智能运维助手，通过自然语言对话执行服务器运维操作。

## 技术栈

- **后端**: Go 1.22+ / Chi / JWT / SQLite (纯 Go 驱动)
- **前端**: React 19 / TypeScript / Vite / TailwindCSS v4 / xterm.js
- **架构**: 单二进制部署，前端内嵌，零外部依赖

## 快速开始

### 环境要求

- Go 1.22+
- Node.js 18+
- npm 9+

### 配置

```bash
cp .env.example .env
# 编辑 .env，填入 LLM_API_KEY 和 LLM_BASE_URL（必填）
```

### 启动开发环境

```bash
# 后端
go run ./cmd/server/

# 前端（另一个终端）
cd web && npm install && npm run dev
```

后端默认监听 `http://localhost:8080`，前端 dev server 默认 `http://localhost:5173`（自动代理 API 到 8080）。

### 构建生产版本

```bash
# 构建前端 + 后端
cd web && npm run build && cd ..
go build -ldflags "-s -w" -o ops-agent ./cmd/server/

# 运行
./ops-agent
```

## 项目结构

```
ops-agent/
├── cmd/
│   ├── server/          # HTTP 服务端入口
│   ├── cli/             # CLI 客户端入口
│   └── rtunnel/         # 反向 TCP 隧道工具
├── internal/
│   ├── agent/           # Agent 循环、多 Agent 协作、压缩、熔断
│   ├── api/             # HTTP handler：认证、限速、终端、文件系统
│   ├── audit/           # 审计日志写入
│   ├── config/          # 配置加载、.env 解析
│   ├── llm/             # LLM 客户端、模型池、热重载
│   ├── mcp/             # MCP 协议客户端
│   ├── permission/      # 权限模式管理
│   ├── safety/          # 命令安全校验、注入检测、风险预览
│   ├── store/           # SQLite 存储、会话持久化
│   └── tools/           # 工具注册与实现（探针、bash、文件操作等）
├── web/
│   ├── src/
│   │   ├── components/  # UI 组件
│   │   ├── pages/       # 页面（登录、设置、桌面模式等）
│   │   ├── hooks/       # React hooks（SSE、健康检查、资源轮询）
│   │   ├── stores/      # Zustand 状态管理
│   │   ├── lib/         # 工具函数（认证、demo 模式）
│   │   └── types/       # TypeScript 类型定义
│   └── public/          # 静态资源
├── deploy/
│   ├── build-loong64.sh              # 龙芯架构构建脚本
│   ├── ops-agent-deploy-loong64/     # 龙芯部署包模板
│   │   ├── install.sh                # 交互式安装脚本
│   │   ├── uninstall.sh              # 卸载脚本
│   │   ├── .env.example              # 环境变量模板
│   │   ├── providers.json.example    # 多模型配置模板
│   │   └── web/                      # 前端静态文件（构建时填充）
│   └── ops-agent-deploy/             # amd64 部署包模板
├── docs/                # 架构文档
├── .env.example         # 环境变量模板
├── providers.json.example  # 多模型配置模板
└── go.mod
```

## 配置项

通过 `.env` 文件或环境变量配置：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `LLM_API_KEY` | (必填) | LLM API 密钥 |
| `LLM_BASE_URL` | (必填) | LLM API 地址 |
| `LLM_MODEL` | `mimo-v2.5-pro` | 默认模型 ID |
| `PORT` | `8080` | 服务监听端口 |
| `JWT_SECRET` | 自动生成 | JWT 签名密钥。以 `dev-` 开头则跳过认证 |
| `DB_PATH` | `./data/ops-agent.db` | SQLite 数据库路径 |
| `ADMIN_PASSWORD` | `admin123` | 管理员登录密码 |

`.env` 文件中已设置的变量不会被环境变量覆盖。值不需要加引号。

### 多模型配置

复制 `providers.json.example` 为 `providers.json`，可配置多个 LLM 供应商并热切换：

```json
[
  {
    "id": "deepseek-flash",
    "name": "DeepSeek V4 Flash",
    "provider": "deepseek",
    "base_url": "https://api.deepseek.com/v1",
    "api_key": "sk-your-key",
    "model_id": "deepseek-v4-flash",
    "context_window": 128000,
    "max_output": 8192,
    "is_active": true,
    "can_reason": true
  }
]
```

## API 概览

### 公开端点（无需认证）

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/auth/login` | 登录，返回 JWT |
| GET | `/health` | 健康检查 |
| GET | `/health/deep` | 深度检查（含 LLM 连通性） |
| GET | `/version` | 版本信息 |

### 认证端点

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/auth/lockouts` | 查看被锁定的 IP |
| DELETE | `/api/v1/auth/lockout/{ip}` | 解锁 IP（仅 localhost） |

### 对话

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/chat/stream` | SSE 流式对话 |
| POST | `/api/v1/chat` | 同步对话 |

### 工具管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/tools` | 工具列表 |
| GET | `/api/v1/tools/status` | 工具启用状态 |
| POST | `/api/v1/tools/toggle` | 切换工具状态 |

### 桌面探针

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/desktop/probe/{name}` | 调用探针（只读） |
| POST | `/api/v1/desktop/action/{name}` | 触发写操作（需确认） |

### 其他

| 方法 | 路径 | 说明 |
|------|------|------|
| GET/PUT | `/api/v1/permission/mode` | 权限模式 |
| GET/PUT | `/api/v1/configs` | 配置管理 |
| GET/PUT | `/api/v1/models/pool` | 模型池 |
| POST | `/api/v1/models/switch` | 切换模型 |
| POST | `/api/v1/safety/preview` | 风险预览 |
| POST | `/api/v1/safety/confirm` | 确认风险操作 |
| GET/POST | `/api/v1/sessions` | 会话管理 |
| GET/POST | `/api/v1/fs/*` | 文件系统操作 |
| POST | `/api/v1/terminal/exec` | 终端命令执行 |
| GET | `/api/v1/audit/logs` | 审计日志 |
| GET/POST/DELETE | `/api/v1/mcp/servers` | MCP 服务器管理 |

## 内置工具

### 只读探针

| 工具 | 说明 |
|------|------|
| `probe_disk` | 磁盘使用情况 |
| `probe_large_files` | 大文件查找 |
| `probe_process` | 进程列表 (CPU/MEM Top10) |
| `probe_top` | 系统负载概览 |
| `probe_memory` | 内存使用情况 |
| `probe_network_connections` | 网络连接和监听端口 |
| `probe_network_interfaces` | 网络接口信息 |
| `probe_logs_journal` | systemd 日志 |
| `probe_logs_file` | 日志文件查看 |
| `probe_service_status` | 服务状态 |
| `probe_file_holders` | 文件占用查看 (lsof) |
| `probe_system_info` | 系统基本信息 |
| `file_view` | 文件内容查看（带行号） |

### 写操作（需风险确认）

| 工具 | 说明 |
|------|------|
| `bash` | 受限 Shell 命令执行 |
| `service_control` | 服务启停 (start/stop/restart) |
| `truncate_log_file` | 清空日志文件 |
| `delete_file` | 删除文件 |
| `vacuum_journal` | 清理 journal 日志 |
| `logrotate_now` | 执行日志轮转 |
| `kill_process` | 终止进程 (SIGTERM) |

### 多 Agent

| 工具 | 说明 |
|------|------|
| `multi_agent_analyze` | 多 Agent 协作分析（Planner->Executor->Verifier） |

## 安全机制

- **JWT 认证**: 生产模式强制 JWT，Token 有效期 24 小时
- **登录保护**: 同一 IP 5 次失败后锁定 3 分钟
- **限速**: 200 请求/分钟（按 Token 或 IP）
- **命令安全校验**: 白名单 + 注入检测 + 解析验证
- **风险预览**: 写操作需用户确认后才执行
- **权限模式**: default（逐条确认）/ auto_approve（自动批准）/ plan（计划模式）
- **审计日志**: 所有操作记录可追溯

## 龙芯部署

### 构建

```bash
bash deploy/build-loong64.sh [版本号]
# 默认版本号 1.0.0
# 产物: deploy/ops-agent-loong64-v1.0.0.tar.gz
```

### 安装

```bash
tar xzf ops-agent-loong64-v1.0.0.tar.gz
cd ops-agent-deploy-loong64
bash install.sh
```

install.sh 会自动完成：架构检查 -> 依赖检查 -> LLM 配置 -> systemd 服务注册 -> 健康检查。

### 卸载

```bash
bash uninstall.sh
```

## 反向隧道 (rtunnel)

用于让 NAT 后的服务器可被外部 SSH 访问。

```bash
# 公网服务器
rtunnel -mode server -tunnel-port 7000 -ssh-port 2222 -secret YOUR_SECRET

# 内网服务器
rtunnel -mode client -tunnel <公网IP>:7000 -forward 127.0.0.1:22 -secret YOUR_SECRET

# SSH 连接
ssh -p 2222 user@<公网IP>
```

断线自动重连，纯 Go 静态编译，零依赖。

## 测试

```bash
# Go 测试
go test ./...

# 前端类型检查
cd web && npx tsc --noEmit
```

## 开发约定

- 后端代码在 `internal/` 下，按功能分包，不跨包直接访问内部实现
- 前端认证统一使用 `authFetch`（`web/src/lib/auth.ts`），自动携带 JWT
- 工具注册在 `internal/tools/register.go`，新增工具实现 `Tool` 接口即可
- API 路由定义在 `cmd/server/main.go`
- 环境变量值不加引号，`LoadDotEnv` 会自动剥离引号
