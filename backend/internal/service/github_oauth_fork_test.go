package service

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func pendingOAuthTokenTestService(secret string) *AuthService {
	return &AuthService{cfg: &config.Config{JWT: config.JWTConfig{Secret: secret}}}
}

// pending token 是 GitHub「需要邀请码」分支里唯一能把 start 时捕获的推荐码带到
// complete-registration 的载体，往返必须原样保留。
func TestPendingOAuthTokenCarriesAffiliateCode(t *testing.T) {
	svc := pendingOAuthTokenTestService("test-secret")

	token, err := svc.CreatePendingOAuthToken("github-1@github-oauth.invalid", "octocat", "  REF123  ")
	require.NoError(t, err)

	identity, err := svc.VerifyPendingOAuthToken(token)
	require.NoError(t, err)
	require.Equal(t, PendingOAuthIdentity{
		Email:    "github-1@github-oauth.invalid",
		Username: "octocat",
		AffCode:  "REF123",
	}, identity)
}

func TestPendingOAuthTokenOmitsEmptyAffiliateCode(t *testing.T) {
	svc := pendingOAuthTokenTestService("test-secret")

	token, err := svc.CreatePendingOAuthToken("github-2@github-oauth.invalid", "hubot", "")
	require.NoError(t, err)

	identity, err := svc.VerifyPendingOAuthToken(token)
	require.NoError(t, err)
	require.Equal(t, PendingOAuthIdentity{Email: "github-2@github-oauth.invalid", Username: "hubot"}, identity)

	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	require.NoError(t, err)
	require.NotContains(t, string(payload), "aff_code")
}

func TestPendingOAuthTokenRejectsTamperedOrForeignTokens(t *testing.T) {
	svc := pendingOAuthTokenTestService("test-secret")
	token, err := svc.CreatePendingOAuthToken("github-3@github-oauth.invalid", "hubot", "REF123")
	require.NoError(t, err)

	_, err = svc.VerifyPendingOAuthToken(token + "x")
	require.ErrorIs(t, err, ErrInvalidToken)

	_, err = pendingOAuthTokenTestService("other-secret").VerifyPendingOAuthToken(token)
	require.ErrorIs(t, err, ErrInvalidToken)

	now := time.Now()
	wrongPurpose := jwt.NewWithClaims(jwt.SigningMethodHS256, &pendingOAuthClaims{
		Email:    "github-3@github-oauth.invalid",
		Username: "hubot",
		AffCode:  "REF123",
		Purpose:  "something_else",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	})
	signed, err := wrongPurpose.SignedString([]byte("test-secret"))
	require.NoError(t, err)
	_, err = svc.VerifyPendingOAuthToken(signed)
	require.ErrorIs(t, err, ErrInvalidToken)
}
