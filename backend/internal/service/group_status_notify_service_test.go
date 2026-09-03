package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// ---------- 测试桩 ----------

type groupStatusNotifySettingRepo struct {
	SettingRepository
	values map[string]string
}

func (r *groupStatusNotifySettingRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		if v, ok := r.values[key]; ok {
			out[key] = v
		}
	}
	return out, nil
}

type serverChanTestRequest struct {
	Method string
	Path   string
	Title  string
	Desp   string
}

type serverChanTestResponse struct {
	Status int
	Body   string
}

// serverChanTestServer 记录每次请求，并按顺序返回预设响应（用尽后重复最后一个）。
type serverChanTestServer struct {
	server    *httptest.Server
	mu        sync.Mutex
	requests  []serverChanTestRequest
	responses []serverChanTestResponse
}

func newServerChanTestServer(t *testing.T, responses ...serverChanTestResponse) *serverChanTestServer {
	t.Helper()
	ts := &serverChanTestServer{responses: responses}
	ts.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		ts.mu.Lock()
		ts.requests = append(ts.requests, serverChanTestRequest{
			Method: r.Method,
			// 用转义形式，才能验证 sendkey 里的特殊字符确实被 PathEscape 过
			Path:  r.URL.EscapedPath(),
			Title: r.FormValue("title"),
			Desp:  r.FormValue("desp"),
		})
		idx := len(ts.requests) - 1
		resp := serverChanTestResponse{Status: http.StatusOK, Body: `{"code":0}`}
		if len(ts.responses) > 0 {
			if idx < len(ts.responses) {
				resp = ts.responses[idx]
			} else {
				resp = ts.responses[len(ts.responses)-1]
			}
		}
		ts.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.Status)
		_, _ = w.Write([]byte(resp.Body))
	}))
	t.Cleanup(ts.server.Close)
	return ts
}

func (ts *serverChanTestServer) count() int {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return len(ts.requests)
}

func (ts *serverChanTestServer) request(i int) serverChanTestRequest {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.requests[i]
}

func newServerChanClientForTest(baseURL string) *serverChanClient {
	return &serverChanClient{
		baseURL: baseURL,
		// 不真正等待，只尊重 ctx 取消
		sleep: func(ctx context.Context, _ time.Duration) error { return ctx.Err() },
	}
}

func newGroupStatusNotifyServiceForTest(baseURL string, values map[string]string) *GroupStatusNotifyService {
	return &GroupStatusNotifyService{
		settingRepo: &groupStatusNotifySettingRepo{values: values},
		client:      newServerChanClientForTest(baseURL),
	}
}

func serverChanEnabledSettings(uid, sendkey string) map[string]string {
	return map[string]string{
		SettingKeyGroupStatusNotifyServerChanEnabled: "true",
		SettingKeyGroupStatusNotifyServerChanUID:     uid,
		SettingKeyGroupStatusNotifyServerChanSendKey: sendkey,
		SettingKeySiteName:                           "My Gateway",
	}
}

func groupStatusNotifyDownEvent() *GroupStatusEvent {
	httpCode := 502
	latency := int64(1234)
	return &GroupStatusEvent{
		GroupID:     7,
		ConfigID:    3,
		EventType:   GroupStatusEventDown,
		FromStatus:  GroupRuntimeStatusUp,
		ToStatus:    GroupRuntimeStatusDown,
		HTTPCode:    &httpCode,
		LatencyMS:   &latency,
		SubStatus:   "http_5xx",
		ErrorDetail: "upstream returned 502\nbad gateway",
		ObservedAt:  time.Date(2026, 9, 3, 10, 30, 0, 0, time.UTC),
	}
}

// ---------- serverChanClient ----------

func TestServerChanClient_EndpointUsesDefaultHostAndEscapesSendKey(t *testing.T) {
	c := newServerChanClient()
	require.Equal(t, "https://12345.push.ft07.com/send/sctp_abc%2Fdef.send", c.endpoint("12345", "sctp_abc/def"))

	c.baseURL = "http://127.0.0.1:9/"
	require.Equal(t, "http://127.0.0.1:9/send/key.send", c.endpoint("12345", "key"))
}

func TestServerChanClient_SendPostsFormToEndpoint(t *testing.T) {
	ts := newServerChanTestServer(t)
	c := newServerChanClientForTest(ts.server.URL)

	err := c.send(context.Background(), "12345", "sctp_abc/def", "hello", "world")
	require.NoError(t, err)
	require.Equal(t, 1, ts.count())

	got := ts.request(0)
	require.Equal(t, http.MethodPost, got.Method)
	require.Equal(t, "/send/"+url.PathEscape("sctp_abc/def")+".send", got.Path)
	require.Equal(t, "hello", got.Title)
	require.Equal(t, "world", got.Desp)
}

func TestServerChanClient_SendReturnsErrorOnNonZeroCode(t *testing.T) {
	ts := newServerChanTestServer(t, serverChanTestResponse{Status: http.StatusOK, Body: `{"code":40001,"message":"bad key"}`})
	c := newServerChanClientForTest(ts.server.URL)

	err := c.send(context.Background(), "12345", "sctp_secret", "t", "d")
	require.Error(t, err)
	require.Contains(t, err.Error(), "bad key")
	require.NotContains(t, err.Error(), "sctp_secret")
}

func TestServerChanClient_SendFallsBackToMsgAndCode(t *testing.T) {
	ts := newServerChanTestServer(t,
		serverChanTestResponse{Status: http.StatusOK, Body: `{"code":1,"msg":"legacy msg"}`},
		serverChanTestResponse{Status: http.StatusOK, Body: `{"code":2}`},
	)
	c := newServerChanClientForTest(ts.server.URL)

	err := c.send(context.Background(), "12345", "k", "t", "d")
	require.ErrorContains(t, err, "legacy msg")

	err = c.send(context.Background(), "12345", "k", "t", "d")
	require.ErrorContains(t, err, "code 2")
}

func TestServerChanClient_SendReturnsErrorOnHTTPError(t *testing.T) {
	ts := newServerChanTestServer(t, serverChanTestResponse{Status: http.StatusInternalServerError, Body: `oops`})
	c := newServerChanClientForTest(ts.server.URL)

	err := c.send(context.Background(), "12345", "k", "t", "d")
	require.ErrorContains(t, err, "http 500")
}

func TestServerChanClient_SendWithRetry_RetriesThenSucceeds(t *testing.T) {
	ts := newServerChanTestServer(t,
		serverChanTestResponse{Status: http.StatusBadGateway, Body: ``},
		serverChanTestResponse{Status: http.StatusOK, Body: `{"code":500,"message":"busy"}`},
		serverChanTestResponse{Status: http.StatusOK, Body: `{"code":0}`},
	)
	c := newServerChanClientForTest(ts.server.URL)

	err := c.sendWithRetry(context.Background(), "12345", "k", "t", "d")
	require.NoError(t, err)
	require.Equal(t, 3, ts.count())
}

func TestServerChanClient_SendWithRetry_GivesUpAfterMaxAttempts(t *testing.T) {
	ts := newServerChanTestServer(t, serverChanTestResponse{Status: http.StatusBadGateway, Body: ``})
	c := newServerChanClientForTest(ts.server.URL)

	err := c.sendWithRetry(context.Background(), "12345", "k", "t", "d")
	require.ErrorContains(t, err, "http 502")
	require.Equal(t, groupStatusNotifyMaxAttempts, ts.count())
}

func TestServerChanClient_SendWithRetry_StopsWhenContextCancelled(t *testing.T) {
	ts := newServerChanTestServer(t)
	c := newServerChanClientForTest(ts.server.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := c.sendWithRetry(ctx, "12345", "k", "t", "d")
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, 0, ts.count())
}

func TestServerChanBackoff(t *testing.T) {
	require.Equal(t, time.Second, serverChanBackoff(1))
	require.Equal(t, 2*time.Second, serverChanBackoff(2))
	require.Equal(t, 4*time.Second, serverChanBackoff(3))
	require.Equal(t, 30*time.Second, serverChanBackoff(10))
	require.Equal(t, time.Second, serverChanBackoff(0))
}

func TestRedactServerChanSecret(t *testing.T) {
	key := "sctp_abc/def"
	err := errors.New(`Post "https://1.push.ft07.com/send/sctp_abc%2Fdef.send": dial tcp: sctp_abc/def timeout`)
	redacted := redactServerChanSecret(err, key)
	require.NotContains(t, redacted.Error(), key)
	require.NotContains(t, redacted.Error(), url.PathEscape(key))
	require.Contains(t, redacted.Error(), "***")

	require.Nil(t, redactServerChanSecret(nil, key))
	plain := errors.New("nothing secret here")
	require.Same(t, plain, redactServerChanSecret(plain, key))
	require.Same(t, plain, redactServerChanSecret(plain, ""))
}

func TestValidateServerChanUID(t *testing.T) {
	require.NoError(t, ValidateServerChanUID("12345"))
	require.NoError(t, ValidateServerChanUID(" abc_DEF-9 "))

	for _, bad := range []string{"", "evil.com", "a b", "uid/x", strings.Repeat("a", 65)} {
		require.ErrorIs(t, ValidateServerChanUID(bad), ErrGroupStatusNotifyInvalidUID, "uid=%q", bad)
	}
}

// ---------- 消息构建 ----------

func TestBuildGroupStatusNotifyMessage_Down(t *testing.T) {
	group := &Group{ID: 7, Name: "Claude 高速", Platform: PlatformAnthropic}
	title, desp := buildGroupStatusNotifyMessage("My Gateway", group, groupStatusNotifyDownEvent())

	require.Equal(t, "[My Gateway] 分组「Claude 高速」状态变红", title)
	require.Contains(t, desp, "**分组**：Claude 高速（#7 / anthropic）")
	require.Contains(t, desp, "**状态**：正常 → 不可用")
	require.Contains(t, desp, "**子状态**：http_5xx")
	require.Contains(t, desp, "**HTTP**：502")
	require.Contains(t, desp, "**延迟**：1234 ms")
	require.Contains(t, desp, "**错误**：upstream returned 502 bad gateway")
	require.Contains(t, desp, "**时间**：")
}

func TestBuildGroupStatusNotifyMessage_Up(t *testing.T) {
	group := &Group{ID: 7, Name: "", Platform: ""}
	event := &GroupStatusEvent{
		EventType:  GroupStatusEventUp,
		FromStatus: GroupRuntimeStatusDown,
		ToStatus:   GroupRuntimeStatusDegraded,
		ObservedAt: time.Now(),
	}
	title, desp := buildGroupStatusNotifyMessage("", group, event)

	require.Equal(t, "[Sub2API] 分组「#7」已恢复", title)
	require.Contains(t, desp, "**分组**：#7（#7 / -）")
	require.Contains(t, desp, "**状态**：不可用 → 降级")
	require.Contains(t, desp, "**子状态**：-")
	require.Contains(t, desp, "**HTTP**：-")
	require.Contains(t, desp, "**延迟**：-")
	require.Contains(t, desp, "**错误**：-")
}

func TestTruncateGroupStatusErrorDetail(t *testing.T) {
	require.Equal(t, "-", truncateGroupStatusErrorDetail("  \n "))
	require.Equal(t, "a b c", truncateGroupStatusErrorDetail("a\nb\t c"))

	long := strings.Repeat("错", groupStatusNotifyErrorDetailMax+5)
	got := truncateGroupStatusErrorDetail(long)
	require.Equal(t, groupStatusNotifyErrorDetailMax+3, len([]rune(got)))
	require.True(t, strings.HasSuffix(got, "..."))
}

// ---------- GroupStatusNotifyService ----------

func TestGroupStatusNotifyService_Deliver_SendsDownEvent(t *testing.T) {
	ts := newServerChanTestServer(t)
	svc := newGroupStatusNotifyServiceForTest(ts.server.URL, serverChanEnabledSettings("12345", "sctp_secret"))

	svc.deliver(&Group{ID: 7, Name: "G7", Platform: PlatformOpenAI}, groupStatusNotifyDownEvent())

	require.Equal(t, 1, ts.count())
	got := ts.request(0)
	require.Equal(t, "/send/sctp_secret.send", got.Path)
	require.Equal(t, "[My Gateway] 分组「G7」状态变红", got.Title)
	require.Contains(t, got.Desp, "**状态**：正常 → 不可用")
}

func TestGroupStatusNotifyService_Deliver_SkipsWhenGloballyDisabled(t *testing.T) {
	ts := newServerChanTestServer(t)
	values := serverChanEnabledSettings("12345", "sctp_secret")
	values[SettingKeyGroupStatusNotifyServerChanEnabled] = "false"
	svc := newGroupStatusNotifyServiceForTest(ts.server.URL, values)

	svc.deliver(&Group{ID: 7, Name: "G7"}, groupStatusNotifyDownEvent())
	require.Equal(t, 0, ts.count())
}

func TestGroupStatusNotifyService_Deliver_SkipsWhenSendKeyMissing(t *testing.T) {
	ts := newServerChanTestServer(t)
	svc := newGroupStatusNotifyServiceForTest(ts.server.URL, serverChanEnabledSettings("12345", ""))

	svc.deliver(&Group{ID: 7, Name: "G7"}, groupStatusNotifyDownEvent())
	require.Equal(t, 0, ts.count())
}

func TestGroupStatusNotifyService_Deliver_SkipsWhenUIDInvalid(t *testing.T) {
	ts := newServerChanTestServer(t)
	svc := newGroupStatusNotifyServiceForTest(ts.server.URL, serverChanEnabledSettings("evil.com", "sctp_secret"))

	svc.deliver(&Group{ID: 7, Name: "G7"}, groupStatusNotifyDownEvent())
	require.Equal(t, 0, ts.count())
}

func TestGroupStatusNotifyService_NotifyTransition_PushesAsynchronously(t *testing.T) {
	ts := newServerChanTestServer(t)
	svc := newGroupStatusNotifyServiceForTest(ts.server.URL, serverChanEnabledSettings("12345", "sctp_secret"))

	cfg := &GroupStatusConfig{GroupID: 7, NotifyEnabled: true}
	svc.NotifyTransition(&Group{ID: 7, Name: "G7"}, cfg, groupStatusNotifyDownEvent())

	require.Eventually(t, func() bool { return ts.count() == 1 }, 5*time.Second, 10*time.Millisecond)
}

func TestGroupStatusNotifyService_NotifyTransition_SkipsGroupWithNotifyDisabled(t *testing.T) {
	ts := newServerChanTestServer(t)
	svc := newGroupStatusNotifyServiceForTest(ts.server.URL, serverChanEnabledSettings("12345", "sctp_secret"))

	cfg := &GroupStatusConfig{GroupID: 7, NotifyEnabled: false}
	svc.NotifyTransition(&Group{ID: 7, Name: "G7"}, cfg, groupStatusNotifyDownEvent())

	time.Sleep(50 * time.Millisecond)
	require.Equal(t, 0, ts.count())
}

func TestGroupStatusNotifyService_NotifyTransition_IgnoresNonTransitionEvents(t *testing.T) {
	ts := newServerChanTestServer(t)
	svc := newGroupStatusNotifyServiceForTest(ts.server.URL, serverChanEnabledSettings("12345", "sctp_secret"))

	cfg := &GroupStatusConfig{GroupID: 7, NotifyEnabled: true}
	event := groupStatusNotifyDownEvent()
	event.EventType = "flap"
	svc.NotifyTransition(&Group{ID: 7, Name: "G7"}, cfg, event)
	svc.NotifyTransition(nil, cfg, groupStatusNotifyDownEvent())
	svc.NotifyTransition(&Group{ID: 7}, nil, groupStatusNotifyDownEvent())
	svc.NotifyTransition(&Group{ID: 7}, cfg, nil)

	var nilSvc *GroupStatusNotifyService
	nilSvc.NotifyTransition(&Group{ID: 7}, cfg, groupStatusNotifyDownEvent())

	time.Sleep(50 * time.Millisecond)
	require.Equal(t, 0, ts.count())
}

func TestGroupStatusNotifyService_SendTest_FallsBackToSavedConfig(t *testing.T) {
	ts := newServerChanTestServer(t)
	svc := newGroupStatusNotifyServiceForTest(ts.server.URL, serverChanEnabledSettings("12345", "sctp_saved"))

	require.NoError(t, svc.SendTest(context.Background(), "", ""))
	require.Equal(t, 1, ts.count())
	got := ts.request(0)
	require.Equal(t, "/send/sctp_saved.send", got.Path)
	require.Equal(t, "[My Gateway] 分组运行状态测试推送", got.Title)
}

func TestGroupStatusNotifyService_SendTest_PrefersProvidedValues(t *testing.T) {
	ts := newServerChanTestServer(t)
	values := serverChanEnabledSettings("12345", "sctp_saved")
	values[SettingKeyGroupStatusNotifyServerChanEnabled] = "false" // 测试推送不要求总开关打开
	svc := newGroupStatusNotifyServiceForTest(ts.server.URL, values)

	require.NoError(t, svc.SendTest(context.Background(), " 999 ", " sctp_new "))
	require.Equal(t, "/send/sctp_new.send", ts.request(0).Path)
}

func TestGroupStatusNotifyService_SendTest_RequiresConfig(t *testing.T) {
	ts := newServerChanTestServer(t)
	svc := newGroupStatusNotifyServiceForTest(ts.server.URL, map[string]string{})

	err := svc.SendTest(context.Background(), "", "")
	require.ErrorIs(t, err, ErrGroupStatusNotifyNotConfigured)
	require.Equal(t, 0, ts.count())
}

func TestGroupStatusNotifyService_SendTest_RejectsInvalidUID(t *testing.T) {
	ts := newServerChanTestServer(t)
	svc := newGroupStatusNotifyServiceForTest(ts.server.URL, map[string]string{})

	err := svc.SendTest(context.Background(), "evil.com", "sctp_x")
	require.ErrorIs(t, err, ErrGroupStatusNotifyInvalidUID)
	require.Equal(t, 0, ts.count())
}

func TestGroupStatusNotifyService_SendTest_RedactsSendKeyInError(t *testing.T) {
	ts := newServerChanTestServer(t, serverChanTestResponse{Status: http.StatusOK, Body: `{"code":40001,"message":"invalid sctp_secret"}`})
	svc := newGroupStatusNotifyServiceForTest(ts.server.URL, map[string]string{})

	err := svc.SendTest(context.Background(), "12345", "sctp_secret")
	require.Error(t, err)
	require.NotContains(t, err.Error(), "sctp_secret")
}

// ---------- 探测落库 → 通知钩子 ----------

type groupStatusNotifyProbeRepo struct {
	GroupStatusRepository
	event *GroupStatusEvent
}

func (r *groupStatusNotifyProbeRepo) SaveProbeResult(_ context.Context, result *GroupStatusProbeResult) (*GroupStatusState, *GroupStatusEvent, error) {
	return &GroupStatusState{GroupID: result.GroupID, LatestStatus: result.Status, StableStatus: result.Status}, r.event, nil
}

type recordingGroupStatusNotifier struct {
	calls  int
	groups []*Group
	cfgs   []*GroupStatusConfig
	events []*GroupStatusEvent
}

func (n *recordingGroupStatusNotifier) NotifyTransition(group *Group, cfg *GroupStatusConfig, event *GroupStatusEvent) {
	n.calls++
	n.groups = append(n.groups, group)
	n.cfgs = append(n.cfgs, cfg)
	n.events = append(n.events, event)
}

func TestGroupStatusProbeService_SaveProbeExecution_NotifiesOnTransition(t *testing.T) {
	event := groupStatusNotifyDownEvent()
	notifier := &recordingGroupStatusNotifier{}
	svc := &GroupStatusProbeService{repo: &groupStatusNotifyProbeRepo{event: event}}
	svc.SetTransitionNotifier(notifier)

	group := &Group{ID: 7, Name: "G7"}
	cfg := &GroupStatusConfig{ID: 3, GroupID: 7, NotifyEnabled: true, SlowLatencyMS: 15000}
	execution, err := svc.saveProbeExecution(context.Background(), group, cfg, nil, &GroupStatusProbeResult{Status: GroupRuntimeStatusDown})
	require.NoError(t, err)
	require.Same(t, event, execution.Event)

	require.Equal(t, 1, notifier.calls)
	require.Same(t, group, notifier.groups[0])
	require.Same(t, cfg, notifier.cfgs[0])
	require.Same(t, event, notifier.events[0])
}

func TestGroupStatusProbeService_SaveProbeExecution_NoEventNoNotify(t *testing.T) {
	notifier := &recordingGroupStatusNotifier{}
	svc := &GroupStatusProbeService{repo: &groupStatusNotifyProbeRepo{event: nil}}
	svc.SetTransitionNotifier(notifier)

	cfg := &GroupStatusConfig{ID: 3, GroupID: 7, NotifyEnabled: true, SlowLatencyMS: 15000}
	_, err := svc.saveProbeExecution(context.Background(), &Group{ID: 7}, cfg, nil, &GroupStatusProbeResult{Status: GroupRuntimeStatusUp})
	require.NoError(t, err)
	require.Equal(t, 0, notifier.calls)
}

func TestGroupStatusProbeService_SaveProbeExecution_WithoutNotifier(t *testing.T) {
	svc := &GroupStatusProbeService{repo: &groupStatusNotifyProbeRepo{event: groupStatusNotifyDownEvent()}}

	cfg := &GroupStatusConfig{ID: 3, GroupID: 7, SlowLatencyMS: 15000}
	execution, err := svc.saveProbeExecution(context.Background(), &Group{ID: 7}, cfg, nil, &GroupStatusProbeResult{Status: GroupRuntimeStatusDown})
	require.NoError(t, err)
	require.NotNil(t, execution.Event)
}

// ---------- notify_enabled 配置 ----------

func TestNormalizeGroupStatusConfig_NotifyEnabled(t *testing.T) {
	group := &Group{ID: 7, Platform: PlatformAnthropic}
	base := GroupStatusConfigUpsertInput{
		Enabled:        true,
		ProbeModel:     "claude-sonnet-4-5",
		ProbePrompt:    "ping",
		ValidationMode: GroupStatusValidationNonEmpty,
	}

	cfg, err := NormalizeGroupStatusConfig(group, &base)
	require.NoError(t, err)
	require.True(t, cfg.NotifyEnabled, "nil → 默认开启")

	off := base
	disabled := false
	off.NotifyEnabled = &disabled
	cfg, err = NormalizeGroupStatusConfig(group, &off)
	require.NoError(t, err)
	require.False(t, cfg.NotifyEnabled)

	require.True(t, DefaultGroupStatusConfig(group).NotifyEnabled)
}
