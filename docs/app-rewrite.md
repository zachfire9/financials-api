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
- Current-investment tracking API and storage model
- Projection/calculation engine after the investment inventory is built out
- HTTP API contract
- Public-safe deployment docs and examples
- Tests for investment CRUD, calculations, request validation, and API behavior
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

A NoSQL-style store makes sense if the first persisted data is the current investment inventory, because each investment record is document-shaped and the fields may evolve as account types, holdings, balances, and metadata get refined. It should also support later projection inputs without forcing a storage rewrite.

Recommended AWS-swappable approach:

- Model persistence around DynamoDB-style access patterns from the start: investment ID, user/owner scope if needed later, investment/account type, institution/name, created/updated timestamps, and versioned JSON documents.
- Keep an `InvestmentRepository` interface in the application layer first; add a scenario/projection repository later only when projections need saved inputs or outputs.
- Implement local development with either DynamoDB Local for closest AWS parity or a simple file/embedded adapter for convenience.
- Treat DynamoDB as the likely AWS target if/when the app moves from local-only hosting to an AWS-backed deployment.
- Keep calculation logic independent from persistence so the projection engine remains deterministic and easy to test.
- Avoid designing around database-specific query features until the app has real access patterns.

If the app quickly needs relational querying, reporting across many investments, or ad hoc analytics, revisit SQLite/Postgres/Aurora. For the initial local-first investment inventory with an AWS migration path, a DynamoDB-shaped document model is a reasonable default.

## Initial API scope

Start with the smallest useful backend contract for current investments. Projections should wait until the investment inventory model and UI workflow are built out.

1. Health endpoint
   - `GET /health`
   - Confirms the server is running.

2. Current investments endpoints
   - `GET /investments`
   - `POST /investments`
   - `GET /investments/{id}`
   - `PUT /investments/{id}`
   - `DELETE /investments/{id}` if deletion is useful for local cleanup.
   - Tracks fake/example-safe fields first: name, type/category, institution label, balance/value, contribution metadata, and timestamps.

3. Projection placeholder only
   - Keep projection concepts in the plan, but do not build `POST /projections` until investment entry/storage is working.
   - Avoid locking projection request/response shapes before the investment model settles.

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
FINANCIALS_TABLE_NAME=financials-investments-dev
FINANCIALS_DATA_PATH=./data/investments.json
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

### Step 3: Current investment model and tests

- [x] **Status:** Completed
- **Branch:** `step-03-current-investment-model`
- **Pull Request:** TBD
- Define the first investment request/response models.
- Add deterministic fake investment fixtures.
- Implement investment validation and repository behavior test-first.

### Step 4: Current investments API

- [ ] **Status:** Pending
- **Branch:** TBD
- **Pull Request:** TBD
- Wire the investment repository into `/investments` endpoints.
- Add endpoint tests for create/list/read/update/delete behavior and validation failures.
- Document example requests/responses with fake data.

### Step 5: Local configuration and storage workflow

- [ ] **Status:** Pending
- **Branch:** TBD
- **Pull Request:** TBD
- Add `.env.example` with placeholders only.
- Confirm `.env` is ignored.
- Add the first repository adapter behind an interface.
- Document local startup commands, storage driver selection, and config precedence.

### Step 6: Projection planning

- [ ] **Status:** Pending
- **Branch:** TBD
- **Pull Request:** TBD
- Use the completed investment model as the input foundation for projection planning.
- Define projection request/response shapes after current investments are working.
- Create a sibling UI plan against the concrete investment API first, then extend it for projections when the API contract is ready.

## Open decisions

- Go HTTP stack/router choice: standard library only vs `chi` as the first router dependency.
- Local storage adapter choice: DynamoDB Local for AWS parity vs simple JSON/file adapter for lowest-friction local development.
- Initial investment fields and account categories for the current-investments workflow.
- Whether investment deletion is needed immediately or whether archive/inactive status is safer.
- Whether authentication is needed for local-only use, and if so which lightweight mechanism fits best.
- Projection assumptions, calculation behavior, and `POST /projections` contract after the current investment workflow is built out.

## Verification expectations

Each implementation step should include:

- Unit tests for calculation logic where applicable
- Endpoint tests for API behavior where applicable
- README updates for new commands
- Public-safe config examples only
- A repo-wide check that no real private details were committed
