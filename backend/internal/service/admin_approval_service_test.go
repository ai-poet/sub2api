package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// ---------- 桩 ----------

type approvalRepoStub struct {
	mu    sync.Mutex
	seq   int64
	items map[int64]*AdminApprovalRequest
}

func newApprovalRepoStub() *approvalRepoStub {
	return &approvalRepoStub{items: map[int64]*AdminApprovalRequest{}}
}

func (r *approvalRepoStub) Create(_ context.Context, req *AdminApprovalRequest) (*AdminApprovalRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	clone := *req
	clone.ID = r.seq
	r.items[clone.ID] = &clone
	out := clone
	return &out, nil
}

func (r *approvalRepoStub) GetByID(_ context.Context, id int64) (*AdminApprovalRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok {
		return nil, ErrApprovalNotFound
	}
	out := *item
	return &out, nil
}

func (r *approvalRepoStub) List(_ context.Context, f *AdminApprovalFilter) ([]*AdminApprovalRequest, int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*AdminApprovalRequest, 0)
	for _, item := range r.items {
		if f != nil && f.RequesterUserID != nil && item.RequesterUserID != *f.RequesterUserID {
			continue
		}
		clone := *item
		out = append(out, &clone)
	}
	return out, int64(len(out)), nil
}

func (r *approvalRepoStub) CountPending(_ context.Context, now time.Time, requester *int64) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var n int64
	for _, item := range r.items {
		if item.Status == ApprovalStatusPending && item.ExpiresAt.After(now) && (requester == nil || item.RequesterUserID == *requester) {
			n++
		}
	}
	return n, nil
}

func (r *approvalRepoStub) TransitionToExecuting(_ context.Context, id, approverID int64, approverEmail string, now time.Time) (*AdminApprovalRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok {
		return nil, ErrApprovalNotFound
	}
	if item.Status != ApprovalStatusPending || !item.ExpiresAt.After(now) {
		return nil, ErrApprovalNotPending
	}
	item.Status = ApprovalStatusExecuting
	item.DecidedByUserID = &approverID
	item.DecidedByEmail = approverEmail
	item.DecidedAt = &now
	out := *item
	return &out, nil
}

func (r *approvalRepoStub) FinishExecution(_ context.Context, id int64, status string, statusCode int, body, errText string, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok || item.Status != ApprovalStatusExecuting {
		return nil
	}
	item.Status = status
	item.ExecutedAt = &now
	item.ResultStatusCode = &statusCode
	item.ResultBody = body
	item.ResultError = errText
	return nil
}

func (r *approvalRepoStub) Decide(_ context.Context, id int64, toStatus string, actorID int64, actorEmail, reason string, now time.Time) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[id]
	if !ok || item.Status != ApprovalStatusPending || !item.ExpiresAt.After(now) {
		return false, nil
	}
	item.Status = toStatus
	item.DecidedByUserID = &actorID
	item.DecidedByEmail = actorEmail
	item.DecisionReason = reason
	item.DecidedAt = &now
	return true, nil
}

func (r *approvalRepoStub) MarkNotified(_ context.Context, id int64, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if item, ok := r.items[id]; ok {
		item.NotifiedAt = &now
	}
	return nil
}

func (r *approvalRepoStub) ExpirePending(_ context.Context, now time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var n int64
	for _, item := range r.items {
		if item.Status == ApprovalStatusPending && !item.ExpiresAt.After(now) {
			item.Status = ApprovalStatusExpired
			n++
		}
	}
	return n, nil
}

func (r *approvalRepoStub) FailStuckExecuting(_ context.Context, before time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var n int64
	for _, item := range r.items {
		if item.Status == ApprovalStatusExecuting && item.DecidedAt != nil && item.DecidedAt.Before(before) {
			item.Status = ApprovalStatusFailed
			item.ResultError = "execution_timeout"
			n++
		}
	}
	return n, nil
}

type approvalEncryptorStub struct{}

func (approvalEncryptorStub) Encrypt(plaintext string) (string, error) {
	return "enc:" + base64.StdEncoding.EncodeToString([]byte(plaintext)), nil
}

func (approvalEncryptorStub) Decrypt(ciphertext string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(ciphertext, "enc:"))
	return string(raw), err
}

type approvalUserReaderStub map[int64]*User

func (s approvalUserReaderStub) GetByID(_ context.Context, id int64) (*User, error) {
	u, ok := s[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	clone := *u
	return &clone, nil
}

type approvalNotifierStub struct {
	mu    sync.Mutex
	calls []string
}

func (n *approvalNotifierStub) NotifyRequested(req *AdminApprovalRequest, origin string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.calls = append(n.calls, req.Action+"@"+origin)
}

func newApprovalServiceForTest(repo *approvalRepoStub) *AdminApprovalService {
	svc := NewAdminApprovalService(repo, nil, nil, nil, nil, approvalEncryptorStub{})
	svc.users = approvalUserReaderStub{
		1: {ID: 1, Email: "admin@example.com", Role: RoleAdmin, Status: StatusActive},
		2: {ID: 2, Email: "ops@example.com", Role: RoleOperator, Status: StatusActive},
		7: {ID: 7, Email: "target@example.com", Role: RoleUser, Status: StatusActive},
	}
	return svc
}

func balanceCapture(body string) *AdminApprovalCaptureInput {
	return &AdminApprovalCaptureInput{
		Method:          http.MethodPost,
		RouteTemplate:   "/api/v1/admin/users/:id/balance",
		Path:            "/api/v1/admin/users/7/balance",
		RawQuery:        "note=x",
		Params:          map[string]string{"id": "7"},
		ContentType:     "application/json",
		Action:          "admin.users.balance.create",
		Body:            []byte(body),
		RequesterUserID: 2,
		RequesterEmail:  "ops@example.com",
		RequesterIP:     "192.0.2.10",
		RequestOrigin:   "https://console.example.com",
	}
}

var adminActor = ApprovalActor{UserID: 1, Email: "admin@example.com", Role: RoleAdmin, AuthMethod: AuditAuthMethodJWT, SessionID: "sess-1", ClientIP: "10.0.0.1"}

// ---------- Capture ----------

func TestAdminApprovalService_CaptureEncryptsRedactsAndSummarizes(t *testing.T) {
	repo := newApprovalRepoStub()
	svc := newApprovalServiceForTest(repo)
	notifier := &approvalNotifierStub{}
	svc.SetNotifier(notifier)
	now := time.Date(2026, 9, 16, 10, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return now }

	body := `{"balance":10,"operation":"add","password":"s3cret"}`
	created, err := svc.Capture(context.Background(), balanceCapture(body))
	require.NoError(t, err)
	require.Equal(t, ApprovalStatusPending, created.Status)
	require.Equal(t, "admin.users.balance.create", created.Action)
	require.Equal(t, "target@example.com", created.TargetSummary)
	require.Equal(t, ApprovalTargetUser, created.TargetType)
	require.NotNil(t, created.TargetID)
	require.Equal(t, int64(7), *created.TargetID)
	require.Equal(t, now.Add(AdminApprovalTTL), created.ExpiresAt)
	require.Equal(t, "note=x", created.RequestQuery)

	plain, err := approvalEncryptorStub{}.Decrypt(created.RequestBodyEnc)
	require.NoError(t, err)
	require.JSONEq(t, body, plain)
	require.NotContains(t, created.RequestBodyRedacted, "s3cret", "展示副本必须脱敏")
	require.Contains(t, created.RequestBodyRedacted, "balance")
	require.Len(t, created.RequestBodySHA256, 64)
	require.Equal(t, []string{"admin.users.balance.create@https://console.example.com"}, notifier.calls)
}

func TestAdminApprovalService_CaptureTargetSummaries(t *testing.T) {
	svc := newApprovalServiceForTest(newApprovalRepoStub())
	cases := []struct {
		name    string
		in      *AdminApprovalCaptureInput
		typ     string
		summary string
	}{
		{"create user", &AdminApprovalCaptureInput{Method: http.MethodPost, RouteTemplate: "/api/v1/admin/users", Path: "/api/v1/admin/users", ContentType: "application/json", Body: []byte(`{"email":"new@example.com","password":"x"}`), RequesterUserID: 2}, ApprovalTargetUser, "new@example.com"},
		{"batch limits", &AdminApprovalCaptureInput{Method: http.MethodPost, RouteTemplate: "/api/v1/admin/users/batch-limits", Path: "/api/v1/admin/users/batch-limits", ContentType: "application/json", Body: []byte(`{"user_ids":[1,2,3]}`), RequesterUserID: 2}, ApprovalTargetUsersBatch, "3 users"},
		{"batch all", &AdminApprovalCaptureInput{Method: http.MethodPost, RouteTemplate: "/api/v1/admin/users/batch-concurrency", Path: "/api/v1/admin/users/batch-concurrency", ContentType: "application/json", Body: []byte(`{"all":true,"concurrency":2,"mode":"set"}`), RequesterUserID: 2}, ApprovalTargetUsersBatch, "all users"},
		{"assign subscription", &AdminApprovalCaptureInput{Method: http.MethodPost, RouteTemplate: "/api/v1/admin/subscriptions/assign", Path: "/api/v1/admin/subscriptions/assign", ContentType: "application/json", Body: []byte(`{"user_id":7,"group_id":3}`), RequesterUserID: 2}, ApprovalTargetSubscription, "target@example.com · group #3"},
		{"unknown user falls back to id", &AdminApprovalCaptureInput{Method: http.MethodPost, RouteTemplate: "/api/v1/admin/users/:id/balance", Path: "/api/v1/admin/users/99/balance", Params: map[string]string{"id": "99"}, RequesterUserID: 2}, ApprovalTargetUser, "user #99"},
		{"subscription without readers falls back", &AdminApprovalCaptureInput{Method: http.MethodPost, RouteTemplate: "/api/v1/admin/subscriptions/:id/revoke", Path: "/api/v1/admin/subscriptions/5/revoke", Params: map[string]string{"id": "5"}, RequesterUserID: 2}, ApprovalTargetSubscription, "subscription #5"},
		{"api key without reader falls back", &AdminApprovalCaptureInput{Method: http.MethodPut, RouteTemplate: "/api/v1/admin/api-keys/:id", Path: "/api/v1/admin/api-keys/11", Params: map[string]string{"id": "11"}, ContentType: "application/json", Body: []byte(`{"group_id":1}`), RequesterUserID: 2}, ApprovalTargetAPIKey, "api key #11"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			created, err := svc.Capture(context.Background(), tc.in)
			require.NoError(t, err)
			require.Equal(t, tc.typ, created.TargetType)
			require.Equal(t, tc.summary, created.TargetSummary)
		})
	}
}

func TestAdminApprovalService_CaptureRejectsRoleChanges(t *testing.T) {
	svc := newApprovalServiceForTest(newApprovalRepoStub())
	create := func(body string) *AdminApprovalCaptureInput {
		return &AdminApprovalCaptureInput{Method: http.MethodPost, RouteTemplate: "/api/v1/admin/users", Path: "/api/v1/admin/users", ContentType: "application/json", Body: []byte(body), RequesterUserID: 2}
	}
	update := func(id, body string) *AdminApprovalCaptureInput {
		return &AdminApprovalCaptureInput{Method: http.MethodPut, RouteTemplate: "/api/v1/admin/users/:id", Path: "/api/v1/admin/users/" + id, Params: map[string]string{"id": id}, ContentType: "application/json", Body: []byte(body), RequesterUserID: 2}
	}

	_, err := svc.Capture(context.Background(), create(`{"email":"a@b.c","password":"x","role":"operator"}`))
	require.ErrorIs(t, err, ErrApprovalActionForbidden)
	_, err = svc.Capture(context.Background(), create(`{"email":"a@b.c","password":"x","role":"admin"}`))
	require.ErrorIs(t, err, ErrApprovalActionForbidden)
	_, err = svc.Capture(context.Background(), create(`{"email":"a@b.c","password":"x","role":"user"}`))
	require.NoError(t, err)
	_, err = svc.Capture(context.Background(), create(`{"email":"a@b.c","password":"x"}`))
	require.NoError(t, err)

	_, err = svc.Capture(context.Background(), update("7", `{"role":"operator"}`))
	require.ErrorIs(t, err, ErrApprovalActionForbidden, "普通用户改为运维")
	_, err = svc.Capture(context.Background(), update("2", `{"role":"user"}`))
	require.ErrorIs(t, err, ErrApprovalActionForbidden, "运维降为普通用户")
	_, err = svc.Capture(context.Background(), update("7", `{"role":"user","notes":"same role is fine"}`))
	require.NoError(t, err)
	_, err = svc.Capture(context.Background(), update("7", `{"status":"disabled"}`))
	require.NoError(t, err, "不带 role 的编辑可入队")
	_, err = svc.Capture(context.Background(), update("404", `{"role":"user"}`))
	require.NoError(t, err, "目标不存在时入队，重放时 404")

	// 无法核实目标角色时 fail-closed
	svc.users = nil
	_, err = svc.Capture(context.Background(), update("7", `{"role":"user"}`))
	require.ErrorIs(t, err, ErrApprovalActionForbidden)
}

func TestAdminApprovalService_CaptureLimitsAndValidation(t *testing.T) {
	repo := newApprovalRepoStub()
	svc := newApprovalServiceForTest(repo)

	_, err := svc.Capture(context.Background(), balanceCapture(`not json`))
	require.ErrorIs(t, err, ErrApprovalBodyNotJSON)

	in := balanceCapture(`{"balance":1}`)
	in.ContentType = "text/plain"
	_, err = svc.Capture(context.Background(), in)
	require.ErrorIs(t, err, ErrApprovalBodyNotJSON)

	in = balanceCapture(strings.Repeat("x", AuditRequestBodyCaptureLimit+1))
	_, err = svc.Capture(context.Background(), in)
	require.ErrorIs(t, err, ErrApprovalBodyTooLarge)

	_, err = svc.Capture(context.Background(), &AdminApprovalCaptureInput{Method: http.MethodPost, RouteTemplate: "/x"})
	require.Error(t, err, "缺少申请人")

	svc.createLimit = newApprovalCreateLimiter(2, time.Minute)
	for i := 0; i < 2; i++ {
		_, err = svc.Capture(context.Background(), balanceCapture(`{"balance":1,"operation":"add"}`))
		require.NoError(t, err)
	}
	_, err = svc.Capture(context.Background(), balanceCapture(`{"balance":1,"operation":"add"}`))
	require.ErrorIs(t, err, ErrApprovalRateLimited)

	svc.createLimit = newApprovalCreateLimiter(1000, time.Minute)
	for len(repo.items) < AdminApprovalPendingLimitPerUser {
		_, err = svc.Capture(context.Background(), balanceCapture(`{"balance":1,"operation":"add"}`))
		require.NoError(t, err)
	}
	_, err = svc.Capture(context.Background(), balanceCapture(`{"balance":1,"operation":"add"}`))
	require.ErrorIs(t, err, ErrApprovalPendingLimit)

	other := balanceCapture(`{"balance":1,"operation":"add"}`)
	other.RequesterUserID = 3
	_, err = svc.Capture(context.Background(), other)
	require.NoError(t, err, "上限按申请人计")

	empty := NewAdminApprovalService(nil, nil, nil, nil, nil, nil)
	_, err = empty.Capture(context.Background(), balanceCapture(`{}`))
	require.ErrorIs(t, err, ErrApprovalGateUnavailable)
}

// ---------- Approve / Reject / Cancel ----------

type capturedReplay struct {
	req    *http.Request
	body   []byte
	replay *ApprovalReplay
}

func replayDispatcher(status int, respBody string, sink *capturedReplay) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 0)
		if r.Body != nil {
			b, _ := readAllForTest(r)
			buf = b
		}
		sink.req = r
		sink.body = buf
		sink.replay, _ = ApprovalReplayFromContext(r.Context())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(respBody))
	})
}

func readAllForTest(r *http.Request) ([]byte, error) {
	var out []byte
	buf := make([]byte, 4096)
	for {
		n, err := r.Body.Read(buf)
		out = append(out, buf[:n]...)
		if err != nil {
			break
		}
	}
	return out, nil
}

func TestAdminApprovalService_ApproveReplaysAsAdmin(t *testing.T) {
	repo := newApprovalRepoStub()
	svc := newApprovalServiceForTest(repo)
	sink := &capturedReplay{}
	svc.SetDispatcher(replayDispatcher(http.StatusOK, `{"code":0,"data":{"ok":true}}`, sink))

	created, err := svc.Capture(context.Background(), balanceCapture(`{"balance":10,"operation":"add"}`))
	require.NoError(t, err)

	updated, result, err := svc.Approve(context.Background(), created.ID, adminActor)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, result.StatusCode)
	require.Equal(t, ApprovalStatusApproved, updated.Status)
	require.NotNil(t, updated.DecidedByUserID)
	require.Equal(t, int64(1), *updated.DecidedByUserID)
	require.Equal(t, "admin@example.com", updated.DecidedByEmail)
	require.Empty(t, updated.ResultError)
	require.Contains(t, updated.ResultBody, `"ok":true`)

	require.NotNil(t, sink.req)
	require.Equal(t, http.MethodPost, sink.req.Method)
	require.Equal(t, "/api/v1/admin/users/7/balance", sink.req.URL.Path)
	require.Equal(t, "note=x", sink.req.URL.RawQuery)
	require.Equal(t, "application/json", sink.req.Header.Get("Content-Type"))
	require.Equal(t, "approval-1", sink.req.Header.Get("Idempotency-Key"))
	require.Equal(t, "1", sink.req.Header.Get("X-Admin-UI-Request"))
	require.Equal(t, "10.0.0.1:0", sink.req.RemoteAddr)
	require.JSONEq(t, `{"balance":10,"operation":"add"}`, string(sink.body))
	require.NotNil(t, sink.replay)
	require.Equal(t, int64(1), sink.replay.ApprovalID)
	require.Equal(t, int64(2), sink.replay.RequesterUserID)
	require.Equal(t, int64(1), sink.replay.ApproverUserID)
	require.Equal(t, RoleAdmin, sink.replay.ApproverRole)
	require.Equal(t, "sess-1", sink.replay.ApproverSessionID)

	_, _, err = svc.Approve(context.Background(), created.ID, adminActor)
	require.ErrorIs(t, err, ErrApprovalNotPending, "重复通过")
	_, _, err = svc.Approve(context.Background(), 999, adminActor)
	require.ErrorIs(t, err, ErrApprovalNotFound)
}

func TestAdminApprovalService_ApproveRecordsFailures(t *testing.T) {
	repo := newApprovalRepoStub()
	svc := newApprovalServiceForTest(repo)
	svc.SetDispatcher(replayDispatcher(http.StatusForbidden, `{"code":"STEP_UP_REQUIRED","message":"step up"}`, &capturedReplay{}))

	created, err := svc.Capture(context.Background(), balanceCapture(`{"balance":10,"operation":"add"}`))
	require.NoError(t, err)
	updated, result, err := svc.Approve(context.Background(), created.ID, adminActor)
	require.NoError(t, err)
	require.Equal(t, http.StatusForbidden, result.StatusCode)
	require.Equal(t, ApprovalStatusFailed, updated.Status)
	require.Equal(t, "STEP_UP_REQUIRED", updated.ResultError)

	svc.SetDispatcher(replayDispatcher(http.StatusInternalServerError, `oops`, &capturedReplay{}))
	created, err = svc.Capture(context.Background(), balanceCapture(`{"balance":10,"operation":"add"}`))
	require.NoError(t, err)
	updated, _, err = svc.Approve(context.Background(), created.ID, adminActor)
	require.NoError(t, err)
	require.Equal(t, ApprovalStatusFailed, updated.Status)
	require.Equal(t, "http_500", updated.ResultError)

	// panic 的 handler 也只会让这一条变 failed
	svc.SetDispatcher(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }))
	created, err = svc.Capture(context.Background(), balanceCapture(`{"balance":10,"operation":"add"}`))
	require.NoError(t, err)
	updated, result, err = svc.Approve(context.Background(), created.ID, adminActor)
	require.NoError(t, err)
	require.Equal(t, http.StatusInternalServerError, result.StatusCode)
	require.Equal(t, ApprovalStatusFailed, updated.Status)
}

func TestAdminApprovalService_ApproveGuards(t *testing.T) {
	repo := newApprovalRepoStub()
	svc := newApprovalServiceForTest(repo)
	created, err := svc.Capture(context.Background(), balanceCapture(`{"balance":10,"operation":"add"}`))
	require.NoError(t, err)

	_, _, err = svc.Approve(context.Background(), created.ID, adminActor)
	require.ErrorIs(t, err, ErrApprovalDispatcherMissing, "未注入引擎不能通过")

	svc.SetDispatcher(replayDispatcher(http.StatusOK, `{}`, &capturedReplay{}))
	apiKeyActor := adminActor
	apiKeyActor.AuthMethod = AuditAuthMethodAdminAPIKey
	_, _, err = svc.Approve(context.Background(), created.ID, apiKeyActor)
	require.ErrorIs(t, err, ErrApprovalApproverInvalid, "admin API key 会话不能审批")

	operatorActor := ApprovalActor{UserID: 2, Role: RoleOperator, AuthMethod: AuditAuthMethodJWT}
	_, _, err = svc.Approve(context.Background(), created.ID, operatorActor)
	require.ErrorIs(t, err, ErrApprovalApproverInvalid)

	spoofed := ApprovalActor{UserID: 2, Role: RoleAdmin, AuthMethod: AuditAuthMethodJWT}
	_, _, err = svc.Approve(context.Background(), created.ID, spoofed)
	require.ErrorIs(t, err, ErrApprovalApproverInvalid, "以库里的角色为准，而不是上下文声称的角色")

	stored := repo.items[created.ID]
	require.Equal(t, ApprovalStatusPending, stored.Status, "被拒绝的审批人不应改变申请状态")

	// 过期的申请不能通过
	stored.ExpiresAt = time.Now().Add(-time.Minute)
	_, _, err = svc.Approve(context.Background(), created.ID, adminActor)
	require.ErrorIs(t, err, ErrApprovalNotPending)
}

func TestAdminApprovalService_RejectCancelAndVisibility(t *testing.T) {
	repo := newApprovalRepoStub()
	svc := newApprovalServiceForTest(repo)
	first, err := svc.Capture(context.Background(), balanceCapture(`{"balance":1,"operation":"add"}`))
	require.NoError(t, err)
	second, err := svc.Capture(context.Background(), balanceCapture(`{"balance":2,"operation":"add"}`))
	require.NoError(t, err)

	rejected, err := svc.Reject(context.Background(), first.ID, adminActor, "  not now  ")
	require.NoError(t, err)
	require.Equal(t, ApprovalStatusRejected, rejected.Status)
	require.Equal(t, "not now", rejected.DecisionReason)
	_, err = svc.Reject(context.Background(), first.ID, adminActor, "again")
	require.ErrorIs(t, err, ErrApprovalNotPending)
	_, err = svc.Reject(context.Background(), second.ID, ApprovalActor{UserID: 2, Role: RoleOperator, AuthMethod: AuditAuthMethodJWT}, "")
	require.ErrorIs(t, err, ErrApprovalApproverInvalid)

	stranger := ApprovalActor{UserID: 3, Role: RoleOperator}
	_, err = svc.Cancel(context.Background(), second.ID, stranger)
	require.ErrorIs(t, err, ErrApprovalNotFound, "他人的申请对 operator 不可见")
	_, err = svc.Get(context.Background(), second.ID, stranger)
	require.ErrorIs(t, err, ErrApprovalNotFound)

	owner := ApprovalActor{UserID: 2, Email: "ops@example.com", Role: RoleOperator}
	got, err := svc.Get(context.Background(), second.ID, owner)
	require.NoError(t, err)
	require.Equal(t, second.ID, got.ID)
	cancelled, err := svc.Cancel(context.Background(), second.ID, owner)
	require.NoError(t, err)
	require.Equal(t, ApprovalStatusCancelled, cancelled.Status)
	_, err = svc.Cancel(context.Background(), second.ID, owner)
	require.ErrorIs(t, err, ErrApprovalNotPending)
	_, err = svc.Cancel(context.Background(), 404, owner)
	require.ErrorIs(t, err, ErrApprovalNotFound)
}

func TestAdminApprovalService_ListPendingCountAndSweep(t *testing.T) {
	repo := newApprovalRepoStub()
	svc := newApprovalServiceForTest(repo)
	base := time.Date(2026, 9, 16, 10, 0, 0, 0, time.UTC)
	svc.now = func() time.Time { return base }

	live, err := svc.Capture(context.Background(), balanceCapture(`{"balance":1,"operation":"add"}`))
	require.NoError(t, err)
	stale, err := svc.Capture(context.Background(), balanceCapture(`{"balance":2,"operation":"add"}`))
	require.NoError(t, err)
	repo.items[stale.ID].ExpiresAt = base.Add(-time.Hour)
	stuckAt := base.Add(-time.Hour)
	stuck, err := svc.Capture(context.Background(), balanceCapture(`{"balance":3,"operation":"add"}`))
	require.NoError(t, err)
	repo.items[stuck.ID].Status = ApprovalStatusExecuting
	repo.items[stuck.ID].DecidedAt = &stuckAt

	count, err := svc.PendingCount(context.Background(), nil)
	require.NoError(t, err)
	require.Equal(t, int64(1), count, "过期的不计入待审")

	list, err := svc.List(context.Background(), &AdminApprovalFilter{PageSize: 500})
	require.NoError(t, err)
	statuses := map[int64]string{}
	for _, item := range list.Items {
		statuses[item.ID] = item.Status
	}
	require.Equal(t, ApprovalStatusPending, statuses[live.ID])
	require.Equal(t, ApprovalStatusExpired, statuses[stale.ID], "过期的 pending 展示为 expired")
	require.Equal(t, 100, list.PageSize, "page_size 上限 100")

	expired, failed, err := svc.SweepOnce(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(1), expired)
	require.Equal(t, int64(1), failed)
	require.Equal(t, ApprovalStatusExpired, repo.items[stale.ID].Status)
	require.Equal(t, ApprovalStatusFailed, repo.items[stuck.ID].Status)
	require.Equal(t, "execution_timeout", repo.items[stuck.ID].ResultError)
}

func TestApprovalReplayContextRoundTrip(t *testing.T) {
	_, ok := ApprovalReplayFromContext(context.Background())
	require.False(t, ok)
	ctx := WithApprovalReplay(context.Background(), &ApprovalReplay{ApprovalID: 9, ApproverUserID: 1, ApproverRole: RoleAdmin})
	replay, ok := ApprovalReplayFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, int64(9), replay.ApprovalID)
	require.Equal(t, ctx, WithApprovalReplay(ctx, nil), "nil 标记不改变 ctx")
}

func TestExtractApprovalErrorCode(t *testing.T) {
	require.Equal(t, "STEP_UP_REQUIRED", extractApprovalErrorCode(`{"code":"STEP_UP_REQUIRED"}`))
	require.Equal(t, "ADMIN_ALREADY_EXISTS", extractApprovalErrorCode(`{"reason":"ADMIN_ALREADY_EXISTS","message":"x"}`))
	require.Equal(t, "code_40001", extractApprovalErrorCode(`{"code":40001,"message":"x"}`))
	require.Equal(t, "", extractApprovalErrorCode(`not json`))
	require.Equal(t, "", extractApprovalErrorCode(``))

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(`{"code":0}`), &envelope))
	require.Equal(t, "", extractApprovalErrorCode(`{"code":0}`))
	require.False(t, errors.Is(ErrApprovalNotPending, ErrApprovalNotFound))
}

func TestAdminApprovalService_ApproveBatch(t *testing.T) {
	repo := newApprovalRepoStub()
	svc := newApprovalServiceForTest(repo)
	svc.SetDispatcher(replayDispatcher(http.StatusOK, `{"code":0}`, &capturedReplay{}))

	first, err := svc.Capture(context.Background(), balanceCapture(`{"balance":1,"operation":"add"}`))
	require.NoError(t, err)
	second, err := svc.Capture(context.Background(), balanceCapture(`{"balance":2,"operation":"add"}`))
	require.NoError(t, err)
	third, err := svc.Capture(context.Background(), balanceCapture(`{"balance":3,"operation":"add"}`))
	require.NoError(t, err)
	_, err = svc.Reject(context.Background(), second.ID, adminActor, "no")
	require.NoError(t, err)

	results, err := svc.ApproveBatch(context.Background(), []int64{first.ID, second.ID, third.ID, third.ID, 999, 0}, adminActor)
	require.NoError(t, err)
	require.Len(t, results, 4, "去重且忽略非法 id")
	byID := map[int64]ApprovalBatchItem{}
	for _, item := range results {
		byID[item.ID] = item
	}
	require.Equal(t, ApprovalStatusApproved, byID[first.ID].Status)
	require.Equal(t, http.StatusOK, byID[first.ID].StatusCode)
	require.Equal(t, ApprovalBatchStatusSkipped, byID[second.ID].Status)
	require.Equal(t, "APPROVAL_NOT_PENDING", byID[second.ID].Error)
	require.Equal(t, ApprovalStatusApproved, byID[third.ID].Status)
	require.Equal(t, ApprovalBatchStatusSkipped, byID[999].Status)
	require.Equal(t, "APPROVAL_NOT_FOUND", byID[999].Error)

	_, err = svc.ApproveBatch(context.Background(), []int64{0, -1}, adminActor)
	require.ErrorIs(t, err, ErrApprovalBatchEmpty)
	tooMany := make([]int64, AdminApprovalBatchLimit+1)
	for i := range tooMany {
		tooMany[i] = int64(i + 1)
	}
	_, err = svc.ApproveBatch(context.Background(), tooMany, adminActor)
	require.ErrorIs(t, err, ErrApprovalBatchTooLarge)
	_, err = svc.ApproveBatch(context.Background(), []int64{first.ID}, ApprovalActor{UserID: 2, Role: RoleOperator, AuthMethod: AuditAuthMethodJWT})
	require.ErrorIs(t, err, ErrApprovalApproverInvalid)
}
