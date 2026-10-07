import { useState } from 'react';
import { useUser } from '@clerk/clerk-react';
import { ChartIcon, ListIcon, SettingsIcon } from './icons';
import { useNavigate } from 'react-router-dom';

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
    const { theme, toggleTheme } = useTheme()
    const { user } useUser()
    const navigate = useNavigate();

    const toggleLabel = collapsed ? '開く' : '閉じる';
    const displayNamee = user?.fullName ?? user?.username ?? 'ゲスト';
    const email = user?.primaryEmailAddress?.emailAddress ?? ''
    const initial = (displayNamee || 'H').charAt(0).toUpperCase();
}
