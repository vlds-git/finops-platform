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
    () => api.get('/costs/trends', { params: { period: period.replace('d', '') } }).then(r => r.data)
  );
}

export function useServices(startDate?: string, endDate?: string) {
  return useQuery<ServiceCost[]>(
    ['services', startDate, endDate],
    () => api.get('/costs/services', { params: { start_date: startDate, end_date: endDate } }).then(r => r.data),
    { enabled: !!startDate && !!endDate }
  );
}

export function useRegions(startDate?: string, endDate?: string) {
  return useQuery<{ name: string; cost: number }[]>(
    ['regions', startDate, endDate],
    () => api.get('/costs/regions', { params: { start_date: startDate, end_date: endDate } }).then(r => r.data),
    { enabled: !!startDate && !!endDate }
  );
}

export function useProviders(startDate?: string, endDate?: string) {
  return useQuery<{ name: string; cost: number }[]>(
    ['providers', startDate, endDate],
    () => api.get('/costs/providers', { params: { start_date: startDate, end_date: endDate } }).then(r => r.data),
    { enabled: !!startDate && !!endDate }
  );
}

export function useKPIs(startDate?: string, endDate?: string) {
  return useQuery<{ name: string; value: number; unit?: string }[]>(
    ['kpis', startDate, endDate],
    () => api.get('/kpis', { params: { start_date: startDate, end_date: endDate } }).then(r => r.data),
    { enabled: !!startDate && !!endDate }
  );
}

export function useExecutiveDashboard(startDate?: string, endDate?: string) {
  return useQuery<ExecutiveDashboard>(
    ['executive-dashboard', startDate, endDate],
    () => api.get('/dashboard/executive', { params: { start_date: startDate, end_date: endDate } }).then(r => r.data),
    { enabled: !!startDate && !!endDate }
  );
}

export function useOperationalDashboard(startDate?: string, endDate?: string) {
  return useQuery<OperationalDashboard>(
    ['operational-dashboard', startDate, endDate],
    () => api.get('/dashboard/operational', { params: { start_date: startDate, end_date: endDate } }).then(r => r.data),
    { enabled: !!startDate && !!endDate }
  );
}
