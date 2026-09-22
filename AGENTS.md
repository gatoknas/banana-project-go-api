# Project Instructions

## Table-Driven Tests (TDT)

Any change to Go source files under `internal/` or `cmd/` must add or update the sibling `_test.go` file using the **table-driven test** pattern (a slice of anonymous structs named `tt`/`tests`, subtests via `t.Run(tt.name, ...)`, `wantErr` assertions, and mocked interfaces).

Before reporting a task complete, run the TDT gate:

```bash
go run ./cmd/testgate -staged
```

Do **not** bypass the gate with `git commit --no-verify`.

See `.kilo/skills/go-master-api/SKILL.md` for the full TDT mandate and `.kilo/command/tdt.md` for the one-keystroke test-generation flow.
