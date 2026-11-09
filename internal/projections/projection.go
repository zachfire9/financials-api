package projections

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const (
	minimumYears = 1
	maximumYears = 75

	minimumAnnualReturnRateBasisPoints = -10000
	maximumAnnualReturnRateBasisPoints = 100000
	basisPointsPerWhole                = 10000
)

// currencyCodePattern matches ISO 4217-style codes: exactly three uppercase letters.
var currencyCodePattern = regexp.MustCompile(`^[A-Z]{3}$`)

type Phase string

const (
	PhaseStarting Phase = "starting"
	PhaseSaving   Phase = "saving"
	// PhaseTransition covers the years between the first and the last person's
	// retirement: some accounts still receive contributions while the household
	// withdraws GapAnnualWithdrawalCents from the pooled balance.
	PhaseTransition Phase = "transition"
	PhaseDrawdown   Phase = "drawdown"
)

// Request contains the pure domain inputs needed to calculate a whole-year projection.
//
// AnnualWithdrawalInflationRateBasisPoints is the projection's single inflation
// rate. It grows the annual withdrawal every year (including saving years, so
// AnnualWithdrawalCents is in today's dollars) and grows the contribution of
// each item with InflateAnnualContribution set during saving years.
//
// People lets members of a household retire at different times. Each person
// has their own SavingYears (years until they retire), and each item may name
// its owner with OwnerID; items without an owner are joint. When People is set:
//
//   - an item receives contributions until its owner retires (joint items until
//     the last person retires), then switches to its drawdown return rate;
//   - years up to the first retirement are saving years;
//   - years between the first and the last retirement are transition years, in
//     which the household withdraws GapAnnualWithdrawalCents from all items;
//   - the full AnnualWithdrawalCents starts after the last retirement and runs
//     for DrawdownYears.
//
// Request.SavingYears must then be left at 0. Both withdrawal amounts are in
// today's dollars and grow with inflation from year 1.
//
// Setting any of SavingYears, DrawdownYears, AnnualWithdrawalCents, the
// inflation rate, People or GapAnnualWithdrawalCents selects the phase
// projection. The legacy Years-only (accumulation) projection therefore always
// has 0% inflation, so InflateAnnualContribution has no effect there.
type Request struct {
	Years                                    int           `json:"years"`
	SavingYears                              int           `json:"savingYears"`
	DrawdownYears                            int           `json:"drawdownYears"`
	AnnualWithdrawalCents                    int64         `json:"annualWithdrawalCents"`
	AnnualWithdrawalInflationRateBasisPoints int           `json:"annualWithdrawalInflationRateBasisPoints"`
	People                                   []PersonInput `json:"people,omitempty"`
	GapAnnualWithdrawalCents                 int64         `json:"gapAnnualWithdrawalCents,omitempty"`
	Items                                    []ItemInput   `json:"items"`
}

// PersonInput is one member of the household and the number of years until
// they retire and stop contributing to the items they own.
type PersonInput struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	SavingYears int    `json:"savingYears"`
}

// ItemInput is one configurable financial item used by the projection engine.
type ItemInput struct {
	ID                                  string `json:"id"`
	Name                                string `json:"name"`
	AmountCents                         int64  `json:"amountCents"`
	Currency                            string `json:"currency"`
	AnnualReturnRateBasisPoints         int    `json:"annualReturnRateBasisPoints"`
	DrawdownAnnualReturnRateBasisPoints *int   `json:"drawdownAnnualReturnRateBasisPoints,omitempty"`
	AnnualContributionCents             int64  `json:"annualContributionCents"`
	InflateAnnualContribution           bool   `json:"inflateAnnualContribution"`
	OwnerID                             string `json:"ownerId,omitempty"`
	SortOrder                           int    `json:"sortOrder"`
}

// Projection is the deterministic whole-year projection result.
//
// Years = SavingYears + TransitionYears + DrawdownYears. TransitionYears is
// only non-zero when people retire in different years.
type Projection struct {
	Years           int             `json:"years"`
	SavingYears     int             `json:"savingYears"`
	TransitionYears int             `json:"transitionYears"`
	DrawdownYears   int             `json:"drawdownYears"`
	Currency        string          `json:"currency"`
	Items           []ProjectedItem `json:"items"`
	Totals          []YearlyBalance `json:"totals"`
}

// ProjectedItem contains the per-year projection series for one input item.
type ProjectedItem struct {
	ID                                  string          `json:"id"`
	Name                                string          `json:"name"`
	StartingAmountCents                 int64           `json:"startingAmountCents"`
	AnnualReturnRateBasisPoints         int             `json:"annualReturnRateBasisPoints"`
	DrawdownAnnualReturnRateBasisPoints *int            `json:"drawdownAnnualReturnRateBasisPoints,omitempty"`
	AnnualContributionCents             int64           `json:"annualContributionCents"`
	InflateAnnualContribution           bool            `json:"inflateAnnualContribution"`
	OwnerID                             string          `json:"ownerId,omitempty"`
	YearlyBalances                      []YearlyBalance `json:"yearlyBalances"`
}

// YearlyBalance is the end-of-year balance snapshot for a projection year.
type YearlyBalance struct {
	Year                    int   `json:"year"`
	Phase                   Phase `json:"phase"`
	BalanceCents            int64 `json:"balanceCents"`
	ContributionCents       int64 `json:"contributionCents"`
	WithdrawalCents         int64 `json:"withdrawalCents"`
	GrowthCents             int64 `json:"growthCents"`
	UnfundedWithdrawalCents int64 `json:"unfundedWithdrawalCents"`
}

// ValidationError groups one or more projection validation failures.
type ValidationError struct {
	Problems []string
}

func (err ValidationError) Error() string {
	return "invalid projection request: " + strings.Join(err.Problems, "; ")
}

// Calculate projects all input items using annual compounding and end-of-year contributions.
func Calculate(request Request) (Projection, error) {
	if err := validate(request); err != nil {
		return Projection{}, err
	}

	items := append([]ItemInput(nil), request.Items...)
	sort.Slice(items, func(i, j int) bool {
		if items[i].SortOrder != items[j].SortOrder {
			return items[i].SortOrder < items[j].SortOrder
		}
		if items[i].ID != items[j].ID {
			return items[i].ID < items[j].ID
		}
		return items[i].Name < items[j].Name
	})

	if usesPhaseProjection(request) {
		return calculatePhaseProjection(request, items), nil
	}
	return calculateAccumulationProjection(request.Years, items), nil
}

func calculateAccumulationProjection(years int, items []ItemInput) Projection {
	projection := Projection{
		Years:    years,
		Currency: items[0].Currency,
		Items:    make([]ProjectedItem, 0, len(items)),
		Totals:   make([]YearlyBalance, years+1),
	}
	for year := range projection.Totals {
		projection.Totals[year].Year = year
	}

	for _, input := range items {
		projected := newProjectedItem(input, calculateAccumulationItemBalances(input, years))
		addToTotals(projection.Totals, projected.YearlyBalances)
		projection.Items = append(projection.Items, projected)
	}

	return projection
}

func calculatePhaseProjection(request Request, items []ItemInput) Projection {
	firstRetirement, lastRetirement := retirementYears(request)
	retirementYearByItem := itemRetirementYears(request, items, lastRetirement)
	totalYears := lastRetirement + request.DrawdownYears
	projection := Projection{
		Years:           totalYears,
		SavingYears:     firstRetirement,
		TransitionYears: lastRetirement - firstRetirement,
		DrawdownYears:   request.DrawdownYears,
		Currency:        items[0].Currency,
		Items:           make([]ProjectedItem, 0, len(items)),
		Totals:          make([]YearlyBalance, totalYears+1),
	}
	for year := range projection.Totals {
		projection.Totals[year].Year = year
		projection.Totals[year].Phase = phaseForYear(year, firstRetirement, lastRetirement)
	}

	balances := make([]int64, len(items))
	annualContributions := make([]int64, len(items))
	itemYearlyBalances := make([][]YearlyBalance, len(items))
	for index, item := range items {
		balances[index] = item.AmountCents
		annualContributions[index] = item.AnnualContributionCents
		itemYearlyBalances[index] = append(itemYearlyBalances[index], YearlyBalance{
			Year:         0,
			Phase:        PhaseStarting,
			BalanceCents: item.AmountCents,
		})
	}

	// Withdrawal amounts are in today's dollars: year y uses the amount grown by
	// y-1 years of inflation, compounded (and rounded) one year at a time.
	annualWithdrawalCents := request.AnnualWithdrawalCents
	gapWithdrawalCents := request.GapAnnualWithdrawalCents

	for year := 1; year <= totalYears; year++ {
		phase := phaseForYear(year, firstRetirement, lastRetirement)
		priorBalances := append([]int64(nil), balances...)
		growthCents := make([]int64, len(items))
		contributionCents := make([]int64, len(items))
		availableBalances := make([]int64, len(items))
		for index, item := range items {
			returnRate := item.AnnualReturnRateBasisPoints
			if year > retirementYearByItem[index] && item.DrawdownAnnualReturnRateBasisPoints != nil {
				returnRate = *item.DrawdownAnnualReturnRateBasisPoints
			}
			growthCents[index] = roundBasisPointGrowth(balances[index], returnRate)
			if year <= retirementYearByItem[index] {
				contributionCents[index] = annualContributions[index]
			}
			availableBalances[index] = balances[index] + growthCents[index] + contributionCents[index]
		}

		withdrawals := make([]int64, len(items))
		unfundedWithdrawals := make([]int64, len(items))
		if phase != PhaseSaving {
			withdrawalCents := annualWithdrawalCents
			if phase == PhaseTransition {
				withdrawalCents = gapWithdrawalCents
			}
			for index := range availableBalances {
				if availableBalances[index] < 0 {
					availableBalances[index] = 0
				}
			}
			withdrawals, unfundedWithdrawals = fundWithdrawal(withdrawalCents, priorBalances, availableBalances)
		}

		for index, item := range items {
			currentBalance := availableBalances[index] - withdrawals[index]
			itemYearlyBalances[index] = append(itemYearlyBalances[index], YearlyBalance{
				Year:                    year,
				Phase:                   phase,
				BalanceCents:            currentBalance,
				ContributionCents:       contributionCents[index],
				WithdrawalCents:         withdrawals[index],
				GrowthCents:             growthCents[index],
				UnfundedWithdrawalCents: unfundedWithdrawals[index],
			})
			balances[index] = currentBalance
			if contributionCents[index] != 0 && item.InflateAnnualContribution {
				annualContributions[index] += roundBasisPointGrowth(contributionCents[index], request.AnnualWithdrawalInflationRateBasisPoints)
			}
		}

		annualWithdrawalCents += roundBasisPointGrowth(annualWithdrawalCents, request.AnnualWithdrawalInflationRateBasisPoints)
		gapWithdrawalCents += roundBasisPointGrowth(gapWithdrawalCents, request.AnnualWithdrawalInflationRateBasisPoints)
	}

	for index, item := range items {
		projected := newProjectedItem(item, itemYearlyBalances[index])
		addToTotals(projection.Totals, projected.YearlyBalances)
		projection.Items = append(projection.Items, projected)
	}

	return projection
}

// retirementYears returns the years of the first and the last retirement.
// Without people, the whole household retires after request.SavingYears.
func retirementYears(request Request) (int, int) {
	if len(request.People) == 0 {
		return request.SavingYears, request.SavingYears
	}
	first, last := request.People[0].SavingYears, request.People[0].SavingYears
	for _, person := range request.People[1:] {
		first = min(first, person.SavingYears)
		last = max(last, person.SavingYears)
	}
	return first, last
}

// itemRetirementYears returns, per item, the last year it receives
// contributions: its owner's retirement, or the last retirement for joint items.
func itemRetirementYears(request Request, items []ItemInput, lastRetirement int) []int {
	savingYearsByPerson := make(map[string]int, len(request.People))
	for _, person := range request.People {
		savingYearsByPerson[person.ID] = person.SavingYears
	}
	years := make([]int, len(items))
	for index, item := range items {
		years[index] = lastRetirement
		if personYears, ok := savingYearsByPerson[item.OwnerID]; ok {
			years[index] = personYears
		}
	}
	return years
}

func validate(request Request) error {
	var problems []string
	phaseProjection := usesPhaseProjection(request)
	if request.Years != 0 && phaseProjection {
		problems = append(problems, "years and savingYears/drawdownYears are mutually exclusive")
	}
	if !phaseProjection {
		if request.Years < minimumYears || request.Years > maximumYears {
			problems = append(problems, fmt.Sprintf("years must be between %d and %d", minimumYears, maximumYears))
		}
	} else {
		if request.SavingYears < 0 || request.SavingYears > maximumYears {
			problems = append(problems, fmt.Sprintf("savingYears must be between 0 and %d", maximumYears))
		}
		problems = append(problems, peopleProblems(request)...)
		if request.DrawdownYears < 0 || request.DrawdownYears > maximumYears {
			problems = append(problems, fmt.Sprintf("drawdownYears must be between 0 and %d", maximumYears))
		}
		_, lastRetirement := retirementYears(request)
		if lastRetirement+request.DrawdownYears < minimumYears {
			problems = append(problems, "savingYears and drawdownYears must include at least one projected year")
		}
		if request.AnnualWithdrawalCents < 0 {
			problems = append(problems, "annualWithdrawalCents must be greater than or equal to 0")
		}
		if request.DrawdownYears > 0 && request.AnnualWithdrawalCents <= 0 {
			problems = append(problems, "annualWithdrawalCents is required when drawdownYears is greater than 0")
		}
		if request.AnnualWithdrawalInflationRateBasisPoints < 0 || request.AnnualWithdrawalInflationRateBasisPoints > maximumAnnualReturnRateBasisPoints {
			problems = append(problems, fmt.Sprintf("annualWithdrawalInflationRateBasisPoints must be between 0 and %d", maximumAnnualReturnRateBasisPoints))
		}
	}
	if len(request.Items) == 0 {
		problems = append(problems, "items are required")
	}

	personIDs := make(map[string]bool, len(request.People))
	for _, person := range request.People {
		personIDs[person.ID] = true
	}
	var currency string
	for index, item := range request.Items {
		if item.OwnerID != "" && !personIDs[item.OwnerID] {
			if len(request.People) == 0 {
				problems = append(problems, fmt.Sprintf("items[%d].ownerId requires people", index))
			} else {
				problems = append(problems, fmt.Sprintf("items[%d].ownerId must match a person id", index))
			}
		}
		if strings.TrimSpace(item.Name) == "" {
			problems = append(problems, fmt.Sprintf("items[%d].name is required", index))
		}
		if item.AmountCents < 0 {
			problems = append(problems, fmt.Sprintf("items[%d].amountCents must be greater than or equal to 0", index))
		}
		if strings.TrimSpace(item.Currency) == "" {
			problems = append(problems, fmt.Sprintf("items[%d].currency is required", index))
		} else {
			if !currencyCodePattern.MatchString(item.Currency) {
				problems = append(problems, fmt.Sprintf("items[%d].currency must be a 3-letter uppercase code such as USD", index))
			}
			if currency == "" {
				currency = item.Currency
			} else if item.Currency != currency {
				problems = append(problems, "all items must use the same currency")
			}
		}
		if item.AnnualReturnRateBasisPoints < minimumAnnualReturnRateBasisPoints || item.AnnualReturnRateBasisPoints > maximumAnnualReturnRateBasisPoints {
			problems = append(problems, fmt.Sprintf("items[%d].annualReturnRateBasisPoints must be between %d and %d", index, minimumAnnualReturnRateBasisPoints, maximumAnnualReturnRateBasisPoints))
		}
		if item.DrawdownAnnualReturnRateBasisPoints != nil && (*item.DrawdownAnnualReturnRateBasisPoints < minimumAnnualReturnRateBasisPoints || *item.DrawdownAnnualReturnRateBasisPoints > maximumAnnualReturnRateBasisPoints) {
			problems = append(problems, fmt.Sprintf("items[%d].drawdownAnnualReturnRateBasisPoints must be between %d and %d", index, minimumAnnualReturnRateBasisPoints, maximumAnnualReturnRateBasisPoints))
		}
		if item.AnnualContributionCents < 0 {
			problems = append(problems, fmt.Sprintf("items[%d].annualContributionCents must be greater than or equal to 0", index))
		}
		if item.SortOrder < 0 {
			problems = append(problems, fmt.Sprintf("items[%d].sortOrder must be greater than or equal to 0", index))
		}
	}

	if len(problems) > 0 {
		return ValidationError{Problems: problems}
	}
	return nil
}

func peopleProblems(request Request) []string {
	var problems []string
	if request.GapAnnualWithdrawalCents < 0 {
		problems = append(problems, "gapAnnualWithdrawalCents must be greater than or equal to 0")
	}
	if len(request.People) == 0 {
		if request.GapAnnualWithdrawalCents > 0 {
			problems = append(problems, "gapAnnualWithdrawalCents requires people")
		}
		return problems
	}
	if request.SavingYears != 0 {
		problems = append(problems, "savingYears must be 0 when people are provided; set savingYears per person")
	}
	seen := make(map[string]bool, len(request.People))
	for index, person := range request.People {
		if strings.TrimSpace(person.ID) == "" {
			problems = append(problems, fmt.Sprintf("people[%d].id is required", index))
		} else if seen[person.ID] {
			problems = append(problems, fmt.Sprintf("people[%d].id must be unique", index))
		}
		seen[person.ID] = true
		if strings.TrimSpace(person.Name) == "" {
			problems = append(problems, fmt.Sprintf("people[%d].name is required", index))
		}
		if person.SavingYears < 0 || person.SavingYears > maximumYears {
			problems = append(problems, fmt.Sprintf("people[%d].savingYears must be between 0 and %d", index, maximumYears))
		}
	}
	return problems
}

func newProjectedItem(input ItemInput, yearlyBalances []YearlyBalance) ProjectedItem {
	return ProjectedItem{
		ID:                                  input.ID,
		Name:                                input.Name,
		StartingAmountCents:                 input.AmountCents,
		AnnualReturnRateBasisPoints:         input.AnnualReturnRateBasisPoints,
		DrawdownAnnualReturnRateBasisPoints: input.DrawdownAnnualReturnRateBasisPoints,
		AnnualContributionCents:             input.AnnualContributionCents,
		InflateAnnualContribution:           input.InflateAnnualContribution,
		OwnerID:                             input.OwnerID,
		YearlyBalances:                      yearlyBalances,
	}
}

func calculateAccumulationItemBalances(input ItemInput, years int) []YearlyBalance {
	balances := make([]YearlyBalance, 0, years+1)
	previousBalance := input.AmountCents
	balances = append(balances, YearlyBalance{
		Year:         0,
		BalanceCents: previousBalance,
	})

	for year := 1; year <= years; year++ {
		growthCents := roundBasisPointGrowth(previousBalance, input.AnnualReturnRateBasisPoints)
		currentBalance := previousBalance + growthCents + input.AnnualContributionCents
		balances = append(balances, YearlyBalance{
			Year:              year,
			BalanceCents:      currentBalance,
			ContributionCents: input.AnnualContributionCents,
			GrowthCents:       growthCents,
		})
		previousBalance = currentBalance
	}

	return balances
}

func addToTotals(totals []YearlyBalance, balances []YearlyBalance) {
	for year, balance := range balances {
		totals[year].BalanceCents += balance.BalanceCents
		totals[year].ContributionCents += balance.ContributionCents
		totals[year].WithdrawalCents += balance.WithdrawalCents
		totals[year].GrowthCents += balance.GrowthCents
		totals[year].UnfundedWithdrawalCents += balance.UnfundedWithdrawalCents
	}
}

func phaseForYear(year int, firstRetirement int, lastRetirement int) Phase {
	if year == 0 {
		return PhaseStarting
	}
	if year <= firstRetirement {
		return PhaseSaving
	}
	if year <= lastRetirement {
		return PhaseTransition
	}
	return PhaseDrawdown
}

func usesPhaseProjection(request Request) bool {
	return request.SavingYears != 0 || request.DrawdownYears != 0 || request.AnnualWithdrawalCents != 0 ||
		request.AnnualWithdrawalInflationRateBasisPoints != 0 || len(request.People) > 0 || request.GapAnnualWithdrawalCents != 0
}

// fundWithdrawal splits one drawdown year's withdrawal across items.
//
// Each item first covers a share proportional to its start-of-year balance,
// capped at what it has available after growth. Any shortfall is then taken
// from items that still have money, proportionally to what they have left,
// until it is covered or every item is empty. Only what remains uncovered is
// reported as unfunded, attributed by the items' original shares (or evenly
// when no item had a balance). For every year:
// sum(withdrawals) + sum(unfunded) == withdrawalCents.
func fundWithdrawal(withdrawalCents int64, priorBalances []int64, availableBalances []int64) ([]int64, []int64) {
	withdrawals := make([]int64, len(availableBalances))
	unfunded := make([]int64, len(availableBalances))
	if withdrawalCents <= 0 || len(availableBalances) == 0 {
		return withdrawals, unfunded
	}

	shares := allocateWithdrawal(withdrawalCents, priorBalances)
	shortfallCents := withdrawalCents
	for index, share := range shares {
		withdrawals[index] = min(share, availableBalances[index])
		shortfallCents -= withdrawals[index]
	}

	remaining := make([]int64, len(availableBalances))
	for shortfallCents > 0 {
		for index := range remaining {
			remaining[index] = availableBalances[index] - withdrawals[index]
		}
		extra := allocateWithdrawal(shortfallCents, remaining)
		takenCents := int64(0)
		for index := range extra {
			take := min(extra[index], remaining[index])
			withdrawals[index] += take
			takenCents += take
		}
		if takenCents == 0 {
			break
		}
		shortfallCents -= takenCents
	}

	if shortfallCents > 0 {
		weights := shares
		if sumCents(shares) == 0 {
			weights = make([]int64, len(shares))
			for index := range weights {
				weights[index] = 1
			}
		}
		unfunded = allocateWithdrawal(shortfallCents, weights)
	}

	return withdrawals, unfunded
}

func sumCents(values []int64) int64 {
	total := int64(0)
	for _, value := range values {
		total += value
	}
	return total
}

func allocateWithdrawal(withdrawalCents int64, priorBalances []int64) []int64 {
	allocations := make([]int64, len(priorBalances))
	if withdrawalCents <= 0 || len(priorBalances) == 0 {
		return allocations
	}

	totalPriorBalance := int64(0)
	for _, balance := range priorBalances {
		if balance > 0 {
			totalPriorBalance += balance
		}
	}
	if totalPriorBalance <= 0 {
		return allocations
	}

	allocated := int64(0)
	for index, balance := range priorBalances {
		if balance <= 0 {
			continue
		}
		allocations[index] = withdrawalCents * balance / totalPriorBalance
		allocated += allocations[index]
	}

	remaining := withdrawalCents - allocated
	for index := range allocations {
		if remaining == 0 {
			break
		}
		if priorBalances[index] <= 0 {
			continue
		}
		allocations[index]++
		remaining--
	}

	return allocations
}

func roundBasisPointGrowth(balanceCents int64, annualReturnRateBasisPoints int) int64 {
	numerator := balanceCents * int64(annualReturnRateBasisPoints)
	if numerator >= 0 {
		return (numerator + basisPointsPerWhole/2) / basisPointsPerWhole
	}
	return (numerator - basisPointsPerWhole/2) / basisPointsPerWhole
}
