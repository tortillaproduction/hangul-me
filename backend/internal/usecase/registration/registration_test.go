package registration_test

import (
	"context"

	"github.com/google/uuid"
	"github.com/hangulme/hangul-me/backend/internal/domain/entry"
	"github.com/hangulme/hangul-me/backend/internal/domain/user"
	"github.com/hangulme/hangul-me/backend/internal/domain/userword"
	"github.com/hangulme/hangul-me/backend/internal/usecase/lookup"
	"github.com/hangulme/hangul-me/backend/internal/usecase/registration"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// --- テストダブル -----------------------------------------------------------
type fakeUserRepo struct{ u *user.User }

func (f *fakeUserRepo) EnsureByClerkID(context.Context, string, string, string) (*user.User, error) {
	return f.u, nil
}

func (f *fakeUserRepo) FindByID(context.Context, uuid.UUID) (*user.User, error) {
	if f.u == nil {
		return nil, user.ErrNotFound
	}
	return f.u, nil
}

type fakeEntryRepo struct {
	byID    map[uuid.UUID]*entry.Entry
	byKey   map[string]*entry.Entry
	created []*entry.Entry
}

func newFakeEntryRepo() *fakeEntryRepo {
	return &fakeEntryRepo{
		byID:  map[uuid.UUID]*entry.Entry{},
		byKey: map[string]*entry.Entry{},
	}
}

func (f *fakeEntryRepo) FindByID(_ context.Context, id uuid.UUID) (*entry.Entry, error) {
	if e, ok := f.byID[id]; ok {
		return e, nil
	}
	return nil, entry.ErrNotFound
}

func (f *fakeEntryRepo) SearchByReading(context.Context, string, int) ([]entry.ScoredEntry, error) {
	return nil, nil
}

func (f *fakeEntryRepo) FindByHangulAndMeaning(_ context.Context, h, m string) (*entry.Entry, error) {
	if e, ok := f.byKey[h+"\x00"+m]; ok {
		return e, nil
	}
	return nil, entry.ErrNotFound
}

func (f *fakeEntryRepo) Create(_ context.Context, e *entry.Entry) error {
	f.created = append(f.created, e)
	f.byID[e.ID] = e
	f.byKey[e.Hangul+"\x00"+e.MeaningJA] = e
	return nil
}

func (f *fakeEntryRepo) ListExamples(context.Context, uuid.UUID) ([]entry.Example, error) {
	return nil, nil
}

type fakeUserWordRepo struct {
	items  []*userword.UserWord
	create error
}

func (f *fakeUserWordRepo) Create(_ context.Context, w *userword.UserWord) error {
	if f.create != nil {
		return f.create
	}
	f.items = append(f.items, w)
	return nil
}

func (f *fakeUserWordRepo) FindByID(context.Context, uuid.UUID, uuid.UUID) (*userword.UserWord, error) {
	return nil, userword.ErrNotFound
}

func (f *fakeUserWordRepo) List(context.Context, uuid.UUID, userword.ListFilter) ([]*userword.UserWord, error) {
	return f.items, nil
}

func (f *fakeUserWordRepo) CountByUser(context.Context, uuid.UUID) (int, error) {
	return len(f.items), nil
}

func (f *fakeUserWordRepo) ExistsByEntry(_ context.Context, _, entryID uuid.UUID) (bool, error) {
	for _, w := range f.items {
		if w.EntryID == entryID {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeUserWordRepo) Update(context.Context, *userword.UserWord) error   { return nil }
func (f *fakeUserWordRepo) Delete(context.Context, uuid.UUID, uuid.UUID) error { return nil }

type fakeSearchLogRepo struct{ logs []*lookup.SearchLog }

func (f *fakeSearchLogRepo) Create(_ context.Context, l *lookup.SearchLog) error {
	f.logs = append(f.logs, l)
	return nil
}

// --- 仕様 -------------------------------------------------------------------

var _ = Describe("単語の登録", func() {
	var (
		ctx       context.Context
		users     *fakeUserRepo
		entries   *fakeEntryRepo
		userWords *fakeUserWordRepo
		logs      *fakeSearchLogRepo
		uc        *registration.UseCase
		usr       *user.User
		existing  *entry.Entry
	)

	BeforeEach(func() {
		ctx = context.Background()
		usr = user.New("clerk_123", "test@example.com", "テスト")
		users = &fakeUserRepo{u: usr}
		entries = newFakeEntryRepo()
		userWords = &fakeUserWordRepo{}
		logs = &fakeSearchLogRepo{}

		var err error
		existing, err = entry.New("안녕", "アンニョン", "あんにょん", "annyeong", "やあ", entry.SourceCurated)
		Expect(err).NotTo(HaveOccurred())
		entries.byID[existing.ID] = existing
		entries.byKey["안녕\x00やあ"] = existing

		uc = registration.New(users, entries, userWords, logs)
	})

	Describe("辞書DBにある候補を登録するとき", func() {
		It("単語帳に追加される", func() {
			got, err := uc.Register(ctx, registration.Input{
				UserID:   usr.ID,
				EntryID:  &existing.ID,
				RawQuery: "アンニョン",
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(got.EntryID).To(Equal(existing.ID))
			Expect(userWords.items).To(HaveLen(1))
			Expect(entries.created).To(BeEmpty(), "既存の単語なのに辞書へ新規作成している")
		})

		It("「どこで聞いた？」とメモを保持する", func() {
			got, err := uc.Register(ctx, registration.Input{
				UserID:           usr.ID,
				EntryID:          &existing.ID,
				Memo:             "ドラマの第1話で聞いた",
				EncounteredTitle: "愛の不時着",
				RawQuery:         "アンニョン",
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Memo).To(Equal("ドラマの第1話で聞いた"))
			Expect(got.EncounteredTitle).To(Equal("愛の不時着"))
		})

		It("選択された候補として検索ログを1件残す", func() {
			_, err := uc.Register(ctx, registration.Input{
				UserID:     usr.ID,
				EntryID:    &existing.ID,
				RawQuery:   "アンニョン",
				Normalized: "あんにょん",
				MatchedVia: "db_trgm",
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(logs.logs).To(HaveLen(1))
			Expect(logs.logs[0].SelectedEntryID).NotTo(BeNil())
			Expect(*logs.logs[0].SelectedEntryID).To(Equal(existing.ID))
			Expect(logs.logs[0].MatchedVia).To(Equal("db_trgm"))
		})

		It("検索語が無ければログを残さない（一覧からの追加など）", func() {
			_, err := uc.Register(ctx, registration.Input{UserID: usr.ID, EntryID: &existing.ID})
			Expect(err).NotTo(HaveOccurred())
			Expect(logs.logs).To(BeEmpty())
		})
	})

	Describe("AI生成候補を登録するとき", func() {
		It("辞書エントリを作ってから単語帳に追加する", func() {
			got, err := uc.Register(ctx, registration.Input{
				UserID:      usr.ID,
				Hangul:      "고마워",
				ReadingKana: "コマウォ",
				Romanized:   "gomawo",
				MeaningJA:   "ありがとう",
				RawQuery:    "コマウォ",
				MatchedVia:  "ai_fallback",
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(entries.created).To(HaveLen(1))
			Expect(entries.created[0].Source).To(Equal(entry.SourceAIGenerated))
			Expect(got.EntryID).To(Equal(entries.created[0].ID))
		})

		It("保存時に音韻分析の値が入る", func() {
			_, err := uc.Register(ctx, registration.Input{
				UserID:      usr.ID,
				Hangul:      "대박",
				ReadingKana: "テバク",
				MeaningJA:   "すごい",
			})

			Expect(err).NotTo(HaveOccurred())
			created := entries.created[0]
			Expect(created.InitialConsonant).To(Equal("ㄷ"))
			Expect(created.HasFinalConsonant).To(BeTrue())
			Expect(created.FinalConsonant).To(Equal("ㄱ"))
		})

		It("既にDBにある単語なら辞書を重複して作らない", func() {
			_, err := uc.Register(ctx, registration.Input{
				UserID:      usr.ID,
				Hangul:      "안녕",
				ReadingKana: "アンニョン",
				MeaningJA:   "やあ",
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(entries.created).To(BeEmpty())
			Expect(userWords.items[0].EntryID).To(Equal(existing.ID))
		})

		It("情報が足りなければエラーにする", func() {
			_, err := uc.Register(ctx, registration.Input{UserID: usr.ID, Hangul: "안녕"})
			Expect(err).To(MatchError(registration.ErrInvalidInput))
		})
	})

	Describe("登録できない場合", func() {
		It("同じ単語の二重登録を拒む", func() {
			_, err := uc.Register(ctx, registration.Input{UserID: usr.ID, EntryID: &existing.ID})
			Expect(err).NotTo(HaveOccurred())

			_, err = uc.Register(ctx, registration.Input{UserID: usr.ID, EntryID: &existing.ID})
			Expect(err).To(MatchError(userword.ErrAlreadyExists))
		})

		It("プランの上限に達していたら拒む", func() {
			usr.WordLimit = 0
			_, err := uc.Register(ctx, registration.Input{UserID: usr.ID, EntryID: &existing.ID})
			Expect(err).To(MatchError(userword.ErrLimitExceeded))
		})

		It("存在しない辞書エントリは拒む", func() {
			missing := uuid.New()
			_, err := uc.Register(ctx, registration.Input{UserID: usr.ID, EntryID: &missing})
			Expect(err).To(MatchError(entry.ErrNotFound))
		})
	})
})
