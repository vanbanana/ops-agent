---
name: disk-cleanup
description: 磁盘空间清理的专业知识和最佳实践
---

# 磁盘清理 Skill

## 安全规则
- 永远不删除 /var/log 下正在被进程写入的文件（先用 lsof 检查）
- 优先清理: /tmp 超过 7 天的文件, /var/cache, *.gz 旧归档日志
- 清理前必须走风险预演流程（POST /safety/preview）
- 不能删除 .journal 文件，用 journalctl --vacuum-time 代替

## 推荐清理顺序（风险从低到高）
1. 包管理器缓存: `apt clean` / `yum clean all`
2. 旧日志归档: `find /var/log -name '*.gz' -mtime +30`
3. systemd journal: `journalctl --vacuum-time=7d`
4. Docker: `docker system prune -f`
5. /tmp 旧文件: `find /tmp -mtime +7 -delete`
6. 大文件排查: `find / -xdev -size +100M -exec ls -lh {} \;`

## 阈值判断
- 使用率 > 80%: ⚠️ 建议清理，给出建议
- 使用率 > 90%: 🔴 需要立即清理，列出可清理项
- 使用率 > 95%: 🚨 紧急，优先清理日志和缓存

## 常见大文件位置
- /var/log/messages, /var/log/syslog
- /var/lib/docker/overlay2/
- /var/cache/apt/ 或 /var/cache/yum/
- 用户 home 下的 .cache/
- core dump 文件: /var/crash/, /tmp/core.*
