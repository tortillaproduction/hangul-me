package hangul_test

import (
	"github.com/hangulme/hangul-me/backend/internal/domain/hangul"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ハングルの字母分解", func() {

	Describe("Decompose", func() {
		DescribeTable("1音節を初声・中声・終声に分解する",
			func(char rune, initial, medial, final string) {
				s, ok := hangul.Decompose(char)
				Expect(ok).To(BeTrue())
				Expect(s.Initial).To(Equal(initial))
				Expect(s.Medial).To(Equal(medial))
				Expect(s.Final).To(Equal(final))
			},
			Entry("안（パッチムあり）", '안', "ㅇ", "ㅏ", "ㄴ"),
			Entry("녕（パッチムあり）", '녕', "ㄴ", "ㅕ", "ㅇ"),
			Entry("가（パッチムなし）", '가', "ㄱ", "ㅏ", ""),
			Entry("힣（最後の音節）", '힣', "ㅎ", "ㅣ", "ㅎ"),
			Entry("박（濃音でない終声ㄱ）", '박', "ㅂ", "ㅏ", "ㄱ"),
			Entry("짜（濃音の初声）", '짜', "ㅉ", "ㅏ", ""),
		)

		It("ハングル音節でない文字は分解しない", func() {
			for _, r := range []rune{'a', 'あ', 'ア', '漢', 'ㄱ'} {
				_, ok := hangul.Decompose(r)
				Expect(ok).To(BeFalse(), "対象外の文字を分解してしまった: %c", r)
			}
		})
	})

	Describe("Compose", func() {
		It("Decompose の逆変換になっている", func() {
			for _, original := range []rune{'안', '녕', '가', '힣', '박'} {
				s, ok := hangul.Decompose(original)
				Expect(ok).To(BeTrue())

				got, ok := hangul.Compose(s.Initial, s.Medial, s.Final)
				Expect(ok).To(BeTrue())
				Expect(got).To(Equal(original))
			}
		})

		It("未知の字母では組み立てない", func() {
			_, ok := hangul.Compose("x", "ㅏ", "")
			Expect(ok).To(BeFalse())
		})
	})

	Describe("Analyze", func() {
		Context("分析用カラムの値を導出するとき", func() {
			DescribeTable("先頭音節の初声と末尾音節のパッチムを返す",
				func(word, initial, final string, hasFinal bool) {
					a := hangul.Analyze(word)
					Expect(a.InitialConsonant).To(Equal(initial))
					Expect(a.FinalConsonant).To(Equal(final))
					Expect(a.HasFinalConsonant).To(Equal(hasFinal))
				},
				// 末尾にパッチムがある語。
				Entry("안녕", "안녕", "ㅇ", "ㅇ", true),
				Entry("대박", "대박", "ㄷ", "ㄱ", true),
				Entry("화이팅", "화이팅", "ㅎ", "ㅇ", true),
				// 末尾にパッチムがない語。
				Entry("안녕하세요", "안녕하세요", "ㅇ", "", false),
				Entry("진짜", "진짜", "ㅈ", "", false),
				Entry("감사합니다", "감사합니다", "ㄱ", "", false),
				Entry("어떻게", "어떻게", "ㅇ", "", false),
			)

			It("音節をすべて保持する", func() {
				a := hangul.Analyze("안녕")
				Expect(a.Syllables).To(HaveLen(2))
				Expect(a.Syllables[0].Char).To(Equal("안"))
				Expect(a.Syllables[1].Char).To(Equal("녕"))
			})

			It("ハングル以外の文字は無視する", func() {
				a := hangul.Analyze("안녕 (挨拶) hello")
				Expect(a.Syllables).To(HaveLen(2))
				Expect(a.InitialConsonant).To(Equal("ㅇ"))
			})

			It("ハングルを含まない場合は空の分析結果を返す", func() {
				a := hangul.Analyze("annyeong")
				Expect(a.InitialConsonant).To(BeEmpty())
				Expect(a.FinalConsonant).To(BeEmpty())
				Expect(a.HasFinalConsonant).To(BeFalse())
				Expect(a.Syllables).To(BeEmpty())
			})

			It("空文字でも落ちない", func() {
				Expect(func() { hangul.Analyze("") }).NotTo(Panic())
			})
		})
	})

	Describe("字母一覧", func() {
		It("初声は19種、パッチムは27種を返す", func() {
			Expect(hangul.InitialConsonants()).To(HaveLen(19))
			Expect(hangul.FinalConsonants()).To(HaveLen(27))
		})

		It("返した一覧を書き換えても内部状態は壊れない", func() {
			list := hangul.InitialConsonants()
			list[0] = "書き換え"
			Expect(hangul.InitialConsonants()[0]).To(Equal("ㄱ"))
		})
	})

	Describe("ContainsHangul", func() {
		It("ハングルの有無を判定する", func() {
			Expect(hangul.ContainsHangul("안녕")).To(BeTrue())
			Expect(hangul.ContainsHangul("アンニョン")).To(BeFalse())
			Expect(hangul.ContainsHangul("")).To(BeFalse())
		})
	})
})
