// Package registration は「候補を選んで単語帳に登録する」ユースケースです。
package registration

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/hangulme/hangul-me/backend/internal/domain/entry"
	"github.com/hangulme/hangul-me/backend/internal/domain/user"
	"github.com/hangulme/hangul-me/backend/internal/domain/userword"
	"github.com/hangulme/hangul-me/backend/internal/usecase/lookup"
)

// Input は登録リクエストです。
//
// EntryID が指定されていれば既存の辞書エントリを登録します。
// nil の場合はAI生成候補とみなし、先に辞書エントリを作ってから登録します。
type Input struct {
	UserID  uuid.UUID
	EntryID *uuid.UUID

	// AI生成候補を登録する場合に必要な値
	Hangul          string
	ReadingKana     string
	ReadingHiragana string
	Romanized       string
	MeaningJA       string

	// ユーザー入力
	Memo             string
	EncounteredTitle string

	// ログ用（どの検索から登録されたか）
	RawQuery   string
	Normalized string
	MatchedVia string
}

var ErrInvalidInput = errors.New("registration: 登録に必要な情報が不足しています")

// UseCase は単語登録のユースケースです。
type UseCase struct {
	users     user.Repository
	entries   entry.Repository
	userWords userword.Repository
	logs      lookup.SearchLogRepository
	now       func() time.Time
}

func New(
	users user.Repository,
	entries entry.Repository,
	userWords userword.Repository,
	logs lookup.SearchLogRepository,
) *UseCase {
	return &UseCase{
		users:     users,
		entries:   entries,
		userWords: userWords,
		logs:      logs,
		now:       time.Now,
	}
}

// Register は候補を単語帳に登録し、登録された単語を返します。
//
// 手順：
//  1. プランの上限チェック
//  2. AI生成候補なら辞書エントリを先に作る（既知なら再利用）
//  3. 単語帳に登録（同じ単語の二重登録は弾く）
//  4. 検索ログを1件残す（選択された候補のみ）
func (u *UseCase) Register(ctx context.Context, in Input) (*userword.UserWord, error) {
	usr, err := u.users.FindByID(ctx, in.UserID)
	if err != nil {
		return nil, err
	}

	count, err := u.userWords.CountByUser(ctx, in.UserID)
	if err != nil {
		return nil, err
	}
	if !usr.CanRegisterMore(count) {
		return nil, userword.ErrLimitExceeded
	}

	target, err := u.resolveEntry(ctx, in)
	if err != nil {
		return nil, err
	}

	exists, err := u.userWords.ExistsByEntry(ctx, in.UserID, target.ID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, userword.ErrAlreadyExists
	}

	w := userword.New(in.UserID, target.ID, in.Memo, in.EncounteredTitle)
	w.RegisteredAt = u.now()
	w.UpdatedAt = w.RegisteredAt
	if err := u.userWords.Create(ctx, w); err != nil {
		return nil, err
	}
	w.Entry = target

	u.writeSearchLog(ctx, in, target.ID)
	return w, nil
}

// resolveEntry は登録対象の辞書エントリを確定します。
func (u *UseCase) resolveEntry(ctx context.Context, in Input) (*entry.Entry, error) {
	if in.EntryID != nil {
		return u.entries.FindByID(ctx, *in.EntryID)
	}

	if in.Hangul == "" || in.ReadingKana == "" || in.MeaningJA == "" {
		return nil, ErrInvalidInput
	}

	// AI候補でも、登録の瞬間にもう一度DBと照合する（重複辞書を作らない）
	if known, err := u.entries.FindByHangulAndMeaning(ctx, in.Hangul, in.MeaningJA); err == nil && known != nil {
		return known, nil
	} else if err != nil && !errors.Is(err, entry.ErrNotFound) {
		return nil, err
	}

	e, err := entry.New(
		in.Hangul, in.ReadingKana, in.ReadingHiragana, in.Romanized, in.MeaningJA, entry.SourceAIGenerated,
	)
	if err != nil {
		return nil, err
	}
	if err := u.entries.Create(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

// writeSearchLog は検索ログを残します。
// 失敗しても登録自体は成功扱いにします（ログは補助的な情報のため）。
func (u *UseCase) writeSearchLog(ctx context.Context, in Input, entryID uuid.UUID) {
	if u.logs == nil || in.RawQuery == "" {
		return
	}

	via := in.MatchedVia
	if via == "" {
		via = "db_trgm"
	}
	userID := in.UserID
	eid := entryID
	_ = u.logs.Create(ctx, &lookup.SearchLog{
		ID:              uuid.New(),
		UserID:          &userID,
		RawInput:        in.RawQuery,
		NormalizedInput: in.Normalized,
		MatchedVia:      via,
		SelectedEntryID: &eid,
		CreatedAt:       u.now(),
	})
}
