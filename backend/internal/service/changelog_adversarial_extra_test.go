//go:build unit

package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

// TestSettingService_UpdateSettings_ChangelogBoundaryRejection verifies service-layer
// rejection of changelog data exceeding limits.
func TestSettingService_UpdateSettings_ChangelogBoundaryRejection(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		errCode string
	}{
		{
			name:    "51 entries rejected",
			input:   string(mustMarshalJSON(makeValidChangelogEntries(MaxChangelogEntries + 1))),
			errCode: "INVALID_CHANGELOG_TOO_MANY_ENTRIES",
		},
		{
			name:    "version 51 chars rejected",
			input:   `[{"version":"` + strings.Repeat("v", MaxChangelogVersion+1) + `","published_at":"2026-01-01","title":"A","items":["item"],"enabled":true}]`,
			errCode: "INVALID_CHANGELOG_VERSION_TOO_LONG",
		},
		{
			name:    "title 201 chars rejected",
			input:   `[{"version":"1.0","published_at":"2026-01-01","title":"` + strings.Repeat("t", MaxChangelogTitle+1) + `","items":["item"],"enabled":true}]`,
			errCode: "INVALID_CHANGELOG_TITLE_TOO_LONG",
		},
		{
			name:    "21 items rejected",
			input:   `[{"version":"1.0","published_at":"2026-01-01","title":"A","items":["item"` + strings.Repeat(",\"item\"", MaxChangelogItems) + `],"enabled":true}]`,
			errCode: "INVALID_CHANGELOG_TOO_MANY_ITEMS",
		},
		{
			name:    "item 2001 chars rejected",
			input:   `[{"version":"1.0","published_at":"2026-01-01","title":"A","items":["` + strings.Repeat("i", MaxChangelogItemLen+1) + `"],"enabled":true}]`,
			errCode: "INVALID_CHANGELOG_ITEM_TOO_LONG",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &settingUpdateRepoStub{}
			svc := NewSettingService(repo, &config.Config{})
			err := svc.UpdateSettings(t.Context(), &SystemSettings{
				ClientChangelogEntries: tt.input,
			})
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.errCode)
			require.Nil(t, repo.updates, "repo must not be called when validation fails")
		})
	}
}

// TestFilterAndSortPublicChangelogEntries_ExtremeWhitespace verifies extreme
// whitespace-only inputs are all stripped from items.
func TestFilterAndSortPublicChangelogEntries_ExtremeWhitespace(t *testing.T) {
	entries := []ClientChangelogEntry{
		{
			Version: "1.0",
			Title:   "Whitespace",
			Items:   []string{"", " ", "\t", "\n", "\r\n", " ", " ", "　", "valid"},
			Enabled: true,
		},
	}
	raw, _ := json.Marshal(entries)

	result := filterAndSortPublicChangelogEntries(string(raw))
	require.Len(t, result, 1)
	require.Len(t, result[0].Items, 1)
	require.Equal(t, "valid", result[0].Items[0])
}

// TestValidateChangelogEntries_BlankTitleWithBlankItemsFails verifies that when
// both title (after trim) and all items (after trim) are blank, validation fails.
func TestValidateChangelogEntries_BlankTitleWithBlankItemsFails(t *testing.T) {
	entries := []ClientChangelogEntry{
		{Version: "1.0", PublishedAt: "2026-01-01", Title: "   ", Items: []string{"", "  "}, Enabled: true},
	}
	err := ValidateChangelogEntries(entries)
	require.Error(t, err)
	require.Contains(t, err.Error(), "INVALID_CHANGELOG_CONTENT")
}

// TestValidateChangelogEntries_EmptyItemsWithTitlePasses verifies that empty
// items with a non-empty title passes validation.
func TestValidateChangelogEntries_EmptyItemsWithTitlePasses(t *testing.T) {
	entries := []ClientChangelogEntry{
		{Version: "1.0", PublishedAt: "2026-01-01", Title: "Title Only", Items: []string{}, Enabled: true},
	}
	err := ValidateChangelogEntries(entries)
	require.NoError(t, err)
}

func mustMarshalJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
