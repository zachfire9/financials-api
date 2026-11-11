package projections

import (
	"reflect"
	"strings"
	"testing"
)

func spendingOf(projection Projection) []YearlySpending {
	return projection.Spending
}

// One saving year, then three drawdown years with 10% inflation:
//
//	          living (10%)  healthcare (20%)  travel (first 2 years)  total
//	year 2:   1100          1200              550                     2850
//	year 3:   1210          1440              605                     3255
//	year 4:   1331          1728              -                       3059
func TestCalculateItemizedExpenses(t *testing.T) {
	projection, err := Calculate(Request{
		SavingYears:                              1,
		DrawdownYears:                            3,
		AnnualWithdrawalInflationRateBasisPoints: 1000,
		Expenses: []ExpenseInput{
			{ID: "living", Name: "Living", AnnualAmountCents: 1000},
			{ID: "health", Name: "Healthcare", AnnualAmountCents: 1000, InflationRateBasisPoints: intPointer(2000)},
			{ID: "travel", Name: "Travel", AnnualAmountCents: 500, Years: 2},
		},
		Items: []ItemInput{{ID: "a", Name: "Savings", AmountCents: 1000000, Currency: "USD", SortOrder: 1}},
	})
	if err != nil {
		t.Fatalf("expected projection to calculate, got %v", err)
	}

	want := []YearlySpending{{Year: 1}, {Year: 2, ExpensesCents: 2850}, {Year: 3, ExpensesCents: 3255}, {Year: 4, ExpensesCents: 3059}}
	if !reflect.DeepEqual(spendingOf(projection), want) {
		t.Fatalf("spending: expected %+v, got %+v", want, projection.Spending)
	}
	for year, wantWithdrawal := range []int64{0, 0, 2850, 3255, 3059} {
		if got := projection.Totals[year].WithdrawalCents; got != wantWithdrawal {
			t.Fatalf("year %d: expected withdrawal %d, got %d", year, wantWithdrawal, got)
		}
	}
}

// No inflation, five drawdown years:
//
//	                    year 1  year 2  year 3  year 4  year 5
//	living (all)         100     100     100     100     100
//	insurance (1-2)       50      50       -       -       -
//	care (from 4)          -       -       -     200     200
//	travel (years: 3)     10      10      10       -       -   (older "years" field)
func TestCalculateExpensesWithinYearRanges(t *testing.T) {
	projection, err := Calculate(Request{
		DrawdownYears: 5,
		Expenses: []ExpenseInput{
			{Name: "Living", AnnualAmountCents: 100},
			{Name: "Health insurance", AnnualAmountCents: 50, ToYear: 2},
			{Name: "Long-term care", AnnualAmountCents: 200, FromYear: 4},
			{Name: "Travel", AnnualAmountCents: 10, Years: 3},
		},
		Items: []ItemInput{{ID: "a", Name: "Savings", AmountCents: 100000, Currency: "USD", SortOrder: 1}},
	})
	if err != nil {
		t.Fatalf("expected projection to calculate, got %v", err)
	}
	got := make([]int64, len(projection.Spending))
	for index, spending := range projection.Spending {
		got[index] = spending.ExpensesCents
	}
	if want := []int64{160, 160, 110, 300, 300}; !reflect.DeepEqual(got, want) {
		t.Fatalf("expected expenses %v, got %v", want, got)
	}
}

// An expense that starts later is still in today's dollars: with 10%
// inflation, $100 starting in drawdown year 3 (projection year 3) has grown
// for two years by then.
func TestCalculateLaterExpensesKeepGrowingWithInflationBeforeTheyStart(t *testing.T) {
	projection, err := Calculate(Request{
		DrawdownYears:                            3,
		AnnualWithdrawalInflationRateBasisPoints: 1000,
		Expenses:                                 []ExpenseInput{{Name: "Care", AnnualAmountCents: 10000, FromYear: 3}},
		Items:                                    []ItemInput{{ID: "a", Name: "Savings", AmountCents: 100000, Currency: "USD", SortOrder: 1}},
	})
	if err != nil {
		t.Fatalf("expected projection to calculate, got %v", err)
	}
	if got := projection.Spending[2].ExpensesCents; got != 12100 || projection.Spending[1].ExpensesCents != 0 {
		t.Fatalf("expected nothing until year 3, then 12100, got %+v", projection.Spending)
	}
}

func TestCalculateExpensesStartAfterTheLastRetirement(t *testing.T) {
	projection, err := Calculate(Request{
		People:                   []PersonInput{{ID: "alex", Name: "Alex", SavingYears: 1}, {ID: "sam", Name: "Sam", SavingYears: 2}},
		DrawdownYears:            2,
		GapAnnualWithdrawalCents: 300,
		Expenses:                 []ExpenseInput{{Name: "Travel", AnnualAmountCents: 1000, Years: 1}},
		Items:                    []ItemInput{{ID: "j", Name: "Joint", AmountCents: 100000, Currency: "USD", SortOrder: 1}},
	})
	if err != nil {
		t.Fatalf("expected projection to calculate, got %v", err)
	}
	want := []YearlySpending{{Year: 1}, {Year: 2, ExpensesCents: 300}, {Year: 3, ExpensesCents: 1000}, {Year: 4}}
	if !reflect.DeepEqual(projection.Spending, want) {
		t.Fatalf("expected the gap amount in the transition year and travel only in the first drawdown year, got %+v", projection.Spending)
	}
}

// $300,000 at 6% with a $1,798.66 payment. After year 1, $296,315.88 is owed
// (a standard amortization table says $296,315.99 for the $1,798.65 payment),
// and the loan is paid off in year 30.
func TestCalculateAmortizesTheMortgage(t *testing.T) {
	projection, err := Calculate(Request{
		SavingYears:   31,
		DrawdownYears: 1, AnnualWithdrawalCents: 100,
		Mortgage: &MortgageInput{BalanceCents: 30000000, AnnualRateBasisPoints: 600, MonthlyPaymentCents: 179866},
		Items:    []ItemInput{{ID: "a", Name: "Savings", AmountCents: 1000000, Currency: "USD", SortOrder: 1}},
	})
	if err != nil {
		t.Fatalf("expected projection to calculate, got %v", err)
	}

	mortgage := projection.Mortgage
	if mortgage == nil || mortgage.PayoffYear != 30 {
		t.Fatalf("expected payoff in year 30, got %+v", mortgage)
	}
	if got := mortgage.Years[0]; got != (MortgageYear{Year: 0, BalanceCents: 30000000}) {
		t.Fatalf("year 0: expected today's balance, got %+v", got)
	}
	if got := mortgage.Years[1]; got != (MortgageYear{Year: 1, BalanceCents: 29631588, PaymentCents: 2158392, InterestCents: 1789980}) {
		t.Fatalf("year 1: unexpected amortization %+v", got)
	}
	if got := mortgage.Years[30]; got.BalanceCents != 0 || got.PaymentCents != 2157549 {
		t.Fatalf("year 30: expected the final, slightly smaller payments, got %+v", got)
	}
	if got := mortgage.Years[31]; got.PaymentCents != 0 || got.PaidFromSavings {
		t.Fatalf("year 31: expected nothing owed, got %+v", got)
	}
	for _, year := range mortgage.Years {
		if year.PaidFromSavings {
			t.Fatalf("payments before retirement should come from salary, got %+v", year)
		}
	}
}

// 0% APR, $100,000 at $1,000 a month: $12,000 a year, paid off in year 9
// with a final $4,000. Saving ends after year 7, so years 8 and 9 add the
// payment to the $5,000 withdrawal.
func TestCalculateAddsMortgagePaymentsToDrawdownWithdrawals(t *testing.T) {
	projection, err := Calculate(Request{
		SavingYears:           7,
		DrawdownYears:         3,
		AnnualWithdrawalCents: 500000,
		Mortgage:              &MortgageInput{BalanceCents: 10000000, AnnualRateBasisPoints: 0, MonthlyPaymentCents: 100000},
		Items:                 []ItemInput{{ID: "a", Name: "Savings", AmountCents: 100000000, Currency: "USD", SortOrder: 1}},
	})
	if err != nil {
		t.Fatalf("expected projection to calculate, got %v", err)
	}

	if projection.Mortgage.PayoffYear != 9 {
		t.Fatalf("expected payoff in year 9, got %d", projection.Mortgage.PayoffYear)
	}
	wantSpending := map[int]YearlySpending{
		7:  {Year: 7},
		8:  {Year: 8, ExpensesCents: 500000, MortgagePaymentCents: 1200000},
		9:  {Year: 9, ExpensesCents: 500000, MortgagePaymentCents: 400000},
		10: {Year: 10, ExpensesCents: 500000},
	}
	for year, want := range wantSpending {
		if got := projection.Spending[year-1]; got != want {
			t.Fatalf("year %d spending: expected %+v, got %+v", year, want, got)
		}
	}
	for year, want := range map[int]int64{7: 0, 8: 1700000, 9: 900000, 10: 500000} {
		if got := projection.Totals[year].WithdrawalCents; got != want {
			t.Fatalf("year %d: expected withdrawal %d, got %d", year, want, got)
		}
	}
	if !projection.Mortgage.Years[8].PaidFromSavings || projection.Mortgage.Years[7].PaidFromSavings {
		t.Fatalf("expected only drawdown-year payments to come from savings")
	}
}

func TestCalculateWithoutExpensesOrMortgageOmitsTheirResults(t *testing.T) {
	projection, err := Calculate(Request{
		SavingYears: 1, DrawdownYears: 1, AnnualWithdrawalCents: 100,
		Items: []ItemInput{{ID: "a", Name: "Savings", AmountCents: 1000, Currency: "USD", SortOrder: 1}},
	})
	if err != nil {
		t.Fatalf("expected projection to calculate, got %v", err)
	}
	if projection.Mortgage != nil {
		t.Fatalf("expected no mortgage results, got %+v", projection.Mortgage)
	}
	if want := []YearlySpending{{Year: 1}, {Year: 2, ExpensesCents: 100}}; !reflect.DeepEqual(projection.Spending, want) {
		t.Fatalf("expected spending to show the single withdrawal, got %+v", projection.Spending)
	}
}

func TestCalculateRejectsInvalidExpensesAndMortgages(t *testing.T) {
	item := ItemInput{ID: "a", Name: "Savings", AmountCents: 1000, Currency: "USD", SortOrder: 1}
	tests := []struct {
		name    string
		request Request
		want    string
	}{
		{"withdrawal and expenses", Request{SavingYears: 1, DrawdownYears: 1, AnnualWithdrawalCents: 100, Expenses: []ExpenseInput{{Name: "Food", AnnualAmountCents: 1}}, Items: []ItemInput{item}}, "annualWithdrawalCents must be 0 when expenses are provided"},
		{"blank expense name", Request{DrawdownYears: 1, Expenses: []ExpenseInput{{Name: " ", AnnualAmountCents: 1}}, Items: []ItemInput{item}}, "expenses[0].name is required"},
		{"negative expense", Request{DrawdownYears: 1, Expenses: []ExpenseInput{{Name: "Food", AnnualAmountCents: -1}}, Items: []ItemInput{item}}, "expenses[0].annualAmountCents must be greater than or equal to 0"},
		{"expense years too long", Request{DrawdownYears: 1, Expenses: []ExpenseInput{{Name: "Food", AnnualAmountCents: 1, Years: 76}}, Items: []ItemInput{item}}, "expenses[0].years must be between 0 and 75"},
		{"expense inflation negative", Request{DrawdownYears: 1, Expenses: []ExpenseInput{{Name: "Food", AnnualAmountCents: 1, InflationRateBasisPoints: intPointer(-1)}}, Items: []ItemInput{item}}, "expenses[0].inflationRateBasisPoints"},
		{"range backwards", Request{DrawdownYears: 1, Expenses: []ExpenseInput{{Name: "Food", AnnualAmountCents: 1, FromYear: 5, ToYear: 3}}, Items: []ItemInput{item}}, "expenses[0].fromYear must not be after toYear"},
		{"toYear and years", Request{DrawdownYears: 1, Expenses: []ExpenseInput{{Name: "Food", AnnualAmountCents: 1, ToYear: 3, Years: 3}}, Items: []ItemInput{item}}, "expenses[0] can set toYear or years, not both"},
		{"fromYear too late", Request{DrawdownYears: 1, Expenses: []ExpenseInput{{Name: "Food", AnnualAmountCents: 1, FromYear: 76}}, Items: []ItemInput{item}}, "expenses[0].fromYear must be between 0 and 75"},
		{"mortgage without balance", Request{SavingYears: 1, Mortgage: &MortgageInput{MonthlyPaymentCents: 100}, Items: []ItemInput{item}}, "mortgage.balanceCents must be greater than 0"},
		{"mortgage rate too high", Request{SavingYears: 1, Mortgage: &MortgageInput{BalanceCents: 1000, AnnualRateBasisPoints: 10001, MonthlyPaymentCents: 1000}, Items: []ItemInput{item}}, "mortgage.annualRateBasisPoints must be between 0 and 10000"},
		{"payment below interest", Request{SavingYears: 1, Mortgage: &MortgageInput{BalanceCents: 30000000, AnnualRateBasisPoints: 600, MonthlyPaymentCents: 150000}, Items: []ItemInput{item}}, "mortgage.monthlyPaymentCents must be more than the first month's interest (150000)"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Calculate(test.request)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected error containing %q, got %v", test.want, err)
			}
		})
	}
}
