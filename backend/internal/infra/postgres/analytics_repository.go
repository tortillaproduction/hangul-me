package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/hangulme/hangul-me/backend/internal/domain/analytics"
)

// AnalyticsRepository は分析ビュー向けの集計クエリです。
// クエリ本体のリファレンスは db/analytics.sql にもまとめています。
type AnalyticsRepository struct{ db *sql.DB }

func NewAnalyticsRepository(db *sql.DB) *AnalyticsRepository {
	return &AnalyticsRepository{
		db: db,
	}
}

var _ analytics.Repository = (*AnalyticsRepository)(nil)

func (r *AnalyticsRepository) OverviewForUser(ctx context.Context, userID uuid.UUID) (*analytics.Overview, error) {
	ov := &analytics.Overview{}

	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*),
				COUNT(*) FILTER (WHERE registered_at >= current_date - INTERVAL '7 days')
		FROM user_words WHERE user_id = $1`, userID).
		Scan(&ov.TotalWords, &ov.RegisteredThisWeek); err != nil {
		return nil, fmt.Errorf("analytics: 件数集計に失敗しました: %w", err)
	}

	var err error
	if ov.MostSearched, err = r.userMostSearched(ctx, userID, 10); err != nil {
		return nil, err
	}
	if ov.MostViewed, err = r.userMostViewed(ctx, userID, 10); err != nil {
		return nil, err
	}
	if ov.InitialConsonants, err = r.consonantBuckets(ctx, userID, true); err != nil {
		return nil, err
	}
	if ov.FinalConsonants, err = r.consonantBuckets(ctx, userID, false); err != nil {
		return nil, err
	}
	if ov.EncounteredTitles, err = r.titleBuckets(ctx, userID); err != nil {
		return nil, err
	}
	if ov.RegistrationTrend, err = r.registrationTrend(ctx, userID); err != nil {
		return nil, err
	}
	return ov, nil
}

// GlobalMostSearched はアプリ全体でよく選択された単語トップNです。
func (r *AnalyticsRepository) GlobalMostSearched(ctx context.Context, limit int) ([]analytics.RankedWord, error) {
	const q = `
		SELECT d.id, d.hangul, d.reading_kana, d.meaning_ja, COUNT(*) AS search_count
		FROM search_queries sql
		JOIN dictionary_entries d ON d.id = sq.selected_entry_id
		WHERE sq.selected_entry_id IS NOT NULL
		GROUP BY d.id
		ORDER BY search_count DESC, d.hangul
		LIMIT $1`

	return r.rankedWords(ctx, q, limit)
}

func (r *AnalyticsRepository) userMostSearched(ctx context.Context, userID uuid.UUID, limit int) ([]analytics.RankedWord, error) {
	const q = `
		SELECT d.id, d.hangul, d.reading_kana, d.meaning_ja, COUNT(*) AS search_count
		FROM search_queries sq
		JOIN dictionary_entries d ON d.id = sq.selected_entry_id
		WHERE sq.user_id = $1 AND sq.selected_entry_id IS NOT NULL
		GROUP BY d.id
		ORDER BY search_count DESC, d.hangul
		LIMIT $2`

	return r.rankedWords(ctx, q, userID, limit)
}

// userMostViewed は「よく見返した単語」です。
// 90日以内は生ログ、それ以前は日次集計を参照し、境界で重複を数えないようにしています。
func (r *AnalyticsRepository) userMostViewed(ctx context.Context, userID uuid.UUID, limit int) ([]analytics.RankedWord, error) {
	const q = `
		WITH purge_boundary AS (
			SELECT (current_date - INTERVAL '90 days')::date AS d
		),
		recent AS (
			SELECT uw.id AS user_word_id, COUNT(*) AS cnt
			FROM word_view_logs wvl
			JOIN user_words uw ON uw.id = wvl.user_word_id
			CROSS JOIN purge_boundary pb
			WHERE uw.user_id = $1 AND wvl.viewed_at::date > pb.b
			GROUP BY uw.id
		),
		archived AS (
			SELECT uwdv.user_word_id, SUM(uwdv.view_count) AS cnt
			FROM user_word_daily_views uwdv
			JOIN user_words uw ON uw.id = uwdv.user_word_id
			CROSS JOIN purge_boundary pb
			WHERE uw.user_id = $1 AND uwdv.view_date <= pb.d
			GROUP BY uwdv.user_word_id
		)
		SELECT d.id, d.hangul, d.reading_kana, d.meaning_ja,
			(COALESCE(r.cnt, 0) + COALESCE(a.cnt, 0))::int AS view_count
		FROM user_words uw
		JOIN dictionary_entries d ON d.id = uw.entry_id
		LEFT JOIN recent r ON r.user_word_id = uw.id
		LEFT JOIN archived a ON a.user_word_id = uw.id
		WHERE uw.user_id = $1
		ORDER BY view_count DESC, uw.registered_at DESC
		LIMIT $2`

	return r.rankedWords(ctx, q, userID, limit)
}

func (r *AnalyticsRepository) rankedWords(ctx context.Context, q string, args ...any) ([]analytics.RankedWord, error) {
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("analytics: ランキング集計に失敗しました: %w", err)
	}
	defer rows.Close()

	out := []analytics.RankedWord{}
	for rows.Next() {
		var w analytics.RankedWord
		if err := rows.Scan(&w.EntryID, &w.Hangul, &w.ReadingKana, &w.MeaningJA, &w.Count); err != nil {
			return nil, fmt.Errorf("analytics: ランキングの読み取りに失敗しました: %w", err)
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// consonantBuckets は初声（initial=true）またはパッチム（initial=false）の分布です。
func (r *AnalyticsRepository) consonantBuckets(ctx context.Context, userID uuid.UUID, initial bool) ([]analytics.ConsonantBucket, error) {
	col := "COALESCE(d.final_consonant, '')"
	if initial {
		col = "d.initial_consonant"
	}
	q := fmt.Sprintf(`
		SELECT %s AS consonant, COUNT(*)::int
		FROM user_words uw
		JOIN dictionary_entries d ON d.id = uw.entry_id
		WHERE uw.user_id = $1
		GROUP BY 1
		ORDER BY 2 DESC, 1`, col)

	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("analytics: 字母分布の集計に失敗しました: %w", err)
	}
	defer rows.Close()

	out := []analytics.ConsonantBucket{}
	for rows.Next() {
		var b analytics.ConsonantBucket
		if err := rows.Scan(&b.Consonant, &b.Count); err != nil {
			return nil, fmt.Errorf("analytics: 字母分布の読み取りに失敗しました: %w", err)
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// titleBuckets は「どこで聞いた？」別の登録数です。
func (r *AnalyticsRepository) titleBuckets(ctx context.Context, userID uuid.UUID) ([]analytics.TitleBucket, error) {
	const q = `
	SELECT COALESCE(NULLIF(encountered_title, ''), '') AS title, COUNT(*)::int
	FROM user_words
	WHERE user_id = $1
	GROUP BY 1
	ORDER BY 2 DESC, 1`

	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("analytics: 作品別集計に失敗しました: %w", err)
	}
	defer rows.Close()

	out := []analytics.TitleBucket{}
	for rows.Next() {
		var b analytics.TitleBucket
		if err := rows.Scan(&b.Title, &b.Count); err != nil {
			return nil, fmt.Errorf("analysis: 作品別集計の読み取りに失敗しました: %w", err)
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// registrationTrend は直近90日の日別登録数です。
func (r *AnalyticsRepository) registrationTrend(ctx context.Context, userID uuid.UUID) ([]analytics.DailyCount, error) {
	const q = `
		SELECT to_char(registered_at::date, 'YYYY-MM-DD'), COUNT(*)::int
		FROM user_words
		WHERE user_id = $1 AND registered_at >= current_date - INTERVAL '90 days'
		GROUP BY 1
		ORDER BY 1`

	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("analysis: 推移集計に失敗しました: %w", err)
	}
	defer rows.Close()

	out := []analytics.DailyCount{}
	for rows.Next() {
		var d analytics.DailyCount
		if err := rows.Scan(&d.Date, &d.Count); err != nil {
			return nil, fmt.Errorf("analysis: 推移の読み取りに失敗しました: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// RollupViews は閲覧の生ログを日次集計に丸め、90日より古い生ログを削除します。
// cmd/rillup から日次で実行します。冪等です。
func (r *AnalyticsRepository) RollupViews(ctx context.Context, retentionDays int) (rolled, purged int64, err error) {
	if retentionDays <= 0 {
		retentionDays = 90
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, fmt.Errorf("analytics: トランザクション開始に失敗しました: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx, `
		INSERT INTO user_word_daily_views (user_word_id, view_date, view_count)
		SELECT user_word_id, viewed_at::date, COUNT(*)
		FROM word_view_logs
		WHERE viewed_at::date <= current_date
		GROUP BY user_word_id, viewed_at::date
		ON CONFLICT (user_word_id, view_date)
		DO UPDATE SET view_count = EXCLUDED.view_count`)
	if err != nil {
		return 0, 0, fmt.Errorf("analytics: 日次集計に失敗しました: %w", err)
	}
	if rolled, err = res.RowsAffected(); err != nil {
		return 0, 0, fmt.Errorf("analytics: 集計件数の取得に失敗しました: %w", err)
	}

	res, err = tx.ExecContext(ctx, fmt.Sprintf(`
		DELETE FROM word_view_logs
		WHERE viewed_at::date <= (current_date - INTERVAL '%d days')::date`, retentionDays))
	if err != nil {
		return 0, 0, fmt.Errorf("analytics: 生ログの削除に失敗しました: %w", err)
	}
	if purged, err = res.RowsAffected(); err != nil {
		return 0, 0, fmt.Errorf("analytics: 削除件数の取得に失敗しました: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("analytics: コミットに失敗しました: %w", err)
	}
	return rolled, purged, nil
}
