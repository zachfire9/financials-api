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
