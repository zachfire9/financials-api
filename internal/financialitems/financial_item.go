package financialitems

import (
	"fmt"
	"strings"
	"time"
)

const (
	minimumAnnualReturnRateBasisPoints = -10000
	maximumAnnualReturnRateBasisPoints = 100000
)

// FinancialItem is the API-facing representation of a configurable projection input.
type FinancialItem struct {
	ID                          string    `json:"id"`
	Name                        string    `json:"name"`
	AmountCents                 int64     `json:"amountCents"`
	Currency                    string    `json:"currency"`
	AnnualReturnRateBasisPoints int       `json:"annualReturnRateBasisPoints"`
	AnnualContributionCents     int64     `json:"annualContributionCents"`
	SortOrder                   int       `json:"sortOrder"`
	CreatedAt                   time.Time `json:"createdAt"`
	UpdatedAt                   time.Time `json:"updatedAt"`
}

// CreateFinancialItemRequest contains user-editable fields for a new financial item.
type CreateFinancialItemRequest struct {
	Name                        string `json:"name"`
	AmountCents                 int64  `json:"amountCents"`
	Currency                    string `json:"currency"`
	AnnualReturnRateBasisPoints int    `json:"annualReturnRateBasisPoints"`
	AnnualContributionCents     int64  `json:"annualContributionCents"`
	SortOrder                   int    `json:"sortOrder"`
}

// UpdateFinancialItemRequest contains the full editable financial item payload.
type UpdateFinancialItemRequest struct {
	Name                        string `json:"name"`
	AmountCents                 int64  `json:"amountCents"`
	Currency                    string `json:"currency"`
	AnnualReturnRateBasisPoints int    `json:"annualReturnRateBasisPoints"`
	AnnualContributionCents     int64  `json:"annualContributionCents"`
	SortOrder                   int    `json:"sortOrder"`
}

// Validate checks that a create request is public-safe and internally consistent.
func (request CreateFinancialItemRequest) Validate() error {
	return validateFinancialItemFields(
		request.Name,
		request.AmountCents,
		request.Currency,
		request.AnnualReturnRateBasisPoints,
		request.AnnualContributionCents,
		request.SortOrder,
	)
}

// Validate checks that an update request is public-safe and internally consistent.
func (request UpdateFinancialItemRequest) Validate() error {
	return validateFinancialItemFields(
		request.Name,
		request.AmountCents,
		request.Currency,
		request.AnnualReturnRateBasisPoints,
		request.AnnualContributionCents,
		request.SortOrder,
	)
}

func validateFinancialItemFields(name string, amountCents int64, currency string, annualReturnRateBasisPoints int, annualContributionCents int64, sortOrder int) error {
	var problems []string

	if strings.TrimSpace(name) == "" {
		problems = append(problems, "name is required")
	}
	if amountCents < 0 {
		problems = append(problems, "amountCents must be greater than or equal to 0")
	}
	if strings.TrimSpace(currency) == "" {
		problems = append(problems, "currency is required")
	}
	if currency != strings.ToUpper(currency) {
		problems = append(problems, "currency must be uppercase ISO-style text")
	}
	if annualReturnRateBasisPoints < minimumAnnualReturnRateBasisPoints || annualReturnRateBasisPoints > maximumAnnualReturnRateBasisPoints {
		problems = append(problems, fmt.Sprintf("annualReturnRateBasisPoints must be between %d and %d", minimumAnnualReturnRateBasisPoints, maximumAnnualReturnRateBasisPoints))
	}
	if annualContributionCents < 0 {
		problems = append(problems, "annualContributionCents must be greater than or equal to 0")
	}
	if sortOrder < 0 {
		problems = append(problems, "sortOrder must be greater than or equal to 0")
	}

	if len(problems) > 0 {
		return ValidationError{Problems: problems}
	}
	return nil
}

// ValidationError groups one or more financial item validation failures.
type ValidationError struct {
	Problems []string
}

func (err ValidationError) Error() string {
	return "invalid financial item: " + strings.Join(err.Problems, "; ")
}
