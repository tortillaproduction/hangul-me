import { useEffect, useState } from 'react';
import type { CurrentUser } from '../api/types';
import { useTheme } from '../hooks/useTheme';
import { api } from '@/api/client';

export function SettingsPage() {
  const [me, setMe] = useState<CurrentUser | null>(null);
  const { theme, toggleTheme } = useTheme();

  useEffect(() => {
    let alive = true;
    api
      .me()
      .then((u) => alive && setMe(u))
      .catch(() => alive && setMe(null));

    return () => {
      alive = false;
    };
  }, []);

  return (
    <div className="mx-auto max-w-3xl px-6 py-7">
      <h1 className="mb-6 text-xl font-bold">アプリ設定</h1>

      <section className="mb-4 rounded-xl border border-surface-border bg-surface-raised px-5 py-4">
        <h2 className="mb-3 text-sm font-semibold">アカウント</h2>
        {me ? (
          <dl className="space-y-2 text-sm">
            <Row label="メールアドレス" value={me.email} />
            <Row label="表示名" value={me.displayName || '-'} />
            <Row label="プラン" value={me.plan === 'premium' ? 'プレミアム' : 'フリー'} />
            <Row label="登録できる単語数" value={`${me.wordLimit} 語まで`} />
          </dl>
        ) : (
          <p className="text-sm opacity-40">読み込み中...</p>
        )}
      </section>

      <section className="rounded-xl border border-surface-border bg-surface-raised px-5 py-4">
        <h2 className="mb-3 text-sm font-semibold">表示</h2>
        <div className="flex items-center justify-between">
          <div>
            <p className="text-sm">テーマ</p>
            <p className="mt-0.5 text-xs opacity-45">
              現在: {theme === 'hangulme-dark' ? 'ダーク' : 'ライト'}
            </p>
          </div>
          <button
            type="button"
            onClick={toggleTheme}
            className="rounded-lg border border-surface-border px-3 py-1.5 text-xs transition hover:bg-surface-hover"
          >
            切り替える
          </button>
        </div>
      </section>
    </div>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-4">
      <dt className="opacity-50">{label}</dt>
      <dd className="truncate">{value}</dd>
    </div>
  );
}
