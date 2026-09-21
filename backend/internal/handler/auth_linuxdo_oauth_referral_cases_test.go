package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/ent/pendingauthsession"
	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestLinuxDoOAuthStartCapturesAffiliateCodeCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newLinuxDoOAuthTestHandler(t, false, linuxDoReferralTestConfig("https://connect.linux.do"))

	start := func(target string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodGet, target, nil)
		handler.LinuxDoOAuthStart(c)
		require.Equal(t, http.StatusFound, recorder.Code, recorder.Body.String())
		return recorder
	}

	recorder := start("/api/v1/auth/oauth/linuxdo/start?redirect=/dashboard&aff_code=REF123")
	cookie := findCookie(recorder.Result().Cookies(), oauthAffiliateCodeCookieName)
	require.NotNil(t, cookie)
	require.Equal(t, "REF123", decodeCookieValueForTest(t, cookie.Value))
	require.Equal(t, oauthPendingBrowserCookiePath, cookie.Path)
	require.Equal(t, oauthPendingCookieMaxAgeSec, cookie.MaxAge)
	require.True(t, cookie.HttpOnly)

	// ?aff= 别名同样接受。
	recorder = start("/api/v1/auth/oauth/linuxdo/start?aff=REF456")
	cookie = findCookie(recorder.Result().Cookies(), oauthAffiliateCodeCookieName)
	require.NotNil(t, cookie)
	require.Equal(t, "REF456", decodeCookieValueForTest(t, cookie.Value))

	// 没带码就清掉，免得上一次尝试残留的码粘住。
	recorder = start("/api/v1/auth/oauth/linuxdo/start?redirect=/dashboard")
	requireCookieCleared(t, recorder, oauthAffiliateCodeCookieName)
}

// 开了邀请码：回调只能建 pending session，码要跟着 session 走到 complete-registration。
func TestLinuxDoOAuthCallbackCarriesAffiliateCodeIntoPendingSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := newLinuxDoReferralUpstream(t, "ref-invite-1", "linuxdo_ref_invite")
	handler, client := newLinuxDoOAuthHandlerAndClient(t, true, linuxDoReferralTestConfig(upstream.URL))
	t.Cleanup(func() { _ = client.Close() })

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = newLinuxDoReferralCallbackRequest("state-ref-invite", "verifier-ref-invite", "browser-ref-invite", "REF123")

	handler.LinuxDoOAuthCallback(c)

	require.Equal(t, http.StatusFound, recorder.Code)

	// 这条路的契约是"码进了 pending session"：cookie 本身 10 分钟自然过期，且回调里 deferred 的清理在 redirect 之后写不进响应头。
	sessionCookie := findCookie(recorder.Result().Cookies(), oauthPendingSessionCookieName)
	require.NotNil(t, sessionCookie)
	session, err := client.PendingAuthSession.Query().
		Where(pendingauthsession.SessionTokenEQ(decodeCookieValueForTest(t, sessionCookie.Value))).
		Only(context.Background())
	require.NoError(t, err)
	require.Equal(t, "REF123", session.LocalFlowState[oauthAffiliateCodeStateKey])
}

// 默认配置下新用户直登不经过前端：推荐关系必须在回调里靠 cookie 建立。
func TestLinuxDoOAuthCallbackDirectLoginBindsReferral(t *testing.T) {
	gin.SetMode(gin.TestMode)
	upstream := newLinuxDoReferralUpstream(t, "ref-direct-1", "linuxdo_ref_direct")
	referrals := &oauthPendingFlowReferralRepoStub{}
	handler, client := newOAuthPendingFlowTestHandlerWithDependencies(t, oauthPendingFlowTestHandlerOptions{
		referralRepo:  referrals,
		settingValues: map[string]string{service.SettingKeyReferralEnabled: "true"},
	})
	t.Cleanup(func() { _ = client.Close() })
	configureLinuxDoOAuthTestHandler(handler, linuxDoReferralTestConfig(upstream.URL))
	referrer := seedReferrer(t, client, "REF123")

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = newLinuxDoReferralCallbackRequest("state-ref-direct", "verifier-ref-direct", "browser-ref-direct", "REF123")

	handler.LinuxDoOAuthCallback(c)

	require.Equal(t, http.StatusFound, recorder.Code)
	require.Contains(t, recorder.Header().Get("Location"), "access_token=")
	requireCookieCleared(t, recorder, oauthAffiliateCodeCookieName)

	referee, err := client.User.Query().
		Where(dbuser.EmailEQ(linuxDoSyntheticEmail("ref-direct-1"))).
		Only(context.Background())
	require.NoError(t, err)
	require.Len(t, referrals.created, 1)
	require.Equal(t, referrer.ID, referrals.created[0].ReferrerID)
	require.Equal(t, referee.ID, referrals.created[0].RefereeID)
}

// 补邀请码时 body 不带 aff_code（sessionStorage 丢了的情形），靠 start 时存进 session 的那份。
func TestCompleteLinuxDoOAuthRegistrationFallsBackToSessionAffiliateCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	referrals := &oauthPendingFlowReferralRepoStub{}
	handler, client := newOAuthPendingFlowTestHandlerWithDependencies(t, oauthPendingFlowTestHandlerOptions{
		referralRepo:  referrals,
		settingValues: map[string]string{service.SettingKeyReferralEnabled: "true"},
	})
	t.Cleanup(func() { _ = client.Close() })
	ctx := context.Background()
	referrer := seedReferrer(t, client, "REF789")

	session, err := client.PendingAuthSession.Create().
		SetSessionToken("linuxdo-ref-complete-session").
		SetIntent("login").
		SetProviderType("linuxdo").
		SetProviderKey("linuxdo").
		SetProviderSubject("ref-complete-1").
		SetResolvedEmail("linuxdo-ref-complete-1@linuxdo-connect.invalid").
		SetBrowserSessionKey("linuxdo-ref-complete-browser").
		SetUpstreamIdentityClaims(map[string]any{"username": "linuxdo_ref_complete"}).
		SetLocalFlowState(map[string]any{
			oauthCompletionResponseKey: map[string]any{
				"redirect": "/dashboard",
				"error":    "invitation_required",
			},
			oauthAffiliateCodeStateKey: "REF789",
		}).
		SetExpiresAt(time.Now().UTC().Add(10 * time.Minute)).
		Save(ctx)
	require.NoError(t, err)

	body := bytes.NewBufferString(`{"invitation_code":"invite-ref","adopt_display_name":false,"adopt_avatar":false}`)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/oauth/linuxdo/complete-registration", body)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: oauthPendingSessionCookieName, Value: encodeCookieValue(session.SessionToken)})
	req.AddCookie(&http.Cookie{Name: oauthPendingBrowserCookieName, Value: encodeCookieValue("linuxdo-ref-complete-browser")})
	c.Request = req

	handler.CompleteLinuxDoOAuthRegistration(c)

	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	referee, err := client.User.Query().Where(dbuser.EmailEQ(session.ResolvedEmail)).Only(ctx)
	require.NoError(t, err)
	require.Len(t, referrals.created, 1)
	require.Equal(t, referrer.ID, referrals.created[0].ReferrerID)
	require.Equal(t, referee.ID, referrals.created[0].RefereeID)
}
