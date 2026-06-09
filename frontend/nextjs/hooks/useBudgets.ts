'use client';

import { useQuery } from 'react-query';
import api from '@/lib/api';
import type { Budget } from '@/types';

export function useBudgets() {
  return useQuery<Budget[]>(
    'budgets',
    () => api.get('/budgets').then(r => r.data)
  );
}
