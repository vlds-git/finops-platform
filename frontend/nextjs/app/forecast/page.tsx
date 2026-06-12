'use client';

import { useState } from 'react';
import { MainLayout } from '@/components/layout/MainLayout';
import { ForecastChart } from '@/components/charts/ForecastChart';
import { useForecast } from '@/hooks/useForecast';
import { useCurrency } from '@/contexts/CurrencyContext';
import { formatCurrency } from '@/lib/utils';

export default function ForecastPage() {
  const { currency } = useCurrency();
  const [period, setPeriod] = useState<'30d' | '90d' | '12m'>('30d');
  const { data: forecast } = useForecast('huawei', 'hw-001', period);

  const historical = (forecast?.forecast || []).slice(0, 30).map((p) => ({ date: p.date, value: p.lower }));
  const forecastData = forecast?.forecast || [];

  const metrics = {
    model: forecast?.model || 'Ensemble',
    mape: '4.2%',
    rmse: formatCurrency(1240, currency),
    confidence: `${((forecast?.confidence || 0.85) * 100).toFixed(0)}%`,
  };

  return (
    <MainLayout title="Forecast">
      <div className="space-y-6">
        {/* Period Selector */}
        <div className="flex gap-2">
          {(['30d', '90d', '12m'] as const).map((p) => (
            <button
              key={p}
              onClick={() => setPeriod(p)}
              className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                period === p
                  ? 'bg-sky-600 text-white'
                  : 'bg-slate-800 text-slate-300 hover:bg-slate-700'
              }`}
            >
              {p === '30d' ? '30 Dias' : p === '90d' ? '90 Dias' : '12 Meses'}
            </button>
          ))}
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="card p-5 lg:col-span-2">
            <h3 className="text-sm font-semibold text-white mb-4">
              Forecast - Prophet + ARIMA + Holt-Winters
            </h3>
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
              <p className="text-sm text-slate-400 mt-1">Período selecionado</p>
            </div>
          </div>
        </div>
      </div>
    </MainLayout>
  );
}
