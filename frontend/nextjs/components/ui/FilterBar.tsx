'use client';

import { useState } from 'react';
import { Calendar } from 'lucide-react';

export type PeriodOption = '24h' | '48h' | '7d' | '30d' | '90d' | 'custom';

export interface PeriodRange {
  startDate: string;
  endDate: string;
}

interface FilterBarProps {
  onPeriodChange?: (option: PeriodOption, range?: PeriodRange) => void;
}

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
    case '30d':
      start.setDate(end.getDate() - 30);
      break;
    case '90d':
      start.setDate(end.getDate() - 90);
      break;
    default:
      start.setDate(end.getDate() - 30);
  }
  return { startDate: formatDateInput(start), endDate: formatDateInput(end) };
}

const periodLabels: Record<PeriodOption, string> = {
  '24h': 'Últimas 24h',
  '48h': 'Últimas 48h',
  '7d': 'Últimos 7 dias',
  '30d': 'Últimos 30 dias',
  '90d': 'Últimos 90 dias',
  custom: 'Personalizado',
};

export function FilterBar({ onPeriodChange }: FilterBarProps) {
  const [period, setPeriod] = useState<PeriodOption>('30d');
  const [customRange, setCustomRange] = useState<PeriodRange>(() => getRangeForOption('30d'));
  const [isOpen, setIsOpen] = useState(false);

  const handleOptionChange = (option: PeriodOption) => {
    setPeriod(option);
    setIsOpen(false);
    if (option === 'custom') {
      onPeriodChange?.(option, customRange);
    } else {
      const range = getRangeForOption(option);
      setCustomRange(range);
      onPeriodChange?.(option, range);
    }
  };

  const handleCustomChange = (field: keyof PeriodRange, value: string) => {
    const next = { ...customRange, [field]: value };
    setCustomRange(next);
    if (period === 'custom') {
      onPeriodChange?.('custom', next);
    }
  };

  return (
    <div className="flex items-center gap-3">
      <div className="relative">
        <div className="flex items-center gap-2 bg-slate-800 rounded-lg px-3 py-2 border border-slate-700">
          <Calendar className="w-4 h-4 text-slate-400" />
          <button
            type="button"
            onClick={() => setIsOpen(!isOpen)}
            className="bg-transparent text-sm text-white outline-none text-left min-w-[140px]"
          >
            {periodLabels[period]}
          </button>
        </div>
        {isOpen && (
          <div className="absolute top-full left-0 mt-1 w-48 bg-slate-800 border border-slate-700 rounded-lg shadow-lg z-50 py-1">
            {(Object.keys(periodLabels) as PeriodOption[]).map((option) => (
              <button
                key={option}
                type="button"
                onClick={() => handleOptionChange(option)}
                className={`w-full text-left px-3 py-2 text-sm hover:bg-slate-700 ${
                  period === option ? 'text-sky-400' : 'text-white'
                }`}
              >
                {periodLabels[option]}
              </button>
            ))}
          </div>
        )}
      </div>

      {period === 'custom' && (
        <div className="flex items-center gap-2">
          <input
            type="date"
            value={customRange.startDate}
            onChange={(e) => handleCustomChange('startDate', e.target.value)}
            className="bg-slate-800 border border-slate-700 text-white text-sm rounded-lg px-2 py-2 outline-none"
          />
          <span className="text-slate-400 text-sm">até</span>
          <input
            type="date"
            value={customRange.endDate}
            onChange={(e) => handleCustomChange('endDate', e.target.value)}
            className="bg-slate-800 border border-slate-700 text-white text-sm rounded-lg px-2 py-2 outline-none"
          />
        </div>
      )}
    </div>
  );
}
