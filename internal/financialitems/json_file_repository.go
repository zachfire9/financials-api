package financialitems

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// JSONFileRepository stores financial items in a local JSON file for low-friction local development.
type JSONFileRepository struct {
	path     string
	inMemory *InMemoryRepository
}

type jsonFileRepositoryState struct {
	Sequence int64                    `json:"sequence"`
	Items    map[string]FinancialItem `json:"items"`
}

// NewJSONFileRepository creates a repository backed by a local JSON file.
func NewJSONFileRepository(path string) (*JSONFileRepository, error) {
	if path == "" {
		return nil, fmt.Errorf("json file repository path is required")
	}

	repository := &JSONFileRepository{
		path:     path,
		inMemory: NewInMemoryRepository(),
	}
	if err := repository.load(); err != nil {
		return nil, err
	}
	return repository, nil
}

func (repository *JSONFileRepository) Create(ctx context.Context, request CreateFinancialItemRequest) (FinancialItem, error) {
	item, err := repository.inMemory.Create(ctx, request)
	if err != nil {
		return FinancialItem{}, err
	}
	if err := repository.save(); err != nil {
		return FinancialItem{}, err
	}
	return item, nil
}

func (repository *JSONFileRepository) List(ctx context.Context) ([]FinancialItem, error) {
	return repository.inMemory.List(ctx)
}

func (repository *JSONFileRepository) Get(ctx context.Context, id string) (FinancialItem, error) {
	return repository.inMemory.Get(ctx, id)
}

func (repository *JSONFileRepository) Update(ctx context.Context, id string, request UpdateFinancialItemRequest) (FinancialItem, error) {
	item, err := repository.inMemory.Update(ctx, id, request)
	if err != nil {
		return FinancialItem{}, err
	}
	if err := repository.save(); err != nil {
		return FinancialItem{}, err
	}
	return item, nil
}

func (repository *JSONFileRepository) Delete(ctx context.Context, id string) error {
	if err := repository.inMemory.Delete(ctx, id); err != nil {
		return err
	}
	return repository.save()
}

func (repository *JSONFileRepository) ExportBackup(ctx context.Context) (Backup, error) {
	return repository.inMemory.ExportBackup(ctx)
}

func (repository *JSONFileRepository) ImportBackup(ctx context.Context, backup Backup) ([]FinancialItem, error) {
	items, err := repository.inMemory.ImportBackup(ctx, backup)
	if err != nil {
		return nil, err
	}
	if err := repository.save(); err != nil {
		return nil, err
	}
	return items, nil
}

func (repository *JSONFileRepository) load() error {
	contents, err := os.ReadFile(repository.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read financial items JSON file: %w", err)
	}
	if len(contents) == 0 {
		return nil
	}

	var state jsonFileRepositoryState
	if err := json.Unmarshal(contents, &state); err != nil {
		return fmt.Errorf("parse financial items JSON file: %w", err)
	}
	if state.Items == nil {
		state.Items = make(map[string]FinancialItem)
	}

	repository.inMemory.mu.Lock()
	defer repository.inMemory.mu.Unlock()
	repository.inMemory.sequence = state.Sequence
	repository.inMemory.items = state.Items
	return nil
}

func (repository *JSONFileRepository) save() error {
	repository.inMemory.mu.RLock()
	state := jsonFileRepositoryState{
		Sequence: repository.inMemory.sequence,
		Items:    make(map[string]FinancialItem, len(repository.inMemory.items)),
	}
	for id, item := range repository.inMemory.items {
		state.Items[id] = item
	}
	repository.inMemory.mu.RUnlock()

	contents, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode financial items JSON file: %w", err)
	}
	contents = append(contents, '\n')

	if err := os.MkdirAll(filepath.Dir(repository.path), 0o700); err != nil {
		return fmt.Errorf("create financial items JSON directory: %w", err)
	}
	if err := os.WriteFile(repository.path, contents, 0o600); err != nil {
		return fmt.Errorf("write financial items JSON file: %w", err)
	}
	return nil
}
