// Package lookup は「単語を調べる」ユースケースです。
package lookup

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hangulme/hangul-me/backend/internal/domain/entry"
	"github.com/hangulme/hangul-me/backend/internal/domain/matching"
)

// SearchLog は検索ログの書き込み境界です。
// ユーザーが実際に選んだ候補だけを記録します（表示しただけは記録しない）。
type SearchLog struct {
	ID              uuid.UUID
	UserID          *uuid.UUID
	RawInput        string
	NormalizedInput string
	MatchedVia      string
	SelectedEntryID *uuid.UUID
	CreatedAt       time.Time
}

// SearchLogRepository は検索ログの永続化境界です。
type SearchLogRepository interface {
	Create(ctx context.Context, l *SearchLog) error
}

// Candidate はAPIに返す候補です。
type Candidate struct {
	// EntryID は辞書DBに実体がある場合のみ非nil。
	// nilの場合はAI生成候補で、登録時に辞書へ保存されます。
	EntryID     *uuid.UUID
	Hangul      string
	ReadingKana string
	Romanized   string
	MeaningJA   string
	// Score / Origin はログと精度改善のための内部値です。UIには出しません。
	Score  float64
	Origin string
}

// Output は検索結果です。
type Output struct {
	Query      string
	Candidates []Candidate
	MatchedVia string
}

// Usecase は単語検索のユースケースです。
type Usecase struct {
	matcher *matching.Service
}

func New(matcher *matching.Service) *Usecase { return &Usecase{matcher: matcher} }

// Search はうろ覚えの読みから候補一覧を返します。
//
// 入力中のリアルタイム検索から呼ばれるため、ここでは検索ログを書きません。
// ログは「候補が選ばれて登録された」タイミング（registration）で1件だけ残します。
func (u *Usecase) Search(ctx context.Context, raw string) (*Output, error) {
	res, err := u.matcher.Match(ctx, raw)
	if err != nil {
		return nil, err
	}

	out := &Output{
		Query:      raw,
		MatchedVia: string(res.MatchedVia),
	}
	out.Candidates = make([]Candidate, 0, len(res.Candidates))
	for _, c := range res.Candidates {
		cand := Candidate{
			Hangul:      c.Entry.Hangul,
			ReadingKana: c.Entry.ReadingKana,
			Romanized:   c.Entry.Romanized,
			MeaningJA:   c.Entry.MeaningJA,
			Score:       c.Score,
			Origin:      string(c.Origin),
		}
		if c.Persisted {
			id := c.Entry.ID
			cand.EntryID = &id
		}
		out.Candidates = append(out.Candidates, cand)
	}
	return out, nil
}

// EntryDetail は辞書エントリの詳細（例文込み）を返します。
func (u *Usecase) EntryDetail(ctx context.Context, repo entry.Repository, id uuid.UUID) (*entry.Entry, error) {
	e, err := repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	ex, err := repo.ListExamples(ctx, id)
	if err != nil {
		return nil, err
	}
	e.Examples = ex
	return e, nil
}
