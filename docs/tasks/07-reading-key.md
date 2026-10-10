# 07. 読みキー

- ブランチ: `feat/reading-key`
- 前提: 06 がマージ済み（ベースラインの数値がある）
- 参照: 共通の決まりは [README.md](README.md)

## 目的

「じんじゃ」で 진짜（チンチャ）が出るように、**聞き間違えやすい音の違いを畳み込んだ検索用の読み（読みキー）** を導入します。

---

## 1. 用語を追加する（実装より先）

`docs/ubiquitous-language.md` に追加し、変更履歴に1行追記します。

| 用語 | コード上の名前 | 意味 |
|---|---|---|
| 読みキー | `reading_key` / `phonetic.Key` | 聞き間違えやすい音の違いを畳み込んだ、検索専用の読み。UIには出さない |
| 畳み込み表 | `foldTable` | 読みキーを作るときの「この音とこの音は同じとみなす」対応表 |

## 2. ドメインに `phonetic` パッケージを作る

`internal/domain/phonetic/` を新設します。

- 理由: 読みキーは `entry`（保存時に計算）と `matching`（検索時に計算）の両方が使います。今 `Normalize` は `matching` にありますが、`matching` は `entry` を import しているため、`entry` から `matching` を呼ぶと循環します
- `matching/normalize.go` の `Normalize` と `DetectScript` を `phonetic` へ移し、呼び出し側を直してください。`matching` 側に薄いラッパーを残す必要はありません

```go
// Normalize は表記ゆれを揃えます（既存の処理をそのまま移す）。
func Normalize(raw string) string

// Key は Normalize の結果に畳み込み表を当てた、検索専用の読みキーを返します。
func Key(raw string) string
```

## 3. 畳み込み表（初期案）

**正解はありません。評価の数値を見て調整してください。**

かな入力用:

| 畳み込む音 | 代表 | 韓国語側の理由 |
|---|---|---|
| が・ぎ・ぐ・げ・ご | か行 | ㄱ は語頭でカ、語中でガに聞こえる |
| ざ・ず・ぜ・ぞ | さ行 | ㅅ・ㅈ の聞こえ方の揺れ |
| じ・ぢ | ち | ㅈ は語頭でチ、語中でジに聞こえる |
| だ・で・ど | た行 | ㄷ |
| づ | つ | |
| ば行・ぱ行 | ぱ行 | ㅂ・ㅍ・ㅃ（**は行には寄せない**。ㅎ と区別するため） |
| ぁぃぅぇぉ・ゃゅょ | 大きい字 | 「にょ／によ」「うぉ／うお」の揺れ |
| っ | 削除 | 濃音・パッチムの促音化を聞き取れるかどうかの揺れ |
| を | お | |
| ゔ | ぷ | |

ローマ字入力用:

| 畳み込む綴り | 代表 |
|---|---|
| 子音の重ね（kk, tt, pp, ss, jj） | 1文字 |
| g/k、d/t、b/p、j/ch | k、t、p、ch |
| eo/o、eu/u、ae/e | o、u、e |

- 表は `phonetic` パッケージ内の1か所にまとめ、**SQL側に同じ処理を書かない**
- 畳み込みは「同じとみなしてよい音を減らす」方向にだけ使う。読みキー単体で順位を決めず、元の読みとの類似度と併用する

## 4. DB

`db/migrations/0005_reading_key.sql`

```sql
ALTER TABLE dictionary_entries ADD COLUMN reading_key text NOT NULL DEFAULT '';
CREATE INDEX idx_entries_reading_key_trgm ON dictionary_entries USING gin (reading_key gin_trgm_ops);
```

- `entry.Entry` に `ReadingKey` を足し、`ApplyPhonetics()` の中で `phonetic.Key(ReadingHiragana)` から設定する。**手で値を書かない**
- 既存行は `seedgen` の再実行で埋まる（`domain/dictionary` が `ApplyPhonetics()` を通すため）。別途 backfill は不要

## 5. 検索クエリ

`infra/postgres/entry_repository.go` の `SearchByReading` を、正規化済みの読みと読みキーの両方を受け取る形にします。

- スコア = `GREATEST(今の3系統の類似度, similarity(reading_key, $key))`
- 読みキーが完全一致したら、スコアの下限を 0.9 にする
- 前方一致の判定（`prefix_hit`）に読みキーも加える

## 6. テスト

- 表の各行が畳み込まれること（`DescribeTable`）
- 「じんじゃ」と「ちんちゃ」、「ごまうぉ」と「こまうぉ」、「おぱ」と「おっぱ」、「あんによん」と「あんにょん」が同じキーになること
- **畳み込みすぎていないこと**: 「はな」と「ぱな」、「さらん」と「ちゃらん」は別のキーのままであること

## 7. ドキュメント

| ファイル | 更新内容 |
|---|---|
| `docs/folder-structure.md` | `domain/phonetic`（`matching/normalize.go` の移動も反映） |
| `README.md` | 「主要な設計判断」の「候補マッチング」に読みキーの説明を追加 |
| `CLAUDE.md` | 「分析用カラム」の項に `reading_key` を追加（`phonetic.Key` が唯一の生成元） |

## 完了の定義

`CLAUDE.md` の「完了の定義」に加えて:

- `evalsearch` を実行し、ベースラインとの比較（全体と揺れの種類ごと）を PR に載せる
- 数値が下がった揺れの種類があれば、理由の見立てを書く
