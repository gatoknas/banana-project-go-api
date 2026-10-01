# GitHub Issues to File (Gate 1)

Board: [Projects Board #6](https://github.com/users/gatoknas/projects/6).
Create each issue in its repo, apply the label, add to Project #6 with status `Ready`.
Present URLs and STOP for approval before any code changes.

---

## Issue A

- **Repo**: `gatoknas/banana-project-go-api`
- **Label**: `repo:api`
- **Title**: `feat(api): enlarge supplier.phone and add address for multi-value contact info`
- **Size**: M

**Body**

### Context
Suppliers can have more than one phone number and more than one address. The current
`suppliers.phone` column is `VARCHAR(20)` and there is no address column. Store multiple
values as a comma-separated list in a single field per attribute.

### Scope
- Migration `0006_supplier_contact_info.sql`: `phone` → `VARCHAR(255)`; add idempotent
  `address VARCHAR(255) NULL`.
- Sync `db/schema.sql`.
- Add `Address` to the model and to `SupplierRequest`; trim/blank→nil in create/update.
- Add `address` to all supplier repository SQL (INSERT/SELECT/Scan/UPDATE).
- Regenerate swagger `docs/docs.go`.
- Table-driven tests for the changed service and repository paths.

### Acceptance Criteria
- [ ] `phone` values longer than 20 chars persist and round-trip.
- [ ] `address` persists on create/update and is returned on get/list; omitted → `NULL`.
- [ ] Migration `0006` is idempotent on a DB at `0005`; `db/schema.sql` matches.
- [ ] `go test ./...` passes; TDT gate `go run ./cmd/testgate -staged` passes.

### Dependencies
None (parent of web + bruno work).

---

## Issue B

- **Repo**: `gatoknas/banana-project-web`
- **Label**: `repo:web`
- **Title**: `feat(web): support multiple phones & addresses in supplier form and views`
- **Size**: M

**Body**

### Context
Follow-up to the API change that adds `address` and widens `phone`. The web app does not
yet expose or display the address.

### Scope
- `src/types.ts`: add `address?: string | null` to `Supplier` and `SupplierRequest`.
- `SuppliersView.vue`: include `address` in `initialFormState`, `editSupplier`,
  `saveSupplier`; add an address textarea; add comma-separated guidance; render address in
  grid and list views; include it in the search filter.

### Acceptance Criteria
- [ ] Create/edit round-trips `address`.
- [ ] Grid and list views render `address` (or "No registrada").
- [ ] Search matches on `address`.
- [ ] `npm run lint` and `npm run build` pass with zero errors; `secret_scanner.py` passes.

### Dependencies
Depends on Issue A (API field must exist).

---

## Issue C

- **Repo**: `gatoknas/banana-project-bruno-collection`
- **Label**: `repo:bruno`
- **Title**: `test(bruno): update supplier requests for address and multi-value phone`
- **Size**: S

**Body**

### Context
The API supplier payload gains `address` and multi-value phone support; the Bruno
collection must stay in sync with the OpenAPI spec.

### Scope
- `create_supplier.bru` / `update_supplier.bru`: add `address`, use a multi-value phone
  example, update the `docs` blocks.

### Acceptance Criteria
- [ ] Both requests include `address` and match the OpenAPI schema.
- [ ] Collection sync check reports no drift/orphan.

### Dependencies
Depends on Issue A (docs regenerated).
