import { useCallback, useEffect, useState } from 'react';
import { Navigate, Route, Routes, useNavigate } from 'react-router-dom';
import { SignedIn, SignedOut, useAuth } from '@clerk/clerk-react';
import { setTokenGetter } from './api/client';
import type { Word } from './api/types';
import { AppLayout } from './components/AppLayout';
import { ToastProvider } from './components/ToastProvider';
import { LookupModal } from './features/lookup/LookupModal';
import { AnalyticsPage } from './pages/AnalyticsPage';
import { AuthPage } from './pages/AuthPage';
import { SettingsPage } from './pages/SettingsPage';
import { WordDetailPage } from './pages/WordDetailPage';
import { WordListPage } from './pages/WordListPage';
import { useTheme } from './hooks/useTheme';

export default function App() {
  useTheme(); // 起動時に data-theme を確定させる（既定はダーク）

  return (
    <ToastProvider>
      <SignedIn>
        <AuthenticatedApp />
      </SignedIn>
      <SignedOut>
        <Routes>
          <Route path="/sign-up" element={<AuthPage initialMode="signUp" />} />
          <Route path="*" element={<AuthPage initialMode="signIn" />} />
        </Routes>
      </SignedOut>
    </ToastProvider>
  );
}

function AuthenticatedApp() {
  const { getToken } = useAuth();
  const navigate = useNavigate();
  const [lookupOpen, setLookupOpen] = useState(false);
  const [refreshKey, setRefreshKey] = useState(0);

  // APIクライアントに Clerk のセッショントークンを供給する
  useEffect(() => {
    setTokenGetter(() => getToken());
  }, [getToken]);

  // 登録が完了したら一覧を更新し、その単語の個別ページへ遷移する
  const handleRegistered = useCallback(
    (word: Word) => {
      setRefreshKey((k) => k + 1);
      navigate(`/words/${word.id}`);
    },
    [navigate],
  );

  return (
    <>
      <AppLayout onOpenLookup={() => setLookupOpen(true)}>
        <Routes>
          <Route path="/" element={<Navigate to="/words" replace />} />
          <Route
            path="/words"
            element={
              <WordListPage onOpenLookup={() => setLookupOpen(true)} refreshKey={refreshKey} />
            }
          />
          <Route path="/words/:id" element={<WordDetailPage />} />
          <Route path="/analytics" element={<AnalyticsPage />} />
          <Route path="/settings" element={<SettingsPage />} />
          <Route path="*" element={<Navigate to="/words" replace />} />
        </Routes>
      </AppLayout>

      <LookupModal
        open={lookupOpen}
        onClose={() => setLookupOpen(false)}
        onRegistered={handleRegistered}
      />
    </>
  );
}
