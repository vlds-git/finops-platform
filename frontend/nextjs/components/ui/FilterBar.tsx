'use client';

import { useState } from 'react';
import { Calendar, Cloud } from 'lucide-react';

interface FilterBarProps {
  onPeriodChange?: (period: string) => void;
  onProviderChange?: (provider: string) => void;
}

export function FilterBar({ onPeriodChange, onProviderChange }: FilterBarProps) {
  const [period, setPeriod] = useState('30');
  const [provider, setProvider] = useState('all');

  return (
    <div className="flex items-center gap-3">
      <div className="flex items-center gap-2 bg-slate-800 rounded-lg px-3 py-2 border border-slate-700">
        <Calendar className="w-4 h-4 text-slate-400" />
        <select 
          className="bg-transparent text-sm text-white outline-none cursor-pointer"
          value={period}
          onChange={(e) => {
            setPeriod(e.target.value);
            onPeriodChange?.(e.target.value);
          }}
        >
          <option value="7">Últimos 7 dias</option>
          <option value="30">Últimos 30 dias</option>
          <option value="90">Últimos 90 dias</option>
          <option value="365">Último ano</option>
        </select>
      </div>

      <div className="flex items-center gap-2 bg-slate-800 rounded-lg px-3 py-2 border border-slate-700">
        <Cloud className="w-4 h-4 text-slate-400" />
        <select 
          className="bg-transparent text-sm text-white outline-none cursor-pointer"
          value={provider}
          onChange={(e) => {
            setProvider(e.target.value);
            onProviderChange?.(e.target.value);
          }}
        >
          <option value="all">Todos</option>
          <option value="huawei">Huawei Cloud</option>
          <option value="azure">Azure</option>
          <option value="aws">AWS</option>
        </select>
      </div>
    </div>
  );
}
