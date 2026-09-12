//go:build unit

package ip

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMaskIP(t *testing.T) {
	cases := map[string]string{
		"203.0.113.42":               "203.0.113.x",
		" 10.0.0.1 ":                 "10.0.0.x",
		"2001:db8:1:2:3:4:5:6":       "2001:db8:1::x",
		"::1":                        "0:0:0::x",
		"::ffff:192.0.2.128":         "192.0.2.x",
		"":                           "",
		"not-an-ip":                  "",
		"203.0.113.42:8080":          "",
		"203.0.113.42, 198.51.100.1": "",
	}
	for input, want := range cases {
		require.Equalf(t, want, MaskIP(input), "MaskIP(%q)", input)
	}
}
