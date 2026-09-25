// Package matching はうろ覚えの読みから単語候補を組み立てるドメインサービスです
//
// 方針（ハイブリッド）：
//  1. まず辞書DBを pg_trgm で曖昧検索する（速い・安い）
//  2. 候補が質・量ともに足りて入れば、そこで終わり
//  3. 足りない時だけ LLM に問い合わせる
//  4. LLMが返した候補は必ずDBと再照合し、基地の単語なら辞書側の情報を優先する
package matching

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/hangulme/hangul-me/backend/internal/domain/entry"
)

// Origin は候補がどこから来たかを表します（ログ・精度改善用。UIには出しません）
type Origin string

const (
	OriginDBTrgm     Origin = "db_trgm"
	OriginAIFallback Origin = "ai_fallback"
	OriginNone       Origin = "none"
)

// Candidate は利用者に提示する候補1件です
type Candidate struct {
	Entry  *entry.Entry
	Score  float64
	Origin Origin
	// Persisted は辞書DBに実体があるか
	// false の場合、登録時に先に dictionary_entries へ保存する必要があります
	Persisted bool
}

// Result はマッチングの結果一式です
type Result struct {
	Query      string
	Normalized string
	Script     ScriptKind
	Candidates []Candidate
	// MatchedVia は結果全体の主たる取得経路
	// search_queries に記録します
	MatchedVia Origin
}

// AIGenerator は候補不足時のフォールバック先です
// infra/llm が実装します
type AIGenerator interface {
	// Suggest は正規化済みの読みから候補を生成します。
	// 失敗しても致命的ではないため、呼び出し側はエラーを握り潰して、
	// DB候補のみで応答できます。
	Suggest(ctx context.Context, normalized, raw string, limit int) ([]entry.Entry, error)
}

// Config はマッチングの閾値設定です
type Config struct {
	// Limit は返す候補の最大件数
	Limit int
	// MinScore はこの類似度未満のDB候補を捨てる下限
	MinScore float64
	// SufficientCount この件数以上「良い」候補が取れたらAIを呼びません
	SufficientCount int
	// StrongScore 「良い候補」とみなす類似度
	StrongScore float64
}

// DefaultConfig は実運用の初期値です
func DefaultConfig() Config {
	return Config{
		Limit:           8,
		MinScore:        0.15,
		SufficientCount: 3,
		StrongScore:     0.45,
	}
}

// Service は候補マッチングのドメインサービスです
type Service struct {
	entries entry.Repository
	ai      AIGenerator // nil 可（AIフォールバック無効）
	cfg     Config
}

// NewService はサービスを作ります
// ai に nil を渡すとDB検索のみになります
func NewService(entries entry.Repository, ai AIGenerator, cfg Config) *Service {
	if cfg.Limit <= 0 {
		cfg = DefaultConfig()
	}
	return &Service{entries: entries, ai: ai, cfg: cfg}
}

// ErrEmptyQuery は空文字で検索された場合に返ります
var ErrEmptyQuery = errors.New("matching: 検索文字列が空です")

// Match はうろ覚えの読みから候補を組み立てます
func (s *Service) Match(ctx context.Context, raw string) (*Result, error) {
	normalized := Normalize(raw)
	if normalized == "" {
		return nil, ErrEmptyQuery
	}

	res := &Result{
		Query:      raw,
		Normalized: normalized,
		Script:     DetectScript(raw),
		MatchedVia: OriginNone,
	}

	// 1) DB曖昧検索
	scored, err := s.entries.SearchByReading(ctx, normalized, s.cfg.Limit)
	if err != nil {
		return nil, err
	}
	for _, sc := range scored {
		if sc.Score < s.cfg.MinScore {
			continue
		}
		res.Candidates = append(res.Candidates, Candidate{
			Entry:     sc.Entry,
			Score:     sc.Score,
			Origin:    OriginDBTrgm,
			Persisted: true,
		})
	}
	if len(res.Candidates) > 0 {
		res.MatchedVia = OriginDBTrgm
	}

	// 2) 閾値判定 - 十分ならAIを呼ばない
	if s.ai == nil || s.sufficient(res.Candidates) {
		s.finalize(res)
		return res, nil
	}

	// 3) AIフォールバック（失敗してもDB候補だけで返す）
	generated, aiErr := s.ai.Suggest(ctx, normalized, raw, s.cfg.Limit)
	if aiErr != nil || len(generated) == 0 {
		s.finalize(res)
		return res, nil
	}

	// 4) 生成候補をDBと再照合
	for i := range generated {
		g := generated[i]
		g.ApplyPhonetics()

		if known, err := s.entries.FindByHangulAndMeaning(ctx, g.Hangul, g.MeaningJA); err == nil && known != nil {
			// 既知の単語ならDB側を正とする（重複追加しない）
			if containsEntry(res.Candidates, known.Hangul, known.MeaningJA) {
				continue
			}
			res.Candidates = append(res.Candidates, Candidate{
				Entry:     known,
				Score:     s.cfg.StrongScore,
				Origin:    OriginDBTrgm,
				Persisted: true,
			})
			continue
		} else if err != nil && !errors.Is(err, entry.ErrNotFound) {
			return nil, err
		}

		if containsEntry(res.Candidates, g.Hangul, g.MeaningJA) {
			continue
		}
		g.Source = entry.SourceAIGenerated
		res.Candidates = append(res.Candidates, Candidate{
			Entry:     &g,
			Score:     s.cfg.MinScore, // 未検証なのでDB一致より低く置く
			Origin:    OriginAIFallback,
			Persisted: false,
		})
		if res.MatchedVia == OriginNone {
			res.MatchedVia = OriginAIFallback
		}
	}

	s.finalize(res)
	return res, nil
}

// sufficient は「AIを呼ばずに済むか」を判定します
func (s *Service) sufficient(cands []Candidate) bool {
	strong := 0
	for _, c := range cands {
		if c.Score >= s.cfg.StrongScore {
			strong++
		}
	}
	return strong >= s.cfg.SufficientCount
}

// finalize はスコア降順に並べ、上限件数で切り詰めます
func (s *Service) finalize(res *Result) {
	sort.SliceStable(res.Candidates, func(i, j int) bool {
		if res.Candidates[i].Score != res.Candidates[j].Score {
			return res.Candidates[i].Score > res.Candidates[j].Score
		}
		return res.Candidates[i].Entry.Hangul < res.Candidates[j].Entry.Hangul
	})
	if len(res.Candidates) > s.cfg.Limit {
		res.Candidates = res.Candidates[:s.cfg.Limit]
	}
	if len(res.Candidates) == 0 {
		res.MatchedVia = OriginNone
	}
}

func containsEntry(cands []Candidate, hangulText, meaning string) bool {
	for _, c := range cands {
		if c.Entry == nil {
			continue
		}
		if strings.EqualFold(c.Entry.Hangul, hangulText) && c.Entry.MeaningJA == meaning {
			return true
		}
	}
	return false
}
