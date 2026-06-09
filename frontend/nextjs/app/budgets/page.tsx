'use client';

import { MainLayout } from '@/components/layout/MainLayout';
import { DataTable } from '@/components/ui/DataTable';
import { Badge } from '@/components/ui/Badge';
import { useBudgets } from '@/hooks/useBudgets';
import type { Budget } from '@/types';
import { formatCurrency } from '@/lib/utils';
import { Plus } from 'lucide-react';

const mockBudgets: Budget[] = [
  {
    id: '1', name: 'Produção - Q1', amount: 150000, spent: 120000, remaining: 30000,
    period: 'quarterly', start_date: '2024-01-01', end_date: '2024-03-31',
    alert_threshold: 0.90, provider: 'all', account_id: 'all', created_at: '2024-01-01',
  },
  {
    id: '2', name: 'Desenvolvimento', amount: 50000, spent: 35000, remaining: 15000,
    period: 'quarterly', start_date: '2024-01-01', end_date: '2024-03-31',
    alert_threshold: 0.85, provider: 'all', account_id: 'all', created_at: '2024-01-01',
  },
  {
    id: '3', name: 'Staging', amount: 30000, spent: 28000, remaining: 2000,
    period: 'quarterly', start_date: '2024-01-01', end_date: '2024-03-31',
    alert_threshold: 0.90, provider: 'all', account_id: 'all', created_at: '2024-01-01',
  },
  {
    id: '4', name: 'Disaster Recovery', amount: 20000, spent: 2000, remaining: 18000,
    period: 'annual', start_date: '2024-01-01', end_date: '2024-12-31',
    alert_threshold: 0.95, provider: 'all', account_id: 'all', created_at: '2024-01-01',
  },
];

export default function BudgetsPage() {
  const { data: budgets } = useBudgets();
  const totalBudgeted = mockBudgets.reduce((s, b) => s + b.amount, 0);
  const totalSpent = mockBudgets.reduce((s, b) => s + b.spent, 0);

  const columns = [
    { key: 'name', header: 'Nome' },
    { key: 'period', header: 'Período' },
    { key: 'amount', header: 'Orçado', render: (b: Budget) => formatCurrency(b.amount) },
    { key: 'spent', header: 'Realizado', render: (b: Budget) => formatCurrency(b.spent) },
    { key: 'pct', header: '%', render: (b: Budget) => {
      const pct = (b.spent / b.amount) * 100;
      return <span className={pct > 90 ? 'text-amber-400' : 'text-white'}>{pct.toFixed(0)}%</span>;
    }},
    { key: 'status', header: 'Status', render: (b: Budget) => {
      const pct = b.spent / b.amount;
      return <Badge variant={pct > b.alert_threshold ? 'warning' : 'success'}>
        {pct > b.alert_threshold ? 'Warning' : 'On Track'}
      </Badge>;
    }},
    { key: 'alert', header: 'Alerta', render: (b: Budget) => `${(b.alert_threshold * 100).toFixed(0)}%` },
  ];

  return (
    <MainLayout title="Budgets">
      <div className="space-y-6">
        <div className="flex justify-between items-center">
          <div className="flex gap-4">
            <div className="card p-4">
              <p className="text-sm text-slate-400">Total Orçado</p>
              <p className="text-xl font-bold text-white">{formatCurrency(totalBudgeted)}</p>
            </div>
            <div className="card p-4">
              <p className="text-sm text-slate-400">Total Realizado</p>
              <p className="text-xl font-bold text-white">{formatCurrency(totalSpent)}</p>
            </div>
            <div className="card p-4">
              <p className="text-sm text-slate-400">Variação</p>
              <p className="text-xl font-bold text-emerald-400">
                {((1 - totalSpent / totalBudgeted) * 100).toFixed(0)}%
              </p>
            </div>
          </div>
          <button className="px-4 py-2 bg-sky-600 hover:bg-sky-500 text-white rounded-lg text-sm font-medium transition-colors flex items-center gap-2">
            <Plus className="w-4 h-4" />
            Novo Budget
          </button>
        </div>

        <DataTable columns={columns} data={mockBudgets} />
      </div>
    </MainLayout>
  );
}
