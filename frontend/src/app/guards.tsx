'use client';

import React, { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import Layout from '../components/Layout';
import { useAuth } from '../contexts/AuthContext';

export function PrivatePage({ children, adminOnly = false }: { children: React.ReactNode; adminOnly?: boolean }) {
  const { user, isAdmin, ready } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!ready) return;
    if (!user) router.replace('/login');
    else if (adminOnly && !isAdmin) router.replace('/');
  }, [adminOnly, isAdmin, ready, router, user]);

  if (!ready || !user || (adminOnly && !isAdmin)) {
    return (
      <div className="min-h-screen bg-gray-50 dark:bg-slate-900 flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-indigo-600" />
      </div>
    );
  }

  return <Layout>{children}</Layout>;
}

export function PublicPage({ children }: { children: React.ReactNode }) {
  const { user, ready } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (ready && user) router.replace('/');
  }, [ready, router, user]);

  if (!ready || user) return null;
  return <>{children}</>;
}
