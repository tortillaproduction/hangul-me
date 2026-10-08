import { useEffect, useState } from 'react';
import type { CurrentUser } from '../api/types';
import { useTheme } from '../hooks/useTheme';

export function SettingsPage() {
    const [me, setMe] = useState<CurrentUser | null>(null);
    const { theme, toggleTheme } = useTheme();

    useEffect(() => {
        // ここにユーザー情報を取得するロジックを記述
    }, [])

    return ()
}

function Row({ label, value }: { label: string; value: string }) {
    return (
        <div className="flex items-center justify-between gap-4">
            <dt className="opacity-50">{label}</dt>
            <dd className="truncate">{value}</dd>
        </div>
    )
}
