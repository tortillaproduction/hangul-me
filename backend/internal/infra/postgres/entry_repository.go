package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/hangulme/hangul-me/backend/internal/domain/entry"
)

// EntryRepository は ductionary_entries / example_sentences のリポジトリです。
type EntryRepository struct {
	db *sql.DB
}

func NewEntryRepository(db *sql.DB) *EntryRepository {
	return &EntryRepository{
		db: db,
	}
}

var _ entry.Repository = (*EntryRepository)(nil)

const entryColumns = `
	id, hangul, reading_kana, reading_hiragana, romanized, meaning_ja, source,
	initial_consonant, has_final_consonant, COALESCE(final_consonant, '') AS final_consonant,
	COALESCE(audio_url, '') AS audio_url, created_at, updated_at`

func scanEntry(row scannable) (*entry.Entry, error) {
	var e entry.Entry
	var src string

	err := row.Scan(
		&e.ID, &e.Hangul, &e.ReadingKana, &e.ReadingHiragana, &e.Romanized, &e.MeaningJA,
		&src, &e.InitialConsonant, &e.HasFinalConsonant, &e.FinalConsonant, &e.AudioURL,
		&e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	e.Source = entry.Source(src)
	return &e, nil
}

func (r *EntryRepository) FindByID(ctx context.Context, id uuid.UUID) (*entry.Entry, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+entryColumns+` FROM dictionary_entries WHERE id = $1`, id)

	e, err := scanEntry(row)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, entry.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("entry: 取得に失敗しました: %w", err)
	}
	return e, nil
}

func (r *EntryRepository) FindByHangulAndMeaning(ctx context.Context, hangulText, meaningJA string) (*entry.Entry, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+entryColumns+` FROM dictionary_entries WHERE hangul = $1 AND meaning_ja = $2`, hangulText, meaningJA)

	e, err := scanEntry(row)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, entry.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("entry: 重複確認に失敗しました: %w", err)
	}
	return e, nil
}

// SearchByReading は pg_trgm で読みを曖昧検索します。
//
// カタカナ読み・ひらがな読み・ローマ字綴りの3系統に対して類似度を取り、
// その最大値をその単語のスコアとします。
// 入力は正規化済み（ひらがな or 小文字英字）である前提です。
//
// 入力途中は trigram 類似度が低く出るため、前方一致したものは
// 優先的に上位へ引き上げています（リアルタイム検索の体感を保つため）。
func (r *EntryRepository) SearchByReading(ctx context.Context, normalized string, limit int) ([]entry.ScoredEntry, error) {
	if limit <= 0 {
		limit = 8
	}

	const q = `
		WITH scored AS (
			SELECT ` + entryColumns + `,
				GREATEST(
					similarity(reading_hiragana, $1),
					similarity(reading_kana, $1),
					similarity(romanized, $1)
				) AS score,
				(reading_hiragana LIKE $2 OR romanized LIKE $2 OR reading_kana LIKE $2) AS prefix_hit
			FROM dictionary_entries
		)
		SELECT * FROM scored
		WHERE score > 0 OR prefix_hit
		ORDER BY prefix_hit DESC, score DESC, hangul
		LIMIT $3`

	rows, err := r.db.QueryContext(ctx, q, normalized, normalized+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("entry: 曖昧検索に失敗しました: %w", err)
	}
	defer rows.Close()

	var out []entry.ScoredEntry
	for rows.Next() {
		var e entry.Entry
		var src string
		var score float64
		var prefixHit bool

		if err := rows.Scan(
			&e.ID, &e.Hangul, &e.ReadingKana, &e.ReadingHiragana, &e.Romanized, &e.MeaningJA,
			&src, &e.InitialConsonant, &e.HasFinalConsonant, &e.FinalConsonant, &e.AudioURL,
			&e.CreatedAt, &e.UpdatedAt,
			&score, &prefixHit,
		); err != nil {
			return nil, fmt.Errorf("entry: 検索結果の読み取りに失敗しました: %w", err)
		}

		e.Source = entry.Source(src)
		if prefixHit && score < 0.6 {
			score = 0.6
		}

		copied := e
		out = append(out, entry.ScoredEntry{Entry: &copied, Score: score})
	}
	return out, rows.Err()
}

func (r *EntryRepository) Create(ctx context.Context, e *entry.Entry) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	e.ApplyPhonetics()

	var finalConsonant any
	if e.FinalConsonant != "" {
		finalConsonant = e.FinalConsonant
	}

	const q = `
		INSERT INTO dictionary_entries
			(id, hangul, reading_kana, reading_hiragana, romanized, meaning_ja, source,
			 initial_consonant, has_final_consonant, final_consonant, audio_url)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10, NULLIF($11,''))
		ON CONFLICT (hangul, meaning_ja) DO UPDATE
			SET updated_at = now()
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRowContext(
		ctx, q,
		e.ID, e.Hangul, e.ReadingKana, e.ReadingHiragana, e.Romanized, e.MeaningJA, string(e.Source),
		e.InitialConsonant, e.HasFinalConsonant, finalConsonant, e.AudioURL,
	).Scan(&e.ID, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return fmt.Errorf("entry: 保存に失敗しました: %w", err)
	}
	return nil
}

func (r *EntryRepository) ListExamples(ctx context.Context, entryID uuid.UUID) ([]entry.Example, error) {
	const q = `
		SELECT id, sentence_ko, sentence_ja, COALESCE(source_title, '')
		FROM example_sentences
		WHERE entry_id = $1
		ORDER BY created_at`
	rows, err := r.db.QueryContext(ctx, q, entryID)
	if err != nil {
		return nil, fmt.Errorf("entry: 例文の取得に失敗しました: %w", err)
	}
	defer rows.Close()

	var out []entry.Example
	for rows.Next() {
		var ex entry.Example
		if err := rows.Scan(&ex.ID, &ex.SentenceKO, &ex.SentenceJA, &ex.SourceTitle); err != nil {
			return nil, fmt.Errorf("entry: 例文の読み取りに失敗しました: %w", err)
		}
		out = append(out, ex)
	}
	return out, rows.Err()
}
