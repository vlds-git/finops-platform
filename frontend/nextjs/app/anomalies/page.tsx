'use client';

import { MainLayout } from '@/components/layout/MainLayout';
import { AnomalyChart } from '@/components/charts/AnomalyChart';
import { DataTable } from '@/components/ui/DataTable';
import { Badge } from '@/components/ui/Badge';
import { useAnomalies } from '@/hooks/useAnomalies';
import type { AnomalyPoint } from '@/types';
import { formatCurrency } from '@/lib/utils';

const mockDates = Array.from({ length: 90 }, (_, i) =>
  new Date(Date.now() - (90 - i) * 24 * 60 * 60 * 1000).toISOString().split('T')[0]
);

const mockData = mockDates.map((_, i) => {
  let val = 5000 + i * 20 + Math.sin(i / 7) * 300 + Math.random() * 150;
  if (i === 20 || i === 45 || i === 70) val *= 2.5;
  return val;
});

const mockAnomalies: AnomalyPoint[] = [
  { date: mockDates[20], value: 12500, expected: 5200, deviation: 7300, severity: 'critical', score: 0.95, method: 'Ensemble' },
  { date: mockDates[45], value: 8900, expected: 5400, deviation: 3500, severity: 'high', score: 0.82, method: 'Isolation Forest' },
  { date: mockDates[70], value: 7200, expected: 5600, deviation: 1600, severity: 'medium', score: 0.65, method: 'Z-Score' },
];

export default function AnomaliesPage() {
  const { data: anomalies } = useAnomalies('huawei', 'hw-001');

  const columns = [
    { key: 'date', header: 'Data' },
    { key: 'value', header: 'Valor', render: (a: AnomalyPoint) => <span className="text-red-400 font-medium">{formatCurrency(a.value)}</span> },
    { key: 'expected', header: 'Esperado', render: (a: AnomalyPoint) => <span className="text-slate-400">{formatCurrency(a.expected)}</span> },
    { key: 'deviation', header: 'Desvio', render: (a: AnomalyPoint) => <span className="text-red-400">+{((a.deviation / a.expected) * 100).toFixed(0)}%</span> },
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
            <AnomalyChart data={mockData} dates={mockDates} anomalies={mockAnomalies} className="h-[400px]" />
          </div>

          <div className="space-y-4">
            <div className="card p-5">
              <h4 className="text-sm font-semibold text-white mb-3">Resumo</h4>
              <div className="space-y-3">
                <div className="flex justify-between"><span className="text-slate-400">Total Anomalias</span><span className="text-white font-medium">3</span></div>
                <div className="flex justify-between"><span className="text-slate-400">Impacto Total</span><span className="text-red-400 font-medium">R$ 12.450</span></div>
                <div className="flex justify-between"><span className="text-slate-400">Baseline Média</span><span className="text-white font-medium">R$ 5.200</span></div>
                <div className="flex justify-between"><span className="text-slate-400">Desvio Padrão</span><span className="text-white font-medium">R$ 890</span></div>
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

        <DataTable columns={columns} data={mockAnomalies} />
      </div>
    </MainLayout>
  );
}
