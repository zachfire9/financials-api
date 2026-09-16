package projections

import (
	"strings"
	"testing"
)

func TestCalculateProjectsPerItemBalancesAndAggregateTotals(t *testing.T) {
	projection, err := Calculate(Request{
		Years: 2,
		Items: []ItemInput{
			{
				ID:                          "item_000002",
				Name:                        "Example savings",
				AmountCents:                 10000,
				Currency:                    "USD",
				AnnualReturnRateBasisPoints: 500,
				AnnualContributionCents:     1000,
				SortOrder:                   2,
			},
			{
				ID:                          "item_000001",
				Name:                        "Example brokerage",
				AmountCents:                 20000,
				Currency:                    "USD",
				AnnualReturnRateBasisPoints: 1000,
				AnnualContributionCents:     2000,
				SortOrder:                   1,
			},
		},
	})
	if err != nil {
		t.Fatalf("expected projection to calculate, got %v", err)
	}

	if projection.Years != 2 || projection.Currency != "USD" {
		t.Fatalf("unexpected projection metadata: %+v", projection)
	}
	if len(projection.Items) != 2 {
		t.Fatalf("expected two projected items, got %d", len(projection.Items))
	}
	if projection.Items[0].ID != "item_000001" || projection.Items[1].ID != "item_000002" {
		t.Fatalf("expected items sorted by sortOrder then ID, got %+v", projection.Items)
	}

	assertYearlyBalance(t, projection.Items[0].YearlyBalances[0], 0, 20000, 0, 0)
	assertYearlyBalance(t, projection.Items[0].YearlyBalances[1], 1, 24000, 2000, 2000)
	assertYearlyBalance(t, projection.Items[0].YearlyBalances[2], 2, 28400, 2000, 2400)

	assertYearlyBalance(t, projection.Items[1].YearlyBalances[0], 0, 10000, 0, 0)
	assertYearlyBalance(t, projection.Items[1].YearlyBalances[1], 1, 11500, 1000, 500)
	assertYearlyBalance(t, projection.Items[1].YearlyBalances[2], 2, 13075, 1000, 575)

	if len(projection.Totals) != 3 {
		t.Fatalf("expected totals for year 0 through year 2, got %d", len(projection.Totals))
	}
	assertYearlyBalance(t, projection.Totals[0], 0, 30000, 0, 0)
	assertYearlyBalance(t, projection.Totals[1], 1, 35500, 3000, 2500)
	assertYearlyBalance(t, projection.Totals[2], 2, 41475, 3000, 2975)
}

func TestCalculateProjectsSavingAndDrawdownPhases(t *testing.T) {
	projection, err := Calculate(Request{
		SavingYears:           1,
		DrawdownYears:         1,
		AnnualWithdrawalCents: 4200,
		Items: []ItemInput{
			{
				ID:                                  "item_000001",
				Name:                                "Example brokerage",
				AmountCents:                         10000,
				Currency:                            "USD",
				AnnualReturnRateBasisPoints:         1000,
				DrawdownAnnualReturnRateBasisPoints: intPtr(0),
				AnnualContributionCents:             1000,
				SortOrder:                           1,
			},
			{
				ID:                                  "item_000002",
				Name:                                "Example IRA",
				AmountCents:                         30000,
				Currency:                            "USD",
				AnnualReturnRateBasisPoints:         0,
				DrawdownAnnualReturnRateBasisPoints: intPtr(-1000),
				AnnualContributionCents:             0,
				SortOrder:                           2,
			},
		},
	})
	if err != nil {
		t.Fatalf("expected drawdown projection to calculate, got %v", err)
	}

	if projection.Years != 2 || projection.SavingYears != 1 || projection.DrawdownYears != 1 || projection.Currency != "USD" {
		t.Fatalf("unexpected projection metadata: %+v", projection)
	}

	brokerage := projection.Items[0]
	if brokerage.DrawdownAnnualReturnRateBasisPoints == nil || *brokerage.DrawdownAnnualReturnRateBasisPoints != 0 {
		t.Fatalf("expected brokerage drawdown return rate to be preserved, got %+v", brokerage.DrawdownAnnualReturnRateBasisPoints)
	}
	assertYearlyPhaseBalance(t, brokerage.YearlyBalances[0], 0, PhaseStarting, 10000, 0, 0, 0, 0)
	assertYearlyPhaseBalance(t, brokerage.YearlyBalances[1], 1, PhaseSaving, 12000, 1000, 0, 1000, 0)
	assertYearlyPhaseBalance(t, brokerage.YearlyBalances[2], 2, PhaseDrawdown, 10800, 0, 1200, 0, 0)

	ira := projection.Items[1]
	assertYearlyPhaseBalance(t, ira.YearlyBalances[0], 0, PhaseStarting, 30000, 0, 0, 0, 0)
	assertYearlyPhaseBalance(t, ira.YearlyBalances[1], 1, PhaseSaving, 30000, 0, 0, 0, 0)
	assertYearlyPhaseBalance(t, ira.YearlyBalances[2], 2, PhaseDrawdown, 24000, 0, 3000, -3000, 0)

	assertYearlyPhaseBalance(t, projection.Totals[0], 0, PhaseStarting, 40000, 0, 0, 0, 0)
	assertYearlyPhaseBalance(t, projection.Totals[1], 1, PhaseSaving, 42000, 1000, 0, 1000, 0)
	assertYearlyPhaseBalance(t, projection.Totals[2], 2, PhaseDrawdown, 34800, 0, 4200, -3000, 0)
}

func TestCalculateUsesAccumulationReturnDuringDrawdownWhenNoDrawdownRateIsProvided(t *testing.T) {
	projection, err := Calculate(Request{
		SavingYears:           0,
		DrawdownYears:         1,
		AnnualWithdrawalCents: 1000,
		Items: []ItemInput{{
			ID:                          "item_000001",
			Name:                        "Example balanced fund",
			AmountCents:                 10000,
			Currency:                    "USD",
			AnnualReturnRateBasisPoints: 1000,
		}},
	})
	if err != nil {
		t.Fatalf("expected drawdown projection to calculate, got %v", err)
	}

	assertYearlyPhaseBalance(t, projection.Items[0].YearlyBalances[1], 1, PhaseDrawdown, 10000, 0, 1000, 1000, 0)
	assertYearlyPhaseBalance(t, projection.Totals[1], 1, PhaseDrawdown, 10000, 0, 1000, 1000, 0)
}

func TestCalculateInflatesAnnualWithdrawalsDuringDrawdown(t *testing.T) {
	projection, err := Calculate(Request{
		SavingYears:                              0,
		DrawdownYears:                            3,
		AnnualWithdrawalCents:                    6000000,
		AnnualWithdrawalInflationRateBasisPoints: 300,
		Items: []ItemInput{{
			ID:                          "item_000001",
			Name:                        "Example retirement account",
			AmountCents:                 20000000,
			Currency:                    "USD",
			AnnualReturnRateBasisPoints: 0,
		}},
	})
	if err != nil {
		t.Fatalf("expected inflation-adjusted drawdown projection to calculate, got %v", err)
	}

	assertYearlyPhaseBalance(t, projection.Items[0].YearlyBalances[1], 1, PhaseDrawdown, 14000000, 0, 6000000, 0, 0)
	assertYearlyPhaseBalance(t, projection.Items[0].YearlyBalances[2], 2, PhaseDrawdown, 7820000, 0, 6180000, 0, 0)
	assertYearlyPhaseBalance(t, projection.Items[0].YearlyBalances[3], 3, PhaseDrawdown, 1454600, 0, 6365400, 0, 0)

	assertYearlyPhaseBalance(t, projection.Totals[1], 1, PhaseDrawdown, 14000000, 0, 6000000, 0, 0)
	assertYearlyPhaseBalance(t, projection.Totals[2], 2, PhaseDrawdown, 7820000, 0, 6180000, 0, 0)
	assertYearlyPhaseBalance(t, projection.Totals[3], 3, PhaseDrawdown, 1454600, 0, 6365400, 0, 0)
}

func TestCalculateInflatesAnnualWithdrawalDuringSavingYearsBeforeDrawdown(t *testing.T) {
	projection, err := Calculate(Request{
		SavingYears:                              1,
		DrawdownYears:                            2,
		AnnualWithdrawalCents:                    6000000,
		AnnualWithdrawalInflationRateBasisPoints: 300,
		Items: []ItemInput{{
			ID:                          "item_000001",
			Name:                        "Example retirement account",
			AmountCents:                 20000000,
			Currency:                    "USD",
			AnnualReturnRateBasisPoints: 0,
		}},
	})
	if err != nil {
		t.Fatalf("expected inflation-adjusted drawdown projection to calculate, got %v", err)
	}

	assertYearlyPhaseBalance(t, projection.Items[0].YearlyBalances[1], 1, PhaseSaving, 20000000, 0, 0, 0, 0)
	assertYearlyPhaseBalance(t, projection.Items[0].YearlyBalances[2], 2, PhaseDrawdown, 13820000, 0, 6180000, 0, 0)
	assertYearlyPhaseBalance(t, projection.Items[0].YearlyBalances[3], 3, PhaseDrawdown, 7454600, 0, 6365400, 0, 0)

	assertYearlyPhaseBalance(t, projection.Totals[2], 2, PhaseDrawdown, 13820000, 0, 6180000, 0, 0)
	assertYearlyPhaseBalance(t, projection.Totals[3], 3, PhaseDrawdown, 7454600, 0, 6365400, 0, 0)
}

func TestCalculateFloorsDrawdownBalancesAndReportsUnfundedWithdrawal(t *testing.T) {
	projection, err := Calculate(Request{
		SavingYears:           0,
		DrawdownYears:         1,
		AnnualWithdrawalCents: 15000,
		Items: []ItemInput{{
			ID:                          "item_000001",
			Name:                        "Example small account",
			AmountCents:                 10000,
			Currency:                    "USD",
			AnnualReturnRateBasisPoints: 0,
		}},
	})
	if err != nil {
		t.Fatalf("expected drawdown projection to calculate, got %v", err)
	}

	assertYearlyPhaseBalance(t, projection.Items[0].YearlyBalances[1], 1, PhaseDrawdown, 0, 0, 10000, 0, 5000)
	assertYearlyPhaseBalance(t, projection.Totals[1], 1, PhaseDrawdown, 0, 0, 10000, 0, 5000)
}

func TestCalculateAllowsNegativeReturnAssumptions(t *testing.T) {
	projection, err := Calculate(Request{
		Years: 2,
		Items: []ItemInput{{
			ID:                          "item_000001",
			Name:                        "Example conservative account",
			AmountCents:                 10000,
			Currency:                    "USD",
			AnnualReturnRateBasisPoints: -250,
			AnnualContributionCents:     1000,
		}},
	})
	if err != nil {
		t.Fatalf("expected negative return within bounds to calculate, got %v", err)
	}

	assertYearlyBalance(t, projection.Items[0].YearlyBalances[1], 1, 10750, 1000, -250)
	assertYearlyBalance(t, projection.Items[0].YearlyBalances[2], 2, 11481, 1000, -269)
}

func TestCalculateUsesHalfAwayFromZeroRounding(t *testing.T) {
	projection, err := Calculate(Request{
		Years: 1,
		Items: []ItemInput{
			{
				ID:                          "positive_rounding",
				Name:                        "Positive rounding example",
				AmountCents:                 50,
				Currency:                    "USD",
				AnnualReturnRateBasisPoints: 100,
			},
			{
				ID:                          "negative_rounding",
				Name:                        "Negative rounding example",
				AmountCents:                 50,
				Currency:                    "USD",
				AnnualReturnRateBasisPoints: -100,
				SortOrder:                   1,
			},
		},
	})
	if err != nil {
		t.Fatalf("expected rounding examples to calculate, got %v", err)
	}

	assertYearlyBalance(t, projection.Items[0].YearlyBalances[1], 1, 51, 0, 1)
	assertYearlyBalance(t, projection.Items[1].YearlyBalances[1], 1, 49, 0, -1)
}

func TestCalculateRejectsInvalidRequests(t *testing.T) {
	tests := []struct {
		name    string
		request Request
		want    string
	}{
		{
			name: "zero years",
			request: Request{Years: 0, Items: []ItemInput{{
				Name:     "Example",
				Currency: "USD",
			}}},
			want: "years",
		},
		{
			name: "years above upper bound",
			request: Request{Years: 76, Items: []ItemInput{{
				Name:     "Example",
				Currency: "USD",
			}}},
			want: "years",
		},
		{
			name: "v1 years cannot be combined with v2 phase fields",
			request: Request{Years: 10, SavingYears: 5, Items: []ItemInput{{
				Name:     "Example",
				Currency: "USD",
			}}},
			want: "mutually exclusive",
		},
		{
			name: "drawdown years require withdrawal amount",
			request: Request{SavingYears: 1, DrawdownYears: 1, Items: []ItemInput{{
				Name:     "Example",
				Currency: "USD",
			}}},
			want: "annualWithdrawalCents",
		},
		{
			name: "phase horizons require at least one projected year",
			request: Request{SavingYears: 0, DrawdownYears: 0, Items: []ItemInput{{
				Name:     "Example",
				Currency: "USD",
			}}},
			want: "years",
		},
		{
			name: "invalid withdrawal inflation rate",
			request: Request{SavingYears: 1, AnnualWithdrawalInflationRateBasisPoints: -1, Items: []ItemInput{{
				Name:     "Example",
				Currency: "USD",
			}}},
			want: "annualWithdrawalInflationRateBasisPoints",
		},
		{
			name: "invalid drawdown return rate",
			request: Request{SavingYears: 1, Items: []ItemInput{{
				Name:                                "Example",
				Currency:                            "USD",
				DrawdownAnnualReturnRateBasisPoints: intPtr(100001),
			}}},
			want: "drawdownAnnualReturnRateBasisPoints",
		},
		{
			name:    "no items",
			request: Request{Years: 10},
			want:    "items",
		},
		{
			name: "invalid item fields",
			request: Request{Years: 10, Items: []ItemInput{{
				Name:                        " ",
				AmountCents:                 -1,
				Currency:                    "usd",
				AnnualReturnRateBasisPoints: 100001,
				AnnualContributionCents:     -1,
				SortOrder:                   -1,
			}}},
			want: "name",
		},
		{
			name: "currency mismatch",
			request: Request{Years: 10, Items: []ItemInput{
				{Name: "Example USD", Currency: "USD"},
				{Name: "Example EUR", Currency: "EUR"},
			}},
			want: "currency",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Calculate(test.request)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected error to contain %q, got %v", test.want, err)
			}
		})
	}
}

func assertYearlyBalance(t *testing.T, got YearlyBalance, year int, balanceCents int64, contributionCents int64, growthCents int64) {
	t.Helper()
	if got.Year != year || got.BalanceCents != balanceCents || got.ContributionCents != contributionCents || got.GrowthCents != growthCents {
		t.Fatalf("unexpected yearly balance: got %+v, want year=%d balance=%d contribution=%d growth=%d", got, year, balanceCents, contributionCents, growthCents)
	}
}

func assertYearlyPhaseBalance(t *testing.T, got YearlyBalance, year int, phase Phase, balanceCents int64, contributionCents int64, withdrawalCents int64, growthCents int64, unfundedWithdrawalCents int64) {
	t.Helper()
	if got.Year != year || got.Phase != phase || got.BalanceCents != balanceCents || got.ContributionCents != contributionCents || got.WithdrawalCents != withdrawalCents || got.GrowthCents != growthCents || got.UnfundedWithdrawalCents != unfundedWithdrawalCents {
		t.Fatalf("unexpected yearly phase balance: got %+v, want year=%d phase=%s balance=%d contribution=%d withdrawal=%d growth=%d unfundedWithdrawal=%d", got, year, phase, balanceCents, contributionCents, withdrawalCents, growthCents, unfundedWithdrawalCents)
	}
}

func intPtr(value int) *int {
	return &value
}
