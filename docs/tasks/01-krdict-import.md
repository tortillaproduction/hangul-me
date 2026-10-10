# 01. 公式辞典を参照用テーブルに取り込む

- ブランチ: `feat/krdict-import`
- 前提: なし（最初の作業）
- 参照: [../krdict-data.md](../krdict-data.md)、共通の決まりは [README.md](README.md)

## 目的

国立国語院「韓国語基礎辞典」の全データ（見出し語 56,555）を、**照合のための参照用テーブル**に取り込みます。
このPRでは辞書DB（`dictionary_entries`）にも画面にも触りません。

---

## 1. 用語を追加する（実装より先）

`docs/ubiquitous-language.md` に以下を追加し、変更履歴に1行追記してください。

| 用語 | コード上の名前 | 意味 |
|---|---|---|
| 公式辞典 | `krdict` | 国立国語院「韓国語基礎辞典」。辞書DBの正しさの根拠 |
| 見出し語 | `headword` / `krdict_entries` | 公式辞典の1項目（例: 괜찮다）。識別コードを持つ |
| 語義 | `sense` / `krdict_senses` | 見出し語の意味の1つ（例: 괜찮다 の「大丈夫だ」）。日本語訳が付く |
| 公式の活用形 | `official form` | 公式辞典に発音付きで載っている活用形（例: 괜찮아 [괜차나]） |
| 公式の発音 | `pronunciation` | 公式辞典に載っている発音のハングル表記（例: 감사합니다 → [감사함니다]） |

## 2. Git で追跡しないファイル

`.gitignore` に次の行が入っていることを確認します（すでに追加済みのはずです。無ければ追加）。

```gitignore
/data/
*:Zone.Identifier
```

## 3. テーブル

`db/migrations/0003_krdict_reference.sql`

```sql
CREATE TABLE krdict_entries (
    id             integer PRIMARY KEY,   -- 公式辞典の識別コード（API の target_code と同じ）
    headword       text    NOT NULL,
    homonym_number integer NOT NULL DEFAULT 0,
    lexical_unit   text    NOT NULL,      -- 단어 / 구 / 관용구 / 속담 / 문법‧표현
    part_of_speech text,
    level          text,                  -- 초급 / 중급 / 고급 / 없음
    pronunciations text[]  NOT NULL DEFAULT '{}',  -- 長音記号を除いた発音
    source_version text    NOT NULL       -- 例: '20260919'
);
CREATE INDEX idx_krdict_entries_headword ON krdict_entries (headword);

CREATE TABLE krdict_senses (
    entry_id      integer NOT NULL REFERENCES krdict_entries(id) ON DELETE CASCADE,
    sense_no      integer NOT NULL,       -- データ内の語義番号（att: id）
    position      integer NOT NULL,       -- データ内の並び順（表示順）
    definition_ko text    NOT NULL,
    ja_word       text,                   -- 日本語の訳語（整形前の値）。無ければ NULL
    ja_definition text,
    PRIMARY KEY (entry_id, sense_no)
);

CREATE TABLE krdict_forms (
    entry_id       integer NOT NULL REFERENCES krdict_entries(id) ON DELETE CASCADE,
    form           text    NOT NULL,      -- 活用形（例: 괜찮아）
    pronunciations text[]  NOT NULL DEFAULT '{}',
    PRIMARY KEY (entry_id, form)
);
CREATE INDEX idx_krdict_forms_form ON krdict_forms (form);
```

## 4. 取り込みコマンド

`backend/cmd/krdictimport` を追加します。

```bash
cd backend
go run ./cmd/krdictimport -src ../data/krdict/전체_내려받기_한국어기초사전_json_20260919.zip
```

- ZIP を直接読む（展開しない）。1ファイル約100MBあるので、**`json.Decoder` で `LexicalEntry` を1件ずつ読む**（全体をメモリに載せない）
- 1回のトランザクションで参照テーブルを入れ替える（何度実行しても同じ結果になる）
- 実行後に件数を出力する: 見出し語数、日本語訳のある見出し語の割合、発音のある見出し語の割合
  - [../krdict-data.md](../krdict-data.md) の実測値（56,555 / 97.0% / 82.5%）と大きくずれたら、パースの誤りを疑って報告する

### 整形

公式の値は**意味を変えずに**整形するだけにします。

| 項目 | 整形 |
|---|---|
| 発音 | 長音記号 `ː` と半角の `:` を除く。前後の空白を除く |
| 発音（異常値） | ハングル音節と空白以外の文字（漢字・字母単体・`/` など）を含むものは、**取り込まずに警告として一覧に出す** |
| 日本語訳 | 前後のタブ・空白を除く。`(対訳語無し)` は NULL にする |
| 同形語番号 | 無い場合は 0 |

## 5. ドメインとリポジトリ

- `internal/domain/krdict/` に見出し語・語義・活用形のモデルと `Repository` インターフェースを置く
  - `FindByHeadword(ctx, headword)`、`FindByForm(ctx, form)`、`FindEntry(ctx, id)`、`FindSense(ctx, entryID, senseNo)`
- 実装は `internal/infra/postgres/krdict_repository.go`
- JSON のパース処理は `infra` 側に置く（`domain` は JSON の形式を知らない）

## 6. テスト

- パース: `feat` が配列のときとオブジェクトのときの両方、`(対訳語無し)`、長音記号、異常値の発音
- テスト用の小さな JSON を `testdata/` に置く（実データを丸ごと置かない）

## 7. ドキュメント

| ファイル | 更新内容 |
|---|---|
| `docs/folder-structure.md` | `domain/krdict`、`cmd/krdictimport`、`data/`（追跡しない）、`docs/tasks/`、`docs/krdict-data.md` |
| `README.md` | セットアップに「ZIP を `data/krdict/` に置く → `go run ./cmd/krdictimport -src ...`」を追加（NeonDB でも同じ手順） |
| `.devcontainer/post-create.sh` | `data/krdict/` に ZIP があれば `krdictimport` を実行する案内を出す（自動実行はしない） |

## 完了の定義

`CLAUDE.md` の「完了の定義」に加えて:

- `docker compose down -v && docker compose up -d db` → `krdictimport` が通る
- 取り込み件数が実測値と一致する（ずれがあれば理由を PR に書く）
- 警告として除外した発音の一覧を PR に載せる
