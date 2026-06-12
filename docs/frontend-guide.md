# 前端开发指南

## 技术栈

- React 19 + TypeScript
- Vite 8 (构建工具)
- TailwindCSS v4 (样式)
- xterm.js (终端)
- Zustand (状态管理)
- react-markdown + remark-gfm (Markdown 渲染)
- lightweight-charts (图表)
- allotment (可调整面板)

## 项目结构

```
web/src/
├── App.tsx                    # 根组件：路由、认证状态、布局
├── main.tsx                   # 入口
├── index.css                  # 全局样式 + TailwindCSS
├── components/
│   ├── AppHeader.tsx          # 顶部导航栏
│   ├── ChatInput.tsx          # 对话输入框
│   ├── ChatMessage.tsx        # 消息渲染
│   ├── CodeBlock.tsx          # 代码块（语法高亮）
│   ├── DataPanel.tsx          # 数据面板
│   ├── HealthPanel.tsx        # 健康状态面板
│   ├── MonitorPanel.tsx       # 监控面板
│   ├── MultiAgentChat.tsx     # 多 Agent 对话视图
│   ├── PermissionBanner.tsx   # 权限确认横幅
│   ├── PlanApproval.tsx       # 计划审批
│   ├── ResourceStrip.tsx      # 顶部资源条
│   ├── RightPanel.tsx         # 右侧面板（推理链路/数据/健康）
│   ├── RiskPreviewModal.tsx   # 风险预览弹窗
│   ├── SessionList.tsx        # 会话列表
│   ├── SideNav.tsx            # 侧边导航
│   ├── StatusBar.tsx          # 状态栏
│   ├── StreamingText.tsx      # 流式文本渲染
│   ├── TerminalDrawer.tsx     # 终端抽屉
│   ├── ThinkingBlock.tsx      # 思考过程折叠块
│   ├── ToolCallCard.tsx       # 工具调用卡片
│   ├── CircuitOpenBanner.tsx  # 熔断提示
│   ├── DestructiveWarning.tsx # 危险操作警告
│   ├── FileTree.tsx           # 文件树
│   ├── OutputPersistedBadge.tsx # 输出截断标记
│   └── desktop/               # 桌面模式组件
│       ├── DesktopIcons.tsx
│       ├── Taskbar.tsx
│       ├── Window.tsx
│       └── apps/              # 桌面应用
│           ├── FileManagerApp.tsx
│           ├── LogApp.tsx
│           ├── MonitorApp.tsx
│           ├── NetworkApp.tsx
│           ├── ProbeApp.tsx
│           ├── ProcessApp.tsx
│           ├── SecurityApp.tsx
│           ├── ServiceApp.tsx
│           ├── TerminalApp.tsx
│           └── TrashApp.tsx
├── pages/
│   ├── LoginPage.tsx          # 登录页
│   ├── SettingsPage.tsx       # 设置页
│   ├── ModelSettings.tsx      # 模型管理
│   ├── PermissionSettings.tsx # 权限设置
│   ├── ToolsSettings.tsx      # 工具管理
│   ├── AuditPage.tsx          # 审计日志
│   └── DesktopMode.tsx        # 桌面模式
├── hooks/
│   ├── useSSE.ts              # SSE 流式对话 hook
│   ├── useHealth.ts           # 健康检查轮询
│   ├── useResourcePolling.ts  # 资源数据轮询（需 authToken）
│   └── useWindowManager.ts    # 桌面窗口管理
├── stores/
│   └── chatStore.ts           # Zustand 全局状态
├── lib/
│   ├── auth.ts                # 认证工具函数
│   └── demo.ts                # Demo 模式
└── types/
    └── api.ts                 # TypeScript 类型定义
```

## 认证对接

### authFetch

所有需要认证的 API 请求必须使用 `authFetch`（`web/src/lib/auth.ts`），不要直接用 `fetch`：

```typescript
import { authFetch } from '../lib/auth'

const res = await authFetch('/api/v1/tools')
const data = await res.json()
```

`authFetch` 自动：
- 从 localStorage 读取 token 并添加 `Authorization` 头
- 收到 401 时清除 token 并触发 `auth:expired` 事件
- 多个并发 401 只触发一次事件（防抖）

### 认证状态管理

App.tsx 监听 `auth:expired` 事件，token 过期时自动跳转登录页：

```typescript
useEffect(() => {
  const onAuthExpired = () => setAuthTokenState(null)
  window.addEventListener('auth:expired', onAuthExpired)
  return () => window.removeEventListener('auth:expired', onAuthExpired)
}, [])
```

### 登录流程

1. LoginPage 调用 `POST /api/v1/auth/login`
2. 从 `json.data.token` 获取 token（不是 `json.token`）
3. 调用 `setAuthToken(token)` 存储
4. App.tsx 检测到 authToken 变化，渲染主界面

## 资源轮询

`useResourcePolling` 每 30 秒轮询 5 个探针（disk/top/memory/process/network_connections）。

**关键：** 只在 `connected && authToken` 时轮询，未登录不发请求，避免 401 循环。

## SSE 对话

`useSSE` hook 处理流式对话：

```typescript
const { sendMessage, isStreaming, stopStream } = useSSE()
```

事件类型和处理见 [frontend-integration.md](frontend-integration.md) 第 3 节。

## 开发命令

```bash
cd web
npm install          # 安装依赖
npm run dev          # 启动开发服务器 (localhost:5173)
npm run build        # 构建生产版本
npx tsc --noEmit     # 类型检查
npm run lint         # ESLint 检查
```

## Vite 代理配置

开发时前端 5173 端口，后端 8080 端口。Vite 已配置代理：

```typescript
// vite.config.ts
server: {
  proxy: {
    '/api': 'http://localhost:8080',
    '/health': 'http://localhost:8080',
    '/version': 'http://localhost:8080',
  }
}
```
