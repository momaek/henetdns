---
name: henetdns-maintainer
description: Maintain the henetdns Go CLI for Hurricane Electric Hosted DNS. Use when an agent such as OpenClaw or Claude Code needs repo-specific guidance to inspect or change Cobra commands, login/session handling, HTML parsers, SQLite cache/storage code, record workflows, or tests in this repository.
---

# Henetdns Maintainer

## Overview

Use this skill to make targeted changes in `henetdns` without breaking the CLI, HE login flow, or cache behavior.
Keep the main workflow here and load [references/repo-map.md](references/repo-map.md) when the task spans multiple packages or touches HE-specific behavior.

## Start With The Right Context

- Read `README.md` or `README.zh-CN.md` first when the task changes CLI behavior, flags, or examples.
- Read [references/repo-map.md](references/repo-map.md) before editing multiple packages, changing request flow, or adding tests in an unfamiliar area.
- Prefer targeted reads over broad scans. In most tasks the relevant code is in one of: `internal/cli`, `internal/auth`, `internal/henet`, `internal/httpclient`, `internal/store`.

## Follow The Repo Workflow

1. Trace the full path before editing.
   For CLI tasks, start at `cmd/henetdns/main.go` and `internal/cli/*.go`.
   For service behavior, continue through `internal/app/runtime.go` into `internal/auth`, `internal/henet`, `internal/httpclient`, and `internal/store`.
2. Preserve existing contracts unless the user explicitly asks to change them.
   Keep list commands cache-first by default.
   Keep `--cache-only` and `--refresh` mutually exclusive.
   Keep record upsert/delete based on exact matching semantics, with MX priority included when set.
3. Validate at the package level first, then widen if needed.
   Run the smallest relevant `go test` target for parser/auth/http/store changes.
   Run `go test ./...` after cross-package or user-visible behavior changes.
4. Use local documentation instead of guessing remote behavior.
   If HE login or zone-page behavior is involved, read `docs/curl.md` before changing success markers or request order.

## Guardrails

- Do not treat login success as "response returned a new cookie". This repo intentionally verifies page markers after a bootstrap `GET` and login `POST`.
- Do not bypass cache semantics accidentally when touching `zones list` or `records list`.
- Do not change parser markers without updating the relevant parser/auth tests.
- Do not add broad abstractions unless they simplify a real workflow in this small codebase.

## Validation Checklist

- For auth, parser, store, or HTTP client changes, run the closest package tests first.
- For command-surface changes, run `go test ./...` and sanity-check the README examples you affected.
- If the task depends on real HE behavior, describe what was verified locally and what still depends on remote/manual validation.
