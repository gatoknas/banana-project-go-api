# Plan: Supplier Multi-Value Contact Info

Companion to `spec.md`. Describes the technical design, file-level changes, migration
strategy, verification, and rollback.

## 1. Architecture Overview

```
Web (Vue/TS)          API (Go)                         DB (PostgreSQL)
SuppliersView.vue  ->  handler -> service -> repository  ->  suppliers table
types.ts               swagger docs regenerated            migration 0006
```

- The web sends a single `address` string (comma-separated when multiple).
- The service trims `address`; blank becomes `nil`.
- The repository persists/reads the new `address` column and the enlarged `phone`.
- No new endpoint; existing `/api/v1/suppliers` contract gains one optional field.

## 2. Data Model & Migration

New file `db/migrations/0006_supplier_contact_info.sql` (idempotent; the runner executes
`*.sql` alphabetically with no tracking table):

```sql
-- Migration 0006: widen suppliers.phone and add suppliers.address.
-- Safe and idempotent migration for PostgreSQL.

ALTER TABLE suppliers ALTER COLUMN phone TYPE VARCHAR(255);
ALTER TABLE suppliers ADD COLUMN IF NOT EXISTS address VARCHAR(255) NULL;
```

Also update the canonical `db/schema.sql` suppliers block so a fresh database matches:

```sql
phone   VARCHAR(255) NOT NULL,
address VARCHAR(255) NULL,
```

**Rollback**: `ALTER TABLE suppliers DROP COLUMN IF EXISTS address;` and
`ALTER TABLE suppliers ALTER COLUMN phone TYPE VARCHAR(20);` (only if no value exceeds 20
chars). Rollback is manual and documented for operators.

## 3. Go API Changes

- `internal/models/inventory.go` — add to `Supplier`:
  `Address *string \`json:"address" db:"address"\``.
- `internal/service/supplier.go` — add `Address *string \`json:"address"\`` to
  `SupplierRequest`; trim and blank→nil in both `CreateSupplier` and `UpdateSupplier`;
  set `Address` on the `models.Supplier` built in each.
- `internal/repository/supplier.go` — add `address` to:
  - `Create` INSERT column/value lists,
  - `GetByID`, `GetByTaxID`, `List` SELECT column lists + `rows.Scan` args,
  - `Update` SET clause + `ExecContext` args.
- `internal/handlers/supplier.go` — swagger `@Description` mentions the address/multi-value
  behavior (no signature change).
- `docs/docs.go` — regenerate via `swag init` (generated artifact; excluded from the test
  gate).

## 4. Web Changes

- `src/types.ts` — add `address?: string | null` to `Supplier` and `SupplierRequest`.
- `src/views/SuppliersView.vue`:
  - `initialFormState()` gains `address: ''`.
  - `editSupplier()` copies `supplier.address || ''`.
  - `saveSupplier()` sends trimmed `address` or `null`.
  - Form: address `<textarea>`; phone/address placeholders indicate comma separation.
  - Grid card + list table: render `address` (or "No registrada").
  - `filteredSuppliers` includes `address` in the match.
- `src/services/supplierService.ts` — no change (payload type carries the field).

## 5. Bruno Changes

- `create_supplier.bru` / `update_supplier.bru`: add `"address"`, use a multi-value phone
  example, update the `docs` block.
- Run the collection sync check against the regenerated OpenAPI.

## 6. Verification

- **Go**: `go test ./...` from `banana-project-go-api`; TDT gate
  `go run ./cmd/testgate -staged`. New TDT tests for service create/update
  (address set + omitted) and repository (`go-sqlmock`) query/scan.
- **Web**: `npm run lint` and `npm run build`; `python scripts/secret_scanner.py`.
- **Bruno**: sync check reports no drift/orphan.

## 7. Risks

- `internal/database/migrate.go` is currently untracked on `main`; the migration relies on
  it. Confirm it is part of the baseline before branching.
- Both `banana-project-go-api` and `banana-project-web` `main` carry unrelated uncommitted
  or untracked work. Stage only task-scoped files.
- The PAT used by the GitHub MCP must have the `project` scope for Project #6.
- `phone`/`address` length limits are enforced by the database; a service-level check is
  deferred (see spec open questions).

## 8. Rollout

1. Merge API first (schema + field), then web, then Bruno.
2. Deploy API and confirm migration 0006 applied.
3. Deploy web.
