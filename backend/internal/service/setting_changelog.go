package service

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Changelog validation limits
const (
	MaxChangelogEntries = 50
	MaxChangelogItems   = 20
	MaxChangelogVersion = 50
	MaxChangelogTitle   = 200
	MaxChangelogItemLen = 5000
)

func parseClientChangelogEntries(raw string) []ClientChangelogEntry {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return []ClientChangelogEntry{}
	}
	var entries []ClientChangelogEntry
	if err := json.Unmarshal([]byte(raw), &entries); err != nil {
		return []ClientChangelogEntry{}
	}
	return entries
}

// filterAndSortPublicChangelogEntries parses raw JSON, filters to enabled entries,
// filters out blank items, and sorts by published_at descending (empty dates last).
func filterAndSortPublicChangelogEntries(raw string) []ClientChangelogEntry {
	entries := parseClientChangelogEntries(raw)
	if len(entries) == 0 {
		return []ClientChangelogEntry{}
	}

	// Filter: only enabled entries
	filtered := make([]ClientChangelogEntry, 0, len(entries))
	for _, entry := range entries {
		if !entry.Enabled {
			continue
		}
		// Filter out blank items
		cleanItems := make([]string, 0, len(entry.Items))
		for _, item := range entry.Items {
			if strings.TrimSpace(item) != "" {
				cleanItems = append(cleanItems, item)
			}
		}
		entry.Items = cleanItems
		filtered = append(filtered, entry)
	}

	// Sort by published_at descending, empty dates last (stable)
	sort.SliceStable(filtered, func(i, j int) bool {
		dateI := strings.TrimSpace(filtered[i].PublishedAt)
		dateJ := strings.TrimSpace(filtered[j].PublishedAt)
		if dateI == "" && dateJ == "" {
			return false // maintain stable order
		}
		if dateI == "" {
			return false // empty dates go last
		}
		if dateJ == "" {
			return true // non-empty dates go first
		}
		return dateI > dateJ // descending order
	})

	return filtered
}

// ValidateChangelogEntries validates a slice of changelog entries.
// Returns an error if any entry has invalid version, invalid date, or missing content.
// Also enforces limits: entries count, items count, version/title/item length.
func ValidateChangelogEntries(entries []ClientChangelogEntry) error {
	if len(entries) > MaxChangelogEntries {
		return infraerrors.BadRequest("INVALID_CHANGELOG_TOO_MANY_ENTRIES",
			fmt.Sprintf("too many changelog entries (max %d)", MaxChangelogEntries))
	}
	for i, entry := range entries {
		if strings.TrimSpace(entry.Version) == "" {
			return infraerrors.BadRequest("INVALID_CHANGELOG_VERSION", fmt.Sprintf("changelog entry %d: version is required", i+1))
		}
		if len(entry.Version) > MaxChangelogVersion {
			return infraerrors.BadRequest("INVALID_CHANGELOG_VERSION_TOO_LONG",
				fmt.Sprintf("changelog entry %d: version too long (max %d characters)", i+1, MaxChangelogVersion))
		}
		if strings.TrimSpace(entry.PublishedAt) != "" {
			if _, err := time.Parse("2006-01-02", strings.TrimSpace(entry.PublishedAt)); err != nil {
				return infraerrors.BadRequest("INVALID_CHANGELOG_DATE", fmt.Sprintf("changelog entry %d: published_at must be YYYY-MM-DD format", i+1))
			}
		}
		if len(entry.Title) > MaxChangelogTitle {
			return infraerrors.BadRequest("INVALID_CHANGELOG_TITLE_TOO_LONG",
				fmt.Sprintf("changelog entry %d: title too long (max %d characters)", i+1, MaxChangelogTitle))
		}
		if len(entry.Items) > MaxChangelogItems {
			return infraerrors.BadRequest("INVALID_CHANGELOG_TOO_MANY_ITEMS",
				fmt.Sprintf("changelog entry %d: too many items (max %d)", i+1, MaxChangelogItems))
		}
		// At least one of title or non-empty items must be present
		hasTitle := strings.TrimSpace(entry.Title) != ""
		hasItems := false
		for j, item := range entry.Items {
			if len(item) > MaxChangelogItemLen {
				return infraerrors.BadRequest("INVALID_CHANGELOG_ITEM_TOO_LONG",
					fmt.Sprintf("changelog entry %d, item %d: item too long (max %d characters)", i+1, j+1, MaxChangelogItemLen))
			}
			if strings.TrimSpace(item) != "" {
				hasItems = true
			}
		}
		if !hasTitle && !hasItems {
			return infraerrors.BadRequest("INVALID_CHANGELOG_CONTENT", fmt.Sprintf("changelog entry %d: at least one of title or non-empty items is required", i+1))
		}
	}
	return nil
}
