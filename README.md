# henetdns

CLI tool for Hurricane Electric hosted DNS management.

## Installation

```bash
go install github.com/wentx/henetdns/cmd/henetdns@latest
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
henetdns login --username your@email.com
# Password will be prompted if not provided via --password or HE_PASS
```

### List Zones

```bash
henetdns zones list
henetdns zones list --json
```

### List Records

```bash
henetdns records list --zone example.com
henetdns records list --zone 123456 --json
```

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

## Data Storage

Session cookies and cached data are stored in SQLite at `~/.config/henetdns/client.db` by default.
