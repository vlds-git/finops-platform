'use client';

import { useQuery } from 'react-query';
import api from '@/lib/api';
import type { AnomalyData } from '@/types';

export function useAnomalies(provider: string, accountId: string, method = 'ensemble') {
  return useQuery<AnomalyData>(
    ['anomalies', provider, accountId, method],
    () => api.post('/anomalies', { provider, account_id: accountId, method }).then(r => r.data)
  );
}
