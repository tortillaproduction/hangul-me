import { useCallback, useEffect, useState } from 'react';

export type ThemeName = 'hangulme-dark' | 'hangulme-light';

const STORAGE_KEY = 'hangulme:theme';

function readStored(): ThemeName {
  try {
    const v = localStorage.getItem(STORAGE_KEY);
    if (v === 'hangulme-light' || v === 'hangulme-dark') return v;
  } catch {
    // プライベートモード等で読めない場合はダークにする
  }
  return 'hangulme-dark';
}

/** テーマの切り替え。規定はダークで、選択は localStorage に保持します。 */
export function useTheme() {
  const [theme, setTheme] = useState<ThemeName>(readStored);

  useEffect(() => {
    document.documentElement.setAttribute('data-theme', theme);
    try {
      localStorage.setItem(STORAGE_KEY, theme);
    } catch {
      // 保存できなくても表示は続行する
    }
  }, [theme]);

  const toggleTheme = useCallback(() => {
    setTheme((t) => (t === 'hangulme-dark' ? 'hangulme-light' : 'hangulme-dark'));
  }, []);

  return { theme, toggleTheme };
}
