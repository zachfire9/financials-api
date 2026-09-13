package investments

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

// ErrInvestmentNotFound is returned when an investment ID does not exist.
var ErrInvestmentNotFound = errors.New("investment not found")

// Repository defines storage behavior for current investment records.
type Repository interface {
	Create(ctx context.Context, request CreateInvestmentRequest) (Investment, error)
	List(ctx context.Context) ([]Investment, error)
	Get(ctx context.Context, id string) (Investment, error)
	Update(ctx context.Context, id string, request UpdateInvestmentRequest) (Investment, error)
	Delete(ctx context.Context, id string) error
}

// InMemoryRepository is a simple deterministic local adapter for tests and early local development.
type InMemoryRepository struct {
	mu          sync.RWMutex
	sequence    int64
	investments map[string]Investment
}

// NewInMemoryRepository creates an empty repository backed by process memory.
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		investments: make(map[string]Investment),
	}
}

// Create validates and stores a new investment.
func (repository *InMemoryRepository) Create(ctx context.Context, request CreateInvestmentRequest) (Investment, error) {
	if err := ctx.Err(); err != nil {
		return Investment{}, err
	}
	if err := request.Validate(); err != nil {
		return Investment{}, err
	}

	repository.mu.Lock()
	defer repository.mu.Unlock()

	repository.sequence++
	now := time.Now().UTC()
	investment := Investment{
		ID:                      fmt.Sprintf("inv_%06d", repository.sequence),
		Name:                    request.Name,
		Type:                    request.Type,
		Institution:             request.Institution,
		BalanceCents:            request.BalanceCents,
		Currency:                request.Currency,
		AnnualContributionCents: request.AnnualContributionCents,
		CreatedAt:               now,
		UpdatedAt:               now,
	}

	repository.investments[investment.ID] = investment
	return investment, nil
}

// List returns investments in creation order.
func (repository *InMemoryRepository) List(ctx context.Context) ([]Investment, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repository.mu.RLock()
	defer repository.mu.RUnlock()

	investments := make([]Investment, 0, len(repository.investments))
	for _, investment := range repository.investments {
		investments = append(investments, investment)
	}
	sort.Slice(investments, func(i, j int) bool {
		return investments[i].ID < investments[j].ID
	})

	return investments, nil
}

// Get returns one investment by ID.
func (repository *InMemoryRepository) Get(ctx context.Context, id string) (Investment, error) {
	if err := ctx.Err(); err != nil {
		return Investment{}, err
	}

	repository.mu.RLock()
	defer repository.mu.RUnlock()

	investment, ok := repository.investments[id]
	if !ok {
		return Investment{}, ErrInvestmentNotFound
	}
	return investment, nil
}

// Update replaces the editable fields for an existing investment.
func (repository *InMemoryRepository) Update(ctx context.Context, id string, request UpdateInvestmentRequest) (Investment, error) {
	if err := ctx.Err(); err != nil {
		return Investment{}, err
	}
	if err := request.Validate(); err != nil {
		return Investment{}, err
	}

	repository.mu.Lock()
	defer repository.mu.Unlock()

	existing, ok := repository.investments[id]
	if !ok {
		return Investment{}, ErrInvestmentNotFound
	}

	now := time.Now().UTC()
	if !now.After(existing.UpdatedAt) {
		now = existing.UpdatedAt.Add(time.Nanosecond)
	}

	updated := Investment{
		ID:                      existing.ID,
		Name:                    request.Name,
		Type:                    request.Type,
		Institution:             request.Institution,
		BalanceCents:            request.BalanceCents,
		Currency:                request.Currency,
		AnnualContributionCents: request.AnnualContributionCents,
		CreatedAt:               existing.CreatedAt,
		UpdatedAt:               now,
	}
	repository.investments[id] = updated
	return updated, nil
}

// Delete removes one investment by ID.
func (repository *InMemoryRepository) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	repository.mu.Lock()
	defer repository.mu.Unlock()

	if _, ok := repository.investments[id]; !ok {
		return ErrInvestmentNotFound
	}
	delete(repository.investments, id)
	return nil
}
