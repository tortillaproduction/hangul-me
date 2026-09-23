# syntax=docker/dockerfile:1

# ---------------------------------------------------------------------------
# dev : ソースをバインドマウントして go run で起動する開発用ステージ
# ---------------------------------------------------------------------------
FROM golang:1.24-bookworm AS dev

WORKDIR /app

# 依存だけ先に解決してレイヤーキャッシュを効かせる
COPY go.mod go.sum ./
RUN go mod download

COPY . .

EXPOSE 8080
CMD ["go", "run", "./cmd/api"]

# ---------------------------------------------------------------------------
# builder : 本番用バイナリのビルド
# ---------------------------------------------------------------------------
FROM golang:1.24-bookworm AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# 静的リンクにして distroless で動かせるようにする
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/rollup ./cmd/rollup

# ---------------------------------------------------------------------------
# runtime : 本番用。シェルを持たない最小イメージ
# ---------------------------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

WORKDIR /app
COPY --from=builder /out/api /app/api
COPY --from=builder /out/rollup /app/rollup

USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/api"]
