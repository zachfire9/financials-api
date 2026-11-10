package projections

import (
	"reflect"
	"strings"
	"testing"
)

func intPointer(value int) *int {
	return &value
}

func balancesOf(item ProjectedItem) []int64 {
	balances := make([]int64, len(item.YearlyBalances))
	for index, balance := range item.YearlyBalances {
		balances[index] = balance.BalanceCents
	}
	return balances
}

// Alex retires after year 1, Sam after year 3. With 0% returns:
//
//	year 1 (saving):     every item +100 contribution -> 1100 each
//	year 2 (transition): Alex stops contributing. Sam's account can't be
//	                     withdrawn from until Sam retires, so gap 300 comes from
//	                     Alex's and the joint account: 150/150
//	year 3 (transition): gap 300 split by prior balances 950/1050 -> 143/157
//	year 4 (drawdown):   nobody contributes; everything is available;
//	                     600 split by 807/1300/993 -> 157/251/192
func TestCalculateStopsContributionsAtEachOwnersRetirement(t *testing.T) {
	projection, err := Calculate(Request{
		People: []PersonInput{
			{ID: "alex", Name: "Alex", SavingYears: 1},
			{ID: "sam", Name: "Sam", SavingYears: 3},
		},
		DrawdownYears:            1,
		GapAnnualWithdrawalCents: 300,
		AnnualWithdrawalCents:    600,
		Items: []ItemInput{
			{ID: "a", Name: "Alex 401k", AmountCents: 1000, Currency: "USD", AnnualContributionCents: 100, OwnerID: "alex", SortOrder: 1},
			{ID: "s", Name: "Sam 401k", AmountCents: 1000, Currency: "USD", AnnualContributionCents: 100, OwnerID: "sam", SortOrder: 2},
			{ID: "j", Name: "Joint brokerage", AmountCents: 1000, Currency: "USD", AnnualContributionCents: 100, SortOrder: 3},
		},
	})
	if err != nil {
		t.Fatalf("expected projection to calculate, got %v", err)
	}

	if projection.Years != 4 || projection.SavingYears != 1 || projection.TransitionYears != 2 || projection.DrawdownYears != 1 {
		t.Fatalf("unexpected phase lengths: years=%d saving=%d transition=%d drawdown=%d",
			projection.Years, projection.SavingYears, projection.TransitionYears, projection.DrawdownYears)
	}

	wantPhases := []Phase{PhaseStarting, PhaseSaving, PhaseTransition, PhaseTransition, PhaseDrawdown}
	for year, total := range projection.Totals {
		if total.Phase != wantPhases[year] {
			t.Fatalf("year %d: expected phase %q, got %q", year, wantPhases[year], total.Phase)
		}
	}

	want := map[string][]int64{
		"a": {1000, 1100, 950, 807, 650},
		"s": {1000, 1100, 1200, 1300, 1049},
		"j": {1000, 1100, 1050, 993, 801},
	}
	for _, item := range projection.Items {
		if got := balancesOf(item); !reflect.DeepEqual(got, want[item.ID]) {
			t.Fatalf("%s balances: expected %v, got %v", item.ID, want[item.ID], got)
		}
	}

	wantContributions := map[string][]int64{
		"a": {0, 100, 0, 0, 0},
		"s": {0, 100, 100, 100, 0},
		"j": {0, 100, 100, 100, 0},
	}
	for _, item := range projection.Items {
		for year, balance := range item.YearlyBalances {
			if balance.ContributionCents != wantContributions[item.ID][year] {
				t.Fatalf("%s year %d: expected contribution %d, got %d", item.ID, year, wantContributions[item.ID][year], balance.ContributionCents)
			}
		}
	}

	wantTotals := []struct{ balance, contribution, withdrawal int64 }{
		{3000, 0, 0}, {3300, 300, 0}, {3200, 200, 300}, {3100, 200, 300}, {2500, 0, 600},
	}
	for year, total := range projection.Totals {
		got := struct{ balance, contribution, withdrawal int64 }{total.BalanceCents, total.ContributionCents, total.WithdrawalCents}
		if got != wantTotals[year] {
			t.Fatalf("year %d totals: expected %+v, got %+v", year, wantTotals[year], got)
		}
	}
	if got := []int64{projection.Totals[2].LockedBalanceCents, projection.Totals[3].LockedBalanceCents, projection.Totals[4].LockedBalanceCents}; !reflect.DeepEqual(got, []int64{1200, 1300, 0}) {
		t.Fatalf("expected Sam's balance to be locked until Sam retires, got %v", got)
	}
	if projection.Items[0].OwnerID != "alex" || projection.Items[2].OwnerID != "" {
		t.Fatalf("expected owners to be echoed, got %q and %q", projection.Items[0].OwnerID, projection.Items[2].OwnerID)
	}
}

func TestCalculateSwitchesToDrawdownReturnWhenTheOwnerRetires(t *testing.T) {
	projection, err := Calculate(Request{
		People: []PersonInput{
			{ID: "alex", Name: "Alex", SavingYears: 1},
			{ID: "sam", Name: "Sam", SavingYears: 2},
		},
		Items: []ItemInput{
			{ID: "a", Name: "Alex IRA", AmountCents: 10000, Currency: "USD", AnnualReturnRateBasisPoints: 1000, DrawdownAnnualReturnRateBasisPoints: intPointer(0), OwnerID: "alex", SortOrder: 1},
			{ID: "s", Name: "Sam IRA", AmountCents: 10000, Currency: "USD", AnnualReturnRateBasisPoints: 1000, DrawdownAnnualReturnRateBasisPoints: intPointer(0), OwnerID: "sam", SortOrder: 2},
		},
	})
	if err != nil {
		t.Fatalf("expected projection to calculate, got %v", err)
	}

	if got := balancesOf(projection.Items[0]); !reflect.DeepEqual(got, []int64{10000, 11000, 11000}) {
		t.Fatalf("Alex should stop growing at 10%% after retiring, got %v", got)
	}
	if got := balancesOf(projection.Items[1]); !reflect.DeepEqual(got, []int64{10000, 11000, 12100}) {
		t.Fatalf("Sam should keep 10%% growth while working, got %v", got)
	}
}

func TestCalculateInflatesGapAndFullWithdrawalsFromYearOne(t *testing.T) {
	projection, err := Calculate(Request{
		People: []PersonInput{
			{ID: "alex", Name: "Alex", SavingYears: 0},
			{ID: "sam", Name: "Sam", SavingYears: 1},
		},
		DrawdownYears:                            1,
		GapAnnualWithdrawalCents:                 1000,
		AnnualWithdrawalCents:                    2000,
		AnnualWithdrawalInflationRateBasisPoints: 1000,
		Items: []ItemInput{
			{ID: "j", Name: "Joint savings", AmountCents: 100000, Currency: "USD", SortOrder: 1},
		},
	})
	if err != nil {
		t.Fatalf("expected projection to calculate, got %v", err)
	}

	if projection.SavingYears != 0 || projection.TransitionYears != 1 {
		t.Fatalf("expected no saving years and one transition year, got saving=%d transition=%d", projection.SavingYears, projection.TransitionYears)
	}
	if got := projection.Totals[1].WithdrawalCents; got != 1000 {
		t.Fatalf("year 1 should withdraw the gap amount in today's dollars, got %d", got)
	}
	if got := projection.Totals[2].WithdrawalCents; got != 2200 {
		t.Fatalf("year 2 should withdraw the full amount after one year of 10%% inflation, got %d", got)
	}
}

func TestCalculateWithPeopleRetiringTogetherMatchesHouseholdSavingYears(t *testing.T) {
	items := []ItemInput{
		{ID: "a", Name: "Brokerage", AmountCents: 250000, Currency: "USD", AnnualReturnRateBasisPoints: 650, AnnualContributionCents: 12000, InflateAnnualContribution: true, OwnerID: "alex", SortOrder: 1},
		{ID: "s", Name: "Savings", AmountCents: 90000, Currency: "USD", AnnualReturnRateBasisPoints: 300, DrawdownAnnualReturnRateBasisPoints: intPointer(200), AnnualContributionCents: 4000, SortOrder: 2},
	}
	household, err := Calculate(Request{
		SavingYears: 5, DrawdownYears: 6, AnnualWithdrawalCents: 40000, AnnualWithdrawalInflationRateBasisPoints: 250,
		Items: withoutOwners(items),
	})
	if err != nil {
		t.Fatalf("household projection: %v", err)
	}
	people, err := Calculate(Request{
		People:        []PersonInput{{ID: "alex", Name: "Alex", SavingYears: 5}, {ID: "sam", Name: "Sam", SavingYears: 5}},
		DrawdownYears: 6, AnnualWithdrawalCents: 40000, AnnualWithdrawalInflationRateBasisPoints: 250,
		Items: items,
	})
	if err != nil {
		t.Fatalf("people projection: %v", err)
	}

	if people.TransitionYears != 0 || !reflect.DeepEqual(household.Totals, people.Totals) {
		t.Fatalf("expected identical totals when everyone retires together\nhousehold: %+v\npeople:    %+v", household.Totals, people.Totals)
	}
}

func TestCalculateIgnoresOwnersWithoutPeople(t *testing.T) {
	item := ItemInput{ID: "a", Name: "Alex 401k", AmountCents: 1000, Currency: "USD", AnnualContributionCents: 100, OwnerID: "alex", SortOrder: 1}
	withOwner, err := Calculate(Request{SavingYears: 2, Items: []ItemInput{item}})
	if err != nil {
		t.Fatalf("expected owners to be ignored without people, got %v", err)
	}
	withoutOwner, err := Calculate(Request{SavingYears: 2, Items: withoutOwners([]ItemInput{item})})
	if err != nil {
		t.Fatalf("projection without owner: %v", err)
	}
	if !reflect.DeepEqual(withOwner.Totals, withoutOwner.Totals) {
		t.Fatalf("expected identical totals, got %+v and %+v", withOwner.Totals, withoutOwner.Totals)
	}
}

func withoutOwners(items []ItemInput) []ItemInput {
	copied := append([]ItemInput(nil), items...)
	for index := range copied {
		copied[index].OwnerID = ""
	}
	return copied
}

func TestCalculateRejectsInvalidPeopleRequests(t *testing.T) {
	item := ItemInput{ID: "a", Name: "Account", AmountCents: 1000, Currency: "USD", SortOrder: 1}
	owned := item
	owned.OwnerID = "alex"
	strangerOwned := item
	strangerOwned.OwnerID = "pat"
	twoPeople := []PersonInput{{ID: "alex", Name: "Alex", SavingYears: 2}, {ID: "sam", Name: "Sam", SavingYears: 4}}

	tests := []struct {
		name    string
		request Request
		want    string
	}{
		{"unknown owner", Request{People: twoPeople, Items: []ItemInput{strangerOwned}}, "items[0].ownerId must match a person id"},
		{"household saving years with people", Request{People: twoPeople, SavingYears: 3, Items: []ItemInput{item}}, "savingYears must be 0 when people are provided"},
		{"duplicate person id", Request{People: []PersonInput{{ID: "alex", Name: "Alex", SavingYears: 1}, {ID: "alex", Name: "Al", SavingYears: 2}}, Items: []ItemInput{item}}, "people[1].id must be unique"},
		{"missing person id", Request{People: []PersonInput{{Name: "Alex", SavingYears: 1}}, Items: []ItemInput{item}}, "people[0].id is required"},
		{"missing person name", Request{People: []PersonInput{{ID: "alex", SavingYears: 1}}, Items: []ItemInput{item}}, "people[0].name is required"},
		{"person years out of range", Request{People: []PersonInput{{ID: "alex", Name: "Alex", SavingYears: 76}}, Items: []ItemInput{item}}, "people[0].savingYears must be between 0 and 75"},
		{"gap without people", Request{SavingYears: 2, GapAnnualWithdrawalCents: 500, Items: []ItemInput{item}}, "gapAnnualWithdrawalCents requires people"},
		{"negative gap", Request{People: twoPeople, GapAnnualWithdrawalCents: -1, Items: []ItemInput{item}}, "gapAnnualWithdrawalCents must be greater than or equal to 0"},
		{"no projected years", Request{People: []PersonInput{{ID: "alex", Name: "Alex", SavingYears: 0}}, Items: []ItemInput{item}}, "must include at least one projected year"},
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
