package financialitems

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

// ErrFinancialItemNotFound is returned when a financial item ID does not exist.
var ErrFinancialItemNotFound = errors.New("financial item not found")

// Repository defines storage behavior for configurable financial item records.
type Repository interface {
	Create(ctx context.Context, request CreateFinancialItemRequest) (FinancialItem, error)
	List(ctx context.Context) ([]FinancialItem, error)
	Get(ctx context.Context, id string) (FinancialItem, error)
	Update(ctx context.Context, id string, request UpdateFinancialItemRequest) (FinancialItem, error)
	Delete(ctx context.Context, id string) error
	ExportBackup(ctx context.Context) (Backup, error)
	ImportBackup(ctx context.Context, backup Backup) ([]FinancialItem, error)
}

// InMemoryRepository is a simple deterministic local adapter for tests and early local development.
type InMemoryRepository struct {
	mu       sync.RWMutex
	sequence int64
	items    map[string]FinancialItem
}

// NewInMemoryRepository creates an empty repository backed by process memory.
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		items: make(map[string]FinancialItem),
	}
}

// Create validates and stores a new financial item.
func (repository *InMemoryRepository) Create(ctx context.Context, request CreateFinancialItemRequest) (FinancialItem, error) {
	if err := ctx.Err(); err != nil {
		return FinancialItem{}, err
	}
	if err := request.Validate(); err != nil {
		return FinancialItem{}, err
	}

	repository.mu.Lock()
	defer repository.mu.Unlock()

	repository.sequence++
	now := time.Now().UTC()
	item := FinancialItem{
		ID:                                  fmt.Sprintf("item_%06d", repository.sequence),
		Name:                                request.Name,
		AmountCents:                         request.AmountCents,
		Currency:                            request.Currency,
		AnnualReturnRateBasisPoints:         request.AnnualReturnRateBasisPoints,
		DrawdownAnnualReturnRateBasisPoints: copyOptionalInt(request.DrawdownAnnualReturnRateBasisPoints),
		AnnualContributionCents:             request.AnnualContributionCents,
		SortOrder:                           request.SortOrder,
		CreatedAt:                           now,
		UpdatedAt:                           now,
	}

	repository.items[item.ID] = item
	return item, nil
}

// List returns financial items by sort order, then creation order.
func (repository *InMemoryRepository) List(ctx context.Context) ([]FinancialItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repository.mu.RLock()
	defer repository.mu.RUnlock()

	items := make([]FinancialItem, 0, len(repository.items))
	for _, item := range repository.items {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].SortOrder != items[j].SortOrder {
			return items[i].SortOrder < items[j].SortOrder
		}
		return items[i].ID < items[j].ID
	})

	return items, nil
}

// Get returns one financial item by ID.
func (repository *InMemoryRepository) Get(ctx context.Context, id string) (FinancialItem, error) {
	if err := ctx.Err(); err != nil {
		return FinancialItem{}, err
	}

	repository.mu.RLock()
	defer repository.mu.RUnlock()

	item, ok := repository.items[id]
	if !ok {
		return FinancialItem{}, ErrFinancialItemNotFound
	}
	return item, nil
}

// Update replaces the editable fields for an existing financial item.
func (repository *InMemoryRepository) Update(ctx context.Context, id string, request UpdateFinancialItemRequest) (FinancialItem, error) {
	if err := ctx.Err(); err != nil {
		return FinancialItem{}, err
	}
	if err := request.Validate(); err != nil {
		return FinancialItem{}, err
	}

	repository.mu.Lock()
	defer repository.mu.Unlock()

	existing, ok := repository.items[id]
	if !ok {
		return FinancialItem{}, ErrFinancialItemNotFound
	}

	now := time.Now().UTC()
	if !now.After(existing.UpdatedAt) {
		now = existing.UpdatedAt.Add(time.Nanosecond)
	}

	updated := FinancialItem{
		ID:                                  existing.ID,
		Name:                                request.Name,
		AmountCents:                         request.AmountCents,
		Currency:                            request.Currency,
		AnnualReturnRateBasisPoints:         request.AnnualReturnRateBasisPoints,
		DrawdownAnnualReturnRateBasisPoints: copyOptionalInt(request.DrawdownAnnualReturnRateBasisPoints),
		AnnualContributionCents:             request.AnnualContributionCents,
		SortOrder:                           request.SortOrder,
		CreatedAt:                           existing.CreatedAt,
		UpdatedAt:                           now,
	}
	repository.items[id] = updated
	return updated, nil
}

// Delete removes one financial item by ID.
func (repository *InMemoryRepository) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	repository.mu.Lock()
	defer repository.mu.Unlock()

	if _, ok := repository.items[id]; !ok {
		return ErrFinancialItemNotFound
	}
	delete(repository.items, id)
	return nil
}

func copyOptionalInt(value *int) *int {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
