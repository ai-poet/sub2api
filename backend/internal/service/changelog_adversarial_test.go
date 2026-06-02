//go:build unit

package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// ==================== Adversarial Tests ====================

// TestFilterAndSortPublicChangelogEntries_XSS attempts to inject XSS through changelog items.
// The filtering layer should preserve the content (sanitization is the frontend's job).
func TestFilterAndSortPublicChangelogEntries_XSS(t *testing.T) {
	xssPayload := `<script>alert('xss')</script>`
	entries := []ClientChangelogEntry{
		{Version: "1.0", Title: "XSS Test", Items: []string{xssPayload, "normal item"}, Enabled: true},
	}
	raw, _ := json.Marshal(entries)

	result := filterAndSortPublicChangelogEntries(string(raw))
	require.Len(t, result, 1)
	require.Len(t, result[0].Items, 2)
	require.Equal(t, xssPayload, result[0].Items[0])
}

// TestFilterAndSortPublicChangelogEntries_WhitespaceItems verifies various whitespace-only items are stripped.
func TestFilterAndSortPublicChangelogEntries_WhitespaceItems(t *testing.T) {
	entries := []ClientChangelogEntry{
		{
			Version: "1.0",
			Title:   "Whitespace",
			Items:   []string{"", " ", "\t", "\n", "\r\n", " ", "valid"},
			Enabled: true,
		},
	}
	raw, _ := json.Marshal(entries)

	result := filterAndSortPublicChangelogEntries(string(raw))
	require.Len(t, result, 1)
	require.Len(t, result[0].Items, 1)
	require.Equal(t, "valid", result[0].Items[0])
}

// TestFilterAndSortPublicChangelogEntries_LargeInput verifies behavior with many entries.
func TestFilterAndSortPublicChangelogEntries_LargeInput(t *testing.T) {
	entries := make([]ClientChangelogEntry, 100)
	for i := 0; i < 100; i++ {
		entries[i] = ClientChangelogEntry{
			Version:     "1.0." + string(rune('0'+i%10)),
			PublishedAt: "2026-01-" + string(rune('0'+(i%28)+1)),
			Title:       "Entry " + string(rune('0'+i%10)),
			Items:       []string{"item"},
			Enabled:     i%2 == 0,
		}
	}
	raw, _ := json.Marshal(entries)

	result := filterAndSortPublicChangelogEntries(string(raw))
	// Only even-indexed entries are enabled
	require.Len(t, result, 50)
}

// TestFilterAndSortPublicChangelogEntries_NullDateHandling handles null published_at from JSON.
func TestFilterAndSortPublicChangelogEntries_NullDateHandling(t *testing.T) {
	raw := `[{"version":"1.0","published_at":null,"title":"A","items":["x"],"enabled":true},{"version":"1.1","published_at":"2026-01-01","title":"B","items":["y"],"enabled":true}]`

	result := filterAndSortPublicChangelogEntries(raw)
	require.Len(t, result, 2)
	// Entry with date should come first (non-empty dates sort before empty)
	require.Equal(t, "1.1", result[0].Version)
	require.Equal(t, "1.0", result[1].Version)
}

// TestValidateChangelogEntries_XSSInFields verifies validation accepts XSS payloads
// (sanitization is frontend responsibility, backend should not reject valid-looking content).
func TestValidateChangelogEntries_XSSInFields(t *testing.T) {
	entries := []ClientChangelogEntry{
		{
			Version:     "1.0",
			PublishedAt: "2026-01-01",
			Title:       `<img src=x onerror=alert(1)>`,
			Items:       []string{`<script>alert('xss')</script>`},
			Enabled:     true,
		},
	}
	err := ValidateChangelogEntries(entries)
	require.NoError(t, err)
}

// TestValidateChangelogEntries_UnicodeContent verifies unicode content passes validation.
func TestValidateChangelogEntries_UnicodeContent(t *testing.T) {
	entries := []ClientChangelogEntry{
		{
			Version:     "1.0",
			PublishedAt: "2026-01-01",
			Title:       "你好世界 🌍 émojis",
			Items:       []string{"日本語", "🔥", "𝕦𝕟𝕚𝕔𝕠𝕕𝕖"},
			Enabled:     true,
		},
	}
	err := ValidateChangelogEntries(entries)
	require.NoError(t, err)
}

// TestValidateChangelogEntries_VeryLongVersion verifies very long version strings are accepted.
func TestValidateChangelogEntries_VeryLongVersion(t *testing.T) {
	entries := []ClientChangelogEntry{
		{
			Version:     strings.Repeat("v", 1000),
			PublishedAt: "2026-01-01",
			Title:       "Long",
			Items:       []string{},
			Enabled:     true,
		},
	}
	err := ValidateChangelogEntries(entries)
	require.NoError(t, err)
}

// TestValidateChangelogEntries_MultipleErrors verifies only the first error is returned.
func TestValidateChangelogEntries_MultipleErrors(t *testing.T) {
	entries := []ClientChangelogEntry{
		{Version: "", Title: "", Items: []string{}, Enabled: true},
		{Version: "", Title: "", Items: []string{}, Enabled: true},
	}
	err := ValidateChangelogEntries(entries)
	require.Error(t, err)
	require.Contains(t, err.Error(), "INVALID_CHANGELOG_VERSION")
}

// TestParseClientChangelogEntries_MalformedJSON verifies graceful handling of malformed JSON.
func TestParseClientChangelogEntries_MalformedJSON(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{`[`, 0},                    // incomplete JSON array
		{`[{`, 0},                   // incomplete object
		{`[{}]`, 1},                 // valid array with empty object
		{`not json at all`, 0},      // completely invalid
		{`null`, 0},                 // null
		{`{"version":"1.0"}`, 0},    // object instead of array
		{`[1,2,3]`, 0},              // array of wrong type
	}

	for _, tt := range tests {
		result := parseClientChangelogEntries(tt.input)
		require.Len(t, result, tt.want, "input: %s", tt.input)
	}
}

// TestFilterAndSortPublicChangelogEntries_AllDisabled returns empty when all entries are disabled.
func TestFilterAndSortPublicChangelogEntries_AllDisabled(t *testing.T) {
	entries := []ClientChangelogEntry{
		{Version: "1.0", Title: "A", Items: []string{"x"}, Enabled: false},
		{Version: "1.1", Title: "B", Items: []string{"y"}, Enabled: false},
	}
	raw, _ := json.Marshal(entries)

	result := filterAndSortPublicChangelogEntries(string(raw))
	require.Empty(t, result)
}

// TestFilterAndSortPublicChangelogEntries_AllBlankItems results in entry with empty items but still present.
func TestFilterAndSortPublicChangelogEntries_AllBlankItems(t *testing.T) {
	entries := []ClientChangelogEntry{
		{Version: "1.0", Title: "A", Items: []string{"", " "}, Enabled: true},
	}
	raw, _ := json.Marshal(entries)

	result := filterAndSortPublicChangelogEntries(string(raw))
	require.Len(t, result, 1)
	require.Empty(t, result[0].Items)
}

// TestFilterAndSortPublicChangelogEntries_SameDateStableSort verifies stable sort for identical dates.
func TestFilterAndSortPublicChangelogEntries_SameDateStableSort(t *testing.T) {
	entries := []ClientChangelogEntry{
		{Version: "1.0", PublishedAt: "2026-01-01", Title: "First", Items: []string{"a"}, Enabled: true},
		{Version: "1.1", PublishedAt: "2026-01-01", Title: "Second", Items: []string{"b"}, Enabled: true},
		{Version: "1.2", PublishedAt: "2026-01-01", Title: "Third", Items: []string{"c"}, Enabled: true},
	}
	raw, _ := json.Marshal(entries)

	result := filterAndSortPublicChangelogEntries(string(raw))
	require.Len(t, result, 3)
	// Stable sort should preserve original order for equal dates
	require.Equal(t, "1.0", result[0].Version)
	require.Equal(t, "1.1", result[1].Version)
	require.Equal(t, "1.2", result[2].Version)
}
