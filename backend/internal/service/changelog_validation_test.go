//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// ==================== Changelog Validation Tests (UT-03) ====================

func TestValidateChangelogEntries(t *testing.T) {
	tests := []struct {
		name    string
		entries []ClientChangelogEntry
		wantErr bool
		errCode string
	}{
		{
			name: "valid entry",
			entries: []ClientChangelogEntry{
				{Version: "1.0", PublishedAt: "2026-01-01", Title: "First", Items: []string{"a"}, Enabled: true},
			},
			wantErr: false,
		},
		{
			name: "empty version",
			entries: []ClientChangelogEntry{
				{Version: "", PublishedAt: "2026-01-01", Title: "First", Items: []string{"a"}, Enabled: true},
			},
			wantErr: true,
			errCode: "INVALID_CHANGELOG_VERSION",
		},
		{
			name: "empty title with non-empty items passes",
			entries: []ClientChangelogEntry{
				{Version: "1.0", PublishedAt: "2026-01-01", Title: "", Items: []string{"a"}, Enabled: true},
			},
			wantErr: false,
		},
		{
			name: "empty title and empty items fails",
			entries: []ClientChangelogEntry{
				{Version: "1.0", PublishedAt: "2026-01-01", Title: "", Items: []string{}, Enabled: true},
			},
			wantErr: true,
			errCode: "INVALID_CHANGELOG_CONTENT",
		},
		{
			name: "items non-empty title can be empty",
			entries: []ClientChangelogEntry{
				{Version: "1.0", PublishedAt: "2026-01-01", Title: "", Items: []string{"a"}, Enabled: true},
			},
			wantErr: false,
		},
		{
			name: "invalid date format",
			entries: []ClientChangelogEntry{
				{Version: "1.0", PublishedAt: "invalid-date", Title: "First", Items: []string{}, Enabled: true},
			},
			wantErr: true,
			errCode: "INVALID_CHANGELOG_DATE",
		},
		{
			name: "invalid month",
			entries: []ClientChangelogEntry{
				{Version: "1.0", PublishedAt: "2026-13-01", Title: "First", Items: []string{}, Enabled: true},
			},
			wantErr: true,
			errCode: "INVALID_CHANGELOG_DATE",
		},
		{
			name: "invalid day",
			entries: []ClientChangelogEntry{
				{Version: "1.0", PublishedAt: "2026-01-32", Title: "First", Items: []string{}, Enabled: true},
			},
			wantErr: true,
			errCode: "INVALID_CHANGELOG_DATE",
		},
		{
			name:    "empty array is valid",
			entries: []ClientChangelogEntry{},
			wantErr: false,
		},
		{
			name:    "nil array is valid",
			entries: nil,
			wantErr: false,
		},
		{
			name: "null items with title passes",
			entries: []ClientChangelogEntry{
				{Version: "1.0", PublishedAt: "2026-01-01", Title: "First", Items: nil, Enabled: true},
			},
			wantErr: false,
		},
		{
			name: "null items with empty title fails",
			entries: []ClientChangelogEntry{
				{Version: "1.0", PublishedAt: "2026-01-01", Title: "", Items: nil, Enabled: true},
			},
			wantErr: true,
			errCode: "INVALID_CHANGELOG_CONTENT",
		},
		{
			name: "whitespace-only date treated as empty",
			entries: []ClientChangelogEntry{
				{Version: "1.0", PublishedAt: "   ", Title: "First", Items: []string{"a"}, Enabled: true},
			},
			wantErr: false,
		},
		{
			name: "valid leap year date",
			entries: []ClientChangelogEntry{
				{Version: "1.0", PublishedAt: "2024-02-29", Title: "Leap", Items: []string{"a"}, Enabled: true},
			},
			wantErr: false,
		},
		{
			name: "invalid leap year date",
			entries: []ClientChangelogEntry{
				{Version: "1.0", PublishedAt: "2023-02-29", Title: "Not Leap", Items: []string{"a"}, Enabled: true},
			},
			wantErr: true,
			errCode: "INVALID_CHANGELOG_DATE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateChangelogEntries(tt.entries)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errCode != "" {
					require.Contains(t, err.Error(), tt.errCode)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// helper: create n valid changelog entries
func makeValidChangelogEntries(n int) []ClientChangelogEntry {
	entries := make([]ClientChangelogEntry, n)
	for i := range entries {
		entries[i] = ClientChangelogEntry{
			Version:     "1.0." + string(rune('0'+i%10)),
			PublishedAt: "2026-01-01",
			Title:       "Entry",
			Items:       []string{"item"},
			Enabled:     true,
		}
	}
	return entries
}

func TestValidateChangelogEntries_BoundaryEntryCount(t *testing.T) {
	t.Run("at limit is valid", func(t *testing.T) {
		err := ValidateChangelogEntries(makeValidChangelogEntries(MaxChangelogEntries))
		require.NoError(t, err)
	})
	t.Run("over limit is invalid", func(t *testing.T) {
		err := ValidateChangelogEntries(makeValidChangelogEntries(MaxChangelogEntries + 1))
		require.Error(t, err)
		require.Contains(t, err.Error(), "INVALID_CHANGELOG_TOO_MANY_ENTRIES")
	})
}

func TestValidateChangelogEntries_BoundaryVersionLength(t *testing.T) {
	t.Run("at limit is valid", func(t *testing.T) {
		entries := []ClientChangelogEntry{
			{Version: string(make([]byte, MaxChangelogVersion)), PublishedAt: "2026-01-01", Title: "A", Items: []string{"a"}, Enabled: true},
		}
		err := ValidateChangelogEntries(entries)
		require.NoError(t, err)
	})
	t.Run("over limit is invalid", func(t *testing.T) {
		entries := []ClientChangelogEntry{
			{Version: string(make([]byte, MaxChangelogVersion+1)), PublishedAt: "2026-01-01", Title: "A", Items: []string{"a"}, Enabled: true},
		}
		err := ValidateChangelogEntries(entries)
		require.Error(t, err)
		require.Contains(t, err.Error(), "INVALID_CHANGELOG_VERSION_TOO_LONG")
	})
}

func TestValidateChangelogEntries_BoundaryTitleLength(t *testing.T) {
	t.Run("at limit is valid", func(t *testing.T) {
		entries := []ClientChangelogEntry{
			{Version: "1.0", PublishedAt: "2026-01-01", Title: string(make([]byte, MaxChangelogTitle)), Items: []string{"a"}, Enabled: true},
		}
		err := ValidateChangelogEntries(entries)
		require.NoError(t, err)
	})
	t.Run("over limit is invalid", func(t *testing.T) {
		entries := []ClientChangelogEntry{
			{Version: "1.0", PublishedAt: "2026-01-01", Title: string(make([]byte, MaxChangelogTitle+1)), Items: []string{"a"}, Enabled: true},
		}
		err := ValidateChangelogEntries(entries)
		require.Error(t, err)
		require.Contains(t, err.Error(), "INVALID_CHANGELOG_TITLE_TOO_LONG")
	})
}

func TestValidateChangelogEntries_BoundaryItemsCount(t *testing.T) {
	t.Run("at limit is valid", func(t *testing.T) {
		items := make([]string, MaxChangelogItems)
		for i := range items {
			items[i] = "item"
		}
		entries := []ClientChangelogEntry{
			{Version: "1.0", PublishedAt: "2026-01-01", Title: "A", Items: items, Enabled: true},
		}
		err := ValidateChangelogEntries(entries)
		require.NoError(t, err)
	})
	t.Run("over limit is invalid", func(t *testing.T) {
		items := make([]string, MaxChangelogItems+1)
		for i := range items {
			items[i] = "item"
		}
		entries := []ClientChangelogEntry{
			{Version: "1.0", PublishedAt: "2026-01-01", Title: "A", Items: items, Enabled: true},
		}
		err := ValidateChangelogEntries(entries)
		require.Error(t, err)
		require.Contains(t, err.Error(), "INVALID_CHANGELOG_TOO_MANY_ITEMS")
	})
}

func TestValidateChangelogEntries_BoundaryItemLength(t *testing.T) {
	t.Run("at limit is valid", func(t *testing.T) {
		entries := []ClientChangelogEntry{
			{Version: "1.0", PublishedAt: "2026-01-01", Title: "A", Items: []string{string(make([]byte, MaxChangelogItemLen))}, Enabled: true},
		}
		err := ValidateChangelogEntries(entries)
		require.NoError(t, err)
	})
	t.Run("over limit is invalid", func(t *testing.T) {
		entries := []ClientChangelogEntry{
			{Version: "1.0", PublishedAt: "2026-01-01", Title: "A", Items: []string{string(make([]byte, MaxChangelogItemLen+1))}, Enabled: true},
		}
		err := ValidateChangelogEntries(entries)
		require.Error(t, err)
		require.Contains(t, err.Error(), "INVALID_CHANGELOG_ITEM_TOO_LONG")
	})
}
