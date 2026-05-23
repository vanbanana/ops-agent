# OPS-Agent 项目记忆

此文件自动注入 system prompt。修改后重启服务生效。

## 目标服务器信息

- 当前开发测试环境：macOS (Apple M1)
- 生产目标：Linux (Ubuntu 22.04 / CentOS 7+)
- 关键服务：nginx, mysql, redis, docker

## 常用命令

- 查看 nginx 状态：`systemctl status nginx`
- 查看 MySQL 状态：`systemctl status mysql` 或 `systemctl status mysqld`
- Docker 容器列表：`docker ps -a`
- 实时日志：`journalctl -u <service> -f --no-pager -n 50`

## 用户偏好

- 报告格式：先结论后细节，表格展示关键指标
- 磁盘告警阈值：> 80% 警告，> 90% 危险
- 内存告警阈值：可用 < 500MB 警告
- 负载告警阈值：> CPU核数 × 2 警告

## 注意事项

- macOS 环境下部分 Linux 命令不可用（systemctl、journalctl），用 macOS 等价命令
- 工具探针已自动适配 macOS/Linux 差异，不需要手动区分
