//go:build unit

package service

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type changelogReleaseClientStub struct {
	GitHubReleaseClient

	mu       sync.Mutex
	releases []*GitHubRelease
	err      error
	repos    []string
	block    chan struct{}
}

func (s *changelogReleaseClientStub) FetchRecentReleases(ctx context.Context, repo string, perPage int) ([]*GitHubRelease, error) {
	if s.block != nil {
		<-s.block
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.repos = append(s.repos, repo)
	return s.releases, s.err
}

func (s *changelogReleaseClientStub) calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.repos)
}

func (s *changelogReleaseClientStub) set(releases []*GitHubRelease, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.releases = releases
	s.err = err
}

type changelogSettingRepoStub struct {
	SettingRepository
	value      string
	windowsURL string
	macosURL   string
	err        error
}

func (s *changelogSettingRepoStub) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	if s.err != nil {
		return nil, s.err
	}
	return map[string]string{
		SettingKeyClientChangelogGitHubRepo: s.value,
		SettingKeyClientDownloadWindowsURL:  s.windowsURL,
		SettingKeyClientDownloadMacOSURL:    s.macosURL,
	}, nil
}

func newChangelogServiceForTest(client *changelogReleaseClientStub, repo string, now *time.Time) *ClientChangelogService {
	svc := NewClientChangelogService(client, &changelogSettingRepoStub{value: repo, windowsURL: "https://example.com/setup.exe"})
	svc.now = func() time.Time { return *now }
	return svc
}

func release(tag, published, body string) *GitHubRelease {
	return &GitHubRelease{TagName: tag, Name: tag, PublishedAt: published, Body: body}
}

func waitForCalls(t *testing.T, client *changelogReleaseClientStub, want int) {
	t.Helper()
	require.Eventually(t, func() bool { return client.calls() >= want }, time.Second, 5*time.Millisecond)
}

func TestSplitReleaseNotes(t *testing.T) {
	body := "## What's Changed\r\n" +
		"- First change\r\n" +
		"  continues here\r\n" +
		"  - nested detail\r\n" +
		"\r\n" +
		"- Second change with `code`\r\n" +
		"* Third change\r\n" +
		"\r\n" +
		"A closing paragraph\r\n" +
		"spanning two lines\r\n" +
		"\r\n" +
		"**Full Changelog**: https://github.com/o/r/compare/v1...v2\r\n"

	require.Equal(t, []string{
		"First change\n  continues here\n  - nested detail",
		"Second change with `code`",
		"Third change",
		"A closing paragraph\nspanning two lines",
	}, SplitReleaseNotes(body))
}

func TestSplitReleaseNotes_Empty(t *testing.T) {
	require.Empty(t, SplitReleaseNotes(""))
	require.Empty(t, SplitReleaseNotes("\n\n## Heading only\n"))
}

func TestReleasesToChangelogEntries(t *testing.T) {
	entries := ReleasesToChangelogEntries([]*GitHubRelease{
		release("v0.2.0", "2026-09-23T11:46:53Z", "- older"),
		{TagName: "v0.2.2", Name: "v0.2.2", PublishedAt: "2026-09-25T00:00:00Z", Body: "- draft", Draft: true},
		{TagName: "v0.2.2-beta", Name: "v0.2.2-beta", PublishedAt: "2026-09-25T00:00:00Z", Body: "- beta", Prerelease: true},
		{TagName: "v0.2.1", Name: "Images and caching", PublishedAt: "2026-09-24T11:16:42Z", Body: "- newer\n- also"},
		release("v0.1.0", "2026-09-01T00:00:00Z", ""),
		nil,
	})

	require.Equal(t, []ClientChangelogEntry{
		{Version: "0.2.1", PublishedAt: "2026-09-24T11:16:42Z", Title: "Images and caching", Items: []string{"newer", "also"}},
		{Version: "0.2.0", PublishedAt: "2026-09-23T11:46:53Z", Items: []string{"older"}},
	}, entries)
}

func TestNormalizeGitHubRepo(t *testing.T) {
	for input, want := range map[string]string{
		"":                           "",
		"  ai-poet/agent-client ":    "ai-poet/agent-client",
		"https://github.com/o/r":     "o/r",
		"https://github.com/o/r.git": "o/r",
		"github.com/o/r/":            "o/r",
	} {
		got, err := NormalizeGitHubRepo(input)
		require.NoError(t, err, input)
		require.Equal(t, want, got, input)
	}
	for _, input := range []string{"owner", "o/r/extra", "https://gitlab.com/o/r", "o/r?x=1", "../r"} {
		_, err := NormalizeGitHubRepo(input)
		require.Error(t, err, input)
	}
}

func TestClientChangelogService_CachesWhileFresh(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	client := &changelogReleaseClientStub{releases: []*GitHubRelease{release("v1.0.0", "2026-09-24T00:00:00Z", "- one")}}
	svc := newChangelogServiceForTest(client, "", &now)

	first := svc.Entries(context.Background())
	require.Len(t, first, 1)
	require.Equal(t, []string{DefaultClientChangelogGitHubRepo}, client.repos)

	now = now.Add(clientChangelogFreshTTL - time.Second)
	require.Equal(t, first, svc.Entries(context.Background()))
	require.Equal(t, 1, client.calls())
}

func TestClientChangelogService_ServesStaleWhileRefreshing(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	client := &changelogReleaseClientStub{releases: []*GitHubRelease{release("v1.0.0", "2026-09-24T00:00:00Z", "- one")}}
	svc := newChangelogServiceForTest(client, "o/r", &now)
	require.Len(t, svc.Entries(context.Background()), 1)

	client.set([]*GitHubRelease{
		release("v1.1.0", "2026-09-25T00:00:00Z", "- two"),
		release("v1.0.0", "2026-09-24T00:00:00Z", "- one"),
	}, nil)
	now = now.Add(clientChangelogFreshTTL + time.Second)

	require.Len(t, svc.Entries(context.Background()), 1, "stale data is returned without waiting")
	waitForCalls(t, client, 2)
	require.Eventually(t, func() bool { return len(svc.Entries(context.Background())) == 2 }, time.Second, 5*time.Millisecond)
	require.Equal(t, []string{"o/r", "o/r"}, client.repos)
}

func TestClientChangelogService_KeepsStaleDataAndBacksOffOnFailure(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	client := &changelogReleaseClientStub{releases: []*GitHubRelease{release("v1.0.0", "2026-09-24T00:00:00Z", "- one")}}
	svc := newChangelogServiceForTest(client, "", &now)
	require.Len(t, svc.Entries(context.Background()), 1)

	client.set(nil, errors.New("rate limited"))
	now = now.Add(clientChangelogFreshTTL + time.Second)
	require.Len(t, svc.Entries(context.Background()), 1)
	waitForCalls(t, client, 2)

	// 失败后的退避期内不再请求 GitHub，旧数据照常返回。
	require.Eventually(t, func() bool {
		svc.mu.Lock()
		defer svc.mu.Unlock()
		return !svc.failedAt.IsZero()
	}, time.Second, 5*time.Millisecond)
	now = now.Add(clientChangelogFailureBackoff - time.Second)
	require.Len(t, svc.Entries(context.Background()), 1)
	require.Equal(t, 2, client.calls())
}

func TestClientChangelogService_ColdFailureReturnsEmpty(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	client := &changelogReleaseClientStub{err: errors.New("offline")}
	svc := newChangelogServiceForTest(client, "", &now)

	entries := svc.Entries(context.Background())
	require.NotNil(t, entries)
	require.Empty(t, entries)

	require.Empty(t, svc.Entries(context.Background()))
	require.Equal(t, 1, client.calls(), "backs off after a cold failure")
}

func TestClientChangelogService_RefetchesWhenRepoChanges(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	client := &changelogReleaseClientStub{releases: []*GitHubRelease{release("v1.0.0", "2026-09-24T00:00:00Z", "- one")}}
	settings := &changelogSettingRepoStub{value: "o/first", macosURL: "curl -fsSL https://example.com/install.sh | sh"}
	svc := NewClientChangelogService(client, settings)
	svc.now = func() time.Time { return now }

	require.Len(t, svc.Entries(context.Background()), 1)
	settings.value = "o/second"
	require.Len(t, svc.Entries(context.Background()), 1)
	require.Equal(t, []string{"o/first", "o/second"}, client.repos)
}

func TestClientChangelogService_ColdRequestStopsWaitingWhenCallerLeaves(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	client := &changelogReleaseClientStub{block: make(chan struct{})}
	svc := newChangelogServiceForTest(client, "", &now)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.Empty(t, svc.Entries(ctx))
	close(client.block)
}

func TestClientChangelogService_HiddenWithoutClientDownloads(t *testing.T) {
	client := &changelogReleaseClientStub{releases: []*GitHubRelease{release("v1.0.0", "2026-09-24T00:00:00Z", "- one")}}
	svc := NewClientChangelogService(client, &changelogSettingRepoStub{value: "o/r", windowsURL: "  "})

	entries := svc.Entries(context.Background())
	require.NotNil(t, entries)
	require.Empty(t, entries)
	require.Zero(t, client.calls(), "no GitHub request when the site offers no client")
}

func TestClientChangelogService_HiddenWhenSettingsUnreadable(t *testing.T) {
	client := &changelogReleaseClientStub{releases: []*GitHubRelease{release("v1.0.0", "2026-09-24T00:00:00Z", "- one")}}
	svc := NewClientChangelogService(client, &changelogSettingRepoStub{err: errors.New("db down")})

	require.Empty(t, svc.Entries(context.Background()))
	require.Zero(t, client.calls())
}
