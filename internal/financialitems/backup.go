package financialitems

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const BackupSchemaVersion = 1

var generatedItemIDPattern = regexp.MustCompile(`^item_(\d{6})$`)

// Backup is the public JSON backup shape for exporting and replacing saved financial items.
type Backup struct {
	SchemaVersion int             `json:"schemaVersion"`
	ExportedAt    time.Time       `json:"exportedAt"`
	Items         []FinancialItem `json:"items"`
}

// ExportBackup returns a public-safe JSON backup for saved financial items.
func (repository *InMemoryRepository) ExportBackup(ctx context.Context) (Backup, error) {
	items, err := repository.List(ctx)
	if err != nil {
		return Backup{}, err
	}
	return Backup{
		SchemaVersion: BackupSchemaVersion,
		ExportedAt:    time.Now().UTC(),
		Items:         items,
	}, nil
}

// ImportBackup replaces all saved financial items with a validated backup payload.
func (repository *InMemoryRepository) ImportBackup(ctx context.Context, backup Backup) ([]FinancialItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := ValidateBackup(backup); err != nil {
		return nil, err
	}

	items := make(map[string]FinancialItem, len(backup.Items))
	sequence := int64(0)
	for _, item := range backup.Items {
		copy := item
		copy.DrawdownAnnualReturnRateBasisPoints = copyOptionalInt(item.DrawdownAnnualReturnRateBasisPoints)
		items[copy.ID] = copy
		if matches := generatedItemIDPattern.FindStringSubmatch(copy.ID); matches != nil {
			parsed, err := strconv.ParseInt(matches[1], 10, 64)
			if err == nil && parsed > sequence {
				sequence = parsed
			}
		}
	}

	repository.mu.Lock()
	repository.items = items
	repository.sequence = sequence
	repository.mu.Unlock()

	return repository.List(ctx)
}

// ValidateBackup checks that a backup can safely replace repository contents.
func ValidateBackup(backup Backup) error {
	var problems []string
	if backup.SchemaVersion != BackupSchemaVersion {
		problems = append(problems, fmt.Sprintf("schemaVersion must be %d", BackupSchemaVersion))
	}

	seenIDs := make(map[string]struct{}, len(backup.Items))
	for index, item := range backup.Items {
		prefix := fmt.Sprintf("items[%d]", index)
		if strings.TrimSpace(item.ID) == "" {
			problems = append(problems, prefix+".id is required")
		} else if _, ok := seenIDs[item.ID]; ok {
			problems = append(problems, prefix+".id must be unique")
		} else {
			seenIDs[item.ID] = struct{}{}
		}
		if item.CreatedAt.IsZero() {
			problems = append(problems, prefix+".createdAt is required")
		}
		if item.UpdatedAt.IsZero() {
			problems = append(problems, prefix+".updatedAt is required")
		}

		if err := validateFinancialItemFields(
			item.Name,
			item.AmountCents,
			item.Currency,
			item.AnnualReturnRateBasisPoints,
			item.DrawdownAnnualReturnRateBasisPoints,
			item.AnnualContributionCents,
			item.SortOrder,
		); err != nil {
			problems = append(problems, prefix+": "+err.Error())
		}
	}
	if len(problems) > 0 {
		return ValidationError{Problems: problems}
	}
	return nil
}

func sortFinancialItems(items []FinancialItem) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].SortOrder != items[j].SortOrder {
			return items[i].SortOrder < items[j].SortOrder
		}
		return items[i].ID < items[j].ID
	})
}
