package matching_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hangulme/hangul-me/backend/internal/domain/matching"
)

var _ = Describe("うろ覚えの読みの正規化", func() {

	Describe("Normalize", func() {
		Context("表記系をまたいで同じ形に畳むとき", func() {
			It("カタカナ・ひらがなを同一視する", func() {
				Expect(matching.Normalize("チンチャ")).To(Equal(matching.Normalize("ちんちゃ")))
				Expect(matching.Normalize("アンニョン")).To(Equal("あんにょん"))
			})

			It("アルファベットは小文字に畳む", func() {
				Expect(matching.Normalize("Annyeong")).To(Equal("annyeong"))
				Expect(matching.Normalize("JINJJA")).To(Equal("jinjja"))
			})

			It("全角英数字を半角にする", func() {
				Expect(matching.Normalize("ＡＮＮＹＥＯＮＧ")).To(Equal("annyeong"))
			})
		})

		Context("表記ゆれを吸収するとき", func() {
			DescribeTable("長音・記号・空白を落とす",
				func(input, expected string) {
					Expect(matching.Normalize(input)).To(Equal(expected))
				},
				Entry("長音符", "アンニョーン", "あんにょん"),
				Entry("波ダッシュ", "アンニョ〜ン", "あんにょん"),
				Entry("中黒", "アン・ニョン", "あんにょん"),
				Entry("空白", "アン ニョン", "あんにょん"),
				Entry("感嘆符", "アンニョン！", "あんにょん"),
				Entry("前後の空白", "  ちんちゃ  ", "ちんちゃ"),
			)

			It("小書き文字は畳まない（別の音として扱う）", func() {
				Expect(matching.Normalize("オッパ")).To(Equal("おっぱ"))
				Expect(matching.Normalize("オツパ")).NotTo(Equal(matching.Normalize("オッパ")))
			})
		})

		It("空文字・記号だけの入力では空文字を返す", func() {
			Expect(matching.Normalize("")).To(BeEmpty())
			Expect(matching.Normalize("　 ー〜・")).To(BeEmpty())
		})

		It("ハングル直接入力はそのまま残す", func() {
			Expect(matching.Normalize("안녕")).To(Equal("안녕"))
		})
	})

	Describe("DetectScript", func() {
		DescribeTable("入力の表記系を推定する",
			func(input string, expected matching.ScriptKind) {
				Expect(matching.DetectScript(input)).To(Equal(expected))
			},
			Entry("カタカナ", "アンニョン", matching.ScriptKana),
			Entry("ひらがな", "あんにょん", matching.ScriptKana),
			Entry("ローマ字", "annyeong", matching.ScriptLatin),
			Entry("ハングル", "안녕", matching.ScriptHangul),
			Entry("空文字", "", matching.ScriptUnknown),
			Entry("記号だけ", "!!!", matching.ScriptUnknown),
		)
	})
})
