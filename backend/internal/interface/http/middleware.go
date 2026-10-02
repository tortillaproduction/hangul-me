package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/hangulme/hangul-me/backend/internal/domain/user"
	"github.com/hangulme/hangul-me/backend/internal/infra/authz"
)

type ctxKey string

const ctxKeyUser ctxKey = "currentUser"

// CurrentUser はコンテキストから認証済み利用者を取り出します。
func CurrentUser(ctx context.Context) (*user.User, bool) {
	u, ok := ctx.Value(ctxKeyUser).(*user.User)
	return u, ok
}

// TokenVerifier は Clerk トークンの検証境界です（テストで差し替え可能）。
type TokenVerifier interface {
	Verify(ctx context.Context, token string) (*authz.Claims, error)
}

// Authenticator は Bearer トークンを検証し、認証済み利用者をコンテキストに格納します。
// 未登録ユーザーは初回アクセス時に自動でプロビジョニングされます。
type Authenticator struct {
	Verifier TokenVerifier
	users    user.Repository
	logger   *slog.Logger
}

func NewAuthenticator(
	v TokenVerifier,
	users user.Repository,
	logger *slog.Logger,
) *Authenticator {
	return &Authenticator{
		Verifier: v,
		users:    users,
		logger:   logger,
	}
}

// Middleware は認証必須ルートに適用します。
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized", "ログインが必要です")
			return
		}

		claims, err := a.Verifier.Verify(r.Context(), token)
		if err != nil {
			if !errors.Is(err, authz.ErrNoToken) {
				a.logger.Debug("トークンの検証に失敗", slog.String("error", err.Error()))
			}
			writeError(w, http.StatusUnauthorized, "unauthorized", "セッションの有効期限が切れています")
			return
		}

		u, err := a.users.EnsureByClerkID(r.Context(), claims.ClerkUserID, claims.Email, claims.DisplayName)
		if err != nil {
			a.logger.Error("ユーザーのプロビジョニングに失敗", slog.String("error", err.Error()))
			writeError(w, http.StatusInternalServerError, "internal", "ユーザー情報の取得に失敗しました")
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyUser, u)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}

	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	return strings.TrimSpace(parts[1])
}

// CORS は許可オリジンからのアクセスだけを通します。
func CORS(allowed []string) func(http.Handler) http.Handler {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, o := range allowed {
		allowedSet[o] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if _, ok := allowedSet[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type")
				w.Header().Set("Access-Control-Max-Age", "600")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequestLogger はアクセスログを出します。
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			logger.Info(
				"request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rec.status),
				slog.Duration("took", time.Since(start)),
			)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// Recoverer は panic を 500 に変換します。
func ReCoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic", slog.Any("recovered", rec), slog.String("path", r.URL.Path))
					writeError(w, http.StatusInternalServerError, "internal", "サーバーエラーが発生しました")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
