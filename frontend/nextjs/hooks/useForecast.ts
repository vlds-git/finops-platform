'use client';

import { useQuery } from 'react-query';
import api from '@/lib/api';
import type { ForecastData } from '@/types';

export function useForecast(provider: string, accountId: string, period: string, model = 'ensemble') {
  return useQuery<ForecastData>(
    ['forecast', provider, accountId, period, model],
    () => api.post('/forecast', { provider, account_id: accountId, period, model }).then(r => r.data)
  );
}
