// Command rollup は 閲覧ログの日次ロールアップと古い生ログの削除を行います。
//
// 日次バッチ（cron / スケジュールジョブ）から実行してください。
// 冪等なので、同じ日に複数回実行しても結果は変わりません。
//
// go run ./cmd/rollup
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/hangulme/hangul-me/backend/config"
	"github.com/hangulme/hangul-me/backend/internal/infra/postgres"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("設定の読み込みに失敗しました", slog.String("error", err.Error()))
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	db, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("DB接続に失敗しました", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() { _ = db.Close() }()

	repo := postgres.NewAnalyticsRepository(db)
	rolled, purged, err := repo.RollupViews(ctx, cfg.ViewRetentionDays)
	if err != nil {
		logger.Error("ロールアップに失敗しました", slog.String("error", err.Error()))
		os.Exit(1)
	}

	logger.Info(
		"ロールアップ完了",
		slog.Int64("rolledUpRows", rolled),
		slog.Int64("purgedRawLogs", purged),
		slog.Int("retentionDays", cfg.ViewRetentionDays),
	)
}
