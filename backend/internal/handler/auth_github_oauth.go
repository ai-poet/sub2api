package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/oauth"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/imroc/req/v3"
	"github.com/tidwall/gjson"
)

const (
	gitHubOAuthCookiePath        = "/api/v1/auth/oauth/github"
	gitHubOAuthStateCookieName   = "github_oauth_state"
	gitHubOAuthVerifierCookie    = "github_oauth_verifier"
	gitHubOAuthRedirectCookie    = "github_oauth_redirect"
	gitHubOAuthCookieMaxAgeSec   = 10 * 60 // 10 minutes
	gitHubOAuthDefaultRedirectTo = "/dashboard"
	gitHubOAuthDefaultFrontendCB = "/auth/github/callback"

	gitHubOAuthMaxSubjectLen = 64 - len("github-")

	gitHubAuthorizeURL  = "https://github.com/login/oauth/authorize"
	gitHubTokenURL      = "https://github.com/login/oauth/access_token"
	gitHubUserInfoURL   = "https://api.github.com/user"
	gitHubUserEmailsURL = "https://api.github.com/user/emails"
	gitHubOAuthScopes   = "read:user user:email"
)

type gitHubTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope,omitempty"`
}

type gitHubTokenExchangeError struct {
	StatusCode          int
	ProviderError       string
	ProviderDescription string
	Body                string
}

func (e *gitHubTokenExchangeError) Error() string {
	if e == nil {
		return ""
	}
	parts := []string{fmt.Sprintf("token exchange status=%d", e.StatusCode)}
	if strings.TrimSpace(e.ProviderError) != "" {
		parts = append(parts, "error="+strings.TrimSpace(e.ProviderError))
	}
	if strings.TrimSpace(e.ProviderDescription) != "" {
		parts = append(parts, "error_description="+strings.TrimSpace(e.ProviderDescription))
	}
	return strings.Join(parts, " ")
}

// setGitHubCookie sets a cookie scoped to the GitHub OAuth callback path.
func setGitHubCookie(c *gin.Context, name string, value string, maxAgeSec int, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     gitHubOAuthCookiePath,
		MaxAge:   maxAgeSec,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// clearGitHubCookie clears a cookie scoped to the GitHub OAuth callback path.
func clearGitHubCookie(c *gin.Context, name string, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     gitHubOAuthCookiePath,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// GitHubOAuthStart 启动 GitHub OAuth 登录流程。
// GET /api/v1/auth/oauth/github/start?redirect=/dashboard
func (h *AuthHandler) GitHubOAuthStart(c *gin.Context) {
	cfg, err := h.getGitHubOAuthConfig(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	state, err := oauth.GenerateState()
	if err != nil {
		response.ErrorFrom(c, infraerrors.InternalServer("OAUTH_STATE_GEN_FAILED", "failed to generate oauth state").WithCause(err))
		return
	}

	redirectTo := sanitizeFrontendRedirectPath(c.Query("redirect"))
	if redirectTo == "" {
		redirectTo = gitHubOAuthDefaultRedirectTo
	}

	secureCookie := isRequestHTTPS(c)
	setGitHubCookie(c, gitHubOAuthStateCookieName, encodeCookieValue(state), gitHubOAuthCookieMaxAgeSec, secureCookie)
	setGitHubCookie(c, gitHubOAuthRedirectCookie, encodeCookieValue(redirectTo), gitHubOAuthCookieMaxAgeSec, secureCookie)

	redirectURI := strings.TrimSpace(cfg.RedirectURL)
	if redirectURI == "" {
		response.ErrorFrom(c, infraerrors.InternalServer("OAUTH_CONFIG_INVALID", "oauth redirect url not configured"))
		return
	}

	authURL, err := buildGitHubAuthorizeURL(cfg, state, redirectURI)
	if err != nil {
		response.ErrorFrom(c, infraerrors.InternalServer("OAUTH_BUILD_URL_FAILED", "failed to build oauth authorization url").WithCause(err))
		return
	}

	c.Redirect(http.StatusFound, authURL)
}

// GitHubOAuthCallback 处理 OAuth 回调：创建/登录用户，然后重定向到前端。
// GET /api/v1/auth/oauth/github/callback?code=...&state=...
func (h *AuthHandler) GitHubOAuthCallback(c *gin.Context) {
	cfg, cfgErr := h.getGitHubOAuthConfig(c.Request.Context())
	if cfgErr != nil {
		response.ErrorFrom(c, cfgErr)
		return
	}

	frontendCallback := strings.TrimSpace(cfg.FrontendRedirectURL)
	if frontendCallback == "" {
		frontendCallback = gitHubOAuthDefaultFrontendCB
	}

	if providerErr := strings.TrimSpace(c.Query("error")); providerErr != "" {
		redirectOAuthError(c, frontendCallback, "provider_error", providerErr, c.Query("error_description"))
		return
	}

	code := strings.TrimSpace(c.Query("code"))
	state := strings.TrimSpace(c.Query("state"))
	if code == "" || state == "" {
		redirectOAuthError(c, frontendCallback, "missing_params", "missing code/state", "")
		return
	}

	secureCookie := isRequestHTTPS(c)
	defer func() {
		clearGitHubCookie(c, gitHubOAuthStateCookieName, secureCookie)
		clearGitHubCookie(c, gitHubOAuthVerifierCookie, secureCookie)
		clearGitHubCookie(c, gitHubOAuthRedirectCookie, secureCookie)
	}()

	expectedState, err := readCookieDecoded(c, gitHubOAuthStateCookieName)
	if err != nil || expectedState == "" || state != expectedState {
		redirectOAuthError(c, frontendCallback, "invalid_state", "invalid oauth state", "")
		return
	}

	redirectTo, _ := readCookieDecoded(c, gitHubOAuthRedirectCookie)
	redirectTo = sanitizeFrontendRedirectPath(redirectTo)
	if redirectTo == "" {
		redirectTo = gitHubOAuthDefaultRedirectTo
	}

	redirectURI := strings.TrimSpace(cfg.RedirectURL)
	if redirectURI == "" {
		redirectOAuthError(c, frontendCallback, "config_error", "oauth redirect url not configured", "")
		return
	}

	tokenResp, err := githubExchangeCode(c.Request.Context(), cfg, code, redirectURI)
	if err != nil {
		description := ""
		var exchangeErr *gitHubTokenExchangeError
		if errors.As(err, &exchangeErr) && exchangeErr != nil {
			log.Printf(
				"[GitHub OAuth] token exchange failed: status=%d provider_error=%q provider_description=%q body=%s",
				exchangeErr.StatusCode,
				exchangeErr.ProviderError,
				exchangeErr.ProviderDescription,
				truncateLogValue(exchangeErr.Body, 2048),
			)
			description = exchangeErr.Error()
		} else {
			log.Printf("[GitHub OAuth] token exchange failed: %v", err)
			description = err.Error()
		}
		redirectOAuthError(c, frontendCallback, "token_exchange_failed", "failed to exchange oauth code", singleLine(description))
		return
	}

	email, username, subject, err := githubFetchUserInfo(c.Request.Context(), cfg, tokenResp)
	if err != nil {
		log.Printf("[GitHub OAuth] userinfo fetch failed: %v", err)
		redirectOAuthError(c, frontendCallback, "userinfo_failed", "failed to fetch user info", "")
		return
	}

	// 安全考虑：不要把第三方返回的 email 直接映射到本地账号（可能与本地邮箱用户冲突导致账号被接管）。
	// 统一使用基于 subject 的稳定合成邮箱来做账号绑定。
	if subject != "" {
		email = githubSyntheticEmail(subject)
	}

	// 传入空邀请码；如果需要邀请码，服务层返回 ErrOAuthInvitationRequired
	tokenPair, _, err := h.authService.LoginOrRegisterOAuthWithTokenPair(c.Request.Context(), email, username, "")
	if err != nil {
		if errors.Is(err, service.ErrOAuthInvitationRequired) {
			pendingToken, tokenErr := h.authService.CreatePendingOAuthToken(email, username)
			if tokenErr != nil {
				redirectOAuthError(c, frontendCallback, "login_failed", "service_error", "")
				return
			}
			fragment := url.Values{}
			fragment.Set("error", "invitation_required")
			fragment.Set("pending_oauth_token", pendingToken)
			fragment.Set("redirect", redirectTo)
			redirectWithFragment(c, frontendCallback, fragment)
			return
		}
		// 避免把内部细节泄露给客户端；给前端保留结构化原因与提示信息即可。
		redirectOAuthError(c, frontendCallback, "login_failed", infraerrors.Reason(err), infraerrors.Message(err))
		return
	}

	fragment := url.Values{}
	fragment.Set("access_token", tokenPair.AccessToken)
	fragment.Set("refresh_token", tokenPair.RefreshToken)
	fragment.Set("expires_in", fmt.Sprintf("%d", tokenPair.ExpiresIn))
	fragment.Set("token_type", "Bearer")
	fragment.Set("redirect", redirectTo)
	redirectWithFragment(c, frontendCallback, fragment)
}

type completeGitHubOAuthRequest struct {
	PendingOAuthToken string `json:"pending_oauth_token" binding:"required"`
	InvitationCode    string `json:"invitation_code"     binding:"required"`
}

// CompleteGitHubOAuthRegistration completes a pending OAuth registration by validating
// the invitation code and creating the user account.
// POST /api/v1/auth/oauth/github/complete-registration
func (h *AuthHandler) CompleteGitHubOAuthRegistration(c *gin.Context) {
	var req completeGitHubOAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "INVALID_REQUEST", "message": err.Error()})
		return
	}

	email, username, err := h.authService.VerifyPendingOAuthToken(req.PendingOAuthToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "INVALID_TOKEN", "message": "invalid or expired registration token"})
		return
	}

	tokenPair, _, err := h.authService.LoginOrRegisterOAuthWithTokenPair(c.Request.Context(), email, username, req.InvitationCode)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_in":    tokenPair.ExpiresIn,
		"token_type":    "Bearer",
	})
}

func (h *AuthHandler) getGitHubOAuthConfig(ctx context.Context) (config.GitHubOAuthConfig, error) {
	if h != nil && h.settingSvc != nil {
		return h.settingSvc.GetGitHubOAuthConfig(ctx)
	}
	if h == nil || h.cfg == nil {
		return config.GitHubOAuthConfig{}, infraerrors.ServiceUnavailable("CONFIG_NOT_READY", "config not loaded")
	}
	if !h.cfg.GitHub.Enabled {
		return config.GitHubOAuthConfig{}, infraerrors.NotFound("OAUTH_DISABLED", "oauth login is disabled")
	}
	return h.cfg.GitHub, nil
}

func githubExchangeCode(
	ctx context.Context,
	cfg config.GitHubOAuthConfig,
	code string,
	redirectURI string,
) (*gitHubTokenResponse, error) {
	client := req.C().SetTimeout(30 * time.Second)

	form := url.Values{}
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)

	r := client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json")

	resp, err := r.SetFormDataFromValues(form).Post(gitHubTokenURL)
	if err != nil {
		return nil, fmt.Errorf("request token: %w", err)
	}
	body := strings.TrimSpace(resp.String())
	if !resp.IsSuccessState() {
		providerErr, providerDesc := parseOAuthProviderError(body)
		return nil, &gitHubTokenExchangeError{
			StatusCode:          resp.StatusCode,
			ProviderError:       providerErr,
			ProviderDescription: providerDesc,
			Body:                body,
		}
	}

	accessToken := strings.TrimSpace(gjson.Get(body, "access_token").String())
	if accessToken == "" {
		// GitHub may return 200 with an error field
		providerErr, providerDesc := parseOAuthProviderError(body)
		return nil, &gitHubTokenExchangeError{
			StatusCode:          resp.StatusCode,
			ProviderError:       providerErr,
			ProviderDescription: providerDesc,
			Body:                body,
		}
	}

	tokenType := strings.TrimSpace(gjson.Get(body, "token_type").String())
	if tokenType == "" {
		tokenType = "Bearer"
	}
	scope := strings.TrimSpace(gjson.Get(body, "scope").String())

	return &gitHubTokenResponse{
		AccessToken: accessToken,
		TokenType:   tokenType,
		Scope:       scope,
	}, nil
}

func githubFetchUserInfo(
	ctx context.Context,
	cfg config.GitHubOAuthConfig,
	token *gitHubTokenResponse,
) (email string, username string, subject string, err error) {
	client := req.C().SetTimeout(30 * time.Second)
	authorization, err := buildBearerAuthorization(token.TokenType, token.AccessToken)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid token for userinfo request: %w", err)
	}

	resp, err := client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", authorization).
		Get(gitHubUserInfoURL)
	if err != nil {
		return "", "", "", fmt.Errorf("request userinfo: %w", err)
	}
	if !resp.IsSuccessState() {
		return "", "", "", fmt.Errorf("userinfo status=%d", resp.StatusCode)
	}

	body := resp.String()

	// GitHub returns id as a number
	idResult := gjson.Get(body, "id")
	if !idResult.Exists() {
		return "", "", "", errors.New("userinfo missing id field")
	}
	subject = strconv.FormatInt(idResult.Int(), 10)
	if !isSafeGitHubSubject(subject) {
		return "", "", "", errors.New("userinfo returned invalid id field")
	}

	username = strings.TrimSpace(gjson.Get(body, "login").String())
	email = strings.TrimSpace(gjson.Get(body, "email").String())

	// GitHub may return null email if the user's email is private.
	// Fetch from /user/emails and pick the primary verified one.
	if email == "" {
		email = githubFetchPrimaryEmail(ctx, authorization)
	}

	if email == "" {
		email = githubSyntheticEmail(subject)
	}

	if username == "" {
		username = "github_" + subject
	}

	return email, username, subject, nil
}

// githubFetchPrimaryEmail fetches the user's primary verified email from /user/emails.
func githubFetchPrimaryEmail(ctx context.Context, authorization string) string {
	client := req.C().SetTimeout(15 * time.Second)
	resp, err := client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json").
		SetHeader("Authorization", authorization).
		Get(gitHubUserEmailsURL)
	if err != nil || !resp.IsSuccessState() {
		return ""
	}

	body := resp.String()
	// Response is a JSON array of {email, primary, verified, visibility}
	var primaryEmail string
	gjson.Parse(body).ForEach(func(_, value gjson.Result) bool {
		if value.Get("primary").Bool() && value.Get("verified").Bool() {
			primaryEmail = strings.TrimSpace(value.Get("email").String())
			return false // stop iteration
		}
		return true
	})
	return primaryEmail
}

func buildGitHubAuthorizeURL(cfg config.GitHubOAuthConfig, state string, redirectURI string) (string, error) {
	u, err := url.Parse(gitHubAuthorizeURL)
	if err != nil {
		return "", fmt.Errorf("parse authorize_url: %w", err)
	}

	q := u.Query()
	q.Set("client_id", cfg.ClientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("scope", gitHubOAuthScopes)
	q.Set("state", state)

	u.RawQuery = q.Encode()
	return u.String(), nil
}

func isSafeGitHubSubject(subject string) bool {
	subject = strings.TrimSpace(subject)
	if subject == "" || len(subject) > gitHubOAuthMaxSubjectLen {
		return false
	}
	for _, r := range subject {
		switch {
		case r >= '0' && r <= '9':
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r == '_' || r == '-':
		default:
			return false
		}
	}
	return true
}

func githubSyntheticEmail(subject string) string {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return ""
	}
	return "github-" + subject + service.GitHubOAuthSyntheticEmailDomain
}
