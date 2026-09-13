package financialitems

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestJSONFileRepositoryPersistsFinancialItemsAcrossInstances(t *testing.T) {
	path := filepath.Join(t.TempDir(), "financial-items.json")
	firstRepository, err := NewJSONFileRepository(path)
	if err != nil {
		t.Fatalf("create first repository: %v", err)
	}

	created, err := firstRepository.Create(context.Background(), FakeCreateFinancialItemRequest())
	if err != nil {
		t.Fatalf("create financial item: %v", err)
	}

	secondRepository, err := NewJSONFileRepository(path)
	if err != nil {
		t.Fatalf("create second repository: %v", err)
	}

	items, err := secondRepository.List(context.Background())
	if err != nil {
		t.Fatalf("list persisted financial items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one persisted item, got %d", len(items))
	}
	if items[0].ID != created.ID || items[0].Name != created.Name {
		t.Fatalf("unexpected persisted item: %+v", items[0])
	}
}

func TestJSONFileRepositoryUpdatesAndDeletesPersistedFinancialItems(t *testing.T) {
	path := filepath.Join(t.TempDir(), "financial-items.json")
	repository, err := NewJSONFileRepository(path)
	if err != nil {
		t.Fatalf("create repository: %v", err)
	}

	created, err := repository.Create(context.Background(), FakeCreateFinancialItemRequest())
	if err != nil {
		t.Fatalf("create financial item: %v", err)
	}
	updated, err := repository.Update(context.Background(), created.ID, UpdateFinancialItemRequest{
		Name:                        "Example updated item",
		AmountCents:                 2200000,
		Currency:                    "USD",
		AnnualReturnRateBasisPoints: 600,
		AnnualContributionCents:     120000,
		SortOrder:                   4,
	})
	if err != nil {
		t.Fatalf("update financial item: %v", err)
	}

	reopenedRepository, err := NewJSONFileRepository(path)
	if err != nil {
		t.Fatalf("reopen repository: %v", err)
	}
	got, err := reopenedRepository.Get(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("get persisted update: %v", err)
	}
	if got.Name != updated.Name || got.AmountCents != updated.AmountCents {
		t.Fatalf("unexpected persisted update: %+v", got)
	}

	if err := reopenedRepository.Delete(context.Background(), created.ID); err != nil {
		t.Fatalf("delete persisted item: %v", err)
	}
	finalRepository, err := NewJSONFileRepository(path)
	if err != nil {
		t.Fatalf("reopen final repository: %v", err)
	}
	_, err = finalRepository.Get(context.Background(), created.ID)
	if !errors.Is(err, ErrFinancialItemNotFound) {
		t.Fatalf("expected deleted item to stay deleted, got %v", err)
	}
}
