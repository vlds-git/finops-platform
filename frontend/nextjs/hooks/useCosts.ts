'use client';

import { useQuery } from 'react-query';
import api from '@/lib/api';
import { useCurrency } from '@/contexts/CurrencyContext';
import type { CostSummary, CostTrend, ServiceCost, ExecutiveDashboard, OperationalDashboard } from '@/types';

export function useCosts(startDate?: string, endDate?: string, provider?: string) {
  const { currency } = useCurrency();
  return useQuery<CostSummary>(
    ['costs', startDate, endDate, provider, currency],
    () => api.get('/costs', { params: { start_date: startDate, end_date: endDate, provider, currency } }).then(r => r.data),
    { enabled: !!startDate && !!endDate }
  );
}

export function useCostTrends(startDate?: string, endDate?: string) {
  const { currency } = useCurrency();
  return useQuery<CostTrend[]>(
    ['costs-trends', startDate, endDate, currency],
    () => api.get('/costs/trends', { params: { start_date: startDate, end_date: endDate, currency } }).then(r => r.data),
    { enabled: !!startDate && !!endDate }
  );
}

export function useServices(startDate?: string, endDate?: string) {
  const { currency } = useCurrency();
  return useQuery<ServiceCost[]>(
    ['services', startDate, endDate, currency],
    () => api.get('/costs/services', { params: { start_date: startDate, end_date: endDate, currency } }).then(r => r.data),
    { enabled: !!startDate && !!endDate }
  );
}

export interface RegionCost {
  region: string;
  cost: number;
}

export function useRegions(startDate?: string, endDate?: string) {
  const { currency } = useCurrency();
  return useQuery<RegionCost[]>(
    ['regions', startDate, endDate, currency],
    () => api.get('/costs/regions', { params: { start_date: startDate, end_date: endDate, currency } }).then(r => r.data),
    { enabled: !!startDate && !!endDate }
  );
}

export interface ProviderCost {
  provider: string;
  cost: number;
}

export function useProviders(startDate?: string, endDate?: string) {
  const { currency } = useCurrency();
  return useQuery<ProviderCost[]>(
    ['providers', startDate, endDate, currency],
    () => api.get('/costs/providers', { params: { start_date: startDate, end_date: endDate, currency } }).then(r => r.data),
    { enabled: !!startDate && !!endDate }
  );
}

export function useKPIs(startDate?: string, endDate?: string) {
  const { currency } = useCurrency();
  return useQuery<{ name: string; value: number; unit?: string }[]>(
    ['kpis', startDate, endDate, currency],
    () => api.get('/kpis', { params: { start_date: startDate, end_date: endDate, currency } }).then(r => r.data),
    { enabled: !!startDate && !!endDate }
  );
}

export function useExecutiveDashboard(startDate?: string, endDate?: string) {
  const { currency } = useCurrency();
  return useQuery<ExecutiveDashboard>(
    ['executive-dashboard', startDate, endDate, currency],
    () => api.get('/dashboard/executive', { params: { start_date: startDate, end_date: endDate, currency } }).then(r => r.data),
    { enabled: !!startDate && !!endDate }
  );
}

export function useOperationalDashboard(startDate?: string, endDate?: string) {
  const { currency } = useCurrency();
  return useQuery<OperationalDashboard>(
    ['operational-dashboard', startDate, endDate, currency],
    () => api.get('/dashboard/operational', { params: { start_date: startDate, end_date: endDate, currency } }).then(r => r.data),
    { enabled: !!startDate && !!endDate }
  );
}
