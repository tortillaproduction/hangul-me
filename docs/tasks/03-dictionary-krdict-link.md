# 03. 辞書を公式辞典にひも付け、DBで正確さを保証する

- ブランチ: `feat/dictionary-krdict-link`
- 前提: 02 がマージ済み
- 参照: 共通の決まりは [README.md](README.md)

## 目的

「辞書DBには公式辞典で確認できたものしか入らない」を、アプリのコードではなく **DBの制約で保証** します。
あわせて、正確でない値が辞書に入る今の経路を2つ塞ぎます。

1. AIフォールバックの候補を誰かが登録すると、AIが作ったハングル・読み・意味がそのまま共有の辞書に入る（`usecase/registration` の `resolveEntry`）
2. 登録APIが、クライアントから送られたハングル・読み・意味をそのまま保存する（改ざんされた値が入り得る）

**注意:** このPRのマイグレーションで、開発用シードの15件は消えます。04 の初期投入が終わるまで、開発環境の辞書は空になります。

---

## 1. 用語を追加する（実装より先）

`docs/ubiquitous-language.md` に追加し、変更履歴に1行追記します。

| 用語 | コード上の名前 | 意味 |
|---|---|---|
| 辞典で確認できない語 | `unverified` | AIが出した候補のうち、公式辞典に見つからないもの。登録できない |

## 2. マイグレーション

`db/migrations/0004_dictionary_krdict_link.sql`

```sql
ALTER TABLE dictionary_entries
    ADD COLUMN krdict_entry_id integer,
    ADD COLUMN krdict_sense_no integer,
    ADD COLUMN form_source text;   -- 'headword' / 'official' / 'derived'

-- 公式辞典にひも付かない既存行（開発用シードの15件）を消す。
-- 単語帳から参照されている行があれば、黙って消さずにマイグレーションを失敗させる。
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM user_words uw
        JOIN dictionary_entries d ON d.id = uw.entry_id
        WHERE d.krdict_entry_id IS NULL
    ) THEN
        RAISE EXCEPTION '公式辞典にひも付かない単語が単語帳から参照されています。手動で移行してください';
    END IF;
END $$;
DELETE FROM dictionary_entries WHERE krdict_entry_id IS NULL;

ALTER TABLE dictionary_entries
    ALTER COLUMN krdict_entry_id SET NOT NULL,
    ALTER COLUMN krdict_sense_no SET NOT NULL,
    ALTER COLUMN form_source SET NOT NULL,
    ADD CONSTRAINT dictionary_entries_krdict_sense_fk
        FOREIGN KEY (krdict_entry_id, krdict_sense_no) REFERENCES krdict_senses (entry_id, sense_no),
    ADD CONSTRAINT dictionary_entries_form_source_check
        CHECK (form_source IN ('headword', 'official', 'derived')),
    ADD CONSTRAINT dictionary_entries_krdict_form_key
        UNIQUE (krdict_entry_id, krdict_sense_no, hangul);

-- source は公式辞典由来のみにする
ALTER TABLE dictionary_entries DROP CONSTRAINT IF EXISTS dictionary_entries_source_check;
UPDATE dictionary_entries SET source = 'krdict';
ALTER TABLE dictionary_entries
    ALTER COLUMN source SET DEFAULT 'krdict',
    ADD CONSTRAINT dictionary_entries_source_check CHECK (source = 'krdict');
```

- 制約名は実際のスキーマを確認して合わせること
- `domain/entry` の `Source` から `SourceAIGenerated`・`SourceUserSuggested`・`SourceCurated` を削除し、`SourceKrdict` だけにする
- `entry.Entry` に `KrdictEntryID`・`KrdictSenseNo`・`FormSource` を足す

## 3. 公式辞典から辞書の単語を組み立てる

`internal/domain/dictionary/`（新設）に、**公式辞典の見出し語・語義・形から `entry.Entry` を組み立てる唯一の処理**を置きます。04 の初期投入と、下の登録処理の両方がこれを使います。

入力: 見出し語の識別コード、語義番号、形（例: 괜찮아요）
出力: `entry.Entry`（またはエラー）

- 形が、見出し語そのもの・公式の活用形・02 の規則で作れる形のどれでもなければエラー
- ハングル: その形
- カナ読み・ひらがな・ローマ字: 形の発音から 02 の `PronunciationToKana`・`PronunciationToRomanized` で作る。発音が取れなければエラー
- 意味（`meaning_ja`）: 語義の日本語訳を**機械的に整形した値**
  - 【】があれば中身、無ければかな（例: 「だいじょうぶだ【大丈夫だ】」→「大丈夫だ」）
  - 「。」区切りは「 / 」に（例: 「ゆく・いく【行く】。うつる【移る】」→「行く / 移る」）
  - 日本語訳が無い語義はエラー
- 分析用カラム: `ApplyPhonetics()` で生成
- `FormSource`: 見出し語そのものなら `headword`、公式の活用形（＋요）なら `official`、規則で作った形なら `derived`

意味の整形規則はこのパッケージの1か所にまとめ、テストを書いてください。

## 4. AIフォールバックの候補の扱い

AIの役割を「うろ覚えの音から、**どの単語か当てる**」だけに限定します。AIが返した読みや意味は、**一切保存も表示もしません**。

`domain/matching` の `Service.Match` の手順4（AI候補の再照合）を次のように変えます。

1. AIが返したハングルを、辞書DB（`dictionary_entries`）で探す → あれば、その単語を候補にする（今と同じ）
2. 無ければ、公式辞典の参照テーブルで探す: 見出し語（`FindByHeadword`）、公式の活用形（`FindByForm`）、末尾の「요」を外した形
3. 見つかったら、**公式辞典の値で候補を組み立てる**（上の 3 の処理を使う）。語義が複数あれば、候補に語義の一覧を付ける
4. どちらにも無ければ「辞典で確認できない語」として返す（`Origin: unverified`）

## 5. 登録の扱い

`usecase/registration` を変えます。**クライアントから送られたハングル・読み・意味は信用しません。**

- `Input` から `Hangul`・`ReadingKana`・`ReadingHiragana`・`Romanized`・`MeaningJA` を削除する
- 辞書DBにある候補: 今と同じく `EntryID` で登録
- 参照テーブルで見つかった候補: `KrdictEntryID`・`KrdictSenseNo`・`Form` を受け取り、サーバー側で上の 3 の処理を通して `dictionary_entries` に追加してから登録する
- 「辞典で確認できない語」: **登録できない**。APIは 422 を返す

## 6. 画面（この範囲だけ変える）

- 候補一覧で「辞典で確認できない語」には「辞典で確認できない語のため、登録できません」と表示し、登録ボタンを出さない
- 登録モーダルで、語義が複数ある候補は語義（日本語訳）を選べるようにする。語義が1つなら選択肢を出さない

UI文言はユビキタス言語一覧に従ってください。

## 7. テスト

- `domain/dictionary`: 形の種類ごとの組み立て、意味の整形、規則で作れない形・日本語訳の無い語義・発音の無い形がエラーになること
- `domain/matching`: AI候補が参照テーブルで見つかったときに公式の値で候補が組み立てられること、見つからないときに `unverified` になること、AIが返した読み・意味が候補に混ざらないこと
- `usecase/registration`: 辞典で確認できない語の登録が拒否されること、存在しない語義番号が拒否されること

## 8. ドキュメント

| ファイル | 更新内容 |
|---|---|
| `docs/folder-structure.md` | `domain/dictionary` |
| `README.md` | 「主要な設計判断」に追記: 辞書は公式辞典にひも付く／AIは単語を当てるだけで、値は公式辞典から作る。「候補マッチング」の手順5を書き換える。APIの登録リクエストの変更 |
| `CLAUDE.md` | 「設計上の約束」に追記: 辞書DBには公式辞典にひも付く単語しか入れない／ハングル・読み・意味を手書き・AI生成で入れない／`entry.Entry` の組み立ては `domain/dictionary` を通す。「候補マッチング」の項の「AI が生成した候補は、必ず辞書DBと再照合」を、公式辞典との照合に書き換える |

## 完了の定義

`CLAUDE.md` の「完了の定義」に加えて:

- `docker compose down -v && docker compose up -d db` でマイグレーションが最初から通る
- `SELECT count(*) FROM dictionary_entries WHERE krdict_entry_id IS NULL;` が 0
