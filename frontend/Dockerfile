# syntax=docker/dockerfile:1

# ---------------------------------------------------------------------------
# dev : Vite の開発サーバーを立てるステージ
# ---------------------------------------------------------------------------
FROM node:26-bookworm-slim AS dev

WORKDIR /app

# 依存だけ先に入れてレイヤーキャッシュを効かせる
COPY package.json package-lock.json ./
RUN npm ci

COPY . .

EXPOSE 5173
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0"]

# ---------------------------------------------------------------------------
# builder : 本番用の静的ファイルを生成
# ---------------------------------------------------------------------------
FROM node:26-bookworm-slim AS builder

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .

# Clerk の Publishable Key はビルド時に埋め込まれる
ARG VITE_CLERK_PUBLISHABLE_KEY
ARG VITE_API_BASE_URL
ENV VITE_CLERK_PUBLISHABLE_KEY=$VITE_CLERK_PUBLISHABLE_KEY
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL

RUN npm run build

# ---------------------------------------------------------------------------
# runtime : 静的配信（Vercel を使わず自前で配る場合の選択肢）
# ---------------------------------------------------------------------------
FROM nginx:1.27-alpine AS runtime

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80
