'use client';

import { MainLayout } from '@/components/layout/MainLayout';
import { AnomalyChart } from '@/components/charts/AnomalyChart';
import { DataTable } from '@/components/ui/DataTable';
import { Badge } from '@/components/ui/Badge';
import { useAnomalies } from '@/hooks/useAnomalies';
import { useCurrency } from '@/contexts/CurrencyContext';
import type { AnomalyPoint } from '@/types';
import { formatCurrency } from '@/lib/utils';

export default function AnomaliesPage() {
  const { currency } = useCurrency();
  const { data: anomalies } = useAnomalies('huawei', 'hw-001');

  const anomalyList = anomalies?.anomalies || [];
  const dates = anomalyList.map((a) => a.date);
  const values = anomalyList.map((a) => a.value);

  const columns = [
    { key: 'date', header: 'Data' },
    { key: 'value', header: 'Valor', render: (a: AnomalyPoint) => <span className="text-red-400 font-medium">{formatCurrency(a.value, currency)}</span> },
    { key: 'expected', header: 'Esperado', render: (a: AnomalyPoint) => <span className="text-slate-400">{formatCurrency(a.expected, currency)}</span> },
    { key: 'deviation', header: 'Desvio', render: (a: AnomalyPoint) => <span className="text-red-400">+{a.expected > 0 ? ((a.deviation / a.expected) * 100).toFixed(0) : 0}%</span> },
    { key: 'severity', header: 'Severidade', render: (a: AnomalyPoint) => (
      <Badge variant={a.severity === 'critical' ? 'danger' : a.severity === 'high' ? 'warning' : 'info'}>
        {a.severity}
      </Badge>
    )},
    { key: 'method', header: 'Método' },
    { key: 'action', header: 'Ação', render: () => <button className="text-sky-400 hover:text-sky-300 text-sm">Investigar</button> },
  ];

  return (
    <MainLayout title="Anomalias">
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
              <h4 className="text-sm font-semibold text-white mb-3">Métodos Ativos</h4>
              <div className="flex gap-2 flex-wrap">
                <Badge variant="success">Isolation Forest</Badge>
                <Badge variant="success">Z-Score</Badge>
                <Badge variant="success">Rolling Average</Badge>
              </div>
            </div>
          </div>
        </div>

        <DataTable columns={columns} data={anomalyList} />
      </div>
    </MainLayout>
  );
}
