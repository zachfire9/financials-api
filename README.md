# Financials API

A local-first personal financial planning API, rebuilt from the old prototype as a Go service. It powers the Financials Planner UI ([financials-ui](https://github.com/zachfire9/financials-ui)), which answers: how long will my money actually last in retirement?

## Current status

- Runtime: Go HTTP API, deployed on AWS Lambda (arm64, `provided.al2023`) behind an API Gateway HTTP API in `ephemeral` mode
- Implemented endpoints: `GET /health`, `/financial-items` create/list/read/update/delete behavior, `GET`/`POST /financial-items/backup`, and accumulation/drawdown `POST /projections`
- Implemented domain pieces: financial item request/response models, validation, deterministic fake fixtures, repository behavior tests, projection calculation logic, drawdown-capable projection engine models, inflation-adjusted drawdown withdrawals, repository-backed per-item drawdown return wiring, per-item contribution inflation flags, JSON backup replacement imports, request-supplied projection item support for browser-owned sessions, and projection/drawdown UI integration tracking
- Implemented local storage options: process-local memory, explicit ephemeral/non-durable memory, and gitignored JSON file storage
- Implemented deploy-readiness option: placeholder-configured CORS allowed origins and optional shared-token access control for static hosting
- Not planned unless needed: stronger Cognito/OIDC/Lambda-authorizer identity or cloud persistence
- Runtime/deployment specifics: represented with placeholders only; real local values belong in ignored `.env` files

## Repository layout

- `cmd/api`: local `net/http` server
- `cmd/lambda`: Lambda entry point
- `internal/httpapi`: HTTP handler, routing, CORS, access token gate, error responses
- `internal/financialitems`: item model, validation, memory and JSON file repositories, backup import/export
- `internal/projections`: projection engine (accumulation and drawdown)
- `internal/config`: environment and `.env` loading
- `internal/app`: wires config, storage and handler together
- `internal/lambdahttp`: API Gateway HTTP API event adapter for the handler
- `infra/iam/`: IAM policies for the GitHub Actions deploy role (see [infra/iam/README.md](infra/iam/README.md))
- `template.yaml`, `Makefile`: SAM template and Lambda build target

## Planning documents

- [App Rewrite Plan](docs/app-rewrite.md)
- [Projection Planning](docs/projection-planning.md)
- [Drawdown Projection Planning](docs/drawdown-projection-planning.md)

## Local development

Requirements:

- Go 1.27+ (the version in `go.mod`; an older Go downloads it automatically)

Local config:

```powershell
Copy-Item .env.example .env
```

Then edit `.env` for local-only overrides. `.env` and `/data/` are gitignored; keep real machine-specific paths, hostnames, and private financial data out of commits.

Configuration precedence:

1. Process environment variables
2. Local `.env`
3. Built-in defaults

A variable set in the process environment always wins over `.env`, even when it's empty; an empty value then means "use the built-in default".

Supported config values:

- `FINANCIALS_API_ADDR`: Go `http.Server` bind address, default `:8080`
- `FINANCIALS_STORAGE_DRIVER`: `memory`, `ephemeral`, or `json`, default `memory`
- `FINANCIALS_STORAGE_PATH`: required when `FINANCIALS_STORAGE_DRIVER=json`, for example `./data/financial-items.json`
- `FINANCIALS_ALLOWED_ORIGINS`: optional comma-separated browser origins allowed to call the API directly, blank by default for the local Vite proxy workflow
- `FINANCIALS_ACCESS_TOKEN`: optional shared token; when set, non-health requests must send it in `X-Financials-Access-Token` (see below)

Run checks:

```powershell
go vet ./...
go test -race ./...
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

`ephemeral` is for the browser-owned workflow: the UI keeps items in the browser and sends them with every projection request. In this mode the API serves only `GET /health` and `POST /projections`; the financial item and backup routes return `404`, and a projection request without `items` returns `400` instead of reading server-side storage. This keeps Lambda/process memory from acting as an unreliable, per-instance session store. Use `memory` or `json` for the full CRUD API locally.

## AWS serverless backend

The initial AWS backend path is optimized for the browser-owned ephemeral workflow: API Gateway HTTP API invokes a Go Lambda function, and the React app sends request-supplied projection items instead of relying on server-side session storage.

GitHub Actions deployment setup is documented in [GitHub Actions API deploy](docs/github-actions-deploy.md). The workflow uses AWS OIDC and GitHub Secrets/Variables so deploy credentials and shared tokens stay out of git.

Committed AWS files are public-safe placeholders only:

- `template.yaml`: SAM template for HTTP API + Lambda using `FINANCIALS_STORAGE_DRIVER=ephemeral` by default.
- `Makefile`: SAM makefile target that builds the Lambda custom-runtime `bootstrap` for `provided.al2023`.
- `infra/iam/`: trust and permission policies for the GitHub Actions deploy role, with account-specific IDs as placeholders. The role is managed by hand in AWS; change these files in a PR, then apply them.

Build the Lambda artifact locally:

```powershell
$env:ARTIFACTS_DIR=".aws-sam/build/FinancialsApiFunction"; New-Item -ItemType Directory -Force $env:ARTIFACTS_DIR | Out-Null; make build-FinancialsApiFunction
```

Deploy with private/environment-specific values supplied at deploy time, not committed to git:

```powershell
sam build; sam deploy --guided --profile zachfire9
```

Use placeholder/default settings for fake-data smoke tests only. Do not put real financial data through a public unauthenticated API. Persistent DynamoDB storage is intentionally deferred until after the ephemeral AWS path and access-control requirements are reviewed.

## CORS and deployed access control

The local Vite development workflow still uses the UI dev-server proxy, so CORS can stay disabled by leaving `FINANCIALS_ALLOWED_ORIGINS` blank.

When a static-hosted UI needs to call this API directly, set placeholder-style allowed origins in local/private runtime config:

```powershell
$env:FINANCIALS_ALLOWED_ORIGINS="https://<static-ui-host.example>"; go run ./cmd/api
```

Multiple origins can be comma-separated:

```env
FINANCIALS_ALLOWED_ORIGINS=https://<static-ui-host.example>,http://localhost:5173
```

For deployed personal-use testing before real financial data, configure a shared access token outside git. When `FINANCIALS_ACCESS_TOKEN` is set, all non-health API requests must include the same value in `X-Financials-Access-Token`:

```powershell
$env:FINANCIALS_ACCESS_TOKEN="<private-access-token>"
```

For SAM deploys, pass the token as a private parameter override along with the allowed static UI origin:

```powershell
sam deploy --stack-name financials-api --profile zachfire9 --capabilities CAPABILITY_IAM --parameter-overrides FinancialsStorageDriver=ephemeral FinancialsAllowedOrigins=https://<cloudfront-domain> FinancialsAccessToken=<private-access-token>
```

This shared-token gate is a pragmatic personal-use blocker, not user identity or a replacement for Cognito/OIDC/Lambda-authorizer auth if the app becomes multi-user or stores real cloud data.

The static UI bundles the token into its JavaScript, so treat it as readable by anyone who loads the site. Cost is bounded by the guardrails below instead.

### Cost guardrails

- **API Gateway throttling:** `FinancialsThrottlingRateLimit` (default `2` requests/second) and `FinancialsThrottlingBurstLimit` (default `5`) apply to every route. The deploy workflow sets them from the `FINANCIALS_THROTTLING_RATE_LIMIT` / `FINANCIALS_THROTTLING_BURST_LIMIT` repo variables. Requests above the limit get `429 Too Many Requests` from API Gateway and never invoke Lambda. Throttles are best-effort targets, not exact ceilings.
- **Lambda reserved concurrency (optional):** set `FinancialsReservedConcurrency` (for example `2`) as a backstop. Leave it blank if the account's Lambda concurrency quota is low, because AWS requires at least 100 unreserved executions to remain.
- **AWS Budgets alerts:** configured at the account level (outside this repo) for actual and forecasted monthly spend. Budgets alert only; they do not stop spending, and billing data lags actual usage.

Requests that pass the throttle but fail the access-token check still invoke Lambda and are billed, so the throttle is what bounds worst-case cost.

Keep real deployed origins, access tokens, private LAN hostnames/IPs, and environment-specific deployment values in ignored `.env` files or private operator notes, not in committed docs.

Check the health endpoint:

```powershell
Invoke-RestMethod http://localhost:8080/health
```

Expected response:

```json
{"status":"ok"}
```

## Financial items API

Financial items are generic projection inputs such as example savings, brokerage, or goal balances. Use fake/example data in committed docs and tests only. `drawdownAnnualReturnRateBasisPoints` is optional; omit it to reuse the regular `annualReturnRateBasisPoints` once drawdown begins. `ownerId` is optional: it names the person in a projection's `people` who owns the account (see [Retiring at different times](#retiring-at-different-times)); omit it for joint accounts. It's an opaque string of at most 64 characters. `withdrawalPriority` and `withdrawalAvailableAfterYear` are optional; see [Withdrawal order](#withdrawal-order).

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

## Errors

Every error response is JSON: `{"error":"<message>"}`.

| Status | When |
|--------|------|
| `400` | Validation failure, unknown JSON field, malformed JSON, or extra data after the JSON body |
| `401` | `FINANCIALS_ACCESS_TOKEN` is set and the request's `X-Financials-Access-Token` is missing or wrong |
| `404` | Unknown route, or unknown item ID |
| `405` | Known route, wrong method (the `Allow` header lists the right ones) |
| `413` | Request body over 1 MiB |
| `500` | Server error. The message is always `internal error`; details go to the server log only |

## JSON backup export/import

`GET /financial-items/backup` exports all saved financial items as a public JSON shape with `schemaVersion`, `exportedAt`, and `items`. Real backup files can contain private financial data; keep them local and out of git.

Export an example backup:

```powershell
Invoke-RestMethod http://localhost:8080/financial-items/backup | ConvertTo-Json -Depth 10 | Set-Content .\financials-backup.example.local.json
```

Import replaces the current saved financial items with the validated backup payload. The import preserves explicit item IDs, sort order, timestamps, and optional drawdown return assumptions when the payload is valid.

Restore checklist:

1. Start the API with the intended storage adapter.
2. Confirm the backup file is local/private and not under source control.
3. Import the JSON backup.
4. List financial items and recalculate projections.

```powershell
$backup = Get-Content .\financials-backup.example.local.json -Raw
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
  "transitionYears": 0,
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
- Drawdown-capable requests use `savingYears` (or per-person `people`, below) plus optional `drawdownYears`; `years` is mutually exclusive with those phase fields.
- `drawdownYears` greater than `0` requires `annualWithdrawalCents`.
- `annualWithdrawalInflationRateBasisPoints` is optional and defaults to `0`; `300` means the requested withdrawal grows by 3% after each projection year, so one saving year makes the first drawdown withdrawal $61,800 from a $60,000 base.
- `drawdownAnnualReturnRateBasisPoints` is optional per item; if omitted, drawdown years use the normal `annualReturnRateBasisPoints`.
- All items in one projection must use the same currency.
- Hypothetical `items` are validated but not saved.
- Unknown JSON fields return `400` to catch request typos.

### Retiring at different times

For a household where people retire in different years, send `people` instead of `savingYears`. Each person has an `id`, a `name` and their own `savingYears` (years until they retire), and each item can set `ownerId` to one of those IDs. Items without an `ownerId` are joint.

```powershell
Invoke-RestMethod http://localhost:8080/projections -Method Post -ContentType 'application/json' -Body '{"people":[{"id":"person-1","name":"Example A","savingYears":1},{"id":"person-2","name":"Example B","savingYears":3}],"drawdownYears":1,"gapAnnualWithdrawalCents":300,"annualWithdrawalCents":600,"items":[{"id":"a","name":"A 401k","amountCents":1000,"currency":"USD","annualReturnRateBasisPoints":0,"annualContributionCents":100,"ownerId":"person-1","sortOrder":1},{"id":"b","name":"B 401k","amountCents":1000,"currency":"USD","annualReturnRateBasisPoints":0,"annualContributionCents":100,"ownerId":"person-2","sortOrder":2},{"id":"j","name":"Joint brokerage","amountCents":1000,"currency":"USD","annualReturnRateBasisPoints":0,"annualContributionCents":100,"sortOrder":3}]}'
```

How it works:

- An item receives contributions until its owner retires; joint items until the last person retires. From the year after its owner retires, an item uses its `drawdownAnnualReturnRateBasisPoints` if set.
- Years up to the first retirement have phase `saving`.
- Years between the first and the last retirement have phase `transition`. Accounts of people still working keep contributing, and the household withdraws `gapAnnualWithdrawalCents` (optional, default `0`) from the items that are available: by default the retired person's accounts and joint accounts (see [Withdrawal order](#withdrawal-order)).
- After the last retirement, the full `annualWithdrawalCents` is withdrawn for `drawdownYears`, with phase `drawdown`.
- Both withdrawal amounts are in today's dollars and grow by `annualWithdrawalInflationRateBasisPoints` every year from year 1.
- The response adds `transitionYears`: `years` = `savingYears` (to the first retirement) + `transitionYears` + `drawdownYears`.
- With `people`, the request's `savingYears` must be `0` and every `ownerId` must match a person. Without `people`, `ownerId` is ignored (every item counts as joint), and `gapAnnualWithdrawalCents` isn't allowed. If everyone retires in the same year, the result matches the household `savingYears` projection.

### Withdrawal order

By default, a withdrawal is split across every item in proportion to its balance at the start of the year. Two optional item fields change that:

- `withdrawalPriority`: `1` (first), `2` (normal, also used when omitted or `0`) or `3` (last). Withdrawals come from the first group until it's empty, then the normal group, then the last. Items in the same group share a withdrawal in proportion to their balances.
- `withdrawalAvailableAfterYear`: the last year the item can't be withdrawn from. Omitted means the default: with `people`, an item with an owner becomes available after its owner retires; otherwise it's available from year 1. `0` makes an item available from year 1 even if its owner is still working.

If the available items can't cover a withdrawal, the rest is reported as `unfundedWithdrawalCents`, as before. In years with withdrawals, `lockedBalanceCents` (on each item and in `totals`) is the end-of-year balance of items that couldn't be withdrawn from, so a shortfall with money still locked away can be told apart from running out.

Requests that set neither field and have no `people` project exactly as before. With `people`, accounts of someone still working are no longer withdrawn from during transition years unless they set `withdrawalAvailableAfterYear: 0`.

### Itemized expenses

Instead of a single `annualWithdrawalCents`, a request can list `expenses`. Each drawdown year's withdrawal is the total of the expenses that apply that year:

- `name` and `annualAmountCents` (today's dollars) are required; `id` is optional.
- `inflationRateBasisPoints` (optional) overrides the request's inflation rate for that expense, for example faster-rising healthcare.
- `fromYear` and `toYear` (optional) limit the expense to a range of drawdown years, inclusive, where `1` is the first year after the last retirement: for example `toYear: 4` for health insurance before Medicare, `fromYear: 25` for long-term care. `0` or omitted leaves that end open. The older `years` field still works as `toYear`.

Each expense grows with its inflation rate every year from year 1, like `annualWithdrawalCents`, including the years before its range starts, so a later expense is still entered in today's dollars. With `expenses`, `annualWithdrawalCents` must be `0`. Expenses apply in drawdown years only; transition years still use `gapAnnualWithdrawalCents`.

### Mortgage

`mortgage` takes `balanceCents` (owed today), `annualRateBasisPoints` (the APR) and `monthlyPaymentCents` (principal and interest; put property tax and insurance in expenses). The mortgage is amortized month by month: each month's interest is the balance times APR/12, rounded to the cent, and the last payment covers only what's left. The payment must be more than the first month's interest.

Payments made in drawdown years are added to that year's withdrawal. Before then they're assumed to come from salary, so they only show in the mortgage results.

The response adds:

- `mortgage.payoffYear`: the year the balance reaches 0, or `0` if it isn't paid off within the projection.
- `mortgage.years`: one entry per year from year 0 with `balanceCents` (owed at year end), `paymentCents`, `interestCents` and `paidFromSavings`.

Every phase projection also returns `spending`: one entry per year from year 1, with the planned `expensesCents` (expenses, or the annual or gap withdrawal) and `mortgagePaymentCents` taken from savings that year. When expenses are itemized, drawdown years also list `expenses`: each expense that applies that year as `{id, name, amountCents}`, in request order, with its inflation included. The amounts add up to `expensesCents`.

## Public repo boundaries

This repo is public, so it must not contain:

- Real financial data
- Secrets, API keys, passwords, or tokens
- LAN IPs, hostnames, router/firewall details, or machine names
- Local database paths, backup paths, or user-specific service names
- Environment-specific production configuration

Use committed examples with placeholders only, and keep real runtime values in ignored local files such as `.env`.
