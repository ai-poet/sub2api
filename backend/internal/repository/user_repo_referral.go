package repository

import (
	"context"

	dbuser "github.com/Wei-Shaw/sub2api/ent/user"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// SetReferralCodeIfEmpty 仅当用户当前 referral_code 为空时写入，返回是否写入。
// WHERE 谓词是对读取路径的兜底：即使 userEntityToService 的 ReferralCode 映射
// 再次在上游合并中丢失（服务层误判"无码"），也不会覆盖库中已有的码。
// 谓词范围与迁移 053 的 partial unique index 一致（非空码 + 未删除）。
// 唯一索引冲突（码撞车）映射为 service.ErrReferralCodeTaken，供调用方换码重试。
func (r *userRepository) SetReferralCodeIfEmpty(ctx context.Context, userID int64, code string) (bool, error) {
	client := clientFromContext(ctx, r.client)
	n, err := client.User.Update().
		Where(dbuser.IDEQ(userID), dbuser.ReferralCodeEQ(""), dbuser.DeletedAtIsNil()).
		SetReferralCode(code).
		Save(ctx)
	if err != nil {
		return false, translatePersistenceError(err, service.ErrUserNotFound, service.ErrReferralCodeTaken)
	}
	return n > 0, nil
}

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
