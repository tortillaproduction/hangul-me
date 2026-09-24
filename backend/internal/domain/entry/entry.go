// Package entry は辞書エントリ（単語）のドメインモデルです
package entry

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hangulme/hangul-me/backend/internal/domain/hangul"
)

// Source は辞書エントリの由来です
type Source string

const (
	SourceCurated       Source = "curated"        // 自前で整備した辞書データ
	SourceAIGenerated   Source = "ai_generated"   // LLMフォールバックで生成
	SourceUserSuggested Source = "user_suggested" // ユーザーが提案
)

// Valid は既知の由来かを返します
func (s Source) Valid() bool {
	switch s {
	case SourceCurated, SourceAIGenerated, SourceUserSuggested:
		return true
	}
	return false
}

var (
	ErrEmptyHangul   = errors.New("entry: ハングルが空です")
	ErrEmptyReading  = errors.New("entry: 読み（カタカナ）が空です")
	ErrEmptyMeaning  = errors.New("entry: 意味が空です")
	ErrInvalidSource = errors.New("entry: 未知の由来です")
	ErrNotFound      = errors.New("entry: 見つかりません")
)

// Entry は辞書エントリ（単語1件）です
//
// InitialConsonant / FinalConsonant / HasFinalConsonant は
// ハングルから機械的に導出される分析用の値で、外部から直接は設定しません
type Entry struct {
	ID                uuid.UUID
	Hangul            string
	ReadingKana       string
	ReadingHiragana   string
	Romanized         string
	MeaningJA         string
	Source            Source
	InitialConsonant  string
	HasFinalConsonant bool
	FinalConsonant    string
	AudioURL          string
	Examples          []Example
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Example は例文です
type Example struct {
	ID          uuid.UUID
	SentenceKO  string
	SentenceJA  string
	SourceTitle string
}

// New は入力値を検証し、音韻分析を済ませた Entry を作ります
func New(hangulText, readingKana, readingHiragana, romanized, meaningJA string, src Source) (*Entry, error) {
	hangulText = strings.TrimSpace(hangulText)
	readingKana = strings.TrimSpace(readingKana)
	meaningJA = strings.TrimSpace(meaningJA)

	if hangulText == "" {
		return nil, ErrEmptyHangul
	}
	if readingKana == "" {
		return nil, ErrEmptyReading
	}
	if meaningJA == "" {
		return nil, ErrEmptyMeaning
	}
	if !src.Valid() {
		return nil, ErrInvalidSource
	}

	e := &Entry{
		ID:              uuid.New(),
		Hangul:          hangulText,
		ReadingKana:     readingKana,
		ReadingHiragana: strings.TrimSpace(readingHiragana),
		Romanized:       strings.TrimSpace(strings.ToLower(romanized)),
		MeaningJA:       meaningJA,
		Source:          src,
	}

	e.ApplyPhoneetics()
	return e, nil
}

// ApplyPhoneetics はハングルから分析用カラムの値を導出して自身に反映します
// ハングルを更新した場合は必ず呼び出します
func (e *Entry) ApplyPhoneetics() {
	a := hangul.Analyze(e.Hangul)
	e.InitialConsonant = a.InitialConsonant
	e.HasFinalConsonant = a.HasFinalConsonant
	e.FinalConsonant = a.FinalConsonant
}

// Repository は辞書エントリの永続化境界です
type Repository interface {
	// FindByID は1件取得します
	// 見つからなければ ErrNotFound
	FindByID(ctx context.Context, id uuid.UUID) (*Entry, error)

	// SearchByReading は正規化済みの読みで曖昧検索し、類似度順に返します
	SearchByReading(ctx context.Context, normalized string, limit int) ([]ScoredEntry, error)

	// FindByHangulAndMeaning は重複判定に使います
	// 見つからなければ ErrNotFound
	FindByHangulAndMeaning(ctx context.Context, hangulText, meaningJA string) (*Entry, error)

	// Create は新規エントリを保存します
	Create(ctx context.Context, e *Entry) error

	// ListExamples は例文を取得します
	ListExamples(ctx context.Context, entryID uuid.UUID) ([]Example, error)
}

// ScoredEntry は類似度スコア付きの検索結果です
type ScoredEntry struct {
	Entry *Entry
	// Score は 0.0~1.0
	// pg_trgm の similarity() 由来
	Score float64
}
