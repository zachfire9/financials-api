# Financials API

A local-first personal financial planning API, rebuilt from the old prototype as a Go service.

The first implementation phase focuses on the API skeleton and generic financial item tracking. Projection features will build on those configurable inputs once the model and workflow are stable.

## Current status

- Runtime: Go HTTP API
- Current branch focus: generic financial item model, validation, and repository behavior
- Implemented endpoint: `GET /health`
- Implemented domain pieces: financial item request/response models, validation, deterministic fake fixtures, and in-memory repository behavior tests
- Next planned area: financial items HTTP API
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
