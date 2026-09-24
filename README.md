# 🍌 Banana Project API

> **A robust, modern REST API powering the core operations of the Banana Project ecosystem.**

---

## 📖 Purpose

The **Banana Project API** serves as the central nervous system for managing commerce and inventory operations. It is designed to handle sales, products, and users seamlessly. By providing a clean and documented interface, this API ensures that client applications—from point-of-sale systems to management dashboards—can operate reliably and efficiently.

---

## 🚀 Technologies Used

This project is built using modern, standard, and high-performance technologies to ensure reliability and speed:

| Technology | Description |
| :--- | :--- |
| **Go (Golang)** | The core language, chosen for its exceptional concurrency support, performance, and simplicity. |
| **net/http** | Utilized for robust routing leveraging the standard `net/http` package. |
| **PostgreSQL & pgx** | A powerful object-relational database paired with a pure Go driver for high performance. |
| **Zap** | The industry standard for high-performance, structured logging in Go. |
| **Lipgloss** | The Go community standard that allows you to style terminal outputs exactly like CSS. |
| **JWT** | Secure, stateless JSON Web Tokens for user authentication and authorization. |
| **Swaggo & Scalar** | Automated OpenAPI specification generation directly from source code (e.g., via `swag init`), served beautifully via Scalar UI. |

---

## 📚 API Documentation

The API includes built-in, interactive documentation. Once the server is running, you can explore all endpoints, expected payloads, and responses by visiting the `/docs` route in your browser. The live spec is served from `/docs/swagger.json`.

All `/api/v1/*` routes require a JWT sent as `Authorization: Bearer <token>`; the spec marks them with the `BearerAuth` security scheme. The generated spec is verified in CI (`.github/workflows/swagger-check.yml`) and can be regenerated with:

```bash
go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g cmd/api/main.go -o docs
```

---

## 📊 Google Sheets Bank-Receipt Ingestion

This API can read digital-payment receipt registers (e.g. **Nequi Bre-B**) from a Google Sheets spreadsheet, parse the sale details, and store them in the `email_receipts` table in paginated 50-row chunks.

### Configuration

Set the following environment variables (see `.env.example`):

| Variable | Description |
| :--- | :--- |
| `GOOGLE_SHEETS_SPREADSHEET_ID` | Google Spreadsheet ID (default: `1W_qTHc3RxTwwjCItOwCrRbnI34-63nLuLqG5it5LY7E`) |
| `GOOGLE_SHEETS_RANGE` | Sheet/Tab name (default: `Datos_Ventas`) |
| `GOOGLE_SHEETS_CHUNK_SIZE` | Batch size for paginated row fetches (default: `50`) |
| `GOOGLE_SHEETS_CREDENTIALS_JSON` | Service Account JSON credentials (inline string or base64) |
| `GOOGLE_APPLICATION_CREDENTIALS` | Path to Service Account JSON key file |
| `EMAIL_SYNC_CRON` | Optional cron expression for background sync. Empty disables it (on-demand only) |

If Google credentials are missing, the API still starts but the sync integration is disabled (the sync endpoint returns an informative error).

### Endpoints

- `POST /api/v1/email-receipts/sync` *(admin)* — on-demand sync; optional body `{ "from": "2026-09-01", "to": "2026-09-21" }`. Reads sheet rows in chunks, parses and upserts new receipts, and returns a summary (`fetched`, `imported`, `skipped`, `errors`).
- `GET /api/v1/email-receipts` *(admin / salesperson)* — lists stored receipts, optionally filtered with `?from=YYYY-MM-DD&to=YYYY-MM-DD`.
- `GET /api/v1/email-receipts/summary` *(admin / salesperson)* — revenue metrics, growth, and daily timeline buckets.

Ingestion is idempotent: receipts are deduplicated on the `message_id` (Column J: `ID Mensaje`), so re-running a sync is always safe.

### Database

Apply migrations to an existing database:

```bash
# Using Go migration runner (cross-platform, loads .env automatically)
go run ./cmd/migrate

# Or using psql directly
psql "$DATABASE_URL" -f db/migrations/0002_add_cafeteria_category.sql
```

New environments get the full schema and seed data automatically from `db/schema.sql` and `db/seeds.sql`.

---

> **🇨🇴 Made in Envigado, Colombia**
> *Built with passion and lots of excelent coffee.*

---

## 🧪 Testing & TDT Gate

This project enforces **table-driven unit tests** on every new or modified Go file under `internal/`. Coverage is checked with `go-test-coverage`; a gate runs before each commit and on every CI run.

### Run the gate locally

```bash
# Pre-commit scope (staged files)
go run ./cmd/testgate -staged

# Diff against the base branch (CI mode)
go run ./cmd/testgate -base origin/main

# Require 80% for the whole changed package instead of the changed file
go run ./cmd/testgate -staged -scope package

# Audit the whole repository (reports all remaining legacy gaps)
go run ./cmd/testgate -all
```

### Install the pre-commit hook

```bash
lefthook install
```

Then every `git commit` runs the gate on staged files. It can be bypassed with `--no-verify`, but the CI workflow (`.github/workflows/tdt-gate.yml`) is the real gate.

### What the gate checks

1. **Test presence** — every changed non-test `.go` file under `internal/` must have a sibling `_test.go`, excluding `internal/models`, `cmd/`, `db/`, `docs/`, generated `*.pb.go`, and `vendor/`.
2. **Coverage** — every changed file must report at least 80% statement coverage. Pass `-scope package` to require 80% for the whole changed package instead.

The floor is per changed file by default because the existing packages are far below 80% overall (`internal/handlers` is at 6.7%, `internal/service` at 10.9%). A package-scoped floor would block any edit to those packages until every file in them is tested.

See `.kilo/command/tdt.md` for the one-keystroke flow and `.kilo/skills/go-master-api/SKILL.md` for the TDT mandate.
