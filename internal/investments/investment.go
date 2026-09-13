package investments

import (
	"fmt"
	"strings"
	"time"
)

// InvestmentType describes the broad category for a current investment record.
type InvestmentType string

const (
	InvestmentTypeTaxableBrokerage InvestmentType = "taxable_brokerage"
	InvestmentTypeRetirement401k   InvestmentType = "retirement_401k"
	InvestmentTypeTraditionalIRA   InvestmentType = "traditional_ira"
	InvestmentTypeRothIRA          InvestmentType = "roth_ira"
	InvestmentTypeHSA              InvestmentType = "hsa"
	InvestmentTypeCash             InvestmentType = "cash"
	InvestmentTypeOther            InvestmentType = "other"
)

var supportedInvestmentTypes = map[InvestmentType]struct{}{
	InvestmentTypeTaxableBrokerage: {},
	InvestmentTypeRetirement401k:   {},
	InvestmentTypeTraditionalIRA:   {},
	InvestmentTypeRothIRA:          {},
	InvestmentTypeHSA:              {},
	InvestmentTypeCash:             {},
	InvestmentTypeOther:            {},
}

// Investment is the API-facing representation of a current investment.
type Investment struct {
	ID                      string         `json:"id"`
	Name                    string         `json:"name"`
	Type                    InvestmentType `json:"type"`
	Institution             string         `json:"institution"`
	BalanceCents            int64          `json:"balanceCents"`
	Currency                string         `json:"currency"`
	AnnualContributionCents int64          `json:"annualContributionCents"`
	CreatedAt               time.Time      `json:"createdAt"`
	UpdatedAt               time.Time      `json:"updatedAt"`
}

// CreateInvestmentRequest contains user-editable fields for a new investment.
type CreateInvestmentRequest struct {
	Name                    string         `json:"name"`
	Type                    InvestmentType `json:"type"`
	Institution             string         `json:"institution"`
	BalanceCents            int64          `json:"balanceCents"`
	Currency                string         `json:"currency"`
	AnnualContributionCents int64          `json:"annualContributionCents"`
}

// UpdateInvestmentRequest contains the full editable investment payload.
type UpdateInvestmentRequest struct {
	Name                    string         `json:"name"`
	Type                    InvestmentType `json:"type"`
	Institution             string         `json:"institution"`
	BalanceCents            int64          `json:"balanceCents"`
	Currency                string         `json:"currency"`
	AnnualContributionCents int64          `json:"annualContributionCents"`
}

// Validate checks that a create request is public-safe and internally consistent.
func (request CreateInvestmentRequest) Validate() error {
	return validateInvestmentFields(
		request.Name,
		request.Type,
		request.Institution,
		request.BalanceCents,
		request.Currency,
		request.AnnualContributionCents,
	)
}

// Validate checks that an update request is public-safe and internally consistent.
func (request UpdateInvestmentRequest) Validate() error {
	return validateInvestmentFields(
		request.Name,
		request.Type,
		request.Institution,
		request.BalanceCents,
		request.Currency,
		request.AnnualContributionCents,
	)
}

func validateInvestmentFields(name string, investmentType InvestmentType, institution string, balanceCents int64, currency string, annualContributionCents int64) error {
	var problems []string

	if strings.TrimSpace(name) == "" {
		problems = append(problems, "name is required")
	}
	if _, ok := supportedInvestmentTypes[investmentType]; !ok {
		problems = append(problems, fmt.Sprintf("type %q is not supported", investmentType))
	}
	if strings.TrimSpace(institution) == "" {
		problems = append(problems, "institution is required")
	}
	if balanceCents < 0 {
		problems = append(problems, "balanceCents must be greater than or equal to 0")
	}
	if strings.TrimSpace(currency) == "" {
		problems = append(problems, "currency is required")
	}
	if currency != strings.ToUpper(currency) {
		problems = append(problems, "currency must be uppercase ISO-style text")
	}
	if annualContributionCents < 0 {
		problems = append(problems, "annualContributionCents must be greater than or equal to 0")
	}

	if len(problems) > 0 {
		return ValidationError{Problems: problems}
	}
	return nil
}

// ValidationError groups one or more investment validation failures.
type ValidationError struct {
	Problems []string
}

func (err ValidationError) Error() string {
	return "invalid investment: " + strings.Join(err.Problems, "; ")
}
