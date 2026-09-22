# Financials App Local Hosting Plan

## Goal

Rebuild the old Financials API/UI pair as a local-first personal financial planning app while keeping the public repositories general-purpose and safe to share.

The API repo should own the core domain model, calculation logic, API contract, and backend deployment shape. The UI repo should also track the cross-repo sequencing so the sibling UI can be rebuilt against the working financial-items API before projection logic is implemented.

## Guiding principles

- Keep the application useful as a generic financial planning tool.
- Keep private home-network, machine, and financial details out of git.
- Commit placeholders and examples only.
- Prefer a simple local deployment path before adding optional production-grade complexity.
- Treat API contract design as the foundation for the sibling UI rewrite.
- Exercise the existing financial-items API through a very small UI before adding projection calculation complexity.

## Repository split

### financials-api

The API repo should contain:

- Domain model and validation rules
- Generic financial item tracking API and storage model
- Projection/calculation engine after the financial item workflow is built out
- HTTP API contract
- Public-safe deployment docs and examples
- Tests for financial item CRUD, calculations, request validation, and API behavior
- Local configuration template files such as `.env.example`

The API repo should not contain:

- Real financial account values
- Real LAN hostnames/IPs/ports
- Machine-specific paths
- Actual `.env` files
- Private backup destinations

### financials-ui

The UI repo should contain:

- React frontend app
- API client typed against the backend contract
- Public-safe local development instructions
- Placeholder configuration examples

Repurpose the old Django/Heroku-era UI repo as a lightweight Vite + React + TypeScript single-page app. Start with financial item CRUD only, then add projection screens after the projection API exists. Keep the UI build compatible with static hosting targets such as AWS Amplify by using placeholder environment variables for API base URLs and committing no private runtime values.

## Recommended backend shape

Use Go for the backend implementation and choose libraries that can run locally now but map cleanly to AWS later.

Suggested default:

- Go HTTP API using the standard library first, with a small router such as `chi` only if routing grows beyond a few endpoints
- Handler/core split that can run behind a normal HTTP server locally and an AWS Lambda/API Gateway adapter later
- Plain Go structs for request/response models and domain entities
- Go's built-in `testing` package for calculation and endpoint tests
- OpenAPI documentation generated or maintained once the first contract stabilizes
- Persistence behind a small repository interface so local storage and AWS storage are adapters, not domain concerns
- `.env` loading for local settings, with process environment values taking precedence

## Persistence recommendation

A NoSQL-style store makes sense if the first persisted data is generic financial planning inputs, because each financial item record is document-shaped and the fields may evolve as balances, return assumptions, contribution metadata, and UI ordering get refined. It should also support later projection scenarios without forcing a storage rewrite.

Recommended AWS-swappable approach:

- Model persistence around DynamoDB-style access patterns from the start: financial item ID, user/owner scope if needed later, name, amount, return assumptions, contribution metadata, sort order, created/updated timestamps, and versioned JSON documents.
- Keep a `FinancialItemRepository` interface in the application layer first; add a scenario/projection repository later only when projections need saved inputs or outputs.
- Implement local development with either DynamoDB Local for closest AWS parity or a simple file/embedded adapter for convenience.
- Treat DynamoDB as the likely AWS target if/when the app moves from local-only hosting to an AWS-backed deployment.
- Keep calculation logic independent from persistence so the projection engine remains deterministic and easy to test.
- Avoid designing around database-specific query features until the app has real access patterns.

If the app quickly needs relational querying, reporting across many financial items, or ad hoc analytics, revisit SQLite/Postgres/Aurora. For the initial local-first financial item inventory with an AWS migration path, a DynamoDB-shaped document model is a reasonable default.

## Initial API scope

Start with the smallest useful backend contract for configurable financial items. The next validation step is a very small UI that exercises those endpoints locally before projection logic is built.

1. Health endpoint
   - `GET /health`
   - Confirms the server is running.

2. Financial items endpoints
   - `GET /financial-items`
   - `POST /financial-items`
   - `GET /financial-items/{id}`
   - `PUT /financial-items/{id}`
   - `DELETE /financial-items/{id}` if deletion is useful for local cleanup.
   - Tracks fake/example-safe fields first: name, amount, currency, annual return rate in basis points, annual contribution, sort order, and timestamps.

3. UI-driven API smoke testing
   - Repurpose `financials-ui` into a Vite + React + TypeScript app.
   - Use the existing financial-items endpoints for list/create/update/delete flows.
   - Run the UI and API on the development machine with placeholder bind addresses and verify another device on the same private network can access the UI.
   - Use a dev proxy first to avoid adding CORS before it is needed.

4. Projection planning
   - Use financial items as the projection input foundation.
   - Keep v1 deterministic: whole years, annual compounding, end-of-year contributions, per-item series, and aggregate totals.
   - See [Projection Planning](projection-planning.md) for proposed request/response shapes and the later staged implementation steps.

5. Example data only
   - Include fake example requests/responses.
   - Do not include real household data.

## Local hosting approach

Document local hosting generically in the public repo:

- Run the API as a normal local service.
- Bind to configurable host/port placeholders.
- Put real runtime values in ignored local config.
- Keep firewall, router, hostname, and backup specifics in private notes only.

Public docs may use examples like:

```env
FINANCIALS_API_HOST=127.0.0.1
FINANCIALS_API_PORT=8000
FINANCIALS_STORAGE_DRIVER=local
FINANCIALS_TABLE_NAME=financials-items-dev
FINANCIALS_DATA_PATH=./data/financial-items.json
```

For DynamoDB Local parity testing, use the same table-oriented configuration with a local endpoint override in an ignored `.env` file.

Private/operator docs may define the real values for a specific machine or LAN. Those details should stay outside git.

For the initial UI smoke test, run both services on the development machine with configurable placeholder bind addresses. The committed docs should describe the pattern generically, for example `http://<dev-machine-private-ip>:<ui-port>`, without committing the real home-network address. The Vite dev server can proxy `/api/*` to the local API during development; later static hosting such as AWS Amplify will need an explicit API base URL and API CORS support.

## Suggested staged PR plan

Track each step as a living checklist. Each implementation PR should update this section with the branch, PR link, and status so `master` reflects completed work after Zach merges the PR.

### Step 1: Planning baseline

- [x] **Status:** Completed
- **Branch:** `docs/reset-api-plan`
- **Pull Request:** [#1](https://github.com/zachfire9/financials-api/pull/1)
- Remove the old prototype application files.
- Add this public-safe architecture/local-hosting plan.
- Update the README to explain the repo reset.

### Step 2: Go API skeleton

- [x] **Status:** Completed
- **Branch:** `step-02-go-api-skeleton`
- **Pull Request:** [#2](https://github.com/zachfire9/financials-api/pull/2)
- Add the Go module and backend project structure.
- Add dependency management.
- Add a health endpoint.
- Add Go test tooling and one passing endpoint test.

### Step 3: Financial item model and tests

- [x] **Status:** Completed
- **Branch:** `step-03-current-investment-model`
- **Pull Request:** [#3](https://github.com/zachfire9/financials-api/pull/3)
- Define the first configurable financial item request/response models.
- Add deterministic fake financial item fixtures.
- Implement financial item validation and repository behavior test-first.

### Step 4: Financial items API

- [x] **Status:** Completed
- **Branch:** `step-04-financial-items-api`
- **Pull Request:** [#4](https://github.com/zachfire9/financials-api/pull/4)
- Wire the financial item repository into `/financial-items` endpoints.
- Add endpoint tests for create/list/read/update/delete behavior and validation failures.
- Document example requests/responses with fake data.

### Step 5: Local configuration and storage workflow

- [x] **Status:** Completed
- **Branch:** `step-05-local-config-storage`
- **Pull Request:** [#5](https://github.com/zachfire9/financials-api/pull/5)
- Add `.env.example` with placeholders only.
- Confirm `.env` is ignored.
- Add the first repository adapter behind an interface.
- Document local startup commands, storage driver selection, and config precedence.

### Step 6: Projection planning

- [x] **Status:** Completed
- **Branch:** `step-06-projection-planning`
- **Pull Request:** [#6](https://github.com/zachfire9/financials-api/pull/6)
- Use the completed financial item model as the input foundation for projection planning.
- Define projection request/response shapes after financial item CRUD is working.
- Create the staged backend/UI plan, now intentionally putting a basic `financials-ui` rebuild and local-network smoke test before projection implementation.

### Step 7: Repurpose `financials-ui` as a basic React app shell

- [x] **Status:** Completed
- **Branch:** `step-07-react-app-shell`
- **Pull Request:** [financials-ui #1](https://github.com/zachfire9/financials-ui/pull/1)
- Replace the old Django/Heroku-era UI with a Vite + React + TypeScript app.
- Keep the first UI branch focused on project scaffolding, public-safe config examples, local run/build commands, and a minimal app shell.
- Use static-hosting-friendly conventions so the app can later run in AWS Amplify (`npm run build` output in `dist/`).

### Step 8: Wire UI to the financial-items API

- [x] **Status:** Completed
- **Branch:** `step-08-financial-items-crud`
- **Pull Request:** [financials-ui #2](https://github.com/zachfire9/financials-ui/pull/2)
- Add a typed API client for the existing `/financial-items` contract.
- Implement list/create/update/delete flows against the running local API.
- Add basic loading, empty, validation-error, and stale-data/transient-error handling.
- Use fake/example data in tests and docs only.

### Step 9: Local home-network smoke test

- [x] **Status:** Completed
- **Branch:** `step-09-local-network-smoke-test`
- **Pull Request:** [financials-ui #3](https://github.com/zachfire9/financials-ui/pull/3)
- Run the API and UI dev servers on the development machine using placeholder bind-address documentation.
- Configure the UI dev proxy so browser calls can go through the UI server during local testing.
- Verify another device on the same private network can load the UI and exercise financial item CRUD.
- Keep real LAN addresses, hostnames, firewall/router details, and machine-specific notes out of git.

### Step 10: Financial-items UI polish

- [x] **Status:** Completed
- **Branch:** `step-10-ui-polish-drag-sort`
- **Pull Request:** [financials-ui #4](https://github.com/zachfire9/financials-ui/pull/4)
- Remove the intro/hero container.
- Keep sort order hidden from the form and support drag-and-drop reordering.
- Display API-backed cents values as human-readable dollar inputs while editing.

### Step 11: API CORS and deploy-readiness prep

- [x] **Status:** Completed
- **Branch:** `step-11-api-cors-deploy-readiness`
- **Pull Request:** [financials-api #7](https://github.com/zachfire9/financials-api/pull/7)
- Add API CORS support and configuration only after the local proxy-based UI workflow is proven.
- Document placeholder allowed-origin settings for future static hosting such as AWS Amplify.
- Keep real deployed origins and private runtime values in ignored local config or private operator notes.

### Step 12: Projection calculation engine

- [x] **Status:** Completed
- **Branch:** `step-12-projection-calculation-engine`
- **Pull Request:** [financials-api #8](https://github.com/zachfire9/financials-api/pull/8)
- Create projection domain models in `internal/projections`.
- Implement deterministic whole-year projection calculations test-first.
- Cover per-item yearly balances, aggregate totals, validation, currency mismatches, negative return assumptions, and rounding behavior.

### Step 13: Projection API endpoint

- [x] **Status:** Completed
- **Branch:** `step-13-projection-api-endpoint`
- **Pull Request:** [financials-api #9](https://github.com/zachfire9/financials-api/pull/9)
- Add `POST /projections` to the HTTP handler tree.
- Support repository-backed projections when `items` is omitted.
- Support caller-supplied hypothetical items without saving them.
- Document fake/example request and response payloads.

### Step 14: Projection UI

- [x] **Status:** Completed
- **Branch:** `step-14-projection-ui`
- **Pull Request:** [financials-ui #5](https://github.com/zachfire9/financials-ui/pull/5)
- Extend `financials-ui` with projection request controls and an early table view once the projection API exists.
- Keep typed API client boundaries, placeholder-only config, and stale-data handling.
- Present projection rows grouped by year, with item balances and the combined balance at the far right.

### Step 15: Drawdown projection planning

- [x] **Status:** Completed
- **Branch:** `step-15-drawdown-projection-planning`
- **Pull Request:** [#10](https://github.com/zachfire9/financials-api/pull/10)
- Define the next projection contract for explicit saving years plus optional drawdown years.
- Capture open decisions for withdrawal timing, allocation, depletion behavior, and default horizons before implementation.
- Add follow-up backend/API/UI steps for the drawdown-capable projection workflow.

### Step 16: Drawdown calculation engine

- [x] **Status:** Completed
- **Branch:** `step-16-drawdown-calculation-engine`
- **Pull Request:** [#11](https://github.com/zachfire9/financials-api/pull/11)
- Extend `internal/projections` test-first with explicit saving/drawdown phases, optional per-item drawdown return rates, and inflation-adjusted drawdown withdrawals.
- Preserve v1 accumulation-only behavior while adding annual withdrawals and phase-aware yearly totals.
- Keep HTTP handler wiring out of this step.

### Step 17: Drawdown projection API contract

- [x] **Status:** Completed
- **Branch:** `step-17-drawdown-projection-api-contract`
- **Pull Request:** [#12](https://github.com/zachfire9/financials-api/pull/12)
- Extend `POST /projections` request/response handling for `savingYears`, optional `drawdownYears`, `annualWithdrawalCents`, `annualWithdrawalInflationRateBasisPoints`, and drawdown-specific validation.
- Preserve existing `years` requests for the current UI until the UI migrates.
- Document fake/example payloads only.

### Step 18: Drawdown projection UI

- [x] **Status:** Completed
- **Branch:** `step-18-drawdown-projection-ui`
- **Pull Request:** [financials-ui #6](https://github.com/zachfire9/financials-ui/pull/6)
- Add saving-years, drawdown-years, annual-withdrawal, and withdrawal-inflation controls to `financials-ui` after the API supports them.
- Show phase labels in the existing year-grouped projection results.
- Keep charts optional until the drawdown table workflow is proven.

### Step 19: Per-item drawdown return assumptions

- [x] **Status:** Completed
- **Branch:** `step-19-per-item-drawdown-return-assumptions` / `step-19-per-item-drawdown-return-ui`
- **Pull Request:** [financials-api #14](https://github.com/zachfire9/financials-api/pull/14), [financials-ui #7](https://github.com/zachfire9/financials-ui/pull/7)
- Add a persisted optional drawdown return rate per financial item so each item can use one return assumption while saving and a different return assumption once drawdown begins.
- Backend scope: extend financial item request/response/storage models with optional `drawdownAnnualReturnRateBasisPoints`; validate it with the same basis-point bounds as accumulation return; keep existing items compatible by falling back to `annualReturnRateBasisPoints` when omitted.
- Projection scope: pass the saved per-item drawdown return into repository-backed `POST /projections` requests; preserve the already-supported hypothetical `items[].drawdownAnnualReturnRateBasisPoints` behavior.
- UI scope: add an optional "Drawdown return (%)" field to create/edit forms and projection display, with clear fallback wording when blank.
- Docs scope: update fake examples only; do not commit real account-specific return assumptions.

### Step 20: JSON backup export/import

- [x] **Status:** Completed
- **Branch:** `step-20-json-backup-export-import` / `step-20-json-backup-ui`
- **Pull Request:** [financials-api #15](https://github.com/zachfire9/financials-api/pull/15), [financials-ui #8](https://github.com/zachfire9/financials-ui/pull/8)
- Add a public-safe JSON backup workflow so all saved financial data can be exported, saved locally, and re-imported after memory/file storage is cleared.
- Backend scope: add tested export/import endpoints for financial items using the existing repository boundary; validate import payload shape, reject malformed data, preserve explicit IDs/sort order/timestamps/drawdown return assumptions where safe, and make import replacement semantics explicit.
- UI scope: add export/download and import/upload controls that use JSON files only, show success/error states, and refresh the financial-items list plus projections after import.
- Docs scope: document fake/example backup files only, warn that real financial backup JSON should stay out of git, and include a restore checklist.

### Step 21: Optional annual contribution inflation

- [x] **Status:** Completed
- **Branch:** `step-21-annual-contribution-inflation` / `step-21-annual-contribution-inflation-ui`
- **Pull Request:** [financials-api #16](https://github.com/zachfire9/financials-api/pull/16), [financials-ui #9](https://github.com/zachfire9/financials-ui/pull/9)
- Add an item-level option for annual contributions to grow by the projection's configured withdrawal inflation rate.
- Backend scope: extend financial item and projection item handling with `inflateAnnualContribution`; when enabled on an item, apply `annualWithdrawalInflationRateBasisPoints` to that item's annual contribution after each projection year during saving years, while preserving fixed-contribution behavior for omitted or false items.
- UI scope: add the checkbox to the financial item create/edit form and item summaries, not the projection controls, so users can choose which accounts (for example 401k vs IRA) match inflation.
- Test scope: add RED/GREEN projection-engine and endpoint tests covering fixed contributions by default, inflated contributions when enabled, rounding behavior, and interaction with `savingYears`/drawdown boundaries.
- Docs scope: update fake projection examples and explain that this is a projection setting, not a persisted change to the saved financial item amount.

### Step 22: Ephemeral import/export session mode

- [x] **Status:** Completed
- **Branch:** `step-22-ephemeral-session-mode` / `step-22-ephemeral-session-ui`
- **Pull Request:** API [#17](https://github.com/zachfire9/financials-api/pull/17) / UI [#10](https://github.com/zachfire9/financials-ui/pull/10)
- Add a browser-owned, non-persistent mode for privacy-first AWS usage where users import a local JSON backup, work with the data in React state, and export JSON again before closing the browser if they want to keep changes.
- Backend scope: add explicit `FINANCIALS_STORAGE_DRIVER=ephemeral` support as a non-durable runtime signal for stateless/request-supplied projection workflows; keep Lambda/process memory out of the deployed persistence story because Lambda containers are reused, discarded, and scaled independently of browser sessions.
- UI scope: when `VITE_FINANCIALS_SESSION_MODE=ephemeral` is enabled, load items from JSON import into browser state, perform create/edit/delete/reorder locally, send the current in-memory items as caller-supplied `items` in `POST /projections`, and provide a clear export/download path for saving changes.
- UX scope: show explicit copy that refresh/close loses unsaved session data in ephemeral mode; hide or disable API-backed save/load controls so the user does not confuse browser memory with durable storage.
- Test scope: cover import, local edit/delete/reorder behavior, projection requests with request-body items, export output, and stale/error states using fake data only.
- Docs scope: describe this as a privacy/cost option for deployed/static hosting that avoids server-side storage of financial data; do not present Lambda in-memory storage as reliable session storage.

### Step 23: AWS serverless ephemeral backend

- [x] **Status:** Completed
- **Branch:** `step-23-aws-serverless-ephemeral-backend`
- **Pull Request:** [#18](https://github.com/zachfire9/financials-api/pull/18)
- Add the cost-effective ephemeral AWS backend path using AWS SAM, API Gateway HTTP API, and a Go Lambda function.
- Architecture recommendation: browser calls API Gateway HTTP API, API Gateway invokes the Go Lambda handler, and the UI sends request-supplied projection items for browser-owned sessions instead of relying on Lambda/process memory as session storage.
- Backend scope: add a Lambda entrypoint and HTTP API v2 adapter that reuse the existing `net/http` handler and repository interface; keep local `memory`/`json` drivers unchanged for development.
- Infrastructure scope: add a SAM template with the Lambda function, HTTP API routes, public-safe parameters, and outputs only; do not add DynamoDB until persistent deployed storage is intentionally chosen later.
- Cost recommendation: prefer Lambda + HTTP API with no database for the first AWS ephemeral path; avoid EC2, ECS/Fargate, App Runner, RDS, and DynamoDB until the app needs server-side persistence or access patterns that justify the additional service.
- Security scope: do not deploy real financial data to a public unauthenticated API; include placeholder-only configuration for allowed origins and authentication settings, with real values kept outside git.
- Test/verification scope: run Go tests, compile the Lambda bootstrap, validate/build with SAM where available, and use fake data only for deployed smoke tests.

### Step 24: Static frontend AWS deploy workflow

- [x] **Status:** Completed in UI repo; API plan tracking update pending merge
- **Branch:** `step-24-static-frontend-aws-deploy` / API tracking branch `step-24-static-frontend-aws-deploy-tracking`
- **Pull Request:** [financials-ui #11](https://github.com/zachfire9/financials-ui/pull/11) / API tracking [#19](https://github.com/zachfire9/financials-api/pull/19)
- Add a low-cost static hosting workflow for the Vite UI using S3 plus CloudFront, with the API base URL supplied through environment-specific build/deploy configuration.
- UI scope: make the production build consume a placeholder API base URL for the deployed API, keep local dev proxy behavior unchanged, document how ephemeral mode vs persistent mode changes frontend behavior, and keep a placeholder-only `.env.production.example` while ignoring real `.env.production` values.
- Infrastructure/deploy scope: add a UI-side PowerShell deploy script that syncs `dist/` to a provided S3 bucket and optionally creates a CloudFront invalidation; keep real bucket names, distribution IDs, custom domains, API URLs, and credentials out of committed docs unless they are intentionally public-safe placeholders.
- Verification scope: run `npm test`, `npm run build`, `npm audit --omit=dev --audit-level=moderate`, and a static preview smoke test against fake data / placeholder deployment config.
- Cost recommendation: keep S3/CloudFront as the initial static hosting path because the baseline cost is near zero for small personal traffic and it fits explicit AWS infrastructure planning; keep Amplify Hosting documented as a later migration option if its familiar workflow and GitHub-connected deploys become worth the extra service abstraction.

### Step 25: SAM-managed frontend hosting infrastructure

- [ ] **Status:** Pending
- **Branch:** `step-25-frontend-sam-hosting-infra`
- **Pull Request:** TBD
- Add deployable AWS SAM/CloudFormation infrastructure for the static UI so the S3 bucket, CloudFront distribution, origin access control, bucket policy, SPA fallback behavior, and stack outputs are versioned instead of created manually.
- Infrastructure scope: add a UI repo `template.yaml` using plain CloudFormation resources under SAM, including a private S3 bucket for `dist/`, CloudFront Origin Access Control, a CloudFront distribution, a bucket policy that allows only CloudFront reads, and outputs for the bucket name, distribution ID, and CloudFront URL.
- Configuration scope: add placeholder-safe deploy docs and either a `samconfig.example.toml` or documented `sam deploy --guided --profile zachfire9` workflow; keep real stack names, bucket names, domains, API URLs, distribution IDs, and credentials out of committed files unless intentionally public-safe.
- Deploy scope: keep SAM responsible for infrastructure only, and keep the existing UI PowerShell script responsible for `npm run build` asset sync and optional CloudFront invalidation using the SAM stack outputs.
- Optional domain scope: leave ACM certificate and Route 53 alias parameters optional/later unless a custom domain is chosen; the default first deploy can use the generated CloudFront domain.
- Verification scope: validate the SAM template where tooling is available, run `npm test`, run `npm run build`, smoke-test the static output locally, and confirm the stack outputs provide everything needed by `scripts/deploy-static.ps1`.
- Cost recommendation: continue with private S3 + CloudFront as the lowest-complexity, low-cost production-ish path; avoid Amplify Hosting until GitHub-connected branch deploys become worth the extra abstraction.

### Step 26: Deployed access control before real data

- [ ] **Status:** Pending
- **Branch:** `step-26-deployed-access-control`
- **Pull Request:** TBD
- Add an explicit deployed access-control step before storing or processing real financial data through the AWS-hosted app.
- Recommendation: start with API Gateway API key + usage plan as the smallest acceptable first protection for personal fake-data testing, then move to Cognito, OIDC, or another stronger identity flow if the app becomes multi-user or internet-facing beyond personal testing.
- Backend/static hosting scope: add API Gateway API key enforcement and a usage plan for the Lambda HTTP API path, document that this is a pragmatic gate rather than true user identity auth, pass any frontend/runtime secret values outside git, and keep all real keys out of public docs.
- Verification scope: prove unauthenticated requests fail, authenticated fake-data requests pass, the static frontend can be configured with protected API access outside git, and both persistent DynamoDB mode and ephemeral browser-owned mode remain clear to users.

## Open decisions

- Go HTTP stack/router choice: standard library only vs `chi` as the first router dependency.
- Local storage adapter choice: simple JSON/file storage is implemented for lowest-friction local development; the first AWS path is ephemeral/no-database, and DynamoDB remains the likely adapter if persistent deployed storage is added later.
- Initial financial item fields are set: name, amount, currency, annual return rate basis points, annual contribution, sort order, ID, timestamps, optional drawdown return rate, and optional contribution-inflation flag.
- Whether financial item deletion is needed immediately or whether archive/inactive status is safer.
- Deployed authentication/access control is required before using real financial data in AWS; the first planned protection is API Gateway API key + usage plan as a pragmatic personal-use gate, with Cognito/OIDC/Lambda authorizer or equivalent identity available later if the app becomes multi-user.
- UI stack recommendation: Vite + React + TypeScript, with static build output suitable for S3/CloudFront or AWS Amplify later.
- Local UI/API smoke testing should use placeholder bind-address docs and keep real LAN details out of git.
- Projection v1 request/response shape is implemented for accumulation-only projections.
- Drawdown v2 questions are proposed in `docs/drawdown-projection-planning.md`, including per-item drawdown return rates, withdrawal inflation, withdrawal timing, allocation order, depletion behavior, contribution behavior during drawdown, and default drawdown horizon.

## Verification expectations

Each implementation step should include:

- Unit tests for calculation logic where applicable
- Endpoint tests for API behavior where applicable
- README updates for new commands
- Public-safe config examples only
- A repo-wide check that no real private details were committed
