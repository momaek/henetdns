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

## Upgrading

The CLI never phones home — update checks only run when you ask:

```bash
henetdns version --check   # compare the current version with the latest GitHub release
henetdns upgrade           # download the latest release and replace this binary in place
```

`upgrade` verifies the release checksum before swapping the binary and is a no-op when already up to date (`--force` reinstalls anyway). Both commands support `--json`.

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

### Add Zone

Add a primary zone, as with "Add a new domain" on the he.net dashboard. If the zone already exists it is printed unchanged (with a note on stderr), so the command is safe to re-run.

```bash
henetdns zones add example.com --json
```

he.net warns that a new zone is not properly delegated until the registrar points the domain's nameservers at `ns1.he.net` through `ns5.he.net`; that warning is not treated as an error.

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

Create record if not exists (exact match by type, name, value, and priority for MX/SRV):

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

# SRV: value is "weight port target" with --priority, or the zone-file form
# "priority weight port target". Priority defaults to 0.
henetdns records upsert \
  --zone example.com \
  --type SRV \
  --name _imaps._tcp \
  --value "0 1 993 imap.example.net"

henetdns records upsert \
  --zone example.com \
  --type CAA \
  --name @ \
  --value '0 issue "letsencrypt.org"'
```

`--ttl` must be one of the values the he.net zone editor offers: 300, 900, 1800, 3600, 7200, 14400, 28800, 43200, 86400, 172800.

`--name` accepts a short name (`www`), a fully-qualified name (`www.example.com`), or `@` for the zone apex.

### Delete Record

Delete exact matching record:

```bash
henetdns records delete \
  --zone example.com \
  --type A \
  --name www \
  --value 192.168.1.1
```

For `TXT` records, `--value` matches with or without the surrounding double quotes shown in `records list` output. When no record matches, the error lists close matches (same name, other type/value).

## Supported Record Types

All types offered by the he.net zone editor:

A, AAAA, AFSDB, ALIAS, CAA, CNAME, HINFO, HTTPS, LOC, MX, NAPTR, NS, PTR, RP, SPF, SRV, SSHFP, SVCB, TXT

- `MX` and `SRV` take `--priority N --priority-set` (defaults: MX 10, SRV 0).
- Hostname values (CNAME, MX, NS, ALIAS, PTR, SRV target) match case-insensitively and ignore a trailing dot.
- Deleting an `NS` record at the zone apex is refused, since it would break delegation for the whole zone.

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
