'use client';

import { useState, useEffect } from 'react';
import { MainLayout } from '@/components/layout/MainLayout';
import { AnomalyChart } from '@/components/charts/AnomalyChart';
import { DataTable } from '@/components/ui/DataTable';
import { Badge } from '@/components/ui/Badge';
import { useAnomalies } from '@/hooks/useAnomalies';
import { useCurrency } from '@/contexts/CurrencyContext';
import type { AnomalyPoint } from '@/types';
import type { PeriodOption, PeriodRange } from '@/components/ui/FilterBar';
import { formatCurrency } from '@/lib/utils';

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
    case '30d':
    default:
      start.setDate(end.getDate() - 30);
  }
  return { startDate: formatDateInput(start), endDate: formatDateInput(end) };
}

export default function AnomaliesPage() {
  const { currency } = useCurrency();
  const [range, setRange] = useState<PeriodRange>(() => getRangeForOption('30d'));

  useEffect(() => {
    setRange(getRangeForOption('30d'));
  }, []);

  const { data: anomalies } = useAnomalies(range.startDate, range.endDate, 'huawei');

  const anomalyList = anomalies?.anomalies || [];
  const dates = anomalyList.map((a) => a.date);
  const values = anomalyList.map((a) => a.value);

  const columns = [
    { key: 'date', header: 'Data' },
    { key: 'value', header: 'Valor', render: (a: AnomalyPoint) => <span className="text-red-400 font-medium">{formatCurrency(a.value, currency)}</span> },
    { key: 'expected', header: 'Esperado', render: (a: AnomalyPoint) => <span className="text-slate-400">{formatCurrency(a.expected, currency)}</span> },
    { key: 'deviation', header: 'Desvio (Z)', render: (a: AnomalyPoint) => <span className="text-red-400">{a.deviation.toFixed(2)}σ</span> },
    { key: 'severity', header: 'Severidade', render: (a: AnomalyPoint) => (
      <Badge variant={a.severity === 'critical' ? 'danger' : a.severity === 'high' ? 'warning' : 'info'}>
        {a.severity}
      </Badge>
    )},
    { key: 'method', header: 'Método' },
    { key: 'action', header: 'Ação', render: () => <button className="text-sky-400 hover:text-sky-300 text-sm">Investigar</button> },
  ];

  return (
    <MainLayout title="Anomalias" onPeriodChange={(_, newRange) => newRange && setRange(newRange)}>
      <div className="space-y-6">
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="card p-5 lg:col-span-2">
            <h3 className="text-sm font-semibold text-white mb-4">Timeline de Anomalias</h3>
            <AnomalyChart data={values} dates={dates} anomalies={anomalyList} className="h-[400px]" />
          </div>

          <div className="space-y-4">
            <div className="card p-5">
              <h4 className="text-sm font-semibold text-white mb-3">Resumo</h4>
              <div className="space-y-3">
                <div className="flex justify-between"><span className="text-slate-400">Total Anomalias</span><span className="text-white font-medium">{anomalies?.total_anomalies || 0}</span></div>
                <div className="flex justify-between"><span className="text-slate-400">Impacto Total</span><span className="text-red-400 font-medium">{formatCurrency(anomalies?.total_impact || 0, currency)}</span></div>
                <div className="flex justify-between"><span className="text-slate-400">Baseline Média</span><span className="text-white font-medium">{formatCurrency(anomalies?.baseline_mean || 0, currency)}</span></div>
                <div className="flex justify-between"><span className="text-slate-400">Desvio Padrão</span><span className="text-white font-medium">{formatCurrency(anomalies?.baseline_std || 0, currency)}</span></div>
              </div>
            </div>

            <div className="card p-5">
              <h4 className="text-sm font-semibold text-white mb-3">Método Ativo</h4>
              <div className="flex gap-2 flex-wrap">
                <Badge variant="success">Z-Score</Badge>
              </div>
            </div>
          </div>
        </div>

        <DataTable columns={columns} data={anomalyList} />
      </div>
    </MainLayout>
  );
}
