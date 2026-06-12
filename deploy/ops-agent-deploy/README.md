# OPS-Agent 部署指南

Linux 运维智能体 v1.0.0

## 系统要求

- Linux amd64 (WSL2 / Ubuntu / CentOS / 麒麟)
- 网络连接（调用 LLM API）

## 快速部署（30秒）

```bash
# 1. 解压
tar -xzf ops-agent-deploy.tar.gz
cd ops-agent-deploy

# 2. 配置 API Key
cp .env.example .env
# 编辑 .env，填入 LLM_API_KEY

# 3. 启动
bash install.sh
```

浏览器访问 `http://localhost:8080`

## 配置说明

编辑 `.env` 文件:

```env
# LLM 配置（必填）
LLM_API_KEY=your-api-key
LLM_BASE_URL=https://api.deepseek.com
LLM_MODEL=deepseek-v4-flash

# 服务端口
PORT=8080

# 数据库路径
DB_PATH=./data/ops-agent.db

# JWT密钥（生产环境请改）
JWT_SECRET=your-secret-here
```

### 支持的 LLM 供应商

| 供应商 | Base URL | 模型 |
|--------|----------|------|
| DeepSeek | https://api.deepseek.com | deepseek-v4-flash / deepseek-v4-pro |
| 小米 MiMo | https://token-plan-cn.xiaomimimo.com/v1 | mimo-v2.5-pro / mimo-v2-flash |
| 通义千问 | https://dashscope.aliyuncs.com/compatible-mode/v1 | qwen3.6-plus |
| OpenAI | https://api.openai.com/v1 | gpt-5.5 |

## 文件结构

```
ops-agent-deploy/
  ops-agent          # 服务端二进制（单文件，无依赖）
  web/               # 前端静态文件
  data/              # 数据库目录（自动创建）
  .env               # 配置文件
  install.sh         # 一键启动脚本
  providers.json.example  # 多模型配置模板
```

## 后台运行

```bash
nohup ./ops-agent > ops-agent.log 2>&1 &
echo $! > ops-agent.pid
```

停止:
```bash
kill $(cat ops-agent.pid)
```

## 功能概览

- 自然语言运维：对话式管理 Linux 服务器
- 21+ 内置工具：磁盘/内存/进程/网络/日志/服务探针
- 安全护栏：命令白名单 + 注入检测 + 权限确认
- 多 Agent 协作：复杂问题自动拆解并行分析
- MCP 协议：可扩展外部工具插件
- 推理链路溯源：完整审计日志
- 实时终端：Web 终端直接执行命令
