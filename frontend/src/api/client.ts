import {
  AnalyticsOverview,
  CurrentUser,
  LookupResult,
  RankedWord,
  RegisterWordInput,
  Word,
} from './types';

const BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '';

/** APIが返すエラー。code で UI側の出し分けをします。 */
export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly code: string,
    message: string,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

/** Clerk のセッショントークンを取り出す関数。 App 起動時に差し込みます。 */
type TokenGetter = () => Promise<string | null>;

let getToken: TokenGetter = async () => null;

export function setTokenGetter(fn: TokenGetter) {
  getToken = fn;
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const token = await getToken();
  const headers = new Headers(init.headers);
  headers.set('Content-Type', 'application/json');
  if (token) headers.set('Authorization', `Bearer ${token}`);

  const res = await fetch(`${BASE_URL}${path}`, { ...init, headers });

  if (res.status === 204) return undefined as T;

  const text = await res.text();
  const body = text ? JSON.parse(text) : [];

  if (!res.ok) {
    const err = body?.error ?? {};
    throw new ApiError(res.status, err.code ?? 'unknown', err.message ?? '通信に失敗しました');
  }
  return body as T;
}

export const api = {
  me: () => request<CurrentUser>('/api/me'),

  /** 入力中のリアルタイム検索 */
  lookup: (query: string, signal?: AbortSignal) =>
    request<LookupResult>(`/api/lookup?q=${encodeURIComponent(query)}`, { signal }),

  listWords: (params: { encounteredTitle?: string } = {}) => {
    const qs = new URLSearchParams();
    if (params.encounteredTitle) qs.set('encounteredTitle', params.encounteredTitle);
    const suffix = qs.toString() ? `?${qs}` : '';
    return request<{ words: Word[] }>(`/api/words${suffix}`).then((r) => r.words ?? []);
  },

  getWord: (id: string) => request<Word>(`/api/words/${id}`),

  registerWord: (input: RegisterWordInput) =>
    request<Word>(`/api/words`, { method: 'POST', body: JSON.stringify(input) }),

  updateWord: (
    id: string,
    patch: { memo?: string; encounteredTitle?: string; masteryLevel?: number },
  ) => request<Word>(`/api/words/${id}`, { method: 'PATCH', body: JSON.stringify(patch) }),

  deleteWord: (id: string) => request<void>(`/api/words/${id}`, { method: 'DELETE' }),

  analytics: () => request<AnalyticsOverview>('/api/analytics'),

  trending: (limit = 10) =>
    request<{ words: RankedWord[] }>(`/api/analytics/trending?limit=${limit}`).then(
      (r) => r.words ?? [],
    ),
};
