'use client';

import { useQuery } from 'react-query';
import api from '@/lib/api';
import type { ForecastData } from '@/types';

export function useForecast(startDate?: string, endDate?: string, provider?: string, accountId?: string, forecastDays = 30) {
  return useQuery<ForecastData>(
    ['forecast', startDate, endDate, provider, accountId, forecastDays],
    () => api.get('/forecast', { params: { start_date: startDate, end_date: endDate, provider, account_id: accountId, forecast_days: forecastDays } }).then(r => r.data),
    { enabled: !!startDate && !!endDate }
  );
}
