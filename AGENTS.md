# AGENTS.md

Guidance for coding agents working in this repo. [README.md](README.md) has the full API reference and setup details.

## What this is

A Go HTTP API for a personal retirement planner. It stores generic financial items and runs deterministic year-by-year projections (saving, then drawdown). The UI lives in the sibling repo [financials-ui](https://github.com/zachfire9/financials-ui).

The same `http.Handler` runs in two places:

- **Locally**, through `cmd/api` (plain `net/http`).
- **On AWS**, through `cmd/lambda` and `internal/lambdahttp`, behind an API Gateway HTTP API.

Production runs in `ephemeral` storage mode: only `GET /health` and `POST /projections` are served, and the browser sends the items with each projection request. `memory` and `json` modes serve the full CRUD and backup API for local development.

## Commands

```powershell
go vet ./...
go test -race ./...           # run before every commit
go run ./cmd/api              # local server on :8080
```

Build the Lambda binary the way SAM does (Linux, arm64):

```powershell
$env:GOOS="linux"; $env:GOARCH="arm64"; go build -tags lambda.norpc -o bootstrap ./cmd/lambda
```

The Go version comes from `go.mod`. CI reads it from there too (`go-version-file: go.mod`), so bump it only in `go.mod`.

## Code layout

| Path | Contents |
|------|----------|
| `internal/httpapi` | Routes, CORS, access token gate, request decoding, JSON errors |
| `internal/financialitems` | Item model and validation, repositories (memory, JSON file), backups |
| `internal/projections` | Projection engine and its validation |
| `internal/config` | Config from process env, then `.env`, then defaults |
| `internal/app` | Builds the handler from config |
| `internal/lambdahttp` | API Gateway event adapter |
| `infra/iam/` | Deploy role IAM policies (applied by hand, see below) |

## Conventions

- **Money is integer cents; rates are integer basis points.** `amountCents`, `annualReturnRateBasisPoints` (700 = 7%). Never use floats for stored or returned amounts.
- **Errors are JSON** (`{"error": "..."}`) via `writeError`. 4xx messages explain the problem to the user. 5xx messages are always `internal error`, and the real error is logged.
- **Request bodies** go through `decodeJSONBody`: unknown fields, trailing data and bodies over 1 MiB are rejected.
- **The JSON file repository** holds `writeMu` across each change and its save, and saves through a temp file plus rename. Keep both if you touch it.
- **Tests sit next to the code** (`*_test.go`), use the standard `testing` package (no assertion libraries), and use fake data (`FakeCreateFinancialItemRequest` and similar). New behavior needs a test; bug fixes need a test that fails without the fix.
- **Formatting:** `gofmt`. Line endings are LF (`.gitattributes`).
- **Commit messages** use conventional prefixes: `feat:`, `fix:`, `chore:`, `docs:`.

## Public repo rules

This repo is public. Never commit:

- Real financial data, backups, or `/data/` contents
- Secrets or tokens (`FINANCIALS_ACCESS_TOKEN` lives in GitHub Secrets and local `.env` only)
- The AWS account ID, real API IDs, or other environment-specific values. Use placeholders like `<aws-account-id>` and `<api-id>`.
- LAN IPs, hostnames or machine names

Filled-in IAM policies (`/*-policy*.json`, `/*-trust*.json` at the repo root) are gitignored. Keep it that way. When staging, add specific paths rather than `git add -A`, because local working notes may be untracked in the tree.

## Deploying

- **Every push to `master` deploys** through `.github/workflows/deploy-api.yml`: tests, `sam build`, `sam deploy`, then a `/health` smoke test. Work on a branch and open a PR; don't push to `master` directly.
- **Deploy settings** come from GitHub repo variables and secrets (see [docs/github-actions-deploy.md](docs/github-actions-deploy.md)). Parameters that aren't passed keep their previous stack values.
- **The deploy role's IAM policies** are managed by hand in AWS. `infra/iam/` is the source of truth: change the files in a PR, then the owner applies them with the commands in [infra/iam/README.md](infra/iam/README.md). The role is scoped to this stack's resources, so a change that needs a new kind of AWS resource also needs a policy update, or the deploy fails with AccessDenied.
- **Template changes deploy automatically.** Changing `template.yaml` in a way that replaces the HTTP API changes the API URL, which breaks the UI and the pinned `<api-id>` in the deploy policy. Avoid that unless it's intended.
