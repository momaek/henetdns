# henetdns

CLI tool for Hurricane Electric hosted DNS management.

[中文文档](README.zh-CN.md)

## Installation

Download the prebuilt binary for your platform from the [Releases page](https://github.com/momaek/henetdns/releases/latest) (no Go toolchain needed), or run:

```bash
# Detects OS/arch, downloads the latest release, installs to ~/.local/bin
os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m); case "$arch" in x86_64) arch=amd64;; aarch64|arm64) arch=arm64;; esac
tag=$(curl -fsSL https://api.github.com/repos/momaek/henetdns/releases/latest | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -1)
mkdir -p ~/.local/bin
curl -fsSL "https://github.com/momaek/henetdns/releases/download/${tag}/henetdns_${tag#v}_${os}_${arch}.tar.gz" | tar -xz -C ~/.local/bin henetdns
henetdns --version   # ensure ~/.local/bin is on PATH
```

On Windows, download the `.zip` for your architecture from the Releases page.

Alternatively, build from source with Go ≥ 1.24:

```bash
go install github.com/momaek/henetdns/cmd/henetdns@latest
```

## Configuration

Configure via command-line flags or environment variables:

| Flag | Environment | Description |
|------|-------------|-------------|
| `--base-url` | `HENETDNS_BASE_URL` | HE DNS base URL (default: `https://dns.he.net`) |
| `--data-dir` | `HENETDNS_DATA_DIR` | Data directory for session and cache (default: `~/.config/henetdns`) |
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

- Default list behavior is cache-first. It reads the local `cache.json` first, then falls back to remote fetch when cache is empty.
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

## AI Agent Integration

No MCP server is needed. Any shell-capable agent (Claude Code, OpenClaw, etc.) drives henetdns directly through the CLI — every command supports `--json` for machine-readable output.

```bash
henetdns login --username your_username   # once; session saved to session.json
henetdns zones list --json
henetdns records list --zone example.com --json
henetdns records upsert --zone example.com --type A --name www --value 1.2.3.4 --json
```

A ready-to-use agent skill lives in [`skills/henetdns/`](skills/henetdns/SKILL.md): point your agent at it and it knows the commands, flags, JSON shapes, and typical workflows.

## Data Storage

Session cookies and cached data are stored as JSON files under `~/.config/henetdns/` by default (`session.json` and `cache.json`).
