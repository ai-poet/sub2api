package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSanitizeProbeErrorDetail(t *testing.T) {
	t.Run("url error strips upstream url", func(t *testing.T) {
		err := &url.Error{
			Op:  "Post",
			URL: "https://new-api-upstream.zeabur.app/v1/responses",
			Err: context.DeadlineExceeded,
		}
		got := sanitizeProbeErrorDetail(err)
		require.Equal(t, "context deadline exceeded", got)
		require.NotContains(t, got, "zeabur.app")
	})

	t.Run("wrapped url error keeps outer context", func(t *testing.T) {
		urlErr := &url.Error{
			Op:  "Post",
			URL: "https://internal-host.example.com/v1/messages",
			Err: context.DeadlineExceeded,
		}
		err := fmt.Errorf("failed to get grok access token: %w", urlErr)
		got := sanitizeProbeErrorDetail(err)
		require.Equal(t, "failed to get grok access token: context deadline exceeded", got)
	})

	t.Run("dns error drops hostname", func(t *testing.T) {
		err := &url.Error{
			Op:  "Post",
			URL: "https://internal-host.example.com/v1/responses",
			Err: &net.OpError{
				Op:  "dial",
				Net: "tcp",
				Err: &net.DNSError{Err: "no such host", Name: "internal-host.example.com", IsNotFound: true},
			},
		}
		got := sanitizeProbeErrorDetail(err)
		require.Equal(t, "dns lookup failed: no such host", got)
		require.NotContains(t, got, "internal-host")
	})

	t.Run("dial error drops address", func(t *testing.T) {
		err := &url.Error{
			Op:  "Post",
			URL: "https://internal-host.example.com/v1/responses",
			Err: &net.OpError{
				Op:   "dial",
				Net:  "tcp",
				Addr: &net.TCPAddr{IP: net.IPv4(10, 0, 0, 1), Port: 443},
				Err:  errors.New("connect: connection refused"),
			},
		}
		got := sanitizeProbeErrorDetail(err)
		require.Equal(t, "dial failed: connect: connection refused", got)
		require.NotContains(t, got, "10.0.0.1")
	})

	t.Run("plain error with embedded url is redacted", func(t *testing.T) {
		err := errors.New(`upstream said: visit https://relay.example.com/console for details`)
		got := sanitizeProbeErrorDetail(err)
		require.Equal(t, "upstream said: visit [upstream] for details", got)
	})

	t.Run("nil error", func(t *testing.T) {
		require.Equal(t, "", sanitizeProbeErrorDetail(nil))
	})
}

func TestRedactProbeUpstreamAddresses(t *testing.T) {
	require.Equal(t, "", redactProbeUpstreamAddresses(""))
	require.Equal(t, "no url here", redactProbeUpstreamAddresses("no url here"))
	require.Equal(
		t,
		`Post "[upstream]: context deadline exceeded`,
		redactProbeUpstreamAddresses(`Post "https://host.example.com/v1/responses": context deadline exceeded`),
	)
}
