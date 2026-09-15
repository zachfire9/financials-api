# Financials API

A local-first personal financial planning API, rebuilt from the old prototype as a Go service.

The first implementation phase focuses on the API skeleton, generic financial item tracking, and deterministic projection workflows. Drawdown projection features will build on the current accumulation-only projection contract once the next backend decisions are pinned.

## Current status

- Runtime: Go HTTP API
- Current branch focus: drawdown calculation engine
- Implemented endpoints: `GET /health`, `/financial-items` create/list/read/update/delete behavior, and accumulation-only `POST /projections`
- Implemented domain pieces: financial item request/response models, validation, deterministic fake fixtures, repository behavior tests, projection calculation logic, drawdown-capable projection engine models, and first projection UI integration
- Implemented local storage options: process-local memory and gitignored JSON file storage
- Implemented deploy-readiness option: placeholder-configured CORS allowed origins for future static hosting
- Next planned area: drawdown `POST /projections` API contract wiring after engine review
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
- `FINANCIALS_STORAGE_DRIVER`: `memory` or `json`, default `memory`
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

Financial items are generic projection inputs such as example savings, brokerage, or goal balances. Use fake/example data in committed docs and tests only.

Create an item:

```powershell
Invoke-RestMethod http://localhost:8080/financial-items -Method Post -ContentType 'application/json' -Body '{"name":"Example brokerage","amountCents":1250000,"currency":"USD","annualReturnRateBasisPoints":700,"annualContributionCents":300000,"sortOrder":1}'
```

Expected response shape:

```json
{
  "id": "item_000001",
  "name": "Example brokerage",
  "amountCents": 1250000,
  "currency": "USD",
  "annualReturnRateBasisPoints": 700,
  "annualContributionCents": 300000,
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
Invoke-RestMethod http://localhost:8080/financial-items/item_000001 -Method Put -ContentType 'application/json' -Body '{"name":"Example down payment fund","amountCents":1500000,"currency":"USD","annualReturnRateBasisPoints":400,"annualContributionCents":250000,"sortOrder":2}'
```

Delete one item:

```powershell
Invoke-RestMethod http://localhost:8080/financial-items/item_000001 -Method Delete
```

Validation failures return `400` with an error message. Missing item IDs return `404`.

## Projection API

`POST /projections` calculates a deterministic whole-year projection with fake/example inputs or with the current saved financial items.

Calculate from the current repository-backed financial items by omitting `items` or sending an empty `items` array:

```powershell
Invoke-RestMethod http://localhost:8080/projections -Method Post -ContentType 'application/json' -Body '{"years":10}'
```

Calculate a hypothetical unsaved scenario by providing `items`:

```powershell
Invoke-RestMethod http://localhost:8080/projections -Method Post -ContentType 'application/json' -Body '{"years":2,"items":[{"name":"Example brokerage","amountCents":1250000,"currency":"USD","annualReturnRateBasisPoints":700,"annualContributionCents":300000,"sortOrder":1}]}'
```

Expected response shape excerpt:

```json
{
  "years": 2,
  "currency": "USD",
  "items": [
    {
      "id": "",
      "name": "Example brokerage",
      "startingAmountCents": 1250000,
      "annualReturnRateBasisPoints": 700,
      "annualContributionCents": 300000,
      "yearlyBalances": [
        {
          "year": 0,
          "balanceCents": 1250000,
          "contributionCents": 0,
          "growthCents": 0
        },
        {
          "year": 1,
          "balanceCents": 1637500,
          "contributionCents": 300000,
          "growthCents": 87500
        }
      ]
    }
  ],
  "totals": [
    {
      "year": 0,
      "balanceCents": 1250000,
      "contributionCents": 0,
      "growthCents": 0
    }
  ]
}
```

Projection rules:

- `years` must be between `1` and `75`.
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
