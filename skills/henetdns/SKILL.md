---
name: henetdns
description: Manage Hurricane Electric (he.net) hosted DNS from the command line with the henetdns CLI. Use when an agent needs to log in to he.net, list DNS zones, list records, or create/delete A/AAAA/TXT/CNAME/MX records. Drives the CLI directly over the shell — no MCP server or extra configuration required.
---

# henetdns

`henetdns` is a small CLI for managing DNS hosted on Hurricane Electric (https://dns.he.net).
Use it to log in once, then list zones, list records, and create or delete records.

Always pass `--json` so output is machine-readable.

## Installation (do this first)

Before running any command, make sure the `henetdns` binary is available. Check, and install only if missing:

```bash
command -v henetdns >/dev/null 2>&1 && henetdns --version
```

If that prints a version, skip to Authentication. Otherwise install with whichever is available:

**Option A — prebuilt binary (no Go toolchain needed):**

```bash
# Detects OS/arch, downloads the latest release, installs to ~/.local/bin
os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m); case "$arch" in x86_64) arch=amd64;; aarch64|arm64) arch=arm64;; esac
tag=$(curl -fsSL https://api.github.com/repos/momaek/henetdns/releases/latest | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -1)
ver=${tag#v}
mkdir -p ~/.local/bin
curl -fsSL "https://github.com/momaek/henetdns/releases/download/${tag}/henetdns_${ver}_${os}_${arch}.tar.gz" | tar -xz -C ~/.local/bin henetdns
export PATH="$HOME/.local/bin:$PATH"   # add to shell profile to persist
henetdns --version
```

**Option B — via Go (requires Go ≥ 1.24):**

```bash
go install github.com/momaek/henetdns/cmd/henetdns@latest
# binary lands in $(go env GOPATH)/bin — ensure that is on PATH
```

If `henetdns` is already installed but seems outdated (e.g. a documented flag or command is missing), run `henetdns upgrade --json` to self-update to the latest release, or `henetdns version --check --json` to compare first. There is no background update check.

A valid he.net account is required. Log in once per machine (see below); the session cookie is persisted and reused.

## Authentication

Log in before any zone/record command. The session is stored in `~/.config/henetdns/session.json` and reused automatically until it expires.

```bash
# Password from env (preferred for non-interactive agents):
HE_PASS='your_password' henetdns login --username your_username --json
```

- Username can also come from `HE_USERNAME` (or legacy `HE_EMAIL`).
- Without `HE_PASS` / `--password`, login prompts interactively for the password.
- If a later command fails with `authentication required` ("run login first" / "session expired"), re-run `login`.

## Commands

### List zones

```bash
henetdns zones list --json
```

Cache-first by default. Add `--refresh` to fetch live from he.net (needs a session), or `--cache-only` to never touch the network.

### List records in a zone

```bash
henetdns records list --zone example.com --json
```

`--zone` accepts a zone name (`example.com`) or numeric zone ID. Same `--refresh` / `--cache-only` flags as `zones list`.

### Create a record (idempotent upsert)

```bash
henetdns records upsert --zone example.com --type A --name www --value 1.2.3.4 --json
```

- Supported types: `A`, `AAAA`, `TXT`, `CNAME`, `MX`.
- `--name` accepts a short name (`www`), a fully-qualified name (`www.example.com`), or `@` for the zone apex — all equivalent.
- `--ttl` defaults to 300.
- Creates the record only if an identical one does not already exist, so it is safe to re-run.
- For `MX`, set priority with `--priority N --priority-set` (defaults to 10).

### Delete a record (exact match)

```bash
henetdns records delete --zone example.com --type A --name www --value 1.2.3.4 --json
```

All of `--type`, `--name`, `--value` must match an existing record exactly. `--name` accepts short or fully-qualified names (see upsert). For `TXT` records, pass `--value` with or without the surrounding double quotes shown in `records list` output — both match. For `MX`, also pass `--priority N --priority-set`. Locked records cannot be deleted. When no record matches, the error lists close matches (same name, other type/value) to help correct the command.

## Output Shapes (`--json`)

`zones list` → array of `{ "id", "name" }`.

`records list` → array of:

```json
{
  "zone_id": "1234567",
  "record_id": "98765432",
  "name": "www.example.com",
  "type": "A",
  "ttl": 300,
  "priority": 10,
  "value": "1.2.3.4",
  "dynamic": false,
  "locked": false
}
```

`login` / `upsert` / `delete` → `{ "message": "login ok" | "upsert ok" | "delete ok" }`.

## Typical Agent Workflow

1. `henetdns login` (once, if not already authenticated).
2. `henetdns zones list --json` to discover zones.
3. `henetdns records list --zone <zone> --json` to inspect current records.
4. `henetdns records upsert ...` or `records delete ...` to change them.
5. Re-run `records list --zone <zone> --refresh --json` to confirm the change landed.

## Notes & Guardrails

- Prefer the zone **name** over numeric IDs in commands; the CLI resolves it.
- `upsert` will not modify an existing record's value — it only creates a missing one. To change a value, `delete` the old record then `upsert` the new one.
- `--cache-only` and `--refresh` are mutually exclusive.
- Global flags: `--data-dir` (or `HENETDNS_DATA_DIR`) changes where session/cache live; `--timeout` sets the HTTP timeout.
- Errors are written to stderr and exit non-zero; check the exit code rather than parsing stderr.
