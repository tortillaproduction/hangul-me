import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type { Word } from '../api/types';

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
    // ここに単語一覧を取得するロジックを記述
  }, [refreshKey]);

  const thisWeek = words.filter((w) => {
    const d = new Date(w.registeredAt).getTime();
    return Date.now() - d < 7 * 24 * 60 * 60 * 1000;
  }).length;

  return ()
}

function Stat({ label, value, unit }: { label: string; value: string; unit: string}) {
    return (
        <div className="rounded-xl border border-surface-border bg-surface-raised px-4 py-3">
            <p className="text-xs opacity-50">{label}</p>
            <p className="mt-1 text-2xl font-bold">
                {value}
                <span className="ml-1 text-xs font-normal opacity-50">{unit}</span>
            </p>
        </div>
    )
}
