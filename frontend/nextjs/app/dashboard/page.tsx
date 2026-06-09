'use client';

import { MainLayout } from '@/components/layout/MainLayout';
import { KPICard } from '@/components/ui/KPICard';
import { TrendChart } from '@/components/charts/TrendChart';
import { BarChart } from '@/components/charts/BarChart';
import { PieChart } from '@/components/charts/PieChart';
import { useExecutiveDashboard, useCostTrends } from '@/hooks/useCosts';
import type { KPIData, CostTrend } from '@/types';

const mockKPIs: KPIData[] = [
  { name: 'Custo Total (30d)', value: 185000, unit: 'BRL', trend: 0.082, status: 'warning' },
  { name: 'Forecast (30d)', value: 195000, unit: 'BRL', trend: 0.054, status: 'warning' },
  { name: 'Economia Potencial', value: 32000, unit: 'BRL', trend: -0.153, status: 'good' },
  { name: 'Budget Utilization', value: 0.72, unit: 'ratio', trend: 0.05, target: 0.80, status: 'good' },
];

const mockTrends: CostTrend[] = Array.from({ length: 30 }, (_, i) => ({
  date: new Date(Date.now() - (29 - i) * 24 * 60 * 60 * 1000).toISOString().split('T')[0],
  cost: 5000 + i * 50 + Math.sin(i / 5) * 500 + Math.random() * 300,
  usage: 100 + i * 2,
}));

const mockServices = [
  { name: 'Compute', value: 85000 },
  { name: 'Storage', value: 35000 },
  { name: 'Network', value: 25000 },
  { name: 'Database', value: 25000 },
  { name: 'Others', value: 15000 },
];

const mockApps = [
  { name: 'ERP', value: 45000 },
  { name: 'CRM', value: 32000 },
  { name: 'Data Lake', value: 28000 },
  { name: 'E-commerce', value: 25000 },
  { name: 'Others', value: 15000 },
];

export default function DashboardPage() {
  const { data: dashboard } = useExecutiveDashboard();
  const { data: trends } = useCostTrends();

  return (
    <MainLayout title="Dashboard Executivo">
      <div className="space-y-6">
        {/* KPI Cards */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {mockKPIs.map((kpi) => (
            <KPICard key={kpi.name} data={kpi} />
          ))}
        </div>

        {/* Charts Row 1 */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className="card p-5">
            <h3 className="text-sm font-semibold text-white mb-4">Tendência de Custos</h3>
            <TrendChart data={mockTrends} />
          </div>
          <div className="card p-5">
            <h3 className="text-sm font-semibold text-white mb-4">Custos por Serviço</h3>
            <BarChart data={mockServices} horizontal />
          </div>
        </div>

        {/* Charts Row 2 */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="card p-5">
            <h3 className="text-sm font-semibold text-white mb-4">Top Aplicações</h3>
            <PieChart data={mockApps} />
          </div>
          <div className="card p-5">
            <h3 className="text-sm font-semibold text-white mb-4">Custos por Provedor</h3>
            <BarChart 
              data={[
                { name: 'Huawei', value: 85000 },
                { name: 'Azure', value: 65000 },
                { name: 'AWS', value: 35000 },
              ]} 
              color="#38bdf8"
            />
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
