# Financials API

A local-first personal financial planning API, rebuilt from the old prototype as a Go service.

The first implementation phase focuses on the API skeleton, generic financial item tracking, and deterministic projection workflows. Drawdown projection features will build on the current accumulation-only projection contract once the next backend decisions are pinned.

## Current status

- Runtime: Go HTTP API
- Current branch focus: AWS serverless ephemeral backend
- Implemented endpoints: `GET /health`, `/financial-items` create/list/read/update/delete behavior, `GET`/`POST /financial-items/backup`, and accumulation/drawdown `POST /projections`
- Implemented domain pieces: financial item request/response models, validation, deterministic fake fixtures, repository behavior tests, projection calculation logic, drawdown-capable projection engine models, inflation-adjusted drawdown withdrawals, repository-backed per-item drawdown return wiring, per-item contribution inflation flags, JSON backup replacement imports, request-supplied projection item support for browser-owned sessions, and projection/drawdown UI integration tracking
- Implemented local storage options: process-local memory, explicit ephemeral/non-durable memory, and gitignored JSON file storage
- Implemented deploy-readiness option: placeholder-configured CORS allowed origins for future static hosting
- Next planned area: deployed access control using API Gateway API key + usage plan as the first pragmatic gate before real data
- Runtime/deployment specifics: represented with placeholders only; real local values belong in ignored `.env` files

## Planning documents

- [App Rewrite Plan](docs/app-rewrite.md)
- [Projection Planning](docs/projection-planning.md)
- [Drawdown Projection Planning](docs/drawdown-projection-planning.md)

## Local development

Requirements:

- Go 1.22+

Local config:

```powershell
Copy-Item .env.example .env
```

Then edit `.env` for local-only overrides. `.env` and `/data/` are gitignored; keep real machine-specific paths, hostnames, and private financial data out of commits.

Configuration precedence:

1. Process environment variables
2. Local `.env`
3. Built-in defaults

Supported config values:

- `FINANCIALS_API_ADDR`: Go `http.Server` bind address, default `:8080`
- `FINANCIALS_STORAGE_DRIVER`: `memory`, `ephemeral`, or `json`, default `memory`
- `FINANCIALS_STORAGE_PATH`: required when `FINANCIALS_STORAGE_DRIVER=json`, for example `./data/financial-items.json`
- `FINANCIALS_ALLOWED_ORIGINS`: optional comma-separated browser origins allowed to call the API directly, blank by default for the local Vite proxy workflow

Run tests:

```powershell
go test ./...
```

Start the API locally:

```powershell
go run ./cmd/api
```

The service listens on `:8080` by default. Override the bind address with `FINANCIALS_API_ADDR`, for example:

```powershell
$env:FINANCIALS_API_ADDR=":8081"; go run ./cmd/api
```

To persist local fake/test financial items across restarts with the JSON adapter:

```powershell
$env:FINANCIALS_STORAGE_DRIVER="json"; $env:FINANCIALS_STORAGE_PATH="./data/financial-items.json"; go run ./cmd/api
```

Do not commit the generated `/data/financial-items.json` file.

For a deployed/static UI that keeps financial items in the browser instead of saving them through API CRUD, run the API in explicit non-durable mode and send projection inputs in the `POST /projections` request body:

```powershell
$env:FINANCIALS_STORAGE_DRIVER="ephemeral"; go run ./cmd/api
```

`ephemeral` uses the same process-local memory repository as `memory`; it is only a clear runtime signal that the intended workflow is browser-owned import/export plus request-supplied projection items. Do not treat Lambda/process memory as a reliable browser session store.

## AWS serverless backend

The initial AWS backend path is optimized for the browser-owned ephemeral workflow: API Gateway HTTP API invokes a Go Lambda function, and the React app sends request-supplied projection items instead of relying on server-side session storage.

Committed AWS files are public-safe placeholders only:

- `template.yaml`: SAM template for HTTP API + Lambda using `FINANCIALS_STORAGE_DRIVER=ephemeral` by default.
- `Makefile`: SAM makefile target that builds the Lambda custom-runtime `bootstrap` for `provided.al2023`.

Build the Lambda artifact locally:

```powershell
$env:ARTIFACTS_DIR=".aws-sam/build/FinancialsApiFunction"; New-Item -ItemType Directory -Force $env:ARTIFACTS_DIR | Out-Null; make build-FinancialsApiFunction
```

Deploy with private/environment-specific values supplied at deploy time, not committed to git:

```powershell
sam build; sam deploy --guided --profile zachfire9
```

Use placeholder/default settings for fake-data smoke tests only. Do not put real financial data through a public unauthenticated API. Persistent DynamoDB storage is intentionally deferred until after the ephemeral AWS path and access-control requirements are reviewed.

## CORS and deploy-readiness

The local Vite development workflow still uses the UI dev-server proxy, so CORS can stay disabled by leaving `FINANCIALS_ALLOWED_ORIGINS` blank.

When a future static-hosted UI needs to call this API directly, set placeholder-style allowed origins in local/private runtime config:

```powershell
$env:FINANCIALS_ALLOWED_ORIGINS="https://<static-ui-host.example>"; go run ./cmd/api
```

Multiple origins can be comma-separated:

```env
FINANCIALS_ALLOWED_ORIGINS=https://<static-ui-host.example>,http://localhost:5173
```

Keep real deployed origins, private LAN hostnames/IPs, and environment-specific deployment values in ignored `.env` files or private operator notes, not in committed docs.

Check the health endpoint:

```powershell
Invoke-RestMethod http://localhost:8080/health
```

Expected response:

```json
{"status":"ok"}
```

## Financial items API

Financial items are generic projection inputs such as example savings, brokerage, or goal balances. Use fake/example data in committed docs and tests only. `drawdownAnnualReturnRateBasisPoints` is optional; omit it to reuse the regular `annualReturnRateBasisPoints` once drawdown begins.

Create an item:

```powershell
Invoke-RestMethod http://localhost:8080/financial-items -Method Post -ContentType 'application/json' -Body '{"name":"Example brokerage","amountCents":1250000,"currency":"USD","annualReturnRateBasisPoints":700,"drawdownAnnualReturnRateBasisPoints":400,"annualContributionCents":300000,"inflateAnnualContribution":true,"sortOrder":1}'
```

Expected response shape:

```json
{
  "id": "item_000001",
  "name": "Example brokerage",
  "amountCents": 1250000,
  "currency": "USD",
  "annualReturnRateBasisPoints": 700,
  "drawdownAnnualReturnRateBasisPoints": 400,
  "annualContributionCents": 300000,
  "inflateAnnualContribution": true,
  "sortOrder": 1,
  "createdAt": "2026-01-01T00:00:00Z",
  "updatedAt": "2026-01-01T00:00:00Z"
}
```

List items:

```powershell
Invoke-RestMethod http://localhost:8080/financial-items
```

Read one item:

```powershell
Invoke-RestMethod http://localhost:8080/financial-items/item_000001
```

Update one item:

```powershell
Invoke-RestMethod http://localhost:8080/financial-items/item_000001 -Method Put -ContentType 'application/json' -Body '{"name":"Example down payment fund","amountCents":1500000,"currency":"USD","annualReturnRateBasisPoints":400,"drawdownAnnualReturnRateBasisPoints":250,"annualContributionCents":250000,"inflateAnnualContribution":false,"sortOrder":2}'
```

Delete one item:

```powershell
Invoke-RestMethod http://localhost:8080/financial-items/item_000001 -Method Delete
```

Validation failures return `400` with an error message. Missing item IDs return `404`.

## JSON backup export/import

`GET /financial-items/backup` exports all saved financial items as a public JSON shape with `schemaVersion`, `exportedAt`, and `items`. Real backup files can contain private financial data; keep them local and out of git.

Export an example backup:

```powershell
Invoke-RestMethod http://localhost:8080/financial-items/backup | ConvertTo-Json -Depth 10 | Set-Content .\\financials-backup.example.local.json
```

Import replaces the current saved financial items with the validated backup payload. The import preserves explicit item IDs, sort order, timestamps, and optional drawdown return assumptions when the payload is valid.

Restore checklist:

1. Start the API with the intended storage adapter.
2. Confirm the backup file is local/private and not under source control.
3. Import the JSON backup.
4. List financial items and recalculate projections.

```powershell
$backup = Get-Content .\\financials-backup.example.local.json -Raw
Invoke-RestMethod http://localhost:8080/financial-items/backup -Method Post -ContentType 'application/json' -Body $backup
```

Validation failures return `400` and leave existing repository contents unchanged.

## Projection API

`POST /projections` calculates a deterministic whole-year projection with fake/example inputs or with the current saved financial items. It supports the original accumulation-only `years` shape and the newer explicit saving/drawdown shape.

Calculate from the current repository-backed financial items by omitting `items` or sending an empty `items` array:

```powershell
Invoke-RestMethod http://localhost:8080/projections -Method Post -ContentType 'application/json' -Body '{"years":10}'
```

Calculate a hypothetical unsaved accumulation scenario by providing `items`:

```powershell
Invoke-RestMethod http://localhost:8080/projections -Method Post -ContentType 'application/json' -Body '{"years":2,"items":[{"name":"Example brokerage","amountCents":1250000,"currency":"USD","annualReturnRateBasisPoints":700,"annualContributionCents":300000,"inflateAnnualContribution":true,"sortOrder":1}]}'
```

Calculate a hypothetical unsaved drawdown scenario. Set `inflateAnnualContribution` to `true` on individual items whose saving-year contributions should grow by `annualWithdrawalInflationRateBasisPoints`; omit it or set it to `false` on items whose contributions should stay fixed. Projection calculations preserve the base annual contribution value on each item.

```powershell
Invoke-RestMethod http://localhost:8080/projections -Method Post -ContentType 'application/json' -Body '{"savingYears":1,"drawdownYears":2,"annualWithdrawalCents":6000000,"annualWithdrawalInflationRateBasisPoints":300,"items":[{"name":"Example retirement account","amountCents":20000000,"currency":"USD","annualReturnRateBasisPoints":0,"drawdownAnnualReturnRateBasisPoints":0,"annualContributionCents":100000,"inflateAnnualContribution":true,"sortOrder":1}]}'
```

Expected response shape excerpt:

```json
{
  "years": 3,
  "savingYears": 1,
  "drawdownYears": 2,
  "currency": "USD",
  "items": [
    {
      "id": "",
      "name": "Example retirement account",
      "startingAmountCents": 20000000,
      "annualReturnRateBasisPoints": 0,
      "drawdownAnnualReturnRateBasisPoints": 0,
      "annualContributionCents": 100000,
      "inflateAnnualContribution": true,
      "yearlyBalances": [
        {
          "year": 0,
          "phase": "starting",
          "balanceCents": 20000000,
          "contributionCents": 0,
          "withdrawalCents": 0,
          "growthCents": 0,
          "unfundedWithdrawalCents": 0
        },
        {
          "year": 1,
          "phase": "saving",
          "balanceCents": 20100000,
          "contributionCents": 100000,
          "withdrawalCents": 0,
          "growthCents": 0,
          "unfundedWithdrawalCents": 0
        },
        {
          "year": 2,
          "phase": "drawdown",
          "balanceCents": 13920000,
          "contributionCents": 0,
          "withdrawalCents": 6180000,
          "growthCents": 0,
          "unfundedWithdrawalCents": 0
        }
      ]
    }
  ]
}
```

Projection rules:

- Accumulation-only requests use `years`, which must be between `1` and `75`.
- Drawdown-capable requests use `savingYears` plus optional `drawdownYears`; `years` is mutually exclusive with those phase fields.
- `drawdownYears` greater than `0` requires `annualWithdrawalCents`.
- `annualWithdrawalInflationRateBasisPoints` is optional and defaults to `0`; `300` means the requested withdrawal grows by 3% after each projection year, so one saving year makes the first drawdown withdrawal $61,800 from a $60,000 base.
- `drawdownAnnualReturnRateBasisPoints` is optional per item; if omitted, drawdown years use the normal `annualReturnRateBasisPoints`.
- All items in one projection must use the same currency.
- Hypothetical `items` are validated but not saved.
- Unknown JSON fields return `400` to catch request typos.

## Public repo boundaries

This repo is public, so it must not contain:

- Real financial data
- Secrets, API keys, passwords, or tokens
- LAN IPs, hostnames, router/firewall details, or machine names
- Local database paths, backup paths, or user-specific service names
- Environment-specific production configuration

Use committed examples with placeholders only, and keep real runtime values in ignored local files such as `.env` when implementation begins.
