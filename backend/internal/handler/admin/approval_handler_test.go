package admin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 内存审批仓储：只实现 handler 测试需要的行为。
type approvalRepoMem struct {
	mu    sync.Mutex
	seq   int64
	items map[int64]*service.AdminApprovalRequest
}

func newApprovalRepoMem() *approvalRepoMem {
	return &approvalRepoMem{items: map[int64]*service.AdminApprovalRequest{}}
}

func (r *approvalRepoMem) Create(_ context.Context, req *service.AdminApprovalRequest) (*service.AdminApprovalRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	clone := *req
	clone.ID = r.seq
	r.items[clone.ID] = &clone
	out := clone
	return &out, nil
}

func (r *approvalRepoMem) GetByID(_ context.Context, id int64) (*service.AdminApprovalRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok {
		return nil, service.ErrApprovalNotFound
	}
	out := *item
	return &out, nil
}

func (r *approvalRepoMem) List(_ context.Context, f *service.AdminApprovalFilter) ([]*service.AdminApprovalRequest, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*service.AdminApprovalRequest, 0)
	for _, item := range r.items {
		if f != nil && f.RequesterUserID != nil && item.RequesterUserID != *f.RequesterUserID {
			continue
		}
		if f != nil && f.Status != "" && f.Status != service.ApprovalStatusFilterProcessed && item.Status != f.Status {
			continue
		}
		clone := *item
		out = append(out, &clone)
	}
	return out, int64(len(out)), nil
}

func (r *approvalRepoMem) CountPending(_ context.Context, now time.Time, requester *int64) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var n int64
	for _, item := range r.items {
		if item.Status == service.ApprovalStatusPending && item.ExpiresAt.After(now) && (requester == nil || item.RequesterUserID == *requester) {
			n++
		}
	}
	return n, nil
}

func (r *approvalRepoMem) TransitionToExecuting(_ context.Context, id, approverID int64, approverEmail string, now time.Time) (*service.AdminApprovalRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok {
		return nil, service.ErrApprovalNotFound
	}
	if item.Status != service.ApprovalStatusPending || !item.ExpiresAt.After(now) {
		return nil, service.ErrApprovalNotPending
	}
	item.Status = service.ApprovalStatusExecuting
	item.DecidedByUserID = &approverID
	item.DecidedByEmail = approverEmail
	item.DecidedAt = &now
	out := *item
	return &out, nil
}

func (r *approvalRepoMem) FinishExecution(_ context.Context, id int64, status string, statusCode int, body, errText string, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if item, ok := r.items[id]; ok && item.Status == service.ApprovalStatusExecuting {
		item.Status = status
		item.ExecutedAt = &now
		item.ResultStatusCode = &statusCode
		item.ResultBody = body
		item.ResultError = errText
	}
	return nil
}

func (r *approvalRepoMem) Decide(_ context.Context, id int64, toStatus string, actorID int64, actorEmail, reason string, now time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok || item.Status != service.ApprovalStatusPending || !item.ExpiresAt.After(now) {
		return false, nil
	}
	item.Status = toStatus
	item.DecidedByUserID = &actorID
	item.DecidedByEmail = actorEmail
	item.DecisionReason = reason
	item.DecidedAt = &now
	return true, nil
}

func (r *approvalRepoMem) MarkNotified(context.Context, int64, time.Time) error { return nil }

func (r *approvalRepoMem) ExpirePending(context.Context, time.Time) (int64, error) { return 0, nil }

func (r *approvalRepoMem) FailStuckExecuting(context.Context, time.Time) (int64, error) {
	return 0, nil
}

type approvalEncMem struct{}

func (approvalEncMem) Encrypt(plaintext string) (string, error) {
	return base64.StdEncoding.EncodeToString([]byte(plaintext)), nil
}

func (approvalEncMem) Decrypt(ciphertext string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	return string(raw), err
}

type approvalTestIdentity struct {
	userID     int64
	email      string
	role       string
	authMethod string
}

func newApprovalTestRouter(t *testing.T, repo *approvalRepoMem, identity *approvalTestIdentity) (*gin.Engine, *service.AdminApprovalService) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	svc := service.NewAdminApprovalService(repo, nil, nil, nil, nil, approvalEncMem{})
	svc.SetDispatcher(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"data":{"replayed":true}}`))
	}))
	h := NewApprovalHandler(svc)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: identity.userID, Concurrency: 1})
		c.Set(string(middleware.ContextKeyUserRole), identity.role)
		c.Set(middleware.ContextKeyAuthEmail, identity.email)
		c.Set("auth_method", identity.authMethod)
		c.Next()
	})
	router.GET("/admin/approvals", h.List)
	router.GET("/admin/approvals/pending-count", h.PendingCount)
	router.GET("/admin/approvals/:id", h.Get)
	router.POST("/admin/approvals/:id/approve", h.Approve)
	router.POST("/admin/approvals/:id/reject", h.Reject)
	router.POST("/admin/approvals/:id/cancel", h.Cancel)
	return router, svc
}

func seedApproval(t *testing.T, repo *approvalRepoMem, requester int64, email string) *service.AdminApprovalRequest {
	t.Helper()
	created, err := repo.Create(context.Background(), &service.AdminApprovalRequest{
		Status:              service.ApprovalStatusPending,
		Action:              "admin.users.balance.create",
		Method:              http.MethodPost,
		RouteTemplate:       "/api/v1/admin/users/:id/balance",
		RequestPath:         "/api/v1/admin/users/7/balance",
		RequestBodyRedacted: `{"balance":10}`,
		TargetSummary:       "target@example.com",
		RequesterUserID:     requester,
		RequesterEmail:      email,
		RequesterIP:         "192.0.2.10",
		ResultBody:          `{"secret":"x"}`,
		ExpiresAt:           time.Now().Add(time.Hour),
	})
	require.NoError(t, err)
	return created
}

func doApproval(router *gin.Engine, method, target, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestApprovalHandler_OperatorSeesOnlyOwnRequestsWithProjection(t *testing.T) {
	repo := newApprovalRepoMem()
	mine := seedApproval(t, repo, 2, "ops@example.com")
	seedApproval(t, repo, 3, "other@example.com")
	router, _ := newApprovalTestRouter(t, repo, &approvalTestIdentity{userID: 2, email: "ops@example.com", role: service.RoleOperator, authMethod: "jwt"})

	rec := doApproval(router, http.MethodGet, "/admin/approvals?status=pending", "")
	require.Equal(t, http.StatusOK, rec.Code)
	var list struct {
		Data struct {
			Items []map[string]any `json:"items"`
			Total int64            `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
	require.Equal(t, int64(1), list.Data.Total)
	require.Len(t, list.Data.Items, 1)
	require.EqualValues(t, mine.ID, list.Data.Items[0]["id"])
	require.NotContains(t, list.Data.Items[0], "requester_ip", "operator 视图不带申请人 IP")
	require.NotContains(t, list.Data.Items[0], "result_body", "operator 视图不带重放响应体")
	require.Equal(t, `{"balance":10}`, list.Data.Items[0]["request_body"])

	rec = doApproval(router, http.MethodGet, "/admin/approvals/pending-count", "")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"pending":1`)

	rec = doApproval(router, http.MethodGet, "/admin/approvals/2", "")
	require.Equal(t, http.StatusNotFound, rec.Code, "他人的申请对 operator 不可见")

	rec = doApproval(router, http.MethodPost, "/admin/approvals/2/cancel", "")
	require.Equal(t, http.StatusNotFound, rec.Code)
	rec = doApproval(router, http.MethodPost, "/admin/approvals/1/cancel", "")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"status":"cancelled"`)
	require.NotContains(t, rec.Body.String(), "requester_ip")
}

func TestApprovalHandler_AdminApprovesAndRejects(t *testing.T) {
	repo := newApprovalRepoMem()
	first := seedApproval(t, repo, 2, "ops@example.com")
	second := seedApproval(t, repo, 3, "other@example.com")
	router, _ := newApprovalTestRouter(t, repo, &approvalTestIdentity{userID: 1, email: "admin@example.com", role: service.RoleAdmin, authMethod: "jwt"})

	rec := doApproval(router, http.MethodGet, "/admin/approvals", "")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"total":2`)
	require.Contains(t, rec.Body.String(), `"requester_ip":"192.0.2.10"`, "管理员视图带申请人 IP")

	rec = doApproval(router, http.MethodGet, "/admin/approvals/pending-count", "")
	require.Contains(t, rec.Body.String(), `"pending":2`)

	rec = doApproval(router, http.MethodPost, "/admin/approvals/1/approve", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var approved struct {
		Data struct {
			Approval struct {
				Status           string `json:"status"`
				ResultStatusCode int    `json:"result_status_code"`
			} `json:"approval"`
			Replay struct {
				StatusCode int `json:"status_code"`
			} `json:"replay"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &approved))
	require.Equal(t, service.ApprovalStatusApproved, approved.Data.Approval.Status)
	require.Equal(t, http.StatusOK, approved.Data.Approval.ResultStatusCode)
	require.Equal(t, http.StatusOK, approved.Data.Replay.StatusCode)
	_ = first

	rec = doApproval(router, http.MethodPost, "/admin/approvals/1/approve", "")
	require.Equal(t, http.StatusConflict, rec.Code, "重复通过")

	rec = doApproval(router, http.MethodPost, "/admin/approvals/2/reject", `{"reason":"duplicate request"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Contains(t, rec.Body.String(), `"status":"rejected"`)
	require.Contains(t, rec.Body.String(), `"decision_reason":"duplicate request"`)
	_ = second

	rec = doApproval(router, http.MethodPost, "/admin/approvals/abc/reject", "")
	require.Equal(t, http.StatusBadRequest, rec.Code)
	rec = doApproval(router, http.MethodPost, "/admin/approvals/2/reject", `{"reason":"`+strings.Repeat("x", 600)+`"}`)
	require.Equal(t, http.StatusBadRequest, rec.Code, "理由过长")
	rec = doApproval(router, http.MethodGet, "/admin/approvals/999", "")
	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestApprovalHandler_AdminAPIKeySessionCannotApprove(t *testing.T) {
	repo := newApprovalRepoMem()
	seedApproval(t, repo, 2, "ops@example.com")
	router, _ := newApprovalTestRouter(t, repo, &approvalTestIdentity{userID: 1, email: "admin@example.com", role: service.RoleAdmin, authMethod: service.AuditAuthMethodAdminAPIKey})

	rec := doApproval(router, http.MethodPost, "/admin/approvals/1/approve", "")
	require.Equal(t, http.StatusForbidden, rec.Code)
	require.Contains(t, rec.Body.String(), "APPROVAL_APPROVER_INVALID")
	stored, err := repo.GetByID(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, service.ApprovalStatusPending, stored.Status)
}

func TestApprovalHandler_BatchApprove(t *testing.T) {
	repo := newApprovalRepoMem()
	seedApproval(t, repo, 2, "ops@example.com")
	seedApproval(t, repo, 3, "other@example.com")
	router, _ := newApprovalTestRouter(t, repo, &approvalTestIdentity{userID: 1, email: "admin@example.com", role: service.RoleAdmin, authMethod: "jwt"})
	router.POST("/admin/approvals/batch-approve", NewApprovalHandler(service.NewAdminApprovalService(repo, nil, nil, nil, nil, approvalEncMem{})).BatchApprove)

	rec := doApproval(router, http.MethodPost, "/admin/approvals/batch-approve", `{"ids":[1,2,1]}`)
	require.Equal(t, http.StatusServiceUnavailable, rec.Code, "该 handler 实例未注入 dispatcher，必须 fail-closed")

	// 用带 dispatcher 的服务重新挂路由
	router2, svc := newApprovalTestRouter(t, repo, &approvalTestIdentity{userID: 1, email: "admin@example.com", role: service.RoleAdmin, authMethod: "jwt"})
	router2.POST("/admin/approvals/batch-approve", NewApprovalHandler(svc).BatchApprove)
	rec = doApproval(router2, http.MethodPost, "/admin/approvals/batch-approve", `{"ids":[1,2,1]}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var resp struct {
		Data struct {
			Approved int `json:"approved"`
			Failed   int `json:"failed"`
			Skipped  int `json:"skipped"`
			Results  []struct {
				ID     int64  `json:"id"`
				Status string `json:"status"`
			} `json:"results"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 2, resp.Data.Approved)
	require.Equal(t, 0, resp.Data.Failed)
	require.Equal(t, 0, resp.Data.Skipped)
	require.Len(t, resp.Data.Results, 2, "重复 id 只处理一次")

	rec = doApproval(router2, http.MethodPost, "/admin/approvals/batch-approve", `{"ids":[]}`)
	require.Equal(t, http.StatusBadRequest, rec.Code)
	rec = doApproval(router2, http.MethodPost, "/admin/approvals/batch-approve", `{"ids":[1]}`)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"skipped":1`, "已通过的再次批量通过只会被跳过")

	operatorRouter, opSvc := newApprovalTestRouter(t, repo, &approvalTestIdentity{userID: 2, email: "ops@example.com", role: service.RoleOperator, authMethod: "jwt"})
	operatorRouter.POST("/admin/approvals/batch-approve", NewApprovalHandler(opSvc).BatchApprove)
	rec = doApproval(operatorRouter, http.MethodPost, "/admin/approvals/batch-approve", `{"ids":[2]}`)
	require.Equal(t, http.StatusForbidden, rec.Code)
}
