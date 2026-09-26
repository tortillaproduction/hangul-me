// Package wordbook は単語帳（登録単語一覧・個別ページ）のユースケースです。
package wordbook

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hangulme/hangul-me/backend/internal/domain/entry"
	"github.com/hangulme/hangul-me/backend/internal/domain/userword"
)

// Usecase は単語帳の参照・更新を担います。
type Usecase struct {
	userWords userword.Repository
	entries   entry.Repository
	views     userword.ViewRecorder
	now       func() time.Time
}

func New(userWords userword.Repository, entries entry.Repository, views userword.ViewRecorder) *Usecase {
	return &Usecase{
		userWords: userWords,
		entries:   entries,
		views:     views,
		now:       time.Now,
	}
}

// List は登録単語一覧を返します。
func (u *Usecase) List(ctx context.Context, userID uuid.UUID, f userword.ListFilter) ([]*userword.UserWord, error) {
	if f.Limit <= 0 || f.Limit > 200 {
		f.Limit = 50
	}
	return u.userWords.List(ctx, userID, f)
}

// Detail は単語の個別ページ用データを返し、閲覧を1件記録します。
//
// 閲覧の記録は生ログ（word_view_logs）に積み、
// 日次バッチで user_word_daily_views に丸めて90日より古い生ログは消します。
// 記録に失敗しても閲覧自体は成功として返します。
func (u *Usecase) Detail(ctx context.Context, userID, id uuid.UUID) (*userword.UserWord, error) {
	w, err := u.userWords.FindByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	if w.Entry != nil {
		ex, err := u.entries.ListExamples(ctx, w.EntryID)
		if err != nil {
			return nil, err
		}
		w.Entry.Examples = ex
	}
	if u.views != nil {
		_ = u.views.RecordView(ctx, w.ID, u.now())
	}

	return w, nil
}

// UpdateInput は単語帳の編集内容です。nil のフィールドは変更しません。
type UpdateInput struct {
	Memo             *string
	EncounteredTitle *string
	MasteryLevel     *userword.MasteryLevel
}

// Update はメモ・「どこで聞いた？」・定着度を更新します。
func (u *Usecase) Update(ctx context.Context, userID, id uuid.UUID, in UpdateInput) (*userword.UserWord, error) {
	w, err := u.userWords.FindByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	if in.Memo != nil {
		w.Memo = *in.Memo
	}
	if in.EncounteredTitle != nil {
		w.EncounteredTitle = *in.EncounteredTitle
	}
	if in.MasteryLevel != nil && in.MasteryLevel.Valid() {
		w.MasteryLevel = *in.MasteryLevel
	}

	w.UpdatedAt = u.now()
	if err := u.userWords.Update(ctx, w); err != nil {
		return nil, err
	}

	return w, nil
}

// Delete は単語帳から削除します。
func (u *Usecase) Delete(ctx context.Context, userID, id uuid.UUID) error {
	return u.userWords.Delete(ctx, userID, id)
}
