package matching

import (
	"strings"
	"unicode"
)

// ScriptKind は入力された読みの表記系です。
type ScriptKind string

const (
	ScriptKana    ScriptKind = "kana"
	ScriptLatin   ScriptKind = "latin"
	ScriptHangul  ScriptKind = "hangul"
	ScriptUnknown ScriptKind = "unknown"
)

// Normalize はうろ覚えの読みを比較可能な形に揃えます。
//
//	・カタカナ -> ひらがな に畳む（「チンチャ」「ちんちゃ」を同一視）。
//	・アルファベットは小文字化。
//	・長音符「ー」、中黒、空白、記号を除去（「アンニョ～ン」対策）。
//	・全角英数字は半角へ。
//
// UIには出さない内部処理です。
func Normalize(raw string) string {
	var b strings.Builder
	b.Grow(len(raw))

	for _, r := range raw {
		switch {
		case r == 'ー' || r == '〜' || r == '～' || r == '・' || r == 'ｰ':
			// 長音・波ダッシュ・中黒は表記ゆれなので落とす。
			continue
		case unicode.IsSpace(r):
			continue
		case r >= 'Ａ' && r <= 'Ｚ':
			b.WriteRune(r - 'Ａ' + 'a')
		case r >= 'ａ' && r <= 'ｚ':
			b.WriteRune(r - 'ａ' + 'a')
		case r >= '０' && r <= '９':
			b.WriteRune(r - '０' + '0')
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r + ('a' - 'A'))
		case isKatakana(r):
			b.WriteRune(katakanaToHiragana(r))
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			continue
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// DetectScript は入力の表記系を推定します。
// プレースホルダの出し分けや検索ログの分析に使います。
func DetectScript(raw string) ScriptKind {
	var kana, latin, hangulCount int
	for _, r := range raw {
		switch {
		case isKatakana(r) || isHiragana(r):
			kana++
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z'):
			latin++
		case r >= 0xAC00 && r <= 0xD7A3:
			hangulCount++
		}
	}
	switch {
	case hangulCount > 0 && hangulCount >= kana && hangulCount >= latin:
		return ScriptHangul
	case kana > 0 && kana >= latin:
		return ScriptKana
	case latin > 0:
		return ScriptLatin
	default:
		return ScriptUnknown
	}
}

func isKatakana(r rune) bool { return r >= 'ァ' && r <= 'ヶ' }
func isHiragana(r rune) bool { return r >= 'ぁ' && r <= 'ゖ' }

// katakanaToHiragana はカタカナをひらがなに畳みます（U+30A1..U+30F6）。
func katakanaToHiragana(r rune) rune {
	if isKatakana(r) {
		return r - 0x60
	}
	return r
}
