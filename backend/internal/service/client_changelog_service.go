package service

import (
	"context"
	"log/slog"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"golang.org/x/sync/singleflight"
)

// ClientChangelogService 从 GitHub Releases 读取桌面客户端的更新日志（fork 本地）。
//
// 发版流水线写进 release 的说明就是唯一数据源，取代原先在后台逐条手填的
// client_changelog_entries。后台只保留一个「仓库」设置，留空时用
// DefaultClientChangelogGitHubRepo。
//
// 缓存：进程内缓存，新鲜期内直接返回；过期后先返回旧数据、后台刷新
// （stale-while-revalidate），只有冷启动才同步等 GitHub。拉取失败时保留旧数据，
// 并在 clientChangelogFailureBackoff 内不再重试，免得匿名流量把未认证的
// 60 次/小时额度打光。
type ClientChangelogService struct {
	releases    GitHubReleaseClient
	settingRepo SettingRepository
	now         func() time.Time

	flight singleflight.Group

	mu        sync.Mutex
	repo      string
	entries   []ClientChangelogEntry
	fetchedAt time.Time
	failedAt  time.Time
}

// ClientChangelogEntry 是一次客户端发版在官网更新日志里的样子。
type ClientChangelogEntry struct {
	Version string `json:"version"`
	// PublishedAt 是 RFC 3339 时间，由前端按访客时区格式化。
	PublishedAt string `json:"published_at"`
	// Title 只在 release 名称不只是版本号时才有值。
	Title string `json:"title"`
	// Items 是 release 说明拆出的条目，每条都是 Markdown。
	Items []string `json:"items"`
}

const (
	// DefaultClientChangelogGitHubRepo 是桌面客户端发版的仓库。
	DefaultClientChangelogGitHubRepo = "ai-poet/agent-client"

	clientChangelogFreshTTL        = 15 * time.Minute
	clientChangelogFailureBackoff  = 2 * time.Minute
	clientChangelogFetchTimeout    = 15 * time.Second
	clientChangelogReleasePageSize = 30
)

var gitHubRepoPattern = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,38})/[A-Za-z0-9._-]{1,100}$`)

// NewClientChangelogService 创建客户端更新日志服务。
func NewClientChangelogService(releases GitHubReleaseClient, settingRepo SettingRepository) *ClientChangelogService {
	return &ClientChangelogService{
		releases:    releases,
		settingRepo: settingRepo,
		now:         time.Now,
	}
}

// NormalizeGitHubRepo 把后台填写的仓库归一成 owner/repo。
// 接受 owner/repo、https://github.com/owner/repo(.git) 和留空（用默认仓库）。
func NormalizeGitHubRepo(raw string) (string, error) {
	repo := strings.TrimSpace(raw)
	if repo == "" {
		return "", nil
	}
	for _, prefix := range []string{"https://github.com/", "http://github.com/", "github.com/"} {
		if len(repo) >= len(prefix) && strings.EqualFold(repo[:len(prefix)], prefix) {
			repo = repo[len(prefix):]
			break
		}
	}
	repo = strings.TrimSuffix(strings.TrimSuffix(repo, "/"), ".git")
	if !gitHubRepoPattern.MatchString(repo) {
		return "", infraerrors.BadRequest("INVALID_CLIENT_CHANGELOG_REPO",
			"client changelog repository must look like owner/repo")
	}
	return repo, nil
}

// Entries 返回最近的客户端发版记录，新的在前。
// GitHub 不可用且没有缓存时返回空列表，不返回错误：官网只需要显示空状态。
// 没有配置任何客户端下载链接时更新日志不对外展示，直接返回空列表，也不去请求 GitHub。
func (s *ClientChangelogService) Entries(ctx context.Context) []ClientChangelogEntry {
	repo, enabled := s.resolveSource(ctx)
	if !enabled {
		return []ClientChangelogEntry{}
	}

	s.mu.Lock()
	sameRepo := s.repo == repo
	cached := sameRepo && !s.fetchedAt.IsZero()
	fresh := cached && s.now().Sub(s.fetchedAt) < clientChangelogFreshTTL
	backingOff := sameRepo && !s.failedAt.IsZero() && s.now().Sub(s.failedAt) < clientChangelogFailureBackoff
	var entries []ClientChangelogEntry
	if cached {
		entries = s.entries
	}
	s.mu.Unlock()

	switch {
	case fresh || backingOff:
		return nonNilEntries(entries)
	case cached:
		go func() { <-s.refresh(repo) }()
		return entries
	}

	select {
	case <-s.refresh(repo):
	case <-ctx.Done():
	}
	return nonNilEntries(s.snapshot(repo))
}

// resolveSource 读取后台设置：返回要同步的仓库，以及是否配置了客户端下载链接。
// 设置读不出来时按未配置处理，宁可暂时不展示也不在没有客户端的站点上露出更新日志。
func (s *ClientChangelogService) resolveSource(ctx context.Context) (string, bool) {
	if s.settingRepo == nil {
		return DefaultClientChangelogGitHubRepo, true
	}
	values, err := s.settingRepo.GetMultiple(ctx, []string{
		SettingKeyClientChangelogGitHubRepo,
		SettingKeyClientDownloadWindowsURL,
		SettingKeyClientDownloadMacOSURL,
	})
	if err != nil {
		slog.Warn("client_changelog_settings_read_failed", "error", err)
		return "", false
	}
	if strings.TrimSpace(values[SettingKeyClientDownloadWindowsURL]) == "" &&
		strings.TrimSpace(values[SettingKeyClientDownloadMacOSURL]) == "" {
		return "", false
	}
	repo, normErr := NormalizeGitHubRepo(values[SettingKeyClientChangelogGitHubRepo])
	if normErr != nil || repo == "" {
		return DefaultClientChangelogGitHubRepo, true
	}
	return repo, true
}

// refresh 拉取一次 release 列表并写入缓存；同一仓库的并发刷新合并成一次请求。
// 拉取不跟随调用方的请求上下文，访客断开不会让其他等待者一起失败。
func (s *ClientChangelogService) refresh(repo string) <-chan singleflight.Result {
	return s.flight.DoChan(repo, func() (any, error) {
		ctx, cancel := context.WithTimeout(context.Background(), clientChangelogFetchTimeout)
		defer cancel()
		releases, err := s.releases.FetchRecentReleases(ctx, repo, clientChangelogReleasePageSize)

		s.mu.Lock()
		defer s.mu.Unlock()
		if s.repo != repo {
			s.repo = repo
			s.entries = nil
			s.fetchedAt = time.Time{}
			s.failedAt = time.Time{}
		}
		if err != nil {
			s.failedAt = s.now()
			slog.Warn("client_changelog_fetch_failed", "repo", repo, "error", err)
			return nil, err
		}
		s.entries = ReleasesToChangelogEntries(releases)
		s.fetchedAt = s.now()
		s.failedAt = time.Time{}
		return nil, nil
	})
}

func (s *ClientChangelogService) snapshot(repo string) []ClientChangelogEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.repo != repo {
		return nil
	}
	return s.entries
}

func nonNilEntries(entries []ClientChangelogEntry) []ClientChangelogEntry {
	if entries == nil {
		return []ClientChangelogEntry{}
	}
	return entries
}

// ReleasesToChangelogEntries 把 GitHub release 转成更新日志条目：跳过草稿、预发布
// 和没有说明的 release，按发布时间倒序。
func ReleasesToChangelogEntries(releases []*GitHubRelease) []ClientChangelogEntry {
	entries := make([]ClientChangelogEntry, 0, len(releases))
	for _, release := range releases {
		if release == nil || release.Draft || release.Prerelease {
			continue
		}
		tag := strings.TrimSpace(release.TagName)
		version := strings.TrimPrefix(strings.TrimPrefix(tag, "v"), "V")
		if version == "" {
			continue
		}
		title := strings.TrimSpace(release.Name)
		if title == tag || strings.TrimPrefix(strings.TrimPrefix(title, "v"), "V") == version {
			title = ""
		}
		items := SplitReleaseNotes(release.Body)
		if title == "" && len(items) == 0 {
			continue
		}
		entries = append(entries, ClientChangelogEntry{
			Version:     version,
			PublishedAt: normalizeReleaseTime(release.PublishedAt),
			Title:       title,
			Items:       items,
		})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].PublishedAt > entries[j].PublishedAt
	})
	return entries
}

func normalizeReleaseTime(raw string) string {
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// SplitReleaseNotes 把 release 说明拆成条目：每个顶层列表项一条（续行和嵌套列表
// 跟着它），其余段落各成一条。标题行和 GitHub 自动生成的 "Full Changelog" 行丢掉。
func SplitReleaseNotes(body string) []string {
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	items := make([]string, 0)
	var current []string
	inBullet := false
	flush := func() {
		if text := strings.TrimSpace(strings.Join(current, "\n")); text != "" {
			items = append(items, text)
		}
		current = nil
		inBullet = false
	}
	for _, line := range lines {
		line = strings.TrimRight(line, " \t")
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
			// 列表项内部的空行不打断它，下一个顶层列表项或段落才会。
			if !inBullet {
				flush()
			}
		case isMarkdownHeading(trimmed), strings.HasPrefix(trimmed, "**Full Changelog**"):
			flush()
		case isTopLevelBullet(line):
			flush()
			current = append(current, strings.TrimSpace(line[2:]))
			inBullet = true
		case inBullet && line != trimmed:
			current = append(current, line)
		default:
			if inBullet {
				flush()
			}
			current = append(current, trimmed)
		}
	}
	flush()
	return items
}

func isMarkdownHeading(line string) bool {
	rest := strings.TrimLeft(line, "#")
	level := len(line) - len(rest)
	return level >= 1 && level <= 6 && (rest == "" || rest[0] == ' ')
}

func isTopLevelBullet(line string) bool {
	if len(line) < 2 || line[1] != ' ' {
		return false
	}
	switch line[0] {
	case '-', '*', '+':
		return true
	}
	return false
}
