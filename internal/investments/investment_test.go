package investments

import (
	"strings"
	"testing"
)

func TestValidateCreateInvestmentRequestAcceptsDeterministicFixture(t *testing.T) {
	request := FakeCreateInvestmentRequest()

	if err := request.Validate(); err != nil {
		t.Fatalf("expected fake request to be valid, got %v", err)
	}
}

func TestValidateCreateInvestmentRequestRequiresName(t *testing.T) {
	request := FakeCreateInvestmentRequest()
	request.Name = "   "

	err := request.Validate()
	if err == nil {
		t.Fatal("expected missing name to fail validation")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Fatalf("expected name validation error, got %v", err)
	}
}

func TestValidateCreateInvestmentRequestRejectsNegativeBalance(t *testing.T) {
	request := FakeCreateInvestmentRequest()
	request.BalanceCents = -1

	err := request.Validate()
	if err == nil {
		t.Fatal("expected negative balance to fail validation")
	}
	if !strings.Contains(err.Error(), "balanceCents") {
		t.Fatalf("expected balanceCents validation error, got %v", err)
	}
}

func TestValidateCreateInvestmentRequestRejectsUnsupportedType(t *testing.T) {
	request := FakeCreateInvestmentRequest()
	request.Type = InvestmentType("crypto-moonshot")

	err := request.Validate()
	if err == nil {
		t.Fatal("expected unsupported type to fail validation")
	}
	if !strings.Contains(err.Error(), "type") {
		t.Fatalf("expected type validation error, got %v", err)
	}
}
