'use client';

import React, { createContext, useContext, useEffect, useState } from 'react';
import api from '../api/client';

export interface User {
  id: number;
  name: string;
  username: string;
  role: 'admin' | 'member';
  team_id: number | null;
}

interface AuthContextType {
  user: User | null;
  login: (token: string, user: User) => void;
  logout: () => void;
  isAdmin: boolean;
  ready: boolean;
}

const AuthContext = createContext<AuthContextType>({ user: null, login: () => {}, logout: () => {}, isAdmin: false, ready: false });

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [ready, setReady] = useState(false);
  const [user, setUser] = useState<User | null>(null);

  useEffect(() => {
    let mounted = true;
    api.get<User>('/auth/me')
      .then(({ data }) => {
        if (!mounted) return;
        setUser(data);
        localStorage.setItem('user', JSON.stringify(data));
      })
      .catch(() => {
        if (!mounted) return;
        localStorage.removeItem('user');
        setUser(null);
      })
      .finally(() => {
        if (mounted) setReady(true);
      });
    return () => { mounted = false; };
  }, []);

  const login = (_token: string, user: User) => {
    localStorage.setItem('user', JSON.stringify(user));
    setUser(user);
  };

  const logout = () => {
    void api.post('/auth/logout').catch(() => {});
    localStorage.removeItem('user');
    setUser(null);
  };

  return (
    <AuthContext.Provider value={{ user, login, logout, isAdmin: user?.role === 'admin', ready }}>
      {children}
    </AuthContext.Provider>
  );
}

export const useAuth = () => useContext(AuthContext);
