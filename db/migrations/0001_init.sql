-- Hangul-me 初期スキーマ
-- 対象: PostgreSQL 15+ (NeonDB)
--
-- 【前提】データベースのロケールは UTF-8 系（C.UTF-8 など）であること。
--   locale=C のまま作ると pg_trgm がカタカナ・ひらがなを単語構成文字として扱わず、
--   show_trgm('あんにょん') が空配列を返します。この状態ではかな入力の
--   曖昧検索がまったく効かず、前方一致にしかヒットしません。
--   確認方法:
--     SELECT show_trgm('あんにょん');   -- 空でなければOK
--   NeonDB はプロジェクト作成時に UTF-8 系ロケールになります。

BEGIN;

CREATE EXTENSION IF NOT EXISTS "pgcrypto";   -- gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS "pg_trgm";    -- 読みの曖昧検索

-- ---------------------------------------------------------------------------
-- users : Clerk連携・課金プラン
-- ---------------------------------------------------------------------------
CREATE TABLE users (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    clerk_user_id text        NOT NULL UNIQUE,
    email         text        NOT NULL UNIQUE,
    display_name  text        NOT NULL DEFAULT '',
    plan          text        NOT NULL DEFAULT 'free'
                              CHECK (plan IN ('free', 'premium')),
    word_limit    int         NOT NULL DEFAULT 100,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------------------
-- dictionary_entries : 正規辞書マスタ
--   initial_consonant / has_final_consonant / final_consonant は
--   「どの音でつまずくか」を可視化するための分析用カラム。
--   登録時にハングルから機械的に導出して保存する。
-- ---------------------------------------------------------------------------
CREATE TABLE dictionary_entries (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    hangul              text        NOT NULL,
    reading_kana        text        NOT NULL,
    reading_hiragana    text        NOT NULL DEFAULT '',
    romanized           text        NOT NULL DEFAULT '',
    meaning_ja          text        NOT NULL,
    source              text        NOT NULL DEFAULT 'curated'
                                    CHECK (source IN ('curated', 'ai_generated', 'user_suggested')),
    initial_consonant   text        NOT NULL DEFAULT '',
    has_final_consonant boolean     NOT NULL DEFAULT false,
    final_consonant     text,
    audio_url           text,
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT dictionary_entries_hangul_meaning_key UNIQUE (hangul, meaning_ja)
);

-- 読みの曖昧検索（pg_trgm）。カタカナ/ひらがな/ローマ字の3系統に張る
CREATE INDEX idx_entries_kana_trgm      ON dictionary_entries USING gin (reading_kana gin_trgm_ops);
CREATE INDEX idx_entries_hiragana_trgm  ON dictionary_entries USING gin (reading_hiragana gin_trgm_ops);
CREATE INDEX idx_entries_romanized_trgm ON dictionary_entries USING gin (romanized gin_trgm_ops);
CREATE INDEX idx_entries_initial        ON dictionary_entries (initial_consonant);
CREATE INDEX idx_entries_final          ON dictionary_entries (final_consonant);

-- ---------------------------------------------------------------------------
-- example_sentences : 例文（1エントリに複数）
-- ---------------------------------------------------------------------------
CREATE TABLE example_sentences (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_id     uuid NOT NULL REFERENCES dictionary_entries(id) ON DELETE CASCADE,
    sentence_ko  text NOT NULL,
    sentence_ja  text NOT NULL,
    source_title text,
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_examples_entry ON example_sentences (entry_id);

-- ---------------------------------------------------------------------------
-- user_words : 個人の単語帳
--   encountered_title = 「この単語はどこで聞いた？」のメモ
-- ---------------------------------------------------------------------------
CREATE TABLE user_words (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    entry_id          uuid NOT NULL REFERENCES dictionary_entries(id) ON DELETE CASCADE,
    memo              text,
    encountered_title text,
    mastery_level     int  NOT NULL DEFAULT 0 CHECK (mastery_level BETWEEN 0 AND 3),
    registered_at     timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT user_words_user_entry_key UNIQUE (user_id, entry_id)
);

CREATE INDEX idx_user_words_user       ON user_words (user_id, registered_at DESC);
CREATE INDEX idx_user_words_encountered ON user_words (user_id, encountered_title);

-- ---------------------------------------------------------------------------
-- search_queries : 検索ログ（マッチング精度改善・人気単語集計）
--   選択された候補のみ selected_entry_id に残す
-- ---------------------------------------------------------------------------
CREATE TABLE search_queries (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           uuid REFERENCES users(id) ON DELETE SET NULL,
    raw_input         text NOT NULL,
    normalized_input  text NOT NULL DEFAULT '',
    matched_via       text NOT NULL CHECK (matched_via IN ('db_trgm', 'ai_fallback', 'none')),
    selected_entry_id uuid REFERENCES dictionary_entries(id) ON DELETE SET NULL,
    created_at        timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_search_selected ON search_queries (selected_entry_id)
    WHERE selected_entry_id IS NOT NULL;
CREATE INDEX idx_search_user     ON search_queries (user_id, created_at DESC);

-- ---------------------------------------------------------------------------
-- audio_assets : v2 発音音声（Cloudflare R2）
-- ---------------------------------------------------------------------------
CREATE TABLE audio_assets (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_id      uuid NOT NULL REFERENCES dictionary_entries(id) ON DELETE CASCADE,
    r2_object_key text NOT NULL,
    voice_type    text NOT NULL DEFAULT 'tts' CHECK (voice_type IN ('native', 'tts')),
    duration_ms   int  NOT NULL DEFAULT 0,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_audio_entry ON audio_assets (entry_id);

-- ---------------------------------------------------------------------------
-- word_view_logs : 個別ページ閲覧の生ログ（直近90日のみ保持）
-- ---------------------------------------------------------------------------
CREATE TABLE word_view_logs (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_word_id uuid NOT NULL REFERENCES user_words(id) ON DELETE CASCADE,
    viewed_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_view_logs_viewed_at ON word_view_logs (viewed_at);
CREATE INDEX idx_view_logs_user_word ON word_view_logs (user_word_id, viewed_at DESC);

-- ---------------------------------------------------------------------------
-- user_word_daily_views : 閲覧の日次集計（恒久保持）
-- ---------------------------------------------------------------------------
CREATE TABLE user_word_daily_views (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_word_id uuid NOT NULL REFERENCES user_words(id) ON DELETE CASCADE,
    view_date    date NOT NULL,
    view_count   int  NOT NULL DEFAULT 0,
    CONSTRAINT user_word_daily_views_key UNIQUE (user_word_id, view_date)
);

CREATE INDEX idx_daily_views_date ON user_word_daily_views (view_date);

COMMIT;
