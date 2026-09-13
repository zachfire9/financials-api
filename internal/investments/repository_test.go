package investments

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestInMemoryRepositoryCreatesInvestmentWithGeneratedFields(t *testing.T) {
	repository := NewInMemoryRepository()
	request := FakeCreateInvestmentRequest()

	investment, err := repository.Create(context.Background(), request)
	if err != nil {
		t.Fatalf("create investment: %v", err)
	}

	if investment.ID == "" {
		t.Fatal("expected generated ID")
	}
	if investment.Name != request.Name {
		t.Fatalf("expected name %q, got %q", request.Name, investment.Name)
	}
	if investment.CreatedAt.IsZero() || investment.UpdatedAt.IsZero() {
		t.Fatalf("expected timestamps, got created=%v updated=%v", investment.CreatedAt, investment.UpdatedAt)
	}
	if !investment.CreatedAt.Equal(investment.UpdatedAt) {
		t.Fatalf("expected created and updated timestamps to match on create")
	}
}

func TestInMemoryRepositoryListsInvestmentsInCreateOrder(t *testing.T) {
	repository := NewInMemoryRepository()
	firstRequest := FakeCreateInvestmentRequest()
	secondRequest := FakeCreateInvestmentRequest()
	secondRequest.Name = "Example emergency fund"
	secondRequest.Type = InvestmentTypeCash

	first, err := repository.Create(context.Background(), firstRequest)
	if err != nil {
		t.Fatalf("create first investment: %v", err)
	}
	second, err := repository.Create(context.Background(), secondRequest)
	if err != nil {
		t.Fatalf("create second investment: %v", err)
	}

	investments, err := repository.List(context.Background())
	if err != nil {
		t.Fatalf("list investments: %v", err)
	}

	if len(investments) != 2 {
		t.Fatalf("expected 2 investments, got %d", len(investments))
	}
	if investments[0].ID != first.ID || investments[1].ID != second.ID {
		t.Fatalf("expected create order [%q, %q], got [%q, %q]", first.ID, second.ID, investments[0].ID, investments[1].ID)
	}
}

func TestInMemoryRepositoryGetsAndUpdatesInvestment(t *testing.T) {
	repository := NewInMemoryRepository()
	created, err := repository.Create(context.Background(), FakeCreateInvestmentRequest())
	if err != nil {
		t.Fatalf("create investment: %v", err)
	}

	got, err := repository.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("get investment: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("expected ID %q, got %q", created.ID, got.ID)
	}

	updated, err := repository.Update(context.Background(), created.ID, UpdateInvestmentRequest{
		Name:                    "Example updated brokerage",
		Type:                    InvestmentTypeTaxableBrokerage,
		Institution:             "Example Institution",
		BalanceCents:            2500000,
		Currency:                "USD",
		AnnualContributionCents: 600000,
	})
	if err != nil {
		t.Fatalf("update investment: %v", err)
	}

	if updated.Name != "Example updated brokerage" {
		t.Fatalf("expected updated name, got %q", updated.Name)
	}
	if !updated.CreatedAt.Equal(created.CreatedAt) {
		t.Fatalf("expected created timestamp to stay %v, got %v", created.CreatedAt, updated.CreatedAt)
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) && !updated.UpdatedAt.Equal(created.UpdatedAt.Add(time.Nanosecond)) {
		t.Fatalf("expected updated timestamp to advance, got created=%v updated=%v", created.UpdatedAt, updated.UpdatedAt)
	}
}

func TestInMemoryRepositoryReturnsNotFoundForMissingInvestment(t *testing.T) {
	repository := NewInMemoryRepository()

	_, err := repository.Get(context.Background(), "missing")
	if !errors.Is(err, ErrInvestmentNotFound) {
		t.Fatalf("expected ErrInvestmentNotFound, got %v", err)
	}

	err = repository.Delete(context.Background(), "missing")
	if !errors.Is(err, ErrInvestmentNotFound) {
		t.Fatalf("expected ErrInvestmentNotFound on delete, got %v", err)
	}
}

func TestInMemoryRepositoryDeletesInvestment(t *testing.T) {
	repository := NewInMemoryRepository()
	created, err := repository.Create(context.Background(), FakeCreateInvestmentRequest())
	if err != nil {
		t.Fatalf("create investment: %v", err)
	}

	if err := repository.Delete(context.Background(), created.ID); err != nil {
		t.Fatalf("delete investment: %v", err)
	}

	_, err = repository.Get(context.Background(), created.ID)
	if !errors.Is(err, ErrInvestmentNotFound) {
		t.Fatalf("expected ErrInvestmentNotFound after delete, got %v", err)
	}
}
