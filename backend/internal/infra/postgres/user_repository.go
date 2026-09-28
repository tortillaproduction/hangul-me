package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/hangulme/hangul-me/backend/internal/domain/user"
)

// UserRepository は users のリポジトリです。
type UserRepository struct{ db *sql.DB }

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

const userColumns = `id, clerk_user_id, email, display_name, plan, word_limit, created_at, updated_at`

func scanUser(row scannable) (*user.User, error) {
	var u user.User
	var plan string

	if err := row.Scan(&u.ID, &u.ClerkUserID, &u.Email, &u.DisplayName, &plan, &u.WordLimit, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, err
	}
	u.Plan = user.Plan(plan)
	return &u, nil
}

// EnsureByClerkID は初回ログイン時にユーザーを作り、既存なら取得します。
// メール・表示名は Clerk 側を正として毎回同期します。
func (r *UserRepository) EnsureByClerkID(ctx context.Context, ClerkUserID, email, displayName string) (*user.User, error) {
	const q = `
		INSERT INTO users (id, clerk_user_id, email, display_name, plan, word_limit)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (clerk_user_id) DO UPDATE
			SET email = EXCLUDED.email,
				display_name = EXCLUDED.display_name,
				updated_at = now()
		RETURNING ` + userColumns

	u := user.New(ClerkUserID, email, displayName)
	row := r.db.QueryRowContext(ctx, q, u.ID, u.ClerkUserID, u.Email, u.DisplayName, string(u.Plan), u.WordLimit)
	got, err := scanUser(row)
	if err != nil {
		return nil, fmt.Errorf("user: プロビジョニングに失敗しました: %w", err)
	}
	return got, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id)
	u, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, user.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("user: 取得に失敗しました: %w", err)
	}
	return u, nil
}

// isUniqueViolation は一意制約違反かを判定します（PostgreSQL 23505）。
func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}
