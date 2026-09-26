// Package hangul はハングル音節の字母分解を扱うドメインロジックです。
//
// 「自分がどの音でつまずいているか」を可視化するために
// 単語の初声（初出頭子音）とパッチム（終声）を機械的に導出します。
//
// ハングル音節は U+AC00..U+D7A3 に並んでおり、次の式で分解できます。
//
//	index   = code - 0xAC00
//	final   = index % 28
//	medial  = (index / 28) % 21
//	initial = index / (28 * 21)
//
// 本パッケージが返す字母は、UIでそのまま表示できる互換字母
// （ㄱ, ㄴ, ㄷ … U+3131 ブロック）です。
package hangul

import "unicode/utf8"

const (
	syllableBase  rune = 0xAC00
	syllableLast  rune = 0xD7A3
	medialCount        = 21
	finalCount         = 28
	initialStride      = medialCount * finalCount // 588
)

// initials は初声19種（互換字母）。
// インデックスは文化意識の initial に対応します。
var initials = [19]string{
	"ㄱ", "ㄲ", "ㄴ", "ㄷ", "ㄸ", "ㄹ", "ㅁ", "ㅂ", "ㅃ", "ㅅ",
	"ㅆ", "ㅇ", "ㅈ", "ㅉ", "ㅊ", "ㅋ", "ㅌ", "ㅍ", "ㅎ",
}

// medials は中声21種（互換字母）。
var medials = [21]string{
	"ㅏ", "ㅐ", "ㅑ", "ㅒ", "ㅓ", "ㅔ", "ㅕ", "ㅖ", "ㅗ", "ㅘ",
	"ㅙ", "ㅚ", "ㅛ", "ㅜ", "ㅝ", "ㅞ", "ㅟ", "ㅠ", "ㅡ", "ㅢ", "ㅣ",
}

// finals は終声28種（互換字母）。
// インデックス0は「パッチムなし」。
var finals = [28]string{
	"", "ㄱ", "ㄲ", "ㄳ", "ㄴ", "ㄵ", "ㄶ", "ㄷ", "ㄹ", "ㄺ",
	"ㄻ", "ㄼ", "ㄽ", "ㄾ", "ㄿ", "ㅀ", "ㅁ", "ㅂ", "ㅄ", "ㅅ",
	"ㅆ", "ㅇ", "ㅈ", "ㅊ", "ㅋ", "ㅌ", "ㅍ", "ㅎ",
}

// Syllable は1音節の字母分解結果です。
type Syllable struct {
	Char    string // 元の音節（例: "안"）。
	Initial string // 初声（例: "ㅇ"）。
	Medial  string // 中声（例: "ㅏ"）。
	Final   string // 終声。パッチムが無ければ空文字。
}

// HasFinal はこの音節がパッチムを持つかを返します。
func (s Syllable) HasFinal() bool { return s.Final != "" }

// Analysis は単語1件分の音韻分析結果です。
// DBの dictionary_entries.initial_consonat / has_final_consonant /
// final_consonant に対応します。
type Analysis struct {
	// InitialConsonant は先頭音節の初声。
	// 「〜ニョン」「〜セヨ」のような語頭の音の傾向を見るために使います。
	InitialConsonant string

	// FinalConsonant は末尾音節のパッチム。無ければ空文字。
	FinalConsonant string

	// HasFinalConsonant は末尾音節がパッチムを持つか。
	HasFinalConsonant bool

	// Syllables は分解できた音節の一覧（ハングル以外の文字は含みません）。
	Syllables []Syllable
}

// IsSyllable は r が現代ハングル音節かを返します。
func IsSyllable(r rune) bool {
	return r >= syllableBase && r <= syllableLast
}

// ContainsHangul は s にハングル音節が1文字でも含まれるかを返します。
func ContainsHangul(s string) bool {
	for _, r := range s {
		if IsSyllable(r) {
			return true
		}
	}
	return false
}

// Decompose は1つのハングル音節を字母に分解します。
// r がハングル音節でない場合は ok=false を返します。
func Decompose(r rune) (Syllable, bool) {
	if !IsSyllable(r) {
		return Syllable{}, false
	}
	idx := int(r - syllableBase)
	return Syllable{
		Char:    string(r),
		Initial: initials[idx/initialStride],
		Medial:  medials[(idx/finalCount)%medialCount],
		Final:   finals[idx%finalCount],
	}, true
}

// Compose は字母から音節を組み立てます。Decompose の逆変換です。
// 未知の字母が渡された場合は ok=false を返します。
func Compose(initial, medial, final string) (rune, bool) {
	i := indexOf(initials[:], initial)
	m := indexOf(medials[:], medial)
	f := indexOf(finals[:], final)
	if i < 0 || m < 0 || f < 0 {
		return 0, false
	}
	return syllableBase + rune(i*initialStride+m*finalCount+f), true
}

// Analyze は単語全体を分析し、分析用カラムに保存する値を返します。
//
// ハングルを1文字も含まない場合は、すべて空の Analysis を返します。
// （AI生成候補がハングルを伴わずに返ってきた場合などを想定）。
func Analyze(word string) Analysis {
	a := Analysis{Syllables: make([]Syllable, 0, utf8.RuneCountInString(word))}
	for _, r := range word {
		if s, ok := Decompose(r); ok {
			a.Syllables = append(a.Syllables, s)
		}
	}
	if len(a.Syllables) == 0 {
		return Analysis{}
	}
	a.InitialConsonant = a.Syllables[0].Initial
	last := a.Syllables[len(a.Syllables)-1]
	a.FinalConsonant = last.Final
	a.HasFinalConsonant = last.HasFinal()
	return a
}

// InitialConsonants は初声の一覧を返します（集計軸の定義に使用）。
func InitialConsonants() []string { return append([]string(nil), initials[:]...) }

// FinalConsonants はパッチムの一覧を返します（先頭の「なし」を除く）。
func FinalConsonants() []string { return append([]string(nil), finals[1:]...) }

func indexOf(list []string, v string) int {
	for i, s := range list {
		if s == v {
			return i
		}
	}
	return -1
}
