import { useState } from 'react';
import { SignOutButton, useUser } from '@clerk/clerk-react';
import { ChartIcon, ListIcon, PanelIcon, SearchPlusIcon, SettingsIcon, ThemeIcon } from './icons';
import { NavLink, useNavigate } from 'react-router-dom';
import { useTheme } from '@/hooks/useTheme';
import { Tooltip } from './Tooltip';
import { NeonLogo } from './NeonLogo';

interface AppLayoutProps {
  children: React.ReactNode;
  /** 「単語を調べる」モーダルを開く */
  onOpenLookup: () => void;
}

const NAV_ITEMS = [
  { to: '/words', label: '登録単語一覧', icon: ListIcon },
  { to: '/analytics', label: '学習の記録', icon: ChartIcon },
  { to: '/settings', label: 'アプリ設定', icon: SettingsIcon },
];

export function AppLayout({ children, onOpenLookup }: AppLayoutProps) {
  const [collapsed, setCollapsed] = useState(false);
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const { theme, toggleTheme } = useTheme();
  const { user } = useUser();
  const navigate = useNavigate();

  const toggleLabel = collapsed ? '開く' : '閉じる';
  const displayName = user?.fullName ?? user?.username ?? 'ゲスト';
  const email = user?.primaryEmailAddress?.emailAddress ?? '';
  const initial = (displayName || 'H').charAt(0).toUpperCase();

  return (
    <div className="flex h-full flex-col bg-surface-base text-base-content">
      {/* ナビゲーションバー */}
      <header className="flex h-topbar shrink-0 items-center gap-3 border-b border-surface-border bg-surface-raised px-3">
        <Tooltip label={toggleLabel}>
          <button
            type="button"
            onClick={() => setCollapsed((c) => !c)}
            className="rounded-md p-2 opacity-70 transition hover:bg-surface-hover hover:opacity-100"
            aria-label={toggleLabel}
          >
            <PanelIcon />
          </button>
        </Tooltip>

        {/* ロゴクリックでメインエリアの既定表示（登録単語一覧）へ戻る */}
        <button
          type="button"
          onClick={() => navigate('/words')}
          className="rounded-md px-2 py-1 transition hover:opacity-80"
          aria-label="Hangul-me トップへ"
        >
          <NeonLogo size={18} />
        </button>

        <div className="ml-auto relative">
          <button
            type="button"
            onClick={() => setUserMenuOpen((o) => !o)}
            className="flex items-center gap-2 rounded-full p-1 transition hover:bg-surface-hover"
            aria-haspopup="menu"
            aria-expanded={userMenuOpen}
          >
            <span className="grid h-8 w-8 place-items-center rounded-full bg-brand-700 text-sm font-semibold text-white">
              {initial}
            </span>
          </button>

          {userMenuOpen && (
            <>
              <div
                className="fixed inset-0 z-10"
                onClick={() => setUserMenuOpen(false)}
                aria-hidden="true"
              />
              <div
                className="absolute right-0 z-20 mt-2 w-64 overflow-hidden rounded-xl border border-surface-border bg-surface-raised shadow-2xl"
                role="menu"
              >
                {/* ユーザー情報: 丸アイコン + 名前(太字) + メールを横並び */}
                <div className="flex items-center gap-3 border-b border-surface-border px-4 py-3">
                  <span className="grid h-9 w-9 shrink-0 place-items-center rounded-full bg-brand-700 text-sm font-semibold text-white">
                    {initial}
                  </span>
                  <span className="min-w-0">
                    <span className="block truncate text-sm font-bold">{displayName}</span>
                    <span className="block truncate text-xs opacity-60">{email}</span>
                  </span>
                </div>
                <SignOutButton>
                  <button
                    type="button"
                    className="w-full px-4 py-2.5 text-left text-sm transition hover:bg-surface-hover"
                    role="menuitem"
                  >
                    ログアウト
                  </button>
                </SignOutButton>
              </div>
            </>
          )}
        </div>
      </header>

      <div className="flex min-h-0 flex-1">
        {/* 左サイドバー */}
        <aside
          className="flex shrink-0 flex-col overflow-hidden border-r border-surface-border bg-surface-raised transition-[width] duration-200"
          style={{ width: collapsed ? 60 : 236 }}
        >
          <div className="p-2">
            <Tooltip label="単語を調べる" enabled={collapsed}>
              <button
                type="button"
                onClick={onOpenLookup}
                className="flex w-full items-center gap-2.5 rounded-lg bg-brand-400 px-3 py-2 text-sm font-semibold text-brand-900 transition hover:bg-brand-300"
              >
                <SearchPlusIcon className="shrink-0" />
                {!collapsed && <span className="truncate">単語を調べる</span>}
              </button>
            </Tooltip>
          </div>

          <nav className="min-h-0 flex-1 overflow-y-auto px-2">
            {NAV_ITEMS.map(({ to, label, icon: Icon }) => (
              <Tooltip key={to} label={label} enabled={collapsed}>
                <NavLink
                  to={to}
                  className={({ isActive }) =>
                    `mb-0.5 flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm transition ${
                      isActive
                        ? 'bg-surface-hover font-medium text-brand-300'
                        : 'opacity-75 hover:bg-surface-hover hover:opacity-100'
                    }`
                  }
                >
                  <Icon className="shrink-0" />
                  {!collapsed && <span className="truncate">{label}</span>}
                </NavLink>
              </Tooltip>
            ))}
          </nav>

          <div className="px-2 pb-3">
            <Tooltip label="テーマ" enabled={collapsed}>
              <button
                type="button"
                onClick={toggleTheme}
                className="flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-sm opacity-75 transition hover:bg-surface-hover hover:opacity-100"
              >
                <ThemeIcon className="shrink-0" />
                {!collapsed && (
                  <span className="truncate">
                    テーマ
                    <span className="ml-1.5 opacity-50">
                      {theme === 'hangulme-dark' ? 'ダーク' : 'ライト'}
                    </span>
                  </span>
                )}
              </button>
            </Tooltip>

            {/* テーマの下に divider、その下は半分の大きさで表示 */}
            <hr className="my-2 border-surface-border" />

            {!collapsed && (
              <div className="flex flex-col">
                <a
                  href="/privacy"
                  className="rounded px-3 py-1 text-[11px] opacity-50 transition hover:opacity-80"
                >
                  プライバシーポリシー
                </a>
                <a
                  href="/terms"
                  className="rounded px-3 py-1 text-[11px] opacity-50 transition hover:opacity-80"
                >
                  利用規約
                </a>
              </div>
            )}
          </div>
        </aside>

        <main className="min-w-0 flex-1 overflow-y-auto">{children}</main>
      </div>
    </div>
  );
}
