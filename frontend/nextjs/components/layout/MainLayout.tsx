'use client';

import { Sidebar } from './Sidebar';
import { Header } from './Header';
import type { PeriodOption, PeriodRange } from '@/components/ui/FilterBar';

interface MainLayoutProps {
  children: React.ReactNode;
  title: string;
  showFilters?: boolean;
  onPeriodChange?: (option: PeriodOption, range?: PeriodRange) => void;
}

export function MainLayout({ children, title, showFilters = true, onPeriodChange }: MainLayoutProps) {
  return (
    <div className="h-screen flex overflow-hidden">
      <Sidebar />
      <main className="flex-1 flex flex-col overflow-hidden">
        <Header title={title} showFilters={showFilters} onPeriodChange={onPeriodChange} />
        <div className="flex-1 overflow-y-auto p-6">
          {children}
        </div>
      </main>
    </div>
  );
}
