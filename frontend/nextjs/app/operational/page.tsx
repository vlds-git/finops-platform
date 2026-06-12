'use client';

import { MainLayout } from '@/components/layout/MainLayout';
import { BarChart } from '@/components/charts/BarChart';
import { useOperationalDashboard } from '@/hooks/useCosts';

export default function OperationalPage() {
  const { data: dashboard } = useOperationalDashboard();

  const byService = dashboard?.by_service?.map((i) => ({ name: i.name, value: i.cost })) || [];
  const byEnvironment = dashboard?.by_environment?.map((i) => ({ name: i.name, value: i.cost })) || [];
  const byRegion = dashboard?.by_region?.map((i) => ({ name: i.name, value: i.cost })) || [];
  const byBU = dashboard?.by_business_unit?.map((i) => ({ name: i.name, value: i.cost })) || [];

  return (
    <MainLayout title="Dashboard Operacional">
      <div className="space-y-6">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className="card p-5">
            <h3 className="text-sm font-semibold text-white mb-4">Custos por Ambiente</h3>
            <BarChart data={byEnvironment} color="#34d399" />
          </div>
          <div className="card p-5">
            <h3 className="text-sm font-semibold text-white mb-4">Custos por Região</h3>
            <BarChart data={byRegion} color="#818cf8" />
          </div>
        </div>
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className="card p-5">
            <h3 className="text-sm font-semibold text-white mb-4">Custos por Serviço</h3>
            <BarChart data={byService} color="#38bdf8" />
          </div>
          <div className="card p-5">
            <h3 className="text-sm font-semibold text-white mb-4">Custos por Business Unit</h3>
            <BarChart data={byBU} horizontal color="#c084fc" />
          </div>
        </div>
      </div>
    </MainLayout>
  );
}
