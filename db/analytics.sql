-- 可視化・集計用クエリ集
-- アプリのダッシュボード（分析ビュー）から実行することを想定したリファレンス。

-- ---------------------------------------------------------------------------
-- 1. アプリ全体でよく検索（＝選択）された単語トップ10
-- ---------------------------------------------------------------------------
SELECT d.id,
       d.hangul,
       d.reading_kana,
       d.meaning_ja,
       COUNT(*) AS search_count
FROM search_queries sq
JOIN dictionary_entries d ON d.id = sq.selected_entry_id
WHERE sq.selected_entry_id IS NOT NULL
GROUP BY d.id
ORDER BY search_count DESC, d.hangul
LIMIT 10;

-- ---------------------------------------------------------------------------
-- 2. ユーザー自身がよく検索した単語トップ10
-- ---------------------------------------------------------------------------
SELECT d.hangul, d.reading_kana, d.meaning_ja, COUNT(*) AS search_count
FROM search_queries sq
JOIN dictionary_entries d ON d.id = sq.selected_entry_id
WHERE sq.user_id = $1
  AND sq.selected_entry_id IS NOT NULL
GROUP BY d.id
ORDER BY search_count DESC
LIMIT 10;

-- ---------------------------------------------------------------------------
-- 3. よく見返している単語トップ10（生ログ＋日次集計を合算）
--    90日以内は word_view_logs、それ以前は user_word_daily_views に存在する。
--    二重計上を避けるため、生ログ側はパージ境界より新しい分だけ数える。
-- ---------------------------------------------------------------------------
WITH purge_boundary AS (
    SELECT (current_date - INTERVAL '90 days')::date AS d
),
recent AS (
    SELECT uw.id AS user_word_id, COUNT(*) AS cnt
    FROM word_view_logs wvl
    JOIN user_words uw ON uw.id = wvl.user_word_id
    CROSS JOIN purge_boundary pb
    WHERE uw.user_id = $1
      AND wvl.viewed_at::date > pb.d
    GROUP BY uw.id
),
archived AS (
    SELECT uwdv.user_word_id, SUM(uwdv.view_count) AS cnt
    FROM user_word_daily_views uwdv
    JOIN user_words uw ON uw.id = uwdv.user_word_id
    CROSS JOIN purge_boundary pb
    WHERE uw.user_id = $1
      AND uwdv.view_date <= pb.d
    GROUP BY uwdv.user_word_id
)
SELECT d.hangul,
       d.reading_kana,
       d.meaning_ja,
       COALESCE(r.cnt, 0) + COALESCE(a.cnt, 0) AS view_count
FROM user_words uw
JOIN dictionary_entries d ON d.id = uw.entry_id
LEFT JOIN recent   r ON r.user_word_id = uw.id
LEFT JOIN archived a ON a.user_word_id = uw.id
WHERE uw.user_id = $1
ORDER BY view_count DESC, uw.registered_at DESC
LIMIT 10;

-- ---------------------------------------------------------------------------
-- 4. 初声（初出頭子音）ごとの登録数 ＝ どの音に偏っているか
-- ---------------------------------------------------------------------------
SELECT d.initial_consonant, COUNT(*) AS word_count
FROM user_words uw
JOIN dictionary_entries d ON d.id = uw.entry_id
WHERE uw.user_id = $1
  AND d.initial_consonant <> ''
GROUP BY d.initial_consonant
ORDER BY word_count DESC;

-- ---------------------------------------------------------------------------
-- 5. パッチムの有無・種類の分布
-- ---------------------------------------------------------------------------
SELECT d.has_final_consonant,
       COALESCE(d.final_consonant, '(なし)') AS final_consonant,
       COUNT(*) AS word_count
FROM user_words uw
JOIN dictionary_entries d ON d.id = uw.entry_id
WHERE uw.user_id = $1
GROUP BY d.has_final_consonant, d.final_consonant
ORDER BY word_count DESC;

-- ---------------------------------------------------------------------------
-- 6. 作品別の登録数（「どこで聞いた？」メモの集計）
-- ---------------------------------------------------------------------------
SELECT COALESCE(NULLIF(uw.encountered_title, ''), '(未記入)') AS encountered_title,
       COUNT(*) AS word_count
FROM user_words uw
WHERE uw.user_id = $1
GROUP BY 1
ORDER BY word_count DESC;

-- ---------------------------------------------------------------------------
-- 7. 登録数の推移（日別・直近90日）
-- ---------------------------------------------------------------------------
SELECT uw.registered_at::date AS day, COUNT(*) AS registered
FROM user_words uw
WHERE uw.user_id = $1
  AND uw.registered_at >= current_date - INTERVAL '90 days'
GROUP BY day
ORDER BY day;
