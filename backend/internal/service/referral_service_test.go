package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

var _ SettingRepository = (*stubSettingRepoForReferralService)(nil)

type stubSettingRepoForReferralService struct {
	values         map[string]string
	setMultipleErr error
}

func (r *stubSettingRepoForReferralService) Get(_ context.Context, key string) (*Setting, error) {
	if value, ok := r.values[key]; ok {
		return &Setting{Key: key, Value: value}, nil
	}
	return nil, ErrSettingNotFound
}

func (r *stubSettingRepoForReferralService) GetValue(_ context.Context, key string) (string, error) {
	if value, ok := r.values[key]; ok {
		return value, nil
	}
	return "", ErrSettingNotFound
}

func (r *stubSettingRepoForReferralService) Set(_ context.Context, key, value string) error {
	r.values[key] = value
	return nil
}

func (r *stubSettingRepoForReferralService) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		if value, ok := r.values[key]; ok {
			result[key] = value
		}
	}
	return result, nil
}

func (r *stubSettingRepoForReferralService) SetMultiple(_ context.Context, settings map[string]string) error {
	if r.setMultipleErr != nil {
		return r.setMultipleErr
	}

	for key, value := range settings {
		r.values[key] = value
	}

	return nil
}

func (r *stubSettingRepoForReferralService) GetAll(_ context.Context) (map[string]string, error) {
	result := make(map[string]string, len(r.values))
	for key, value := range r.values {
		result[key] = value
	}
	return result, nil
}

func (r *stubSettingRepoForReferralService) Delete(_ context.Context, key string) error {
	delete(r.values, key)
	return nil
}

func TestReferralService_UpdateReferralSettings_TriggersCallbackOnSuccess(t *testing.T) {
	settingRepo := &stubSettingRepoForReferralService{values: make(map[string]string)}
	referralService := NewReferralService(nil, nil, nil, settingRepo, nil)

	callbackCalled := 0
	referralService.SetOnSettingsUpdateCallback(func() {
		callbackCalled++
	})

	err := referralService.UpdateReferralSettings(context.Background(), &ReferralSettings{
		Enabled:                  true,
		ReferrerBalanceReward:    1.23,
		ReferrerGroupID:          10,
		ReferrerSubscriptionDays: 7,
		RefereeBalanceReward:     2.34,
		RefereeGroupID:           20,
		RefereeSubscriptionDays:  14,
		MaxPerUser:               99,
	})

	require.NoError(t, err)
	require.Equal(t, 1, callbackCalled)
	require.Equal(t, "true", settingRepo.values[SettingKeyReferralEnabled])
	require.Equal(t, "99", settingRepo.values[SettingKeyReferralMaxPerUser])
}

func TestReferralService_UpdateReferralSettings_DoesNotTriggerCallbackOnFailure(t *testing.T) {
	settingRepo := &stubSettingRepoForReferralService{
		values:         make(map[string]string),
		setMultipleErr: errors.New("set settings failed"),
	}
	referralService := NewReferralService(nil, nil, nil, settingRepo, nil)

	callbackCalled := 0
	referralService.SetOnSettingsUpdateCallback(func() {
		callbackCalled++
	})

	err := referralService.UpdateReferralSettings(context.Background(), &ReferralSettings{})

	require.Error(t, err)
	require.Equal(t, 0, callbackCalled)
}
