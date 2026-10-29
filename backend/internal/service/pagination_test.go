package service

import (
	"math"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

func TestPaginateAPIKeysLargePage(t *testing.T) {
	t.Parallel()
	for _, size := range []int{0, 1, 2, 3, 20, 21, 1000, 1001} {
		params := pagination.PaginationParams{Page: math.MaxInt, PageSize: size}
		require.Empty(t, paginateAPIKeys([]APIKey{{ID: 1}}, params), "page_size=%d", size)
	}
}

func TestPaginateBalanceHistoryCodesLargePage(t *testing.T) {
	t.Parallel()
	for _, size := range []int{0, 1, 2, 3, 20, 21, 1000, 1001} {
		params := pagination.PaginationParams{Page: math.MaxInt, PageSize: size}
		require.Empty(t, paginateBalanceHistoryCodes([]RedeemCode{{ID: 1}}, params), "page_size=%d", size)
	}
}

// 上游的 mergeBalanceHistoryCodes 还会合并邀请返利记录；本仓库移除了邀请返利，
// 切页由 paginateBalanceHistoryCodes 单独负责，这里守住偏移量语义。
func TestPaginateBalanceHistoryCodesOffset(t *testing.T) {
	t.Parallel()
	codes := []RedeemCode{{ID: 1}, {ID: 2}, {ID: 3}, {ID: 4}, {ID: 5}}

	got := paginateBalanceHistoryCodes(codes, pagination.PaginationParams{Page: 2, PageSize: 2})
	require.Len(t, got, 2)
	require.Equal(t, int64(3), got[0].ID)
	require.Equal(t, int64(4), got[1].ID)

	got = paginateBalanceHistoryCodes(codes, pagination.PaginationParams{Page: 3, PageSize: 2})
	require.Len(t, got, 1)
	require.Equal(t, int64(5), got[0].ID)

	require.Empty(t, paginateBalanceHistoryCodes(codes, pagination.PaginationParams{Page: 9, PageSize: 2}))
}
