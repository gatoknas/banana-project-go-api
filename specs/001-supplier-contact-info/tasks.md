# Tasks: Supplier Multi-Value Contact Info

Ordered, dependency-aware task list. Each group maps to one GitHub issue and one branch.

Legend: `[ ]` pending, `[x]` done.

## Issue A — `repo:api` (branch `<N>-supplier-contact-info-api`)

- [ ] A1 `db/migrations/0006_supplier_contact_info.sql`: widen `phone` to `VARCHAR(255)`,
      add idempotent `address VARCHAR(255) NULL`.
- [ ] A2 Update `db/schema.sql` suppliers block (`phone VARCHAR(255)`, `address`).
- [ ] A3 `internal/models/inventory.go`: add `Address *string`.
- [ ] A4 `internal/service/supplier.go`: add `Address` to `SupplierRequest`; trim/blank→nil
      in `CreateSupplier` and `UpdateSupplier`.
- [ ] A5 `internal/repository/supplier.go`: add `address` to INSERT, SELECTs, Scan, UPDATE.
- [ ] A6 `internal/handlers/supplier.go`: update swagger descriptions.
- [ ] A7 Regenerate `docs/docs.go` (`swag init`).
- [ ] A8 TDT tests for service + repository (success and edge paths).
- [ ] A9 Commit `specs/001-supplier-contact-info/spec.md` (and plan/tasks) on this branch.
- [ ] A10 `go test ./...` and `go run ./cmd/testgate -staged` pass.

## Issue B — `repo:web` (branch `<N>-supplier-contact-info-web`, depends on A)

- [ ] B1 `src/types.ts`: add `address` to `Supplier` and `SupplierRequest`.
- [ ] B2 `SuppliersView.vue`: `initialFormState`, `editSupplier`, `saveSupplier` include
      `address`.
- [ ] B3 Form: address textarea + comma-separated guidance on phone/address.
- [ ] B4 Grid card + list table render `address`.
- [ ] B5 `filteredSuppliers` matches `address`.
- [ ] B6 `npm run lint`, `npm run build`, `secret_scanner.py` pass.

## Issue C — `repo:bruno` (branch `<N>-supplier-contact-info-bruno`, depends on A)

- [ ] C1 `create_supplier.bru`: add `address`, multi-value phone example, update `docs`.
- [ ] C2 `update_supplier.bru`: same.
- [ ] C3 Sync check vs regenerated OpenAPI: no drift/orphan.

## Cross-cutting

- [ ] D1 Gate 1: create issues, add to Project #6 `Ready`, present URLs, await approval.
- [ ] D2 Gate 2: branch from latest `main`, move cards to `In Progress`.
- [ ] D3 Gate 3: PRs with `Closes #<N>`, evidence, move cards to `In Review`.
