//go:build unit

package middleware

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// fakeApprovalGate 记录捕获入参，并按需返回预设错误。
type fakeApprovalGate struct {
	mu     sync.Mutex
	inputs []*service.AdminApprovalCaptureInput
	err    error
}

func (g *fakeApprovalGate) Capture(_ context.Context, in *service.AdminApprovalCaptureInput) (*service.AdminApprovalRequest, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.inputs = append(g.inputs, in)
	if g.err != nil {
		return nil, g.err
	}
	return &service.AdminApprovalRequest{
		ID:            42,
		Status:        service.ApprovalStatusPending,
		Action:        in.Action,
		TargetSummary: "user@example.com",
		ExpiresAt:     time.Date(2026, 9, 20, 0, 0, 0, 0, time.UTC),
	}, nil
}

func (g *fakeApprovalGate) last() *service.AdminApprovalCaptureInput {
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.inputs) == 0 {
		return nil
	}
	return g.inputs[len(g.inputs)-1]
}

type approvalAuthFixture struct {
	router    *gin.Engine
	auth      *service.AuthService
	audit     *service.AuditLogService
	auditRepo *capturingAuditRepo
	users     map[int64]*service.User
	gate      *fakeApprovalGate
}

func newApprovalAuthFixture(t *testing.T, gate service.AdminApprovalGate) *approvalAuthFixture {
	t.Helper()
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{JWT: config.JWTConfig{Secret: "test-secret", ExpireHour: 1}}
	authService := service.NewAuthService(nil, nil, nil, nil, cfg, nil, nil, nil, nil, nil, nil, nil, nil)
	users := map[int64]*service.User{
		1: {ID: 1, Email: "admin@example.com", Role: service.RoleAdmin, Status: service.StatusActive, TokenVersion: 1, Concurrency: 3},
		2: {ID: 2, Email: "ops@example.com", Role: service.RoleOperator, Status: service.StatusActive, TokenVersion: 1, Concurrency: 1},
	}
	userRepo := &stubUserRepo{
		getByID: func(_ context.Context, id int64) (*service.User, error) {
			u, ok := users[id]
			if !ok {
				return nil, service.ErrUserNotFound
			}
			clone := *u
			return &clone, nil
		},
	}
	userService := service.NewUserService(userRepo, nil, nil, nil)
	auditRepo := &capturingAuditRepo{}
	auditService := service.NewAuditLogService(auditRepo, nil)
	auditService.Start()
	t.Cleanup(auditService.Stop)

	router := gin.New()
	router.Use(gin.HandlerFunc(NewAdminAuthMiddlewareWithApprovalGate(authService, userService, nil, auditService, gate)))
	echo := func(c *gin.Context) {
		subject, _ := GetAuthSubjectFromContext(c)
		role, _ := GetUserRoleFromContext(c)
		c.JSON(http.StatusOK, gin.H{"actor": subject.UserID, "role": role, "auth_method": c.GetString("auth_method")})
	}
	router.GET("/api/v1/admin/users", echo)
	router.POST("/api/v1/admin/users", echo)
	router.PUT("/api/v1/admin/users/:id", echo)
	router.DELETE("/api/v1/admin/users/:id", echo)
	router.POST("/api/v1/admin/users/:id/balance", echo)
	router.POST("/api/v1/admin/subscriptions/assign", echo)
	router.POST("/api/v1/admin/user-attributes/batch", echo)
	router.POST("/api/v1/admin/approvals/:id/approve", echo)

	f := &approvalAuthFixture{router: router, auth: authService, audit: auditService, auditRepo: auditRepo, users: users}
	if g, ok := gate.(*fakeApprovalGate); ok {
		f.gate = g
	}
	return f
}

func (f *approvalAuthFixture) tokenFor(t *testing.T, userID int64) string {
	t.Helper()
	u := f.users[userID]
	token, err := f.auth.GenerateToken(context.Background(), &service.User{ID: u.ID, Email: u.Email, Role: u.Role, TokenVersion: u.TokenVersion})
	require.NoError(t, err)
	return token
}

func (f *approvalAuthFixture) do(t *testing.T, userID int64, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+f.tokenFor(t, userID))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	f.router.ServeHTTP(w, req)
	return w
}

func TestAdminAuthOperatorApprovalCapture(t *testing.T) {
	gate := &fakeApprovalGate{}
	f := newApprovalAuthFixture(t, gate)

	t.Run("write_in_approval_scope_is_queued_with_202", func(t *testing.T) {
		w := f.do(t, 2, http.MethodPost, "/api/v1/admin/users/7/balance?note=x", `{"balance":10,"operation":"add","password":"p@ss"}`)
		require.Equal(t, http.StatusAccepted, w.Code)
		var resp struct {
			Code int `json:"code"`
			Data struct {
				ApprovalRequestID int64  `json:"approval_request_id"`
				Status            string `json:"status"`
				Action            string `json:"action"`
				TargetSummary     string `json:"target_summary"`
			} `json:"data"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		require.Equal(t, 0, resp.Code)
		require.Equal(t, int64(42), resp.Data.ApprovalRequestID)
		require.Equal(t, service.ApprovalStatusPending, resp.Data.Status)
		require.Equal(t, "admin.users.balance.create", resp.Data.Action)

		in := gate.last()
		require.NotNil(t, in)
		require.Equal(t, http.MethodPost, in.Method)
		require.Equal(t, "/api/v1/admin/users/:id/balance", in.RouteTemplate)
		require.Equal(t, "/api/v1/admin/users/7/balance", in.Path)
		require.Equal(t, "note=x", in.RawQuery)
		require.Equal(t, "7", in.Params["id"])
		require.JSONEq(t, `{"balance":10,"operation":"add","password":"p@ss"}`, string(in.Body))
		require.Equal(t, int64(2), in.RequesterUserID)
		require.Equal(t, "ops@example.com", in.RequesterEmail)
		require.Equal(t, "admin.users.balance.create", in.Action)
		require.NotEmpty(t, in.RequestOrigin)
	})

	t.Run("delete_user_is_refused_without_queueing", func(t *testing.T) {
		before := len(gate.inputs)
		w := f.do(t, 2, http.MethodDelete, "/api/v1/admin/users/7", "")
		require.Equal(t, http.StatusForbidden, w.Code)
		require.Contains(t, w.Body.String(), "OPERATOR_ACTION_FORBIDDEN")
		require.Len(t, gate.inputs, before, "refused actions never reach the gate")
	})

	t.Run("gate_errors_are_mapped_to_http", func(t *testing.T) {
		gate.err = service.ErrApprovalActionForbidden
		w := f.do(t, 2, http.MethodPut, "/api/v1/admin/users/7", `{"role":"operator"}`)
		require.Equal(t, http.StatusForbidden, w.Code)
		require.Contains(t, w.Body.String(), "OPERATOR_ACTION_FORBIDDEN")

		gate.err = service.ErrApprovalPendingLimit
		w = f.do(t, 2, http.MethodPost, "/api/v1/admin/subscriptions/assign", `{"user_id":1,"group_id":2}`)
		require.Equal(t, http.StatusConflict, w.Code)
		require.Contains(t, w.Body.String(), "APPROVAL_PENDING_LIMIT")

		gate.err = errors.New("db down")
		w = f.do(t, 2, http.MethodPost, "/api/v1/admin/users/7/balance", `{"balance":1,"operation":"add"}`)
		require.Equal(t, http.StatusInternalServerError, w.Code)
		require.Contains(t, w.Body.String(), "APPROVAL_CAPTURE_FAILED")
		require.NotContains(t, w.Body.String(), "db down", "internal errors must not leak")
		gate.err = nil
	})

	t.Run("read_shaped_post_and_reads_pass_through", func(t *testing.T) {
		w := f.do(t, 2, http.MethodPost, "/api/v1/admin/user-attributes/batch", `{"user_ids":[1]}`)
		require.Equal(t, http.StatusOK, w.Code)
		require.Contains(t, w.Body.String(), `"role":"operator"`)
		w = f.do(t, 2, http.MethodGet, "/api/v1/admin/users", "")
		require.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("approve_stays_admin_only", func(t *testing.T) {
		w := f.do(t, 2, http.MethodPost, "/api/v1/admin/approvals/42/approve", "")
		require.Equal(t, http.StatusForbidden, w.Code)
		require.Contains(t, w.Body.String(), `"FORBIDDEN"`)
	})

	t.Run("admin_writes_execute_directly", func(t *testing.T) {
		before := len(gate.inputs)
		w := f.do(t, 1, http.MethodPost, "/api/v1/admin/users/7/balance", `{"balance":1,"operation":"add"}`)
		require.Equal(t, http.StatusOK, w.Code)
		require.Contains(t, w.Body.String(), `"role":"admin"`)
		require.Len(t, gate.inputs, before)
	})

	f.audit.Stop()
	var requested, refused int
	for _, entry := range f.auditRepo.snapshot() {
		switch entry.Action {
		case service.AuditActionAdminApprovalRequested:
			requested++
			require.Equal(t, http.StatusAccepted, entry.StatusCode)
			require.Equal(t, "POST /api/v1/admin/users/:id/balance", entry.Method+" "+entry.Path)
			require.Contains(t, entry.RequestBody, `"balance"`)
			require.NotContains(t, entry.RequestBody, "p@ss", "audit body must be redacted")
			require.Equal(t, int64(42), entry.Extra["approval_id"])
		case service.AuditActionAdminApprovalRefused:
			refused++
			require.Equal(t, service.RoleOperator, entry.ActorRole)
		}
	}
	require.Equal(t, 1, requested)
	// 1 删除用户 + 3 次 gate 错误
	require.Equal(t, 4, refused)
}

func TestAdminAuthApprovalReplayMarker(t *testing.T) {
	f := newApprovalAuthFixture(t, &fakeApprovalGate{})

	t.Run("replay_context_authenticates_as_approver_without_token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/7/balance", strings.NewReader(`{"balance":1,"operation":"add"}`))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(service.WithApprovalReplay(req.Context(), &service.ApprovalReplay{
			ApprovalID: 42, RequesterUserID: 2, ApproverUserID: 1, ApproverRole: service.RoleAdmin, ApproverEmail: "admin@example.com",
		}))
		w := httptest.NewRecorder()
		f.router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)
		require.Contains(t, w.Body.String(), `"actor":1`)
		require.Contains(t, w.Body.String(), `"role":"admin"`)
		require.Contains(t, w.Body.String(), `"auth_method":"approval_replay"`)
	})

	t.Run("replay_marker_with_non_admin_role_is_rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/7/balance", nil)
		req = req.WithContext(service.WithApprovalReplay(req.Context(), &service.ApprovalReplay{ApprovalID: 1, ApproverUserID: 2, ApproverRole: service.RoleOperator}))
		w := httptest.NewRecorder()
		f.router.ServeHTTP(w, req)
		require.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("headers_cannot_forge_the_marker", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/7/balance", nil)
		req.Header.Set("X-Approval-Replay", "1")
		req.Header.Set("X-Approval-Id", "42")
		w := httptest.NewRecorder()
		f.router.ServeHTTP(w, req)
		require.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// ---------- 端到端：真实审批服务 + 内存仓储 + 本引擎作为重放目标 ----------

type memApprovalRepo struct {
	mu    sync.Mutex
	seq   int64
	items map[int64]*service.AdminApprovalRequest
}

func newMemApprovalRepo() *memApprovalRepo {
	return &memApprovalRepo{items: map[int64]*service.AdminApprovalRequest{}}
}

func (r *memApprovalRepo) Create(_ context.Context, req *service.AdminApprovalRequest) (*service.AdminApprovalRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	clone := *req
	clone.ID = r.seq
	clone.CreatedAt = time.Now()
	clone.UpdatedAt = clone.CreatedAt
	r.items[clone.ID] = &clone
	out := clone
	return &out, nil
}

func (r *memApprovalRepo) GetByID(_ context.Context, id int64) (*service.AdminApprovalRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok {
		return nil, service.ErrApprovalNotFound
	}
	out := *item
	return &out, nil
}

func (r *memApprovalRepo) List(_ context.Context, _ *service.AdminApprovalFilter) ([]*service.AdminApprovalRequest, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*service.AdminApprovalRequest, 0, len(r.items))
	for _, item := range r.items {
		clone := *item
		out = append(out, &clone)
	}
	return out, int64(len(out)), nil
}

func (r *memApprovalRepo) CountPending(_ context.Context, now time.Time, requester *int64) (int64, error) {
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

func (r *memApprovalRepo) TransitionToExecuting(_ context.Context, id, approverID int64, approverEmail string, now time.Time) (*service.AdminApprovalRequest, error) {
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

func (r *memApprovalRepo) FinishExecution(_ context.Context, id int64, status string, statusCode int, body, errText string, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok || item.Status != service.ApprovalStatusExecuting {
		return nil
	}
	item.Status = status
	item.ExecutedAt = &now
	item.ResultStatusCode = &statusCode
	item.ResultBody = body
	item.ResultError = errText
	return nil
}

func (r *memApprovalRepo) Decide(_ context.Context, id int64, toStatus string, actorID int64, actorEmail, reason string, now time.Time) (bool, error) {
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

func (r *memApprovalRepo) MarkNotified(_ context.Context, id int64, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if item, ok := r.items[id]; ok {
		item.NotifiedAt = &now
	}
	return nil
}

func (r *memApprovalRepo) ExpirePending(_ context.Context, now time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var n int64
	for _, item := range r.items {
		if item.Status == service.ApprovalStatusPending && !item.ExpiresAt.After(now) {
			item.Status = service.ApprovalStatusExpired
			n++
		}
	}
	return n, nil
}

func (r *memApprovalRepo) FailStuckExecuting(_ context.Context, before time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var n int64
	for _, item := range r.items {
		if item.Status == service.ApprovalStatusExecuting && item.DecidedAt != nil && item.DecidedAt.Before(before) {
			item.Status = service.ApprovalStatusFailed
			item.ResultError = "execution_timeout"
			n++
		}
	}
	return n, nil
}

type base64Encryptor struct{}

func (base64Encryptor) Encrypt(plaintext string) (string, error) {
	return "enc:" + base64.StdEncoding.EncodeToString([]byte(plaintext)), nil
}

func (base64Encryptor) Decrypt(ciphertext string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(ciphertext, "enc:"))
	return string(raw), err
}

func TestAdminAuthApprovalReplayEndToEnd(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{JWT: config.JWTConfig{Secret: "test-secret", ExpireHour: 1}}
	authService := service.NewAuthService(nil, nil, nil, nil, cfg, nil, nil, nil, nil, nil, nil, nil, nil)
	users := map[int64]*service.User{
		1: {ID: 1, Email: "admin@example.com", Role: service.RoleAdmin, Status: service.StatusActive, TokenVersion: 1, Concurrency: 3},
		2: {ID: 2, Email: "ops@example.com", Role: service.RoleOperator, Status: service.StatusActive, TokenVersion: 1, Concurrency: 1},
		7: {ID: 7, Email: "target@example.com", Role: service.RoleUser, Status: service.StatusActive, TokenVersion: 1},
	}
	userRepo := &stubUserRepo{getByID: func(_ context.Context, id int64) (*service.User, error) {
		u, ok := users[id]
		if !ok {
			return nil, service.ErrUserNotFound
		}
		clone := *u
		return &clone, nil
	}}
	userService := service.NewUserService(userRepo, nil, nil, nil)
	auditRepo := &capturingAuditRepo{}
	auditService := service.NewAuditLogService(auditRepo, nil)
	auditService.Start()
	t.Cleanup(auditService.Stop)

	repo := newMemApprovalRepo()
	svc := service.NewAdminApprovalService(repo, userService, nil, nil, nil, base64Encryptor{})

	router := gin.New()
	router.Use(gin.HandlerFunc(NewAdminAuthMiddlewareWithApprovalGate(authService, userService, nil, auditService, svc)))
	router.Use(gin.HandlerFunc(NewAuditLogMiddleware(auditService)))
	router.POST("/api/v1/admin/users/:id/balance", func(c *gin.Context) {
		subject, _ := GetAuthSubjectFromContext(c)
		role, _ := GetUserRoleFromContext(c)
		var body map[string]any
		require.NoError(t, c.ShouldBindJSON(&body))
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{
			"actor":           subject.UserID,
			"role":            role,
			"auth_method":     c.GetString("auth_method"),
			"idempotency_key": c.GetHeader("Idempotency-Key"),
			"user":            c.Param("id"),
			"query":           c.Query("note"),
			"body":            body,
		}})
	})
	svc.SetDispatcher(router)

	token, err := authService.GenerateToken(context.Background(), &service.User{ID: 2, Email: "ops@example.com", Role: service.RoleOperator, TokenVersion: 1})
	require.NoError(t, err)

	// 1. operator 发起 → 202，原请求被加密入队
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/7/balance?note=bonus", strings.NewReader(`{"balance":10,"operation":"add","notes":"first"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, http.StatusAccepted, w.Code, w.Body.String())

	var queued struct {
		Data struct {
			ApprovalRequestID int64  `json:"approval_request_id"`
			TargetSummary     string `json:"target_summary"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &queued))
	require.Equal(t, int64(1), queued.Data.ApprovalRequestID)
	require.Equal(t, "target@example.com", queued.Data.TargetSummary)
	stored, err := repo.GetByID(context.Background(), 1)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(stored.RequestBodyEnc, "enc:"))
	require.NotContains(t, stored.RequestBodyRedacted, "enc:")

	// 2. 管理员一键通过 → 以管理员身份重放，handler 看到的是 admin
	updated, replay, err := svc.Approve(context.Background(), 1, service.ApprovalActor{
		UserID: 1, Concurrency: 3, Email: "admin@example.com", Role: service.RoleAdmin, SessionID: "sess-1", AuthMethod: service.AuditAuthMethodJWT, ClientIP: "10.0.0.9",
	})
	require.NoError(t, err)
	require.NotNil(t, replay)
	require.Equal(t, http.StatusOK, replay.StatusCode, replay.Body)
	require.Equal(t, service.ApprovalStatusApproved, updated.Status)
	require.NotNil(t, updated.ResultStatusCode)
	require.Equal(t, http.StatusOK, *updated.ResultStatusCode)

	var replayed struct {
		Data struct {
			Actor          int64          `json:"actor"`
			Role           string         `json:"role"`
			AuthMethod     string         `json:"auth_method"`
			IdempotencyKey string         `json:"idempotency_key"`
			User           string         `json:"user"`
			Query          string         `json:"query"`
			Body           map[string]any `json:"body"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal([]byte(updated.ResultBody), &replayed))
	require.Equal(t, int64(1), replayed.Data.Actor)
	require.Equal(t, service.RoleAdmin, replayed.Data.Role)
	require.Equal(t, service.AuditAuthMethodApprovalReplay, replayed.Data.AuthMethod)
	require.Equal(t, "approval-1", replayed.Data.IdempotencyKey)
	require.Equal(t, "7", replayed.Data.User)
	require.Equal(t, "bonus", replayed.Data.Query)
	require.Equal(t, float64(10), replayed.Data.Body["balance"])

	// 3. 重复通过 → 409
	_, _, err = svc.Approve(context.Background(), 1, service.ApprovalActor{UserID: 1, Role: service.RoleAdmin, AuthMethod: service.AuditAuthMethodJWT})
	require.ErrorIs(t, err, service.ErrApprovalNotPending)

	// 4. 审计：申请入队一条（operator）、重放一条（admin，auth_method=approval_replay，带 approval_id）
	auditService.Stop()
	var requested, replayedAudit int
	for _, entry := range auditRepo.snapshot() {
		switch entry.Action {
		case service.AuditActionAdminApprovalRequested:
			requested++
			require.Equal(t, service.RoleOperator, entry.ActorRole)
			require.Equal(t, int64(1), entry.Extra["approval_id"])
		case "admin.users.balance.create":
			replayedAudit++
			require.Equal(t, service.RoleAdmin, entry.ActorRole)
			require.Equal(t, "admin@example.com", entry.ActorEmail)
			require.Equal(t, service.AuditAuthMethodApprovalReplay, entry.AuthMethod)
			require.Equal(t, http.StatusOK, entry.StatusCode)
			require.EqualValues(t, 1, entry.Extra["approval_id"])
			require.EqualValues(t, 2, entry.Extra["approval_requester_id"])
		}
	}
	require.Equal(t, 1, requested)
	require.Equal(t, 1, replayedAudit)
}
