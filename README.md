# OPS-Agent

AI 驱动的智能运维助手，通过自然语言对话执行服务器运维操作。

## 技术栈

- **后端**: Go 1.22+ / Chi / JWT / SQLite (纯 Go 驱动)
- **前端**: React 19 / TypeScript / Vite / TailwindCSS v4 / xterm.js
- **架构**: 单二进制部署，前端内嵌，零外部依赖

## 快速开始

```bash
cp .env.example .env
# 编辑 .env，填入 LLM_API_KEY 和 LLM_BASE_URL（必填）
go run ./cmd/server/
# 前端开发：cd web && npm install && npm run dev
```

后端默认监听 `http://localhost:8080`，前端 dev server 默认 `http://localhost:5173`。

## 文档

完整交接文档在 [docs/](docs/) 目录：

| 文档 | 说明 |
|------|------|
| [docs/architecture.md](docs/architecture.md) | 系统架构、模块划分、数据流 |
| [docs/api-reference.md](docs/api-reference.md) | 全部 API 端点详细说明 |
| [docs/frontend-guide.md](docs/frontend-guide.md) | 前端开发规范、组件结构、认证对接 |
| [docs/frontend-integration.md](docs/frontend-integration.md) | 前后端对接规范、SSE 协议 |
| [docs/tool-development.md](docs/tool-development.md) | 如何新增工具 |
| [docs/deployment.md](docs/deployment.md) | 部署方式、龙芯构建、运维排障 |

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
│   └── tools/           # 工具注册与实现（22 个内置工具）
├── web/                 # React 前端
├── deploy/              # 部署脚本和模板
└── docs/                # 交接文档
```

## 核心功能

- **对话模式**: 自然语言输入，AI 自动调用工具执行运维操作
- **桌面模式**: 可视化桌面，直接点击查看磁盘/内存/进程/网络/日志
- **多 Agent 协作**: Planner -> Executor -> Verifier 三角色分析复杂问题
- **安全护栏**: 五层校验（白名单 + 注入检测 + Shell 解析 + 危险参数 + 风险评估）
- **权限确认**: 写操作需用户确认，支持 default/auto_approve/plan 三种模式
- **模型热切换**: 运行时切换 LLM 供应商，无需重启
- **MCP 协议**: 支持外部 MCP 服务器工具注册
- **审计溯源**: 所有操作按 trace_id 记录完整链路
- **龙芯部署**: 交叉编译 loong64，一键安装脚本

## 测试

```bash
go test ./...
cd web && npx tsc --noEmit
```
