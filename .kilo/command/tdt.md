---
description: Generate or refresh table-driven tests for the current diff and run the TDT gate
agent: code
---

# /tdt — Table-Driven Test Flow

Run this whenever you have added or modified a Go handler, service, repository, function, helper, or endpoint.

## Steps

1. Load the `go-master-api` skill so the implementation and the test are generated together.
2. Inspect the current diff (`git diff --name-only`) and identify every changed non-test `.go` file under `internal/`.
3. For each changed file, create or update the sibling `<name>_test.go` using the table-driven pattern:
   - slice of anonymous structs (`name`, input, expected, `wantErr`),
   - loop with `t.Run(tt.name, ...)`,
   - assert `(err != nil) != tt.wantErr`,
   - mock dependencies via interfaces.
4. Run the gate:

```bash
go run ./cmd/testgate -staged
```

5. Report per-file results and any remaining failures. Fix tests or implementation before finishing.

## Coverage

The gate enforces an 80% statement coverage floor on every changed `.go` file (excluding `internal/models`, `cmd/`, `db/`, `docs/`, generated `*.pb.go`, and `vendor/`). Pass `-scope package` to apply the floor to the whole changed package instead. Files in packages that have no tests are not enforced until they are touched.
