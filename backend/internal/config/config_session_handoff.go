package config

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/viper"
)

// SessionHandoffConfig 多域名登录交接（fork 本地功能）。
//
// 同一套服务挂在多个域名上（例如主域名 + CDN 加速域名）时，第三方 OAuth 的回调只登记在
// login_origin 上，而 OAuth 的 state / PKCE cookie 只属于发起登录的域名，登录 token 也只存在
// 各域名自己的 localStorage 里。于是从 alias_origins 里的域名进入登录 / 注册页时，前端先跳到
// login_origin 完成登录，再用一次性交接码（绑定接收方生成的 S256 PKCE challenge）把会话带回原域名；
// 每个域名各拿一套独立的 token 家族。login_origin 为空或 alias_origins 为空时功能关闭。
type SessionHandoffConfig struct {
	// LoginOrigin 第三方登录回调所在的域名，形如 https://example.com。
	LoginOrigin string `mapstructure:"login_origin"`
	// AliasOrigins 允许接收交接的其它域名（不含 LoginOrigin）。
	AliasOrigins []string `mapstructure:"alias_origins"`
}

// Enabled 报告交接是否已配置。
func (c SessionHandoffConfig) Enabled() bool {
	return c.LoginOrigin != "" && len(c.AliasOrigins) > 0
}

// AllowsOrigin 报告 origin（已规范化）是否参与交接：登录域名或任一别名域名。
func (c SessionHandoffConfig) AllowsOrigin(origin string) bool {
	if !c.Enabled() || origin == "" {
		return false
	}
	if origin == c.LoginOrigin {
		return true
	}
	for _, alias := range c.AliasOrigins {
		if origin == alias {
			return true
		}
	}
	return false
}

func setSessionHandoffDefaults() {
	viper.SetDefault("session_handoff.login_origin", "")
	viper.SetDefault("session_handoff.alias_origins", []string{})
}

// normalizeSessionHandoffConfig 规范化并校验交接配置；只配了一半视为错误，避免静默失效。
func normalizeSessionHandoffConfig(c *SessionHandoffConfig) error {
	login := strings.TrimSpace(c.LoginOrigin)
	aliases := make([]string, 0, len(c.AliasOrigins))
	seen := map[string]struct{}{}
	for _, raw := range c.AliasOrigins {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		origin, err := NormalizeHandoffOrigin(raw)
		if err != nil {
			return fmt.Errorf("alias_origins: %w", err)
		}
		if _, dup := seen[origin]; dup {
			continue
		}
		seen[origin] = struct{}{}
		aliases = append(aliases, origin)
	}
	if login == "" {
		if len(aliases) > 0 {
			return fmt.Errorf("login_origin is required when alias_origins is set")
		}
		c.LoginOrigin = ""
		c.AliasOrigins = nil
		return nil
	}
	normalizedLogin, err := NormalizeHandoffOrigin(login)
	if err != nil {
		return fmt.Errorf("login_origin: %w", err)
	}
	for _, alias := range aliases {
		if alias == normalizedLogin {
			return fmt.Errorf("alias_origins must not repeat login_origin %s", normalizedLogin)
		}
	}
	c.LoginOrigin = normalizedLogin
	c.AliasOrigins = aliases
	return nil
}

// NormalizeHandoffOrigin 把 scheme://host[:port]（可带结尾斜杠）规范成浏览器 location.origin 的形式：
// 只接受 http / https，小写，去掉默认端口；带路径、查询、片段或用户信息的一律拒绝。
func NormalizeHandoffOrigin(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	u, err := url.Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("invalid origin %q: %w", trimmed, err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", fmt.Errorf("invalid origin %q: scheme must be http or https", trimmed)
	}
	if u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || u.Opaque != "" {
		return "", fmt.Errorf("invalid origin %q: want scheme://host[:port]", trimmed)
	}
	host := strings.ToLower(u.Hostname())
	port := u.Port()
	if (scheme == "https" && port == "443") || (scheme == "http" && port == "80") {
		port = ""
	}
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	if port != "" {
		host += ":" + port
	}
	return scheme + "://" + host, nil
}
