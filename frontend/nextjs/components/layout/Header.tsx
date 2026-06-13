'use client';

import { Search, Bell, DollarSign } from 'lucide-react';
import { FilterBar, type PeriodOption, type PeriodRange } from '@/components/ui/FilterBar';
import { useCurrency } from '@/contexts/CurrencyContext';

interface HeaderProps {
  title: string;
  showFilters?: boolean;
  onPeriodChange?: (option: PeriodOption, range?: PeriodRange) => void;
}

export function Header({ title, showFilters = true, onPeriodChange }: HeaderProps) {
  const { currency, setCurrency } = useCurrency();

  return (
    <header className="bg-slate-800 border-b border-slate-700 px-6 py-4 flex items-center justify-between">
      <div className="flex items-center gap-4">
        <h1 className="text-xl font-semibold text-white">{title}</h1>
        {showFilters && <FilterBar onPeriodChange={onPeriodChange} />}
      </div>
      <div className="flex items-center gap-3">
        <button
          onClick={() => setCurrency(currency === 'USD' ? 'BRL' : 'USD')}
          className="flex items-center gap-2 px-3 py-1.5 rounded-lg bg-slate-700 hover:bg-slate-600 text-slate-200 text-sm transition-colors"
          title="Alternar moeda"
        >
          <DollarSign className="w-4 h-4" />
          <span>{currency}</span>
        </button>
        <button className="p-2 rounded-lg hover:bg-slate-700 text-slate-400 transition-colors">
          <Search className="w-5 h-5" />
        </button>
        <button className="p-2 rounded-lg hover:bg-slate-700 text-slate-400 transition-colors relative">
          <Bell className="w-5 h-5" />
          <span className="absolute top-1 right-1 w-2 h-2 bg-red-500 rounded-full" />
        </button>
      </div>
    </header>
  );
}
