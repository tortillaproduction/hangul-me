import { useState } from 'react';
import * as Clerk from '@clerk/elements/common';
import * as SignIn from '@clerk/elements/sign-in';
import * as SignUp from '@clerk/elements/sign-up';
import { NeonLogo } from '@/components/NeonLogo';

type Mode = 'signIn' | 'signUp';

/**
 * ログイン / 新規登録ページ。
 *
 * Clerk Elements（ヘッドレス）でUIを自前構築しているため、
 * 無料プランのままでも Clerk のブランディングは表示されません。
 */
export function AuthPage({ initialMode = 'signIn' }: { initialMode?: Mode }) {
  const [mode, setMode] = useState<Mode>(initialMode);

  return (
    <div className="flex min-h-full items-center justify-center bg-surface-base px-4 py-16">
      <div className="w-full max-w-sm">
        <div className="mb-10 flex justify-center">
          <NeonLogo size={50} flicker />
        </div>

        <div className="rounded-2xl border border-surface-border bg-surface-raised p-6 shadow-2xl">
          <div
            role="tablist"
            className="mb-6 grid grid-cols-2 gap-1 rounded-lg bg-surface-base p-1"
          >
            {(
              [
                ['signIn', 'ログイン'],
                ['signUp', '新規登録'],
              ] as const
            ).map(([value, label]) => (
              <button
                key={value}
                role="tab"
                type="button"
                aria-selected={mode === value}
                onClick={() => setMode(value)}
                className={`rounded-md px-3 py-2 text-sm transition ${
                  mode === value
                    ? 'bg-surface-hover font-semibold text-brand-300'
                    : 'opacity-60 hover:opacity-90'
                }`}
              >
                {label}
              </button>
            ))}
          </div>

          <h1 className="mb-5 text-lg font-bold">
            {mode === 'signIn' ? 'おかえりなさい' : 'ようこそ'}
          </h1>

          {mode === 'signIn' ? <SignInForm /> : <SignUpForm />}
        </div>

        <p className="mt-6 text-center text-[11px] opacity-40">
          <a href="/terms" className="hover:opacity-80">
            利用規約
          </a>
          <span className="mx-2">·</span>
          <a href="/privacy" className="hover:opacity-80">
            プライバシーポリシー
          </a>
        </p>
      </div>
    </div>
  );
}

const fieldClass =
  'w-full rounded-lg border border-surface-border bg-surface-base px-3 py-2.5 text-sm outline-none transition focus:border-brand-400';
const submitClass =
  'w-full rounded-lg bg-brand-400 px-4 py-2.5 text-sm font-semibold text-brand-900 transition hover:bg-brand-300 disabled:opacity-50';
const oauthClass =
  'flex w-full items-center justify-center gap-2 rounded-lg border border-surface-border px-4 py-2.5 text-sm transition hover:bg-surface-hover';

function Divider() {
  return (
    <div className="my-5 flex items-center gap-3">
      <span className="h-px flex-1 bg-surface-border" />
      <span className="text-[11px] opacity-40">または</span>
      <span className="h-px flex-1 bg-surface-border" />
    </div>
  );
}

function OAuthButtons() {
  return (
    <div className="flex flex-col gap-2">
      <Clerk.Connection name="google" className={oauthClass}>
        <GoogleIcon />
        Googleで続ける
      </Clerk.Connection>
      <Clerk.Connection name="google" className={oauthClass}>
        <AppleIcon />
        Appleで続ける
      </Clerk.Connection>
    </div>
  );
}

function SignInForm() {
  return (
    <SignIn.Root>
      <SignIn.Step name="start" className="flex flex-col gap-3">
        <Clerk.Field name="identifier">
          <Clerk.Label className="mb-1 block text-xs opacity-60">メールアドレス</Clerk.Label>
          <Clerk.Input className={fieldClass} placeholder="you@example.com" />
          <Clerk.FieldError className="mt-1 block text-xs text-error" />
        </Clerk.Field>

        <Clerk.Field name="password">
          <Clerk.Label className="mb-1 block text-xs opacity-60">パスワード</Clerk.Label>
          <Clerk.Input type="password" className={fieldClass} />
          <Clerk.FieldError className="mt-1 block text-xs text-error" />
        </Clerk.Field>

        <SignIn.Action submit className={`${submitClass} mt-2`}>
          ログイン
        </SignIn.Action>

        <Divider />
        <OAuthButtons />
      </SignIn.Step>

      <SignIn.Step name="verifications" className="flex flex-col gap-3">
        <SignIn.Strategy name="email_code">
          <p className="text-sm opacity-70">メールに届いた確認コードを入力してください</p>
          <Clerk.Field name="code">
            <Clerk.Input className={fieldClass} placeholder="123456" />
            <Clerk.FieldError className="mt-1 block text-xs text-error" />
          </Clerk.Field>
          <SignIn.Action submit className={submitClass}>
            確認する
          </SignIn.Action>
        </SignIn.Strategy>
      </SignIn.Step>
    </SignIn.Root>
  );
}

function SignUpForm() {
  return (
    <SignUp.Root>
      <SignUp.Step name="start" className="flex flex-col gap-3">
        <Clerk.Field name="emailAddress">
          <Clerk.Label className="mb-1 block text-xs opacity-60">メールアドレス</Clerk.Label>
          <Clerk.Input className={fieldClass} placeholder="you@example.com" />
          <Clerk.FieldError className="mt-1 block text-xs text-error" />
        </Clerk.Field>

        <Clerk.Field name="password">
          <Clerk.Label className="mb-1 block text-xs opacity-60">パスワード</Clerk.Label>
          <Clerk.Input type="password" className={fieldClass} />
          <Clerk.FieldError className="mt-1 block text-xs text-error" />
        </Clerk.Field>

        <SignUp.Action submit className={`${submitClass} mt-2`}>
          アカウントを作成
        </SignUp.Action>

        <Divider />
        <OAuthButtons />
      </SignUp.Step>

      <SignUp.Step name="verifications" className="flex flex-col gap-3">
        <SignUp.Strategy name="email_code">
          <p className="text-sm opacity-70">メールに届いた確認コードを入力してください。</p>
          <Clerk.Field name="code">
            <Clerk.Input className={fieldClass} placeholder="123456" />
            <Clerk.FieldError className="mt-1 block text-xs text-error" />
          </Clerk.Field>
          <SignUp.Action submit className={submitClass}>
            確認する
          </SignUp.Action>
        </SignUp.Strategy>
      </SignUp.Step>
    </SignUp.Root>
  );
}

function GoogleIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" aria-hidden="true">
      <path
        fill="#4285F4"
        d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 0 1-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1Z"
      />
      <path
        fill="#34A853"
        d="M12 23c2.97 0 5.46-.98 7.28-2.65l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84A11 11 0 0 0 12 23Z"
      />
      <path
        fill="#FBBC05"
        d="M5.84 14.11a6.6 6.6 0 0 1 0-4.22V7.05H2.18a11 11 0 0 0 0 9.9l3.66-2.84Z"
      />
      <path
        fill="#EA4335"
        d="M12 4.75c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 1.46 14.97.5 12 .5A11 11 0 0 0 2.18 7.05l3.66 2.84c.87-2.6 3.3-4.14 6.16-4.14Z"
      />
    </svg>
  );
}

function AppleIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true">
      <path d="M16.36 12.78c-.02-2.3 1.88-3.4 1.96-3.45-1.07-1.56-2.73-1.78-3.32-1.8-1.41-.14-2.76.83-3.47.83-.72 0-1.82-.81-3-.79-1.54.02-2.96.9-3.75 2.27-1.6 2.78-.41 6.89 1.15 9.14.76 1.1 1.67 2.34 2.86 2.3 1.15-.05 1.58-.74 2.97-.74 1.38 0 1.78.74 2.99.72 1.24-.02 2.02-1.12 2.78-2.23.87-1.28 1.23-2.52 1.25-2.58-.03-.01-2.4-.92-2.42-3.67ZM14.1 5.7c.63-.77 1.06-1.83.94-2.9-.91.04-2.01.61-2.66 1.37-.58.68-1.09 1.76-.95 2.8 1.01.08 2.04-.51 2.67-1.27Z" />
    </svg>
  );
}
