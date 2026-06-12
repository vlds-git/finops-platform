'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import api from '@/lib/api';
import type { User } from '@/types';

function parseJwt(token: string): User | null {
  try {
    const base64Payload = token.split('.')[1];
    const payload = atob(base64Payload);
    const parsed = JSON.parse(payload);
    return {
      id: parsed.user_id || parsed.sub || '',
      email: parsed.email || '',
      name: parsed.name || parsed.email || '',
      roles: parsed.roles || [],
      active: true,
      created_at: new Date().toISOString(),
    };
  } catch {
    return null;
  }
}

export function useAuth() {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const router = useRouter();

  useEffect(() => {
    const token = localStorage.getItem('token');
    if (!token) {
      setLoading(false);
      router.push('/login');
      return;
    }

    const parsedUser = parseJwt(token);
    if (!parsedUser) {
      localStorage.removeItem('token');
      router.push('/login');
      setLoading(false);
      return;
    }

    setUser(parsedUser);
    setLoading(false);
  }, [router]);

  const login = async (email: string, password: string) => {
    const res = await api.post('/auth/login', { email, password });
    localStorage.setItem('token', res.data.token);
    setUser(res.data.user);
    router.push('/dashboard');
  };

  const logout = () => {
    localStorage.removeItem('token');
    setUser(null);
    router.push('/login');
  };

  return { user, loading, login, logout };
}
