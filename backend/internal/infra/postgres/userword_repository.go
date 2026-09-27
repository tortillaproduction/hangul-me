package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hangulme/hangul-me/backend/internal/domain/entry"
	"github.com/hangulme/hangul-me/backend/internal/domain/userword"
)

// UserWordRepository は user_words / word_view_logs のリポジトリです。
type UserWordRepository struct{ db *sql.DB }

func NewUserWordRepository(db *sql.DB) *UserWordRepository {
	return &UserWordRepository{
		db: db,
	}
}

var (
	_ userword.Repository   = (*UserWordRepository)(nil)
	_ userword.ViewRecorder = (*UserWordRepository)(nil)
)

// 単語帳は常に辞書エントリと結合して返す（一覧・詳細どちらも表示に必要なため）
const userWordSelect = `
	SELECT uw.id, uw.user_id, uw.entry_id,
	       COALESCE(uw.memo, ''), COALESCE(uw.encountered_title, ''),
	       uw.mastery_level, uw.registered_at, uw.updated_at,
	       e.id, e.hangul, e.reading_kana, e.reading_hiragana, e.romanized, e.meaning_ja,
	       e.source, e.initial_consonant, e.has_final_consonant,
	       COALESCE(e.final_consonant, ''), COALESCE(e.audio_url, '')
	FROM user_words uw
	JOIN dictionary_entries e ON e.id = uw.entry_id`

func scanUserWord(row scannable) (*userword.UserWord, error) {
	var w userword.UserWord
	var e entry.Entry
	var src string
	var level int

	err := row.Scan(
		&w.ID, &w.UserID, &w.EntryID,
		&w.Memo, &w.EncounteredTitle,
		&level, &w.RegisteredAt, &w.UpdatedAt,
		&e.ID, &e.Hangul, &e.ReadingKana, &e.ReadingHiragana, &e.Romanized, &e.MeaningJA,
		&src, &e.InitialConsonant, &e.HasFinalConsonant, &e.FinalConsonant, &e.AudioURL,
	)
	if err != nil {
		return nil, err
	}

	e.Source = entry.Source(src)
	w.MasteryLevel = userword.MasteryLevel(level)
	w.Entry = &e
	return &w, nil
}

func (r *UserWordRepository) Create(ctx context.Context, w *userword.UserWord) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}

	const q = `
		INSERT INTO user_words
			(id, user_id, entry_id, memo, encountered_title, mastery_level, registered_at, updated_at)
		VALUES ($1,$2,$3, NULLIF($4,''), NULLIF($5,''), $6, $7, $8)
		RETURNING registered_at, updated_at`

	err := r.db.QueryRowContext(ctx, q,
		w.ID, w.UserID, w.EntryID, w.Memo, w.EncounteredTitle, int(w.MasteryLevel),
		nowOr(w.RegisteredAt), nowOr(w.UpdatedAt),
	).Scan(&w.RegisteredAt, &w.UpdatedAt)

	if err != nil {
		if isUniqueViolation(err) {
			return userword.ErrAlreadyExists
		}
		return fmt.Errorf("userword: 登録に失敗しました: %w", err)
	}
	return nil
}

func (r *UserWordRepository) FindByID(ctx context.Context, userID, id uuid.UUID) (*userword.UserWord, error) {
	row := r.db.QueryRowContext(ctx, userWordSelect+` WHERE uw.user_id = $1 AND uw.id = $2`, userID, id)
	w, err := scanUserWord(row)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, userword.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("userword: 取得に失敗しました: %w", err)
	}
	return w, nil
}

func (r *UserWordRepository) List(ctx context.Context, userID uuid.UUID, f userword.ListFilter) ([]*userword.UserWord, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}

	q := userWordSelect + `
		WHERE uw.user_id = $1
		  AND ($2 = '' OR uw.encountered_title = $2)
		ORDER BY uw.registered_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.db.QueryContext(ctx, q, userID, f.EncounteredTitle, f.Limit, f.Offset)
	if err != nil {
		return nil, fmt.Errorf("userword: 一覧取得に失敗しました: %w", err)
	}
	defer rows.Close()

	out := []*userword.UserWord{}
	for rows.Next() {
		w, err := scanUserWord(rows)
		if err != nil {
			return nil, fmt.Errorf("userword: 一覧取得の読み取りに失敗しました: %w", err)
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (r *UserWordRepository) CountByUser(ctx context.Context, userID uuid.UUID) (int, error) {
	var n int

	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM user_words WHERE user_id = $1`, userID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("userword: 件数取得に失敗しました: %w", err)
	}
	return n, nil
}

func (r *UserWordRepository) ExistsByEntry(ctx context.Context, userID, entryID uuid.UUID) (bool, error) {
	var exists bool

	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM user_words WHERE user_id = $1 AND entry_id = $2)`, userID, entryID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("userword: 重複確認に失敗しました: %w", err)
	}
	return exists, nil
}

func (r *UserWordRepository) Update(ctx context.Context, w *userword.UserWord) error {
	const q = `
		UPDATE user_words
		SET memo = NULLIF($3,''), encountered_title = NULLIF($4,''),
		    mastery_level = $5, updated_at = now()
		WHERE user_id = $1 AND id = $2
		RETURNING updated_at`

	err := r.db.QueryRowContext(ctx, q, w.UserID, w.ID, w.Memo, w.EncounteredTitle, int(w.MasteryLevel)).
		Scan(&w.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return userword.ErrNotFound
	}

	if err != nil {
		return fmt.Errorf("userword: 更新に失敗しました: %w", err)
	}
	return nil
}

func (r *UserWordRepository) Delete(ctx context.Context, userID, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM user_words WHERE user_id = $1 AND id = $2`, userID, id)
	if err != nil {
		return fmt.Errorf("userword: 削除に失敗しました: %w", err)
	}

	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("userword: 削除結果の確認に失敗しました: %w", err)
	}
	if n == 0 {
		return userword.ErrNotFound
	}
	return nil
}

// RecordView は個別ページの閲覧を生ログに1件積みます。
// 生ログは日次バッチで user_word_daily_views に丸められ、90日で消えます。
func (r *UserWordRepository) RecordView(ctx context.Context, userWordID uuid.UUID, at time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO word_view_logs (user_word_id, viewed_at) VALUES ($1, $2)`,
		userWordID, nowOr(at))
	if err != nil {
		return fmt.Errorf("userword: 閲覧記録に失敗しました: %w", err)
	}
	return nil
}

func nowOr(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now()
	}
	return t
}
