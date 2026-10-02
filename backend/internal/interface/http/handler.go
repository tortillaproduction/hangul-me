package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	domainanalytics "github.com/hangulme/hangul-me/backend/internal/domain/analytics"
	"github.com/hangulme/hangul-me/backend/internal/domain/entry"
	"github.com/hangulme/hangul-me/backend/internal/domain/matching"
	"github.com/hangulme/hangul-me/backend/internal/domain/userword"
	ucanalytics "github.com/hangulme/hangul-me/backend/internal/usecase/analytics"
	"github.com/hangulme/hangul-me/backend/internal/usecase/lookup"
	"github.com/hangulme/hangul-me/backend/internal/usecase/registration"
	"github.com/hangulme/hangul-me/backend/internal/usecase/wordbook"
)

// Handler は HTTP ハンドラ一式です。
type Handler struct {
	lookup    *lookup.UseCase
	register  *registration.UseCase
	wordbook  *wordbook.UseCase
	analytics *ucanalytics.UseCase
	entries   entry.Repository
	logger    *slog.Logger
}

func NewHandler(
	l *lookup.UseCase,
	reg *registration.UseCase,
	wb *wordbook.UseCase,
	an *ucanalytics.UseCase,
	entries entry.Repository,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		lookup:    l,
		register:  reg,
		wordbook:  wb,
		analytics: an,
		entries:   entries,
		logger:    logger,
	}
}

// --- 単語を調べる -----------------------------------------------------------

type candidateResponse struct {
	EntryID     *uuid.UUID `json:"entryId"`
	Hangul      string     `json:"hangul"`
	ReadingKana string     `json:"readingKana"`
	Romanized   string     `json:"romanized"`
	MeaningJA   string     `json:"meaningJa"`
}

type lookupResponse struct {
	Query      string              `json:"query"`
	MatchedVia string              `json:"matchedVia"`
	Candidates []candidateResponse `json:"candidates"`
}

// Lookup は GET /api/lookup?q=... - 入力中のリアルタイム検索用。
func (h *Handler) Lookup(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	out, err := h.lookup.Search(r.Context(), q)
	if errors.Is(err, matching.ErrEmptyQuery) {
		writeJSON(w, http.StatusOK, lookupResponse{Query: q, MatchedVia: "none", Candidates: []candidateResponse{}})
		return
	}
	if err != nil {
		h.fail(w, err, "候補の検索に失敗しました")
		return
	}

	resp := lookupResponse{
		Query:      out.Query,
		MatchedVia: out.MatchedVia,
		Candidates: make([]candidateResponse, 0, len(out.Candidates)),
	}

	for _, c := range out.Candidates {
		resp.Candidates = append(resp.Candidates, candidateResponse{
			EntryID:     c.EntryID,
			Hangul:      c.Hangul,
			ReadingKana: c.ReadingKana,
			Romanized:   c.Romanized,
			MeaningJA:   c.MeaningJA,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- 単語帳 -----------------------------------------------------------------

type registerRequest struct {
	EntryID          *uuid.UUID `json:"entryId"`
	Hangul           string     `json:"hangul"`
	ReadingKana      string     `json:"readingKana"`
	ReadingHiragana  string     `json:"readingHiragana"`
	Romanized        string     `json:"romanized"`
	MeaningJA        string     `json:"meaningJa"`
	Memo             string     `json:"memo"`
	EncounteredTitle string     `json:"encounteredTitle"`
	RawQuery         string     `json:"rawQuery"`
	MatchedVia       string     `json:"matchedVia"`
}

// Register は POST /api/words - 候補を単語帳に登録します。
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	u, ok := CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "ログインが必要です")
		return
	}

	var req registerRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "リクエストの形式が正しくありません")
		return
	}

	got, err := h.register.Register(r.Context(), registration.Input{
		UserID:           u.ID,
		EntryID:          req.EntryID,
		Hangul:           req.Hangul,
		ReadingKana:      req.ReadingKana,
		ReadingHiragana:  req.ReadingHiragana,
		Romanized:        req.Romanized,
		MeaningJA:        req.MeaningJA,
		Memo:             req.Memo,
		EncounteredTitle: req.EncounteredTitle,
		RawQuery:         req.RawQuery,
		Normalized:       matching.Normalize(req.RawQuery),
		MatchedVia:       req.MatchedVia,
	})
	switch {
	case errors.Is(err, userword.ErrAlreadyExists):
		writeError(w, http.StatusConflict, "already_registered", "この単語はすでに単語帳にあります")
		return
	case errors.Is(err, userword.ErrLimitExceeded):
		writeError(w, http.StatusForbidden, "limit_exceeded", "登録できる単語数の上限に達しました")
		return
	case errors.Is(err, registration.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "bad_request", "登録に必要な情報が不足しています")
		return
	case err != nil:
		h.fail(w, err, "単語帳への登録に失敗しました")
		return
	}

	writeJSON(w, http.StatusCreated, toWordResponse(got))
}

// ListWords は GET /api/words - 単語帳の一覧を返します。
func (h *Handler) ListWords(w http.ResponseWriter, r *http.Request) {
	u, ok := CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "ログインが必要です")
		return
	}

	f := userword.ListFilter{
		EncounteredTitle: r.URL.Query().Get("encounteredTitle"),
		Limit:            atoiOr(r.URL.Query().Get("limit"), 50),
		Offset:           atoiOr(r.URL.Query().Get("offset"), 0),
	}

	words, err := h.wordbook.List(r.Context(), u.ID, f)
	if err != nil {
		h.fail(w, err, "単語一覧の取得に失敗しました")
		return
	}

	out := make([]wordResponse, 0, len(words))
	for _, x := range words {
		out = append(out, toWordResponse(x))
	}

	writeJSON(w, http.StatusOK, map[string]any{"words": out})
}

// GetWord は GET /api/words/{id} - 個別ページ。閲覧を1件記録します。
func (h *Handler) GetWord(w http.ResponseWriter, r *http.Request) {
	u, ok := CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "ログインが必要です")
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "IDの形式が正しくありません")
		return
	}

	got, err := h.wordbook.Detail(r.Context(), u.ID, id)
	if errors.Is(err, userword.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "単語が見つかりません")
		return
	}
	if err != nil {
		h.fail(w, err, "単語の取得に失敗しました")
		return
	}

	writeJSON(w, http.StatusOK, toWordResponse(got))
}

type updateWordRequest struct {
	Memo             *string `json:"memo"`
	EncounteredTitle *string `json:"encounteredTitle"`
	MasteryLevel     *int    `json:"masteryLevel"`
}

// UpdateWord は PATCH /api/words/{id}。
func (h *Handler) UpdateWord(w http.ResponseWriter, r *http.Request) {
	u, ok := CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "ログインが必要です")
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "IDの形式が正しくありません")
		return
	}

	var req updateWordRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "リクエストの形式が正しくありません")
		return
	}

	in := wordbook.UpdateInput{
		Memo:             req.Memo,
		EncounteredTitle: req.EncounteredTitle,
	}
	if req.MasteryLevel != nil {
		lv := userword.MasteryLevel(*req.MasteryLevel)
		in.MasteryLevel = &lv
	}

	got, err := h.wordbook.Update(r.Context(), u.ID, id, in)
	if errors.Is(err, userword.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "単語が見つかりません")
		return
	}
	if err != nil {
		h.fail(w, err, "単語の更新に失敗しました")
		return
	}

	writeJSON(w, http.StatusOK, toWordResponse(got))
}

// DeleteWord は DELETE /api/words/{id}。
func (h *Handler) DeleteWord(w http.ResponseWriter, r *http.Request) {
	u, ok := CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "ログインが必要です")
		return
	}

	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "IDの形式が正しくありません")
		return
	}

	if err := h.wordbook.Delete(r.Context(), u.ID, id); err != nil {
		if errors.Is(err, userword.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "単語が見つかりません")
			return
		}

		h.fail(w, err, "単語の削除に失敗しました")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// --- 分析 -------------------------------------------------------------------

// Analytics は GET /api/analytics - 分析ビュー用の集計一式です。
func (h *Handler) Analytics(w http.ResponseWriter, r *http.Request) {
	u, ok := CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "ログインが必要です")
		return
	}

	ov, err := h.analytics.Overview(r.Context(), u.ID)
	if err != nil {
		h.fail(w, err, "集計の取得に失敗しました")
		return
	}

	writeJSON(w, http.StatusOK, toAnalyticsResponse(ov))
}

// TrendingWords は GET /api/analytics/trending - アプリ全体の人気単語トップ10。
func (h *Handler) TrendingWords(w http.ResponseWriter, r *http.Request) {
	limit := atoiOr(r.URL.Query().Get("limit"), 10)
	words, err := h.analytics.GlobalMostSearched(r.Context(), limit)
	if err != nil {
		h.fail(w, err, "人気単語の取得に失敗しました")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"words": toRankedResponse(words)})
}

// Me は GET /api/me - ログイン中の利用者情報。
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	u, ok := CurrentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "ログインが必要です")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":          u.ID,
		"email":       u.Email,
		"displayName": u.DisplayName,
		"plan":        string(u.Plan),
		"wordLimit":   u.WordLimit,
	})
}

// Health は GET /api/health - ヘルスチェック。
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

// --- レスポンス変換 ---------------------------------------------------------

type exampleResponse struct {
	SentenceKO  string `json:"sentenceKo"`
	SentenceJA  string `json:"sentenceJa"`
	SourceTitle string `json:"sourceTitle,omitempty"`
}

type wordResponse struct {
	ID                string            `json:"id"`
	EntryID           string            `json:"entryId"`
	Hangul            string            `json:"hangul"`
	ReadingKana       string            `json:"readingKana"`
	Romanized         string            `json:"romanized"`
	MeaningJA         string            `json:"meaningJa"`
	Memo              string            `json:"memo"`
	EncounteredTitle  string            `json:"encounteredTitle"`
	MasteryLevel      int               `json:"masteryLevel"`
	MasteryLabel      string            `json:"masteryLabel"`
	InitialConsonant  string            `json:"initialConsonant"`
	HasFinalConsonant bool              `json:"hasFinalConsonant"`
	FinalConsonant    string            `json:"finalConsonant"`
	RegisteredAt      string            `json:"registeredAt"`
	Examples          []exampleResponse `json:"examples"`
}

func toWordResponse(w *userword.UserWord) wordResponse {
	out := wordResponse{
		ID:               w.ID.String(),
		EntryID:          w.EntryID.String(),
		Memo:             w.Memo,
		EncounteredTitle: w.EncounteredTitle,
		MasteryLevel:     int(w.MasteryLevel),
		MasteryLabel:     w.MasteryLevel.Label(),
		RegisteredAt:     w.RegisteredAt.Format("2006-01-02T15:04:05Z07:00"),
		Examples:         []exampleResponse{},
	}

	if w.Entry != nil {
		out.Hangul = w.Entry.Hangul
		out.ReadingKana = w.Entry.ReadingKana
		out.Romanized = w.Entry.Romanized
		out.MeaningJA = w.Entry.MeaningJA
		out.InitialConsonant = w.Entry.InitialConsonant
		out.HasFinalConsonant = w.Entry.HasFinalConsonant
		out.FinalConsonant = w.Entry.FinalConsonant
		for _, ex := range w.Entry.Examples {
			out.Examples = append(out.Examples, exampleResponse{
				SentenceKO:  ex.SentenceKO,
				SentenceJA:  ex.SentenceJA,
				SourceTitle: ex.SourceTitle,
			})
		}
	}
	return out
}

type rankedResponse struct {
	EntryID     string `json:"entryId"`
	Hangul      string `json:"hangul"`
	ReadingKana string `json:"readingKana"`
	MeaningJA   string `json:"meaningJa"`
	Count       int    `json:"count"`
}

func toRankedResponse(in []domainanalytics.RankedWord) []rankedResponse {
	out := make([]rankedResponse, 0, len(in))
	for _, w := range in {
		out = append(out, rankedResponse{
			EntryID:     w.EntryID.String(),
			Hangul:      w.Hangul,
			ReadingKana: w.ReadingKana,
			MeaningJA:   w.MeaningJA,
			Count:       w.Count,
		})
	}
	return out
}

func toAnalyticsResponse(ov *domainanalytics.Overview) map[string]any {
	consonants := func(in []domainanalytics.ConsonantBucket) []map[string]any {
		out := make([]map[string]any, 0, len(in))
		for _, b := range in {
			label := b.Consonant
			if label == "" {
				label = "なし"
			}
			out = append(out, map[string]any{
				"consonant": b.Consonant,
				"label":     label,
				"count":     b.Count,
			})
		}
		return out
	}

	titles := make([]map[string]any, 0, len(ov.EncounteredTitles))
	for _, t := range ov.EncounteredTitles {
		label := t.Title
		if label == "" {
			label = "未記入"
		}
		titles = append(titles, map[string]any{
			"title": t.Title,
			"label": label,
			"count": t.Count,
		})
	}

	trend := make([]map[string]any, 0, len(ov.RegistrationTrend))
	for _, d := range ov.RegistrationTrend {
		trend = append(trend, map[string]any{
			"date":  d.Date,
			"count": d.Count,
		})
	}

	return map[string]any{
		"totalWords":         ov.TotalWords,
		"registeredThisWeek": ov.RegisteredThisWeek,
		"mostSearched":       toRankedResponse(ov.MostSearched),
		"mostViewed":         toRankedResponse(ov.MostViewed),
		"initialConsonants":  consonants(ov.InitialConsonants),
		"finalConsonants":    consonants(ov.FinalConsonants),
		"encounteredTitles":  titles,
		"registrationTrend":  trend,
	}
}

// --- 共通ユーティリティ -----------------------------------------------------

func (h *Handler) fail(w http.ResponseWriter, err error, msg string) {
	h.logger.Error(msg, slog.String("error", err.Error()))
	writeError(w, http.StatusInternalServerError, "internal", msg)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func atoiOr(s string, def int) int {
	if s == "" {
		return def
	}

	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
