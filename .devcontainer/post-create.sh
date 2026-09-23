#!/usr/bin/env bash
# Dev Container 作成後に一度だけ走るセットアップ。
# 失敗しても開発は続けられるよう、致命的でないものは警告に留める。
set -uo pipefail

echo "▶ Hangul-me の開発環境をセットアップします"

# --- マウントしたボリュームの所有者を合わせる -------------------------------
sudo chown -R "$(id -u):$(id -g)" "$HOME/.claude" "$HOME/.config/gh" 2>/dev/null || true

# --- .env のひな形を用意 ----------------------------------------------------
for pair in ".:.env.example" "backend:.env.example" "frontend:.env.example"; do
  dir="${pair%%:*}"
  example="${pair##*:}"
  if [ -f "$dir/$example" ] && [ ! -f "$dir/.env" ]; then
    cp "$dir/$example" "$dir/.env"
    echo "  ✔ $dir/.env を作成しました（値は各自で設定してください）"
  fi
done

# --- Go の依存 --------------------------------------------------------------
echo "▶ Go の依存を取得します"
if (cd backend && go mod download && go mod tidy); then
  echo "  ✔ backend の依存を取得しました"
else
  echo "  ⚠ backend の依存取得に失敗しました。ネットワークを確認して 'cd backend && go mod tidy' を手動で実行してください"
fi

# --- Node の依存 ------------------------------------------------------------
echo "▶ Node の依存を取得します"
if (cd frontend && npm ci); then
  echo "  ✔ frontend の依存を取得しました"
else
  echo "  ⚠ frontend の依存取得に失敗しました。'cd frontend && npm install' を手動で実行してください"
fi

# --- 確認 -------------------------------------------------------------------
echo ""
echo "▶ 利用できるツール"
for cmd in go node npm docker gh claude; do
  if command -v "$cmd" > /dev/null 2>&1; then
    printf '  ✔ %-7s %s\n' "$cmd" "$("$cmd" --version 2>/dev/null | head -1)"
  else
    printf '  ✘ %-7s 見つかりません\n' "$cmd"
  fi
done

cat <<'MSG'

セットアップが完了しました。

  docker compose up            開発スタック一式（db / backend / frontend）を起動
  cd backend && go test ./...  バックエンドのテスト
  cd frontend && npm run dev   フロントエンドだけを起動

作業を始める前に CLAUDE.md を確認してください（ブランチ運用と用語の決まりがあります）。
MSG
