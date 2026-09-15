# Drawdown Projection Planning

## Goal

Extend the deterministic projection design from a single accumulation horizon into a two-phase projection that can model saving/accumulation years followed by optional retirement-style drawdown years.

This document is a planning/spec reference only. The active step checklist, branch names, PR links, and implementation status live in [App Rewrite Plan](app-rewrite.md).

## Current foundation

The existing projection API and engine already support a deterministic accumulation projection:

- Whole-year projection periods.
- Per-item starting balances from saved or hypothetical financial items.
- Annual return basis points per item.
- End-of-year annual contributions per item.
- Per-item yearly balances plus aggregate totals.
- Integer cents and deterministic rounding.

The first UI now renders the repository-backed projection as a year-grouped table, so the next backend milestone should define the drawdown contract before changing calculation behavior.

## Recommended v2 scope

Keep the next version small and deterministic:

- Split the horizon into `savingYears` and optional `drawdownYears`.
- Preserve the existing accumulation math during saving years.
- Allow each item to use a different return assumption once drawdown starts.
- Continue returning Year `0` as the starting state.
- Use annual periods only.
- Keep all outputs on demand; do not persist scenarios yet.
- Keep taxes, inflation adjustments, account categories, required minimum distributions, and Monte Carlo behavior out of this step.

## Proposed request shape

The existing `years` field can remain supported as a v1-compatible alias for accumulation-only projections. The drawdown-capable request should add explicit phase fields:

```json
{
  "savingYears": 10,
  "drawdownYears": 30,
  "annualWithdrawalCents": 6000000,
  "items": [
    {
      "name": "Example brokerage",
      "amountCents": 1250000,
      "currency": "USD",
      "annualReturnRateBasisPoints": 700,
      "drawdownAnnualReturnRateBasisPoints": 400,
      "annualContributionCents": 300000,
      "sortOrder": 1
    }
  ]
}
```

Request rules to confirm before implementation:

- `savingYears` should be required for the v2 shape and bounded, recommended `0` through `75`.
- `drawdownYears` should be optional and bounded, recommended `0` through `75`.
- At least one of `savingYears` or `drawdownYears` must be greater than `0`.
- `years` should remain accepted for the current UI and should be mutually exclusive with `savingYears`/`drawdownYears`.
- `annualWithdrawalCents` should be required when `drawdownYears` is greater than `0`.
- `drawdownAnnualReturnRateBasisPoints` should be optional per item; if omitted, drawdown years should keep using that item's `annualReturnRateBasisPoints`.
- Hypothetical `items` should keep the current behavior: validate but do not save.
- Omitted or empty `items` should keep using the repository-backed financial items.
- All projected items must use one currency.
- Unknown JSON fields should continue returning `400`.

## Proposed response shape

Add phase metadata to yearly balances and totals while keeping existing amount fields stable:

```json
{
  "savingYears": 10,
  "drawdownYears": 30,
  "currency": "USD",
  "items": [
    {
      "id": "item_000001",
      "name": "Example brokerage",
      "startingAmountCents": 1250000,
      "annualReturnRateBasisPoints": 700,
      "drawdownAnnualReturnRateBasisPoints": 400,
      "annualContributionCents": 300000,
      "yearlyBalances": [
        {
          "year": 0,
          "phase": "starting",
          "balanceCents": 1250000,
          "contributionCents": 0,
          "withdrawalCents": 0,
          "growthCents": 0
        },
        {
          "year": 1,
          "phase": "saving",
          "balanceCents": 1637500,
          "contributionCents": 300000,
          "withdrawalCents": 0,
          "growthCents": 87500
        },
        {
          "year": 11,
          "phase": "drawdown",
          "balanceCents": 1200000,
          "contributionCents": 0,
          "withdrawalCents": 600000,
          "growthCents": 50000
        }
      ]
    }
  ],
  "totals": [
    {
      "year": 11,
      "phase": "drawdown",
      "balanceCents": 1200000,
      "contributionCents": 0,
      "withdrawalCents": 600000,
      "growthCents": 50000
    }
  ]
}
```

Compatibility option: `years` can remain in the response for v1 accumulation-only requests until the UI migrates to the explicit phase fields.

## Drawdown calculation decisions to confirm

These decisions materially affect implementation, so the planning step should not silently bake them into the engine:

1. **Drawdown return rates**
   - Recommended v1: add optional `drawdownAnnualReturnRateBasisPoints` per item.
   - If omitted, default to the item's accumulation `annualReturnRateBasisPoints` so existing callers do not need extra fields.
   - This keeps conservative retirement-return assumptions possible now without adding account categories or tax modeling.

2. **Withdrawal timing**
   - Recommended default: apply growth first, then subtract the annual withdrawal at the end of the year.
   - Alternative: subtract at the beginning of the year, then grow the remaining balance.

3. **Withdrawal allocation across items**
   - Recommended v1: support a simple default allocation first, then add explicit per-account withdrawal strategy later.
   - Better default than an even split: withdraw proportionally from each item based on the prior year's balance, because it is deterministic and works with any number of items.
   - Future strategy shape should support account priority and/or per-item withdrawal amounts, for example withdrawing from a Roth IRA before other accounts for early-retirement years.
   - Avoid naming the v1 proportional allocation as the final retirement strategy; keep it as the fallback when no explicit strategy is supplied.

4. **Depletion behavior**
   - Recommended v1: floor individual item balances at zero and report any unfunded withdrawal amount.
   - Alternative: allow negative balances to make the shortfall obvious in the same balance field.

5. **Contributions during drawdown**
   - Recommended v1: set contributions to zero during drawdown years.
   - Alternative: allow continuing per-item contributions even during drawdown.

6. **Default drawdown horizon**
   - Recommended v1: no implicit drawdown; `drawdownYears` defaults to `0` unless the UI/user supplies it.
   - If Zach wants a one-field retirement projection later, the UI can provide a default such as `30` years.

## Proposed staged implementation after this planning step

### Step 16: Drawdown calculation engine

- Extend `internal/projections` models with explicit phase fields and optional per-item drawdown return rates.
- Add strict RED/GREEN tests for saving-only compatibility, saving plus drawdown, per-item drawdown return rates, default allocation behavior, zero-floor/depletion behavior, validation, rounding, and response totals.
- Preserve current `years` accumulation behavior for existing callers.
- Keep HTTP wiring out of this step.

### Step 17: Drawdown projection API contract

- Extend `POST /projections` request parsing and JSON response tags.
- Add endpoint tests for `savingYears`, `drawdownYears`, `annualWithdrawalCents`, `drawdownAnnualReturnRateBasisPoints`, compatibility with `years`, unknown field rejection, and repository-backed vs hypothetical item behavior.
- Update README examples with fake data only.

### Step 18: Drawdown projection UI

- Add controls for saving years, optional drawdown years, and annual withdrawal.
- Render saving/drawdown phase labels in the projection results.
- Preserve stale-data fallback and year-grouped item rows.
- Keep charts optional until the contract is proven through table output.

## Out of scope for the drawdown v1 work

- Tax-aware withdrawal ordering.
- User-configured per-account withdrawal schedules, fixed dollar amounts, and account-priority rules such as Roth-first drawdown before age 65. The v1 model should leave room for this by treating proportional allocation as a fallback strategy, not as the permanent API shape.
- Inflation-adjusted spending.
- Social Security, pension, or income streams.
- Required minimum distributions.
- Account categories and tax buckets.
- Monte Carlo or variable annual returns.
- Persisted named scenarios.
- Authentication or multi-user ownership.
