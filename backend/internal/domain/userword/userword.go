// Package userwordは個人の単語帳（登録単語）のドメインモデルです
package userword

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hangulme/hangul-me/backend/internal/domain/entry"
)

var (
	ErrNotFound      = errors.New("userword: 見つかりません")
	ErrAlreadyExists = errors.New("userword: すでに単語帳に登録済みです")
	ErrLimitExceeded = errors.New("userword: プランの登録上限に達しています")
)

// MasteryLevelは定着度です
// 将来の復習機能で更新します
type MasteryLevel int

const (
	MasteryNew      MasteryLevel = 0 // 登録したて
	MasteryLearning MasteryLevel = 1 // うろ覚え
	MasteryFamiliar MasteryLevel = 2 // だいたい覚えた
	MasteryMastered MasteryLevel = 3 // 覚えた
)

// Validは定義済みの段階かを返します
func (m MasteryLevel) Valid() bool { return m >= MasteryNew && m <= MasteryMastered }

// Labelは画面表示用のラベルを返します
func (m MasteryLevel) Label() string {
	switch m {
	case MasteryLearning:
		return "うろ覚え"
	case MasteryFamiliar:
		return "だいたい覚えた"
	case MasteryMastered:
		return "覚えた"
	default:
		return "未学習"
	}
}

// UserWordは「ユーザーが単語帳に登録した単語」です
type UserWord struct {
	ID      uuid.UUID
	UserID  uuid.UUID
	EntryID uuid.UUID
	// Memoはユーザー独自のメモ
	Memo string
	// EncounteredTitleは「この単語はどこで聞いた？」の記録
	// ドラマ・映画名などを入れ、作品別の登録数の集計に使います
	EncounteredTitle string
	MasteryLevel     MasteryLevel
	RegisteredAt     time.Time
	UpdatedAt        time.Time

	// Entryは表示用に結合した辞書エントリ（永続化対象外）
	Entry *entry.Entry
}

// Newは単語帳への登録を作ります
func New(userID, entryID uuid.UUID, memo, encounteredTitle string) *UserWord {
	return &UserWord{
		ID:               uuid.New(),
		UserID:           userID,
		EntryID:          entryID,
		Memo:             strings.TrimSpace(memo),
		EncounteredTitle: strings.TrimSpace(encounteredTitle),
		MasteryLevel:     MasteryNew,
	}
}

// ListFilterは単語帳の絞り込み検索です
type ListFilter struct {
	// EncounteredTitleが非空なら、その作品で絞り込みます
	EncounteredTitle string

	// Limit / Offsetはページング
	// Limitが0なら既定値を使います
	Limit  int
	Offset int
}

// Repositoryは単語帳の永続化境界です
type Repository interface {
	Create(ctx context.Context, w *UserWord) error
	FindByID(ctx context.Context, userID, id uuid.UUID) (*UserWord, error)
	List(ctx context.Context, userID uuid.UUID, f ListFilter) ([]*UserWord, error)
	CountByUser(ctx context.Context, userID uuid.UUID) (int, error)
	ExistsByEntry(ctx context.Context, userID, entryID uuid.UUID) (bool, error)
	Update(ctx context.Context, w *UserWord) error
	Delete(ctx context.Context, userID, id uuid.UUID) error
}

// ViewRecorderは個別ページの閲覧を記録します
// 生ログは直近90日分のみ保持し、日次バッチで集計に丸めます
type ViewRecorder interface {
	RecordView(ctx context.Context, userWordID uuid.UUID, at time.Time) error
}
