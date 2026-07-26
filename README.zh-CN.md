# henetdns

用于管理 Hurricane Electric Hosted DNS 的命令行工具。

[English README](README.md)

## 安装

```bash
go install github.com/momaek/henetdns/cmd/henetdns@latest
```

## 配置

可通过命令行参数或环境变量配置：

| 参数 | 环境变量 | 说明 |
|------|----------|------|
| `--base-url` | `HENETDNS_BASE_URL` | HE DNS 基础地址（默认：`https://dns.he.net`） |
| `--data-dir` | `HENETDNS_DATA_DIR` | 会话与缓存的数据目录（默认：`~/.config/henetdns`） |
| `--username` | `HE_USERNAME` 或 `HE_EMAIL` | 账号用户名 |
| `--password` | `HE_PASS` | 账号密码 |
| `--timeout` | `HENETDNS_TIMEOUT` | HTTP 超时时间（默认：`20s`） |

## 使用

### 登录

```bash
henetdns login --username your_username
# 如果未通过 --password 或 HE_PASS 提供密码，会交互提示输入
```

### 列出 Zone

```bash
henetdns zones list
henetdns zones list --json
henetdns zones list --cache-only
henetdns zones list --refresh
```

### 列出记录

```bash
henetdns records list --zone example.com
henetdns records list --zone 123456 --json
henetdns records list --zone example.com --cache-only
henetdns records list --zone example.com --refresh
```

### 新增记录（幂等 upsert）

若已存在完全匹配记录（类型、名称、值、MX 优先级），则不重复创建：

```bash
henetdns records upsert \
  --zone example.com \
  --type A \
  --name www \
  --value 192.168.1.1 \
  --ttl 300

henetdns records upsert \
  --zone example.com \
  --type MX \
  --name @ \
  --value mail.example.com \
  --priority 10 \
  --priority-set
```

`--name` 可以是短名（`www`）、完整域名（`www.example.com`），或用 `@` 表示 zone 根域名。

### 删除记录

删除完全匹配的记录：

```bash
henetdns records delete \
  --zone example.com \
  --type A \
  --name www \
  --value 192.168.1.1
```

`TXT` 记录的 `--value` 带不带 `records list` 输出中的外层双引号都可以匹配。未找到匹配记录时，错误信息会列出近似候选（同名的其他类型/值）方便排查。

## 缓存行为

- 默认 list 行为是“缓存优先”：先读本地 `cache.json`，缓存为空时再回源请求。
- `--cache-only` 仅读取本地缓存，不发起远端请求。
- `--refresh` 跳过本地缓存，强制回源并刷新缓存。
- `--cache-only` 与 `--refresh` 不能同时使用。

## 支持的记录类型

- A
- AAAA
- TXT
- CNAME
- MX

## AI Agent 集成

不需要 MCP server。任何能跑 shell 的 agent（Claude Code、OpenClaw 等）直接通过 CLI 驱动 henetdns——每个命令都支持 `--json` 输出机器可读结果。

```bash
henetdns login --username your_username   # 登录一次，session 写入 session.json
henetdns zones list --json
henetdns records list --zone example.com --json
henetdns records upsert --zone example.com --type A --name www --value 1.2.3.4 --json
```

仓库内置了一个开箱即用的 agent skill：[`skills/henetdns/`](skills/henetdns/SKILL.md)，把 agent 指向它即可，里面有全部命令、参数、JSON 结构和典型工作流。

## 数据存储

会话 Cookie 和缓存数据默认以 JSON 文件形式存储在 `~/.config/henetdns/`（`session.json` 与 `cache.json`）。
