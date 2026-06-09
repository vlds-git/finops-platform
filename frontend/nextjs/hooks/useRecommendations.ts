'use client';

import { useQuery, useMutation } from 'react-query';
import api from '@/lib/api';
import type { RecommendationData } from '@/types';

export function useRecommendations(provider: string, accountId: string, category = 'all') {
  return useQuery<RecommendationData>(
    ['recommendations', provider, accountId, category],
    () => api.post('/recommendations', { provider, account_id: accountId, category }).then(r => r.data)
  );
}

export function useApplyRecommendation() {
  return useMutation(
    (id: string) => api.post(`/recommendations/${id}/apply`).then(r => r.data)
  );
}
