import { ReactNode, useCallback, useMemo, useRef, useState } from 'react';
import { ToastContext, ToastKind } from './toast-context';
import { AlertIcon, CheckCircleIcon } from './icons';

interface Toast {
  id: number;
  kind: ToastKind;
  title: string;
  detail?: string;
}

/**画面右下に一時的な通知を出します（単語の登録完了など）。 */
export function ToastProvider({ children }: { children: ReactNode }) {
  const [toasts, setToasts] = useState<Toast[]>([]);
  const seq = useRef(0);

  const showToast = useCallback((kind: ToastKind, title: string, detail?: string) => {
    const id = ++seq.current;
    setToasts((prev) => [...prev, { id, kind, title, detail }]);
    window.setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, 4000);
  }, []);

  const value = useMemo(() => ({ showToast }), [showToast]);

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div
        className="pointer-events-none fixed bottom-5 right-5 z-50 flex w-[min(360px,calc(100vw-2.5rem))] flex-col gap-2"
        role="status"
        aria-live="polite"
      >
        {toasts.map((t) => (
          <div
            key={t.id}
            className="pointer-events-auto flex animate-toast-in items-start gap-3 rounded-xl border border-surface-border bg-surface-raised/95 px-4 py-3 shadow-xl backdrop-blur"
          >
            <span className={t.kind === 'success' ? 'text-brand-300' : 'text-error'}>
              {t.kind === 'success' ? <CheckCircleIcon /> : <AlertIcon />}
            </span>
            <div className="min-w-0">
              <p className="text-sm font-semibold">{t.title}</p>
              {t.detail && <p className="mt-0.5 truncate text-xs opacity-70">{t.detail}</p>}
            </div>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}
