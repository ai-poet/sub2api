package handler

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGitHubOAuthStatePayload_JSONRoundTrip(t *testing.T) {
	p := gitHubOAuthStatePayload{
		N: "nonce-value",
		R: "/auth/paseo?endpoint=https%3A%2F%2Fexample.com",
	}
	raw, err := json.Marshal(p)
	require.NoError(t, err)

	var decoded gitHubOAuthStatePayload
	require.NoError(t, json.Unmarshal(raw, &decoded))
	require.Equal(t, p.N, decoded.N)
	require.Equal(t, p.R, decoded.R)
}
