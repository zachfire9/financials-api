package financialitems

import (
	"strings"
	"testing"
)

func TestValidateCreateFinancialItemRequestAcceptsDeterministicFixture(t *testing.T) {
	request := FakeCreateFinancialItemRequest()

	if err := request.Validate(); err != nil {
		t.Fatalf("expected fake request to be valid, got %v", err)
	}
}

func TestValidateCreateFinancialItemRequestRequiresName(t *testing.T) {
	request := FakeCreateFinancialItemRequest()
	request.Name = "   "

	err := request.Validate()
	if err == nil {
		t.Fatal("expected missing name to fail validation")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Fatalf("expected name validation error, got %v", err)
	}
}

func TestValidateCreateFinancialItemRequestRejectsNegativeAmount(t *testing.T) {
	request := FakeCreateFinancialItemRequest()
	request.AmountCents = -1

	err := request.Validate()
	if err == nil {
		t.Fatal("expected negative amount to fail validation")
	}
	if !strings.Contains(err.Error(), "amountCents") {
		t.Fatalf("expected amountCents validation error, got %v", err)
	}
}

func TestValidateCreateFinancialItemRequestRejectsReturnRatesBelowNegativeOneHundredPercent(t *testing.T) {
	request := FakeCreateFinancialItemRequest()
	request.AnnualReturnRateBasisPoints = -10001

	err := request.Validate()
	if err == nil {
		t.Fatal("expected overly negative return rate to fail validation")
	}
	if !strings.Contains(err.Error(), "annualReturnRateBasisPoints") {
		t.Fatalf("expected annualReturnRateBasisPoints validation error, got %v", err)
	}
}

func TestValidateCreateFinancialItemRequestRejectsAbsurdReturnRates(t *testing.T) {
	request := FakeCreateFinancialItemRequest()
	request.AnnualReturnRateBasisPoints = 100001

	err := request.Validate()
	if err == nil {
		t.Fatal("expected absurd return rate to fail validation")
	}
	if !strings.Contains(err.Error(), "annualReturnRateBasisPoints") {
		t.Fatalf("expected annualReturnRateBasisPoints validation error, got %v", err)
	}
}

func TestValidateCreateFinancialItemRequestRejectsNegativeAnnualContribution(t *testing.T) {
	request := FakeCreateFinancialItemRequest()
	request.AnnualContributionCents = -1

	err := request.Validate()
	if err == nil {
		t.Fatal("expected negative annual contribution to fail validation")
	}
	if !strings.Contains(err.Error(), "annualContributionCents") {
		t.Fatalf("expected annualContributionCents validation error, got %v", err)
	}
}

func TestValidateCreateFinancialItemRequestRejectsNegativeSortOrder(t *testing.T) {
	request := FakeCreateFinancialItemRequest()
	request.SortOrder = -1

	err := request.Validate()
	if err == nil {
		t.Fatal("expected negative sort order to fail validation")
	}
	if !strings.Contains(err.Error(), "sortOrder") {
		t.Fatalf("expected sortOrder validation error, got %v", err)
	}
}
