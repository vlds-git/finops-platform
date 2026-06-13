'use client';

import { cn } from '@/lib/utils';
import { Loader2 } from 'lucide-react';

interface Column<T> {
  key: keyof T | string;
  header: string;
  render?: (item: T) => React.ReactNode;
  className?: string;
}

interface DataTableProps<T> {
  columns: Column<T>[];
  data: T[];
  className?: string;
  isLoading?: boolean;
}

export function DataTable<T>({ columns, data, className, isLoading }: DataTableProps<T>) {
  const getValue = (item: T, key: keyof T | string): string => {
    if (item && typeof item === 'object') {
      const value = (item as Record<string, unknown>)[key as string];
      return value !== undefined && value !== null ? String(value) : '';
    }
    return '';
  };

  return (
    <div className={cn("card overflow-hidden", className)}>
      <table className="w-full text-sm text-left">
        <thead className="bg-slate-800 text-slate-400">
          <tr>
            {columns.map((col) => (
              <th key={String(col.key)} className="px-6 py-3 font-medium">
                {col.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-700">
          {isLoading ? (
            <tr>
              <td colSpan={columns.length} className="px-6 py-12 text-center text-slate-400">
                <Loader2 className="w-6 h-6 animate-spin mx-auto mb-2" />
                Carregando...
              </td>
            </tr>
          ) : data.length === 0 ? (
            <tr>
              <td colSpan={columns.length} className="px-6 py-12 text-center text-slate-400">
                Nenhum registro encontrado.
              </td>
            </tr>
          ) : (
            data.map((item, idx) => (
              <tr key={idx} className="hover:bg-slate-800/50 transition-colors">
                {columns.map((col) => (
                  <td key={String(col.key)} className={cn("px-6 py-4", col.className)}>
                    {col.render ? col.render(item) : getValue(item, col.key)}
                  </td>
                ))}
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  );
}
