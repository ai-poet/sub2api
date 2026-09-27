//go:build unit

package testutil

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// 编译期接口断言
var _ service.PersonalTokenRepository = (*MemoryPersonalTokenRepo)(nil)

// MemoryPersonalTokenRepo 是 service.PersonalTokenRepository 的内存实现（按 user_id 唯一，语义同原生 SQL 仓储）。
type MemoryPersonalTokenRepo struct {
	mu     sync.Mutex
	nextID int64
	byUser map[int64]*service.PersonalToken
	// Touches 记录 TouchLastUsed 真正写入的次数（节流生效时不增加）。
	Touches int
}

// NewMemoryPersonalTokenRepo 构造空仓储。
func NewMemoryPersonalTokenRepo() *MemoryPersonalTokenRepo {
	return &MemoryPersonalTokenRepo{byUser: map[int64]*service.PersonalToken{}}
}

func clonePersonalToken(t *service.PersonalToken) *service.PersonalToken {
	if t == nil {
		return nil
	}
	out := *t
	if t.ExpiresAt != nil {
		v := *t.ExpiresAt
		out.ExpiresAt = &v
	}
	if t.LastUsedAt != nil {
		v := *t.LastUsedAt
		out.LastUsedAt = &v
	}
	return &out
}

func (r *MemoryPersonalTokenRepo) Upsert(_ context.Context, token *service.PersonalToken) (*service.PersonalToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	saved := clonePersonalToken(token)
	if existing, ok := r.byUser[token.UserID]; ok {
		saved.ID = existing.ID
	} else {
		r.nextID++
		saved.ID = r.nextID
	}
	saved.LastUsedAt = nil
	saved.LastUsedIP = ""
	if saved.CreatedAt.IsZero() {
		saved.CreatedAt = time.Now()
	}
	r.byUser[token.UserID] = saved
	return clonePersonalToken(saved), nil
}

func (r *MemoryPersonalTokenRepo) GetByHash(_ context.Context, tokenHash string) (*service.PersonalToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.byUser {
		if t.TokenHash == tokenHash {
			return clonePersonalToken(t), nil
		}
	}
	return nil, service.ErrPersonalTokenNotFound
}

func (r *MemoryPersonalTokenRepo) GetByUserID(_ context.Context, userID int64) (*service.PersonalToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if t, ok := r.byUser[userID]; ok {
		return clonePersonalToken(t), nil
	}
	return nil, service.ErrPersonalTokenNotFound
}

func (r *MemoryPersonalTokenRepo) DeleteByUserID(_ context.Context, userID int64) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.byUser[userID]
	delete(r.byUser, userID)
	return ok, nil
}

func (r *MemoryPersonalTokenRepo) List(_ context.Context) ([]*service.PersonalToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*service.PersonalToken, 0, len(r.byUser))
	for _, t := range r.byUser {
		out = append(out, clonePersonalToken(t))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	return out, nil
}

func (r *MemoryPersonalTokenRepo) TouchLastUsed(_ context.Context, id int64, ip string, now time.Time, minInterval time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.byUser {
		if t.ID != id {
			continue
		}
		if t.LastUsedAt != nil && !t.LastUsedAt.Before(now.Add(-minInterval)) && t.LastUsedIP == ip {
			return nil
		}
		v := now
		t.LastUsedAt = &v
		t.LastUsedIP = ip
		r.Touches++
	}
	return nil
}

// StaticPersonalTokenSettings 个人令牌总开关的可变桩。
type StaticPersonalTokenSettings struct {
	mu      sync.Mutex
	enabled bool
}

// NewStaticPersonalTokenSettings 构造开关桩。
func NewStaticPersonalTokenSettings(enabled bool) *StaticPersonalTokenSettings {
	return &StaticPersonalTokenSettings{enabled: enabled}
}

// Set 切换开关。
func (s *StaticPersonalTokenSettings) Set(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enabled = enabled
}

// IsPersonalTokenEnabled 实现 service 侧的开关读取接口。
func (s *StaticPersonalTokenSettings) IsPersonalTokenEnabled(context.Context) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.enabled
}
