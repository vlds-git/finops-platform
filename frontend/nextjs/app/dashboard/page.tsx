'use client';

import { MainLayout } from '@/components/layout/MainLayout';
import { KPICard } from '@/components/ui/KPICard';
import { TrendChart } from '@/components/charts/TrendChart';
import { BarChart } from '@/components/charts/BarChart';
import { useCosts, useCostTrends, useExecutiveDashboard, useRegions, useKPIs } from '@/hooks/useCosts';
import { useCurrency } from '@/contexts/CurrencyContext';
import { useEffect, useState, useMemo } from 'react';
import type { KPIData } from '@/types';
import type { PeriodOption, PeriodRange } from '@/components/ui/FilterBar';

function formatDateInput(d: Date): string {
  return d.toISOString().split('T')[0];
}

function getRangeForOption(option: PeriodOption): PeriodRange {
  const end = new Date();
  const start = new Date();
  switch (option) {
    case '24h':
      start.setHours(end.getHours() - 24);
      break;
    case '48h':
      start.setHours(end.getHours() - 48);
      break;
    case '7d':
      start.setDate(end.getDate() - 7);
      break;
    case '90d':
      start.setDate(end.getDate() - 90);
      break;
    case 'custom':
    case '30d':
    default:
      start.setDate(end.getDate() - 30);
  }
  return { startDate: formatDateInput(start), endDate: formatDateInput(end) };
}

function computePreviousRange(range: PeriodRange): PeriodRange {
  const start = new Date(range.startDate);
  const end = new Date(range.endDate);
  const duration = end.getTime() - start.getTime();
  return {
    startDate: formatDateInput(new Date(start.getTime() - duration - 1)),
    endDate: formatDateInput(new Date(end.getTime() - duration - 1)),
  };
}

function computeTrend(current: number, previous: number): number {
  if (!previous || previous === 0) return 0;
  return (current - previous) / previous;
}

export default function DashboardPage() {
  const { currency } = useCurrency();
  const [period, setPeriod] = useState<PeriodOption>('30d');
  const [range, setRange] = useState<PeriodRange>(() => getRangeForOption('30d'));

  useEffect(() => {
    setRange(getRangeForOption(period));
  }, [period]);

  const handlePeriodChange = (option: PeriodOption, newRange?: PeriodRange) => {
    setPeriod(option);
    if (newRange) setRange(newRange);
  };

  const previousRange = useMemo(() => computePreviousRange(range), [range]);

  const { data: costs } = useCosts(range.startDate, range.endDate);
  const { data: previousCosts } = useCosts(previousRange.startDate, previousRange.endDate);
  const trendDays = useMemo(() => {
    if (period === 'custom') {
      const start = new Date(range.startDate);
      const end = new Date(range.endDate);
      return Math.max(1, Math.round((end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24)));
    }
    return Number(period.replace('d', ''));
  }, [period, range]);
  const { data: trends } = useCostTrends(String(trendDays));
  const { data: dashboard } = useExecutiveDashboard(range.startDate, range.endDate);
  const { data: regions } = useRegions(range.startDate, range.endDate);
  const { data: kpisApi } = useKPIs(range.startDate, range.endDate);

  const totalCost = costs?.total_cost ?? dashboard?.total_cost ?? 0;
  const previousTotalCost = previousCosts?.total_cost ?? 0;

  const kpis: KPIData[] = useMemo(() => {
    const base: KPIData[] = [
      {
        name: 'Custo Total',
        value: totalCost,
        unit: currency,
        trend: computeTrend(totalCost, previousTotalCost),
        status: totalCost > previousTotalCost ? 'warning' : 'good',
      },
      {
        name: 'Forecast (30d)',
        value: dashboard?.forecast_30d ?? 0,
        unit: currency,
        trend: computeTrend(dashboard?.forecast_30d ?? 0, totalCost),
        status: (dashboard?.forecast_30d ?? 0) > totalCost * 1.1 ? 'warning' : 'good',
      },
      {
        name: 'Serviços Ativos',
        value: costs?.service_count ?? 0,
        unit: 'count',
        trend: computeTrend(costs?.service_count ?? 0, previousCosts?.service_count ?? 0),
        status: 'good',
      },
      {
        name: 'Recursos Rastreados',
        value: costs?.resource_count ?? 0,
        unit: 'count',
        trend: computeTrend(costs?.resource_count ?? 0, previousCosts?.resource_count ?? 0),
        status: 'good',
      },
    ];

    if (kpisApi && kpisApi.length > 0) {
      const budgetUtil = kpisApi.find((k) => k.name?.toLowerCase().includes('budget'));
      if (budgetUtil) {
        base.push({
          name: 'Budget Utilization',
          value: budgetUtil.value,
          unit: 'ratio',
          trend: 0,
          target: 0.8,
          status: budgetUtil.value > 0.9 ? 'danger' : budgetUtil.value > 0.8 ? 'warning' : 'good',
        });
      }
    }

    return base;
  }, [costs, previousCosts, dashboard, kpisApi, currency, totalCost, previousTotalCost]);

  const topServices = dashboard?.top_services?.map((s) => ({ name: s.name, value: s.cost })) || [];
  const regionData = regions?.map((r) => ({ name: r.name || 'N/A', value: r.cost })) || [];

  return (
    <MainLayout title="Dashboard Executivo" onPeriodChange={handlePeriodChange}>
      <div className="space-y-6">
        {/* KPI Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {kpis.map((kpi) => (
            <KPICard key={kpi.name} data={kpi} />
          ))}
        </div>

        {/* Charts Row 1 */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className="card p-5">
            <h3 className="text-sm font-semibold text-white mb-4">Tendência de Custos</h3>
            <TrendChart data={trends || []} />
          </div>
          <div className="card p-5">
            <h3 className="text-sm font-semibold text-white mb-4">Custos por Serviço</h3>
            <BarChart data={topServices} horizontal />
          </div>
        </div>

        {/* Charts Row 2 */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className="card p-5">
            <h3 className="text-sm font-semibold text-white mb-4">Custos por Região</h3>
            <BarChart data={regionData} color="#38bdf8" />
          </div>
          <div className="card p-5">
            <h3 className="text-sm font-semibold text-white mb-4">KPIs Estratégicos</h3>
            <div className="space-y-4">
              {(kpisApi && kpisApi.length > 0
                ? kpisApi.slice(0, 4).map((k) => ({ name: k.name, value: Math.min(Math.round(k.value * 100), 100), target: 80 }))
                : [
                    { name: 'Cost Efficiency', value: 85, target: 90 },
                    { name: 'Budget Health', value: 92, target: 95 },
                    { name: 'Tag Coverage', value: 68, target: 80 },
                    { name: 'Savings Realized', value: 78, target: 85 },
                  ]
              ).map((kpi) => (
                <div key={kpi.name}>
                  <div className="flex justify-between text-sm mb-1">
                    <span className="text-slate-400">{kpi.name}</span>
                    <span className={kpi.value >= kpi.target ? 'text-emerald-400' : 'text-amber-400'}>
                      {kpi.value}%
                    </span>
                  </div>
                  <div className="h-2 bg-slate-700 rounded-full">
                    <div
                      className={`h-full rounded-full ${kpi.value >= kpi.target ? 'bg-emerald-500' : 'bg-amber-500'}`}
                      style={{ width: `${Math.min(kpi.value, 100)}%` }}
                    />
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </div>
    </MainLayout>
  );
}
