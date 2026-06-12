'use client';

import { MainLayout } from '@/components/layout/MainLayout';
import { KPICard } from '@/components/ui/KPICard';
import { TrendChart } from '@/components/charts/TrendChart';
import { BarChart } from '@/components/charts/BarChart';
import { PieChart } from '@/components/charts/PieChart';
import { useExecutiveDashboard, useCostTrends, useCosts } from '@/hooks/useCosts';
import { useCurrency } from '@/contexts/CurrencyContext';
import { useEffect, useState } from 'react';
import type { KPIData } from '@/types';

export default function DashboardPage() {
  const { currency } = useCurrency();
  const [startDate, setStartDate] = useState('');
  const [endDate, setEndDate] = useState('');

  useEffect(() => {
    const end = new Date().toISOString().split('T')[0];
    const start = new Date(Date.now() - 30 * 24 * 60 * 60 * 1000).toISOString().split('T')[0];
    setStartDate(start);
    setEndDate(end);
  }, []);

  const { data: dashboard } = useExecutiveDashboard();
  const { data: trends } = useCostTrends('30');
  const { data: costs } = useCosts(startDate, endDate);

  const kpis: KPIData[] = [
    { name: `Custo Total (30d)`, value: costs?.total_cost ?? dashboard?.total_cost ?? 0, unit: currency, trend: 0.05, status: 'warning' },
    { name: 'Forecast (30d)', value: dashboard?.forecast_30d ?? 0, unit: currency, trend: 0.054, status: 'warning' },
    { name: 'Economia Potencial', value: dashboard?.potential_savings ?? 0, unit: currency, trend: -0.153, status: 'good' },
    { name: 'Budget Utilization', value: 0.72, unit: 'ratio', trend: 0.05, target: 0.80, status: 'good' },
  ];

  const topServices = dashboard?.top_services?.map((s) => ({ name: s.name, value: s.cost })) || [];
  const topApps = dashboard?.top_applications?.map((a) => ({ name: a.name, value: a.cost })) || [];
  const providerData = dashboard?.top_services
    ? [{ name: 'Huawei', value: dashboard.top_services.reduce((acc, s) => acc + s.cost, 0) }]
    : [];

  return (
    <MainLayout title="Dashboard Executivo">
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
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="card p-5">
            <h3 className="text-sm font-semibold text-white mb-4">Top Aplicações</h3>
            <PieChart data={topApps} />
          </div>
          <div className="card p-5">
            <h3 className="text-sm font-semibold text-white mb-4">Custos por Provedor</h3>
            <BarChart data={providerData} color="#38bdf8" />
          </div>
          <div className="card p-5">
            <h3 className="text-sm font-semibold text-white mb-4">KPIs Estratégicos</h3>
            <div className="space-y-4">
              {[
                { name: 'Cost Efficiency', value: 85, target: 90 },
                { name: 'Budget Health', value: 92, target: 95 },
                { name: 'Tag Coverage', value: 68, target: 80 },
                { name: 'Savings Realized', value: 78, target: 85 },
              ].map((kpi) => (
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
                      style={{ width: `${kpi.value}%` }}
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
