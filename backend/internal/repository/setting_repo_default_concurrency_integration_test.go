//go:build integration

package repository

import (
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (s *SettingRepoSuite) TestSetMultipleWithDefaultConcurrencyUpdatesOnlyFollowers() {
	follower, err := s.repo.client.User.Create().
		SetEmail("default-follower@example.com").
		SetPasswordHash("hash").
		SetConcurrency(5).
		SetConcurrencyIsDefault(true).
		Save(s.ctx)
	s.Require().NoError(err)
	custom, err := s.repo.client.User.Create().
		SetEmail("default-custom@example.com").
		SetPasswordHash("hash").
		SetConcurrency(7).
		SetConcurrencyIsDefault(false).
		Save(s.ctx)
	s.Require().NoError(err)
	unlimited, err := s.repo.client.User.Create().
		SetEmail("default-unlimited@example.com").
		SetPasswordHash("hash").
		SetConcurrency(0).
		SetConcurrencyIsDefault(false).
		Save(s.ctx)
	s.Require().NoError(err)

	err = s.repo.SetMultipleWithDefaultConcurrency(s.ctx, map[string]string{
		service.SettingKeyDefaultConcurrency: "9",
		"unrelated_setting":                  "kept",
	})
	s.Require().NoError(err)

	storedDefault, err := s.repo.GetValue(s.ctx, service.SettingKeyDefaultConcurrency)
	s.Require().NoError(err)
	s.Require().Equal("9", storedDefault)
	storedOther, err := s.repo.GetValue(s.ctx, "unrelated_setting")
	s.Require().NoError(err)
	s.Require().Equal("kept", storedOther)

	follower, err = s.repo.client.User.Get(s.ctx, follower.ID)
	s.Require().NoError(err)
	s.Require().Equal(9, follower.Concurrency)
	s.Require().True(follower.ConcurrencyIsDefault)
	custom, err = s.repo.client.User.Get(s.ctx, custom.ID)
	s.Require().NoError(err)
	s.Require().Equal(7, custom.Concurrency)
	s.Require().False(custom.ConcurrencyIsDefault)
	unlimited, err = s.repo.client.User.Get(s.ctx, unlimited.ID)
	s.Require().NoError(err)
	s.Require().Zero(unlimited.Concurrency)
	s.Require().False(unlimited.ConcurrencyIsDefault)
}

func (s *SettingRepoSuite) TestSetMultipleWithDefaultConcurrencyAutoFollowAdoptsOnlyPreviousDefaultUsers() {
	legacy, err := s.repo.client.User.Create().
		SetEmail("legacy-default@example.com").
		SetPasswordHash("hash").
		SetConcurrency(5).
		SetConcurrencyIsDefault(false).
		Save(s.ctx)
	s.Require().NoError(err)
	custom, err := s.repo.client.User.Create().
		SetEmail("auto-follow-custom@example.com").
		SetPasswordHash("hash").
		SetConcurrency(7).
		SetConcurrencyIsDefault(false).
		Save(s.ctx)
	s.Require().NoError(err)
	unlimited, err := s.repo.client.User.Create().
		SetEmail("auto-follow-unlimited@example.com").
		SetPasswordHash("hash").
		SetConcurrency(0).
		SetConcurrencyIsDefault(false).
		Save(s.ctx)
	s.Require().NoError(err)
	s.Require().NoError(s.repo.SetMultiple(s.ctx, map[string]string{service.SettingKeyDefaultConcurrency: "5"}))

	s.Require().NoError(s.repo.SetMultipleWithDefaultConcurrency(s.ctx, map[string]string{
		service.SettingKeyDefaultConcurrency:           "9",
		service.SettingKeyDefaultConcurrencyAutoFollow: "true",
	}))

	legacy, err = s.repo.client.User.Get(s.ctx, legacy.ID)
	s.Require().NoError(err)
	s.Require().Equal(9, legacy.Concurrency)
	s.Require().True(legacy.ConcurrencyIsDefault)
	custom, err = s.repo.client.User.Get(s.ctx, custom.ID)
	s.Require().NoError(err)
	s.Require().Equal(7, custom.Concurrency)
	s.Require().False(custom.ConcurrencyIsDefault)
	unlimited, err = s.repo.client.User.Get(s.ctx, unlimited.ID)
	s.Require().NoError(err)
	s.Require().Zero(unlimited.Concurrency)
	s.Require().False(unlimited.ConcurrencyIsDefault)
}
