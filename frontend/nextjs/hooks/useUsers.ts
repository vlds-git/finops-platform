'use client';

import { useQuery, useMutation, useQueryClient } from 'react-query';
import api from '@/lib/api';
import type { User } from '@/types';

export function useUsers() {
  return useQuery<User[]>(
    'users',
    () => api.get('/admin/users').then(r => r.data)
  );
}

export interface CreateUserInput {
  email: string;
  name: string;
  password: string;
  roles: string[];
}

export function useCreateUser() {
  const qc = useQueryClient();
  return useMutation(
    (u: CreateUserInput) => api.post('/admin/users', u).then(r => r.data),
    { onSuccess: () => qc.invalidateQueries('users') }
  );
}

export function useUpdateUser() {
  const qc = useQueryClient();
  return useMutation(
    ({ id, ...u }: Partial<User> & { id: string }) =>
      api.put(`/admin/users/${id}`, u).then(r => r.data),
    { onSuccess: () => qc.invalidateQueries('users') }
  );
}

export function useDeleteUser() {
  const qc = useQueryClient();
  return useMutation(
    (id: string) => api.delete(`/admin/users/${id}`).then(r => r.data),
    { onSuccess: () => qc.invalidateQueries('users') }
  );
}
