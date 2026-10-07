import { useCallback, useRef, useState } from 'react';
import type { ReactElement } from 'react';
import { createPortal } from 'react-dom';

interface TooltipProps {
  label: string;
  /** false の時はツールチップを出しません（サイドバーが開いている間など） */
  enabled?: boolean;
  children: ReactElement;
}

/**
 * サイドバーが閉じているときのアイコン用ツールチップ。
 *
 * サイドバー自体が overflow: hidden のため、内部に絶対配置すると切れてしまいます。
 * そこで body 直下にポータルで描画し、位置は対象要素の矩形から計算します。
 */
export function Tooltip({ label, enabled = true, children }: TooltipProps) {
  const anchorRef = useRef<HTMLElement | null>(null);
  const [pos, setPos] = useState<{ top: number; left: number } | null>(null);

  const show = useCallback(() => {
    if (!enabled || !anchorRef.current) return;
    const r = anchorRef.current.getBoundingClientRect();
    setPos({ top: r.top + r.height / 2, left: r.left + 10 });
  }, [enabled]);

  const hide = useCallback(() => setPos(null), []);

  return (
    <>
      <span
        ref={(el) => {
          anchorRef.current = el;
        }}
        className="contents"
        onMouseEnter={show}
        onMouseLeave={hide}
        onFocus={show}
        onBlur={hide}
      >
        {children}
      </span>
      {pos &&
        createPortal(
          <span
            role="tooltip"
            className="pointer-events-none fixed z--[60] -translate-y-1/2 whitespace-nowrap rounded-md border border-surface-border bg-surface-raised px-2.5 py-1.5 text-xs shadow-lg"
            style={{ top: pos.top, left: pos.left }}
          >
            {label}
          </span>,
          document.body,
        )}
    </>
  );
}
