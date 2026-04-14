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
| `--db-path` | `HENETDNS_DB_PATH` | SQLite 数据库路径（默认：`~/.config/henetdns/client.db`） |
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

### 删除记录

删除完全匹配的记录：

```bash
henetdns records delete \
  --zone example.com \
  --type A \
  --name www \
  --value 192.168.1.1
```

## 缓存行为

- 默认 list 行为是“缓存优先”：先读本地 SQLite 缓存，缓存为空时再回源请求。
- `--cache-only` 仅读取本地缓存，不发起远端请求。
- `--refresh` 跳过本地缓存，强制回源并刷新缓存。
- `--cache-only` 与 `--refresh` 不能同时使用。

## 支持的记录类型

- A
- AAAA
- TXT
- CNAME
- MX

## MCP Server

henetdns 支持以 [Model Context Protocol (MCP)](https://modelcontextprotocol.io) stdio server 模式运行，将 DNS 管理能力作为工具暴露给 AI Agent（如 Claude Desktop）。

### 配置步骤

**1. 通过 CLI 登录一次：**

```bash
henetdns login --username your_username
```

Session cookie 写入 SQLite，MCP server 自动复用，密码不会进入 MCP 层。

**2. 启动 server：**

```bash
henetdns mcp serve
```

### Claude Desktop 配置

添加到 `~/Library/Application Support/Claude/claude_desktop_config.json`（macOS）或 `%APPDATA%\Claude\claude_desktop_config.json`（Windows）：

```json
{
  "mcpServers": {
    "henetdns": {
      "command": "henetdns",
      "args": ["mcp", "serve"]
    }
  }
}
```

### 可用工具

| 工具 | 说明 |
|------|------|
| `list_zones` | 列出所有 DNS Zone。默认缓存优先，`refresh: true` 从 HE.net 拉取。 |
| `list_records` | 列出指定 Zone 的记录（支持 Zone 名称或 ID）。缓存优先，支持 `refresh`。 |
| `upsert_record` | 幂等创建记录，完全匹配时不重复创建。 |
| `delete_record` | 删除精确匹配的记录。 |

Session 过期时工具返回：`"No active session. Run 'henetdns login' to authenticate, then retry."` — 重新执行 `henetdns login` 即可，无需重启 server。

## 数据存储

会话 Cookie 和缓存数据默认存储在 `~/.config/henetdns/client.db`。
