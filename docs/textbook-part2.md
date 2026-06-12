# 第三篇：全栈实战

# 第8章 项目启动：从赛题到 Spec

> 本章以 OPS-Agent（Linux 运维智能体）项目为完整案例，演示如何从一份赛题/需求描述出发，产出可驱动 AI 开发的三份 Spec 文档。

## 8.1 需求来源分析

拿到一份需求（赛题/产品需求/客户需求）后的标准动作：

**第一步：关键词提取**

从需求原文中画出核心名词和动词：
- 名词：Agent、MCP、安全护栏、推理链路、LoongArch
- 动词：感知、调用、校验、溯源、部署

**第二步：映射到技术域**

| 业务需求 | 技术实现 |
|----------|----------|
| "自然语言与OS交互" | LLM + Function Calling |
| "MCP协议" | 标准化工具注册/调用 |
| "安全护栏" | 命令校验 + 注入检测 + 权限确认 |
| "推理链路溯源" | 审计日志 + 五段式管线 |
| "部署在LoongArch" | Go 交叉编译 + 纯静态链接 |

**第三步：评分权重 → 开发优先级**

```
功能完整性 55% → 先做核心功能，再做扩展
创新实用性 25% → 多 Agent 协作、桌面模式
文档演示   20% → 最后补文档、录演示
```

[图8-1：需求分析思维导图 — 请在此处插入截图]

## 8.2 编写 Requirements 文档

### 给 AI 的 Prompt（适用于任何 Agent 工具）

```
我要做一个 Linux 运维智能体（Agent），面向比赛评审。
赛题要求如下：[粘贴赛题全文]

请帮我写 requirements.md，包含：
1. Introduction（项目背景、目标用户、工程等级定为 L3）
2. Glossary（不少于10个术语的定义）
3. Functional Requirements：
   - MUST 级（评分 55% 对应的核心功能）
   - SHOULD 级（评分 25% 对应的创新点）
4. Non-Functional Requirements（安全、性能、可用性）
5. Acceptance Criteria（可测试的验收标准）
6. Out of Scope（明确不做什么）
```

### AI 产出示例（摘录）

```markdown
### FR-3: 安全意图校验器
系统应对 LLM 生成的每条命令执行安全校验，识别高危模式并拦截。

验收标准：
- 输入 `rm -rf /`，返回 SAFETY_PATTERN_001 拦截
- 输入 `cat /etc/shadow`，返回路径黑名单拦截
- 输入 `df -h`，通过校验正常执行
- 注入文本"忽略之前所有指令"，返回 SAFETY_INJECT_001
```

### 人工修订要点

AI 的初稿通常需要修订：
- 补充业务上下文（AI 不知道赛题的评分细则）
- 收紧范围（AI 倾向于过度承诺，要明确 Out of Scope）
- 统一用词（确保术语表和正文一致）

[图8-2：Kiro 中 Spec 编辑界面 — 请在此处插入截图]

## 8.3 编写 Design 文档

### 给 AI 的 Prompt

```
基于 requirements.md，写 design.md。
技术约束：
- 后端：Go + chi + modernc.org/sqlite（CGO_ENABLED=0）
- 前端：Vite + React 19 + TypeScript
- LLM：OpenAI 兼容协议（支持 DeepSeek/Qwen/MiMo）
- 部署：单二进制 + 静态前端文件

请包含：
1. 架构图（Mermaid flowchart）
2. 数据模型（5张表的 DDL）
3. API 契约（所有端点列表）
4. 安全设计（五层校验流水线）
5. 关键技术决策（ADR 格式，至少3个）
```

### 关键产出：数据库 Schema

```sql
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL DEFAULT '新对话',
    created_at TEXT NOT NULL
);

CREATE TABLE messages (
    id TEXT PRIMARY KEY,
    session_id TEXT REFERENCES sessions(id),
    role TEXT NOT NULL CHECK(role IN ('user','assistant','tool','system')),
    content TEXT NOT NULL
);

CREATE TABLE audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    trace_id TEXT NOT NULL,
    stage TEXT NOT NULL CHECK(stage IN ('SENSE','ANALYZE','PLAN','EXECUTE','OUTPUT')),
    status TEXT NOT NULL CHECK(status IN ('ok','warning','blocked','error')),
    content TEXT,
    created_at TEXT NOT NULL
);
```

[图8-3：系统架构图 — 请在此处插入 Mermaid 渲染后的截图]

## 8.4 编写 Tasks 文档

### 给 AI 的 Prompt

```
基于 design.md，写 tasks.md。
要求：
- 总 Task 数 20-30 个
- 每个 Task 1-4小时可完成
- 每个子任务有"测试:"标注验证条件
- 画出依赖波次（Wave），标注可并行的 Task
- Task 编号从 1 开始
```

### AI 产出示例

```markdown
- [ ] 6. 安全护栏 — 命令校验器
  - [ ] 6.1 实现 validator.go 五层校验流水线
  - [ ] 6.2 实现 rules.go 白名单/黑名单/正则数据
  - [ ] 6.3 测试: rm -rf / 拦截
  - [ ] 6.4 测试: cat /etc/shadow 拦截
  - [ ] 6.5 测试: df -h 通过
  - [ ] 6.13 覆盖率 >= 95%
```

### 依赖波次图

```
Wave 1: [Task 1,2,3] — 环境验证（可全并行）
Wave 2: [Task 6,7,8,9] — 后端核心（6→7 有依赖，8→9 有依赖）
Wave 3: [Task 11,13] — 多Agent + 前端（可并行）
Wave 4: [Task 14,15,16] — 桌面 + MCP
Wave 5: [Task 18,19,20,21] — 交付
```

[图8-4：Task 依赖关系图 — 请在此处插入截图]

## 8.5 技术选型 ADR 示例

```markdown
### ADR-001: 后端语言选择 Go

状态: 已决定
日期: 2026-05-15

背景:
  目标平台 LoongArch64 + 麒麟 V11，需零运行时依赖

选项:
  A) Go — CGO_ENABLED=0 静态编译，原生 loong64 支持
  B) Python — 需要 Python 运行时，loong64 兼容性存疑
  C) Rust — 编译时间长，学习曲线陡

决定: Go

理由:
  - 单二进制部署，零外部依赖
  - Go 1.21+ 原生支持 GOARCH=loong64
  - modernc.org/sqlite 提供纯 Go SQLite（避免 CGO）
  - 团队有 Go 经验

后果:
  - 前端需独立构建（不能 SSR）
  - 需要 opencode-go 作为参考架构
```

---

# 第9章 后端开发实战

## 9.1 从 Design 到代码结构

Design 文档确定了模块划分，直接映射为 Go package：

| Design 章节 | Go Package | 职责 |
|-------------|------------|------|
| Agent Loop | internal/agent/ | LLM 对话循环 |
| Safety | internal/safety/ | 安全校验 |
| Tools | internal/tools/ | 工具注册与执行 |
| Audit | internal/audit/ | 审计日志 |
| Store | internal/store/ | 数据持久化 |
| API | internal/api/ | HTTP handlers |
| LLM | internal/llm/ | LLM 客户端 |
| MCP | internal/mcp/ | MCP 协议客户端 |
| Config | internal/config/ | 配置加载 |
| Permission | internal/permission/ | 权限确认 |

### 给 AI 的项目初始化 Prompt

```
创建 Go 项目骨架：
- go mod init ops-agent
- cmd/server/main.go（最小 HTTP 服务，/health 返回 200）
- 上述所有 internal/ 包的空文件
- 验证 GOOS=linux GOARCH=loong64 go build 通过
```

[图9-1：项目目录结构 — 请在此处插入 tree 命令输出截图]

## 9.2 安全护栏的 TDD 开发

安全模块是赛题核心考点。开发策略：**先写测试用例，再让 AI 实现**。

### 步骤 1：定义测试期望

```go
// 先告诉 AI 这些测试必须通过
func TestDangerousCommands(t *testing.T) {
    cases := []struct{cmd string; shouldBlock bool}{
        {"rm -rf /", true},
        {"cat /etc/shadow", true},
        {":(){ :|:& };:", true},  // fork bomb
        {"df -h", false},
        {"ps aux", false},
    }
}
```

### 步骤 2：让 AI 实现使测试通过

```
实现 internal/safety/validator.go，使以下测试全部通过：
[粘贴测试代码]

设计要求：五层校验流水线（白名单→路径检查→参数检查→正则模式→注入检测）
```

### 步骤 3：验证覆盖率

```bash
go test ./internal/safety/ -coverprofile=cover.out
go tool cover -func=cover.out | tail -1
# 输出: total: 96.4%
```

[图9-2：测试覆盖率报告 — 请在此处插入截图]

## 9.3 Agent Loop 架构

Agent Loop 是整个系统的核心调度器。参考 OpenCode 的 `processGeneration` 模式：

```
用户输入
  → SENSE（注入检测，<1ms）
  → 构建消息历史
  → for 循环 {
      → 调 LLM（流式）
      → 收到 text → 推送前端
      → 收到 tool_calls → 执行工具 → 结果追加到历史 → 继续循环
      → 收到 stop → 结束
    }
  → OUTPUT（完成）
```

### 给 AI 的实现 Prompt

```
实现 internal/agent/loop.go，Agent tool-use 循环。
参考架构：opencode-go/internal/llm/agent/agent.go
关键特性：
1. 真正的 SSE 流式（逐 token 推送 text_delta 事件）
2. 工具并行执行（只读工具并行，写工具串行）
3. 写工具执行前触发权限确认（channel 阻塞）
4. 最多 10 轮工具调用
5. 工具输出超 4KB 自动截断
6. 五段式审计日志
```

[图9-3：Agent Loop 流程图 — 请在此处插入截图]

## 9.4 多 Agent 编排

复杂问题（如"系统为什么变慢了"）需要多维度分析。设计 Planner/Executor/Verifier 三角色：

```
Planner: 拆解子任务（2-5个维度）
  ↓
Executor: 并行执行各子任务（调用探针工具）
  ↓
Verifier: 验证信息是否足够回答问题
  ↓ 不够
Planner: 补充子任务（最多3轮）
  ↓ 够了
Synthesizer: 综合所有发现给出最终回答
```

[图9-4：多Agent协作时序图 — 请在此处插入截图]

---

# 第10章 前端开发实战

## 10.1 设计系统约定的固化

在写第一行前端代码之前，通过项目规则文件固定约定：

```markdown
# 前端开发约定（写入 Steering / Rules 文件）

1. 样式：全部 inline style，不用 CSS class
2. 图标：Material Symbols Outlined（<span className="material-symbols-outlined">）
3. 字体：CSS 变量 var(--ops-font-ui) / var(--ops-font-mono)
4. 颜色：CSS 变量 var(--ops-fg-primary) / var(--ops-status-danger) 等
5. 数据来源：每个组件顶部注释标注 // Data: GET /xxx
6. 零假数据：API 未就绪时显示 skeleton 或 --
```

**为什么用 inline style？**
- AI 生成的 CSS class 命名容易冲突
- inline style 自包含，组件可独立理解
- 不需要额外的 CSS 构建配置

## 10.2 从设计稿到组件

### 标准流程

1. 拿到设计稿（截图/Figma/Pencil）
2. 告诉 AI 全局约束（引用 Steering 文件）
3. 逐个组件实现，每个带上参考截图

### 给 AI 的 Prompt 示例

```
复原 ChatInput 组件，参考截图 [贴图]。
功能：
- 多行输入框（自适应高度）
- 底部工具栏：# 命令面板 / 附件 / 上下文圆环 / 模型选择 / 权限模式 / 发送
- / 开头显示命令面板下拉
- Enter 发送，Shift+Enter 换行
约束：inline style，Material Symbols 图标
```

[图10-1：ChatInput 组件效果 — 请在此处插入截图]

## 10.3 SSE 流式通信

前端需要处理后端推送的 Server-Sent Events：

```typescript
// useSSE hook 核心逻辑
const eventSource = new EventSource(url);
eventSource.addEventListener('text_delta', (e) => {
  // 逐 token 追加到消息内容
  dispatch({ type: 'APPEND_DELTA', delta: JSON.parse(e.data).delta });
});
eventSource.addEventListener('execute', (e) => {
  // 工具开始执行
  dispatch({ type: 'ADD_TOOL_CALL', toolCall: JSON.parse(e.data) });
});
```

### 事件类型完整列表

| 事件 | 触发时机 | 前端动作 |
|------|----------|----------|
| start | 请求开始 | 创建空消息 |
| sense | 注入检测完成 | 拦截则显示红色卡片 |
| text_delta | 每个 token | 追加到消息文本 |
| execute | 工具开始执行 | 显示工具卡片(loading) |
| execute_done | 工具执行完成 | 更新工具卡片状态 |
| output | 完整回复 | 最终化消息 |
| done | 结束 | 重置 streaming 状态 |

[图10-2：SSE 事件流时序 — 请在此处插入截图]

## 10.4 状态管理

使用 useReducer 模式（不引入 Redux/Zustand 等外部依赖）：

```typescript
type Action =
  | { type: 'ADD_MESSAGE'; sessionId: string; message: ChatMessage }
  | { type: 'APPEND_DELTA'; sessionId: string; delta: string }
  | { type: 'SET_STREAMING'; streaming: boolean }
  | { type: 'SET_CONTEXT_USAGE'; percent: number }
  // ...
```

**为什么不用 Redux？**
- 项目规模中等，useReducer 足够
- 减少外部依赖（部署更简单）
- AI 生成 reducer 代码比 Redux slice 更直观

---

# 第11章 系统集成与优化

## 11.1 前后端联调

### Steering 文件作为契约

前后端通过一份共享的 API 契约文件对齐：

```markdown
## API 状态表
| 端点 | 方法 | 状态 | 前端组件 |
|------|------|------|----------|
| /health | GET | 已就绪 | AppHeader 健康灯 |
| /api/v1/chat/stream | POST | 已就绪 | useSSE hook |
| /api/v1/models/pool | GET | 已就绪 | ModelSettings |
| /api/v1/terminal/exec | POST | 已就绪 | TerminalDrawer |
```

### 开发模式 vs 生产模式

| 模式 | 前端 | 后端 | 通信 |
|------|------|------|------|
| 开发 | Vite (5173) | Go (8080) | Vite proxy 转发 |
| 生产 | 构建为静态文件 | Go serve `./web/` | 同端口同源 |

[图11-1：开发/生产模式架构对比 — 请在此处插入截图]

## 11.2 AI 辅助代码审查

开发完功能后，切换到"审查模式"做自查：

```
审查以下文件，对照赛题评分标准：
- internal/safety/（安全护栏完整性）
- internal/agent/loop.go（Agent 循环健壮性）
- internal/audit/writer.go（审计链路完整性）

输出格式：阻塞问题 / 警告问题 / 建议
```

### 审查发现的实际问题（本项目真实案例）

| 发现 | 严重度 | 修复 |
|------|--------|------|
| 安全拦截没显示规则 ID | 警告 | 后端传 rule_id，前端展示 |
| allow_session 没真正缓存 | Bug | 修复 Respond 方法 |
| 多 Agent token 计数显示 0 | 警告 | verify() 返回 Usage |
| 权限拒绝没写审计日志 | 警告 | 写入 EXECUTE stage |

[图11-2：代码审查报告示例 — 请在此处插入截图]

## 11.3 MCP 协议集成

### 实现步骤

1. 引入 `mcp-go` SDK
2. 创建 MCP Client Manager
3. 启动时连接远程 MCP Server（如 Context7）
4. ListTools 获取工具列表
5. 注册为本地 Tool（统一接口）
6. LLM 自动感知并调用

### 给 AI 的 Prompt

```
引入 github.com/mark3labs/mcp-go，实现 MCP Client。
参考 opencode-go 的 mcp-tools.go。
支持三种传输：stdio / SSE / streamable HTTP。
预制 Context7（远程 https://mcp.context7.com/mcp）。
外部工具注册到同一个 ToolRegistry。
```

[图11-3：MCP 工具管理界面 — 请在此处插入截图]

---

# 第12章 部署与交付

## 12.1 交叉编译

Go 的交叉编译是零配置的：

```bash
# Linux amd64 (WSL2/云服务器)
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o server ./cmd/server

# Linux LoongArch64 (龙芯)
GOOS=linux GOARCH=loong64 CGO_ENABLED=0 go build -o server-loong64 ./cmd/server
```

关键：`CGO_ENABLED=0` 确保纯静态链接，部署时不需要 glibc 或任何 .so 文件。

## 12.2 前端构建

```bash
cd web && npx vite build
# 产出 web/dist/ 目录（约 1MB）
```

## 12.3 部署包结构

```
ops-agent-deploy/
  ops-agent          # 13MB 单二进制
  web/               # 前端静态文件
    index.html
    assets/
  data/              # 数据库目录（运行时创建）
  .env.example       # 配置模板
  install.sh         # 交互式安装脚本
  README.md          # 部署文档
```

## 12.4 一键部署脚本设计原则

好的部署脚本应该：
1. **交互式引导** — 不假设用户知道怎么填
2. **有默认值** — 能回车跳过的就跳过
3. **环境检查** — 启动前验证端口、网络、磁盘
4. **错误友好** — 出错给中文提示而非 stack trace
5. **幂等** — 重复运行不破坏已有数据

[图12-1：install.sh 运行效果 — 请在此处插入终端截图]

---

# 第四篇：企业实践

# 第13章 团队中的 AI 协作

## 13.1 代码所有权与责任

AI 生成的代码，**提交者负全责**。企业中的实践：

| 规则 | 原因 |
|------|------|
| AI 代码必须 Review 后才能合并 | 避免幻觉代码进入主分支 |
| commit message 标注 AI 辅助 | `feat: add auth [ai-assisted]` |
| 安全敏感代码必须人工审查 | 认证/授权/加密不信任 AI |
| 测试必须人能读懂 | AI 写的测试可能掩盖问题 |

## 13.2 Spec 的版本控制

Spec 文档和代码一起放在 Git 中：
- 需求变了 → 改 requirements.md → 改 tasks.md → 重新生成代码
- Design 变了 → PR 标注"架构变更"→ 需要 Tech Lead 审批

## 13.3 知识管理

Steering 文件 = 团队知识库：
- `coding-standards.md` — 编码规范
- `api-contract.md` — 接口契约
- `onboarding.md` — 新人指引

新成员入项目时，AI 自动加载这些文件，行为与老成员一致。

---

# 第14章 CI/CD 中的 AI 集成

## 14.1 PR 自动审查

```yaml
# .github/workflows/ai-review.yml
on: pull_request
jobs:
  review:
    steps:
      - uses: actions/checkout@v4
      - run: |
          # 获取 diff
          git diff origin/main...HEAD > diff.patch
          # AI 审查
          opencode review diff.patch --rules .opencode/review-rules.md
```

## 14.2 AI 生成测试的 CI 验证

```yaml
# 确保 AI 生成的测试真的能跑
jobs:
  test:
    steps:
      - run: go test ./... -race -count=1
      - run: npx tsc --noEmit
```

## 14.3 安全扫描

```yaml
jobs:
  security:
    steps:
      - run: gosec ./...
      - run: grep -r "api_key\|password\|secret" --include="*.go" | grep -v "_test.go" | grep -v ".env"
```

---

# 第15章 安全合规与伦理

## 15.1 数据安全

| 工具 | 代码发送到哪 | 风险等级 |
|------|-------------|----------|
| Copilot | GitHub/Azure 云 | 中（企业可配置不上传） |
| Cursor | 模型供应商 API | 中 |
| Claude Code | Anthropic API | 中 |
| OpenCode (本地模型) | 不发送 | 低 |
| Kiro | AWS | 中 |

**企业建议**：敏感项目使用本地模型（Ollama + DeepSeek/Qwen 开源版）

## 15.2 AI 幻觉风险

| 风险 | 防护措施 |
|------|----------|
| 调用不存在的 API | 编译验证（Go 静态类型） |
| 编造测试数据 | CI 跑测试（假数据会 fail） |
| 安全漏洞 | SAST 扫描 + 人工审查安全相关代码 |
| 版权代码混入 | 使用有 license filter 的模型 |

## 15.3 合规审计

AI 辅助开发的审计要求：
1. Git log 记录谁提交了什么（即使是 AI 写的）
2. PR Review 记录谁审查了什么
3. 安全敏感变更需要额外审批
4. 部署前的变更确认

---

# 附录

## A. 工具安装速查

| 工具 | 安装方式 |
|------|----------|
| Kiro | https://kiro.dev 下载安装 |
| Cursor | https://cursor.sh 下载安装 |
| Claude Code | `npm install -g @anthropic-ai/claude-code` |
| OpenCode | `go install github.com/opencode-ai/opencode@latest` |
| Copilot | VS Code/JetBrains 插件市场安装 |

## B. Prompt 模板速查

（见第6章详细列表）

## C. 推荐阅读

1. Spec-Driven Development Guide (augmentcode.com)
2. The Pragmatic Programmer, 2nd Edition
3. Clean Code (Robert C. Martin)
4. Anatomy of a Coding Agent (substack.com)
5. AI Agent Best Practices 2026 (medium.com)

## D. 本书配套项目

项目: OPS-Agent (Linux 运维智能体)
代码: 随书提供完整源码 + .kiro/specs/ 文档
技术栈: Go 1.25 + React 19 + TypeScript + SQLite
功能: 23个MCP工具、多Agent协作、安全护栏、实时终端、审计溯源

[图D-1：项目最终运行效果 — 请在此处插入截图]
