# Projection Planning

## Goal

Define the backend contract for deterministic v1 projections now that financial item CRUD, local storage, and the basic UI workflow are in place.

This document is the projection design/spec reference. The active step checklist, implementation status, branch names, and PR links live in [App Rewrite Plan](app-rewrite.md).

## Current foundation

The projection feature should build on the existing financial item shape:

```json
{
  "id": "item_000001",
  "name": "Example brokerage",
  "amountCents": 1250000,
  "currency": "USD",
  "annualReturnRateBasisPoints": 700,
  "annualContributionCents": 300000,
  "sortOrder": 1,
  "createdAt": "2026-01-01T00:00:00Z",
  "updatedAt": "2026-01-01T00:00:00Z"
}
```

The first projection version should treat each financial item as a simple compounding input:

- `amountCents`: starting balance
- `annualReturnRateBasisPoints`: nominal yearly return assumption
- `annualContributionCents`: recurring contribution added once per projected year
- `currency`: response currency, initially require all items in one projection to use the same currency
- `name` and `id`: trace which input produced each projected series

## Recommended v1 projection scope

Keep v1 deterministic and intentionally small:

- Project by whole years only.
- Use annual compounding.
- Add the annual contribution at the end of each projected year.
- Return per-item yearly balances and aggregate yearly totals.
- Do not model taxes, inflation, withdrawals, retirement dates, account categories, or contribution timing variants yet.
- Do not persist projection outputs yet; calculate on demand from current or supplied inputs.

This keeps the calculation easy to test and gives the UI enough structure for a later chart/table after the initial financial-items UI is working.

## Proposed endpoint

`POST /projections`

The endpoint should support two workflows:

1. Use currently stored financial items.
2. Use caller-supplied hypothetical items without saving them.

Request shape:

```json
{
  "years": 10,
  "items": [
    {
      "name": "Example brokerage",
      "amountCents": 1250000,
      "currency": "USD",
      "annualReturnRateBasisPoints": 700,
      "annualContributionCents": 300000,
      "sortOrder": 1
    }
  ]
}
```

Request rules:

- `years` is required and should be bounded, recommended `1` through `75`.
- `items` is optional.
- If `items` is omitted or empty, calculate from the repository's current financial items.
- If `items` is provided, validate it with the same financial item rules and do not save it.
- All projected items must use the same currency for v1.
- Reject unknown JSON fields to catch typo-prone API usage early.

Response shape excerpt:

```json
{
  "years": 10,
  "currency": "USD",
  "items": [
    {
      "id": "item_000001",
      "name": "Example brokerage",
      "startingAmountCents": 1250000,
      "annualReturnRateBasisPoints": 700,
      "annualContributionCents": 300000,
      "yearlyBalances": [
        {
          "year": 0,
          "balanceCents": 1250000,
          "contributionCents": 0,
          "growthCents": 0
        },
        {
          "year": 1,
          "balanceCents": 1637500,
          "contributionCents": 300000,
          "growthCents": 87500
        }
      ]
    }
  ],
  "totals": [
    {
      "year": 0,
      "balanceCents": 1250000,
      "contributionCents": 0,
      "growthCents": 0
    },
    {
      "year": 1,
      "balanceCents": 1637500,
      "contributionCents": 300000,
      "growthCents": 87500
    }
  ]
}
```

## Calculation rule

For each item and projected year:

```text
growthCents = round(previousBalanceCents * annualReturnRateBasisPoints / 10000)
balanceCents = previousBalanceCents + growthCents + annualContributionCents
```

Year `0` should always represent the starting state with zero growth and zero contribution.

Use integer cents throughout. Define rounding behavior in tests before implementation; recommended v1 behavior is half-away-from-zero using integer arithmetic for deterministic results.

## Relationship to the living plan

This document records the projection behavior and API contract decisions. It should stay focused on the projection spec rather than duplicating the staged implementation checklist.

Use [App Rewrite Plan](app-rewrite.md) for the current step sequence, status, branches, and PR links. At the time this spec moved from planning into implementation, the relevant projection milestones were:

- Projection calculation engine in `internal/projections`
- `POST /projections` API endpoint
- Projection UI controls and chart/table views after the API contract exists

## Future features explicitly out of v1

- Monthly compounding or contribution timing options
- Inflation-adjusted dollars
- Tax treatment and account category modeling
- Withdrawals or retirement drawdown planning
- Monte Carlo or variable returns
- Persisted projection scenarios
- Authentication or multi-user ownership

These can be added after the basic UI/API workflow is useful and the deterministic projection contract is implemented and tested.

## Review questions before implementation

- Should v1 calculate from stored items by default when `items` is omitted, or should projections always require an explicit item list?
- Is annual contribution at end-of-year acceptable for v1, or do you want beginning-of-year contribution timing?
- Is the recommended `1` through `75` year bound reasonable?
- Should `annualReturnRateBasisPoints` continue to allow negative values for projection scenarios?
