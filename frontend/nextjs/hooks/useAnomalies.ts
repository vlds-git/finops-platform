'use client';

import { useQuery } from 'react-query';
import api from '@/lib/api';
import type { AnomalyData } from '@/types';

export function useAnomalies(startDate?: string, endDate?: string, provider?: string, accountId?: string) {
  return useQuery<AnomalyData>(
    ['anomalies', startDate, endDate, provider, accountId],
    () => api.get('/anomalies', { params: { start_date: startDate, end_date: endDate, provider, account_id: accountId } }).then(r => r.data),
    { enabled: !!startDate && !!endDate }
  );
}
