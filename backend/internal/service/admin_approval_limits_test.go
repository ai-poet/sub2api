package service

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// approvalLimitSettingRepoStub 只实现 Limits 用到的 GetMultiple；其余方法占位。
type approvalLimitSettingRepoStub struct {
	values map[string]string
	err    error
}

func (r *approvalLimitSettingRepoStub) Get(context.Context, string) (*Setting, error) {
	return nil, errors.New("unused")
}

func (r *approvalLimitSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	if v, ok := r.values[key]; ok {
		return v, nil
	}
	return "", errors.New("not found")
}

func (r *approvalLimitSettingRepoStub) Set(context.Context, string, string) error { return nil }

func (r *approvalLimitSettingRepoStub) GetMultiple(context.Context, []string) (map[string]string, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.values, nil
}

func (r *approvalLimitSettingRepoStub) SetMultiple(context.Context, map[string]string) error {
	return nil
}

func (r *approvalLimitSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	return r.values, nil
}

func (r *approvalLimitSettingRepoStub) Delete(context.Context, string) error { return nil }

func TestParseApprovalLimits(t *testing.T) {
	def := DefaultApprovalLimits()
	require.Equal(t, ApprovalLimits{PendingPerUser: AdminApprovalPendingLimitPerUser, Batch: AdminApprovalBatchLimit}, def)

	// 缺失 / 空白 → 默认
	require.Equal(t, def, ParseApprovalLimits(nil))
	require.Equal(t, def, ParseApprovalLimits(map[string]string{
		SettingKeyApprovalPendingLimitPerUser: "",
		SettingKeyApprovalBatchLimit:          "  ",
	}))

	// 合法值（允许首尾空白）
	require.Equal(t, ApprovalLimits{PendingPerUser: 3, Batch: 7}, ParseApprovalLimits(map[string]string{
		SettingKeyApprovalPendingLimitPerUser: " 3 ",
		SettingKeyApprovalBatchLimit:          "7",
	}))
	require.Equal(t, ApprovalLimits{PendingPerUser: AdminApprovalPendingLimitMax, Batch: AdminApprovalBatchLimitMax}, ParseApprovalLimits(map[string]string{
		SettingKeyApprovalPendingLimitPerUser: "1000",
		SettingKeyApprovalBatchLimit:          "500",
	}))

	// 非法 / 越界各自回退默认，互不影响
	require.Equal(t, ApprovalLimits{PendingPerUser: def.PendingPerUser, Batch: 9}, ParseApprovalLimits(map[string]string{
		SettingKeyApprovalPendingLimitPerUser: "0",
		SettingKeyApprovalBatchLimit:          "9",
	}))
	require.Equal(t, def, ParseApprovalLimits(map[string]string{
		SettingKeyApprovalPendingLimitPerUser: "abc",
		SettingKeyApprovalBatchLimit:          "501",
	}))
	require.Equal(t, def, ParseApprovalLimits(map[string]string{
		SettingKeyApprovalPendingLimitPerUser: "1001",
		SettingKeyApprovalBatchLimit:          "-1",
	}))

	require.True(t, ValidateApprovalLimit(1, AdminApprovalBatchLimitMax))
	require.True(t, ValidateApprovalLimit(AdminApprovalBatchLimitMax, AdminApprovalBatchLimitMax))
	require.False(t, ValidateApprovalLimit(0, AdminApprovalBatchLimitMax))
	require.False(t, ValidateApprovalLimit(AdminApprovalBatchLimitMax+1, AdminApprovalBatchLimitMax))
}

// 上限跟随站点设置：未挂载 / 读取失败回退默认；改设置后下一次入队 / 批量通过立即按新值执行。
func TestAdminApprovalService_LimitsFromSettings(t *testing.T) {
	repo := newApprovalRepoStub()
	svc := newApprovalServiceForTest(repo)
	ctx := context.Background()

	require.Equal(t, DefaultApprovalLimits(), svc.Limits(ctx), "未挂载设置仓储时用默认常量")
	var nilSvc *AdminApprovalService
	require.Equal(t, DefaultApprovalLimits(), nilSvc.Limits(ctx))

	svc.SetSettingRepository(&approvalLimitSettingRepoStub{err: errors.New("db down")})
	require.Equal(t, DefaultApprovalLimits(), svc.Limits(ctx), "设置表读取失败不能把审批锁死")

	settings := &approvalLimitSettingRepoStub{values: map[string]string{
		SettingKeyApprovalPendingLimitPerUser: "2",
		SettingKeyApprovalBatchLimit:          "1",
	}}
	svc.SetSettingRepository(settings)
	require.Equal(t, ApprovalLimits{PendingPerUser: 2, Batch: 1}, svc.Limits(ctx))

	// 每人待审上限按设置生效：第 3 条被拒，调大后立即放行
	svc.createLimit = newApprovalCreateLimiter(1000, time.Minute)
	first, err := svc.Capture(ctx, balanceCapture(`{"balance":1,"operation":"add"}`))
	require.NoError(t, err)
	second, err := svc.Capture(ctx, balanceCapture(`{"balance":1,"operation":"add"}`))
	require.NoError(t, err)
	_, err = svc.Capture(ctx, balanceCapture(`{"balance":1,"operation":"add"}`))
	require.ErrorIs(t, err, ErrApprovalPendingLimit)
	settings.values[SettingKeyApprovalPendingLimitPerUser] = "3"
	_, err = svc.Capture(ctx, balanceCapture(`{"balance":1,"operation":"add"}`))
	require.NoError(t, err)

	// 批量上限按设置生效（校验在审批人与 dispatcher 检查之后）
	svc.SetDispatcher(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"data":{}}`))
	}))
	_, err = svc.ApproveBatch(ctx, []int64{first.ID, second.ID}, adminActor)
	require.ErrorIs(t, err, ErrApprovalBatchTooLarge)
	settings.values[SettingKeyApprovalBatchLimit] = "2"
	results, err := svc.ApproveBatch(ctx, []int64{first.ID, second.ID}, adminActor)
	require.NoError(t, err)
	require.Len(t, results, 2)
}
