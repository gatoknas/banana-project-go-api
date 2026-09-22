# Full Stack Debug: pod-backed native Go API + native web + Firefox

## Goal

Make `F5 -> Full Stack Debug` reliably start, in this order, the full local stack:

1. **PostgreSQL** — pod `banana-pod` container `banana-db` (`:5432`)
2. **pgAdmin** — pod `banana-pod` container `banana-pgadmin` (`:5050`)
3. **Go API** — **native on host** under Delve, binding `:8082` (Delve cannot attach to a containerized binary)
4. **Vue web** — native `npm run serve` (`:8081`)
5. **Firefox Developer Edition** — launched/attached for JS breakpoints

Plain `.\dev.ps1` (API built and run **in** the pod, no Delve) must keep working unchanged.

## Confirmed decisions

- API runs **native under Delve** for debugging (chosen by user). It is intentionally *not* a pod container in `Full Stack Debug`.
- Pod in debug mode = **postgres + pgAdmin only**.
- `Full Stack Debug` also launches **Firefox Developer Edition** (existing web-folder config `Launch Firefox Developer Edition`).
- Start order DB -> pgAdmin -> API -> web is already what `dev.ps1` does (see `dev.ps1:554-597`); keep it.

## How to use it (runbook)

**Debug the full stack — one keystroke:**

1. (Optional) `.\dev.ps1 -Stop` if anything is already running.
2. Open the `banana-project.code-workspace` multi-root workspace in VS Code.
3. Press **F5** and pick **Full Stack Debug**. No terminal steps are needed.

What F5 does, in order:

1. VS Code runs the compound's `preLaunchTask: Dev: DB Only` and **waits for it to exit** before starting any debug session (`banana-project.code-workspace:154`).
2. `dev.ps1 -DbOnly` runs `Ensure-Machine` -> `Retire-Legacy` -> `Ensure-Pod -Mode debug` (postgres + pgAdmin, ports `5432`/`5050`, no `8082`) -> `Remove-ApiContainer` -> `Ensure-Db` -> `Wait-Postgres` (`pg_isready`) -> seed if empty -> `Ensure-Pgadmin` (writes config, starts container, registers `Banana Local`). It then prints "Database ready..." and exits `0` (`dev.ps1:587-607`).
3. Only after step 2 succeeds does VS Code start the three compound sessions **concurrently**: `Debug Go API` (native Delve on `:8082`), `Run Dev Server (npm run serve)` (`:8081`), `Launch Firefox Developer Edition`.

Ordering guarantees and non-guarantees:

- **Guaranteed before the API/web start:** PostgreSQL is accepting connections and pgAdmin is up, because the pre-launch task blocks on `pg_isready` and pgAdmin import.
- **Not sequenced:** API, web dev server, and Firefox start together. The web server does not need the API to boot; if Firefox opened before `:8081` was listening, reload the tab once. Go breakpoints also require the API to finish its first Delve build (a few seconds).

Other entry points (not F5):

- `.\dev.ps1` — `full` mode: pod DB + pgAdmin + **API container**, then native web.
- `.\dev.ps1 -NativeApi` — pod DB + pgAdmin, native API, web.
- `.\dev.ps1 -DbOnly` — DB + pgAdmin only (the step F5 uses).
- `.\dev.ps1 -Reset` — `full` mode with wipe + reseed.
- `.\dev.ps1 -Status` / `.\dev.ps1 -Stop`.
- Ctrl+Shift+B — `Dev: Up (DB + API + Web)`.

Gotchas:

- Do **not** run plain `.\dev.ps1` while F5 is active: both start a web server on `8081`, and plain mode holds `8082` via the pod. Stop one first (`.\dev.ps1 -Stop`).
- F5 does **not** rebuild the API image (it compiles the Go binary for Delve). Only `full` mode builds the image.

## Findings (why this doesn't work today)

- `banana-project.code-workspace:146` compound starts native `Debug Go API` with `preLaunchTask: Dev: DB Only`.
- `dev.ps1 -DbOnly` correctly never starts an API container, **but** `Ensure-Pod` (`dev.ps1:163-183`) always creates the pod with `-p 8082:8082` and never removes a stale `banana-api` container. `dev.ps1:466-468` itself states podman owns host ports `8082`/`5050`.
- Podman pods cannot gain/lose published ports after creation, so the pod must be (re)created with a port set that matches the runtime. Result today: podman reserves host `8082`, the native API under Delve cannot bind it, and `F5` fails with `bind: address already in use`.

**Verify first (before editing):** `.\dev.ps1 -DbOnly` then `podman port banana-pod` (and `podman ps --pod --format '{{.Names}}'`). If `8082` is listed / a `banana-api` container persists with only DB+pgAdmin intended, the diagnosis is confirmed. The fix below is still correct if podman lazily binds, and it future-proofs the transition in both directions.

## Changes

All edited files live in the **parent** folder `C:\Users\danie\Documents\SourceCode\banana-project` (outside this repo's workspace root). The implementing agent needs access to that folder.

### 1. `dev.ps1` — mode-aware pod (primary fix)

**a. Compute mode/ports in `Main`, right after `$jwt` is resolved (`dev.ps1:550`) and before `Ensure-Pod`:**

```powershell
$useApiContainer = -not ($NativeApi -or $DbOnly -or $NoApi)
$podMode = if ($useApiContainer) { 'full' } else { 'debug' }

$podPorts = @("${PgPort}:5432")
if (-not $NoPgadmin) { $podPorts += "${PgadminPort}:80" }
if ($useApiContainer) { $podPorts += "${ApiPort}:8082" }
```

**b. Replace `Ensure-Pod` (`dev.ps1:163-183`) with a labelled, mode-reconciling version. Recreate the pod when the stored mode differs; volumes are preserved.**

```powershell
function Ensure-Pod {
    param([string]$Mode, [string[]]$Ports)

    $label = 'banana.pod.mode'
    if (Test-PodExists) {
        $current = (& podman pod inspect $PodName --format '{{json .Labels}}' 2>$null | Out-String).Trim()
        if ($current -notmatch ([regex]::Escape('"' + $label + '":"' + $Mode + '"'))) {
            Write-Step "Recreating pod '$PodName' for mode '$Mode'"
            & podman pod rm -f $PodName *> $null
            if ($LASTEXITCODE -ne 0) { throw 'podman pod rm failed' }
        }
    }

    if (-not (Test-PodExists)) {
        Write-Step "Creating pod '$PodName' (mode '$Mode'; ports $($Ports -join ', '))"
        $create = @('pod', 'create', '--name', $PodName, '--label', "$label=$Mode")
        foreach ($p in $Ports) { $create += @('-p', $p) }
        & podman @create
        if ($LASTEXITCODE -ne 0) { throw 'podman pod create failed' }
        Write-Ok 'Pod created'
        return
    }

    if ((& podman inspect $PodName --format '{{.State}}' 2>$null) -eq 'Running') {
        Write-Ok "Pod '$PodName' already running"
    } else {
        Write-Step "Starting pod '$PodName'"
        & podman pod start $PodName | Out-Null
        Write-Ok 'Pod started'
    }
}
```

**c. Update the call site (`dev.ps1:554`):**

```powershell
Ensure-Pod -Mode $podMode -Ports $podPorts
```

**d. Remove any stale API container when the API won't be containerized.** Add a helper near `Ensure-ApiContainer` and call it after `Ensure-Pod`:

```powershell
function Remove-ApiContainer {
    if (Test-ContainerExists -Name $ApiContainer) {
        Write-Step "Removing API container '$ApiContainer' (API runs natively for debug)"
        & podman rm -f $ApiContainer *> $null
    }
}
```

Call in `Main` right after `Ensure-Pod`:
```powershell
if (-not $useApiContainer) { Remove-ApiContainer }
```

Migration note: pods created by the current script have no `banana.pod.mode` label, so the first run after this change recreates the pod once. Named volumes `banana-project_banana_pgdata` and `banana_pgadmin` are reused, so DB data and pgAdmin config persist.

**e. Optional:** print the mode in `Show-Status` (`dev.ps1:499-533`).

### 2. `banana-project.code-workspace` — add Firefox to the compound

Edit the `launch.compounds` block (`banana-project.code-workspace:145-156`) so the third configuration is the existing web-folder target:

```json
{
  "name": "Full Stack Debug",
  "configurations": [
    "Debug Go API",
    "Run Dev Server (npm run serve)",
    "Launch Firefox Developer Edition"
  ],
  "preLaunchTask": "Dev: DB Only"
}
```

Leave the tasks block unchanged. `Dev: DB Only` (`-DbOnly`) is the correct pre-launch step: it brings up DB + pgAdmin in `debug` mode (no `8082`) and exits before the compound starts.

Known limitation: compound configurations start concurrently, so Firefox may open a few hundred ms before `npm run serve` is listening and show a connection error. Reload the tab once. Do not add a `preLaunchTask` wait on `8081` — pre-launch tasks run before the web server starts.

### 3. `banana-pod.yaml` — document the two modes (comments only)

Update the header comment to state that `dev.ps1` is authoritative and creates the pod in two modes:

- `debug` (F5 / `-DbOnly` / `-NativeApi`): postgres + pgAdmin only, **no** `8082` publish.
- `full` (plain `.\dev.ps1`): postgres + pgAdmin + API container, `8082` published.

The YAML body stays the `full` reference topology. No functional change is required.

### 4. `developer_manual_run_web_app.md` — keep docs truthful

- In the `VS Code integration` section, state that `Full Stack Debug` = `Dev: DB Only` -> native `Debug Go API` + `Run Dev Server` + `Launch Firefox Developer Edition`, and that the pod intentionally has no API container in this mode.
- Add a troubleshooting row: `bind: address already in use` on `8082` -> the pod was created in `full` mode; rerun `.\dev.ps1 -DbOnly` (the script now recreates the pod in `debug` mode and drops `8082`).

## Validation

1. `.\dev.ps1 -Stop`; optionally `podman pod rm -f banana-pod`.
2. `.\dev.ps1 -DbOnly`, then `podman port banana-pod` shows only `5432` and `5050`; `podman ps --pod --format '{{.Names}}'` shows `banana-db` and `banana-pgadmin` only.
3. F5 -> **Full Stack Debug**: `Dev: DB Only` completes; API binds `8082`. `Invoke-RestMethod http://localhost:8082/status` succeeds; `POST /login` with `admin` / `adminpassword` returns a token; `http://localhost:8081` loads; Firefox attaches. Set a breakpoint in a Go handler and confirm it hits.
4. Regression (container mode): `.\dev.ps1` -> `podman port banana-pod` includes `8082`; `banana-api` container serves `/status`.
5. Mode switch: from `full`, press F5 again. Script logs `Recreating pod ... for mode 'debug'`; native API binds `8082`. Confirm DB data survived (`podman exec banana-db psql -U danieluser -d banana_project_db -c "\dt"`) and pgAdmin `Banana Local` still present.
6. `.\dev.ps1 -Status` reports DB ready, API up, pgAdmin up, web up.

## Notes / out of scope

- No Go source under `internal/` or `cmd/` is touched by the pod changes, so the AGENTS.md TDT gate does not apply.
- Alternative if pod recreation proves unreliable: run the native API on a debug-only port and point a debug web env at it. Not chosen — it breaks the documented `8082` contract.

---

# Follow-up: local web login fails with "CORS sin éxito" / NetworkError

## Symptom and evidence

- Logging in from the F5-launched Firefox at `http://localhost:8081` fails with:
  `Solicitud de origen cruzado bloqueada ... http://localhost:8082/login (Razón: Solicitud CORS sin éxito). Código de estado: (null).`
  and `TypeError: NetworkError when attempting to fetch resource.`
- `Invoke-RestMethod http://localhost:8082/status` works and reports `"database":"connected"`.
- The API's CORS rule is already present and correct: `internal/middleware/cors.go:9` sends `Access-Control-Allow-Origin: *` on every response and answers preflight `OPTIONS` with 200 (`cors.go:14-17`), applied to the whole router at `cmd/api/main.go:103`.
- Status `(null)` + `NetworkError` means the cross-origin request never completed from the browser. So `CORS Failed` is a symptom of a browser-level network failure reaching `localhost:8082` (proxy/IPv6/preflight environment), not a missing header rule.
- The `file:///` line in the console is devtools/source-map noise, not the API call.

## Decision (user-approved)

Use a same-origin **Vue dev-server proxy**: the browser talks only to `http://localhost:8081`, and the dev server forwards the API paths to `http://127.0.0.1:8082`. This removes CORS and browser-to-`8082` network quirks from local debugging. Targeting `127.0.0.1` (not `localhost`) also avoids IPv6 `::1` ambiguity. Production behaviour is unchanged.

## Changes (web repo only — no Go files, TDT gate not triggered)

### 1. `banana-project-web/vue.config.js` — add dev proxy

```js
const { defineConfig } = require('@vue/cli-service')
module.exports = defineConfig({
  transpileDependencies: true,
  devServer: {
    port: 8081,
    proxy: {
      '/login':   { target: 'http://127.0.0.1:8082' },
      '/refresh': { target: 'http://127.0.0.1:8082' },
      '/status':  { target: 'http://127.0.0.1:8082' },
      '/hello':   { target: 'http://127.0.0.1:8082' },
      '/docs':    { target: 'http://127.0.0.1:8082' },
      '/api':     { target: 'http://127.0.0.1:8082' }
    }
  }
})
```

Contexts mirror the routes registered in `cmd/api/main.go`: `/login`, `/refresh`, `/status`, `/hello` (+ `/hello/logo.png`), `/docs` (+ `/docs/swagger.json`), and the protected `/api/v1/*` group. Dev-server HMR/websocket paths are untouched.

### 2. `banana-project-web/src/services/api.ts` — relative base URL in local dev

Change `getBaseUrl()` so that when `VUE_APP_API_URL` is not set and the app runs on `localhost`/`127.0.0.1`, it returns `''` (relative) instead of `http://localhost:8082`, so requests hit the dev server and get proxied. Keep `const BASE_URL = getBaseUrl();` and every `${BASE_URL}${path}` call site unchanged.

```ts
const getBaseUrl = (): string => {
  const configured = process.env.VUE_APP_API_URL;
  if (configured) {
    return configured.endsWith('/') ? configured.slice(0, -1) : configured;
  }
  if (typeof window !== 'undefined') {
    const hostname = window.location.hostname;
    if (hostname === 'localhost' || hostname === '127.0.0.1') {
      return ''; // same-origin; vue.config.js devServer.proxy forwards to the Go API
    }
    return 'https://api.ayurami.com';
  }
  return '';
};
```

### 3. `banana-project-web/.env.development` — use the proxy by default

Set the value empty so the proxy path wins; an absolute value still bypasses the proxy:

```
# Leave empty to use the Vue dev-server proxy (vue.config.js -> 127.0.0.1:8082).
# Set to an absolute API URL (e.g. https://api.ayurami.com) to bypass the proxy.
VUE_APP_API_URL=
```

### 4. `developer_manual_run_web_app.md` — document

- Update the architecture note / Web row: in local dev the browser calls the API through the dev-server proxy (`http://localhost:8081` -> `127.0.0.1:8082`); `VUE_APP_API_URL` still overrides it.
- Add a troubleshooting row: `CORS Failed` / `NetworkError` on `http://localhost:8082/...` -> use the dev proxy; restart `npm run serve` after changing `.env.development` or `vue.config.js`.

## Validation

1. Restart the web dev server (`.env.development` and `vue.config.js` changes require a restart): stop the running `Run Dev Server` session and re-run F5 -> **Full Stack Debug**.
2. Firefox DevTools -> Network: `POST http://localhost:8081/login` returns 200; there is no `OPTIONS` preflight and no browser request to `:8082`.
3. Log in as `admin` / `adminpassword`, then confirm a protected call (`GET http://localhost:8081/api/v1/products`) returns 200 with the token.
4. Direct API access is unaffected: `Invoke-RestMethod http://localhost:8082/status`.
5. If the proxy returns 500/ECONNREFUSED, the dev-server terminal shows the proxy error -> the API is not on `127.0.0.1:8082` (check the pod mode with `.\dev.ps1 -Status`).
6. Production build unaffected: `npm run build` still uses `VUE_APP_API_URL` when set; the `localhost` branch never applies to a non-local host.
