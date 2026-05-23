# 接力提示词 — Agent 对话前端渲染链路重构

> 给下一轮 AI 对话的完整上下文。直接贴给 AI 即可无缝衔接。

## 项目概况

- **项目**: Linux 运维智能体 (ops-agent)
- **前端**: `/Users/ljk/Desktop/yunwei/ops-agent/web/` — Vite 8 + React 19 + TypeScript + TailwindCSS v4
- **后端**: `/Users/ljk/Desktop/yunwei/ops-agent/` — Go + chi + SQLite
- **LLM**: 小米 MiMo (mimo-v2.5-pro)，OpenAI 兼容协议，支持 stream:true
- **后端端口**: 8080，前端 Vite 代理到 8080
- **图标**: Material Symbols Outlined (CDN link 在 index.html)
- **无 UI 组件库**，全部 inline style 手写

## 当前状态

后端已实现：
- 真正的 LLM streaming (`ChatStream` 方法，OpenAI 兼容 SSE)
- Agent loop 发送 `text_delta` 事件（逐 token）
- 工具调用事件：`execute` (开始) → `execute_done` (完成)
- 多 Agent 事件：`agent_role`, `verifier_result`
- 安全拦截：`sense(blocked)` → `error` → `done`

前端刚完成**架构重构**：
- `chatStore.ts` — 消息按 session 索引存储 (`messagesBySession[sessionId]`)，切换会话不搬消息
- `useSSE.ts` — 支持多个并发 SSE 连接，每个 session 独立
- `App.tsx` — SSE 回调通过 `streamingForSessionRef` 写入正确 session

## 下一步要做的事

**重构 `ChatMessage` 组件** — 当前的消息渲染太简陋，需要对标 Kimi/GPT/assistant-ui 的产品级体验。

### 要实现的组件（P0，必须全做）

1. **ThinkingIndicator** — Agent 消息 content 为空且正在 streaming 时，显示脉冲三点动画 + "正在思考..."
2. **ThinkingBlock** — 当 SSE 有 `analyze`/`plan` 事件时，显示折叠块"思考已完成"（可展开看推理过程），像 Kimi 那样
3. **ToolCallCard** — 工具调用 3 状态卡片：
   - running: 旋转图标 + 工具名 + "执行中..."
   - done: ✓ 绿色 + 工具名 + 耗时 + 可展开结果
   - error: ✗ 红色 + 错误信息
4. **MessageActions** — 每条 agent 消息底部操作栏（hover 显示）：
   - 复制按钮（复制 Markdown 原文）
   - 重新生成按钮
   - 停止生成按钮（streaming 时显示）
5. **StreamingCursor** — 流式输出时末尾的闪烁竖线光标 `▊`
6. **CodeBlock** — 代码块增加：语法高亮 + 右上角复制按钮 + 语言标签
7. **SuggestedPrompts** — 空对话时显示 3-4 个快捷提问卡片

### 关键文件

| 文件 | 作用 |
|------|------|
| `web/src/components/ChatMessage.tsx` | **主要改造目标** — 当前只是简单渲染 content |
| `web/src/components/ToolCallCard.tsx` | 已存在但未接入 ChatMessage |
| `web/src/components/StreamingText.tsx` | 已存在（逐字动画，可能需要改为直接渲染 delta） |
| `web/src/stores/chatStore.ts` | 消息状态管理 — `getSessionMessages(state)` 获取当前会话消息 |
| `web/src/App.tsx` | SSE 事件处理 → dispatch 到 store |
| `web/src/types/api.ts` | ChatMessage 类型定义（含 toolCalls, error, isBlocked） |

### ChatMessage 当前数据结构

```typescript
interface ChatMessage {
  id: string
  role: 'user' | 'agent' | 'system'
  content: string              // Markdown 文本
  timestamp: string
  toolCalls?: ToolCallBlock[]  // 工具调用列表
  isBlocked?: boolean          // 安全拦截标记
  error?: SSEErrorData         // 错误信息
}

interface ToolCallBlock {
  tool: string
  args: Record<string, unknown>
  status: 'running' | 'done' | 'error'
  result_preview?: string
  elapsed_ms?: number
}
```

### 设计规范

- 色彩: `--ops-bg-canvas: #171717`, `--ops-bg-surface: #212121`, `--ops-fg-primary: #CCCCCC`
- 状态色: `--ops-status-ok: #009431`, `--ops-status-warn: #F5A214`, `--ops-status-danger: #E01C22`, `--ops-status-info: #3794FF`
- 字体: `--ops-font-mono: Menlo, Monaco, Consolas`, `--ops-font-ui: "Helvetica Neue", Arial`
- 图标: Material Symbols Outlined (`<span className="material-symbols-outlined">icon_name</span>`)
- 全部 inline style，不用 CSS class

### SSE 事件时序（agent 回复时前端收到的顺序）

```
start → sense(ok) → mode_decision → analyze → plan → execute → execute_done → [text_delta × N] → output → done
```

- `text_delta`: `{ delta: "一个", round: 1 }` — 逐 token
- `execute`: `{ tool: "probe_disk", args: {}, security_check: "PASSED" }` — 开始调工具
- `execute_done`: `{ tool: "probe_disk", status: "ok", result_preview: "/ 87%...", elapsed_ms: 23 }` — 完成
- `output`: `{ reply: "完整回复...", tokens_used: {...} }` — 最终完整文本（兜底）

### 参考产品

- **Kimi**: "思考已完成" 折叠块 + 工具调用卡片
- **assistant-ui**: 开源 React Agent Chat 库 (github.com/assistant-ui/assistant-ui)
- **GPT**: 消息操作栏（复制/重新生成）+ 停止按钮

### 注意事项

1. 不要引入新的 npm 依赖（除非必要且已确认），用手写组件
2. 项目已有 `react-markdown` + `remark-gfm` 用于 Markdown 渲染
3. 后端不需要改动 — 所有 SSE 事件已经在发送
4. 测试: 80 个测试全绿（73 单元 + 7 集成），改前端不影响
5. streaming 时 `state.isStreaming = true`，用这个控制"停止生成"按钮显示
