package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// AdminApprovalGate 认证层捕获 operator 写请求时调用的入口。
// 用接口注入到 adminAuth，便于在中间件单测里用桩替换；未注入（nil）时认证层 fail-closed（503）。
type AdminApprovalGate interface {
	Capture(ctx context.Context, in *AdminApprovalCaptureInput) (*AdminApprovalRequest, error)
}

// AdminApprovalCaptureInput 认证层交给审批服务的原始请求快照。
type AdminApprovalCaptureInput struct {
	Method        string
	RouteTemplate string
	Path          string
	RawQuery      string
	Params        map[string]string
	ContentType   string
	Action        string
	Body          []byte

	RequesterUserID int64
	RequesterEmail  string
	RequesterIP     string
	RequestID       string
	// RequestOrigin 捕获时请求的 scheme://host，站点未配置 frontend_url 时用于拼推送链接。
	RequestOrigin string
}

// ApprovalActor 审批 / 拒绝 / 撤回的操作者（从 gin context 组装）。
type ApprovalActor struct {
	UserID         int64
	Concurrency    int
	Email          string
	Role           string
	SessionID      string
	AuthMethod     string
	ClientIP       string
	RequestID      string
	AcceptLanguage string
}

// ApprovalReplayResult 重放结果摘要。
type ApprovalReplayResult struct {
	StatusCode int
	Body       string
	DurationMs int64
}

// 角色变更检查与目标摘要只需要少量读接口；用小接口便于单测打桩。
type approvalUserReader interface {
	GetByID(ctx context.Context, id int64) (*User, error)
}

type approvalSubscriptionReader interface {
	GetByID(ctx context.Context, id int64) (*UserSubscription, error)
}

type approvalGroupReader interface {
	GetByID(ctx context.Context, id int64) (*Group, error)
}

type approvalAPIKeyReader interface {
	GetByID(ctx context.Context, id int64) (*APIKey, error)
}

// AdminApprovalService 审批申请的全部流程：捕获入队、列表、通过（重放）、拒绝、撤回、过期回收。
type AdminApprovalService struct {
	repo      AdminApprovalRepository
	users     approvalUserReader
	subs      approvalSubscriptionReader
	groups    approvalGroupReader
	apiKeys   approvalAPIKeyReader
	encryptor SecretEncryptor

	mu         sync.RWMutex
	notifier   ApprovalNotifier
	dispatcher http.Handler

	now         func() time.Time
	createLimit *approvalCreateLimiter
}

// NewAdminApprovalService 构造审批服务。users / subs / groups / apiKeys 只用于角色变更检查与目标摘要，
// 允许为 nil（摘要退化为 #id，角色变更无法核实时按拒绝处理）。
func NewAdminApprovalService(
	repo AdminApprovalRepository,
	users *UserService,
	subs *SubscriptionService,
	groups GroupRepository,
	apiKeys APIKeyRepository,
	encryptor SecretEncryptor,
) *AdminApprovalService {
	svc := &AdminApprovalService{
		repo:        repo,
		encryptor:   encryptor,
		now:         time.Now,
		createLimit: newApprovalCreateLimiter(AdminApprovalCreateRateLimit, AdminApprovalCreateRateWindow),
	}
	// 显式判空，避免 nil 指针包进非 nil 接口。
	if users != nil {
		svc.users = users
	}
	if subs != nil {
		svc.subs = subs
	}
	if groups != nil {
		svc.groups = groups
	}
	if apiKeys != nil {
		svc.apiKeys = apiKeys
	}
	return svc
}

// SetNotifier 挂上新申请推送钩子（Server酱³）。
func (s *AdminApprovalService) SetNotifier(n ApprovalNotifier) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notifier = n
}

// SetDispatcher 注入重放用的 HTTP 处理器（即 gin 引擎本身），路由注册完成后调用。
func (s *AdminApprovalService) SetDispatcher(h http.Handler) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dispatcher = h
}

func (s *AdminApprovalService) getNotifier() ApprovalNotifier {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.notifier
}

func (s *AdminApprovalService) getDispatcher() http.Handler {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dispatcher
}

// ---------- 捕获 ----------

// approvalBodyPeek 从请求体里窥探角色 / 目标信息，只用于校验与摘要。
type approvalBodyPeek struct {
	Role    string  `json:"role"`
	Email   string  `json:"email"`
	UserID  int64   `json:"user_id"`
	GroupID int64   `json:"group_id"`
	UserIDs []int64 `json:"user_ids"`
	All     bool    `json:"all"`
}

// Capture 校验并落库一条待审申请；返回值即 202 响应体的数据来源。
func (s *AdminApprovalService) Capture(ctx context.Context, in *AdminApprovalCaptureInput) (*AdminApprovalRequest, error) {
	if s == nil || s.repo == nil || s.encryptor == nil {
		return nil, ErrApprovalGateUnavailable
	}
	if in == nil || in.RequesterUserID <= 0 || in.Method == "" || in.RouteTemplate == "" {
		return nil, infraerrors.BadRequest("APPROVAL_INVALID_REQUEST", "invalid approval request")
	}
	if len(in.Body) > AuditRequestBodyCaptureLimit {
		return nil, ErrApprovalBodyTooLarge
	}
	peek := approvalBodyPeek{}
	if len(in.Body) > 0 {
		if !isJSONContentType(in.ContentType) || !json.Valid(in.Body) {
			return nil, ErrApprovalBodyNotJSON
		}
		// 解析失败（例如顶层是数组）不致命：只是拿不到摘要信息。
		_ = json.Unmarshal(in.Body, &peek)
	}
	if err := s.checkRoleChange(ctx, in, &peek); err != nil {
		return nil, err
	}

	now := s.now()
	if !s.createLimit.allow(in.RequesterUserID, now) {
		return nil, ErrApprovalRateLimited
	}
	pending, err := s.repo.CountPending(ctx, now, &in.RequesterUserID)
	if err != nil {
		return nil, fmt.Errorf("count pending approvals: %w", err)
	}
	if pending >= AdminApprovalPendingLimitPerUser {
		return nil, ErrApprovalPendingLimit
	}

	encrypted := ""
	if len(in.Body) > 0 {
		encrypted, err = s.encryptor.Encrypt(string(in.Body))
		if err != nil {
			return nil, fmt.Errorf("encrypt approval body: %w", err)
		}
	}
	sum := sha256.Sum256(in.Body)
	targetType, targetID, summary := s.resolveTarget(ctx, in, &peek)

	// 与迁移 239 的 VARCHAR 长度对齐；Content-Type / Request-ID 等由客户端控制，超长不能让入库 500。
	req := &AdminApprovalRequest{
		Status:              ApprovalStatusPending,
		Action:              clampRunes(in.Action, 128),
		Method:              clampRunes(in.Method, 10),
		RouteTemplate:       clampRunes(in.RouteTemplate, 255),
		RequestPath:         clampRunes(in.Path, 1024),
		RequestQuery:        in.RawQuery,
		ContentType:         clampRunes(in.ContentType, 128),
		RequestBodyEnc:      encrypted,
		RequestBodyRedacted: RedactAuditBody(in.Body, in.ContentType),
		RequestBodySHA256:   hex.EncodeToString(sum[:]),
		TargetType:          clampRunes(targetType, 32),
		TargetID:            targetID,
		TargetSummary:       clampRunes(summary, 255),
		RequesterUserID:     in.RequesterUserID,
		RequesterEmail:      clampRunes(in.RequesterEmail, 255),
		RequesterIP:         clampRunes(in.RequesterIP, 64),
		RequestID:           clampRunes(in.RequestID, 64),
		ExpiresAt:           now.Add(AdminApprovalTTL),
	}
	created, err := s.repo.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create approval request: %w", err)
	}
	if n := s.getNotifier(); n != nil {
		n.NotifyRequested(created, in.RequestOrigin)
	}
	return created, nil
}

func isJSONContentType(ct string) bool {
	ct = strings.ToLower(strings.TrimSpace(ct))
	if ct == "" {
		return true
	}
	return strings.HasPrefix(ct, "application/json") || strings.HasSuffix(strings.SplitN(ct, ";", 2)[0], "+json")
}

// checkRoleChange 角色变更不允许由 operator 发起：新建用户只能是普通用户；
// 编辑用户时 body 里的 role 必须等于目标当前角色。无法核实时按拒绝处理（fail-closed）。
func (s *AdminApprovalService) checkRoleChange(ctx context.Context, in *AdminApprovalCaptureInput, peek *approvalBodyPeek) error {
	role := strings.ToLower(strings.TrimSpace(peek.Role))
	switch {
	case in.Method == http.MethodPost && in.RouteTemplate == "/api/v1/admin/users":
		if role != "" && role != RoleUser {
			return ErrApprovalActionForbidden
		}
	case in.Method == http.MethodPut && in.RouteTemplate == "/api/v1/admin/users/:id":
		if role == "" {
			return nil
		}
		if s.users == nil {
			return ErrApprovalActionForbidden
		}
		id, err := strconv.ParseInt(strings.TrimSpace(in.Params["id"]), 10, 64)
		if err != nil || id <= 0 {
			return ErrApprovalActionForbidden
		}
		target, err := s.users.GetByID(ctx, id)
		if err != nil {
			if errors.Is(err, ErrUserNotFound) {
				// 目标不存在：放行入队，重放时自然 404。
				return nil
			}
			return fmt.Errorf("load target user for role check: %w", err)
		}
		if target == nil || role != strings.ToLower(target.Role) {
			return ErrApprovalActionForbidden
		}
	}
	return nil
}

// resolveTarget 尽力生成人类可读的目标摘要；任何查询失败都退化为 #id，不影响入队。
func (s *AdminApprovalService) resolveTarget(ctx context.Context, in *AdminApprovalCaptureInput, peek *approvalBodyPeek) (string, *int64, string) {
	route := in.RouteTemplate
	paramID := func() *int64 {
		id, err := strconv.ParseInt(strings.TrimSpace(in.Params["id"]), 10, 64)
		if err != nil || id <= 0 {
			return nil
		}
		return &id
	}
	switch {
	case route == "/api/v1/admin/users" && in.Method == http.MethodPost:
		summary := strings.TrimSpace(peek.Email)
		if summary == "" {
			summary = "new user"
		}
		return ApprovalTargetUser, nil, summary
	case strings.HasPrefix(route, "/api/v1/admin/users/batch-"):
		if peek.All {
			return ApprovalTargetUsersBatch, nil, "all users"
		}
		return ApprovalTargetUsersBatch, nil, fmt.Sprintf("%d users", len(peek.UserIDs))
	case strings.HasPrefix(route, "/api/v1/admin/users/:id"):
		id := paramID()
		return ApprovalTargetUser, id, s.userLabel(ctx, id)
	case route == "/api/v1/admin/subscriptions/assign":
		uid := peek.UserID
		return ApprovalTargetSubscription, nil, s.userLabel(ctx, &uid) + " · " + s.groupLabel(ctx, peek.GroupID)
	case route == "/api/v1/admin/subscriptions/bulk-assign":
		return ApprovalTargetUsersBatch, nil, fmt.Sprintf("%d users · %s", len(peek.UserIDs), s.groupLabel(ctx, peek.GroupID))
	case strings.HasPrefix(route, "/api/v1/admin/subscriptions/:id"):
		id := paramID()
		return ApprovalTargetSubscription, id, s.subscriptionLabel(ctx, id)
	case strings.HasPrefix(route, "/api/v1/admin/api-keys/:id"):
		id := paramID()
		return ApprovalTargetAPIKey, id, s.apiKeyLabel(ctx, id)
	}
	return "", nil, ""
}

func (s *AdminApprovalService) userLabel(ctx context.Context, id *int64) string {
	if id == nil || *id <= 0 {
		return "user"
	}
	if s.users != nil {
		if u, err := s.users.GetByID(ctx, *id); err == nil && u != nil && u.Email != "" {
			return u.Email
		}
	}
	return "user #" + strconv.FormatInt(*id, 10)
}

func (s *AdminApprovalService) groupLabel(ctx context.Context, id int64) string {
	if id <= 0 {
		return "group"
	}
	if s.groups != nil {
		if g, err := s.groups.GetByID(ctx, id); err == nil && g != nil && g.Name != "" {
			return g.Name
		}
	}
	return "group #" + strconv.FormatInt(id, 10)
}

func (s *AdminApprovalService) subscriptionLabel(ctx context.Context, id *int64) string {
	if id == nil || *id <= 0 {
		return "subscription"
	}
	fallback := "subscription #" + strconv.FormatInt(*id, 10)
	if s.subs == nil {
		return fallback
	}
	sub, err := s.subs.GetByID(ctx, *id)
	if err != nil || sub == nil {
		return fallback
	}
	user := ""
	if sub.User != nil && sub.User.Email != "" {
		user = sub.User.Email
	} else {
		uid := sub.UserID
		user = s.userLabel(ctx, &uid)
	}
	group := ""
	if sub.Group != nil && sub.Group.Name != "" {
		group = sub.Group.Name
	} else {
		group = s.groupLabel(ctx, sub.GroupID)
	}
	return user + " · " + group
}

func (s *AdminApprovalService) apiKeyLabel(ctx context.Context, id *int64) string {
	if id == nil || *id <= 0 {
		return "api key"
	}
	fallback := "api key #" + strconv.FormatInt(*id, 10)
	if s.apiKeys == nil {
		return fallback
	}
	key, err := s.apiKeys.GetByID(ctx, *id)
	if err != nil || key == nil {
		return fallback
	}
	name := strings.TrimSpace(key.Name)
	if name == "" {
		name = fallback
	}
	uid := key.UserID
	return name + " (" + s.userLabel(ctx, &uid) + ")"
}

// ---------- 查询 ----------

// List 分页列表；展示状态经 EffectiveStatus 归一（过期的 pending 显示为 expired）。
func (s *AdminApprovalService) List(ctx context.Context, filter *AdminApprovalFilter) (*AdminApprovalList, error) {
	if s == nil || s.repo == nil {
		return nil, ErrApprovalGateUnavailable
	}
	f := AdminApprovalFilter{}
	if filter != nil {
		f = *filter
	}
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 {
		f.PageSize = 20
	}
	if f.PageSize > 100 {
		f.PageSize = 100
	}
	items, total, err := s.repo.List(ctx, &f)
	if err != nil {
		return nil, fmt.Errorf("list approval requests: %w", err)
	}
	now := s.now()
	for _, item := range items {
		item.Status = item.EffectiveStatus(now)
	}
	return &AdminApprovalList{Items: items, Total: total, Page: f.Page, PageSize: f.PageSize}, nil
}

// Get 读取单条；非管理员只能看自己的申请，其它一律按不存在处理（不泄露存在性）。
func (s *AdminApprovalService) Get(ctx context.Context, id int64, viewer ApprovalActor) (*AdminApprovalRequest, error) {
	if s == nil || s.repo == nil {
		return nil, ErrApprovalGateUnavailable
	}
	req, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if viewer.Role != RoleAdmin && req.RequesterUserID != viewer.UserID {
		return nil, ErrApprovalNotFound
	}
	req.Status = req.EffectiveStatus(s.now())
	return req, nil
}

// PendingCount 待审数量；requesterUserID 为 nil 表示全站（管理员角标）。
func (s *AdminApprovalService) PendingCount(ctx context.Context, requesterUserID *int64) (int64, error) {
	if s == nil || s.repo == nil {
		return 0, ErrApprovalGateUnavailable
	}
	return s.repo.CountPending(ctx, s.now(), requesterUserID)
}

// ---------- 决策 ----------

// Approve 一键通过：原子转为 executing，以审批人身份在进程内重放原始请求，再按响应写回结果。
func (s *AdminApprovalService) Approve(ctx context.Context, id int64, approver ApprovalActor) (*AdminApprovalRequest, *ApprovalReplayResult, error) {
	if s == nil || s.repo == nil {
		return nil, nil, ErrApprovalGateUnavailable
	}
	if err := s.verifyApprover(ctx, approver); err != nil {
		return nil, nil, err
	}
	dispatcher := s.getDispatcher()
	if dispatcher == nil {
		return nil, nil, ErrApprovalDispatcherMissing
	}

	req, err := s.repo.TransitionToExecuting(ctx, id, approver.UserID, approver.Email, s.now())
	if err != nil {
		return nil, nil, err
	}

	body, err := s.decryptBody(req)
	if err != nil {
		_ = s.repo.FinishExecution(ctx, id, ApprovalStatusFailed, 0, "", "decrypt_failed", s.now())
		return nil, nil, fmt.Errorf("decrypt approval body: %w", err)
	}

	result := s.replay(ctx, req, body, approver, dispatcher)
	status := ApprovalStatusFailed
	errText := ""
	if result.StatusCode >= 200 && result.StatusCode < 300 {
		status = ApprovalStatusApproved
	} else {
		errText = extractApprovalErrorCode(result.Body)
		if errText == "" {
			errText = "http_" + strconv.Itoa(result.StatusCode)
		}
	}
	if err := s.repo.FinishExecution(ctx, id, status, result.StatusCode, truncateApprovalBody(result.Body), errText, s.now()); err != nil {
		return nil, &result, fmt.Errorf("finish approval execution: %w", err)
	}
	updated, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, &result, err
	}
	return updated, &result, nil
}

func (s *AdminApprovalService) verifyApprover(ctx context.Context, approver ApprovalActor) error {
	if approver.UserID <= 0 || approver.AuthMethod == AuditAuthMethodAdminAPIKey {
		return ErrApprovalApproverInvalid
	}
	if s.users != nil {
		u, err := s.users.GetByID(ctx, approver.UserID)
		if err != nil || u == nil || !u.IsAdmin() || !u.IsActive() {
			return ErrApprovalApproverInvalid
		}
		return nil
	}
	if approver.Role != RoleAdmin {
		return ErrApprovalApproverInvalid
	}
	return nil
}

func (s *AdminApprovalService) decryptBody(req *AdminApprovalRequest) ([]byte, error) {
	if req.RequestBodyEnc == "" {
		return nil, nil
	}
	if s.encryptor == nil {
		return nil, ErrApprovalGateUnavailable
	}
	plain, err := s.encryptor.Decrypt(req.RequestBodyEnc)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256([]byte(plain))
	if req.RequestBodySHA256 != "" && hex.EncodeToString(sum[:]) != req.RequestBodySHA256 {
		return nil, errors.New("approval body checksum mismatch")
	}
	return []byte(plain), nil
}

// replay 构造与原请求等价的 *http.Request，挂上 ApprovalReplay 身份后交给 gin 引擎处理。
func (s *AdminApprovalService) replay(ctx context.Context, req *AdminApprovalRequest, body []byte, approver ApprovalActor, dispatcher http.Handler) ApprovalReplayResult {
	replayCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), AdminApprovalReplayTimeout)
	defer cancel()
	replayCtx = WithApprovalReplay(replayCtx, &ApprovalReplay{
		ApprovalID:          req.ID,
		RequesterUserID:     req.RequesterUserID,
		ApproverUserID:      approver.UserID,
		ApproverConcurrency: approver.Concurrency,
		ApproverRole:        approver.Role,
		ApproverEmail:       approver.Email,
		ApproverSessionID:   approver.SessionID,
	})

	target := req.RequestPath
	if req.RequestQuery != "" {
		target += "?" + req.RequestQuery
	}
	httpReq, err := http.NewRequestWithContext(replayCtx, req.Method, target, bytes.NewReader(body))
	if err != nil {
		return ApprovalReplayResult{StatusCode: 0, Body: "build replay request: " + err.Error()}
	}
	httpReq.Host = "sub2api-approval-replay"
	if len(body) > 0 {
		ct := req.ContentType
		if ct == "" {
			ct = "application/json"
		}
		httpReq.Header.Set("Content-Type", ct)
	}
	httpReq.Header.Set("Idempotency-Key", "approval-"+strconv.FormatInt(req.ID, 10))
	httpReq.Header.Set("X-Admin-UI-Request", "1")
	httpReq.Header.Set("User-Agent", "sub2api-approval-replay/1")
	if approver.RequestID != "" {
		httpReq.Header.Set("X-Request-ID", approver.RequestID)
	}
	if approver.AcceptLanguage != "" {
		httpReq.Header.Set("Accept-Language", approver.AcceptLanguage)
	}
	if ip := strings.TrimSpace(approver.ClientIP); ip != "" {
		httpReq.RemoteAddr = net.JoinHostPort(ip, "0")
	}

	w := newApprovalReplayWriter()
	start := time.Now()
	func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("approval replay panicked", "approval_id", req.ID, "panic", fmt.Sprint(r))
				w.wrote = true
				w.statusCode = http.StatusInternalServerError
				w.body.Reset()
				w.body.WriteString(`{"code":500,"message":"replay panicked"}`)
			}
		}()
		dispatcher.ServeHTTP(w, httpReq)
	}()
	return ApprovalReplayResult{
		StatusCode: w.status(),
		Body:       w.body.String(),
		DurationMs: time.Since(start).Milliseconds(),
	}
}

// Reject 管理员拒绝一条待审申请。
func (s *AdminApprovalService) Reject(ctx context.Context, id int64, actor ApprovalActor, reason string) (*AdminApprovalRequest, error) {
	if s == nil || s.repo == nil {
		return nil, ErrApprovalGateUnavailable
	}
	if err := s.verifyApprover(ctx, actor); err != nil {
		return nil, err
	}
	reason = strings.TrimSpace(reason)
	if len([]rune(reason)) > 500 {
		reason = string([]rune(reason)[:500])
	}
	return s.decide(ctx, id, ApprovalStatusRejected, actor, reason)
}

// Cancel 申请人撤回自己的待审申请（管理员也可撤回任意申请）。
func (s *AdminApprovalService) Cancel(ctx context.Context, id int64, actor ApprovalActor) (*AdminApprovalRequest, error) {
	if s == nil || s.repo == nil {
		return nil, ErrApprovalGateUnavailable
	}
	req, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if actor.Role != RoleAdmin && req.RequesterUserID != actor.UserID {
		// 不是本人：按不存在处理，避免泄露他人申请。
		return nil, ErrApprovalNotFound
	}
	return s.decide(ctx, id, ApprovalStatusCancelled, actor, "")
}

func (s *AdminApprovalService) decide(ctx context.Context, id int64, toStatus string, actor ApprovalActor, reason string) (*AdminApprovalRequest, error) {
	ok, err := s.repo.Decide(ctx, id, toStatus, actor.UserID, actor.Email, reason, s.now())
	if err != nil {
		return nil, fmt.Errorf("decide approval request: %w", err)
	}
	if !ok {
		if _, err := s.repo.GetByID(ctx, id); err != nil {
			return nil, err
		}
		return nil, ErrApprovalNotPending
	}
	return s.repo.GetByID(ctx, id)
}

// SweepOnce 把过期的 pending 置为 expired、卡住的 executing 置为 failed。
func (s *AdminApprovalService) SweepOnce(ctx context.Context) (expired, stuck int64, err error) {
	if s == nil || s.repo == nil {
		return 0, 0, ErrApprovalGateUnavailable
	}
	now := s.now()
	expired, err = s.repo.ExpirePending(ctx, now)
	if err != nil {
		return 0, 0, fmt.Errorf("expire pending approvals: %w", err)
	}
	stuck, err = s.repo.FailStuckExecuting(ctx, now.Add(-AdminApprovalExecutingTimeout))
	if err != nil {
		return expired, 0, fmt.Errorf("fail stuck approvals: %w", err)
	}
	return expired, stuck, nil
}

// ---------- 辅助 ----------

// approvalReplayWriter 最小化的 http.ResponseWriter，只记录状态码与响应体。
type approvalReplayWriter struct {
	header     http.Header
	statusCode int
	body       bytes.Buffer
	wrote      bool
}

func newApprovalReplayWriter() *approvalReplayWriter {
	return &approvalReplayWriter{header: http.Header{}}
}

func (w *approvalReplayWriter) Header() http.Header { return w.header }

func (w *approvalReplayWriter) WriteHeader(code int) {
	if w.wrote {
		return
	}
	w.wrote = true
	w.statusCode = code
}

func (w *approvalReplayWriter) Write(p []byte) (int, error) {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	// 只保留入库上限附近的内容，避免异常响应撑爆内存。
	if w.body.Len() < AdminApprovalResultBodyMaxBytes*2 {
		w.body.Write(p)
	}
	return len(p), nil
}

func (w *approvalReplayWriter) status() int {
	if !w.wrote {
		return http.StatusOK
	}
	return w.statusCode
}

func truncateApprovalBody(body string) string {
	if len(body) <= AdminApprovalResultBodyMaxBytes {
		return body
	}
	return body[:AdminApprovalResultBodyMaxBytes] + "...<truncated>"
}

// extractApprovalErrorCode 从统一错误信封里取出业务错误码（code / reason / error 任一字符串字段）。
func extractApprovalErrorCode(body string) string {
	body = strings.TrimSpace(body)
	if body == "" || !strings.HasPrefix(body, "{") {
		return ""
	}
	var envelope map[string]any
	if err := json.Unmarshal([]byte(body), &envelope); err != nil {
		return ""
	}
	for _, key := range []string{"code", "reason", "error"} {
		if v, ok := envelope[key].(string); ok && strings.TrimSpace(v) != "" {
			return truncateApprovalErrorCode(v)
		}
	}
	if v, ok := envelope["code"].(float64); ok && v != 0 {
		return "code_" + strconv.FormatInt(int64(v), 10)
	}
	return ""
}

// clampRunes 按字符数截断（Postgres VARCHAR(n) 以字符计），避免超长字段让入库失败。
func clampRunes(v string, max int) string {
	if max <= 0 || len(v) <= max {
		return v
	}
	r := []rune(v)
	if len(r) <= max {
		return v
	}
	return string(r[:max])
}

func truncateApprovalErrorCode(v string) string {
	v = strings.TrimSpace(v)
	if len(v) > 128 {
		return v[:128]
	}
	return v
}

// approvalCreateLimiter 每个申请人在滑动窗口内的创建次数上限（进程内，防止刷队列）。
type approvalCreateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	buckets map[int64]*approvalCreateBucket
}

type approvalCreateBucket struct {
	windowStart time.Time
	count       int
}

func newApprovalCreateLimiter(limit int, window time.Duration) *approvalCreateLimiter {
	return &approvalCreateLimiter{limit: limit, window: window, buckets: map[int64]*approvalCreateBucket{}}
}

func (l *approvalCreateLimiter) allow(userID int64, now time.Time) bool {
	if l == nil || l.limit <= 0 {
		return true
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	bucket, ok := l.buckets[userID]
	if !ok || now.Sub(bucket.windowStart) >= l.window {
		for id, b := range l.buckets {
			if now.Sub(b.windowStart) >= l.window {
				delete(l.buckets, id)
			}
		}
		l.buckets[userID] = &approvalCreateBucket{windowStart: now, count: 1}
		return true
	}
	if bucket.count >= l.limit {
		return false
	}
	bucket.count++
	return true
}

// AdminApprovalSweeper 定时回收过期 / 卡住的申请。
type AdminApprovalSweeper struct {
	svc      *AdminApprovalService
	interval time.Duration
	stop     chan struct{}
	once     sync.Once
	wg       sync.WaitGroup
}

// NewAdminApprovalSweeper 构造 sweeper；interval <= 0 时使用 10 分钟。
func NewAdminApprovalSweeper(svc *AdminApprovalService, interval time.Duration) *AdminApprovalSweeper {
	if interval <= 0 {
		interval = 10 * time.Minute
	}
	return &AdminApprovalSweeper{svc: svc, interval: interval, stop: make(chan struct{})}
}

// Start 启动后台循环（幂等）。
func (w *AdminApprovalSweeper) Start() {
	if w == nil || w.svc == nil {
		return
	}
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-w.stop:
				return
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				expired, stuck, err := w.svc.SweepOnce(ctx)
				cancel()
				if err != nil {
					slog.Warn("approval sweeper failed", "error", err)
				} else if expired > 0 || stuck > 0 {
					slog.Info("approval sweeper", "expired", expired, "stuck_failed", stuck)
				}
			}
		}
	}()
}

// Stop 停止后台循环并等待退出。
func (w *AdminApprovalSweeper) Stop() {
	if w == nil {
		return
	}
	w.once.Do(func() { close(w.stop) })
	w.wg.Wait()
}

// ---------- 批量通过 ----------

// ApprovalBatchStatusSkipped 批量通过里未执行的条目（状态不对 / 已过期 / 不存在 / 被取消）。
const ApprovalBatchStatusSkipped = "skipped"

// ApprovalBatchItem 批量通过里单条的结果：Status 为 approved / failed（重放已执行）或 skipped（未执行）。
type ApprovalBatchItem struct {
	ID         int64
	Status     string
	Error      string
	StatusCode int
}

// ApproveBatch 逐条一键通过：审批人校验一次，各条顺序重放、互不影响，单条失败不中断整批。
// 去重后为空 → ErrApprovalBatchEmpty；超过 AdminApprovalBatchLimit → ErrApprovalBatchTooLarge。
func (s *AdminApprovalService) ApproveBatch(ctx context.Context, ids []int64, approver ApprovalActor) ([]ApprovalBatchItem, error) {
	if s == nil || s.repo == nil {
		return nil, ErrApprovalGateUnavailable
	}
	if err := s.verifyApprover(ctx, approver); err != nil {
		return nil, err
	}
	if s.getDispatcher() == nil {
		return nil, ErrApprovalDispatcherMissing
	}
	unique := make([]int64, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	if len(unique) == 0 {
		return nil, ErrApprovalBatchEmpty
	}
	if len(unique) > AdminApprovalBatchLimit {
		return nil, ErrApprovalBatchTooLarge
	}

	results := make([]ApprovalBatchItem, 0, len(unique))
	for _, id := range unique {
		item := ApprovalBatchItem{ID: id}
		if err := ctx.Err(); err != nil {
			item.Status = ApprovalBatchStatusSkipped
			item.Error = "context_cancelled"
			results = append(results, item)
			continue
		}
		req, replay, err := s.Approve(ctx, id, approver)
		if err != nil {
			item.Status = ApprovalBatchStatusSkipped
			item.Error = approvalErrorReason(err)
			if replay != nil {
				item.StatusCode = replay.StatusCode
			}
			results = append(results, item)
			continue
		}
		item.Status = req.Status
		item.Error = req.ResultError
		if replay != nil {
			item.StatusCode = replay.StatusCode
		}
		results = append(results, item)
	}
	return results, nil
}

// approvalErrorReason 取业务错误码，没有则退化为截断后的错误文本。
func approvalErrorReason(err error) string {
	if err == nil {
		return ""
	}
	if reason := infraerrors.Reason(err); reason != "" && reason != infraerrors.UnknownReason {
		return reason
	}
	return truncateApprovalErrorCode(err.Error())
}
