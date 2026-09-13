# Financials API

A local-first personal financial planning API, rebuilt from the old prototype as a Go service.

The first implementation phase focuses on the API skeleton and current-investment tracking. Projection features are intentionally deferred until the investment model and workflow are built out.

## Current status

- Runtime: Go HTTP API
- Current branch focus: Go API skeleton with health endpoint
- Implemented endpoint: `GET /health`
- Next planned area: current investment model and storage API
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

## Public repo boundaries

This repo is public, so it must not contain:

- Real financial data
- Secrets, API keys, passwords, or tokens
- LAN IPs, hostnames, router/firewall details, or machine names
- Local database paths, backup paths, or user-specific service names
- Environment-specific production configuration

Use committed examples with placeholders only, and keep real runtime values in ignored local files such as `.env` when implementation begins.
