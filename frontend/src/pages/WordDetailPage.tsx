import { useCallback, useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import type { Word } from '../api/types';
import { useToast } from '../hooks/useToast';

/**
 * 単語の個別ページ。
 * 表示のたびにサーバー側で閲覧が1件記録され、「よく見返した単語」の集計に使われます。
 */
export function WordDetailPage() {
  const { id = '' } = useParams();
  const navigate = useNavigate();
  const { showToast } = useToast();

  const [word, setWord] = useState<Word | null>(null);
  const [loading, setLoading] = useState(true);
  const [memo, setMemo] = useState('');
  const [title, setTitle] = useState('');
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    // ここに単語の情報を取得するロジックを記述
  }, [id]);

  const save = useCallback(async () => {}, []);

  if (loading) {
    return <p className="px-6 py-12 text-center text-sm opacity-50">読み込み中...</p>;
  }

  if (!word) {
    return (
      <div className="px-6 py-12 text-center">
        <p className="text-sm opacity-60">単語が見つかりませんでした。</p>
        <button
          type="button"
          onClick={() => navigate('/words')}
          className="mt-3 text-sm text-brand-300 hover:underline"
        >
          登録単語一覧へ戻る
        </button>
      </div>
    );
  }

  return ()
}

function Card({ title, children }: { title: string; children: React.ReactNode }) {
    return (
        <section className="rounded-xl border border-surface-border bg-surface-raised px-5 py-4">
            <h2 className="mb-3 text-xs font-semibold uppercase tracking-wide opacity-50">{title}</h2>
            {children}
        </section>
    )
}

function Chip({ label }: { label: string }) {
    return (
        <span className="rounded-full border border-surface-border px-2.5 py-1 opacity-70">
            {label}
        </span>
    )
}
