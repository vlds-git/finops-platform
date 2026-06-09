'use client';

import { Search, Bell } from 'lucide-react';
import { FilterBar } from '@/components/ui/FilterBar';

interface HeaderProps {
  title: string;
  showFilters?: boolean;
  onPeriodChange?: (period: string) => void;
  onProviderChange?: (provider: string) => void;
}

export function Header({ title, showFilters = true, onPeriodChange, onProviderChange }: HeaderProps) {
  return (
    <header className="bg-slate-800 border-b border-slate-700 px-6 py-4 flex items-center justify-between">
      <div className="flex items-center gap-4">
        <h1 className="text-xl font-semibold text-white">{title}</h1>
        {showFilters && (
          <FilterBar onPeriodChange={onPeriodChange} onProviderChange={onProviderChange} />
        )}
      </div>
      <div className="flex items-center gap-3">
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
