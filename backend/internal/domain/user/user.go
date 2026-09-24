// Package user は利用者のドメインモデルです。
// 認証そのものは Clerk に委譲し、こちらは Clerk のユーザーIDと
// アプリ内のプラン・上限だけを持ちます。
package user

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("user: 見つかりません")

// Planは課金プランです（v3で有料化導入予定）
type Plan string

const (
	PlanFree    Plan = "free"
	PLanPremium Plan = "premium"
)

// DefaultWordLimitはプランごとの登録上限です
func DefaultWordLimit(p Plan) int {
	if p == PLanPremium {
		return 10000
	}
	return 100
}

// Userは利用者です
type User struct {
	ID          uuid.UUID
	ClerkUserID string
	Email       string
	DisplayName string
	Plan        Plan
	WordLimit   int
	CreatedAt   time.Time
	UpdateAt    time.Time
}

// NewはClerkの情報から利用者を作ります
func New(clerkUserID, email, displayName string) *User {
	return &User{
		ID:          uuid.New(),
		ClerkUserID: clerkUserID,
		Email:       email,
		DisplayName: displayName,
		Plan:        PlanFree,
		WordLimit:   DefaultWordLimit(PlanFree),
	}
}

// CanRegisterMoreは現在の登録数から、さらに登録できるかを返します
func (u *User) CanRegisterMore(current int) bool { return current < u.WordLimit }

// Repositoryは利用者の永続化境界です
type Repository interface {
	// EnsureByClerkIDはClerkのユーザーが未登録なら作成し、既存なら取得します
	// (初回ログイン時のプロビジョニング)
	EnsureByClerkID(ctx context.Context, clerkUserID, email, displayName string) (*User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
}
