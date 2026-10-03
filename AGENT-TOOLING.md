# Agent Tooling Layout — banana-project-go-api

This repository is used with three agent tools: **Antigravity**, **KiloCode**, and **OpenCode**.
The goal is one source of truth with thin, hand-maintained bridges per tool.

## TL;DR

- **Source of truth = `AGENTS.md` + `.agents/`.** All three tools read `AGENTS.md`, and all three discover `.agents/skills/`.
- **Bridges** are `.opencode/` (OpenCode) and `.kilo/` (KiloCode).
- When adding capability, prefer putting it under `.agents/` so it works everywhere; only add a bridge entry when a tool does not read the canonical location.

## Discovery matrix

| Capability | Antigravity (canonical) | OpenCode bridge | KiloCode bridge |
|---|---|---|---|
| Instructions / rules | `.agents/rules/*.md` (needs `trigger:` frontmatter) + `AGENTS.md` | `opencode.json` → `instructions: [".agents/rules/*.md"]` | `kilo.jsonc` → `instructions: [".agents/rules/*.md"]` |
| Skills | `.agents/skills/<name>/SKILL.md` | same dir (native compat) | same dir (native compat) |
| Subagents | `.agents/agents/<name>.md` | `.opencode/agents/<name>.md` | `.kilo/agents/<name>.md` |
| Hooks / plugins | `.agents/hooks.json` | `.opencode/plugins/*.ts` | `.kilo/plugins/*.ts` |
| MCP servers | `.agents/mcp_config.json` | `opencode.json` → `mcp` | `kilo.jsonc` → `mcp` |
| Slash command | skills auto-convert (`/tdt`) | skill tool / `@` | skill listed under `/` |

## Canonical assets (`.agents/`)

- `skills/go-master-api` — Go architecture, Zap logging, OpenAPI, mandatory TDT.
- `skills/tdt` — generate/refresh table-driven tests for the current diff and run the gate.
- `skills/go-memory-perf` — allocation/performance review.
- `skills/go-openapi-docs` — Swagger/OpenAPI annotation standard.
- `skills/secret-guard` — credential/secret audit procedures.
- `skills/github-project-manager` — GitHub Projects #6 lifecycle (issue → branch → PR).
- `rules/github-project-tracking.md` — mandatory issue-first/branch-first/PR-first workflow.
- `rules/secret-prevention.md` — never commit secrets; `.env.example` only.
- `agents/go-test-writer.md` — subagent that writes table-driven Go tests.
- `hooks.json` — Antigravity secret-guard (pre-command) + codegraph auto-sync (post-edit/stop).
- `mcp_config.json` — CodeGraph + GitHub MCP servers (Antigravity).

Rules under `.agents/rules/` **must** start with YAML frontmatter (`trigger:` + `description:`), or Antigravity silently discards them.

## Bridges

### OpenCode (`opencode.json`, `.opencode/`)
- `opencode.json` — `instructions` (bridges `.agents/rules`), `mcp` (CodeGraph + GitHub).
- `.opencode/agents/go-test-writer.md` — subagent (`mode: subagent`, test-only edits).
- `.opencode/plugins/auto-tests.ts` — on `session.idle`, runs `go test ./...` if Go files changed.
- `.opencode/plugins/secret-guard.ts` — blocks `git commit` bash calls when the secret scanner finds issues.

### KiloCode (`kilo.jsonc`, `.kilo/`)
- `kilo.jsonc` — `instructions` (bridges `.agents/rules`), `mcp` (CodeGraph + GitHub).
- `.kilo/agents/go-test-writer.md` — same subagent.
- `.kilo/plugins/auto-tests.ts`, `.kilo/plugins/secret-guard.ts` — same guardrails.

### Antigravity
- Reads `.agents/` natively; no bridge files required.
- Plugins live under `~/.gemini/.../plugins/` (not `.agents/plugins/`).

## The TDT gate (all tools)

1. Any change to non-test `.go` files must add/update the sibling `_test.go` (table-driven).
2. Run `go run ./cmd/testgate -staged` before finishing, or `go test ./...` from this repo.
3. Enforced everywhere: Lefthook pre-commit (`secret-scan` + `tdt-gate`), CI (`tdt-gate.yml`), the `auto-tests` plugins, and root `AGENTS.md` (delegate to `go-test-writer`).

## Environment

- `GITHUB_PERSONAL_ACCESS_TOKEN` — required by the GitHub MCP server.
  - Antigravity: `${GITHUB_PERSONAL_ACCESS_TOKEN}` in `.agents/mcp_config.json`.
  - OpenCode / Kilo: `{env:GITHUB_PERSONAL_ACCESS_TOKEN}` in their config.

## Maintenance rules

- **Skill:** add once under `.agents/skills/<name>/SKILL.md` (name must match the folder). All three tools pick it up.
- **Rule:** add `.agents/rules/<name>.md` with `trigger:` frontmatter. It is auto-bridged via the `instructions` glob; no per-tool edit needed.
- **Subagent:** add the same body in all three locations (`.agents/agents`, `.opencode/agents`, `.kilo/agents`). Prefer markdown over JSON to avoid duplication.
- **MCP server:** add to `.agents/mcp_config.json` (Antigravity), `opencode.json` (OpenCode), and `kilo.jsonc` (Kilo).
- **Hook/plugin:** Antigravity in `.agents/hooks.json`; OpenCode in `.opencode/plugins/`; Kilo in `.kilo/plugins/`.
