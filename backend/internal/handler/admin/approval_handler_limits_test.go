package admin

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

// pending-count 顺带返回生效中的数量上限（未挂载设置仓储时为默认常量），审批页据此分批。
func TestApprovalHandler_PendingCountCarriesLimits(t *testing.T) {
	repo := newApprovalRepoMem()
	seedApproval(t, repo, 2, "ops@example.com")
	router, _ := newApprovalTestRouter(t, repo, &approvalTestIdentity{userID: 1, email: "admin@example.com", role: service.RoleAdmin, authMethod: "jwt"})

	rec := doApproval(router, http.MethodGet, "/admin/approvals/pending-count", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var resp struct {
		Data struct {
			Pending      int64 `json:"pending"`
			PendingLimit int   `json:"pending_limit"`
			BatchLimit   int   `json:"batch_limit"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, int64(1), resp.Data.Pending)
	require.Equal(t, service.AdminApprovalPendingLimitPerUser, resp.Data.PendingLimit)
	require.Equal(t, service.AdminApprovalBatchLimit, resp.Data.BatchLimit)
}

// 绑定层只挡离谱的请求体（硬上限 500）；真正生效的批量上限由服务层按设置校验。
func TestApprovalHandler_BatchApproveLimits(t *testing.T) {
	repo := newApprovalRepoMem()
	router, svc := newApprovalTestRouter(t, repo, &approvalTestIdentity{userID: 1, email: "admin@example.com", role: service.RoleAdmin, authMethod: "jwt"})
	router.POST("/admin/approvals/batch-approve", NewApprovalHandler(svc).BatchApprove)

	buildIDs := func(n int) string {
		ids := make([]int64, n)
		for i := range ids {
			ids[i] = int64(i + 1)
		}
		body, err := json.Marshal(map[string]any{"ids": ids})
		require.NoError(t, err)
		return string(body)
	}

	// 默认上限 50：51 条走到服务层被拒（业务错误，而非绑定错误）
	rec := doApproval(router, http.MethodPost, "/admin/approvals/batch-approve", buildIDs(service.AdminApprovalBatchLimit+1))
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), "too many approval ids")

	// 超过绑定层硬上限（500）直接在绑定阶段 400
	rec = doApproval(router, http.MethodPost, "/admin/approvals/batch-approve", buildIDs(service.AdminApprovalBatchLimitMax+1))
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), "Invalid request body")
}
