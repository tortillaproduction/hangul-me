import { useCallback, useEffect, useLayoutEffect, useRef, useState } from 'react';
import { ApiError, api } from '../../api/client';
import type { Candidate, Word } from '../../api/types';
import { CloseIcon, SearchPlusIcon } from '../../components/icons';
import { useToast } from '../../hooks/useToast';

interface LookupModalProps {
  open: boolean;
  onClose: () => void;
  /** 登録が完了したとき。呼び出し側で個別ページへ遷移します。 */
  onRegistered: (word: Word) => void;
}

type PanelSide = 'right' | 'left' | 'below';

/** 入力欄の右側に出す候補パネルの位置を、ブラウザ幅に応じて決めます。 */
function decidePanelSide(el: HTMLElement | null): PanelSide {
  if (!el) return 'below';
  if (window.innerWidth < 720) return 'below';
  const r = el.getBoundingClientRect();
  const spaceRight = window.innerWidth - r.right;
  const spaceLeft = r.left;
  if (spaceRight >= 320) return 'right';
  if (spaceLeft > spaceRight) return 'left';
  return 'below';
}

/**
 * 「単語を調べる」モーダル。
 * 入力はカタカナ・ひらがな・アルファベットのいずれでも受け付けます。
 */
export function LookupModal({ open, onClose, onRegistered }: LookupModalProps) {
  const [query, setQuery] = useState('');
  const [candidates, setCandidates] = useState<Candidate[]>([]);
  const [matchedVia, setMatchedVia] = useState('none');
  const [highlight, setHighlight] = useState(0);
  const [focused, setFocused] = useState(false);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [side, setSide] = useState<PanelSide>('right');

  const inputRef = useRef<HTMLInputElement>(null);
  const fieldRef = useRef<HTMLDivElement>(null);
  const abortRef = useRef<AbortController | null>(null);
  const { showToast } = useToast();

  // 開いたら入力欄にオートフォーカスし、状態を初期化する
  useEffect(() => {
    if (!open) return;
    setQuery('');
    setCandidates([]);
    setHighlight(0);

    const t = window.setTimeout(() => inputRef.current?.focus(), 30);
    return () => window.clearTimeout(t);
  }, [open]);

  // 入力中のリアルタイム検索（打鍵のたびに前回のリクエストは中断する）
  useEffect(() => {
    if (!open) return;
    const q = query.trim();
    if (!q) {
      setCandidates([]);
      setLoading(false);
      return;
    }

    const controller = new AbortController();
    abortRef.current?.abort();
    abortRef.current = controller;
    setLoading(true);

    const timer = window.setTimeout(async () => {
      try {
        const res = await api.lookup(q, controller.signal);
        setCandidates(res.candidates);
        setMatchedVia(res.matchedVia);
        setHighlight(0);
      } catch (err) {
        if (!(err instanceof DOMException && err.name === 'AbortError')) {
          setCandidates([]);
        }
      } finally {
        if (!controller.signal.aborted) setLoading(false);
      }
    }, 160);

    return () => {
      window.clearTimeout(timer);
      controller.abort();
    };
  }, [query, open]);

// パネル位置はウィンドウ幅の変化に追従させる。
useLayoutEffect(() => {
    if (!open) return;
    const update = () => setSide(decidePanelSide(fieldRef.current));
    update();
    window.addEventListener('resize', update);
    return () => window.removeEventListener('resize', update);
}, [open, candidates.length]);

const register = useCallback(
    async (candidate: Candidate) => {
        if (submitting) return;
        setSubmitting(true);
        try {
            const word = await api.registerWord({
                entryId: candidate.entryId,
                hangul: candidate.hangul,
                readingKana: candidate.readingKana,
                romanized: candidate.romanized,
                meaningJa: candidate.meaningJa,
                rawQuery: query,
                matchedVia,
            });
            showToast('success', '単語帳に登録しました', `${word.hangul} (${word.readingKana})`);
            onRegistered(word);
        } catch (err) {
            const message =
                err instanceof ApiError ? err.message : '登録に失敗しました。時間をおいて試してください';
            showToast('error', '登録できませんでした', message);
        } finally {
            setSubmitting(false);
        }
    },
    [matchedVia, onClose, onRegistered, query, showToast, submitting],
)

const onKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
        e.preventDefault();
        onClose();
        return
    }

    if (!candidates.length) return

    if (e.key === 'ArrowDown') {
        e.preventDefault()
        setHighlight((h) => (h + 1) % candidates.length)
    } else if (e.key === 'ArrowUp') {
        e.preventDefault()
        setHighlight((h) => (h - 1 + candidates.length) % candidates.length)
    } else if (e.key === 'Enter') {
        e.preventDefault()
        const picked = candidates[highlight]
        if (picked) void register(picked)
    }
}

if (!open) return null;

const panelClass =
    side === 'right'
        ? 'md:absolute md:left-[calc(100%+1rem)] md:top-0 md:w-[320px]'
        : side === 'left'
            ? 'md:absolute md:right-[calc(100%+1rem)] md:top-0 md:w-[320px]'
            : 'w-full';

    return (
    <div
      className="fixed inset-0 z-40 flex items-start justify-center bg-black/55 px-4 pt-[14vh] backdrop-blur-sm"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div
        className="relative w-full max-w-lg overflow-visible rounded-2xl border border-surface-border bg-surface-raised shadow-2xl"
        role="dialog"
        aria-modal="true"
        aria-label="単語を調べる"
      >
        <div className="flex items-center justify-between border-b border-surface-border px-5 py-3.5">
          <h2 className="flex items-center gap-2 text-sm font-semibold">
            <SearchPlusIcon className="text-brand-300" />
            単語を調べる
          </h2>
          <button
            type="button"
            onClick={onClose}
            className="rounded-md p-1.5 opacity-60 transition hover:bg-surface-hover hover:opacity-100"
            aria-label="閉じる"
          >
            <CloseIcon width={16} height={16} />
          </button>
        </div>

        <div className="relative p-5">
          <div
            ref={fieldRef}
            data-focused={focused}
            className="neon-focus-ring flex items-center gap-3 rounded-xl bg-surface-base px-4 py-3"
          >
            <SearchPlusIcon className="shrink-0 opacity-50" />
            <input
              ref={inputRef}
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onFocus={() => setFocused(true)}
              onBlur={() => setFocused(false)}
              onKeyDown={onKeyDown}
              placeholder="アンニョン / あんにょん / annyeong"
              className="w-full bg-transparent text-base outline-none placeholder:opacity-40"
              autoComplete="off"
              spellCheck={false}
              role="combobox"
              aria-expanded={candidates.length > 0}
              aria-controls="lookup-candidates"
            />
          </div>

          {(candidates.length > 0 || (query.trim() && !loading)) && (
            <div
              id="lookup-candidates"
              role="listbox"
              className={`mt-3 max-h-[46vh] overflow-y-auto rounded-xl border border-surface-border bg-surface-raised p-1.5 shadow-xl md:mt-0 ${panelClass}`}
            >
              {candidates.length === 0 ? (
                <p className="px-3 py-6 text-center text-sm opacity-50">
                  候補が見つかりませんでした
                </p>
              ) : (
                candidates.map((c, i) => (
                  <button
                    key={`${c.hangul}-${c.meaningJa}`}
                    type="button"
                    role="option"
                    aria-selected={i === highlight}
                    onMouseEnter={() => setHighlight(i)}
                    onClick={() => void register(c)}
                    disabled={submitting}
                    className={`flex w-full items-center justify-between gap-3 rounded-lg px-3 py-2.5 text-left transition ${
                      i === highlight ? 'bg-surface-hover' : 'hover:bg-surface-hover/60'
                    }`}
                  >
                    <span className="min-w-0">
                      <span className="block font-ko text-base font-semibold">{c.hangul}</span>
                      <span className="mt-0.5 block text-xs opacity-60">{c.readingKana}</span>
                    </span>
                    <span className="max-w-[55%] shrink-0 truncate text-right text-xs opacity-75">
                      {c.meaningJa}
                    </span>
                  </button>
                ))
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

