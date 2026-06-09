'use client';

import { MainLayout } from '@/components/layout/MainLayout';
import { MultiCloudChart } from '@/components/charts/MultiCloudChart';
import type { MultiCloudData } from '@/types';
import { formatCurrency } from '@/lib/utils';

const providers: MultiCloudData[] = [
  { provider: 'Huawei Cloud', account_count: 3, cost: 85000, percentage: 46 },
  { provider: 'Azure', account_count: 2, cost: 65000, percentage: 35 },
  { provider: 'AWS', account_count: 4, cost: 35000, percentage: 19 },
];

const chartData = {
  months: ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun'],
  huawei: [75000, 78000, 80000, 82000, 85000, 85000],
  azure: [55000, 58000, 60000, 62000, 65000, 65000],
  aws: [30000, 31000, 32000, 33000, 35000, 35000],
};

export default function MultiCloudPage() {
  return (
    <MainLayout title="Multi-Cloud">
      <div className="space-y-6">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {providers.map((p) => (
            <div key={p.provider} className="card p-5">
              <div className="flex items-center gap-3 mb-4">
                <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${
                  p.provider === 'Huawei Cloud' ? 'bg-red-600' :
                  p.provider === 'Azure' ? 'bg-blue-600' : 'bg-orange-600'
                }`}>
                  <span className="text-white font-bold text-sm">
                    {p.provider === 'Huawei Cloud' ? 'HW' : p.provider === 'Azure' ? 'AZ' : 'AWS'}
                  </span>
                </div>
                <div>
                  <h3 className="font-semibold text-white">{p.provider}</h3>
                  <p className="text-xs text-slate-400">{p.account_count} contas</p>
                </div>
              </div>
              <p className="text-2xl font-bold text-white">{formatCurrency(p.cost)}</p>
              <p className="text-sm text-slate-400">{p.percentage}% do total</p>
              <div className="mt-3 h-1 bg-slate-700 rounded-full">
                <div 
                  className={`h-full rounded-full ${
                    p.provider === 'Huawei Cloud' ? 'bg-red-500' :
                    p.provider === 'Azure' ? 'bg-blue-500' : 'bg-orange-500'
                  }`}
                  style={{ width: `${p.percentage}%` }}
                />
              </div>
            </div>
          ))}
        </div>

        <div className="card p-5">
          <h3 className="text-sm font-semibold text-white mb-4">Comparativo Multi-Cloud</h3>
          <MultiCloudChart data={chartData} className="h-[350px]" />
        </div>
      </div>
    </MainLayout>
  );
}
