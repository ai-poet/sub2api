package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// jsonMap / jsonSlice 断言解码后的 JSON 节点类型，类型不符时让测试失败而不是 panic。
func jsonMap(t *testing.T, v any) map[string]any {
	t.Helper()
	m, ok := v.(map[string]any)
	require.Truef(t, ok, "expected JSON object, got %T", v)
	return m
}

func jsonSlice(t *testing.T, v any) []any {
	t.Helper()
	s, ok := v.([]any)
	require.Truef(t, ok, "expected JSON array, got %T", v)
	return s
}
