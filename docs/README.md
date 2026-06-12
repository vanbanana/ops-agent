# OPS-Agent 交接文档

本目录包含 OPS-Agent 项目的完整交接文档，供接手团队快速了解和二次开发。

## 文档索引

| 文档 | 说明 | 适合谁 |
|------|------|--------|
| [architecture.md](architecture.md) | 系统架构、模块划分、数据流、核心设计 | 全员必读 |
| [api-reference.md](api-reference.md) | 全部 API 端点详细说明（请求/响应/示例） | 后端开发、前端对接 |
| [frontend-guide.md](frontend-guide.md) | 前端开发规范、组件结构、认证对接 | 前端开发 |
| [tool-development.md](tool-development.md) | 如何新增工具、工具接口规范 | 后端开发 |
| [deployment.md](deployment.md) | 部署方式、龙芯构建、运维排障 | 运维、部署 |
| [frontend-integration.md](frontend-integration.md) | 前后端对接规范、SSE 协议、数据来源 | 前端对接 |

## 快速上手

1. 先读 [architecture.md](architecture.md) 了解整体架构
2. 本地启动：`cp .env.example .env && go run ./cmd/server/`
3. 前端开发：`cd web && npm install && npm run dev`
4. 对接 API：参考 [api-reference.md](api-reference.md)
5. 新增工具：参考 [tool-development.md](tool-development.md)

## 技术栈

- 后端: Go 1.22+ / Chi / JWT / SQLite (modernc.org/sqlite, 纯 Go)
- 前端: React 19 / TypeScript / Vite / TailwindCSS v4 / xterm.js
- 部署: 单二进制，前端内嵌，零外部依赖
