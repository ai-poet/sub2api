package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/stretchr/testify/require"
)

// 邀请链接（/register?ref=<码>）来的 LinuxDo 注册：码由 start 捕获进 cookie，
// 回调直登 / 补邀请码 / 补邮箱建号三条路都要能落到 user_referrals。
// 本文件是桩与夹具，用例在 auth_linuxdo_oauth_referral_cases_test.go。

// oauthPendingFlowReferralRepoStub 只记录 RegisterReferral 写下的推荐关系，其余方法在这些流程里用不到。
type oauthPendingFlowReferralRepoStub struct {
	mu      sync.Mutex
	created []service.UserReferral
}

func (s *oauthPendingFlowReferralRepoStub) Create(_ context.Context, ref *service.UserReferral) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	ref.ID = int64(len(s.created) + 1)
	s.created = append(s.created, *ref)
	return nil
}

func (s *oauthPendingFlowReferralRepoStub) GetByRefereeID(_ context.Context, refereeID int64) (*service.UserReferral, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.created {
		if s.created[i].RefereeID == refereeID {
			ref := s.created[i]
			return &ref, nil
		}
	}
	return nil, service.ErrReferralNotFound
}

func (s *oauthPendingFlowReferralRepoStub) GetByID(context.Context, int64) (*service.UserReferral, error) {
	return nil, service.ErrReferralNotFound
}

func (s *oauthPendingFlowReferralRepoStub) UpdateStatus(context.Context, int64, string, *service.ReferralRewardSnapshot) error {
	return nil
}

func (s *oauthPendingFlowReferralRepoStub) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func (s *oauthPendingFlowReferralRepoStub) CountByReferrerID(_ context.Context, referrerID int64) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, ref := range s.created {
		if ref.ReferrerID == referrerID {
			n++
		}
	}
	return n, nil
}

func (s *oauthPendingFlowReferralRepoStub) CountByReferrerIDAndStatus(context.Context, int64, string) (int, error) {
	return 0, nil
}

func (s *oauthPendingFlowReferralRepoStub) SumReferrerBalanceReward(context.Context, int64) (float64, error) {
	return 0, nil
}

func (s *oauthPendingFlowReferralRepoStub) ListByReferrerID(context.Context, int64, pagination.PaginationParams) ([]service.UserReferral, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

func (s *oauthPendingFlowReferralRepoStub) ListAll(context.Context, pagination.PaginationParams) ([]service.UserReferral, *pagination.PaginationResult, error) {
	return nil, nil, nil
}

func linuxDoReferralTestConfig(upstreamURL string) config.LinuxDoConnectConfig {
	return config.LinuxDoConnectConfig{
		Enabled:             true,
		ClientID:            "linuxdo-client",
		ClientSecret:        "linuxdo-secret",
		AuthorizeURL:        upstreamURL + "/authorize",
		TokenURL:            upstreamURL + "/token",
		UserInfoURL:         upstreamURL + "/userinfo",
		Scopes:              "read",
		RedirectURL:         "https://api.example.com/api/v1/auth/oauth/linuxdo/callback",
		FrontendRedirectURL: "/auth/linuxdo/callback",
		TokenAuthMethod:     "client_secret_post",
		UsePKCE:             true,
	}
}

func newLinuxDoReferralUpstream(t *testing.T, subject, username string) *httptest.Server {
	t.Helper()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"access_token":"linuxdo-access","token_type":"Bearer","expires_in":3600}`))
		case "/userinfo":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"` + subject + `","username":"` + username + `","name":"Referred User"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(upstream.Close)
	return upstream
}

func seedReferrer(t *testing.T, client *dbent.Client, code string) *dbent.User {
	t.Helper()
	referrer, err := client.User.Create().
		SetEmail("referrer-" + code + "@example.com").
		SetUsername("referrer-" + code).
		SetPasswordHash("hash").
		SetRole(service.RoleUser).
		SetStatus(service.StatusActive).
		SetReferralCode(code).
		Save(context.Background())
	require.NoError(t, err)
	return referrer
}

func newLinuxDoReferralCallbackRequest(state, verifier, browser, affCode string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/linuxdo/callback?code=code-"+state+"&state="+state, nil)
	req.AddCookie(encodedCookie(linuxDoOAuthStateCookieName, state))
	req.AddCookie(encodedCookie(linuxDoOAuthRedirectCookie, "/dashboard"))
	req.AddCookie(encodedCookie(linuxDoOAuthVerifierCookie, verifier))
	req.AddCookie(encodedCookie(linuxDoOAuthIntentCookieName, oauthIntentLogin))
	req.AddCookie(encodedCookie(oauthPendingBrowserCookieName, browser))
	if affCode != "" {
		req.AddCookie(encodedCookie(oauthAffiliateCodeCookieName, affCode))
	}
	return req
}
