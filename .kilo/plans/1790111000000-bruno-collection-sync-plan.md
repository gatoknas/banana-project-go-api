# Bruno Collection Sync Plan — banana-project-bruno-collection

## Goal

Bring the Bruno collection in sync with the Go API's current surface (19 operations),
remove committed secrets, and add a drift guard so it cannot silently fall behind again.

## Context / findings (verified)

- Collection path (confirmed): `C:\Users\danie\Documents\SourceCode\banana-project\banana-project-bruno-collection`.
  It is **not a git repo** (no `.git`); it is a plain folder of 13 `.bru` requests.
  No CI or script compares it to the API today.
- IDE workspace: `C:\Users\danie\Documents\SourceCode\banana-project\banana-project.code-workspace`
  (multi-root) lists `go-api`, `web`, `mobile` only — the collection is missing, so it is not part of the
  project workspace the IDE/Kilo indexes. The current session root is `banana-project-go-api` alone.
- The Go API (`banana-project-go-api/cmd/api/main.go`) exposes 19 operations.
- **Missing 6 requests:** `POST /refresh`, `GET /api/v1/products/{id}`,
  `PUT /api/v1/products/{id}`, `DELETE /api/v1/products/{id}`,
  `POST /api/v1/email-receipts/sync`, `GET /api/v1/email-receipts`.
- **Stale/broken:**
  - Hardcoded expired JWTs in `get_products.bru:18`, `create_sale.bru:18`, `create_user.bru:18`.
  - `login.bru` stores only `token`, never `refreshToken`.
  - Duplicate `seq: 14` in `delete_user.bru` and `get_categories.bru`; `bruno.json:10` says `filesCount: 7` (stale).
  - Hardcoded `/users/1` and `/users/2` instead of variables.
- The OpenAPI spec at `banana-project-go-api/docs/swagger.json` is the source of truth
  (and now has its own CI drift guard).

## Decisions

- Add the 6 missing requests, hand-authored to match existing `.bru` conventions.
- Drift guard = **PowerShell 7 script inside the collection**
  (`scripts/check-sync.ps1`), reading `../banana-project-go-api/docs/swagger.json`.
  No cross-repo build coupling; no CI wiring (the collection isn't a repo).
- No Go source changes → the `banana-project-go-api` TDT gate does **not** apply.

## Tasks

### 0. Add the collection to the IDE workspace (do first)
- Edit `C:\Users\danie\Documents\SourceCode\banana-project\banana-project.code-workspace` and append to its
  `folders` array: `{ "name": "bruno", "path": "banana-project-bruno-collection" }`.
- Keep the path relative, matching the existing `go-api`/`web`/`mobile` entries (siblings of the file).
- No `tasks`/`launch` changes are required.
- The user reloads `banana-project.code-workspace` in the IDE so the collection becomes a workspace root
  (visible, searchable, and writable by the agent for the work below).

### 1. Fix login + secret cleanup (existing files)
- `login.bru` `script:post-response`: also `bru.setVar("refreshToken", res.body.refreshToken);`.
- `get_products.bru`, `create_sale.bru`, `create_user.bru`: delete the `auth:bearer { token: eyJ... }`
  block and set `auth: inherit`; keep `Authorization: Bearer {{token}}`.
- `get_user.bru`, `update_user.bru`, `delete_user.bru`: replace hardcoded IDs with `{{user_id}}`.
- `environments/Local.bru` and `environments/production.bru`: add `user_id: 1` and `product_id: 1`.

### 2. Add 6 requests at collection root
Follow the shape of `list_users.bru` (auth: inherit + `Authorization: Bearer {{token}}`):

| File | meta.name | Method + URL | Body |
| :--- | :--- | :--- | :--- |
| `refresh.bru` | Refresh Token | `POST {{base_url}}/refresh`, `auth: none` | `{"refreshToken":"{{refreshToken}}"}`; post-response refreshes `token`/`refreshToken` if present |
| `get_product.bru` | Get Product | `GET {{base_url}}/api/v1/products/{{product_id}}` | none |
| `update_product.bru` | Update Product | `PUT {{base_url}}/api/v1/products/{{product_id}}` | same fields as `create_product.bru` (`name`, `description`, `categoryId`, `unitOfMeasureId`, `sellPrice`, `averageCost`, `isForSale`, `requiresRecipe`) |
| `delete_product.bru` | Delete Product | `DELETE {{base_url}}/api/v1/products/{{product_id}}` | none |
| `sync_email_receipts.bru` | Sync Email Receipts | `POST {{base_url}}/api/v1/email-receipts/sync` | `{"from":"2026-09-01","to":"2026-09-21"}` |
| `list_email_receipts.bru` | List Email Receipts | `GET {{base_url}}/api/v1/email-receipts?from=2026-09-01&to=2026-09-21` | none |

### 3. Normalize ordering / metadata
- Renumber `seq` cleanly (removes duplicate 14): Status 1, Hello 2, Login 3, Refresh 4,
  Create User 5, List Users 6, Get User 7, Update User 8, Delete User 9, Get Categories 10,
  Create Product 11, Get Products 12, Get Product 13, Update Product 14, Delete Product 15,
  Create Sale 16, Sync Email Receipts 17, List Email Receipts 18, PROD Hello 19.
- `bruno.json`: set `filesCount` to 19.

### 4. Drift guard — `scripts/check-sync.ps1`
- Params: `-SpecPath` (default `$PSScriptRoot/../../banana-project-go-api/docs/swagger.json`),
  `-CollectionRoot` (default `$PSScriptRoot/..`).
- Parse the spec: for each `paths.*` and each of `get/post/put/delete/patch`, build `METHOD <normalizedPath>`.
- Parse each root `*.bru` (skip `collection.bru`; skip requests whose `url` is not `{{base_url}}*`,
  e.g. `PROD Hello.bru`): read the first `get|post|put|delete|patch {` block, take its `url`, strip
  `{{base_url}}` and any `?...` query.
- Normalize both sides to a common form: `{{param}}` and `{param}` → `{}`; numeric path segments → `{}`.
  (Apply the `{{...}}` rule before the `{...}` rule.)
- Print `MISSING` (in spec, no request) and `ORPHAN` (request, not in spec) lists; exit 1 if either is non-empty.
- Example output on success: `OK: 19/19 operations covered`.

### 5. Document maintenance — `banana-project-bruno-collection/README.md`
- How to import into Bruno, pick `Local`/`production`, run **Login** (captures `token` + `refreshToken`),
  then any request.
- State the contract: `docs/swagger.json` is the source of truth; after adding an API endpoint, add the
  matching `.bru` and run `pwsh -File scripts/check-sync.ps1`.
- Note email endpoints need Gmail env vars on the API, and any authenticated call needs Postgres running.

## Files

- Edit: `C:\Users\danie\Documents\SourceCode\banana-project\banana-project.code-workspace` (add `bruno` folder entry)
- Edit: `banana-project-bruno-collection/{login,get_products,create_sale,create_user,get_user,update_user,delete_user}.bru`, `bruno.json`, `environments/Local.bru`, `environments/production.bru`
- Add: `banana-project-bruno-collection/{refresh,get_product,update_product,delete_product,sync_email_receipts,list_email_receipts}.bru`
- Add: `banana-project-bruno-collection/scripts/check-sync.ps1`, `banana-project-bruno-collection/README.md`

## Validation

- After the workspace edit and IDE reload: 4 roots are listed (go-api, web, mobile, bruno) and
  `banana-project-bruno-collection/bruno.json` is openable/editable from the workspace.
- `pwsh -File scripts/check-sync.ps1` → `OK: 19/19 operations covered`, exit 0.
- Negative check: temporarily remove one `.bru` (or point `-SpecPath` at a copy with an extra path) and
  confirm the script reports `MISSING`/`ORPHAN` and exits 1.
- `Select-String -Path *.bru -Pattern 'eyJ'` returns no matches (no committed JWTs).
- In Bruno: with the API running (`go run ./cmd/api` from `banana-project-go-api`) and Postgres up,
  run Status → Login → Refresh → List Users → Get Categories → Create/Get/Update/Delete Product →
  Create Sale. Email requests are expected to 400/500 when Gmail creds are absent.
- Diff `bruno.json` `filesCount` against the actual count (19).

## Risks / notes

- `banana-project.code-workspace` and the collection both live in the parent folder, which the parent
  `AGENTS.md` states is **not a git repo**. Neither edit is version-controlled, so back up / version the
  workspace file if that matters; the IDE reload is required before the collection is a workspace root.
- Implementation edits paths **outside** the `banana-project-go-api` workspace root, so it needs an agent
  with external-directory write permission (the parent workspace file and the collection folder).
- The drift checker is heuristic: path params are compared structurally (`{id}` ↔ `{{product_id}}` ↔ `1`),
  so it verifies path/method coverage, not request bodies or auth.
- Keep `PROD Hello.bru` (absolute URL) out of the checker's scope; it is intentional.
- `GET /hello/logo.png`, `GET /docs`, and `GET /docs/swagger.json` are intentionally not added.

## Out of scope

- Auto-generating `.bru` files from the spec.
- Adding CI for the collection (not a git repo).
- Documenting request/response bodies beyond what the existing requests already show.

## Next step

Requires an implementation-capable agent with write access to
`C:\Users\danie\Documents\SourceCode\banana-project\banana-project.code-workspace` and
`banana-project-bruno-collection/` (both outside this workspace root). Switch to Code/implementation mode;
after the workspace file is edited, the user must reload the `banana-project.code-workspace` in the IDE.
