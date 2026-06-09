'use client';

import { MainLayout } from '@/components/layout/MainLayout';
import { Badge } from '@/components/ui/Badge';
import { useAlerts } from '@/hooks/useAlerts';
import type { Alert } from '@/types';
import { formatCurrency } from '@/lib/utils';
import { Plus, CheckCircle } from 'lucide-react';

const mockAlerts: Alert[] = [
  {
    id: '1', rule_id: 'r1', rule_name: 'Budget Threshold Exceeded',
    severity: 'critical', message: 'Staging budget at 93% of limit. Threshold: 90%',
    value: 93, threshold: 90, status: 'firing', created_at: new Date().toISOString(),
  },
  {
    id: '2', rule_id: 'r2', rule_name: 'Cost Anomaly Detected',
    severity: 'high', message: 'Compute cost spike detected: R$ 12,500 (expected: R$ 5,200)',
    value: 12500, threshold: 5200, status: 'firing', created_at: new Date().toISOString(),
  },
  {
    id: '3', rule_id: 'r3', rule_name: 'Forecast Alert',
    severity: 'low', message: '30-day forecast exceeds budget by 5%',
    value: 105, threshold: 100, status: 'firing', created_at: new Date().toISOString(),
  },
];

export default function AlertsPage() {
  const { data: alerts } = useAlerts('firing');

  return (
    <MainLayout title="Alertas">
      <div className="space-y-6">
        <div className="flex justify-between items-center">
          <div className="flex gap-2">
            <button className="px-3 py-1.5 bg-sky-600 text-white rounded-lg text-sm">Todas</button>
            <button className="px-3 py-1.5 bg-slate-800 text-slate-300 rounded-lg text-sm hover:bg-slate-700">Firing</button>
            <button className="px-3 py-1.5 bg-slate-800 text-slate-300 rounded-lg text-sm hover:bg-slate-700">Resolved</button>
          </div>
          <button className="px-4 py-2 bg-sky-600 hover:bg-sky-500 text-white rounded-lg text-sm font-medium transition-colors flex items-center gap-2">
            <Plus className="w-4 h-4" />
            Nova Regra
          </button>
        </div>

        <div className="space-y-3">
          {mockAlerts.map((alert) => (
            <div key={alert.id} className="card p-4 flex items-center justify-between">
              <div className="flex items-center gap-4">
                <div className={`w-2 h-2 rounded-full ${
                  alert.status === 'firing' ? 'bg-red-500 animate-pulse' : 'bg-emerald-500'
                }`} />
                <div>
                  <h4 className="text-sm font-medium text-white">{alert.rule_name}</h4>
                  <p className="text-xs text-slate-400">{alert.message}</p>
                </div>
              </div>
              <div className="flex items-center gap-3">
                <Badge variant={
                  alert.severity === 'critical' ? 'danger' : 
                  alert.severity === 'high' ? 'warning' : 'info'
                }>
                  {alert.severity}
                </Badge>
                <span className="text-xs text-slate-400">2 min atrás</span>
                <button className="text-sky-400 hover:text-sky-300 text-sm flex items-center gap-1">
                  <CheckCircle className="w-4 h-4" />
                  Resolver
                </button>
              </div>
            </div>
          ))}
        </div>
      </div>
    </MainLayout>
  );
}
