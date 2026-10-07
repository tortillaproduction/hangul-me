import { createContext } from 'react';

export type ToastKind = 'success' | 'error';

export interface ToastContextValue {
  showToast: (kind: ToastKind, title: string, detail?: string) => void;
}

/**
 * トースト通知のコンテキスト。
 *
 * Provider（コンポーネント）とフックを別ファイルに分けているのは、
 * 1ファイルからコンポーネント以外もexportすると Fast Refresh が効かなくなるためです。
 */
export const ToastContext = createContext<ToastContextValue | null>(null);
