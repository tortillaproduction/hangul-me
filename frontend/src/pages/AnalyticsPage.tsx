import { useEffect, useState } from 'react';
import { api } from '@/api/client';
import type { AnalyticsOverview } from '../api/types';
import { BarList, type BarDatum } from '../components/charts/BarList';
import { TrendChart } from '@/components/charts/TrendChart';

/**
 * 学習の記録（分析ビュー）。
 *
 * どのグラフも系列は一つなので、凡例は置かず値を直接ラベルしています。
 * 色はブランドのvioletを面ごとに検証した段階で統一しています。
 */
export function AnalyticsPage() {
  const [data, setData] = useState<AnalyticsOverview | null>(null);
  const [loading, setLoading] = useState(true);
  const [showTable, setShowTable] = useState(false);

  useEffect(() => {
    let alive = true;
    api
      .analytics()
      .then((d) => alive && setData(d))
      .catch(() => alive && setData(null))
      .finally(() => alive && setLoading(false));

    return () => {
      alive = false;
    };
  }, []);

  if (loading) {
    return <p className="px-6 py-12 text-center text-sm opacity-50">読み込み中...</p>;
  }

  if (!data) {
    return (
      <p className="px-6 py-12 text-center text-sm opacity-50">集計を取得できませんてした。</p>
    );
  }

  const initials: BarDatum[] = data.initialConsonants.map((b) => ({
    label: b.label,
    value: b.count,
  }));

  // 「パッチムなし」は音の種類ではなく有無の話なので、
  // 分布のグラフからは外し、上の一文でまとめて示す。
  const finals: BarDatum[] = data.finalConsonants
    .filter((b) => b.consonant != '')
    .map((b) => ({
      label: b.label,
      value: b.count,
    }));

  const noFinalCount = data.finalConsonants.find((b) => b.consonant == '')?.count ?? 0;

  const withFinalCount = finals.reduce((s, d) => s + d.value, 0);

  const titles: BarDatum[] = data.encounteredTitles.map((b) => ({
    label: b.label,
    value: b.count,
  }));

  const viewed: BarDatum[] = data.mostViewed
    .filter((w) => w.count > 0)
    .map((w) => ({
      label: w.hangul,
      sub: w.readingKana,
      value: w.count,
    }));

  const searched: BarDatum[] = data.mostSearched.map((w) => ({
    label: w.hangul,
    sub: w.readingKana,
    value: w.count,
  }));

  return (
    <div className="viz-root mx-auto max-w-5xl px-6 py-7">
      <div className="mb-6 flex items-center justify-between gap-4">
        <h1 className="text-xl font-bold">学習の記録</h1>
        <button
          type="button"
          onClick={() => setShowTable((v) => !v)}
          className="rounded-lg border border-surface-border px-3 py-1.5 text-xs opacity-70 transition hover:bg-surface-hover hover:opacity-100"
        >
          {showTable ? 'グラフで見る' : '表で見る'}
        </button>
      </div>

      <div className="mb-4 grid sm:grid-cols-2 gap-3">
        <Stat label="登録した単語" value={data.totalWords} unit="語" />
        <Stat label="今週の登録" value={data.registeredThisWeek} unit="語" />
      </div>

      {showTable ? (
        <TableView data={data} />
      ) : (
        <div className="grid md:grid-cols-2 gap-4">
          <Panel
            title="登録数の推移"
            note="直近30日。続けられているかを見るための目安です。"
            className="md:col-span-2"
          >
            <TrendChart data={data.registrationTrend} />
          </Panel>

          <Panel
            title="初声（語頭の子音）の偏り"
            note="どの音から始まる単語をよく拾っているかが分かります。"
          >
            <BarList data={initials} unit="語" emptyMessage="まだ単語がありません" />
          </Panel>

          <Panel
            title="パッチムの分布"
            note={`語末の音の傾向。パッチムあり ${withFinalCount}語 / なし ${noFinalCount}語。`}
          >
            <BarList data={finals} unit="語" emptyMessage="パッチムのある単語がまだありません" />
          </Panel>

          <Panel title="どこで聞いた？" note="作品ごとの登録数。何をきっかけに覚えたかの記録です。">
            <BarList data={titles} unit="語" emptyMessage="まだ記録がありません" />
          </Panel>

          <Panel title="よく見返している単語" note="個別ページを開いた記録の多い順。">
            <BarList data={viewed} unit="語" emptyMessage="まだ閲覧の記録がありません" />
          </Panel>

          <Panel
            title="よく調べた単語"
            note="検索して登録した回数の多い順。"
            className="md:col-span-2"
          >
            <BarList data={searched} unit="語" emptyMessage="まだ検索の記録がありません" />
          </Panel>
        </div>
      )}
    </div>
  );
}

function Stat({ label, value, unit }: { label: string; value: number; unit: string }) {
  return (
    <div className="rounded-xl border border-surface-border bg-surface-raised px-4 py-3">
      <p className="text-xs opacity-50">{label}</p>
      <p className="mt-1 text-2xl font-bold tabular-nums">
        {value}
        <span className="ml-1 text-xs font-normal opacity-50">{unit}</span>
      </p>
    </div>
  );
}

function Panel({
  title,
  note,
  className = '',
  children,
}: {
  title: string;
  note?: string;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <section
      className={`rounded-xl border border-surface-border bg-surface-raised px-5 py-4 ${className}`}
    >
      <h2 className="text-sm font-semibold">{title}</h2>
      {note && <p className="mb-4 mt-0.5 text-[11px] opacity-45">{note}</p>}
      {children}
    </section>
  );
}

/** グラフを読めない環境向けの表ビュー */
function TableView({ data }: { data: AnalyticsOverview }) {
  const sections: { title: string; rows: [string, number][]; unit: string }[] = [
    {
      title: '初声の偏り',
      rows: data.initialConsonants.map((b) => [b.label, b.count]),
      unit: '語',
    },
    {
      title: 'パッチムの分布',
      rows: data.finalConsonants.map((b) => [b.label, b.count]),
      unit: '語',
    },
    {
      title: 'どこで聞いた？',
      rows: data.encounteredTitles.map((b) => [b.label, b.count]),
      unit: '語',
    },
    {
      title: 'よく見返している単語',
      rows: data.mostViewed.map((w) => [`${w.hangul}（${w.readingKana}）`, w.count]),
      unit: '回',
    },
    {
      title: 'よく調べた単語',
      rows: data.mostSearched.map((w) => [`${w.hangul}（${w.readingKana}）`, w.count]),
      unit: '回',
    },
  ];

  return (
    <div className="grid gap-4 md:grid-cols-2">
      {sections.map((s) => (
        <section
          key={s.title}
          className="overflow-hidden rounded-xl border border-surface-border bg-surface-raised"
        >
          <h2 className="border-b border-surface-border px-5 py-3 text-sm font-semibold">
            {s.title}
          </h2>
          {s.rows.length === 0 ? (
            <p className="px-5 py-6 text-sm opacity-40">データがありません</p>
          ) : (
            <table className="w-full text-sm">
              <tbody>
                {s.rows.map(([label, count], i) => (
                  <tr key={i} className="border-b border-surface-border last:border-0">
                    <td className="px-5 py-2">{label}</td>
                    <td className="px-5 py-2 text-right tabular-nums opacity-70">
                      {count}
                      {s.unit}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </section>
      ))}
    </div>
  );
}
