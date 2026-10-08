import { useState } from 'react';

type Mode = 'signIn' | 'signUp';

/**
 * ログイン / 新規登録ページ。
 *
 * Clerk Elements（ヘッドレス）でUIを自前構築しているため、
 * 無料プランのままでも Clerk のブランディングは表示されません。
 */
export function AuthPage({ initialMode = 'signIn' }: { initialMode?: Mode }) {
    const [mode, setMode] = useState<Mode>(initialMode);

    return ()
}

const fieldClass =
    'w-full rounded-lg border border-surface-border bg-surface-base px-3 py-2.5 text-sm outline-none transition focus:border-brand-400'
const submitClass =
    'w-full rounded-lg bg-brand-400 px-4 py-2.5 text-sm font-semibold text-brand-900 transition hover:bg-brand-300 disabled:opacity-50'
const oauthClass =
    'flex w-full items-center justify-center gap-2 rounded-lg border border-surface-border px-4 py-2.5 text-sm transition hover:bg-surface-hover'

function Divider() {
    return (
        <div className="my-5 flex items-center gap-3">
            <span className="h-px flex-1 bg-surface-border" />
            <span className="text-[11px] opacity-40">または</span>
            <span className="h-px flex-1 bg-surface-border" />
        </div>
    )
}

function OAuthButtons() {
    return (
        <div>

        </div>
    )
}

function SignInForm() {
    return ()
}

function SignUpForm() {
    return ()
}

function GoogleIcon() {
    return ()
}

function AppleIcon() {
    return ()
}
