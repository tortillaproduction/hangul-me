# フォルダストラクチャ

プロジェクトの構成と、各ディレクトリが何を担当するかの一覧です。

> **このファイルは常に最新に保ちます。**
> ディレクトリやファイルを追加・移動・削除したら、同じコミットでこの図と `README.md` を更新してください（`CLAUDE.md` 参照）。

---

## 全体図

```
hangul-me/
├── .devcontainer/                  VS Code Dev Container 設定
│   ├── devcontainer.json             Go / Node / Docker / GitHub CLI / Claude Code と拡張機能
│   └── post-create.sh                作成後の依存取得と .env のひな形作成
│
├── .vscode/
│   └── settings.json                 保存時整形・Go/ESLintの共通設定
│
├── backend/                        Go APIサーバー（DDD構成）
│   ├── cmd/
│   │   ├── api/                      APIサーバーのエントリポイント
│   │   └── rollup/                   閲覧ログの日次ロールアップ（バッチ）
│   ├── config/                       環境変数の読み込み
│   ├── internal/
│   │   ├── domain/                   【ドメイン層】ビジネスルールの中心。外部に依存しない
│   │   │   ├── analytics/              集計モデル（可視化の軸の定義）
│   │   │   ├── entry/                  辞書エントリ（単語）とリポジトリ境界
│   │   │   ├── hangul/                 ハングル字母分解・音韻分析
│   │   │   ├── matching/               読みの正規化と候補マッチングのドメインサービス
│   │   │   ├── user/                   利用者・プラン
│   │   │   └── userword/               単語帳（登録単語）
│   │   ├── usecase/                  【ユースケース層】ドメインを組み合わせた操作の単位
│   │   │   ├── analytics/              学習の記録（集計の取得）
│   │   │   ├── lookup/                 単語を調べる
│   │   │   ├── registration/           候補を単語帳へ登録する
│   │   │   └── wordbook/               単語帳の参照・更新・削除
│   │   ├── infra/                    【インフラ層】外部サービスとの接続。domainの境界を実装
│   │   │   ├── authz/                  Clerk のJWT検証（JWKS）／開発用の簡易検証
│   │   │   ├── llm/                    AIフォールバック（Anthropic Messages API）
│   │   │   └── postgres/               NeonDB(PostgreSQL) のリポジトリ実装
│   │   └── interface/                【インターフェース層】外部との入出力
│   │       └── http/                   ルーティング・ハンドラ・ミドルウェア
│   ├── Dockerfile                    開発用 / 本番用のマルチステージ定義
│   ├── .dockerignore
│   ├── .env.example
│   ├── go.mod / go.sum
│   └── （各パッケージ内の *_test.go）  Ginkgo + Gomega によるBDDテスト
│
├── frontend/                       Vite + React + TypeScript
│   ├── src/
│   │   ├── api/                      APIクライアントとレスポンス型
│   │   ├── components/               画面をまたいで使う部品
│   │   │   ├── charts/                 グラフ部品（横棒・推移）
│   │   │   └── toast-context.ts        トーストのContext（Providerと別ファイル）
│   │   ├── features/                 機能単位のまとまり
│   │   │   └── lookup/                 単語を調べるモーダル
│   │   ├── hooks/                    再利用するReactフック（テーマ切り替え・トースト）
│   │   ├── lib/                      補助モジュール（Next.js用スタブなど）
│   │   ├── pages/                    ルーティング単位の画面
│   │   ├── styles/                   Tailwindの入口とネオンロゴ等の共通スタイル
│   │   ├── App.tsx                   ルーティングと認証状態の出し分け
│   │   └── main.tsx                  エントリポイント（ClerkProvider）
│   ├── Dockerfile                    開発用 / 本番用（静的配信）のマルチステージ定義
│   ├── nginx.conf                    本番の静的配信設定（SPAのフォールバック）
│   ├── .dockerignore
│   ├── .env.example
│   ├── eslint.config.js              ESLint（Flat Config）
│   ├── .prettierrc / .prettierignore Prettier
│   ├── index.html
│   ├── package.json
│   ├── tailwind.config.js            ブランドカラーとdaisyUIテーマ
│   ├── tsconfig*.json
│   └── vite.config.ts
│
├── db/                             データベース資産
│   ├── migrations/
│   │   ├── 0001_init.sql               スキーマ定義（拡張・テーブル・インデックス）
│   │   └── 0002_seed_dictionary.sql    開発用のシードデータ
│   ├── jobs/
│   │   └── rollup_views.sql            閲覧ログの日次ロールアップ＋90日パージ
│   └── analytics.sql                 可視化クエリのリファレンス
│
├── docs/                           プロジェクトの共有ドキュメント
│   ├── folder-structure.md           このファイル
│   └── ubiquitous-language.md        ユビキタス言語一覧（用語対応表）
│
├── compose.yaml                    開発用スタックの一括起動（db / backend / frontend / rollup）
├── .env.example                    compose が読む環境変数のひな形
├── CLAUDE.md                       Claude Code 向けのプロジェクト指示書
├── README.md                       セットアップと設計判断
└── .gitignore
```

## compose のサービス

| サービス | 役割 | 備考 |
|---|---|---|
| `db` | PostgreSQL 16 | 初回起動で `db/migrations/` を自動実行。ロケールは `C.UTF-8`（pg_trgm がかなを扱えるように） |
| `backend` | Go APIサーバー | `go run ./cmd/api`。ソースはバインドマウント |
| `frontend` | Vite 開発サーバー | `/api` を `backend:8080` へプロキシ |
| `rollup` | 閲覧ログの日次ロールアップ | 常駐しない。`docker compose run --rm rollup` で実行（profile: `tools`）|

---

## 依存の向き（バックエンド）

DDDの依存関係は常に**内向き**です。`domain` は他のどの層も知りません。

```
interface/http  ──▶  usecase  ──▶  domain
                         │            ▲
                         ▼            │
                       infra ─────────┘
                  （domain が定義した
                    リポジトリ境界を実装する）
```

- `domain` に `database/sql` や `net/http` を持ち込まないこと
- 永続化や外部API呼び出しは、`domain` 側のインターフェースを `infra` が実装する形で繋ぐこと
- `interface/http` はHTTPの都合（JSON・ステータスコード）だけを引き受け、判断は `usecase` に委ねること

## 配置の指針（フロントエンド）

| 置き場所 | 基準 |
|---|---|
| `components/` | 2つ以上の画面から使う、または汎用的な部品 |
| `features/<機能名>/` | 特定の機能にしか使わない部品。機能ごとに閉じる |
| `pages/` | ルーティング1つに対応する画面。データ取得の起点 |
| `hooks/` | 状態や副作用の再利用単位 |
| `api/` | サーバーとの通信と型。ここ以外で `fetch` を直接呼ばない |
