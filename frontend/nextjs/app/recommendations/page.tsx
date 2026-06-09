'use client';

import { MainLayout } from '@/components/layout/MainLayout';
import { Badge } from '@/components/ui/Badge';
import { useRecommendations, useApplyRecommendation } from '@/hooks/useRecommendations';
import type { RecommendationItem } from '@/types';
import { formatCurrency } from '@/lib/utils';
import { Zap, CheckCircle, XCircle, Info } from 'lucide-react';

const mockRecommendations: RecommendationItem[] = [
  {
    id: 'rec-001',
    category: 'rightsizing',
    title: 'Oversized Compute Instance',
    description: 'VM c6.2xlarge running at 15% CPU average. Recommend downgrade to c6.large.',
    resource_id: 'vm-prod-001',
    resource_type: 'Virtual Machine',
    service: 'Compute',
    region: 'sa-brazil-1',
    current_cost: 350,
    projected_cost: 175,
    savings: 175,
    savings_percentage: 50,
    confidence: 0.92,
    priority: 'high',
    justification: 'CPU utilization below 20% for 30 consecutive days. Memory utilization below 40%.',
    action: 'Resize instance from c6.2xlarge to c6.large',
    risk: 'Low - can be rolled back within 5 minutes',
    implementation: 'Stop instance, change type, start instance. Downtime: ~2 minutes.',
    created_at: new Date().toISOString(),
  },
  {
    id: 'rec-002',
    category: 'savings',
    title: 'Reserved Instance Opportunity',
    description: 'Stable compute workload for 6 months. RI would save 40%.',
    resource_id: 'ri-compute-001',
    resource_type: 'Reserved Instance',
    service: 'Compute',
    region: 'sa-brazil-1',
    current_cost: 5000,
    projected_cost: 3000,
    savings: 2000,
    savings_percentage: 40,
    confidence: 0.93,
    priority: 'high',
    justification: 'Workload pattern stable (CV < 5%). No planned architecture changes.',
    action: 'Purchase 1-year All Upfront RI for c6 family',
    risk: 'Low - workload is production-critical and stable',
    implementation: 'Purchase via AWS Console or API. Immediate billing benefit.',
    created_at: new Date().toISOString(),
  },
  {
    id: 'rec-003',
    category: 'idle',
    title: 'Idle Load Balancer',
    description: 'ALB with zero active connections for 45 days.',
    resource_id: 'alb-staging-001',
    resource_type: 'Application Load Balancer',
    service: 'Network',
    region: 'sa-brazil-1',
    current_cost: 45,
    projected_cost: 0,
    savings: 45,
    savings_percentage: 100,
    confidence: 0.95,
    priority: 'medium',
    justification: 'Zero requests processed. No target groups registered.',
    action: 'Delete load balancer',
    risk: 'None - confirmed unused via CloudWatch metrics',
    implementation: 'Remove via console or Terraform. Immediate effect.',
    created_at: new Date().toISOString(),
  },
];

export default function RecommendationsPage() {
  const { data: recommendations } = useRecommendations('huawei', 'hw-001');
  const applyMutation = useApplyRecommendation();

  const totalSavings = mockRecommendations.reduce((sum, r) => sum + r.savings, 0);
  const highPriority = mockRecommendations.filter(r => r.priority === 'high').length;

  return (
    <MainLayout title="Recomendações">
      <div className="space-y-6">
        {/* Summary Cards */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div className="card p-4 text-center">
            <p className="text-3xl font-bold text-emerald-400">{formatCurrency(totalSavings)}</p>
            <p className="text-sm text-slate-400 mt-1">Economia Total Potencial</p>
          </div>
          <div className="card p-4 text-center">
            <p className="text-3xl font-bold text-sky-400">{mockRecommendations.length}</p>
            <p className="text-sm text-slate-400 mt-1">Oportunidades</p>
          </div>
          <div className="card p-4 text-center">
            <p className="text-3xl font-bold text-amber-400">{highPriority}</p>
            <p className="text-sm text-slate-400 mt-1">Alta Prioridade</p>
          </div>
          <div className="card p-4 text-center">
            <p className="text-3xl font-bold text-purple-400">
              {(mockRecommendations.reduce((s, r) => s + r.confidence, 0) / mockRecommendations.length * 100).toFixed(0)}%
            </p>
            <p className="text-sm text-slate-400 mt-1">Confiança Média</p>
          </div>
        </div>

        {/* Recommendation Cards */}
        <div className="space-y-4">
          {mockRecommendations.map((rec) => (
            <div key={rec.id} className="card p-5">
              <div className="flex justify-between items-start mb-3">
                <div>
                  <div className="flex items-center gap-2 mb-1">
                    <Badge variant={rec.category === 'rightsizing' ? 'warning' : rec.category === 'savings' ? 'warning' : 'info'}>
                      {rec.category === 'rightsizing' ? 'Rightsizing' : rec.category === 'savings' ? 'Savings' : 'Idle'}
                    </Badge>
                    <Badge variant={rec.priority === 'high' ? 'danger' : 'warning'}>
                      {rec.priority === 'high' ? 'Alta Prioridade' : 'Média Prioridade'}
                    </Badge>
                  </div>
                  <h3 className="text-lg font-semibold text-white">{rec.title}</h3>
                  <p className="text-sm text-slate-400 mt-1">{rec.description}</p>
                </div>
                <div className="text-right">
                  <p className="text-2xl font-bold text-emerald-400">{formatCurrency(rec.savings)}/mês</p>
                  <p className="text-sm text-slate-400">{rec.savings_percentage}% economia</p>
                </div>
              </div>

              <div className="bg-slate-800 rounded-lg p-3 mt-3 text-sm space-y-1">
                <p className="text-slate-300"><strong>Justificativa:</strong> {rec.justification}</p>
                <p className="text-slate-300"><strong>Ação:</strong> {rec.action}</p>
                <p className="text-slate-300"><strong>Risco:</strong> {rec.risk}</p>
              </div>

              <div className="flex gap-2 mt-3">
                <button 
                  onClick={() => applyMutation.mutate(rec.id)}
                  className="px-4 py-2 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg text-sm font-medium transition-colors flex items-center gap-2"
                >
                  <CheckCircle className="w-4 h-4" />
                  Aplicar
                </button>
                <button className="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-slate-300 rounded-lg text-sm font-medium transition-colors flex items-center gap-2">
                  <XCircle className="w-4 h-4" />
                  Ignorar
                </button>
                <button className="px-4 py-2 bg-slate-700 hover:bg-slate-600 text-slate-300 rounded-lg text-sm font-medium transition-colors flex items-center gap-2">
                  <Info className="w-4 h-4" />
                  Detalhes
                </button>
              </div>
            </div>
          ))}
        </div>
      </div>
    </MainLayout>
  );
}
