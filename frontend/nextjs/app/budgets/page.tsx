'use client';

import { MainLayout } from '@/components/layout/MainLayout';
import { DataTable } from '@/components/ui/DataTable';
import { Badge } from '@/components/ui/Badge';
import { useBudgets } from '@/hooks/useBudgets';
import { useCurrency } from '@/contexts/CurrencyContext';
import type { Budget } from '@/types';
import { formatCurrency } from '@/lib/utils';
import { Plus } from 'lucide-react';

export default function BudgetsPage() {
  const { currency } = useCurrency();
  const { data: budgets = [] } = useBudgets();

  const totalBudgeted = budgets.reduce((s, b) => s + b.amount, 0);
  const totalSpent = budgets.reduce((s, b) => s + b.spent, 0);

  const columns = [
    { key: 'name', header: 'Nome' },
    { key: 'period', header: 'Período' },
    { key: 'amount', header: 'Orçado', render: (b: Budget) => formatCurrency(b.amount, currency) },
    { key: 'spent', header: 'Realizado', render: (b: Budget) => formatCurrency(b.spent, currency) },
    { key: 'pct', header: '%', render: (b: Budget) => {
      const pct = b.amount > 0 ? (b.spent / b.amount) * 100 : 0;
      return <span className={pct > 90 ? 'text-amber-400' : 'text-white'}>{pct.toFixed(0)}%</span>;
    }},
    { key: 'status', header: 'Status', render: (b: Budget) => {
      const pct = b.amount > 0 ? b.spent / b.amount : 0;
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
              <p className="text-xl font-bold text-white">{formatCurrency(totalBudgeted, currency)}</p>
            </div>
            <div className="card p-4">
              <p className="text-sm text-slate-400">Total Realizado</p>
              <p className="text-xl font-bold text-white">{formatCurrency(totalSpent, currency)}</p>
            </div>
            <div className="card p-4">
              <p className="text-sm text-slate-400">Variação</p>
              <p className="text-xl font-bold text-emerald-400">
                {totalBudgeted > 0 ? ((1 - totalSpent / totalBudgeted) * 100).toFixed(0) : 0}%
              </p>
            </div>
          </div>
          <button className="px-4 py-2 bg-sky-600 hover:bg-sky-500 text-white rounded-lg text-sm font-medium transition-colors flex items-center gap-2">
            <Plus className="w-4 h-4" />
            Novo Budget
          </button>
        </div>

        <DataTable columns={columns} data={budgets} />
      </div>
    </MainLayout>
  );
}
