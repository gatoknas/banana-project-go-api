# Swagger/OpenAPI Correctness Fix — banana-project-go-api

## Goal

Make the served OpenAPI spec an accurate contract for the API: correct `/api/v1` paths,
documented Bearer auth/RBAC, and typed success responses. Regenerate all three spec
artifacts and add a CI guard so they cannot drift again.

## Audit findings (verified)

Runtime spec is served from `docs/docs.go` via `swag.ReadDoc("swagger")`
(`cmd/api/main.go:139`); `docs/swagger.json` and `docs/swagger.yaml` are committed but
unused at runtime. All three are currently consistent with each other.

- **Coverage: complete.** All 19 documented operations map to registered handlers,
  including `GET /categories`, `POST /email-receipts/sync`, `GET /email-receipts`.
  `GET /docs` and `GET /docs/swagger.json` are unannotated infra routes.
- **Gap 1 — path prefix.** Protected routes are mounted under `/api/v1/` via
  `http.StripPrefix("/api/v1", ...)` at `cmd/api/main.go:226`, but every protected
  `@Router` omits the prefix (e.g. `/users` at `internal/handlers/user.go:36`).
  14 operations affected across user/product/sale/category/email.
- **Gap 2 — no auth docs.** No `@securityDefinitions` / `@Security`; Bearer JWT and
  RBAC groups (`cmd/api/main.go:168-220`) are invisible.
- **Gap 3 — untyped success bodies.** `Create`/`Update`/`Delete`/`Sync` use
  `{object} map[string]interface{}`; the real `SyncResult{fetched,imported,skipped,errors}`
  (`internal/service/email.go:22`) is not in `definitions`.
- **Gap 4 — no drift guard.** No workflow regenerates or verifies the spec.

## Decisions

- Keep `@BasePath /` and use **absolute `/api/v1/...` paths** in protected `@Router`
  annotations. Public routes keep root paths; no runtime API change; single spec.
- Add a global `BearerAuth` apiKey security scheme; mark each protected operation
  with `@Security BearerAuth`.
- Introduce typed response DTOs in `internal/handlers` and use them in the encoders so
  code and spec cannot diverge.

## Tasks (ordered)

1. **Security scheme** — `cmd/api/main.go` doc block (above `func main`), add:
   ```
   // @securityDefinitions.apikey BearerAuth
   // @in header
   // @name Authorization
   // @description Type "Bearer " followed by the JWT from POST /login.
   ```
   (`cmd/` is excluded from the TDT gate.)

2. **Prefix protected `@Router` paths** (comment-only edits):
   - `internal/handlers/user.go`: `/users` → `/api/v1/users`; `/users/{id}` → `/api/v1/users/{id}` (lines 36, 69, 93, 131, 178)
   - `internal/handlers/product.go`: `/products` and `/products/{id}` → `/api/v1/...` (lines 36, 76, 102, 142, 195)
   - `internal/handlers/sale.go`: `/sales` → `/api/v1/sales` (line 35)
   - `internal/handlers/category.go`: `/categories` → `/api/v1/categories` (line 33)
   - `internal/handlers/email.go`: `/email-receipts`, `/email-receipts/sync` → `/api/v1/...` (lines 38, 80)
   - Leave `auth.go`, `health.go`, `hello.go` public paths unchanged.

3. **Add auth + role docs to each protected operation**: one `// @Security BearerAuth`
   per op, and append the required role to `@Description`
   (admin-only: users CRUD, products create/update/delete, email-receipts/sync;
   admin+salesperson: products list/get, categories list, sales create, email-receipts list).
   Source of truth is the middleware wiring at `cmd/api/main.go:168-220`.

4. **Typed response DTOs** — new `internal/handlers/responses.go`:
   ```go
   type MessageResponse struct {
       Status  string `json:"status"`
       Message string `json:"message"`
       ID      int64  `json:"id,omitempty"`
   }
   type SyncResponse struct {
       Status  string             `json:"status"`
       Message string             `json:"message"`
       Result  service.SyncResult `json:"result"`
   }
   ```
   - Use `MessageResponse` in the encoders of `user.go` Create/Update/Delete,
     `product.go` Create/Update/Delete, `sale.go` Create (`ID` = saleId).
   - Use `SyncResponse` in `email.go` Sync.
   - Update the corresponding `@Success` annotations to the DTOs
     (e.g. `@Success 200 {object} handlers.SyncResponse`).
   - JSON key set must stay identical (`status`, `message`, `id`, `result`).

5. **Regenerate specs** from repo root:
   ```
   go run github.com/swaggo/swag/cmd/swag init -g cmd/api/main.go -o docs
   ```
   `swag` is already a direct dependency (`go.mod:14`). Confirm only intended diffs in
   `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml`. Run twice; the second run
   must produce no diff.

6. **TDT tests** for the changed `internal/` files (mandate in `AGENTS.md`). Follow the
   existing pattern in `internal/handlers/category_test.go`: build a real service with a
   mock repository interface + `sqlmock.New()` where transactions are used; table-driven
   `tt` slices with `t.Run`, `wantErr`-style assertions.
   - New: `internal/handlers/user_test.go`, `sale_test.go`, `product_test.go`,
     `email_test.go`, `responses_test.go`.
   - `internal/handlers/category_test.go` already exists; verify coverage still ≥80% for `category.go`.
   - Each changed non-test file must reach ≥80% changed-file statement coverage.

7. **CI drift guard** — add a step/job (new `.github/workflows/swagger-check.yml`, or a
   step in `tdt-gate.yml`) that runs the regeneration command and fails on
   `git diff --exit-code docs/`.

8. **README consistency** — confirm the endpoint docs in `README.md:53-58` use `/api/v1/...`
   and add a one-line note that `/docs` serves the live spec.

## Files

- Edit: `cmd/api/main.go`, `internal/handlers/{user,product,sale,category,email}.go`, `README.md`
- Add: `internal/handlers/{responses.go,user_test.go,sale_test.go,product_test.go,email_test.go,responses_test.go}`
- Regenerated (do not hand-edit): `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml`
- Add: `.github/workflows/swagger-check.yml` (or edit `tdt-gate.yml`)

## Validation

- `go build ./...` and `go vet ./...`
- `go test ./...`
- `go run ./cmd/testgate -staged`
- Regeneration idempotency (run swag twice, second `git diff --exit-code docs/` clean)
- Start server, `GET /docs/swagger.json`: assert all 14 protected paths begin with
  `/api/v1/`, each has `security: [{BearerAuth: []}]`, and `definitions` contains
  `handlers.MessageResponse`, `handlers.SyncResponse`, `service.SyncResult`.
- Scalar UI at `/docs`: Try-it-out for `GET /api/v1/users` succeeds with a Bearer token
  and 401s without one.

## Risks / notes

- **Largest effort is the TDT gate**, not the annotations: `user.go`, `sale.go`,
  `product.go`, `email.go` have no sibling tests today, so each needs a new table-driven
  test file at ≥80% changed-file coverage or the gate blocks the commit. Do not bypass
  with `--no-verify`.
- Response-DTO refactor touches encode paths; verify the wire shape is byte-for-byte
  equivalent (add-aditive `id` with `omitempty` for Update/Delete).
- If `swag init` output drifts beyond intended changes, pin flags (e.g. `--parseInternal`)
  until the generated set matches the current baseline plus the documented changes.
- `/docs` and `/docs/swagger.json` remain undocumented — explicitly out of scope.

## Out of scope

- Documenting the `/docs` infra routes.
- Versioning public routes under `/api/v1` (rejected: breaking).
- Two-spec / multi-instance generation.
