package repository

import (
	"context"

	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// GetByReferralCode 通过推荐码查找用户
func (r *userRepository) GetByReferralCode(ctx context.Context, code string) (*service.User, error) {
	client := clientFromContext(ctx, r.client)
	m, err := client.User.Query().Where(dbuser.ReferralCodeEQ(code)).Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrUserNotFound, nil)
	}

	out := userEntityToService(m)
	groups, err := r.loadAllowedGroups(ctx, []int64{m.ID})
	if err != nil {
		return nil, err
	}
	if v, ok := groups[m.ID]; ok {
		out.AllowedGroups = v
	}
	return out, nil
}
