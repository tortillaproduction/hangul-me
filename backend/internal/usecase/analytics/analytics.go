// Package analytics は分析ビュー（グラフ）のユースケースです。
package analytics

import (
	"context"

	"github.com/google/uuid"
	domain "github.com/hangulme/hangul-me/backend/internal/domain/analytics"
)

// UseCase は集計の取得を担います。
type UseCase struct {
	repo domain.Repository
}

func New(repo domain.Repository) *UseCase {
	return &UseCase{
		repo: repo,
	}
}

// Overview はユーザー自身の集計一式を返します。
func (u *UseCase) Overview(ctx context.Context, userID uuid.UUID) (*domain.Overview, error) {
	return u.repo.OverviewForUser(ctx, userID)
}

// GlobalMostSearched はアプリ全体でよく調べられた単語トップNを返します。
func (u *UseCase) GlobalMostSearched(ctx context.Context, limit int) ([]domain.RankedWord, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	return u.repo.GlobalMostSearched(ctx, limit)
}
