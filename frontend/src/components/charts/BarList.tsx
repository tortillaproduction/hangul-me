import { useState } from 'react';

export interface BarDatum {
  label: string;
  value: number;
  /** ラベルの補足（ハングルの読みなど） */
  sub?: string;
}

interface BarListProps {
  data: BarDatum[];
  /** 値の単位（「語」「回」など） */
  unit: string;
  /** 表示件数の上限 */
  max?: number;
  emptyMessage?: string;
}

/**
 * 単一系列の量を表す横棒グラフ。
 *
 * 系列が1つなので凡例は置かず、値は各バーに直接ラベルします。
 * 色はブランドのviolet 1色（sequential）で、面ごとに検証済みの段階を使います。
 */
export function BarList({
  data,
  unit,
  max = 8,
  emptyMessage = 'データがありません',
}: BarListProps) {
  const [hovered, setHovered] = useState<number | null>(null);
  const shown = data.slice(0, max);
  const peak = Math.max(...shown.map((d) => d.value));

  if (shown.length === 0) {
    return <p className="py-6 text-center text-sm opacity-40">{emptyMessage}</p>;
  }

  return (
    <ul className="flex flex-col gap-2.5">
      {shown.map((d, i) => (
        <li
          key={`${d.label}-${i}`}
          onMouseEnter={() => setHovered(i)}
          onMouseLeave={() => setHovered(null)}
          className="grid grid-cols-[minmax(4.5rem,8rem)_1fr_auto] items-center gap-3"
        >
          <span className="min-w-0 truncate text-xs" style={{ color: 'var(--viz-text)' }}>
            {d.label}
            {d.sub && (
              <span className="ml-1 opacity-50" style={{ color: 'var(--viz-muted)' }}>
                {d.sub}
              </span>
            )}
          </span>

          {/* トラック上に実データのバーを重ねる。データ端は4px丸め、高さは細め */}
          <span
            className="relative block h-2 rounded-full"
            style={{ background: 'var(--viz-track)' }}
            role="img"
            aria-label={`${d.label}: ${d.value}${unit}`}
          >
            <span
              className="absolute inset-y-0 left-0 rounded-[4px] transition-[width,opacity] duration-300"
              style={{
                width: `${(d.value / peak) * 100}%`,
                background: 'var(--viz-series)',
                opacity: hovered === null || hovered === i ? 1 : 0.55,
              }}
            />
          </span>

          {/* 値は直接ラベル。テキストは系列色ではなく文字色を使う */}
          <span className="tabular-nums text-xs font-medium" style={{ color: 'var(--viz-text)' }}>
            {d.value}
            <span className="ml-0.5 text-[10px] font-normal" style={{ color: 'var(--viz-muted)' }}>
              {unit}
            </span>
          </span>
        </li>
      ))}
    </ul>
  );
}
