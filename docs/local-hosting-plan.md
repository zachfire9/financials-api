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
- Projection/calculation engine
- HTTP API contract
- Public-safe deployment docs and examples
- Tests for calculations, request validation, and API behavior
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

Use Go for the backend implementation.

Suggested default:

- Go HTTP API using the standard library plus a small router such as `chi` if routing grows beyond a few endpoints
- Plain Go structs for request/response models and domain entities
- Go's built-in `testing` package for calculation and endpoint tests
- OpenAPI documentation generated or maintained once the first contract stabilizes
- Embedded local NoSQL persistence if saved scenarios are needed
- `.env` loading for local settings, with process environment values taking precedence

## Persistence recommendation

A NoSQL-style store makes sense if the first persisted data is user-created planning scenarios and projection snapshots, because those records are naturally document-shaped and may evolve as assumptions change.

Recommended local-first approach:

- Start with an embedded Go-friendly NoSQL/key-value store such as `bbolt`.
- Store scenario documents as versioned JSON values behind a repository interface.
- Keep calculation logic independent from persistence so the projection engine remains deterministic and easy to test.
- Avoid requiring an external database server for the first local deployment.
- Leave room to swap the repository implementation later for MongoDB, DynamoDB, or another hosted/document database if remote sync or multi-device use becomes important.

If the app quickly needs relational querying, reporting across many scenarios, or ad hoc analytics, revisit SQLite/Postgres. For the initial local-first scenario workflow, embedded NoSQL is a reasonable default.

## Initial API scope

Start with the smallest useful backend contract:

1. Health endpoint
   - `GET /health`
   - Confirms the server is running.

2. Scenario projection endpoint
   - `POST /projections`
   - Accepts a public-safe request model for planning assumptions and account inputs.
   - Returns yearly projection rows and summary metrics.

3. Example data only
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
FINANCIALS_DATA_PATH=./data/financials.bbolt
```

Private/operator docs may define the real values for a specific machine or LAN. Those details should stay outside git.

## Suggested staged PR plan

### Step 1: Planning baseline

- Remove the old prototype application files.
- Add this public-safe architecture/local-hosting plan.
- Update the README to explain the repo reset.

### Step 2: Go API skeleton

- Add the Go module and backend project structure.
- Add dependency management.
- Add a health endpoint.
- Add Go test tooling and one passing endpoint test.

### Step 3: Domain model and projection tests

- Define the first request/response models.
- Add deterministic projection fixtures.
- Implement the calculation engine test-first.

### Step 4: Projection endpoint

- Wire the calculation engine into `POST /projections`.
- Add endpoint tests for valid input and validation failures.
- Document example requests/responses with fake data.

### Step 5: Local configuration workflow

- Add `.env.example` with placeholders only.
- Confirm `.env` is ignored.
- Document local startup commands and config precedence.

### Step 6: UI contract handoff

- Freeze the initial OpenAPI/response shape enough for UI work.
- Create a sibling UI plan against the concrete API contract.

## Open decisions

- Go HTTP stack/router choice: standard library only vs `chi` as the first router dependency.
- Exact embedded NoSQL store: `bbolt` as the default candidate vs Badger or another Go-native option.
- Whether projections are stateless requests only or saved scenarios from the start.
- Whether authentication is needed for local-only use, and if so which lightweight mechanism fits best.

## Verification expectations

Each implementation step should include:

- Unit tests for calculation logic where applicable
- Endpoint tests for API behavior where applicable
- README updates for new commands
- Public-safe config examples only
- A repo-wide check that no real private details were committed
