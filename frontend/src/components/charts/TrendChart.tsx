import { useMemo, useRef, useState } from 'react';
import type { DailyCount } from '../../api/types';

interface TrendChartProps {
  data: DailyCount[];
  /** 表示する日数（データがない日は0として埋める） */
  days?: number;
}

const W = 720;
const H = 160;
const PAD = { top: 12, right: 12, bottom: 22, left: 28 };

function buildSeries(data: DailyCount[], days: number) {
  const byDate = new Map(data.map((d) => [d.date, d.count]));
  const out: DailyCount[] = [];
  const today = new Date();
  for (let i = days - 1; i >= 0; i--) {
    const d = new Date(today);
    d.setDate(today.getDate() - i);
    const key = d.toISOString().slice(0, 10);
    out.push({ date: key, count: byDate.get(key) ?? 0 });
  }
  return out;
}

/**
 * 登録数の推移（単一系列）。
 * 系列が1つなので凡例は置かず、見出しが系列名を兼ねます。
 * ホバーで縦のクロスヘアと値のツールチップを出します。
 */
export function TrendChart({ data, days = 30 }: TrendChartProps) {
  const svgRef = useRef<SVGSVGElement>(null);
  const [hover, setHover] = useState<number | null>(null);

  const series = useMemo(() => buildSeries(data, days), [data, days]);
  const peak = Math.max(1, ...series.map((d) => d.count));

  const innerW = W - PAD.left - PAD.right;
  const innerH = H - PAD.top - PAD.bottom;
  const x = (i: number) => PAD.left + (innerW * i) / Math.max(1, series.length - 1);
  const y = (v: number) => PAD.top + innerH - (innerH * v) / peak;

  const path = series.map((d, i) => `${i === 0 ? 'M' : 'L'}${x(i)},${y(d.count)}`).join(' ');
  const area = `${path} L${x(series.length - 1)},${PAD.top + innerH} L${x(0)},${PAD.top + innerH} Z`;

  const onMove = (e: React.MouseEvent<SVGSVGElement>) => {
    const rect = svgRef.current?.getBoundingClientRect();
    if (!rect) return;
    const ratio = ((e.clientX - rect.left) / rect.width) * W;
    const i = Math.round(((ratio - PAD.left) / innerW) * (series.length - 1));
    setHover(Math.min(series.length - 1, Math.max(0, i)));
  };

  const total = series.reduce((s, d) => s + d.count, 0);
  const active = hover !== null ? series[hover] : null;

  return (
    <div className="relative">
      <svg
        ref={svgRef}
        viewBox={`0 0 ${W} ${H}`}
        className="w-full"
        style={{ height: H }}
        onMouseMove={onMove}
        onMouseLeave={() => setHover(null)}
        role="img"
        aria-label={`直近${days}日の登録数の推移。合計${total}語`}
      >
        {/* 目盛りは控えめに。0と最大値だけ */}
        {[0, peak].map((v) => (
          <g key={v}>
            <line
              x1={PAD.left}
              x2={W - PAD.right}
              y1={y(v)}
              y2={y(v)}
              stroke="var(--viz-grid)"
              strokeWidth={1}
            />
            <text
              x={PAD.left - 6}
              y={y(v) + 3.5}
              textAnchor="end"
              fontSize={9}
              fill="var(--viz-muted)"
            >
              {v}
            </text>
          </g>
        ))}

        <path d={area} fill="var(--viz-series)" opacity={0.12} />
        <path
          d={path}
          fill="none"
          stroke="var(--viz-series)"
          strokeWidth={2}
          strokeLinecap="round"
          strokeLinejoin="round"
        />

        {active && hover !== null && (
          <g>
            <line
              x1={x(hover)}
              x2={x(hover)}
              y1={PAD.top}
              y2={PAD.top + innerH}
              stroke="var(--viz-muted)"
              strokeWidth={1}
              strokeDasharray="3 3"
            />
            {/* 重なるマークには面色のリングを1本入れて輪郭を保つ */}
            <circle
              cx={x(hover)}
              cy={y(active.count)}
              r={4.5}
              fill="var(--viz-series)"
              stroke="var(--viz-surface)"
              strokeWidth={2}
            />
          </g>
        )}

        {/* 端の日付だけ表示して軸を混ませない */}
        <text x={PAD.left} y={H - 6} fontSize={9} fill="var(--viz-muted)">
          {series[0]?.date.slice(5)}
        </text>
        <text x={W - PAD.right} y={H - 6} fontSize={9} textAnchor="end" fill="var(--viz-muted)">
          {series[series.length - 1]?.date.slice(5)}
        </text>
      </svg>

      {active && hover !== null && (
        <div
          className="pointer-events-none absolute -translate-x-1/2 -translate-y-full rounded-md border border-surface-border bg-surface-raised px-2 py-1 text-[11px] shadow-lg"
          style={{ left: `${(x(hover) / W) * 100}%`, top: `${(y(active.count) / H) * 100}%` }}
        >
          <span className="opacity-60">{active.date}</span>
          <span className="ml-2 font-semibold tabular-nums">{active.count}語</span>
        </div>
      )}
    </div>
  );
}
