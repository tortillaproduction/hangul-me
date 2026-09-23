-- 閲覧ログの日次ロールアップ + 90日パージ
-- 日次バッチ（cron / Neon scheduled job / GoのCLI）から実行する想定。
-- 冪等: 同じ日に複数回実行しても結果は変わらない。

BEGIN;

-- 1) 生ログを日単位に集計して upsert
INSERT INTO user_word_daily_views (user_word_id, view_date, view_count)
SELECT user_word_id,
       viewed_at::date AS view_date,
       COUNT(*)        AS view_count
FROM word_view_logs
WHERE viewed_at::date <= current_date          -- 未来日は対象外
GROUP BY user_word_id, viewed_at::date
ON CONFLICT (user_word_id, view_date)
DO UPDATE SET view_count = EXCLUDED.view_count; -- 集計値で上書き（再実行で二重加算しない）

-- 2) 集計済みの古い生ログを削除（90日より前）
DELETE FROM word_view_logs
WHERE viewed_at::date <= (current_date - INTERVAL '90 days')::date;

COMMIT;
