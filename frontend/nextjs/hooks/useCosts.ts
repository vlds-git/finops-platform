'use client';

import { useQuery } from 'react-query';
import api from '@/lib/api';
import type { CostSummary, CostTrend, ServiceCost, ExecutiveDashboard, OperationalDashboard } from '@/types';

export function useCosts(startDate?: string, endDate?: string, provider?: string) {
  return useQuery<CostSummary>(
    ['costs', startDate, endDate, provider],
    () => api.get('/costs', { params: { start_date: startDate, end_date: endDate, provider } }).then(r => r.data),
    { enabled: !!startDate && !!endDate }
  );
}

export function useCostTrends(period = '30') {
  return useQuery<CostTrend[]>(
    ['costs-trends', period],
    () => api.get('/costs/trends', { params: { period } }).then(r => r.data)
  );
}

export function useServices() {
  return useQuery<ServiceCost[]>(
    'services',
    () => api.get('/costs/services').then(r => r.data)
  );
}

export function useExecutiveDashboard() {
  return useQuery<ExecutiveDashboard>(
    'executive-dashboard',
    () => api.get('/dashboard/executive').then(r => r.data)
  );
}

export function useOperationalDashboard() {
  return useQuery<OperationalDashboard>(
    'operational-dashboard',
    () => api.get('/dashboard/operational').then(r => r.data)
  );
}
