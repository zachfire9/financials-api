package financialitems

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// currencyCodePattern matches ISO 4217-style codes: exactly three uppercase letters.
// Browsers' Intl.NumberFormat throws on malformed codes, so anything else would
// break currency formatting in the UI.
var currencyCodePattern = regexp.MustCompile(`^[A-Z]{3}$`)

// IsValidCurrencyCode reports whether code is exactly three uppercase letters.
func IsValidCurrencyCode(code string) bool {
	return currencyCodePattern.MatchString(code)
}

const (
	minimumAnnualReturnRateBasisPoints = -10000
	maximumAnnualReturnRateBasisPoints = 100000
)

// FinancialItem is the API-facing representation of a configurable projection input.
type FinancialItem struct {
	ID                                  string    `json:"id"`
	Name                                string    `json:"name"`
	AmountCents                         int64     `json:"amountCents"`
	Currency                            string    `json:"currency"`
	AnnualReturnRateBasisPoints         int       `json:"annualReturnRateBasisPoints"`
	DrawdownAnnualReturnRateBasisPoints *int      `json:"drawdownAnnualReturnRateBasisPoints,omitempty"`
	AnnualContributionCents             int64     `json:"annualContributionCents"`
	InflateAnnualContribution           bool      `json:"inflateAnnualContribution"`
	OwnerID                             string    `json:"ownerId,omitempty"`
	WithdrawalPriority                  int       `json:"withdrawalPriority,omitempty"`
	WithdrawalAvailableAfterYear        *int      `json:"withdrawalAvailableAfterYear,omitempty"`
	SortOrder                           int       `json:"sortOrder"`
	CreatedAt                           time.Time `json:"createdAt"`
	UpdatedAt                           time.Time `json:"updatedAt"`
}

// maximumOwnerIDLength caps ownerId, an optional opaque reference to a person
// in projection settings (for example "person-1"). An empty ownerId means the
// item is jointly owned.
const maximumOwnerIDLength = 64

// Withdrawal settings, matching the projection engine: priority 1 (first),
// 2 (normal) or 3 (last), 0 meaning normal; availability as the last year an
// item can't be withdrawn from.
const (
	maximumWithdrawalPriority           = 3
	maximumWithdrawalAvailableAfterYear = 150
)

// CreateFinancialItemRequest contains user-editable fields for a new financial item.
type CreateFinancialItemRequest struct {
	Name                                string `json:"name"`
	AmountCents                         int64  `json:"amountCents"`
	Currency                            string `json:"currency"`
	AnnualReturnRateBasisPoints         int    `json:"annualReturnRateBasisPoints"`
	DrawdownAnnualReturnRateBasisPoints *int   `json:"drawdownAnnualReturnRateBasisPoints,omitempty"`
	AnnualContributionCents             int64  `json:"annualContributionCents"`
	InflateAnnualContribution           bool   `json:"inflateAnnualContribution"`
	OwnerID                             string `json:"ownerId,omitempty"`
	WithdrawalPriority                  int    `json:"withdrawalPriority,omitempty"`
	WithdrawalAvailableAfterYear        *int   `json:"withdrawalAvailableAfterYear,omitempty"`
	SortOrder                           int    `json:"sortOrder"`
}

// UpdateFinancialItemRequest contains the full editable financial item payload.
type UpdateFinancialItemRequest struct {
	Name                                string `json:"name"`
	AmountCents                         int64  `json:"amountCents"`
	Currency                            string `json:"currency"`
	AnnualReturnRateBasisPoints         int    `json:"annualReturnRateBasisPoints"`
	DrawdownAnnualReturnRateBasisPoints *int   `json:"drawdownAnnualReturnRateBasisPoints,omitempty"`
	AnnualContributionCents             int64  `json:"annualContributionCents"`
	InflateAnnualContribution           bool   `json:"inflateAnnualContribution"`
	OwnerID                             string `json:"ownerId,omitempty"`
	WithdrawalPriority                  int    `json:"withdrawalPriority,omitempty"`
	WithdrawalAvailableAfterYear        *int   `json:"withdrawalAvailableAfterYear,omitempty"`
	SortOrder                           int    `json:"sortOrder"`
}

// Validate checks that a create request is public-safe and internally consistent.
func (request CreateFinancialItemRequest) Validate() error {
	return validateFinancialItemFields(
		request.Name,
		request.AmountCents,
		request.Currency,
		request.AnnualReturnRateBasisPoints,
		request.DrawdownAnnualReturnRateBasisPoints,
		request.AnnualContributionCents,
		request.OwnerID,
		request.WithdrawalPriority,
		request.WithdrawalAvailableAfterYear,
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
		request.DrawdownAnnualReturnRateBasisPoints,
		request.AnnualContributionCents,
		request.OwnerID,
		request.WithdrawalPriority,
		request.WithdrawalAvailableAfterYear,
		request.SortOrder,
	)
}

func validateFinancialItemFields(name string, amountCents int64, currency string, annualReturnRateBasisPoints int, drawdownAnnualReturnRateBasisPoints *int, annualContributionCents int64, ownerID string, withdrawalPriority int, withdrawalAvailableAfterYear *int, sortOrder int) error {
	var problems []string

	if strings.TrimSpace(name) == "" {
		problems = append(problems, "name is required")
	}
	if amountCents < 0 {
		problems = append(problems, "amountCents must be greater than or equal to 0")
	}
	if strings.TrimSpace(currency) == "" {
		problems = append(problems, "currency is required")
	} else if !IsValidCurrencyCode(currency) {
		problems = append(problems, "currency must be a 3-letter uppercase code such as USD")
	}
	if annualReturnRateBasisPoints < minimumAnnualReturnRateBasisPoints || annualReturnRateBasisPoints > maximumAnnualReturnRateBasisPoints {
		problems = append(problems, fmt.Sprintf("annualReturnRateBasisPoints must be between %d and %d", minimumAnnualReturnRateBasisPoints, maximumAnnualReturnRateBasisPoints))
	}
	if drawdownAnnualReturnRateBasisPoints != nil && (*drawdownAnnualReturnRateBasisPoints < minimumAnnualReturnRateBasisPoints || *drawdownAnnualReturnRateBasisPoints > maximumAnnualReturnRateBasisPoints) {
		problems = append(problems, fmt.Sprintf("drawdownAnnualReturnRateBasisPoints must be between %d and %d", minimumAnnualReturnRateBasisPoints, maximumAnnualReturnRateBasisPoints))
	}
	if annualContributionCents < 0 {
		problems = append(problems, "annualContributionCents must be greater than or equal to 0")
	}
	if len(ownerID) > maximumOwnerIDLength || strings.TrimSpace(ownerID) != ownerID {
		problems = append(problems, fmt.Sprintf("ownerId must be at most %d characters with no leading or trailing spaces", maximumOwnerIDLength))
	}
	if withdrawalPriority < 0 || withdrawalPriority > maximumWithdrawalPriority {
		problems = append(problems, "withdrawalPriority must be 1 (first), 2 (normal) or 3 (last)")
	}
	if withdrawalAvailableAfterYear != nil && (*withdrawalAvailableAfterYear < 0 || *withdrawalAvailableAfterYear > maximumWithdrawalAvailableAfterYear) {
		problems = append(problems, fmt.Sprintf("withdrawalAvailableAfterYear must be between 0 and %d", maximumWithdrawalAvailableAfterYear))
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
