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

// Request contains the pure domain inputs needed to calculate a whole-year projection.
type Request struct {
	Years int         `json:"years"`
	Items []ItemInput `json:"items"`
}

// ItemInput is one configurable financial item used by the projection engine.
type ItemInput struct {
	ID                          string `json:"id"`
	Name                        string `json:"name"`
	AmountCents                 int64  `json:"amountCents"`
	Currency                    string `json:"currency"`
	AnnualReturnRateBasisPoints int    `json:"annualReturnRateBasisPoints"`
	AnnualContributionCents     int64  `json:"annualContributionCents"`
	SortOrder                   int    `json:"sortOrder"`
}

// Projection is the deterministic whole-year projection result.
type Projection struct {
	Years    int             `json:"years"`
	Currency string          `json:"currency"`
	Items    []ProjectedItem `json:"items"`
	Totals   []YearlyBalance `json:"totals"`
}

// ProjectedItem contains the per-year projection series for one input item.
type ProjectedItem struct {
	ID                          string          `json:"id"`
	Name                        string          `json:"name"`
	StartingAmountCents         int64           `json:"startingAmountCents"`
	AnnualReturnRateBasisPoints int             `json:"annualReturnRateBasisPoints"`
	AnnualContributionCents     int64           `json:"annualContributionCents"`
	YearlyBalances              []YearlyBalance `json:"yearlyBalances"`
}

// YearlyBalance is the end-of-year balance snapshot for a projection year.
type YearlyBalance struct {
	Year              int   `json:"year"`
	BalanceCents      int64 `json:"balanceCents"`
	ContributionCents int64 `json:"contributionCents"`
	GrowthCents       int64 `json:"growthCents"`
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

	projection := Projection{
		Years:    request.Years,
		Currency: items[0].Currency,
		Items:    make([]ProjectedItem, 0, len(items)),
		Totals:   make([]YearlyBalance, request.Years+1),
	}
	for year := range projection.Totals {
		projection.Totals[year].Year = year
	}

	for _, input := range items {
		projected := ProjectedItem{
			ID:                          input.ID,
			Name:                        input.Name,
			StartingAmountCents:         input.AmountCents,
			AnnualReturnRateBasisPoints: input.AnnualReturnRateBasisPoints,
			AnnualContributionCents:     input.AnnualContributionCents,
			YearlyBalances:              calculateItemBalances(input, request.Years),
		}
		for year, balance := range projected.YearlyBalances {
			projection.Totals[year].BalanceCents += balance.BalanceCents
			projection.Totals[year].ContributionCents += balance.ContributionCents
			projection.Totals[year].GrowthCents += balance.GrowthCents
		}
		projection.Items = append(projection.Items, projected)
	}

	return projection, nil
}

func validate(request Request) error {
	var problems []string
	if request.Years < minimumYears || request.Years > maximumYears {
		problems = append(problems, fmt.Sprintf("years must be between %d and %d", minimumYears, maximumYears))
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

func calculateItemBalances(input ItemInput, years int) []YearlyBalance {
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

func roundBasisPointGrowth(balanceCents int64, annualReturnRateBasisPoints int) int64 {
	numerator := balanceCents * int64(annualReturnRateBasisPoints)
	if numerator >= 0 {
		return (numerator + basisPointsPerWhole/2) / basisPointsPerWhole
	}
	return (numerator - basisPointsPerWhole/2) / basisPointsPerWhole
}
