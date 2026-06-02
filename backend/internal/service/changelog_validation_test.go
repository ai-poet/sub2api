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
