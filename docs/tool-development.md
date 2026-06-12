# 工具开发指南

## Tool 接口

所有工具必须实现 `Tool` 接口（定义在 `internal/tools/tool.go`）：

```go
type Tool interface {
    Name() string
    Description() string
    Schema() map[string]any
    Type() ToolType
    Execute(args map[string]any) (string, error)
}
```

### ToolType

| 类型 | 说明 | 权限要求 |
|------|------|---------|
| `ToolTypeReadonly` | 只读探针 | 无需确认 |
| `ToolTypeWrite` | 写操作 | 需用户确认（风险预览） |
| `ToolTypeExternal` | 外部调用 | 视情况而定 |

## 新增工具步骤

### 1. 创建工具文件

在 `internal/tools/` 下创建新文件，如 `probe_docker.go`：

```go
package tools

type ProbeDocker struct{}

func (p *ProbeDocker) Name() string { return "probe_docker" }

func (p *ProbeDocker) Description() string {
    return "查看 Docker 容器列表和状态 (docker ps)"
}

func (p *ProbeDocker) Schema() map[string]any {
    return map[string]any{
        "type": "object",
        "properties": map[string]any{
            "all": map[string]any{
                "type":        "boolean",
                "description": "是否显示已停止的容器",
            },
        },
    }
}

func (p *ProbeDocker) Type() ToolType { return ToolTypeReadonly }

func (p *ProbeDocker) Execute(args map[string]any) (string, error) {
    all := args["all"].(bool)
    cmd := "docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'"
    if all {
        cmd = "docker ps -a --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'"
    }
    return runCommand(cmd)
}
```

### 2. 注册工具

在 `internal/tools/register.go` 的 `RegisterAll` 函数中添加：

```go
registry.Register(&ProbeDocker{})
```

### 3. 安全规则

如果工具执行命令，需要在 `internal/safety/rules.go` 中将命令加入白名单：

```go
var allowedCommands = []string{
    // 已有命令...
    "docker",    // 新增
}
```

### 4. 测试

在 `internal/tools/` 下创建测试文件 `probe_docker_test.go`：

```go
package tools

import "testing"

func TestProbeDocker_Name(t *testing.T) {
    p := &ProbeDocker{}
    if p.Name() != "probe_docker" {
        t.Errorf("expected probe_docker, got %s", p.Name())
    }
}

func TestProbeDocker_Type(t *testing.T) {
    p := &ProbeDocker{}
    if p.Type() != ToolTypeReadonly {
        t.Errorf("expected readonly, got %v", p.Type())
    }
}
```

## 写操作工具

写操作工具的 Type 为 `ToolTypeWrite`，执行前会自动触发风险预览流程：

1. Agent Loop 检测到写工具调用
2. 调用 `safety.Preview()` 生成风险预览
3. 前端显示风险预览弹窗
4. 用户确认后才执行

写操作工具应遵守以下安全规则：
- 禁止删除 `/etc/`、`/boot/`、`/sys/`、`/proc/`、`/usr/`、`/dev/` 下的文件
- 禁止 kill PID <= 1
- 禁止使用 SIGKILL
- 日志操作仅限 `/var/log/` 目录
- 服务操作须匹配 `*.service` 格式

## 工具输出截断

工具输出超过 30000 字符时自动截断，完整输出保存到 `data/outputs/` 目录。用户可通过 `read_tool_output` 工具查看完整输出。

## 多 Agent 工具

`multi_agent_analyze` 是特殊工具，触发 Planner -> Executor -> Verifier 协作流程。如需修改多 Agent 行为，编辑 `internal/agent/multi.go`。

## MCP 外部工具

通过 MCP 协议注册的外部工具不需要实现 Go 接口。在 `providers.json` 或 API 中配置 MCP 服务器即可：

```json
{
  "id": "context7",
  "name": "Context7",
  "transport": "sse",
  "url": "https://mcp.context7.com/mcp",
  "is_active": true
}
```

MCP 工具启动时自动发现并注册，与内置工具统一管理。
