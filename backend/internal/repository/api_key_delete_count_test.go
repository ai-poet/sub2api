package repository

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// fork：API Key 删除频率计数，与上游创建计数同一固定窗口语义。
func TestAPIKeyCacheIncrementDeleteCountUsesFixedWindow(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer func() { _ = client.Close() }()
	cache := NewAPIKeyCache(client)
	counter, ok := cache.(service.APIKeyDeleteCounter)
	require.True(t, ok, "Redis 缓存必须实现删除计数，否则删除限流静默失效")
	ctx := context.Background()
	key := apiKeyDeleteCountKey(7)

	count, err := counter.IncrementDeleteCount(ctx, 7, time.Hour)
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
	require.Equal(t, time.Hour, server.TTL(key))

	// 后续删除不得延长窗口
	server.FastForward(40 * time.Minute)
	count, err = counter.IncrementDeleteCount(ctx, 7, time.Hour)
	require.NoError(t, err)
	require.Equal(t, int64(2), count)
	require.Equal(t, 20*time.Minute, server.TTL(key))

	// 删除计数与创建计数互不影响
	createCount, err := cache.IncrementCreateCount(ctx, 7, time.Hour)
	require.NoError(t, err)
	require.Equal(t, int64(1), createCount)

	// 窗口到期后重新计数
	server.FastForward(21 * time.Minute)
	count, err = counter.IncrementDeleteCount(ctx, 7, time.Hour)
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
}
