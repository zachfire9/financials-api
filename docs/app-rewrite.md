# Financials App Local Hosting Plan

## Goal

Rebuild the old Financials API/UI pair as a local-first personal financial planning app while keeping the public repositories general-purpose and safe to share.

The API repo should own the core domain model, calculation logic, API contract, and backend deployment shape. The UI repo should consume the API contract once the backend response shapes are stable.

## Guiding principles

- Keep the application useful as a generic financial planning tool.
- Keep private home-network, machine, and financial details out of git.
- Commit placeholders and examples only.
- Prefer a simple local deployment path before adding optional production-grade complexity.
- Treat API contract design as the foundation for the sibling UI rewrite.

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

Detailed UI planning should wait until the API contract is stable enough to avoid rework.

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

Start with the smallest useful backend contract for configurable financial items. Projections should wait until the financial item model and UI workflow are built out.

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

3. Projection planning
   - Use financial items as the projection input foundation.
   - Keep v1 deterministic: whole years, annual compounding, end-of-year contributions, per-item series, and aggregate totals.
   - See [Projection Planning](projection-planning.md) for proposed request/response shapes and the next staged implementation steps.

4. Example data only
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
- **Pull Request:** TBD
- Use the completed financial item model as the input foundation for projection planning.
- Define projection request/response shapes after financial item CRUD is working.
- Create the staged backend/UI projection plan: Step 7 calculation engine, Step 8 projection API endpoint, and Step 9 sibling UI planning.

### Step 7: Projection calculation engine

- [ ] **Status:** Pending
- **Branch:** TBD
- **Pull Request:** TBD
- Create projection domain models in `internal/projections`.
- Implement deterministic whole-year projection calculations test-first.
- Cover per-item yearly balances, aggregate totals, validation, currency mismatches, negative return assumptions, and rounding behavior.

### Step 8: Projection API endpoint

- [ ] **Status:** Pending
- **Branch:** TBD
- **Pull Request:** TBD
- Add `POST /projections` to the HTTP handler tree.
- Support repository-backed projections when `items` is omitted.
- Support caller-supplied hypothetical items without saving them.
- Document fake/example request and response payloads.

### Step 9: Sibling UI planning

- [ ] **Status:** Pending
- **Branch:** TBD
- **Pull Request:** TBD
- Create the detailed `financials-ui` plan against the concrete financial items and projection API contracts.
- Plan a React UI with placeholder-only config, typed API client boundaries, and stale-data handling.

## Open decisions

- Go HTTP stack/router choice: standard library only vs `chi` as the first router dependency.
- Local storage adapter choice: DynamoDB Local for AWS parity vs simple JSON/file adapter for lowest-friction local development.
- Initial financial item fields are set: name, amount, currency, annual return rate basis points, annual contribution, sort order, ID, and timestamps.
- Whether financial item deletion is needed immediately or whether archive/inactive status is safer.
- Whether authentication is needed for local-only use, and if so which lightweight mechanism fits best.
- Projection v1 request/response shape is proposed in `docs/projection-planning.md`; review open questions before implementation.
- Projection v1 contribution timing default is end-of-year unless Zach chooses otherwise.

## Verification expectations

Each implementation step should include:

- Unit tests for calculation logic where applicable
- Endpoint tests for API behavior where applicable
- README updates for new commands
- Public-safe config examples only
- A repo-wide check that no real private details were committed
