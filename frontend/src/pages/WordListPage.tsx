import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type { Word } from '../api/types';
import { api } from '@/api/client';
import { SearchPlusIcon } from '@/components/icons';

interface WordListPageProps {
  onOpenLookup: () => void;
  /** 登録直後に一覧を取り直すためのキー */
  refreshKey: number;
}

/** ログイン後の既定表示。登録済みの単語を一覧します。 */
export function WordListPage({ onOpenLookup, refreshKey }: WordListPageProps) {
  const [words, setWords] = useState<Word[]>([]);
  const [loading, setLoading] = useState(true);
  const navigate = useNavigate();

  useEffect(() => {
    let alive = true;
    setLoading(true);

    api
      .listWords()
      .then((list) => alive && setWords(list))
      .catch(() => alive && setWords([]))
      .finally(() => alive && setLoading(false));

    return () => {
      alive = false;
    };
  }, [refreshKey]);

  const thisWeek = words.filter((w) => {
    const d = new Date(w.registeredAt).getTime();
    return Date.now() - d < 7 * 24 * 60 * 60 * 1000;
  }).length;

  return (
    <div className="mx-auto max-w-5xl px-6 py-7">
      <div className="mb-6 flex items-center justify-between gap-4">
        <h1 className="text-xl font-bold">登録単語一覧</h1>
        <button
          type="button"
          className="flex items-center gap-2 rounded-lg bg-brand-400 px-3.5 py-2 text-sm font-semibold text-brand-900 transition hover:bg-brand-300"
        >
          <SearchPlusIcon />
          単語を調べる
        </button>
      </div>

      <div className="mb-6 grid gap-3 sm:grid-cols-3">
        <Stat label="登録した単語" value={`${words.length}`} unit="語" />
        <Stat label="今週の登録" value={`${thisWeek}`} unit="語" />
        <div className="rounded-xl border border-dashed border-surface-border px-4 py-3">
          <p className="text-xs opacity-50">復習が必要な単語</p>
          <p className="mt-1 text-sm opacity-40">近日公開</p>
        </div>
      </div>

      <div className="overflow-x-hidden rounded-xl border border-surface-border bg-surface-raised">
        {loading ? (
          <p className="px-5 py-12 text-center text-sm opacity-50">読み込み中...</p>
        ) : words.length === 0 ? (
          <div className="px-5 py-14 text-center">
            <p className="text-sm opacity-60">まだ単語がありません</p>
            <button
              type="button"
              onClick={onOpenLookup}
              className="mt-3 text-sm font-medium text-brand-300 underline-offset-4"
            >
              うろ覚えの読みから調べてみる
            </button>
          </div>
        ) : (
          <table className="w-full text-sm">
            <tbody>
              {words.map((w) => (
                <tr
                  key={w.id}
                  onClick={() => navigate(`/words/${w.id}`)}
                  className="cursor-pointer border-b border-surface-border transition last:border-0 hover:bg-surface-hover"
                >
                  <td className="px-5 py-3 font-ko text-base font-semibold">{w.hangul}</td>
                  <td className="px-5 py-3 opacity-75">{w.readingKana}</td>
                  <td className="px-5 py-3 text-xs opacity-50">{w.romanized}</td>
                  <td className="px-5 py-3 opacity-80">{w.meaningJa}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}

function Stat({ label, value, unit }: { label: string; value: string; unit: string }) {
  return (
    <div className="rounded-xl border border-surface-border bg-surface-raised px-4 py-3">
      <p className="text-xs opacity-50">{label}</p>
      <p className="mt-1 text-2xl font-bold">
        {value}
        <span className="ml-1 text-xs font-normal opacity-50">{unit}</span>
      </p>
    </div>
  );
}
