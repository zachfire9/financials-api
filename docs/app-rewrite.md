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

- [ ] **Status:** In progress — backend/API and UI support on current Step 20 branches.
- **Branch:** `step-20-json-backup-export-import` / `step-20-json-backup-ui`
- **Pull Request:** TBD
- Add a public-safe JSON backup workflow so all saved financial data can be exported, saved locally, and re-imported after memory/file storage is cleared.
- Backend scope: add tested export/import endpoints for financial items using the existing repository boundary; validate import payload shape, reject malformed data, preserve explicit IDs/sort order/timestamps/drawdown return assumptions where safe, and make import replacement semantics explicit.
- UI scope: add export/download and import/upload controls that use JSON files only, show success/error states, and refresh the financial-items list plus projections after import.
- Docs scope: document fake/example backup files only, warn that real financial backup JSON should stay out of git, and include a restore checklist.

## Open decisions

- Go HTTP stack/router choice: standard library only vs `chi` as the first router dependency.
- Local storage adapter choice: DynamoDB Local for AWS parity vs simple JSON/file adapter for lowest-friction local development.
- Initial financial item fields are set: name, amount, currency, annual return rate basis points, annual contribution, sort order, ID, and timestamps.
- Whether financial item deletion is needed immediately or whether archive/inactive status is safer.
- Whether authentication is needed for local-only use, and if so which lightweight mechanism fits best.
- UI stack recommendation: Vite + React + TypeScript, with static build output suitable for AWS Amplify later.
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
