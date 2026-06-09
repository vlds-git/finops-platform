'use client';

import { TrendingUp, TrendingDown, Minus } from 'lucide-react';
import { formatCurrency, formatPercentage } from '@/lib/utils';
import type { KPIData } from '@/types';

interface KPICardProps {
  data: KPIData;
}

export function KPICard({ data }: KPICardProps) {
  const isPositive = data.trend > 0;
  const isNegative = data.trend < 0;
  const TrendIcon = isPositive ? TrendingUp : isNegative ? TrendingDown : Minus;
  const trendColor = isPositive ? 'text-emerald-400' : isNegative ? 'text-red-400' : 'text-slate-400';
  const statusColor = data.status === 'good' ? 'bg-emerald-500' : data.status === 'warning' ? 'bg-amber-500' : 'bg-red-500';

  return (
    <div className="card p-5">
      <div className="flex justify-between items-start mb-3">
        <div>
          <p className="text-sm text-slate-400">{data.name}</p>
          <p className="text-2xl font-bold text-white mt-1">
            {data.unit === 'USD' || data.unit === 'BRL' 
              ? formatCurrency(data.value, data.unit)
              : data.unit === 'ratio' 
                ? formatPercentage(data.value)
                : data.value.toLocaleString('pt-BR')
            }
          </p>
        </div>
        <div className={`flex items-center gap-1 text-sm ${trendColor}`}>
          <TrendIcon className="w-4 h-4" />
          <span>{formatPercentage(Math.abs(data.trend))}</span>
        </div>
      </div>
      <div className="h-1 bg-slate-700 rounded-full overflow-hidden">
        <div 
          className={`h-full ${statusColor} rounded-full transition-all`}
          style={{ width: `${Math.min((data.value / (data.target || data.value * 1.2)) * 100, 100)}%` }}
        />
      </div>
      {data.target && (
        <p className="text-xs text-slate-500 mt-2">
          Meta: {data.unit === 'ratio' ? formatPercentage(data.target) : formatCurrency(data.target)}
        </p>
      )}
    </div>
  );
}
