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

Use a small Python API unless a later step identifies a stronger reason to choose Go or another stack.

Suggested default:

- FastAPI for HTTP endpoints and OpenAPI generation
- Pydantic models for request/response validation
- Pytest for calculation and endpoint tests
- Local JSON or SQLite persistence depending on the first real data needs
- `.env` loading for local settings, with process environment values taking precedence

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
FINANCIALS_DATA_PATH=./data/dev.db
```

Private/operator docs may define the real values for a specific machine or LAN. Those details should stay outside git.

## Suggested staged PR plan

### Step 1: Planning baseline

- Remove the old prototype application files.
- Add this public-safe architecture/local-hosting plan.
- Update the README to explain the repo reset.

### Step 2: API skeleton

- Add the backend project structure.
- Add dependency management.
- Add a health endpoint.
- Add test tooling and one passing endpoint test.

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

- Python/FastAPI vs Go for the backend implementation.
- JSON file vs SQLite for initial persistence.
- Whether projections are stateless requests only or saved scenarios from the start.
- Whether authentication is needed for local-only use, and if so which lightweight mechanism fits best.

## Verification expectations

Each implementation step should include:

- Unit tests for calculation logic where applicable
- Endpoint tests for API behavior where applicable
- README updates for new commands
- Public-safe config examples only
- A repo-wide check that no real private details were committed
