package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/repository"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/stretchr/testify/require"
)

// 邀请链接（/register?ref=<码>）来的 LinuxDo 注册：码由 start 捕获进 cookie，
// 回调直登 / 补邀请码 / 补邮箱建号三条路都要能落到 user_referrals。
// 本文件是桩与夹具，用例在 auth_linuxdo_oauth_referral_cases_test.go。

// oauthPendingFlowReferralRepoStub delegates to the real affiliate SQL once
// the test database exists, and records bindings for assertions.
type oauthPendingFlowReferralRepoStub struct {
	service.AffiliateRepository
	mu       sync.Mutex
	delegate service.AffiliateRepository
	client   *dbent.Client
	created  []service.AffiliateSummary
}

func (s *oauthPendingFlowReferralRepoStub) SetAffiliateClient(client *dbent.Client) {
	s.client = client
	s.delegate = repository.NewReferralRepository(client)
}

func (s *oauthPendingFlowReferralRepoStub) EnsureUserAffiliate(ctx context.Context, userID int64) (*service.AffiliateSummary, error) {
	return s.delegate.EnsureUserAffiliate(ctx, userID)
}

func (s *oauthPendingFlowReferralRepoStub) GetAffiliateByCode(ctx context.Context, code string) (*service.AffiliateSummary, error) {
	u, err := s.client.User.Query().Where(dbuser.ReferralCodeEQ(code)).Only(ctx)
	if err != nil { return nil, service.ErrAffiliateCodeInvalid }
	return s.delegate.EnsureUserAffiliate(ctx, u.ID)
}

func (s *oauthPendingFlowReferralRepoStub) BindInviter(ctx context.Context, userID, inviterID int64) (bool, error) {
	bound, err := s.delegate.BindInviter(ctx, userID, inviterID)
	if bound {
		s.mu.Lock()
		s.created = append(s.created, service.AffiliateSummary{UserID: userID, InviterID: &inviterID})
		s.mu.Unlock()
	}
	return bound, err
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
