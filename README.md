# Financials API

A local-first personal financial planning API, rebuilt from the old prototype as a Go service.

The first implementation phase focuses on the API skeleton and generic financial item tracking. Projection features will build on those configurable inputs once the model and workflow are stable.

## Current status

- Runtime: Go HTTP API
- Current branch focus: financial items HTTP API
- Implemented endpoints: `GET /health` plus `/financial-items` create/list/read/update/delete behavior
- Implemented domain pieces: financial item request/response models, validation, deterministic fake fixtures, and in-memory repository behavior tests
- Next planned area: local configuration and storage workflow
- Runtime/deployment specifics: intentionally omitted from git until they can be represented with placeholders and local-only config files

## Planning documents

- [App Rewrite Plan](docs/app-rewrite.md)

## Local development

Requirements:

- Go 1.22+

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

## Public repo boundaries

This repo is public, so it must not contain:

- Real financial data
- Secrets, API keys, passwords, or tokens
- LAN IPs, hostnames, router/firewall details, or machine names
- Local database paths, backup paths, or user-specific service names
- Environment-specific production configuration

Use committed examples with placeholders only, and keep real runtime values in ignored local files such as `.env` when implementation begins.
