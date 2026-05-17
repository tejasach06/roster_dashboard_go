'use client';

import React, { createContext, useContext, useEffect, useState } from 'react';

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
    try {
      const stored = localStorage.getItem('user');
      setUser(stored ? JSON.parse(stored) : null);
    } catch {
      setUser(null);
    } finally {
      setReady(true);
    }
  }, []);

  const login = (token: string, user: User) => {
    localStorage.setItem('token', token);
    localStorage.setItem('user', JSON.stringify(user));
    setUser(user);
  };

  const logout = () => {
    localStorage.removeItem('token');
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
