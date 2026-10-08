//go:build unit

package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeHandoffOrigin(t *testing.T) {
	cases := map[string]string{
		"https://cheaprouter.cc":         "https://cheaprouter.cc",
		" HTTPS://CDN.CheapRouter.cc/ ":  "https://cdn.cheaprouter.cc",
		"https://cdn.cheaprouter.cc:443": "https://cdn.cheaprouter.cc",
		"http://localhost:80":            "http://localhost",
		"http://localhost:3000":          "http://localhost:3000",
		"https://[2001:db8::1]:8443":     "https://[2001:db8::1]:8443",
		"https://cheaprouter.cc:8443/":   "https://cheaprouter.cc:8443",
	}
	for raw, want := range cases {
		got, err := NormalizeHandoffOrigin(raw)
		require.NoErrorf(t, err, "raw %q", raw)
		require.Equalf(t, want, got, "raw %q", raw)
	}

	for _, raw := range []string{
		"",
		"cheaprouter.cc",
		"ftp://cheaprouter.cc",
		"javascript:alert(1)",
		"https://cheaprouter.cc/login",
		"https://cheaprouter.cc?x=1",
		"https://cheaprouter.cc#frag",
		"https://user:pass@cheaprouter.cc",
	} {
		_, err := NormalizeHandoffOrigin(raw)
		require.Errorf(t, err, "raw %q", raw)
	}
}

func TestNormalizeSessionHandoffConfig(t *testing.T) {
	cfg := SessionHandoffConfig{
		LoginOrigin:  "https://CheapRouter.cc/",
		AliasOrigins: []string{" https://cdn.cheaprouter.cc ", "https://cdn.cheaprouter.cc:443", "", "https://edge.cheaprouter.cc"},
	}
	require.NoError(t, normalizeSessionHandoffConfig(&cfg))
	require.Equal(t, "https://cheaprouter.cc", cfg.LoginOrigin)
	require.Equal(t, []string{"https://cdn.cheaprouter.cc", "https://edge.cheaprouter.cc"}, cfg.AliasOrigins)
	require.True(t, cfg.Enabled())
	require.True(t, cfg.AllowsOrigin("https://cheaprouter.cc"))
	require.True(t, cfg.AllowsOrigin("https://edge.cheaprouter.cc"))
	require.False(t, cfg.AllowsOrigin("https://other.cheaprouter.cc"))
	require.False(t, cfg.AllowsOrigin(""))

	empty := SessionHandoffConfig{AliasOrigins: []string{"  "}}
	require.NoError(t, normalizeSessionHandoffConfig(&empty))
	require.False(t, empty.Enabled())
	require.False(t, empty.AllowsOrigin("https://cheaprouter.cc"))

	loginOnly := SessionHandoffConfig{LoginOrigin: "https://cheaprouter.cc"}
	require.NoError(t, normalizeSessionHandoffConfig(&loginOnly))
	require.False(t, loginOnly.Enabled(), "no alias means nothing to hand off to")

	require.Error(t, normalizeSessionHandoffConfig(&SessionHandoffConfig{AliasOrigins: []string{"https://cdn.cheaprouter.cc"}}), "aliases without a login origin")
	require.Error(t, normalizeSessionHandoffConfig(&SessionHandoffConfig{LoginOrigin: "https://cheaprouter.cc", AliasOrigins: []string{"https://cheaprouter.cc:443"}}), "alias repeats the login origin")
	require.Error(t, normalizeSessionHandoffConfig(&SessionHandoffConfig{LoginOrigin: "https://cheaprouter.cc/app", AliasOrigins: []string{"https://cdn.cheaprouter.cc"}}))
	require.Error(t, normalizeSessionHandoffConfig(&SessionHandoffConfig{LoginOrigin: "https://cheaprouter.cc", AliasOrigins: []string{"cdn.cheaprouter.cc"}}))
}

// 生产部署常用环境变量：别名列表按逗号分隔，结果与 config.yaml 写法一致。
func TestLoadSessionHandoffFromEnv(t *testing.T) {
	resetViperWithJWTSecret(t)
	t.Setenv("SESSION_HANDOFF_LOGIN_ORIGIN", "https://CheapRouter.cc/")
	t.Setenv("SESSION_HANDOFF_ALIAS_ORIGINS", "https://cdn.cheaprouter.cc, https://edge.cheaprouter.cc")

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "https://cheaprouter.cc", cfg.SessionHandoff.LoginOrigin)
	require.Equal(t, []string{"https://cdn.cheaprouter.cc", "https://edge.cheaprouter.cc"}, cfg.SessionHandoff.AliasOrigins)
	require.True(t, cfg.SessionHandoff.Enabled())
}

func TestLoadSessionHandoffRejectsHalfConfig(t *testing.T) {
	resetViperWithJWTSecret(t)
	t.Setenv("SESSION_HANDOFF_ALIAS_ORIGINS", "https://cdn.cheaprouter.cc")

	_, err := Load()
	require.ErrorContains(t, err, "session_handoff")
}
