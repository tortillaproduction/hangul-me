package matching_test

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/hangulme/hangul-me/backend/internal/domain/entry"
	"github.com/hangulme/hangul-me/backend/internal/domain/matching"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// --- テストダブル -----------------------------------------------------------

type fakeEntryRepo struct {
	searchResult []entry.ScoredEntry
	searchErr    error
	known        map[string]*entry.Entry // key: hangul + "\x00" + meaning
	searchCalls  int
	createdCount int
}

func newFakeEntryRepo() *fakeEntryRepo {
	return &fakeEntryRepo{known: map[string]*entry.Entry{}}
}

func (f *fakeEntryRepo) FindByID(context.Context, uuid.UUID) (*entry.Entry, error) {
	return nil, entry.ErrNotFound
}

func (f *fakeEntryRepo) SearchByReading(_ context.Context, _ string, _ int) ([]entry.ScoredEntry, error) {
	f.searchCalls++
	return f.searchResult, f.searchErr
}

func (f *fakeEntryRepo) FindByHangulAndMeaning(_ context.Context, h, m string) (*entry.Entry, error) {
	if e, ok := f.known[h+"\x00"+m]; ok {
		return e, nil
	}
	return nil, entry.ErrNotFound
}

func (f *fakeEntryRepo) Create(_ context.Context, _ *entry.Entry) error {
	f.createdCount++
	return nil
}

func (f *fakeEntryRepo) ListExamples(context.Context, uuid.UUID) ([]entry.Example, error) {
	return nil, nil
}

type fakeAI struct {
	suggestions []entry.Entry
	err         error
	calls       int
}

func (f *fakeAI) Suggest(context.Context, string, string, int) ([]entry.Entry, error) {
	f.calls++
	return f.suggestions, f.err
}

func mustEntry(h, kana, romaji, meaning string) *entry.Entry {
	e, err := entry.New(h, kana, "", romaji, meaning, entry.SourceCurated)
	Expect(err).NotTo(HaveOccurred())
	return e
}

// --- 仕様 -------------------------------------------------------------------

var _ = Describe("候補マッチング", func() {
	var (
		ctx  context.Context
		repo *fakeEntryRepo
		ai   *fakeAI
	)

	BeforeEach(func() {
		ctx = context.Background()
		repo = newFakeEntryRepo()
		ai = &fakeAI{}
	})

	Describe("入力の検証", func() {
		It("空の入力はエラーにする", func() {
			svc := matching.NewService(repo, ai, matching.DefaultConfig())
			_, err := svc.Match(ctx, "")
			Expect(err).To(MatchError(matching.ErrEmptyQuery))
		})

		It("記号だけの入力もエラーにする（正規化すると空になるため）", func() {
			svc := matching.NewService(repo, ai, matching.DefaultConfig())
			_, err := svc.Match(ctx, "〜〜〜")
			Expect(err).To(MatchError(matching.ErrEmptyQuery))
		})
	})

	Describe("DB曖昧検索で十分な候補が得られる場合", func() {
		BeforeEach(func() {
			repo.searchResult = []entry.ScoredEntry{
				{Entry: mustEntry("안녕", "アンニョン", "annyeong", "やあ"), Score: 0.9},
				{Entry: mustEntry("안녕하세요", "アンニョンハセヨ", "annyeonghaseyo", "こんにちは"), Score: 0.8},
				{Entry: mustEntry("안녕히", "アンニョンヒ", "annyeonghi", "平穏に"), Score: 0.7},
			}
		})

		It("AIを呼ばずに返す", func() {
			svc := matching.NewService(repo, ai, matching.DefaultConfig())
			res, err := svc.Match(ctx, "アンニョン")

			Expect(err).NotTo(HaveOccurred())
			Expect(ai.calls).To(Equal(0), "十分な候補があるのにAIを呼んでいる")
			Expect(res.Candidates).To(HaveLen(3))
			Expect(res.MatchedVia).To(Equal(matching.OriginDBTrgm))
		})

		It("候補はスコアの高い順に並ぶ", func() {
			svc := matching.NewService(repo, ai, matching.DefaultConfig())
			res, err := svc.Match(ctx, "アンニョン")

			Expect(err).NotTo(HaveOccurred())
			Expect(res.Candidates[0].Entry.Hangul).To(Equal("안녕"))
			Expect(res.Candidates[0].Score).To(BeNumerically(">=", res.Candidates[1].Score))
			Expect(res.Candidates[1].Score).To(BeNumerically(">=", res.Candidates[2].Score))
		})

		It("DB由来の候補は登録済みとして扱う", func() {
			svc := matching.NewService(repo, ai, matching.DefaultConfig())
			res, _ := svc.Match(ctx, "アンニョン")

			for _, c := range res.Candidates {
				Expect(c.Persisted).To(BeTrue())
				Expect(c.Origin).To(Equal(matching.OriginDBTrgm))
			}
		})

		It("正規化した読みで検索する", func() {
			svc := matching.NewService(repo, ai, matching.DefaultConfig())
			res, err := svc.Match(ctx, "アンニョーン")

			Expect(err).NotTo(HaveOccurred())
			Expect(res.Normalized).To(Equal("あんにょん"))
			Expect(repo.searchCalls).To(Equal(1))
		})
	})

	Describe("DB候補が閾値を下回る場合", func() {
		BeforeEach(func() {
			repo.searchResult = []entry.ScoredEntry{
				{Entry: mustEntry("안녕", "アンニョン", "annyeong", "やあ"), Score: 0.2},
			}
			ai.suggestions = []entry.Entry{
				*mustEntry("안녕히 가세요", "アンニョンヒカセヨ", "annyeonghi gaseyo", "さようなら"),
			}
		})

		It("AIフォールバックを呼ぶ", func() {
			svc := matching.NewService(repo, ai, matching.DefaultConfig())
			res, err := svc.Match(ctx, "アンニョン")

			Expect(err).NotTo(HaveOccurred())
			Expect(ai.calls).To(Equal(1))
			Expect(res.Candidates).To(HaveLen(2))
		})

		It("AI候補は未登録として印を付ける", func() {
			svc := matching.NewService(repo, ai, matching.DefaultConfig())
			res, _ := svc.Match(ctx, "アンニョン")

			var aiCand *matching.Candidate
			for i := range res.Candidates {
				if res.Candidates[i].Origin == matching.OriginAIFallback {
					aiCand = &res.Candidates[i]
				}
			}
			Expect(aiCand).NotTo(BeNil(), "AI由来の候補が見つからない")
			Expect(aiCand.Persisted).To(BeFalse())
			Expect(aiCand.Entry.Source).To(Equal(entry.SourceAIGenerated))
		})

		It("AI候補にも音韻分析を適用する", func() {
			svc := matching.NewService(repo, ai, matching.DefaultConfig())
			res, _ := svc.Match(ctx, "アンニョン")

			for _, c := range res.Candidates {
				if c.Origin == matching.OriginAIFallback {
					Expect(c.Entry.InitialConsonant).To(Equal("ㅇ"))
				}
			}
		})

		Context("AI候補が既にDBにある単語だったとき", func() {
			BeforeEach(func() {
				known := mustEntry("고마워", "コマウォ", "gomawo", "ありがとう")
				repo.known["고마워\x00ありがとう"] = known
				ai.suggestions = []entry.Entry{
					*mustEntry("고마워", "コマウォ", "gomawo", "ありがとう"),
				}
			})

			It("DB側のエントリを採用して登録済み扱いにする", func() {
				svc := matching.NewService(repo, ai, matching.DefaultConfig())
				res, _ := svc.Match(ctx, "コマウォ")

				var found bool
				for _, c := range res.Candidates {
					if c.Entry.Hangul == "고마워" {
						found = true
						Expect(c.Persisted).To(BeTrue(), "既知の単語がAI候補のままになっている")
						Expect(c.Origin).To(Equal(matching.OriginDBTrgm))
					}
				}
				Expect(found).To(BeTrue())
			})
		})

		Context("AI呼び出しが失敗したとき", func() {
			BeforeEach(func() { ai.err = errors.New("timeout") })

			It("DB候補だけで結果を返す（エラーにしない）", func() {
				svc := matching.NewService(repo, ai, matching.DefaultConfig())
				res, err := svc.Match(ctx, "アンニョン")

				Expect(err).NotTo(HaveOccurred())
				Expect(res.Candidates).To(HaveLen(1))
				Expect(res.MatchedVia).To(Equal(matching.OriginDBTrgm))
			})
		})
	})

	Describe("重複の排除", func() {
		BeforeEach(func() {
			repo.searchResult = []entry.ScoredEntry{
				{Entry: mustEntry("안녕", "アンニョン", "annyeong", "やあ"), Score: 0.3},
			}
			ai.suggestions = []entry.Entry{
				*mustEntry("안녕", "アンニョン", "annyeong", "やあ"), // DB候補と同じ。
			}
		})

		It("同じ単語を二重に出さない", func() {
			svc := matching.NewService(repo, ai, matching.DefaultConfig())
			res, _ := svc.Match(ctx, "アンニョン")
			Expect(res.Candidates).To(HaveLen(1))
		})
	})

	Describe("AIフォールバックを無効にした構成", func() {
		It("DB候補が乏しくてもAIを呼ばない", func() {
			repo.searchResult = []entry.ScoredEntry{
				{Entry: mustEntry("안녕", "アンニョン", "annyeong", "やあ"), Score: 0.2},
			}
			svc := matching.NewService(repo, nil, matching.DefaultConfig())

			res, err := svc.Match(ctx, "アンニョン")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Candidates).To(HaveLen(1))
		})
	})

	Describe("候補が1件も無い場合", func() {
		It("MatchedVia を none にする", func() {
			svc := matching.NewService(repo, nil, matching.DefaultConfig())
			res, err := svc.Match(ctx, "ぬるぽ")

			Expect(err).NotTo(HaveOccurred())
			Expect(res.Candidates).To(BeEmpty())
			Expect(res.MatchedVia).To(Equal(matching.OriginNone))
		})
	})

	Describe("件数の上限", func() {
		It("設定した件数までに切り詰める", func() {
			for i := 0; i < 20; i++ {
				repo.searchResult = append(repo.searchResult, entry.ScoredEntry{
					Entry: mustEntry("안녕", "アンニョン", "annyeong", "やあ"), Score: 0.9,
				})
			}
			cfg := matching.DefaultConfig()
			cfg.Limit = 5
			svc := matching.NewService(repo, ai, cfg)

			res, err := svc.Match(ctx, "アンニョン")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.Candidates).To(HaveLen(5))
		})
	})

	Describe("スコアが下限を下回る候補", func() {
		It("結果から除外する", func() {
			repo.searchResult = []entry.ScoredEntry{
				{Entry: mustEntry("안녕", "アンニョン", "annyeong", "やあ"), Score: 0.9},
				{Entry: mustEntry("없다", "オプタ", "eopda", "ない"), Score: 0.01},
			}
			svc := matching.NewService(repo, nil, matching.DefaultConfig())

			res, _ := svc.Match(ctx, "アンニョン")
			Expect(res.Candidates).To(HaveLen(1))
			Expect(res.Candidates[0].Entry.Hangul).To(Equal("안녕"))
		})
	})
})
