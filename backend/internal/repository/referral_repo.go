package repository

import (
	"context"
	"errors"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/userreferral"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type referralRepository struct {
	client *dbent.Client
}

func NewReferralRepository(client *dbent.Client) service.ReferralRepository {
	return &referralRepository{client: client}
}

func (r *referralRepository) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if fn == nil {
		return nil
	}

	if dbent.TxFromContext(ctx) != nil {
		return fn(ctx)
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		if errors.Is(err, dbent.ErrTxStarted) {
			return fn(ctx)
		}
		return err
	}
	defer func() { _ = tx.Rollback() }()

	txCtx := dbent.NewTxContext(ctx, tx)
	if err := fn(txCtx); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *referralRepository) Create(ctx context.Context, ref *service.UserReferral) error {
	client := clientFromContext(ctx, r.client)
	created, err := client.UserReferral.Create().
		SetReferrerID(ref.ReferrerID).
		SetRefereeID(ref.RefereeID).
		SetStatus(ref.Status).
		Save(ctx)
	if err != nil {
		return translatePersistenceError(err, nil, nil)
	}
	ref.ID = created.ID
	ref.CreatedAt = created.CreatedAt
	ref.UpdatedAt = created.UpdatedAt
	return nil
}

func (r *referralRepository) GetByRefereeID(ctx context.Context, refereeID int64) (*service.UserReferral, error) {
	client := clientFromContext(ctx, r.client)
	m, err := client.UserReferral.Query().
		Where(userreferral.RefereeIDEQ(refereeID)).
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrReferralNotFound, nil)
	}
	return referralEntityToService(m), nil
}

func (r *referralRepository) GetByID(ctx context.Context, id int64) (*service.UserReferral, error) {
	client := clientFromContext(ctx, r.client)
	m, err := client.UserReferral.Query().
		Where(userreferral.IDEQ(id)).
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrReferralNotFound, nil)
	}
	return referralEntityToService(m), nil
}

func (r *referralRepository) UpdateStatus(ctx context.Context, id int64, status string, rewards *service.ReferralRewardSnapshot) error {
	client := clientFromContext(ctx, r.client)
	now := time.Now()
	update := client.UserReferral.UpdateOneID(id).
		SetStatus(status).
		SetUpdatedAt(now)

	if rewards != nil {
		update = update.
			SetReferrerBalanceReward(rewards.ReferrerBalanceReward).
			SetReferrerSubscriptionDays(rewards.ReferrerSubscriptionDays).
			SetRefereeBalanceReward(rewards.RefereeBalanceReward).
			SetRefereeSubscriptionDays(rewards.RefereeSubscriptionDays)

		if rewards.ReferrerGroupID != nil {
			update = update.SetReferrerGroupID(*rewards.ReferrerGroupID)
		}
		if rewards.RefereeGroupID != nil {
			update = update.SetRefereeGroupID(*rewards.RefereeGroupID)
		}
		if status == "rewarded" {
			update = update.
				SetReferrerRewardedAt(now).
				SetRefereeRewardedAt(now)
		}
	}

	_, err := update.Save(ctx)
	return translatePersistenceError(err, service.ErrReferralNotFound, nil)
}

func (r *referralRepository) CountByReferrerID(ctx context.Context, referrerID int64) (int, error) {
	client := clientFromContext(ctx, r.client)
	count, err := client.UserReferral.Query().
		Where(userreferral.ReferrerIDEQ(referrerID)).
		Count(ctx)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *referralRepository) CountByReferrerIDAndStatus(ctx context.Context, referrerID int64, status string) (int, error) {
	client := clientFromContext(ctx, r.client)
	count, err := client.UserReferral.Query().
		Where(
			userreferral.ReferrerIDEQ(referrerID),
			userreferral.StatusEQ(status),
		).
		Count(ctx)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *referralRepository) SumReferrerBalanceReward(ctx context.Context, referrerID int64) (float64, error) {
	client := clientFromContext(ctx, r.client)
	var result []struct {
		Sum float64 `json:"sum"`
	}
	err := client.UserReferral.Query().
		Where(
			userreferral.ReferrerIDEQ(referrerID),
			userreferral.StatusEQ("rewarded"),
		).
		Aggregate(dbent.Sum(userreferral.FieldReferrerBalanceReward)).
		Scan(ctx, &result)
	if err != nil {
		return 0, err
	}
	if len(result) == 0 {
		return 0, nil
	}
	return result[0].Sum, nil
}

func (r *referralRepository) ListByReferrerID(ctx context.Context, referrerID int64, params pagination.PaginationParams) ([]service.UserReferral, *pagination.PaginationResult, error) {
	client := clientFromContext(ctx, r.client)
	query := client.UserReferral.Query().
		Where(userreferral.ReferrerIDEQ(referrerID))

	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	paginationResult := paginationResultFromTotal(int64(total), params)

	items, err := query.
		WithReferee().
		Order(dbent.Desc(userreferral.FieldCreatedAt)).
		Offset(params.Offset()).
		Limit(params.Limit()).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	out := make([]service.UserReferral, 0, len(items))
	for _, m := range items {
		ref := referralEntityToService(m)
		if m.Edges.Referee != nil {
			ref.RefereeEmail = maskEmail(m.Edges.Referee.Email)
		}
		out = append(out, *ref)
	}
	return out, paginationResult, nil
}

func (r *referralRepository) ListAll(ctx context.Context, params pagination.PaginationParams) ([]service.UserReferral, *pagination.PaginationResult, error) {
	client := clientFromContext(ctx, r.client)
	query := client.UserReferral.Query()

	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, nil, err
	}

	paginationResult := paginationResultFromTotal(int64(total), params)

	items, err := query.
		WithReferrer().
		WithReferee().
		Order(dbent.Desc(userreferral.FieldCreatedAt)).
		Offset(params.Offset()).
		Limit(params.Limit()).
		All(ctx)
	if err != nil {
		return nil, nil, err
	}

	out := make([]service.UserReferral, 0, len(items))
	for _, m := range items {
		ref := referralEntityToService(m)
		if m.Edges.Referrer != nil {
			ref.ReferrerEmail = m.Edges.Referrer.Email
		}
		if m.Edges.Referee != nil {
			ref.RefereeEmail = m.Edges.Referee.Email
		}
		out = append(out, *ref)
	}
	return out, paginationResult, nil
}

func maskEmail(email string) string {
	at := -1
	for i, c := range email {
		if c == '@' {
			at = i
			break
		}
	}
	if at <= 0 {
		return "***"
	}
	if at <= 2 {
		return email[:1] + "***" + email[at:]
	}
	return email[:2] + "***" + email[at:]
}

func referralEntityToService(m *dbent.UserReferral) *service.UserReferral {
	if m == nil {
		return nil
	}
	return &service.UserReferral{
		ID:                       m.ID,
		ReferrerID:               m.ReferrerID,
		RefereeID:                m.RefereeID,
		Status:                   m.Status,
		ReferrerBalanceReward:    m.ReferrerBalanceReward,
		ReferrerGroupID:          m.ReferrerGroupID,
		ReferrerSubscriptionDays: m.ReferrerSubscriptionDays,
		ReferrerRewardedAt:       m.ReferrerRewardedAt,
		RefereeBalanceReward:     m.RefereeBalanceReward,
		RefereeGroupID:           m.RefereeGroupID,
		RefereeSubscriptionDays:  m.RefereeSubscriptionDays,
		RefereeRewardedAt:        m.RefereeRewardedAt,
		CreatedAt:                m.CreatedAt,
		UpdatedAt:                m.UpdatedAt,
	}
}
