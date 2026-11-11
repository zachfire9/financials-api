package projections

import (
	"reflect"
	"strings"
	"testing"
)

func withdrawalsOf(item ProjectedItem) []int64 {
	withdrawals := make([]int64, len(item.YearlyBalances))
	for index, balance := range item.YearlyBalances {
		withdrawals[index] = balance.WithdrawalCents
	}
	return withdrawals
}

// 0% returns, 700 a year for 3 years:
//
//	year 1: all from "first" (1000 -> 300)
//	year 2: "first" runs out (300), the rest from "normal" (400)
//	year 3: "normal" gives its last 600, and the remaining 100 comes from "last"
func TestCalculateWithdrawsFromPriorityGroupsInOrder(t *testing.T) {
	projection, err := Calculate(Request{
		DrawdownYears:         3,
		AnnualWithdrawalCents: 700,
		Items: []ItemInput{
			{ID: "last", Name: "Roth IRA", AmountCents: 1000, Currency: "USD", WithdrawalPriority: 3, SortOrder: 1},
			{ID: "normal", Name: "401k", AmountCents: 1000, Currency: "USD", SortOrder: 2},
			{ID: "first", Name: "Brokerage", AmountCents: 1000, Currency: "USD", WithdrawalPriority: 1, SortOrder: 3},
		},
	})
	if err != nil {
		t.Fatalf("expected projection to calculate, got %v", err)
	}

	want := map[string][]int64{
		"first":  {0, 700, 300, 0},
		"normal": {0, 0, 400, 600},
		"last":   {0, 0, 0, 100},
	}
	for _, item := range projection.Items {
		if got := withdrawalsOf(item); !reflect.DeepEqual(got, want[item.ID]) {
			t.Fatalf("%s withdrawals: expected %v, got %v", item.ID, want[item.ID], got)
		}
	}
	if projection.Totals[3].BalanceCents != 900 || projection.Totals[3].UnfundedWithdrawalCents != 0 {
		t.Fatalf("unexpected final totals: %+v", projection.Totals[3])
	}
}

func TestCalculateSplitsWithinAPriorityGroupByBalance(t *testing.T) {
	projection, err := Calculate(Request{
		DrawdownYears:         1,
		AnnualWithdrawalCents: 300,
		Items: []ItemInput{
			{ID: "a", Name: "Brokerage", AmountCents: 1000, Currency: "USD", WithdrawalPriority: 1, SortOrder: 1},
			{ID: "b", Name: "Savings", AmountCents: 2000, Currency: "USD", WithdrawalPriority: 1, SortOrder: 2},
			{ID: "c", Name: "401k", AmountCents: 5000, Currency: "USD", SortOrder: 3},
		},
	})
	if err != nil {
		t.Fatalf("expected projection to calculate, got %v", err)
	}
	got := []int64{projection.Items[0].YearlyBalances[1].WithdrawalCents, projection.Items[1].YearlyBalances[1].WithdrawalCents, projection.Items[2].YearlyBalances[1].WithdrawalCents}
	if !reflect.DeepEqual(got, []int64{100, 200, 0}) {
		t.Fatalf("expected the first group to share 300 by balance and the 401k to be untouched, got %v", got)
	}
}

// The savings account can't be touched until after year 2. 500 a year:
//
//	year 1: brokerage 600 -> 100
//	year 2: brokerage gives its last 100; 400 is unfunded while 1000 sits locked
//	year 3: savings is available and covers 500
func TestCalculateHoldsBackAccountsUntilTheyAreAvailable(t *testing.T) {
	projection, err := Calculate(Request{
		DrawdownYears:         3,
		AnnualWithdrawalCents: 500,
		Items: []ItemInput{
			{ID: "brokerage", Name: "Brokerage", AmountCents: 600, Currency: "USD", SortOrder: 1},
			{ID: "savings", Name: "Savings", AmountCents: 1000, Currency: "USD", WithdrawalAvailableAfterYear: intPointer(2), SortOrder: 2},
		},
	})
	if err != nil {
		t.Fatalf("expected projection to calculate, got %v", err)
	}

	if got := withdrawalsOf(projection.Items[1]); !reflect.DeepEqual(got, []int64{0, 0, 0, 500}) {
		t.Fatalf("savings withdrawals: expected none until year 3, got %v", got)
	}
	year2 := projection.Totals[2]
	if year2.WithdrawalCents != 100 || year2.UnfundedWithdrawalCents != 400 || year2.LockedBalanceCents != 1000 {
		t.Fatalf("year 2: expected 100 withdrawn, 400 unfunded and 1000 locked, got %+v", year2)
	}
	if projection.Totals[3].LockedBalanceCents != 0 || projection.Totals[3].UnfundedWithdrawalCents != 0 {
		t.Fatalf("year 3: expected savings to cover the withdrawal, got %+v", projection.Totals[3])
	}
}

func TestCalculateLetsAnOwnedAccountBeAvailableBeforeItsOwnerRetires(t *testing.T) {
	people := []PersonInput{{ID: "alex", Name: "Alex", SavingYears: 1}, {ID: "sam", Name: "Sam", SavingYears: 3}}
	items := func(samAvailability *int) []ItemInput {
		return []ItemInput{
			{ID: "a", Name: "Alex 401k", AmountCents: 1000, Currency: "USD", AnnualContributionCents: 100, OwnerID: "alex", SortOrder: 1},
			{ID: "s", Name: "Sam 401k", AmountCents: 1000, Currency: "USD", AnnualContributionCents: 100, OwnerID: "sam", WithdrawalAvailableAfterYear: samAvailability, SortOrder: 2},
			{ID: "j", Name: "Joint brokerage", AmountCents: 1000, Currency: "USD", AnnualContributionCents: 100, SortOrder: 3},
		}
	}
	projection, err := Calculate(Request{People: people, DrawdownYears: 1, GapAnnualWithdrawalCents: 300, AnnualWithdrawalCents: 600, Items: items(intPointer(0))})
	if err != nil {
		t.Fatalf("expected projection to calculate, got %v", err)
	}

	// With every account available, the gap withdrawals are shared by all three.
	if got := balancesOf(projection.Items[1]); !reflect.DeepEqual(got, []int64{1000, 1100, 1100, 1097, 885}) {
		t.Fatalf("Sam's balances: expected withdrawals during the gap years, got %v", got)
	}
	if projection.Totals[2].LockedBalanceCents != 0 {
		t.Fatalf("expected nothing locked, got %d", projection.Totals[2].LockedBalanceCents)
	}
}

func TestCalculateSpreadsAShortfallWhenNothingIsAvailable(t *testing.T) {
	projection, err := Calculate(Request{
		DrawdownYears:         1,
		AnnualWithdrawalCents: 300,
		Items: []ItemInput{
			{ID: "a", Name: "IRA", AmountCents: 1000, Currency: "USD", WithdrawalAvailableAfterYear: intPointer(5), SortOrder: 1},
			{ID: "b", Name: "401k", AmountCents: 2000, Currency: "USD", WithdrawalAvailableAfterYear: intPointer(5), SortOrder: 2},
		},
	})
	if err != nil {
		t.Fatalf("expected projection to calculate, got %v", err)
	}
	year1 := projection.Totals[1]
	if year1.WithdrawalCents != 0 || year1.UnfundedWithdrawalCents != 300 || year1.LockedBalanceCents != 3000 {
		t.Fatalf("expected the whole withdrawal to be unfunded with 3000 locked, got %+v", year1)
	}
	if got := projection.Items[1].YearlyBalances[1].UnfundedWithdrawalCents; got != 200 {
		t.Fatalf("expected the shortfall to be spread by balance (200 on the larger account), got %d", got)
	}
}

func TestCalculateRejectsInvalidWithdrawalSettings(t *testing.T) {
	tests := []struct {
		name string
		item ItemInput
		want string
	}{
		{"priority too high", ItemInput{ID: "a", Name: "A", AmountCents: 1, Currency: "USD", WithdrawalPriority: 4, SortOrder: 1}, "items[0].withdrawalPriority must be 1 (first), 2 (normal) or 3 (last)"},
		{"negative priority", ItemInput{ID: "a", Name: "A", AmountCents: 1, Currency: "USD", WithdrawalPriority: -1, SortOrder: 1}, "items[0].withdrawalPriority"},
		{"negative availability", ItemInput{ID: "a", Name: "A", AmountCents: 1, Currency: "USD", WithdrawalAvailableAfterYear: intPointer(-1), SortOrder: 1}, "items[0].withdrawalAvailableAfterYear must be between 0 and 150"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Calculate(Request{DrawdownYears: 1, AnnualWithdrawalCents: 1, Items: []ItemInput{test.item}})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected error containing %q, got %v", test.want, err)
			}
		})
	}
}
