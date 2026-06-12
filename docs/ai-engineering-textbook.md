# AI 辅助软件工程实战

## —— 企业级全栈开发方法论与工具实践

---

**课程定位**: 高校计算机专业高年级选修 / 企业新员工培训
**课时规划**: 48学时（理论16 + 实践32）
**配套案例**: OPS-Agent（Linux 运维智能体，完整项目代码随书提供）

---

# 绪论：AI 正在重塑软件工程

## 0.1 背景与动机

2024-2026 年，AI 编码工具经历了三代演进：

| 代际 | 代表工具 | 能力 | 开发者角色 |
|------|----------|------|-----------|
| 第一代 (2023) | GitHub Copilot | 行级补全 | 写代码，AI 补全 |
| 第二代 (2024) | Cursor, Claude Code | 文件级生成 | 描述意图，AI 写文件 |
| 第三代 (2025-2026) | Kiro, OpenCode, Codex | 项目级 Agent | 定义规格，AI 全栈实现 |

第三代工具不再是"补全助手"，而是**自主 Agent**——它可以读代码、改代码、跑命令、调 API、运行测试，直到任务完成。开发者的角色从"写代码的人"变为"定义规格并审查产出的人"。

这要求开发者掌握一套全新的技能：

- 如何写出 AI 能准确执行的规格文档
- 如何约束 AI 的行为（避免它做多或做错）
- 如何高效审查 AI 的产出
- 如何管理 AI 参与的开发流程

## 0.2 本书结构

```
第一篇：方法论（第1-3章）
  AI 辅助开发的工程流程、角色变化、质量保障

第二篇：工具实践（第4-7章）
  主流 Agent 工具的使用方法（Cursor/Claude Code/OpenCode/Kiro/Copilot）

第三篇：全栈实战（第8-12章）
  以 OPS-Agent 项目为案例，完整复现从需求到部署的全过程

第四篇：企业实践（第13-15章）
  团队协作、CI/CD 集成、安全合规、技术债管理
```

## 0.3 学习目标

完成本课程后，学生能够：
1. 独立使用 AI Agent 工具完成企业级全栈项目开发
2. 编写 AI 可执行的规格文档（需求/设计/任务）
3. 建立 AI 行为约束体系（Steering/Rules/Hooks）
4. 在团队中组织 AI 辅助的代码审查流程
5. 评估 AI 产出的质量并做出技术决策

---

# 第一篇：方法论

# 第1章 AI 辅助软件开发生命周期

## 1.1 传统 SDLC vs AI-Augmented SDLC

传统瀑布/敏捷流程中，每个阶段主要由人完成。AI 时代的变化：

| SDLC 阶段 | 传统方式 | AI 辅助方式 | 人的新职责 |
|-----------|----------|-----------|-----------|
| 需求分析 | PM 写 PRD | AI 从会议纪要/用户反馈提炼需求 | 审核、补充业务上下文 |
| 系统设计 | 架构师画图写文档 | AI 生成架构方案供评审 | 技术决策、trade-off 判断 |
| 编码实现 | 开发者逐行编写 | AI Agent 按 spec 批量生成 | 审查、集成测试 |
| 测试 | QA 写用例执行 | AI 生成测试用例 + 自动执行 | 定义测试策略、边界场景 |
| 部署运维 | DevOps 写脚本 | AI 生成 Dockerfile/CI 配置 | 审批发布、监控告警 |

## 1.2 50-20-30 时间分配原则

2026 年业界共识的最佳实践时间分配：

- **50% 规划** — 写清楚要做什么（Spec）
- **20% 生成** — AI 写代码，人审查
- **30% 验证** — 测试、审查、修复

为什么规划占 50%？因为 AI 的代码质量直接取决于 Spec 的质量。模糊的 prompt 产出模糊的代码；精确的 Spec 产出可部署的系统。

## 1.3 Spec-Driven Development (SDD)

SDD 是第三代 AI 工具的核心方法论。三份文档构成完整规格：

```
Requirements (需求规格)
  ↓ 回答"做什么"
Design (技术设计)
  ↓ 回答"怎么做"
Tasks (实现任务)
  ↓ 回答"分几步做"
Code (代码实现)
  ← AI 根据 Tasks 逐步生成
```

**核心原则**：Spec 是源代码的源头。需求变了改 Spec，从 Spec 重新生成代码。

## 1.4 AI 辅助开发中的角色定义

| 角色 | 职责 | 与 AI 的关系 |
|------|------|-------------|
| Product Owner | 定义业务需求 | 提供业务上下文给 AI |
| Tech Lead | 技术决策 + 架构设计 | 审查 AI 的设计方案 |
| Developer | 编写 Spec + 审查代码 | 引导 AI 实现 + 验证产出 |
| QA | 测试策略 + 边界场景 | AI 生成测试用例，QA 补充边界 |
| DevOps | 部署流程设计 | AI 生成 CI/CD 配置，DevOps 审批 |

---

# 第2章 AI Agent 工具全景

## 2.1 工具分类

| 类别 | 代表工具 | 适用场景 | 集成方式 |
|------|----------|----------|----------|
| IDE 内置 Agent | Cursor, Windsurf | 日常编码，需要 IDE 上下文 | IDE 插件 |
| CLI Agent | Claude Code, OpenCode, Codex | 终端环境、CI/CD、服务器 | 命令行 |
| 规格驱动 IDE | Kiro | 需要 Spec 管理的项目 | 独立 IDE |
| API Agent | GPT-4/Claude API | 自定义 Agent 开发 | HTTP API |
| 辅助补全 | Copilot, Codeium | 行级补全 + 简单生成 | IDE 插件 |

## 2.2 主流工具对比

| 特性 | Cursor | Claude Code | OpenCode | Kiro | Copilot |
|------|--------|-------------|----------|------|---------|
| 运行环境 | IDE (VS Code fork) | 终端 CLI | 终端 CLI | IDE | IDE 插件 |
| 模型 | 多模型 | Claude only | 75+ providers | Claude | GPT/Claude |
| 文件编辑 | 直接 | 直接 | 直接 | 直接 | 建议 |
| 终端执行 | 有 | 有 | 有 | 有 | 无 |
| Spec 管理 | 无 | 无 (.claude/) | 无 | 有 (.kiro/specs) | 无 |
| MCP 支持 | 有 | 有 | 有 | 有 (Powers) | 有限 |
| 开源 | 否 | 否 | 是 | 否 | 否 |
| 价格 | $20/月 | $20/月 (Max) | 免费 | 免费 | $10/月 |

## 2.3 如何选择工具

决策树：

```
需要 Spec 管理吗？
  是 → Kiro
  否 → 
    需要团队协作？
      是 → Cursor (共享规则文件) / Claude Code (.claude/)
      否 →
        需要开源？
          是 → OpenCode
          否 → Claude Code (最强 Agent 能力)
```

## 2.4 通用概念（跨工具适用）

无论用哪个工具，以下概念是通用的：

| 概念 | Cursor | Claude Code | OpenCode | Kiro |
|------|--------|-------------|----------|------|
| 项目级指令 | .cursor/rules | CLAUDE.md | .opencode | .kiro/steering |
| AI 角色定义 | Custom instructions | 无 | 无 | .kiro/skills |
| 自动化 Hook | 无 | Pre/Post hooks | 无 | .kiro/hooks |
| 外部工具 | MCP | MCP | MCP | Powers (MCP) |

---

# 第3章 质量保障体系

## 3.1 AI 产出的常见问题

| 问题类型 | 表现 | 频率 | 影响 |
|----------|------|------|------|
| 幻觉代码 | 调用不存在的 API | 高 | 编译失败 |
| 风格不一致 | 新代码与现有代码风格冲突 | 中 | 维护困难 |
| 过度工程 | 添加不需要的抽象/功能 | 高 | 复杂度膨胀 |
| 安全漏洞 | 硬编码密钥、SQL 注入 | 低但致命 | 安全事故 |
| 测试缺失 | 只写实现不写测试 | 高 | 回归风险 |

## 3.2 防护机制设计

三层防护体系：

**第一层：预防（Spec + Rules）**
- 写清楚需求和约束，减少 AI 的"创造空间"
- 通过项目级规则文件限制 AI 行为

**第二层：检测（验证 + 审查）**
- 每步改动后编译/测试
- Code Review（人或 AI Reviewer 模式）
- 静态分析（lint）

**第三层：修复（诊断 + 重试）**
- 失败两次换策略（不做增量补丁）
- 根因分析而非症状修复

## 3.3 验证铁律

无论使用哪个 AI 工具，以下规则必须遵守：

1. **每次文件修改后立即编译** — 不积累错误
2. **新功能必须有测试** — AI 写的测试也可以
3. **安全问题零容忍** — 密钥不入码，输入必验证
4. **先读后写** — 修改前理解现有代码
5. **提交前 diff review** — 至少扫一眼 AI 改了什么

---

# 第二篇：工具实践

# 第4章 规格文档编写方法

## 4.1 需求文档（Requirements）

### 结构模板

```markdown
# Requirements

## Introduction
项目背景、目标、约束条件

## Glossary
术语定义（消除歧义）

## Functional Requirements
### FR-1: [功能名称]
系统应...
验收标准：
- 条件A成立时，结果B

## Non-Functional Requirements
### NFR-1: 性能
### NFR-2: 安全
### NFR-3: 可用性

## Out of Scope
明确不做什么
```

### 编写原则

- **可测试性**：每条需求都对应一个可验证的测试用例
- **无歧义**：用"应"(shall) 而非"可能"(may)
- **独立性**：每条需求可独立验证
- **可追溯**：需求有编号，后续 design/tasks 可引用

## 4.2 设计文档（Design）

### 结构模板

```markdown
# Design

## Architecture Overview
分层架构图 (Mermaid)

## Data Model
数据库表定义 / 数据结构

## API Contract
端点、方法、请求/响应格式

## Technical Decisions
ADR 格式记录关键决策

## Security Design
认证、授权、数据保护

## Error Handling
错误码定义、降级策略
```

### 与 AI 协作写设计的方法

```
基于 requirements.md 写 design.md。
约束：
- [技术栈]
- [部署环境]
- [性能要求]
请给出至少两个架构方案，比较 trade-off，推荐其一。
```

## 4.3 任务文档（Tasks）

### 结构模板

```markdown
# Tasks

- [ ] 1. 任务名
  - [ ] 1.1 子任务（原子级）
  - [ ] 1.2 测试: [验证条件]

## Dependency Graph
[哪些可并行，哪些有前置依赖]
```

### 粒度标准

| 等级 | 描述 | 示例 |
|------|------|------|
| 太大 | 一天做不完 | "实现前端" |
| 太小 | 改一行 | "在第5行加分号" |
| 刚好 | 1-4小时，一次 commit | "实现登录表单 + 表单校验 + 错误提示" |

---

# 第5章 AI 行为约束系统

## 5.1 项目级规则文件

所有 Agent 工具都支持某种形式的项目级规则：

**Cursor**: `.cursor/rules`
```
You are working on a Go backend + React frontend project.
Always use inline styles, never CSS classes.
Run `go build ./...` after every Go file change.
```

**Claude Code**: `CLAUDE.md`
```markdown
# Project Rules
- Backend: Go + chi router
- Frontend: React 19 + TypeScript + Vite
- Style: inline style only, Material Symbols icons
- Verify: go build after every change
```

**OpenCode**: `.opencode/AGENTS.md`
```markdown
# Agent Instructions
- Read files before modifying them
- Match existing code patterns
- Run tests after changes
```

**Kiro**: `.kiro/steering/*.md`
```markdown
---
inclusion: always
---
# 编码规范
- 全部 inline style
- 图标: Material Symbols Outlined
- 每个组件标注数据来源
```

## 5.2 Skill / Custom Instructions

给 AI 一个"角色"，让它始终按某种规范行事：

**示例：代码审查员角色**
```markdown
你是一个代码审查员。审查代码时：
1. 先识别语言和框架
2. 检查安全漏洞（最高优先）
3. 检查逻辑正确性
4. 检查可维护性
5. 输出结构化报告（阻塞/警告/建议）
```

**示例：全栈工程师角色**
```markdown
你是一个全栈工程师。写代码时：
1. 先读目标文件和直接依赖
2. 匹配现有代码模式（命名、结构、风格）
3. 改完立即验证（编译/测试）
4. 不做需求范围外的事
```

## 5.3 自动化 Hook

Hook 允许在 AI 操作前后自动执行检查：

**保存时自动 lint：**
```json
{
  "event": "fileEdited",
  "patterns": ["*.go"],
  "action": "runCommand",
  "command": "golangci-lint run"
}
```

**AI 写文件前审查：**
```json
{
  "event": "preToolUse",
  "toolTypes": ["write"],
  "action": "askAgent",
  "prompt": "确认这次写操作不会破坏现有测试"
}
```

---

# 第6章 与 AI 高效协作的 Prompt 策略

## 6.1 Prompt 的四个层次

| 层次 | 效果 | 示例 |
|------|------|------|
| 模糊 | 质量不可控 | "帮我写个后端" |
| 具体 | 基本可用 | "用 Go + chi 写一个 REST API" |
| 有约束 | 高质量 | "用 Go + chi 写 REST API，参考 design.md 中的 schema" |
| Spec 驱动 | 可复现 | "实现 Task 6.1: validator.go 五层校验，参考 requirements FR-3" |

## 6.2 关键 Prompt 模板

**新功能实现：**
```
实现 [Task 编号]: [功能描述]。
参考：[设计文档路径]
约束：
- [技术约束]
- [命名规范]
- [测试要求]
铁律：修改前先 read_file，改完 go build 验证。
```

**Bug 修复：**
```
[粘贴错误信息]
在 [文件路径] 中修复这个问题。
不要改其他不相关的文件。
修完运行 [测试命令] 验证。
```

**代码审查：**
```
审查 [文件/目录]。
项目等级：[L1-L5]
重点关注：[安全/性能/可维护性]
输出格式：按严重度分级（阻塞/警告/建议）
```

## 6.3 常见反模式

| 反模式 | 后果 | 正确做法 |
|--------|------|----------|
| 一次给太大的任务 | 输出质量下降 | 拆成原子任务 |
| 不给上下文 | AI 猜测导致不一致 | 引用 Spec + 现有代码 |
| 不验证就继续 | 错误累积 | 每步 build/test |
| AI 报错直接追问 | 陷入补丁循环 | 两次失败后换策略 |
| 不审查 diff | 不知道系统怎么变了 | 至少扫关键文件 |

---

# 第7章 MCP 协议与工具扩展

## 7.1 什么是 MCP

Model Context Protocol (MCP) 是 Anthropic 于 2024 年底推出的开放标准，让 AI Agent 能通过标准协议调用外部工具。

```
AI Agent ←[MCP协议]→ MCP Server ←[执行]→ 外部系统
```

## 7.2 MCP 的传输方式

| 方式 | 场景 | 示例 |
|------|------|------|
| stdio | 本地子进程 | `npx @upstash/context7-mcp` |
| SSE | 远程 HTTP 长连接 | 旧版远程 MCP |
| Streamable HTTP | 远程标准 HTTP | `https://mcp.context7.com/mcp` |

## 7.3 MCP 在开发中的应用

| MCP Server | 用途 |
|------------|------|
| Context7 | 查询最新技术文档 |
| GitHub MCP | 操作 GitHub Issues/PR |
| Grafana MCP | 查询监控指标 |
| Kubernetes MCP | 管理 K8s 集群 |
| 自定义 MCP | 封装企业内部工具 |

---

# 第三篇：全栈实战

# 第8章 项目启动：从赛题到 Spec

（以 OPS-Agent 项目为完整案例）

## 8.1 赛题分析方法

1. 提炼关键词：MCP、安全护栏、推理溯源、LoongArch
2. 映射到技术需求：协议实现、规则引擎、审计日志、交叉编译
3. 评估评分权重：功能55%、创新25%、文档20%
4. 确定 MVP 范围：先做 55% 功能分的核心项

## 8.2 Spec 三文档实操

（使用任何 Agent 工具的 prompt 策略，附完整 prompt 和产出示例）

## 8.3 技术选型决策记录

（Go/SQLite/React 的选型理由，ADR 格式）

---

# 第9章 后端开发实战

## 9.1 从 Design 到代码结构
## 9.2 安全护栏的 TDD 开发过程
## 9.3 Agent Loop 的参考架构（OpenCode 模式）
## 9.4 多 Agent 编排设计

---

# 第10章 前端开发实战

## 10.1 设计系统约定的固化（Steering）
## 10.2 从设计稿到组件的 AI 协作
## 10.3 SSE 流式通信的实现
## 10.4 状态管理策略

---

# 第11章 系统集成与优化

## 11.1 前后端联调流程
## 11.2 AI 辅助的代码审查（Reviewer 模式）
## 11.3 性能优化与体验打磨
## 11.4 MCP 协议集成

---

# 第12章 部署与交付

## 12.1 交叉编译与多平台构建
## 12.2 一键部署脚本设计
## 12.3 生产环境配置管理
## 12.4 交付物清单

---

# 第四篇：企业实践

# 第13章 团队中的 AI 协作

## 13.1 代码所有权与责任
## 13.2 AI 生成代码的 Code Review 流程
## 13.3 Spec 的版本控制与协作
## 13.4 知识管理（Steering 作为团队知识库）

---

# 第14章 CI/CD 中的 AI 集成

## 14.1 PR 自动审查（AI Reviewer）
## 14.2 AI 生成的测试在 CI 中的角色
## 14.3 部署审批与安全扫描
## 14.4 监控与可观测性

---

# 第15章 安全合规与伦理

## 15.1 AI 生成代码的版权问题
## 15.2 数据安全（代码是否发送到云端）
## 15.3 AI 幻觉的风险管理
## 15.4 审计追溯与合规要求

---

# 附录

## A. 工具安装与配置速查
## B. 常用 Prompt 模板库
## C. 项目 Spec 文档完整示例
## D. 参考文献与推荐阅读
