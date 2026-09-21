package projections

import (
	"fmt"
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

type Phase string

const (
	PhaseStarting Phase = "starting"
	PhaseSaving   Phase = "saving"
	PhaseDrawdown Phase = "drawdown"
)

// Request contains the pure domain inputs needed to calculate a whole-year projection.
type Request struct {
	Years                                    int         `json:"years"`
	SavingYears                              int         `json:"savingYears"`
	DrawdownYears                            int         `json:"drawdownYears"`
	AnnualWithdrawalCents                    int64       `json:"annualWithdrawalCents"`
	AnnualWithdrawalInflationRateBasisPoints int         `json:"annualWithdrawalInflationRateBasisPoints"`
	Items                                    []ItemInput `json:"items"`
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
	SortOrder                           int    `json:"sortOrder"`
}

// Projection is the deterministic whole-year projection result.
type Projection struct {
	Years         int             `json:"years"`
	SavingYears   int             `json:"savingYears"`
	DrawdownYears int             `json:"drawdownYears"`
	Currency      string          `json:"currency"`
	Items         []ProjectedItem `json:"items"`
	Totals        []YearlyBalance `json:"totals"`
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
	totalYears := request.SavingYears + request.DrawdownYears
	projection := Projection{
		Years:         totalYears,
		SavingYears:   request.SavingYears,
		DrawdownYears: request.DrawdownYears,
		Currency:      items[0].Currency,
		Items:         make([]ProjectedItem, 0, len(items)),
		Totals:        make([]YearlyBalance, totalYears+1),
	}
	for year := range projection.Totals {
		projection.Totals[year].Year = year
		projection.Totals[year].Phase = phaseForYear(year, request.SavingYears)
	}

	balances := make([]int64, len(items))
	itemYearlyBalances := make([][]YearlyBalance, len(items))
	for index, item := range items {
		balances[index] = item.AmountCents
		itemYearlyBalances[index] = append(itemYearlyBalances[index], YearlyBalance{
			Year:         0,
			Phase:        PhaseStarting,
			BalanceCents: item.AmountCents,
		})
	}

	annualContributions := make([]int64, len(items))
	for index, item := range items {
		annualContributions[index] = item.AnnualContributionCents
	}
	for year := 1; year <= request.SavingYears; year++ {
		for index, item := range items {
			contributionCents := annualContributions[index]
			growthCents := roundBasisPointGrowth(balances[index], item.AnnualReturnRateBasisPoints)
			currentBalance := balances[index] + growthCents + contributionCents
			itemYearlyBalances[index] = append(itemYearlyBalances[index], YearlyBalance{
				Year:              year,
				Phase:             PhaseSaving,
				BalanceCents:      currentBalance,
				ContributionCents: contributionCents,
				GrowthCents:       growthCents,
			})
			balances[index] = currentBalance
			if item.InflateAnnualContribution {
				annualContributions[index] += roundBasisPointGrowth(contributionCents, request.AnnualWithdrawalInflationRateBasisPoints)
			}
		}
	}

	annualWithdrawalCents := request.AnnualWithdrawalCents
	for range request.SavingYears {
		annualWithdrawalCents += roundBasisPointGrowth(annualWithdrawalCents, request.AnnualWithdrawalInflationRateBasisPoints)
	}
	for drawdownYear := 1; drawdownYear <= request.DrawdownYears; drawdownYear++ {
		year := request.SavingYears + drawdownYear
		priorBalances := append([]int64(nil), balances...)
		withdrawalAllocations := allocateWithdrawal(annualWithdrawalCents, priorBalances)
		for index, item := range items {
			drawdownReturnRate := item.AnnualReturnRateBasisPoints
			if item.DrawdownAnnualReturnRateBasisPoints != nil {
				drawdownReturnRate = *item.DrawdownAnnualReturnRateBasisPoints
			}
			growthCents := roundBasisPointGrowth(balances[index], drawdownReturnRate)
			availableBalance := balances[index] + growthCents
			if availableBalance < 0 {
				availableBalance = 0
			}
			withdrawalCents := withdrawalAllocations[index]
			unfundedWithdrawalCents := int64(0)
			if withdrawalCents > availableBalance {
				unfundedWithdrawalCents = withdrawalCents - availableBalance
				withdrawalCents = availableBalance
			}
			currentBalance := availableBalance - withdrawalCents
			itemYearlyBalances[index] = append(itemYearlyBalances[index], YearlyBalance{
				Year:                    year,
				Phase:                   PhaseDrawdown,
				BalanceCents:            currentBalance,
				WithdrawalCents:         withdrawalCents,
				GrowthCents:             growthCents,
				UnfundedWithdrawalCents: unfundedWithdrawalCents,
			})
			balances[index] = currentBalance
		}
		annualWithdrawalCents += roundBasisPointGrowth(annualWithdrawalCents, request.AnnualWithdrawalInflationRateBasisPoints)
	}

	for index, item := range items {
		projected := newProjectedItem(item, itemYearlyBalances[index])
		addToTotals(projection.Totals, projected.YearlyBalances)
		projection.Items = append(projection.Items, projected)
	}

	return projection
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
		if request.DrawdownYears < 0 || request.DrawdownYears > maximumYears {
			problems = append(problems, fmt.Sprintf("drawdownYears must be between 0 and %d", maximumYears))
		}
		if request.SavingYears+request.DrawdownYears < minimumYears {
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

	var currency string
	for index, item := range request.Items {
		if strings.TrimSpace(item.Name) == "" {
			problems = append(problems, fmt.Sprintf("items[%d].name is required", index))
		}
		if item.AmountCents < 0 {
			problems = append(problems, fmt.Sprintf("items[%d].amountCents must be greater than or equal to 0", index))
		}
		if strings.TrimSpace(item.Currency) == "" {
			problems = append(problems, fmt.Sprintf("items[%d].currency is required", index))
		} else {
			if item.Currency != strings.ToUpper(item.Currency) {
				problems = append(problems, fmt.Sprintf("items[%d].currency must be uppercase ISO-style text", index))
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

func newProjectedItem(input ItemInput, yearlyBalances []YearlyBalance) ProjectedItem {
	return ProjectedItem{
		ID:                                  input.ID,
		Name:                                input.Name,
		StartingAmountCents:                 input.AmountCents,
		AnnualReturnRateBasisPoints:         input.AnnualReturnRateBasisPoints,
		DrawdownAnnualReturnRateBasisPoints: input.DrawdownAnnualReturnRateBasisPoints,
		AnnualContributionCents:             input.AnnualContributionCents,
		InflateAnnualContribution:           input.InflateAnnualContribution,
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

func phaseForYear(year int, savingYears int) Phase {
	if year == 0 {
		return PhaseStarting
	}
	if year <= savingYears {
		return PhaseSaving
	}
	return PhaseDrawdown
}

func usesPhaseProjection(request Request) bool {
	return request.SavingYears != 0 || request.DrawdownYears != 0 || request.AnnualWithdrawalCents != 0 || request.AnnualWithdrawalInflationRateBasisPoints != 0
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
