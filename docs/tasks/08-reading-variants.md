# 08. 読みの別表記と、検索精度の最終確認

- ブランチ: `feat/reading-variants`
- 前提: 07 がマージ済み
- 参照: 共通の決まりは [README.md](README.md)

## 目的

畳み込み表で吸収できない揺れ（「かむさ／かんさ」のようなパッチムの聞き方の違い）を、単語ごとの **読みの別表記** で補い、検索精度の目標を達成します。

---

## 1. 用語を追加する（実装より先）

`docs/ubiquitous-language.md` に追加し、変更履歴に1行追記します。

| 用語 | コード上の名前 | 意味 |
|---|---|---|
| 読みの別表記 | `reading_variant` | 1つの単語に対する、正規の読み以外の聞こえ方（例: 진짜 →「ジンジャ」） |

## 2. テーブル

`db/migrations/0006_reading_variants.sql`

```sql
CREATE TABLE entry_reading_variants (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_id    uuid NOT NULL REFERENCES dictionary_entries(id) ON DELETE CASCADE,
    reading     text NOT NULL,           -- 人が読める形（カタカナ）
    reading_key text NOT NULL,           -- phonetic.Key(reading)
    source      text NOT NULL CHECK (source IN ('alt_pronunciation', 'curated', 'search_log')),
    created_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT entry_reading_variants_key UNIQUE (entry_id, reading_key)
);
CREATE INDEX idx_variants_key_trgm ON entry_reading_variants USING gin (reading_key gin_trgm_ops);
```

| `source` | 内容 | 作り方 |
|---|---|---|
| `alt_pronunciation` | 公式の発音が複数あるときの、主でない方から作った読み（例: 맛있어 の [마디써] → マディッソ） | `seedgen` が自動で作る |
| `curated` | 日本人が実際にそう聞こえがちな表記 | `selection.json` の各項目に `"variants": ["ジンジャ"]` と書く。1語あたり0〜3個が目安で、思いつきで水増ししない |
| `search_log` | 将来の「検索ログから別表記を育てる」機能のための予約 | **今回は使わない** |

- 別表記の読みキーが正規の読みキーと同じなら意味が無いので、`seedgen -check` でエラーにする
- 別表記は読みであって、ハングルや意味ではありません。公式辞典に無くてよく、辞書の正確さには影響しません
- **評価セット（`db/seed/eval.json`）の入力を、そのまま別表記にコピーしないこと。** 評価セットのうち少なくとも半数は、別表記に含まれない入力のままにします。そうしないと、評価が正解の丸暗記を測るだけになります

## 3. 検索に組み込む

`SearchByReading` のスコアに「その単語の別表記の読みキーとの類似度の最大値」を加えます。
単語ごとに1行で返すこと（別表記の数だけ同じ単語が重複しないように）。

## 4. ドキュメント

| ファイル | 更新内容 |
|---|---|
| `docs/folder-structure.md` | 追加・変更したファイル |
| `README.md` | 「主要な設計判断」に読みの別表記の説明。`selection.json` の `variants` の書き方 |

## 完了の定義

`CLAUDE.md` の「完了の定義」に加えて:

- 評価セットで **recall@8 が 90% 以上**、**recall@1 が 70% 以上**（AIフォールバックなし）
  - 届かない場合は無理に数値合わせをせず、外れた入力の一覧と原因の見立てを報告する
- `SELECT show_trgm('ちんちゃ');` が空配列でない（ロケールの確認）
- 評価セットのうち、別表記と一致する入力の割合を PR に載せる（50% 以下であること）

PR本文には、06〜08 の数値の推移を表で載せてください。

| 時点 | recall@1 | recall@8 | 評価件数 |
|---|---|---|---|
| ベースライン（06） | | | |
| 読みキーの導入後（07） | | | |
| 別表記の導入後（08・最終） | | | |
