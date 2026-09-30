// Package llm は 候補不足時のAIフォールバックです。
//
// DB曖昧検索で十bんな候補が得られなかった時だけ呼ばれます。
// 生成結果は matching.service 側で必ず辞書DBと再照合されます。
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hangulme/hangul-me/backend/internal/domain/entry"
	"github.com/hangulme/hangul-me/backend/internal/domain/matching"
)

// Client は Anthropic Message API を使った候補生成器です。
type Client struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

var _ matching.AIGenerator = (*Client)(nil)

// Option は Client の設定です。
type Option func(*Client)

func WithModel(m string) Option   { return func(c *Client) { c.model = m } }
func WithBaseURL(u string) Option { return func(c *Client) { c.baseURL = u } }
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.httpClient = h }
}

// New は候補生成クライアントを作ります。
// apiKey が空の場合は nil を返し、呼び出し側はAIフォールバック無しで動きます。
func New(apiKey string, opts ...Option) *Client {
	if apiKey == "" {
		return nil
	}
	c := &Client{
		apiKey:     apiKey,
		model:      "claude-sonnet-4-5",
		baseURL:    "https://api.anthropic.com",
		httpClient: &http.Client{Timeout: 12 * time.Second},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

const systemPrompt = `あなたは韓国語学習アプリの候補生成エンジンです。
日本人がドラマや映画で聞き取った「うろ覚えの韓国語の音」から、実在する韓国語の単語・フレーズを推測します。

制約：
- 実在する韓国語のみ。創作した語は返さない
- 日常会話・ドラマで実際に使われる表現を優先する
- 読みは日本語話者が発音しやすいカタカナ表記にする
- 確信が持てない場合は件数を減らしてよい。無理に埋めない

出力は必ず次のJSONのみ。説明文やコードフェンスは付けない。
{"candidates":[{"hangul":"안녕","reading_kana":"アンニョン","reading_hiragana":"あんにょん","romanized":"annyeong","meaning_ja":"やあ／こんにちは"}]}`

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

type candidatePayload struct {
	Candidates []struct {
		Hangul          string `json:"hangul"`
		ReadingKana     string `json:"reading_kana"`
		ReadingHiragana string `json:"reading_hiragana"`
		Romanized       string `json:"romanized"`
		MeaningJA       string `json:"meaning_ja"`
	} `json:"candidates"`
}

// Suggest は正規化済みの読みから候補を生成します。
func (c *Client) Suggest(ctx context.Context, normalized, raw string, limit int) ([]entry.Entry, error) {
	if limit <= 0 {
		limit = 5
	}
	userMsg := fmt.Sprintf(
		"うろ覚えの読み: 「%s」（正規化後: %s）\nこの音に対応しそうな韓国語を最大%d件、JSONで返してください。",
		raw,
		normalized,
		limit,
	)

	body, err := json.Marshal(anthropicRequest{
		Model:     c.model,
		MaxTokens: 1024,
		System:    systemPrompt,
		Messages:  []anthropicMessage{{Role: "user", Content: userMsg}},
	})
	if err != nil {
		return nil, fmt.Errorf("llm: リクエストの組み立てに失敗しました: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("llm: リクエストの作成に失敗しました: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("llm: 呼び出しに失敗しました: %w", err)
	}
	defer resp.Body.Close()

	raw2, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("llm: レスポンスの読み取りに失敗しました: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("llm: 異常応答 status=%d body=%s", resp.StatusCode, string(raw2))
	}

	var ar anthropicResponse
	if err := json.Unmarshal(raw2, &ar); err != nil {
		return nil, fmt.Errorf("llm: レスポンスの解析に失敗しました: %w", err)
	}

	var text string
	for _, blk := range ar.Content {
		if blk.Type == "text" {
			text += blk.Text
		}
	}
	return parseCandidates(text, limit)
}

// parseCandidates はモデルの出力からJSONを取り出して候補に変換します。
// コードフェンスや前後の説明文が混ざっても最初のJSONオブジェクトを拾います。
func parseCandidates(text string, limit int) ([]entry.Entry, error) {
	start := bytes.IndexByte([]byte(text), '{')
	end := bytes.LastIndexByte([]byte(text), '}')
	if start < 0 || end <= start {
		return nil, fmt.Errorf("llm: JSON が見つかりませんでした")
	}

	var payload candidatePayload
	if err := json.Unmarshal([]byte(text[start:end+1]), &payload); err != nil {
		return nil, fmt.Errorf("llm: JSONの解析に失敗しました: %w", err)
	}

	out := make([]entry.Entry, 0, len(payload.Candidates))
	for _, c := range payload.Candidates {
		e, err := entry.New(
			c.Hangul,
			c.ReadingKana,
			c.ReadingHiragana,
			c.Romanized,
			c.MeaningJA,
			entry.SourceAIGenerated,
		)
		if err != nil {
			continue // 欠損のある候補は黙って捨てる
		}
		out = append(out, *e)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}
