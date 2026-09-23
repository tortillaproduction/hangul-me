/**
 * Next.js を採用していないため、@clerk/elements が任意peer依存として参照する
 * `next/compat/router` と `next/navigation` のスタブです。
 *
 * Clerk Elements はルーターの種類を実行時に判定し、Next でない環境では
 * これらを呼びません。バンドラがモジュール解決に失敗しないようにするための
 * 空実装で、呼ばれた場合も安全に null を返します。
 */

export function useRouter(): null {
  return null;
}

export function useParams(): Record<string, string> {
  return {};
}

export function usePathname(): string {
  return typeof window === 'undefined' ? '/' : window.location.pathname;
}

export function useSearchParams(): URLSearchParams {
  return new URLSearchParams(typeof window === 'undefined' ? '' : window.location.search);
}

export function redirect(): void {
  /* Next 専用APIのため何もしない */
}

export default { useRouter, useParams, usePathname, useSearchParams, redirect };
