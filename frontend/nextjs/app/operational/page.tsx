'use client';

import { MainLayout } from '@/components/layout/MainLayout';
import { BarChart } from '@/components/charts/BarChart';

const envData = [
  { name: 'Production', value: 120000 },
  { name: 'Staging', value: 35000 },
  { name: 'Development', value: 30000 },
  { name: 'QA', value: 15000 },
];

const regionData = [
  { name: 'sa-brazil-1', value: 95000 },
  { name: 'east-us-1', value: 90000 },
  { name: 'west-eu-1', value: 45000 },
  { name: 'ap-singapore-1', value: 20000 },
];

const buData = [
  { name: 'Platform', value: 62000 },
  { name: 'IT', value: 75000 },
  { name: 'Finance', value: 45000 },
  { name: 'Sales', value: 38000 },
  { name: 'Marketing', value: 22000 },
  { name: 'HR', value: 15000 },
];

export default function OperationalPage() {
  return (
    <MainLayout title="Dashboard Operacional">
      <div className="space-y-6">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className="card p-5">
            <h3 className="text-sm font-semibold text-white mb-4">Custos por Ambiente</h3>
            <BarChart data={envData} color="#34d399" />
          </div>
          <div className="card p-5">
            <h3 className="text-sm font-semibold text-white mb-4">Custos por Região</h3>
            <BarChart data={regionData} color="#818cf8" />
          </div>
        </div>
        <div className="card p-5">
          <h3 className="text-sm font-semibold text-white mb-4">Custos por Business Unit</h3>
          <BarChart data={buData} horizontal color="#c084fc" />
        </div>
      </div>
    </MainLayout>
  );
}
