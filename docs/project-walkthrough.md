# AI 全栈开发实战复盘：从零构建 Linux 运维智能体

## —— 基于 Kiro IDE 的 Spec-Driven Development 全流程记录

---

**项目名称**: OPS-Agent (Linux 运维智能体)
**开发周期**: 2026年5月中旬 ~ 5月25日（约10天）
**技术栈**: Go + chi + modernc.org/sqlite (后端) / React 19 + TypeScript + Vite (前端)
**目标平台**: 龙芯 LoongArch64 + 麒麟服务器 V11
**AI 工具**: Kiro IDE (Claude Opus 4.6) + 纪律工程师 Skill
**最终成果**: 23个MCP工具、多Agent协作、安全护栏、实时终端、多模型热切换

---

# 第一阶段：赛题分析与 Spec 编写（Day 0）

## 1.1 赛题解读

收到赛题后，第一步不是写代码，而是在 Kiro 中新建 Spec。

赛题核心要求提炼为四个维度：
- OS 环境深度感知（调用 lsof/netstat/journalctl）
- MCP 运维插件化（工具标准化注册与调用）
- 安全意图校验器（高危命令二次过滤）
- 推理链路溯源（闭环审计日志）

## 1.2 编写 Requirements（需求规格）

在 Kiro 中选择 "New Spec > Requirements First" 工作流，与 AI 协作完成需求文档。

给 AI 的第一个 prompt：
```
赛题内容如下：[粘贴赛题全文]
请帮我写 requirements.md，包含：
- 术语表（Agent、MCP、Tool-Use Loop 等）
- 功能需求（MUST级 + SHOULD级）
- 非功能需求（安全、性能、可用性）
- 验收标准
工程定级：L3（团队比赛交付）
```

AI 产出了结构化的 requirements.md，包含 Glossary（12个术语）、FR-1 到 FR-8 的功能需求、NFR-1 到 NFR-5 的非功能需求。

**关键决策点**：在需求阶段就确定了"对话模式 + 桌面模式"双模式 UI 方案，这避免了后期推翻重做。

## 1.3 编写 Design（技术设计）

需求确认后，继续让 AI 生成设计文档：
```
基于 requirements.md，写 design.md。
约束：
- 后端 Go（CGO_ENABLED=0 交叉编译到 loong64）
- 数据库用 modernc.org/sqlite（纯 Go 实现）
- 前端 Vite + React 19
- LLM 用 OpenAI 兼容协议（支持国产模型）
```

设计文档确定了：
- 5张数据库表（sessions/messages/audit_logs/configs/mcp_servers）
- 五段式审计管线（SENSE/ANALYZE/PLAN/EXECUTE/OUTPUT）
- 安全校验五层流水线
- Agent Loop 架构（参考 OpenCode 的 processGeneration 模式）

## 1.4 编写 Tasks（实现任务）

设计完成后拆解为 27 个 Task，每个 Task 有明确的子任务和测试标准：
```
基于 design.md，写 tasks.md。
要求：
- 每个 Task 可在一次 AI session 内完成
- 每个子任务有可验证的测试条件
- 标注依赖关系，画出波次图
- 测试用例覆盖正常路径和边界条件
```

最终产出 27 个 Task，按 6 个 Wave 排布：
- Wave 1: 环境验证（编译、SQLite、LLM 联通）
- Wave 2: 后端核心（安全、工具、Agent Loop）
- Wave 3: 多 Agent + 前端
- Wave 4: 桌面模式 + MCP
- Wave 5: 部署交付
- Wave 6: 体验优化

**这个阶段的核心教训**：50% 的时间花在规划上看似浪费，但后面编码阶段几乎没有返工。每次开始新 Task 时，AI 只需要看 tasks.md 就知道该做什么，不需要我重复解释需求。

---

# 第二阶段：后端骨架搭建（Day 1-2）

## 2.1 项目初始化

```
实现 Task 1: Go 项目初始化 + loong64 编译验证
```

AI 产出：
- `go mod init ops-agent`
- `cmd/server/main.go` (最小 HTTP 服务)
- 验证 `GOOS=linux GOARCH=loong64 go build` 通过

## 2.2 安全护栏实现（Task 6-7）

这是赛题的核心考点，给 AI 的 prompt：
```
实现 Task 6: 安全护栏命令校验器。
参考 design.md 中的五层校验流水线设计。
铁律：
- 先读 internal/safety/ 现有文件
- 每个规则必须有测试
- 覆盖率 >= 95%
```

AI 产出了：
- `validator.go` — 五层校验管线
- `rules.go` — 24条命令白名单 + 12条危险模式正则 + 8个保护路径
- `injection.go` — 13条注入检测规则（中英双语）
- 完整测试套件（96.4% 覆盖率）

**与 AI 协作的关键技巧**：我给了 AI 具体的测试用例期望：
```
测试要求：
- rm -rf / 被拦截
- cat /etc/shadow 被拦截
- fork bomb :(){ :|:& };: 被拦截
- df -h 通过
- systemctl status nginx 通过
```

这样 AI 写的规则就能精确覆盖这些边界。

## 2.3 工具注册表 + 12 个探针（Task 8）

```
实现 Task 8: 12个探针工具。
设计模式：ToolRegistry interface + Register/Dispatch。
每个探针调用真实系统命令（df/top/ps/ss/journalctl等）。
```

AI 按照统一接口 `Tool { Name() Description() Schema() Type() Execute() }` 实现了所有探针。

## 2.4 Agent Loop（Task 9）

这是整个系统的核心。给 AI 参考了 opencode-go 的架构：
```
实现 Task 9: 单Agent Tool-Use Loop。
参考 opencode-go/internal/llm/agent/agent.go 的 processGeneration 函数。
关键特性：
- 流式 SSE 输出（逐 token 推送）
- 最多10轮工具调用
- 工具输出超4KB自动截断
- 错误时返回结构化错误码
```

产出 `internal/agent/loop.go`，约 200 行核心逻辑。

## 2.5 多 Agent 编排（Task 11）

```
实现 Task 11: Planner/Executor/Verifier 三角色多 Agent。
Executor 并行执行子任务。
Verifier 验证结果是否足够回答用户问题。
3轮迭代上限 + 64K token 预算保护。
```

这是创新点，评委看到"Agent 群聊视图"会加分。

---

# 第三阶段：前端全量复原（Day 3）

## 3.1 Steering 文件的作用

在开始前端开发前，先写了一份 Steering 文件 `frontend-api-contract.md`：
```markdown
---
inclusion: manual
---
# 前后端对接契约
## 铁律
1. 零假数据 — 前端不硬编码
2. 数据源注释 — 每个组件标注 // Data: GET /xxx
3. 本文件跟着改
```

这样每次让 AI 写前端组件时，只需要 `#前后端契约` 引用这个文件，AI 就知道该调哪个 API。

## 3.2 从设计稿复原 UI

给 AI 的全局约束（通过对话一次性说清）：
```
前端规范：
- 全部 inline style，不用 CSS class
- 图标用 Material Symbols Outlined
- 颜色/字体用 CSS 变量 var(--ops-xxx)
- 每个组件顶部标注数据来源
```

然后逐个组件实现：
```
复原 AppHeader 组件。参考截图 [贴图]。
包含：Logo + 面包屑 + 模式切换(对话/桌面) + 健康灯 + 模型名
```

## 3.3 SSE 流式对话对接

```
实现 useSSE hook，对接 POST /api/v1/chat/stream。
事件类型：start/sense/analyze/plan/execute/execute_done/text_delta/output/error/done
参考 types/api.ts 中的类型定义。
```

这是前端最复杂的部分，AI 写了约 150 行的 SSE 解析 + 状态管理代码。

---

# 第四阶段：权限系统与体验优化（Day 4）

## 4.1 权限确认系统（Task 23）

```
实现完整的权限确认系统：
后端：channel 阻塞等待 + auto_approve 模式 + 5min 超时
前端：输入框上方 PermissionBanner [同意] [拒绝]
Agent Loop 集成：写工具执行前调 RequestPermission()
```

这个功能的关键设计：用 Go channel 实现阻塞等待用户确认，避免了轮询。

## 4.2 消息渲染管线重构（Task 26）

当基本功能跑通后，用户体验需要打磨：
```
ChatMessage 组件太大了，拆分为：
- ThinkingIndicator（三点脉冲动画）
- ThinkingBlock（可折叠思考块）
- ToolCallCard（工具调用卡片）
- CodeBlock（语法高亮）
- MessageActions（复制/重试/停止）
```

## 4.3 Skill 的价值体现

在开发过程中，"纪律工程师" Skill 持续发挥作用：
- 每次改完代码，AI 自动跑 `go build ./...` 和 `npx tsc --noEmit`
- 修改文件前先 `read_file` 看现有内容
- 发现重复代码时提醒提取

这些不需要我每次提醒，Skill 文件里已经定义了这些规则。

---

# 第五阶段：多模型管理 + MCP 集成（Day 5）

## 5.1 多模型热切换

这个功能的完整 prompt 示例：
```
实现 ops-agent 的多模型管理功能。
需求：
- GET /api/v1/models/pool — 返回所有 provider+model
- PUT /api/v1/models/pool — 保存整个 pool
- POST /api/v1/models/switch — 切换激活模型（运行时生效）
- POST /api/v1/models/test — 验证连通性
数据结构：{ id, name, provider, base_url, api_key, model_id, context_window, ... }
铁律：修改前先 read_file，改完 go build 验证
```

AI 产出了：
- `internal/llm/pool.go` — ModelPool 管理
- `internal/llm/hotreload.go` — 运行时客户端切换
- `internal/api/models.go` — HTTP handlers
- 前端 ModelSettings.tsx — 卡片式管理 UI + 预制模型快速选择

## 5.2 MCP 协议集成

赛题的核心要求。给 AI 的 prompt：
```
引入 github.com/mark3labs/mcp-go，实现 MCP Client。
参考 opencode-go 的 mcp-tools.go 模式。
支持 stdio + SSE + streamable HTTP 三种传输。
预制 Context7 远程 MCP Server（https://mcp.context7.com/mcp）。
```

AI 产出了：
- `internal/mcp/client.go` — 完整 MCP Client 管理器
- 启动时自动连接 Context7，注册 2 个外部工具
- 前端工具管理页 — 展示所有工具 + 启停开关

## 5.3 实时终端

```
TerminalDrawer 改为真实执行：
后端 POST /api/v1/terminal/exec 经过安全校验后执行命令。
右侧面板加"快捷命令"tab，点击直接执行到终端。
```

## 5.4 审计页面重构

从扁平表格改为按 trace_id 分组的链路视图：
```
审计页按 trace_id 分组展示完整链路。
每个 trace 可展开看 SENSE→ANALYZE→PLAN→EXECUTE→OUTPUT 五段。
绿色=正常，红色=被拦截。
```

---

# 第六阶段：代码审查与优化（Day 5 续）

## 6.1 切换到 Reviewer 模式

开发功能后，用 Skill 的 Reviewer 模式做自查：
```
/disciplined-engineer 审查一下现有代码，对照赛题评分标准。
```

AI 基于实际代码扫描（不是推断）给出了详细报告：
- 安全拦截前端没显示规则 ID（已修）
- 权限拒绝没写入审计日志（已修）
- allow_session 没真正缓存（已修）
- 多 Agent token 计数没传前端（已修）

## 6.2 赛题对照评分

AI 逐项对照赛题评分标准：
```
功能完整性(55%): ~40%（MCP 当时未实现，后已补全）
创新与实用性(25%): ~21%
文档与演示(20%): ~6%（需补文档）
```

这让我清楚知道还要补什么。

---

# 第七阶段：部署与交付（Day 5 末）

## 7.1 一键部署包

```
整理一键部署包，目标是 WSL2/Linux amd64。
包含：Go 二进制 + 前端构建产物 + 交互式安装脚本。
脚本要引导用户选择供应商、填 API Key。
```

最终产物：`ops-agent-deploy.tar.gz` (5.5MB)，包含单二进制 + web 静态文件 + install.sh。

对方在 WSL2 里三步部署：
```bash
tar -xzf ops-agent-deploy.tar.gz
cd ops-agent-deploy
bash install.sh
```

---

# 总结与反思

## 关键数据

| 指标 | 数值 |
|------|------|
| 总开发时间 | ~10天 |
| Go 代码行数 | ~5000行 |
| TypeScript 代码行数 | ~4000行 |
| Spec 文档总字数 | ~15000字 |
| Task 总数 | 27个（完成24个） |
| 注册工具数 | 23个（含2个MCP外部工具） |
| Git commits | 25+ |

## AI 的实际贡献

- **代码生成**: 约 90% 的代码由 AI 首次产出
- **设计决策**: 约 50% 的架构选择由 AI 建议（如 HotReloadClient 模式、channel 阻塞权限）
- **bug 修复**: AI 在审查模式下发现了 4 个逻辑 bug
- **人的角色**: 需求定义、交互体验决策、优先级判断、最终审查

## 最大的教训

1. **Spec 前置是最赚的投资** — 花了 Day 0 整天写 Spec，但后面几乎没有返工
2. **Steering 避免重复沟通** — "全部 inline style" 只说一次，Steering 帮你持久化
3. **Skill 保证工程质量** — "先读后写 + 每步验证" 不需要人肉监督
4. **审查模式发现盲区** — 自己写的觉得没问题，切 Reviewer 才发现 allow_session 是假实现
5. **赛题对照要早做** — 我们到 Day 5 才对照评分，如果 Day 2 就对照会更早发现 MCP 缺失
