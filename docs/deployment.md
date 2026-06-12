# 部署运维文档

## 本地开发

```bash
# 配置
cp .env.example .env
# 编辑 .env，填入 LLM_API_KEY 和 LLM_BASE_URL

# 后端
go run ./cmd/server/

# 前端
cd web && npm install && npm run dev
```

## 生产构建

### 本地平台

```bash
# 构建前端
cd web && npm run build && cd ..

# 构建后端
go build -ldflags "-s -w -X main.Version=1.0.0 -X main.GitCommit=$(git rev-parse --short HEAD) -X 'main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)'" -o ops-agent ./cmd/server/

# 运行
./ops-agent
```

### 龙芯 LoongArch64

```bash
bash deploy/build-loong64.sh [版本号]
# 产物: deploy/ops-agent-loong64-v1.0.0.tar.gz
```

构建脚本自动完成：前端构建 -> Go 交叉编译 (GOOS=linux GOARCH=loong64 CGO_ENABLED=0) -> Web 资源复制 -> 打包。

前提：Go 1.22+ 和 Node.js 18+。

## 部署方式

### 方式一：交互式安装（推荐）

```bash
tar xzf ops-agent-loong64-v1.0.0.tar.gz
cd ops-agent-deploy-loong64
bash install.sh
```

install.sh 自动完成：
1. 架构检查（loongarch64）
2. 依赖检查（df/ps/free/ss/ip 等）
3. LLM 供应商选择（DeepSeek/MiMo/Qwen/自定义）
4. API Key 和管理员密码配置
5. systemd 服务注册（root 用户）
6. 健康检查

### 方式二：手动部署

```bash
# 1. 复制文件
mkdir -p /opt/ops-agent/data
cp ops-agent /opt/ops-agent/
cp -r web /opt/ops-agent/
cp .env.example /opt/ops-agent/.env

# 2. 编辑配置
vi /opt/ops-agent/.env

# 3. 直接运行
cd /opt/ops-agent && ./ops-agent

# 4. 或注册 systemd 服务
cat > /etc/systemd/system/ops-agent.service << EOF
[Unit]
Description=OPS-Agent
After=network.target

[Service]
Type=simple
WorkingDirectory=/opt/ops-agent
ExecStart=/opt/ops-agent/ops-agent
Restart=on-failure
RestartSec=5
EnvironmentFile=/opt/ops-agent/.env

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable ops-agent
systemctl start ops-agent
```

## 配置说明

### .env 文件

```env
# LLM 配置（必填）
LLM_API_KEY=sk-your-key
LLM_BASE_URL=https://api.deepseek.com/v1
LLM_MODEL=deepseek-v4-flash

# 服务配置
PORT=8080
JWT_SECRET=随机生成的密钥
DB_PATH=./data/ops-agent.db
ADMIN_PASSWORD=admin123
```

**重要：** 值不要加引号。`LoadDotEnv` 会自动处理引号，但不加引号最安全。

### 多模型配置

复制 `providers.json.example` 为 `providers.json`，可配置多个 LLM 供应商。运行时通过 API 或前端切换，无需重启。

### JWT 认证模式

- `JWT_SECRET` 以 `dev-` 开头：开发模式，不强制 JWT 认证
- 其他值：生产模式，所有 API（除公开端点外）强制 JWT

## 运维操作

### 查看服务状态

```bash
systemctl status ops-agent
```

### 查看日志

```bash
journalctl -u ops-agent -f
```

### 重启服务

```bash
systemctl restart ops-agent
```

### 解锁 IP

登录失败 5 次后 IP 被锁定 3 分钟：

```bash
# 查看锁定列表
curl http://localhost:8080/api/v1/auth/lockouts

# 解锁指定 IP
curl -X DELETE http://localhost:8080/api/v1/auth/lockout/192.168.1.100
```

### 重置管理员密码

编辑 `.env` 文件中的 `ADMIN_PASSWORD`，然后重启服务。

### 健康检查

```bash
curl http://localhost:8080/health
curl http://localhost:8080/health/deep  # 含 LLM 连通性
```

### 卸载

```bash
bash uninstall.sh
# 或手动：
systemctl stop ops-agent
systemctl disable ops-agent
rm /etc/systemd/system/ops-agent.service
rm -rf /opt/ops-agent
```

## 反向隧道 (rtunnel)

内网服务器没有公网 IP 时，用 rtunnel 反向隧道暴露 SSH：

```bash
# 公网服务器（8.217.225.55）
rtunnel -mode server -tunnel-port 7000 -ssh-port 2222 -secret YOUR_SECRET

# 内网服务器
rtunnel -mode client -tunnel 8.217.225.55:7000 -forward 127.0.0.1:22 -secret YOUR_SECRET

# SSH 连接
ssh -p 2222 vmuser@8.217.225.55
```

特性：断线自动重连（3 秒），纯 Go 静态编译，零依赖。

## 常见问题

### 端口监听失败 `lookup tcp/"8080"`

原因：.env 中 PORT 值带了引号（`PORT="8080"`），Go 解析时引号被当成端口的一部分。

修复：确保 .env 中值不加引号（`PORT=8080`），或使用最新版本（LoadDotEnv 自动剥离引号）。

### 登录后 401 循环

原因：前端 `authFetch` 收到 401 时 `window.location.reload()`，导致无限刷新。

修复：使用最新版本，authFetch 改为事件派发，useResourcePolling 检查 authToken。

### 429 限速

默认 200 请求/分钟。前端资源轮询每 30 秒 5 个请求 = 10 req/min，不会触发限速。如需调整，修改 `cmd/server/main.go` 中的 `NewRateLimiter` 参数。

### 龙芯编译失败

确保 Go 版本 >= 1.22，且支持 loong64 交叉编译：`GOOS=linux GOARCH=loong64 go env` 不报错。
