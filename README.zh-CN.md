# henetdns

用于管理 Hurricane Electric Hosted DNS 的命令行工具。

[English README](README.md)

## 安装

```bash
go install github.com/wentx/henetdns/cmd/henetdns@latest
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

## 数据存储

会话 Cookie 和缓存数据默认存储在 `~/.config/henetdns/client.db`。
