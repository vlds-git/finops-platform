'use client';

import { useState } from 'react';
import { MainLayout } from '@/components/layout/MainLayout';
import { ForecastChart } from '@/components/charts/ForecastChart';
import { useForecast } from '@/hooks/useForecast';
import type { ForecastPoint } from '@/types';

const mockHistorical = Array.from({ length: 60 }, (_, i) => ({
  date: new Date(Date.now() - (60 - i) * 24 * 60 * 60 * 1000).toISOString().split('T')[0],
  value: 5000 + i * 30 + Math.sin(i / 5) * 400 + Math.random() * 200,
}));

const mockForecast30: ForecastPoint[] = Array.from({ length: 30 }, (_, i) => ({
  date: new Date(Date.now() + i * 24 * 60 * 60 * 1000).toISOString().split('T')[0],
  value: 6800 + i * 25 + Math.random() * 150,
  lower: (6800 + i * 25) * 0.85,
  upper: (6800 + i * 25) * 1.15,
}));

const mockForecast90: ForecastPoint[] = Array.from({ length: 90 }, (_, i) => ({
  date: new Date(Date.now() + i * 24 * 60 * 60 * 1000).toISOString().split('T')[0],
  value: 6800 + i * 20 + Math.random() * 200,
  lower: (6800 + i * 20) * 0.80,
  upper: (6800 + i * 20) * 1.20,
}));

const mockForecast365: ForecastPoint[] = Array.from({ length: 365 }, (_, i) => ({
  date: new Date(Date.now() + i * 24 * 60 * 60 * 1000).toISOString().split('T')[0],
  value: 6800 + i * 15 + Math.sin(i / 30) * 500 + Math.random() * 300,
  lower: (6800 + i * 15) * 0.75,
  upper: (6800 + i * 15) * 1.25,
}));

export default function ForecastPage() {
  const [period, setPeriod] = useState<'30d' | '90d' | '12m'>('30d');
  const { data: forecast } = useForecast('huawei', 'hw-001', period);

  const forecastData = {
    '30d': mockForecast30,
    '90d': mockForecast90,
    '12m': mockForecast365,
  }[period];

  const metrics = {
    '30d': { model: 'Ensemble', mape: '4.2%', rmse: 'R$ 1.240', confidence: '85%' },
    '90d': { model: 'Ensemble', mape: '7.8%', rmse: 'R$ 2.850', confidence: '78%' },
    '12m': { model: 'Ensemble', mape: '14.5%', rmse: 'R$ 5.600', confidence: '65%' },
  }[period];

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
            <ForecastChart historical={mockHistorical} forecast={forecastData} className="h-[400px]" />
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
              <h4 className="text-sm font-semibold text-white mb-3">Previsão por Modelo</h4>
              <div className="space-y-2">
                <div className="flex justify-between items-center">
                  <span className="text-slate-400 text-sm">Prophet</span>
                  <span className="text-sky-400 font-medium">R$ 192.400</span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-slate-400 text-sm">ARIMA</span>
                  <span className="text-purple-400 font-medium">R$ 198.100</span>
                </div>
                <div className="flex justify-between items-center">
                  <span className="text-slate-400 text-sm">Holt-Winters</span>
                  <span className="text-orange-400 font-medium">R$ 194.800</span>
                </div>
                <div className="border-t border-slate-700 pt-2 flex justify-between items-center">
                  <span className="text-white font-medium">Ensemble</span>
                  <span className="text-emerald-400 font-bold">R$ 195.000</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </MainLayout>
  );
}
