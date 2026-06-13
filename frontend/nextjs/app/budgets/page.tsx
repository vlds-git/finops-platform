'use client';

import { useState } from 'react';
import { MainLayout } from '@/components/layout/MainLayout';
import { DataTable } from '@/components/ui/DataTable';
import { Badge } from '@/components/ui/Badge';
import { useBudgets, useCreateBudget, useUpdateBudget, useDeleteBudget, useCloudAccounts } from '@/hooks/useBudgets';
import { useCurrency } from '@/contexts/CurrencyContext';
import type { Budget } from '@/types';
import { formatCurrency } from '@/lib/utils';
import { Plus, Edit, Trash2, X } from 'lucide-react';

interface BudgetFormData {
  id?: string;
  name: string;
  amount: number;
  period: string;
  start_date: string;
  end_date: string;
  alert_threshold: number;
  provider: string;
  account_id: string;
}

const emptyForm: BudgetFormData = {
  name: '',
  amount: 0,
  period: 'monthly',
  start_date: '',
  end_date: '',
  alert_threshold: 0.8,
  provider: 'huawei',
  account_id: '',
};

export default function BudgetsPage() {
  const { currency } = useCurrency();
  const { data: budgets = [] } = useBudgets();
  const { data: accounts = [] } = useCloudAccounts();
  const createBudget = useCreateBudget();
  const updateBudget = useUpdateBudget();
  const deleteBudget = useDeleteBudget();
  const [isOpen, setIsOpen] = useState(false);
  const [form, setForm] = useState<BudgetFormData>(emptyForm);

  const totalBudgeted = budgets.reduce((s, b) => s + b.amount, 0);
  const totalSpent = budgets.reduce((s, b) => s + b.spent, 0);

  const openCreate = () => {
    const today = new Date().toISOString().split('T')[0];
    const endOfMonth = new Date();
    endOfMonth.setMonth(endOfMonth.getMonth() + 1);
    setForm({ ...emptyForm, start_date: today, end_date: endOfMonth.toISOString().split('T')[0] });
    setIsOpen(true);
  };

  const openEdit = (b: Budget) => {
    setForm({
      id: b.id,
      name: b.name,
      amount: b.amount,
      period: b.period,
      start_date: b.start_date,
      end_date: b.end_date,
      alert_threshold: b.alert_threshold,
      provider: b.provider,
      account_id: b.account_id,
    });
    setIsOpen(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const payload = {
      name: form.name,
      amount: Number(form.amount),
      period: form.period,
      start_date: form.start_date,
      end_date: form.end_date,
      alert_threshold: Number(form.alert_threshold),
      provider: form.provider,
      account_id: form.account_id,
    };
    if (form.id) {
      await updateBudget.mutateAsync({ id: form.id, ...payload });
    } else {
      await createBudget.mutateAsync(payload);
    }
    setIsOpen(false);
    setForm(emptyForm);
  };

  const handleDelete = async (id: string) => {
    if (confirm('Deseja realmente excluir este budget?')) {
      await deleteBudget.mutateAsync(id);
    }
  };

  const columns = [
    { key: 'name', header: 'Nome' },
    { key: 'provider', header: 'Provedor' },
    { key: 'account_id', header: 'Account' },
    { key: 'period', header: 'Período' },
    { key: 'amount', header: 'Orçado', render: (b: Budget) => formatCurrency(b.amount, currency) },
    { key: 'spent', header: 'Realizado', render: (b: Budget) => formatCurrency(b.spent, currency) },
    { key: 'remaining', header: 'Restante', render: (b: Budget) => formatCurrency(b.remaining, currency) },
    { key: 'pct', header: '%', render: (b: Budget) => {
      const pct = b.amount > 0 ? (b.spent / b.amount) * 100 : 0;
      return <span className={pct > 90 ? 'text-amber-400' : pct > 100 ? 'text-red-400' : 'text-white'}>{pct.toFixed(0)}%</span>;
    }},
    { key: 'status', header: 'Status', render: (b: Budget) => {
      const pct = b.amount > 0 ? b.spent / b.amount : 0;
      let variant: 'success' | 'warning' | 'danger' = 'success';
      let label = 'On Track';
      if (pct > 1) { variant = 'danger'; label = 'Over Budget'; }
      else if (pct > b.alert_threshold) { variant = 'warning'; label = 'Warning'; }
      return <Badge variant={variant}>{label}</Badge>;
    }},
    { key: 'alert', header: 'Alerta', render: (b: Budget) => `${(b.alert_threshold * 100).toFixed(0)}%` },
    { key: 'actions', header: 'Ações', render: (b: Budget) => (
      <div className="flex items-center gap-2">
        <button onClick={() => openEdit(b)} className="text-sky-400 hover:text-sky-300 flex items-center gap-1">
          <Edit className="w-4 h-4" />
        </button>
        <button onClick={() => handleDelete(b.id)} className="text-red-400 hover:text-red-300 flex items-center gap-1">
          <Trash2 className="w-4 h-4" />
        </button>
      </div>
    )},
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
              <p className={`text-xl font-bold ${totalSpent > totalBudgeted ? 'text-red-400' : 'text-emerald-400'}`}>
                {totalBudgeted > 0 ? (((totalSpent - totalBudgeted) / totalBudgeted) * 100).toFixed(0) : 0}%
              </p>
            </div>
          </div>
          <button
            onClick={openCreate}
            className="px-4 py-2 bg-sky-600 hover:bg-sky-500 text-white rounded-lg text-sm font-medium transition-colors flex items-center gap-2"
          >
            <Plus className="w-4 h-4" />
            Novo Budget
          </button>
        </div>

        <DataTable columns={columns} data={budgets} />

        {isOpen && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
            <div className="bg-slate-800 border border-slate-700 rounded-lg p-6 w-full max-w-lg">
              <div className="flex justify-between items-center mb-4">
                <h2 className="text-lg font-semibold text-white">{form.id ? 'Editar Budget' : 'Novo Budget'}</h2>
                <button onClick={() => setIsOpen(false)} className="text-slate-400 hover:text-white">
                  <X className="w-5 h-5" />
                </button>
              </div>
              <form onSubmit={handleSubmit} className="space-y-4">
                <div>
                  <label className="block text-sm text-slate-400 mb-1">Nome</label>
                  <input
                    required
                    value={form.name}
                    onChange={(e) => setForm({ ...form, name: e.target.value })}
                    className="w-full bg-slate-900 border border-slate-700 rounded-lg px-3 py-2 text-white text-sm"
                  />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm text-slate-400 mb-1">Valor Orçado</label>
                    <input
                      required
                      type="number"
                      min={0}
                      step={0.01}
                      value={form.amount}
                      onChange={(e) => setForm({ ...form, amount: Number(e.target.value) })}
                      className="w-full bg-slate-900 border border-slate-700 rounded-lg px-3 py-2 text-white text-sm"
                    />
                  </div>
                  <div>
                    <label className="block text-sm text-slate-400 mb-1">Período</label>
                    <select
                      value={form.period}
                      onChange={(e) => setForm({ ...form, period: e.target.value })}
                      className="w-full bg-slate-900 border border-slate-700 rounded-lg px-3 py-2 text-white text-sm"
                    >
                      <option value="monthly">Mensal</option>
                      <option value="quarterly">Trimestral</option>
                      <option value="yearly">Anual</option>
                      <option value="custom">Personalizado</option>
                    </select>
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm text-slate-400 mb-1">Início</label>
                    <input
                      required
                      type="date"
                      value={form.start_date}
                      onChange={(e) => setForm({ ...form, start_date: e.target.value })}
                      className="w-full bg-slate-900 border border-slate-700 rounded-lg px-3 py-2 text-white text-sm"
                    />
                  </div>
                  <div>
                    <label className="block text-sm text-slate-400 mb-1">Fim</label>
                    <input
                      required
                      type="date"
                      value={form.end_date}
                      onChange={(e) => setForm({ ...form, end_date: e.target.value })}
                      className="w-full bg-slate-900 border border-slate-700 rounded-lg px-3 py-2 text-white text-sm"
                    />
                  </div>
                </div>
                <div>
                  <label className="block text-sm text-slate-400 mb-1">Account</label>
                  <select
                    required
                    value={form.account_id}
                    onChange={(e) => setForm({ ...form, account_id: e.target.value })}
                    className="w-full bg-slate-900 border border-slate-700 rounded-lg px-3 py-2 text-white text-sm"
                  >
                    <option value="">Selecione uma account</option>
                    {accounts.map((a) => (
                      <option key={a.id} value={a.account_id}>
                        {a.account_name} ({a.account_id})
                      </option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-sm text-slate-400 mb-1">Threshold de Alerta ({(form.alert_threshold * 100).toFixed(0)}%)</label>
                  <input
                    type="range"
                    min={0.1}
                    max={1}
                    step={0.05}
                    value={form.alert_threshold}
                    onChange={(e) => setForm({ ...form, alert_threshold: Number(e.target.value) })}
                    className="w-full"
                  />
                </div>
                <div className="flex justify-end gap-3 pt-2">
                  <button
                    type="button"
                    onClick={() => setIsOpen(false)}
                    className="px-4 py-2 rounded-lg border border-slate-600 text-slate-300 text-sm hover:bg-slate-700"
                  >
                    Cancelar
                  </button>
                  <button
                    type="submit"
                    className="px-4 py-2 rounded-lg bg-sky-600 hover:bg-sky-500 text-white text-sm font-medium"
                  >
                    Salvar
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}
      </div>
    </MainLayout>
  );
}
