# Spec: Supplier Multi-Value Contact Info (Phones & Addresses)

- **Feature ID**: 001-supplier-contact-info
- **Status**: Draft (awaiting Gate 1 issue approval)
- **Repos**: `banana-project-go-api`, `banana-project-web`, `banana-project-bruno-collection`
- **Board**: GitHub Project #6 (`gatoknas`)

## 1. Problem

The supplier record stores a single phone number in a small `VARCHAR(20)` column and has
no address field. In practice some suppliers provide more than one phone number and more
than one address. Users want to record all of them without creating duplicate suppliers.

## 2. Goal

Allow a supplier to store multiple phone numbers and multiple addresses in a single
comma-separated field per attribute, and expose these through the API, the web app, and
the API test collection.

## 3. Non-Goals

- Normalizing phones/addresses into child tables.
- Validating or splitting individual phone/address entries on the backend.
- Any change to the mobile app (no supplier code exists there).
- Auto-migrating existing single values into structured lists.

## 4. Data Model

| Column | Before | After |
| --- | --- | --- |
| `suppliers.phone` | `VARCHAR(20) NOT NULL` | `VARCHAR(255) NOT NULL` |
| `suppliers.address` | _(does not exist)_ | `VARCHAR(255) NULL` |

Both fields store a comma-separated list when more than one value is present. `address`
is optional; `phone` remains required.

## 5. Functional Requirements

- **FR-1** — `phone` accepts up to 255 characters and may contain a comma-separated list
  of numbers. It remains required.
- **FR-2** — `address` is optional, accepts up to 255 characters, and may contain a
  comma-separated list of addresses.
- **FR-3** — The API completes a round trip for `address`: it is accepted on create/update
  and returned on get/list.
- **FR-4** — Backward compatibility: existing clients that do not send `address` continue
  to work; the stored value is `NULL`.
- **FR-5** — The migration is idempotent and re-runnable (the runner has no tracking
  table), and `db/schema.sql` reflects the final canonical schema.
- **FR-6** — The web create/edit form exposes `address`, guidance states that phone and
  address accept comma-separated values, and both grid and list views render the stored
  values.
- **FR-7** — The web client-side supplier search also matches on `address`.
- **FR-8** — Bruno supplier requests carry `address` and a multi-value example, and remain
  in sync with the generated OpenAPI specification.

## 6. User Scenarios

1. **Create with multiple contacts** — As an admin, I register a supplier with two phone
   numbers and two addresses in comma-separated form and see them persisted and rendered.
2. **Edit an existing supplier** — As an admin, I add a second address to a supplier that
   previously had none and the change is saved and displayed.
3. **Read without address** — As a salesperson, I list suppliers where `address` was never
   set; the API omits/`null`s it and the UI shows "No registrada".

## 7. Acceptance Criteria

- AC-1 (API): creating a supplier with a >20-char phone value succeeds and the value is
  returned unchanged by `GET /api/v1/suppliers/{id}`.
- AC-2 (API): creating/updating with `address` persists it; omitting it yields `NULL`.
- AC-3 (API): migration `0006` applies cleanly on a database already at `0005`, and
  re-running it is a no-op for `address` and safe for `phone`.
- AC-4 (API): `db/schema.sql` matches the migrated schema.
- AC-5 (API): new/updated Go source has table-driven tests covering a success and an
  edge/error path, and `go test ./...` passes.
- AC-6 (Web): the form, grid, list, and search all handle `address`; `npm run lint` and
  `npm run build` pass with zero errors.
- AC-7 (Bruno): create/update supplier requests include `address` and match the OpenAPI
  schema; no drift/orphan reported by the collection sync check.

## 8. Edge Cases

- `address` sent as empty string or whitespace → stored as `NULL`.
- `address` exactly 255 characters → stored; over 255 → rejected by the DB.
- Phone list longer than 255 characters → rejected by the DB.
- Legacy supplier rows → `address` is `NULL`, all existing behavior unchanged.

## 9. Open Questions

- None blocking. Optional follow-up: enforce a friendly max-length validation in the
  service layer instead of relying on the database constraint.
