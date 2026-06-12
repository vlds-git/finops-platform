'use client';

import { MainLayout } from '@/components/layout/MainLayout';
import { Badge } from '@/components/ui/Badge';
import { useRecommendations, useApplyRecommendation } from '@/hooks/useRecommendations';
import { useCurrency } from '@/contexts/CurrencyContext';
import type { RecommendationItem } from '@/types';
import { formatCurrency } from '@/lib/utils';
import { Zap, CheckCircle, XCircle, Info } from 'lucide-react';

export default function RecommendationsPage() {
  const { currency } = useCurrency();
  const { data: recommendations } = useRecommendations('huawei', 'hw-001');
  const applyMutation = useApplyRecommendation();

  const recs = recommendations?.recommendations || [];
  const totalSavings = recs.reduce((sum, r) => sum + r.savings, 0);
  const highPriority = recs.filter(r => r.priority === 'high').length;
  const avgConfidence = recs.length > 0 ? recs.reduce((s, r) => s + r.confidence, 0) / recs.length : 0;

  return (
    <MainLayout title="Recomendações">
      <div className="space-y-6">
        {/* Summary Cards */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div className="card p-4 text-center">
            <p className="text-3xl font-bold text-emerald-400">{formatCurrency(totalSavings, currency)}</p>
            <p className="text-sm text-slate-400 mt-1">Economia Total Potencial</p>
          </div>
          <div className="card p-4 text-center">
            <p className="text-3xl font-bold text-sky-400">{recs.length}</p>
            <p className="text-sm text-slate-400 mt-1">Oportunidades</p>
          </div>
          <div className="card p-4 text-center">
            <p className="text-3xl font-bold text-amber-400">{highPriority}</p>
            <p className="text-sm text-slate-400 mt-1">Alta Prioridade</p>
          </div>
          <div className="card p-4 text-center">
            <p className="text-3xl font-bold text-purple-400">
              {(avgConfidence * 100).toFixed(0)}%
            </p>
            <p className="text-sm text-slate-400 mt-1">Confiança Média</p>
          </div>
        </div>

        {/* Recommendation Cards */}
        <div className="space-y-4">
          {recs.length === 0 && (
            <div className="card p-6 text-center text-slate-400">
              Nenhuma recomendação disponível no momento.
            </div>
          )}
          {recs.map((rec: RecommendationItem) => (
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
                  <p className="text-2xl font-bold text-emerald-400">{formatCurrency(rec.savings, currency)}/mês</p>
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
