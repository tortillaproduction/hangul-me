// Command api は Hangul-me の APIサーバーです。
package main

import (
	"context"
	"errors"
	"log/slog"
	stdhttp "net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hangulme/hangul-me/backend/config"
	"github.com/hangulme/hangul-me/backend/internal/domain/matching"
	"github.com/hangulme/hangul-me/backend/internal/infra/authz"
	"github.com/hangulme/hangul-me/backend/internal/infra/llm"
	"github.com/hangulme/hangul-me/backend/internal/infra/postgres"
	httpiface "github.com/hangulme/hangul-me/backend/internal/interface/http"
	ucanalytics "github.com/hangulme/hangul-me/backend/internal/usecase/analytics"
	"github.com/hangulme/hangul-me/backend/internal/usecase/lookup"
	"github.com/hangulme/hangul-me/backend/internal/usecase/registration"
	"github.com/hangulme/hangul-me/backend/internal/usecase/wordbook"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	if err := run(logger); err != nil {
		logger.Error("起動に失敗しました", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	// infra
	users := postgres.NewUserRepository(db)
	entries := postgres.NewEntryRepository(db)
	userWords := postgres.NewUserWordRepository(db)
	searchLogs := postgres.NewSearchLogRepository(db)
	analyticsRepo := postgres.NewAnalyticsRepository(db)

	// AIフォールバック（APIキー未設定ならDB検索のみで動く）
	var ai matching.AIGenerator
	if client := llm.New(cfg.AnthropicAPIKey, llm.WithModel(cfg.AnthropicModel)); client != nil {
		ai = client
		logger.Info("AIフォールバックを有効化しました", slog.String("model", cfg.AnthropicModel))
	} else {
		logger.Info("AIフォールバックは無効化です（ANTHROPIC_API_KEY 未設定）")
	}

	// domain / usecase
	matcher := matching.NewService(entries, ai, matching.DefaultConfig())
	lookupUC := lookup.New(matcher)
	registerUC := registration.New(users, entries, userWords, searchLogs)
	wordbookUC := wordbook.New(userWords, entries, userWords)
	analyticsUC := ucanalytics.New(analyticsRepo)

	// interface
	var verifier httpiface.TokenVerifier
	if cfg.AuthEnabled() {
		verifier = authz.NewVerifier(cfg.ClerkIssuer, cfg.ClerkAudience)
		logger.Info("Clerk 検証を有効化しました", slog.String("issuer", cfg.ClerkIssuer))
	} else {
		verifier = authz.NewDevVerifier()
		logger.Warn("CLERK_ISSUER が未設定のため開発用の簡易認証で起動します")
	}

	handler := httpiface.NewHandler(
		lookupUC,
		registerUC,
		wordbookUC,
		analyticsUC,
		entries,
		logger,
	)
	auth := httpiface.NewAuthenticator(verifier, users, logger)
	router := httpiface.Router(handler, auth, cfg.AllowedOrigins, logger)

	srv := &stdhttp.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("APIサーバーを起動しました", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, stdhttp.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		logger.Info("シャットダウンします")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}
