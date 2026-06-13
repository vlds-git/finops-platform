'use client';

import { useState, useEffect, useMemo } from 'react';
import { MainLayout } from '@/components/layout/MainLayout';
import { ForecastChart } from '@/components/charts/ForecastChart';
import { useForecast } from '@/hooks/useForecast';
import { useCostTrends } from '@/hooks/useCosts';
import { useCurrency } from '@/contexts/CurrencyContext';
import { formatCurrency } from '@/lib/utils';
import type { PeriodOption, PeriodRange } from '@/components/ui/FilterBar';

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

export default function ForecastPage() {
  const { currency } = useCurrency();
  const [forecastDays, setForecastDays] = useState(30);
  const [range, setRange] = useState<PeriodRange>(() => getRangeForOption('30d'));

  useEffect(() => {
    setRange(getRangeForOption('30d'));
  }, []);

  const trendDays = useMemo(() => {
    const start = new Date(range.startDate);
    const end = new Date(range.endDate);
    return Math.max(1, Math.round((end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24)));
  }, [range]);

  const { data: forecast } = useForecast(range.startDate, range.endDate, 'huawei', undefined, forecastDays);
  const { data: trends } = useCostTrends(String(trendDays));

  const historical = (trends || []).map((p) => ({ date: p.date, value: p.cost }));
  const forecastData = forecast?.forecast || [];

  const metrics = {
    model: forecast?.model || 'Linear Trend',
    mape: 'N/A',
    rmse: 'N/A',
    confidence: `${((forecast?.confidence || 0.85) * 100).toFixed(0)}%`,
  };

  return (
    <MainLayout title="Forecast" onPeriodChange={(_, newRange) => newRange && setRange(newRange)}>
      <div className="space-y-6">
        <div className="flex items-center gap-4">
          <span className="text-sm text-slate-400">Horizonte:</span>
          {[30, 60, 90].map((d) => (
            <button
              key={d}
              onClick={() => setForecastDays(d)}
              className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                forecastDays === d ? 'bg-sky-600 text-white' : 'bg-slate-800 text-slate-300 hover:bg-slate-700'
              }`}
            >
              {d} Dias
            </button>
          ))}
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="card p-5 lg:col-span-2">
            <h3 className="text-sm font-semibold text-white mb-4">Forecast - Tendência Linear</h3>
            <ForecastChart historical={historical} forecast={forecastData} className="h-[400px]" />
          </div>

          <div className="space-y-4">
            <div className="card p-5">
              <h4 className="text-sm font-semibold text-white mb-3">Métricas do Forecast</h4>
              <div className="space-y-3">
                <div className="flex justify-between">
                  <span className="text-slate-400">Modelo</span>
                  <span className="text-white font-medium">{metrics.model}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-400">MAPE</span>
                  <span className="text-emerald-400">{metrics.mape}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-400">RMSE</span>
                  <span className="text-white font-medium">{metrics.rmse}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-slate-400">Confiança</span>
                  <span className="text-emerald-400">{metrics.confidence}</span>
                </div>
              </div>
            </div>

            <div className="card p-5">
              <h4 className="text-sm font-semibold text-white mb-3">Previsão Total</h4>
              <p className="text-3xl font-bold text-emerald-400">
                {formatCurrency(forecast?.total_forecast || 0, currency)}
              </p>
              <p className="text-sm text-slate-400 mt-1">Próximos {forecastDays} dias</p>
            </div>
          </div>
        </div>
      </div>
    </MainLayout>
  );
}
