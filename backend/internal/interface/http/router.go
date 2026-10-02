package http

import (
	"log/slog"
	"net/http"
)

// Router は API のルーティングを組み立てます。
// Go 1.22 以降の ServeMux（メソッド・パス変数対応）を使い、外部ルータには依存しません。
func Router(h *Handler, auth *Authenticator, allowedOrigins []string, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	// 認証不要
	mux.HandleFunc("GET /healthz", h.Health)
	mux.HandleFunc("GET /api/analytics/trending", h.TrendingWords)

	// 認証必須
	protected := http.NewServeMux()
	protected.HandleFunc("GET /api/me", h.Me)
	protected.HandleFunc("GET /api/lookup", h.Lookup)
	protected.HandleFunc("POST /api/words", h.Register)
	protected.HandleFunc("GET /api/words/{id}", h.GetWord)
	protected.HandleFunc("PATCH /api/words/{id}", h.UpdateWord)
	protected.HandleFunc("DELETE /api/words/{id}", h.DeleteWord)
	protected.HandleFunc("GET /api/analytics", h.Analytics)

	mux.Handle("/api/", auth.Middleware(protected))

	return chain(
		mux,
		ReCoverer(logger),
		RequestLogger(logger),
		CORS(allowedOrigins),
	)
}

// chain はミドルウェアを外側から順に適用します。
func chain(h http.Handler, mw ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}
