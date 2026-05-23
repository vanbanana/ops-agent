# 架构参考 — 从 OpenCode 学到的设计模式

> 基于 opencode (anomalyco/opencode) 源码分析，2026-05-23

## 1. 文件结构设计 (ops-agent 采纳方案)

```
ops-agent/
├── .ops-agent/                    # 项目级配置目录（等价于 .opencode/）
│   ├── config.json                # 主配置文件
│   ├── agents/                    # Agent 定义（Markdown frontmatter）
│   │   ├── build.md               # 默认主 Agent
│   │   ├── plan.md                # 只读分析 Agent
│   │   ├── planner.md             # 多Agent: 任务拆解
│   │   ├── executor.md            # 多Agent: 执行子任务
│   │   ├── verifier.md            # 多Agent: 验证结论
│   │   └── coordinator.md         # 多Agent: 协调汇总
│   ├── skills/                    # Skill 定义（SKILL.md）
│   │   ├── disk-cleanup/SKILL.md  # 磁盘清理专业知识
│   │   ├── nginx-ops/SKILL.md     # Nginx 运维知识
│   │   └── security-audit/SKILL.md
│   ├── tools/                     # 自定义工具（Go 插件或脚本）
│   │   └── custom-check.sh        # 用户自定义检查脚本
│   └── prompts/                   # 可复用 prompt 片段
│       ├── system.txt             # 主 system prompt
│       └── compaction.txt         # 上下文压缩 prompt
├── cmd/server/main.go
├── cmd/cli/main.go
├── internal/...
└── data/                          # 运行时数据（不入库）
    ├── sessions/                  # 会话持久化 (SQLite)
    └── ops-agent.db
```

## 2. Agent 配置格式 (Markdown frontmatter)

```markdown
---
name: planner
description: 拆解复杂运维任务为子任务列表
mode: subagent
model: mimo-v2.5-pro
temperature: 0.3
max_steps: 3
tools:
  probe_*: true
  write_*: false
  shell: false
---

你是运维分析的 Planner 角色。
收到用户问题后，将其拆解为 2-5 个独立的子任务。
每个子任务应该可以被单独执行和验证。

输出格式为 JSON 数组:
[{"id":"1","description":"...","tools":"probe_disk,probe_memory"}]
```

## 3. Skill 格式 (SKILL.md)

```markdown
---
name: disk-cleanup
description: 磁盘空间清理的专业知识和最佳实践
---

# 磁盘清理 Skill

## 安全规则
- 永远不删除 /var/log 下正在被写入的文件
- 优先清理: /tmp, /var/cache, *.gz 旧日志
- 清理前必须走风险预演流程

## 常用命令
- `find /var/log -name '*.gz' -mtime +30 -delete`
- `journalctl --vacuum-time=7d`
- `docker system prune -f`

## 阈值判断
- 使用率 > 80%: 建议清理
- 使用率 > 95%: 紧急清理
```

## 4. 主配置 config.json

```json
{
  "llm": {
    "primary": {
      "provider": "xiaomi",
      "base_url": "https://token-plan-cn.xiaomimimo.com/v1",
      "model": "mimo-v2.5-pro"
    },
    "fallback": [
      {"provider": "deepseek", "base_url": "https://api.deepseek.com/v1", "model": "deepseek-v4-flash"}
    ]
  },
  "agents": {
    "default": "build",
    "multi_agent": {
      "enabled": true,
      "roles": ["planner", "coordinator", "executor", "verifier"]
    }
  },
  "skills": {
    "auto_discover": true,
    "paths": [".ops-agent/skills"]
  },
  "mcp": {
    "servers": {}
  },
  "safety": {
    "command_whitelist_path": ".ops-agent/safety/whitelist.json",
    "injection_rules_path": ".ops-agent/safety/injection-rules.json"
  },
  "session": {
    "max_history_messages": 20,
    "compaction": {
      "auto": true,
      "threshold_ratio": 0.8
    }
  }
}
```

## 5. 上下文压缩设计 (学 OpenCode)

### 触发条件
- API 返回的 usage.total_tokens > model_context * 0.8

### 流程
1. 检测到溢出 → 创建 compaction 任务
2. 把完整对话历史发给 LLM（用 compaction agent）
3. LLM 生成结构化摘要
4. 用摘要替换旧消息，保留最近 2 轮原文
5. 旧工具输出标记为 `[已清除]`

### 压缩 Prompt 模板
```
请为以下对话生成结构化摘要，用于继续后续对话:

## 目标
- [一句话总结用户要解决什么问题]

## 已完成的操作
- [列出已执行的工具和关键结果]

## 当前状态
- [系统当前的关键指标]

## 未完成事项
- [还需要做什么]
```

## 6. 从 OpenCode 学到的关键设计决策

| 决策 | OpenCode 做法 | 我们采纳 |
|------|-------------|---------|
| Agent 定义 | Markdown frontmatter | ✅ 采纳 |
| Skill 发现 | 递归扫描 `**/SKILL.md` | ✅ 采纳 |
| Tool 注册 | 内置 + 自定义 (TypeScript) | 改为 Go plugin/脚本 |
| Session 持久化 | SQLite + migrations | ✅ 采纳 |
| Context compaction | LLM-based 摘要 + 保留最近 N 轮 | ✅ 采纳 |
| Tool 输出修剪 | 保护最近 40K，旧的标记 cleared | ✅ 采纳 |
| Max steps per agent | 可配置 | ✅ 采纳 |
| 多 Agent 通信 | Task tool (primary → subagent) | 改为 Coordinator 协调 |
| 权限系统 | per-agent tool permission | ✅ 采纳 |
