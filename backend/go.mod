module github.com/hangulme/hangul-me/backend

go 1.27

require (
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/google/uuid v1.6.0
	github.com/lib/pq v1.10.9
)

require (
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
)

// テスト依存（Ginkgo/Gomega）。
// ビルド環境から Go モジュールプロキシへ到達できなかったため go.sum 未生成です。
// 初回のみ `go mod tidy` を実行してください（cmd/api のビルドには影響しません）。
require (
	github.com/onsi/ginkgo/v2 v2.22.0
	github.com/onsi/gomega v1.36.1
)
