// Package analytics は分析ビュー（グラフ）のユースケースです。
package analytics

import (
	"context"

	"github.com/google/uuid"
	domain "github.com/hangulme/hangul-me/backend/internal/domain/analytics"
)

// Usecase は集計の取得を担います。
type Usecase struct {
	repo domain.Repository
}

func New(repo domain.Repository) *Usecase {
	return &Usecase{
		repo: repo,
	}
}

// Overview はユーザー自身の集計一式を返します。
func (u *Usecase) Overview(ctx context.Context, userID uuid.UUID) (*domain.Overview, error) {
	return u.repo.OverviewForUser(ctx, userID)
}

// GlobalMostSearched はアプリ全体でよく調べられた単語トップNを返します。
func (u *Usecase) GlobalMostSearched(ctx context.Context, limit int) ([]domain.RankedWord, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	return u.repo.GlobalMostSearched(ctx, limit)
}
