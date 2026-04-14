# henetdns

CLI tool for Hurricane Electric hosted DNS management.

[中文文档](README.zh-CN.md)

## Installation

```bash
go install github.com/momaek/henetdns/cmd/henetdns@latest
```

## Configuration

Configure via command-line flags or environment variables:

| Flag | Environment | Description |
|------|-------------|-------------|
| `--base-url` | `HENETDNS_BASE_URL` | HE DNS base URL (default: `https://dns.he.net`) |
| `--db-path` | `HENETDNS_DB_PATH` | SQLite db path (default: `~/.config/henetdns/client.db`) |
| `--username` | `HE_USERNAME` or `HE_EMAIL` | Account username |
| `--password` | `HE_PASS` | Account password |
| `--timeout` | `HENETDNS_TIMEOUT` | HTTP timeout (default: `20s`) |

## Usage

### Login

```bash
henetdns login --username your_username
# Password will be prompted if not provided via --password or HE_PASS
```

### List Zones

```bash
henetdns zones list
henetdns zones list --json
henetdns zones list --cache-only
henetdns zones list --refresh
```

### List Records

```bash
henetdns records list --zone example.com
henetdns records list --zone 123456 --json
henetdns records list --zone example.com --cache-only
henetdns records list --zone example.com --refresh
```

### Cache Behavior

- Default list behavior is cache-first. It reads local SQLite cache first, then falls back to remote fetch when cache is empty.
- `--cache-only` reads only local cache and never sends remote requests.
- `--refresh` bypasses local cache, always fetches from remote, and refreshes cache.
- `--cache-only` and `--refresh` cannot be used together.

### Upsert Record

Create record if not exists (exact match by type, name, value, and priority for MX):

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

### Delete Record

Delete exact matching record:

```bash
henetdns records delete \
  --zone example.com \
  --type A \
  --name www \
  --value 192.168.1.1
```

## Supported Record Types

- A
- AAAA
- TXT
- CNAME
- MX

## MCP Server

henetdns can run as a [Model Context Protocol (MCP)](https://modelcontextprotocol.io) stdio server, exposing DNS management as tools for AI agents (e.g. Claude Desktop).

### Setup

**1. Login once via CLI:**

```bash
henetdns login --username your_username
```

The session cookie is saved to SQLite. The MCP server reuses it automatically — credentials never enter the MCP layer.

**2. Start the server:**

```bash
henetdns mcp serve
```

### Claude Desktop Configuration

Add to `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

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

### Available Tools

| Tool | Description |
|------|-------------|
| `list_zones` | List all DNS zones. Uses cache by default; `refresh: true` fetches from HE.net. |
| `list_records` | List records for a zone (by name or ID). Cache-first; supports `refresh`. |
| `upsert_record` | Create a record if it doesn't already exist (idempotent). |
| `delete_record` | Delete an exact matching record. |

If the session expires, tools return: `"No active session. Run 'henetdns login' to authenticate, then retry."` — re-run `henetdns login` and the server resumes without restart.

## Data Storage

Session cookies and cached data are stored in SQLite at `~/.config/henetdns/client.db` by default.
