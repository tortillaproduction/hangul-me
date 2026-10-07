import { useContext } from 'react';
import { ToastContext } from '@/components/toast-context';

/** 画面右下のトースト通知を出します。ToastProvider の内側でのみ使えます。 */
export function useToast() {
  const ctx = useContext(ToastContext);
  if (!ctx) throw new Error('useToast は ToastProvider の内側で使ってください');
  return ctx;
}
