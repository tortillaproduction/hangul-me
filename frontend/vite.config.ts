import { fileURLToPath, URL } from 'node:url';
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

const nextStub = fileURLToPath(new URL('./src/lib/next-stub.ts', import.meta.url));

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      // Next.js は不採用のため、@clerk/elements が任意peer依存として参照する
      // next/* をスタブへ差し替える（詳細は src/lib/next-stub.ts）
      'next/compat/router': nextStub,
      'next/navigation': nextStub,
    },
  },
  server: {
    // コンテナ内から起動する場合は 0.0.0.0 で待ち受ける必要がある
    host: true,
    port: 5173,
    proxy: {
      // 開発時は Go APIサーバーへプロキシする。
      // docker compose では VITE_API_PROXY_TARGET=http://backend:8080 を渡す。
      '/api': {
        target: process.env.VITE_API_PROXY_TARGET ?? 'http://localhost:8080',
        changeOrigin: true,
      },
    },
    watch: {
      // バインドマウント経由だと inotify が効かない環境があるため、必要時のみポーリング
      usePolling: process.env.VITE_USE_POLLING === 'true',
    },
  },
});
