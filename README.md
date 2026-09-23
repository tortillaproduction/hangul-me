# Hangul-me

聞こえたままの韓国語を、ハングルで読めるように。

ドラマや映画で耳に残った「うろ覚えの音」（カタカナ・ひらがな・ローマ字）から
正しいハングル・読み・意味・例文にたどり着き、自分の単語帳に貯めていくアプリです。

最終的な目的は、**ハングルを見て意味が分かり、声に出して読めるようになること**。
そのために「頭の中の音」と「ハングルという文字」を結びつける導線に絞って作っています。

---

## 構成

```
hangul-me/
├── .devcontainer/    Dev Container 設定（Go / Node / Docker / GitHub CLI / Claude Code）
├── backend/          Go / DDD（domain・usecase・infra・interface）
├── frontend/         Vite + React + TypeScript + React Router
├── db/
│   ├── migrations/   スキーマ定義とシードデータ
│   ├── jobs/         日次バッチ用SQL
│   └── analytics.sql 可視化クエリのリファレンス
├── docs/
│   ├── folder-structure.md     フォルダストラクチャ図（詳細）
│   └── ubiquitous-language.md  用語対応表（随時追記）
├── compose.yaml      開発スタックの一括起動
└── CLAUDE.md         Claude Code 向けの指示書
```

各ディレクトリの責務と依存の向きは **[docs/folder-structure.md](docs/folder-structure.md)** にまとめています。
開発の決まりごと（用語・ブランチ運用・ドキュメント更新）は **[CLAUDE.md](CLAUDE.md)** を参照してください。

| 領域 | 採用技術 |
|---|---|
| フロントエンド | Vite + React + TypeScript + React Router / Tailwind CSS + daisyUI（Vercel想定） |
| バックエンド | Go（DDD構成）/ テストは Ginkgo + Gomega |
| DB | NeonDB（PostgreSQL）+ `pg_trgm` |
| 認証 | Clerk（UIは Clerk Elements で自前構築。ブランディング表示なし） |
| オブジェクトストレージ | Cloudflare R2（v2の発音音声用、未実装） |
| AI補完 | Anthropic Messages API（候補不足時のみ呼び出し） |

---

## セットアップ

いちばん手軽なのは **Dev Container** か **docker compose** です。
個別に立ち上げたい場合は「手動セットアップ」へ進んでください。

### A. Dev Container で始める（推奨）

VS Code で「Reopen in Container」を選ぶだけで、
Go 1.24 / Node 22 / Docker / GitHub CLI / Claude Code が入った環境が立ち上がります。
依存の取得と `.env` のひな形作成も自動で行われます。

作成後、`.env` に `VITE_CLERK_PUBLISHABLE_KEY` を設定してから次へ進んでください。

### B. docker compose で始める

```bash
cp .env.example .env     # VITE_CLERK_PUBLISHABLE_KEY を設定
docker compose up
```

| サービス | ポート | 内容 |
|---|---|---|
| `frontend` | 5173 | Vite 開発サーバー（`/api` を backend へプロキシ） |
| `backend` | 8080 | Go APIサーバー |
| `db` | 5432 | PostgreSQL 16。初回起動時に `db/migrations/` が自動で流れます |

- `CLERK_ISSUER` を設定しない間は、バックエンドは開発用の簡易認証で動きます
  （`Authorization: Bearer <任意の文字列>` がそのままユーザー識別子になります）
- 閲覧ログのロールアップを試すとき: `docker compose run --rm rollup`
- スキーマを作り直すとき: `docker compose down -v && docker compose up -d db`

> **DBのロケールについて**
> pg_trgm はロケールが `C` のままだとカタカナ・ひらがなを単語構成文字として扱わず、
> かな入力の曖昧検索がまったく効きません（前方一致にしかヒットしません）。
> compose では `C.UTF-8` を指定しています。自前でDBを用意する場合も UTF-8 系にしてください。
> 確認: `SELECT show_trgm('あんにょん');` が空配列でなければOKです。

---

## 手動セットアップ

### 1. データベース

NeonDB でプロジェクトを作り、マイグレーションを流します。

```bash
psql "$DATABASE_URL" -f db/migrations/0001_init.sql
psql "$DATABASE_URL" -f db/migrations/0002_seed_dictionary.sql   # 開発用シード
```

### 2. バックエンド

```bash
cd backend
cp .env.example .env    # 値を埋める
go mod tidy             # 初回のみ（テスト依存の取得）
go run ./cmd/api
```

| 環境変数 | 必須 | 説明 |
|---|---|---|
| `DATABASE_URL` | ✓ | NeonDB の接続文字列 |
| `PORT` | | 既定 `8080` |
| `CLERK_ISSUER` | | Clerk の Frontend API URL。未設定だと開発用の簡易認証で動きます |
| `CLERK_AUDIENCE` | | 設定した場合のみ `aud` を検証 |
| `ANTHROPIC_API_KEY` | | 未設定なら AI フォールバックは無効（DB検索のみ） |
| `ANTHROPIC_MODEL` | | 既定 `claude-sonnet-4-5` |
| `ALLOWED_ORIGINS` | | CORS 許可オリジン（カンマ区切り）。既定 `http://localhost:5173` |
| `VIEW_RETENTION_DAYS` | | 閲覧生ログの保持日数。既定 `90` |

### 3. フロントエンド

```bash
cd frontend
cp .env.example .env    # VITE_CLERK_PUBLISHABLE_KEY を設定
npm install
npm run dev             # http://localhost:5173（/api は :8080 へプロキシ）
```

| スクリプト | 内容 |
|---|---|
| `npm run dev` | 開発サーバー |
| `npm run build` | 型チェック＋本番ビルド |
| `npm run typecheck` | 型チェックのみ |
| `npm run lint` | ESLint |
| `npm run format` | Prettier で整形 |

Clerk のダッシュボードで JWT テンプレートに `email` / `name` を含めておくと、
サーバー側でユーザー名とメールを同期できます。

---

## テスト

ドメイン層とユースケース層を Ginkgo（BDD）で書いています。

```bash
cd backend
go mod tidy
go test ./...
```

- `internal/domain/hangul` — 字母分解・音韻分析
- `internal/domain/matching` — 読みの正規化、DB検索→AIフォールバックの判断
- `internal/usecase/registration` — 候補の登録、重複・上限・検索ログ

---

## 主要な設計判断

### 候補マッチング（ハイブリッド）

1. 入力を正規化（カタカナ→ひらがな、小文字化、長音・記号の除去）
2. `pg_trgm` で辞書DBを曖昧検索（速い・安い）
3. 十分な候補が取れたらそこで終了（**AIは呼ばない**）
4. 足りないときだけ LLM に問い合わせ
5. 生成された候補は必ず辞書DBと再照合し、既知なら辞書側を正とする

AI呼び出しが失敗しても DB 候補だけで応答します。検索は落としません。

### 音韻分析カラム

`dictionary_entries` の `initial_consonant` / `has_final_consonant` / `final_consonant` は、
ハングルから機械的に導出して保存します（`internal/domain/hangul`）。
「どの音でつまずいているか」を後から集計できるようにするためで、
登録のたびに計算し直す必要がありません。

### 閲覧ログの二段構え

個別ページの閲覧は、生ログと日次集計の2テーブルで持ちます。

| テーブル | 粒度 | 保持 |
|---|---|---|
| `word_view_logs` | 閲覧1回ごと | 直近90日 |
| `user_word_daily_views` | 1単語1日1行 | 恒久 |

日次バッチ（`cmd/rollup` または `db/jobs/rollup_views.sql`）が生ログを集計へ丸め、
90日を過ぎた生ログを削除します。冪等なので複数回実行しても値は変わりません。

```bash
cd backend && go run ./cmd/rollup    # cron等から日次で
```

### 検索ログは「選んだ候補」だけ

`search_queries` には、ユーザーが実際に選択して登録した候補のみ記録します。
表示しただけの候補まで残すと、行数が跳ね上がるうえに分析上のノイズが大半になるためです。

---

## API

| メソッド | パス | 認証 | 説明 |
|---|---|:--:|---|
| GET | `/healthz` | – | ヘルスチェック |
| GET | `/api/me` | ✓ | ログイン中の利用者 |
| GET | `/api/lookup?q=` | ✓ | うろ覚えの読みから候補を返す |
| GET | `/api/words` | ✓ | 登録単語一覧 |
| POST | `/api/words` | ✓ | 候補を単語帳へ登録 |
| GET | `/api/words/{id}` | ✓ | 個別ページ（閲覧を1件記録） |
| PATCH | `/api/words/{id}` | ✓ | メモ・「どこで聞いた？」・定着度の更新 |
| DELETE | `/api/words/{id}` | ✓ | 単語帳から削除 |
| GET | `/api/analytics` | ✓ | 学習の記録（集計一式） |
| GET | `/api/analytics/trending` | – | アプリ全体の人気単語トップN |

---

## 未実装 / 今後

- 復習機能（`mastery_level` と閲覧ログは先に用意済み）
- 発音音声（Cloudflare R2 + `audio_assets`。スキーマのみ用意済み）
- 課金プラン（`users.plan` / `word_limit` で上限判定まで実装済み）
- プライバシーポリシー・利用規約ページ（サイドバーからのリンクのみ）
