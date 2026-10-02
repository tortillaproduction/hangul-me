// Package config 環境変数からの設定読み込みです。
package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

// Config アプリ全体の設定です。
type Config struct {
	Port string
	// DatabaseURL は NeonDB の接続文字列です。
	DatabaseURL string
	// ClerkIssuer は Clerk の Frontend API URL（JWKS取得元）。
	ClerkIssuer string
	// ClerkAudience は 任意。設定した場合のみ aud を検証します。
	ClerkAudience string
	// AnthropicAPIKey が空ならAIフォールバックは無効になります。
	AnthropicAPIKey string
	AnthropicModel  string
	// AllowedOrigins は CORS 許可オリジン。
	AllowedOrigins []string
	// ViewRetentionDays は 閲覧生ログの保持日数。
	ViewRetentionDays int
}

var ErrMissingDatabaseURL = errors.New("config: DATABASE_URL が設定されていません")

// Load は環境変数から設定を読み込みます。
func Load() (*Config, error) {
	c := &Config{
		Port:              getenv("PORT", "8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		ClerkIssuer:       strings.TrimRight(os.Getenv("CLERK_ISSUER"), "/"),
		ClerkAudience:     os.Getenv("CLERK_AUDIENCE"),
		AnthropicAPIKey:   os.Getenv("ANTHROPIC_API_KEY"),
		AnthropicModel:    getenv("ANTHROPIC_MODEL", "claude-sonnet-4-5"),
		AllowedOrigins:    splitCSV(getenv("ALLOWED_ORIGINS", "http://localhost:5173")),
		ViewRetentionDays: getenvInt("VIEW_RETENTION_DAYS", 90),
	}

	if c.DatabaseURL == "" {
		return nil, ErrMissingDatabaseURL
	}

	return c, nil
}

// AuthEnabled は Clerk 検証を有効にするかを返します。
// 未設定ならローカル開発用のダミー認証で動かします。
func (c *Config) AuthEnabled() bool { return c.ClerkIssuer != "" }

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getenvInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
