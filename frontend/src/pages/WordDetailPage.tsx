import { useCallback, useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import type { Word } from '../api/types';
import { useToast } from '../hooks/useToast';
import { api, ApiError } from '@/api/client';
import { ArrowLeftIcon } from '@/components/icons';

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
    let alive = true;
    setLoading(true);

    api
      .getWord(id)
      .then((w) => {
        if (!alive) return;
        setWord(w);
        setMemo(w.memo);
        setTitle(w.encounteredTitle);
      })
      .catch(() => alive && setWord(null))
      .finally(() => alive && setLoading(false));

    return () => {
      alive = false;
    };
  }, [id]);

  const save = useCallback(async () => {
    if (!word || saving) return;
    setSaving(true);

    try {
      const updated = await api.updateWord(word.id, { memo, encounteredTitle: title });
      setWord(updated);
      showToast('success', 'メモを保存しました');
    } catch (err) {
      showToast('error', '保存できませんでした', err instanceof ApiError ? err.message : undefined);
    } finally {
      setSaving(false);
    }
  }, [memo, saving, showToast, title, word]);

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

  return (
    <div className="mx-auto max-w-4xl px-6 py-7">
      <button
        type="button"
        onClick={() => navigate('/words')}
        className="mb-5 flex items-center gap-1.5 text-sm opacity-60 transition hover:opacity-100"
      >
        <ArrowLeftIcon width={16} height={16} />
        登録単語一覧
      </button>

      <section className="rounded-2xl border border-surface-border bg-surface-raised px-6 py-7">
        <p className="font-ko text-4xl font-bold">{word.hangul}</p>
        <p className="mt-2 text-lg opacity-80">{word.readingKana}</p>
        <p className="mt-0.5 text-sm opacity-45">{word.romanized}</p>
        <p className="mt-4 text-base">{word.meaningJa}</p>

        <div className="mt-5 flex flex-wrap gap-2 text-xs">
          <Chip label={`初声 ${word.initialConsonant || '—'}`} />
          <Chip
            label={word.hasFinalConsonant ? `パッチム ${word.finalConsonant}` : 'パッチムなし'}
          />
          <Chip label={word.masteryLabel} />
        </div>
      </section>

      <div className="mt-4 grid gap-4 md:grid-cols-2">
        <Card title="例文">
          {word.examples.length === 0 ? (
            <p className="text-sm opacity-40">例文はまだありません</p>
          ) : (
            <ul className="space-y-3">
              {word.examples.map((ex, i) => (
                <li key={i}>
                  <p className="font-ko text-sm">{ex.sentenceKo}</p>
                  <p className="mt-0.5 text-xs opacity-60">{ex.sentenceJa}</p>
                  {ex.sourceTitle && (
                    <p className="mt-0.5 text-[11px] opacity-40">{ex.sourceTitle}</p>
                  )}
                </li>
              ))}
            </ul>
          )}
        </Card>

        <Card title="どこで聞いた？">
          <input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="ドラマ・映画のタイトルなど"
            className="w-full rounded-lg border border-surface-border bg-surface-base px-3 py-2 text-sm outline-none focus:border-brand-400"
          />
          <p className="mt-2 text-[11px] opacity-40">
            記録しておくと、作品ごとの登録数を「学習の記録」で見られます。
          </p>
        </Card>

        <Card title="メモ">
          <textarea
            value={memo}
            onChange={(e) => setMemo(e.target.value)}
            rows={4}
            placeholder="覚え方や気づいたことを書いておけます"
            className="w-full resize-none rounded-lg border border-surface-border bg-surface-base px-3 py-2 text-sm outline-none focus:border-brand-400"
          />
        </Card>

        <Card title="発音音声">
          <p className="text-sm opacity-40">追加予定です</p>
        </Card>
      </div>

      <div className="mt-5 flex justify-end">
        <button
          type="button"
          onClick={() => void save()}
          disabled={saving}
          className="rounded-lg bg-brand-400 px-4 py-2 text-sm font-semibold text-brand-900 transition hover:bg-brand-300 disabled:opacity-50"
        >
          {saving ? '保存中...' : '保存する'}
        </button>
      </div>
    </div>
  );
}

function Card({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="rounded-xl border border-surface-border bg-surface-raised px-5 py-4">
      <h2 className="mb-3 text-xs font-semibold uppercase tracking-wide opacity-50">{title}</h2>
      {children}
    </section>
  );
}

function Chip({ label }: { label: string }) {
  return (
    <span className="rounded-full border border-surface-border px-2.5 py-1 opacity-70">
      {label}
    </span>
  );
}
