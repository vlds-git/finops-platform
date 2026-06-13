'use client';

import { useQuery, useMutation } from 'react-query';
import api from '@/lib/api';
import type { RecommendationData } from '@/types';

export function useRecommendations(startDate?: string, endDate?: string, provider?: string, accountId?: string, category = 'all') {
  return useQuery<RecommendationData>(
    ['recommendations', startDate, endDate, provider, accountId, category],
    () => api.get('/recommendations', { params: { start_date: startDate, end_date: endDate, provider, account_id: accountId, category } }).then(r => r.data),
    { enabled: !!startDate && !!endDate }
  );
}

export function useApplyRecommendation() {
  return useMutation(
    (id: string) => api.post(`/recommendations/${id}/apply`).then(r => r.data)
  );
}
