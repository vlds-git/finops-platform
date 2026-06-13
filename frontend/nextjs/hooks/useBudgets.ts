'use client';

import { useQuery, useMutation, useQueryClient } from 'react-query';
import api from '@/lib/api';
import type { Budget } from '@/types';

export function useBudgets() {
  return useQuery<Budget[]>(
    'budgets',
    () => api.get('/budgets').then(r => r.data)
  );
}

export function useCreateBudget() {
  const qc = useQueryClient();
  return useMutation(
    (b: Omit<Budget, 'id' | 'spent' | 'remaining' | 'created_at'>) =>
      api.post('/budgets', b).then(r => r.data),
    { onSuccess: () => qc.invalidateQueries('budgets') }
  );
}

export function useUpdateBudget() {
  const qc = useQueryClient();
  return useMutation(
    ({ id, ...b }: Partial<Budget> & { id: string }) =>
      api.put(`/budgets/${id}`, b).then(r => r.data),
    { onSuccess: () => qc.invalidateQueries('budgets') }
  );
}

export function useDeleteBudget() {
  const qc = useQueryClient();
  return useMutation(
    (id: string) => api.delete(`/budgets/${id}`).then(r => r.data),
    { onSuccess: () => qc.invalidateQueries('budgets') }
  );
}

export interface CloudAccount {
  id: string;
  provider: string;
  account_id: string;
  account_name: string;
  active: boolean;
}

export function useCloudAccounts() {
  return useQuery<CloudAccount[]>(
    'cloud-accounts',
    () => api.get('/accounts').then(r => r.data),
    { staleTime: 5 * 60 * 1000 }
  );
}
