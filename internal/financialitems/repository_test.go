package financialitems

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestInMemoryRepositoryCreatesFinancialItemWithGeneratedFields(t *testing.T) {
	repository := NewInMemoryRepository()
	request := FakeCreateFinancialItemRequest()

	item, err := repository.Create(context.Background(), request)
	if err != nil {
		t.Fatalf("create financial item: %v", err)
	}

	if item.ID == "" {
		t.Fatal("expected generated ID")
	}
	if item.Name != request.Name {
		t.Fatalf("expected name %q, got %q", request.Name, item.Name)
	}
	if item.AmountCents != request.AmountCents {
		t.Fatalf("expected amount %d, got %d", request.AmountCents, item.AmountCents)
	}
	if item.AnnualReturnRateBasisPoints != request.AnnualReturnRateBasisPoints {
		t.Fatalf("expected return rate %d, got %d", request.AnnualReturnRateBasisPoints, item.AnnualReturnRateBasisPoints)
	}
	if item.AnnualContributionCents != request.AnnualContributionCents {
		t.Fatalf("expected annual contribution %d, got %d", request.AnnualContributionCents, item.AnnualContributionCents)
	}
	if item.SortOrder != request.SortOrder {
		t.Fatalf("expected sort order %d, got %d", request.SortOrder, item.SortOrder)
	}
	if item.CreatedAt.IsZero() || item.UpdatedAt.IsZero() {
		t.Fatalf("expected timestamps, got created=%v updated=%v", item.CreatedAt, item.UpdatedAt)
	}
	if !item.CreatedAt.Equal(item.UpdatedAt) {
		t.Fatalf("expected created and updated timestamps to match on create")
	}
}

func TestInMemoryRepositoryListsFinancialItemsBySortOrderThenCreateOrder(t *testing.T) {
	repository := NewInMemoryRepository()
	firstRequest := FakeCreateFinancialItemRequest()
	firstRequest.Name = "Example brokerage"
	firstRequest.SortOrder = 20
	secondRequest := FakeCreateFinancialItemRequest()
	secondRequest.Name = "Example emergency fund"
	secondRequest.SortOrder = 10

	first, err := repository.Create(context.Background(), firstRequest)
	if err != nil {
		t.Fatalf("create first financial item: %v", err)
	}
	second, err := repository.Create(context.Background(), secondRequest)
	if err != nil {
		t.Fatalf("create second financial item: %v", err)
	}

	items, err := repository.List(context.Background())
	if err != nil {
		t.Fatalf("list financial items: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 financial items, got %d", len(items))
	}
	if items[0].ID != second.ID || items[1].ID != first.ID {
		t.Fatalf("expected sort/create order [%q, %q], got [%q, %q]", second.ID, first.ID, items[0].ID, items[1].ID)
	}
}

func TestInMemoryRepositoryGetsAndUpdatesFinancialItem(t *testing.T) {
	repository := NewInMemoryRepository()
	created, err := repository.Create(context.Background(), FakeCreateFinancialItemRequest())
	if err != nil {
		t.Fatalf("create financial item: %v", err)
	}

	got, err := repository.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("get financial item: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("expected ID %q, got %q", created.ID, got.ID)
	}

	updated, err := repository.Update(context.Background(), created.ID, UpdateFinancialItemRequest{
		Name:                        "Example updated brokerage",
		AmountCents:                 2500000,
		Currency:                    "USD",
		AnnualReturnRateBasisPoints: 650,
		AnnualContributionCents:     600000,
		SortOrder:                   5,
	})
	if err != nil {
		t.Fatalf("update financial item: %v", err)
	}

	if updated.Name != "Example updated brokerage" {
		t.Fatalf("expected updated name, got %q", updated.Name)
	}
	if updated.AmountCents != 2500000 || updated.AnnualReturnRateBasisPoints != 650 || updated.SortOrder != 5 {
		t.Fatalf("expected updated financial fields, got %+v", updated)
	}
	if !updated.CreatedAt.Equal(created.CreatedAt) {
		t.Fatalf("expected created timestamp to stay %v, got %v", created.CreatedAt, updated.CreatedAt)
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) && !updated.UpdatedAt.Equal(created.UpdatedAt.Add(time.Nanosecond)) {
		t.Fatalf("expected updated timestamp to advance, got created=%v updated=%v", created.UpdatedAt, updated.UpdatedAt)
	}
}

func TestInMemoryRepositoryReturnsNotFoundForMissingFinancialItem(t *testing.T) {
	repository := NewInMemoryRepository()

	_, err := repository.Get(context.Background(), "missing")
	if !errors.Is(err, ErrFinancialItemNotFound) {
		t.Fatalf("expected ErrFinancialItemNotFound, got %v", err)
	}

	err = repository.Delete(context.Background(), "missing")
	if !errors.Is(err, ErrFinancialItemNotFound) {
		t.Fatalf("expected ErrFinancialItemNotFound on delete, got %v", err)
	}
}

func TestInMemoryRepositoryDeletesFinancialItem(t *testing.T) {
	repository := NewInMemoryRepository()
	created, err := repository.Create(context.Background(), FakeCreateFinancialItemRequest())
	if err != nil {
		t.Fatalf("create financial item: %v", err)
	}

	if err := repository.Delete(context.Background(), created.ID); err != nil {
		t.Fatalf("delete financial item: %v", err)
	}

	_, err = repository.Get(context.Background(), created.ID)
	if !errors.Is(err, ErrFinancialItemNotFound) {
		t.Fatalf("expected ErrFinancialItemNotFound after delete, got %v", err)
	}
}
