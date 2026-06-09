'use client';

import { cn } from '@/lib/utils';

interface Column<T> {
  key: string;
  header: string;
  render?: (item: T) => React.ReactNode;
  className?: string;
}

interface DataTableProps<T> {
  columns: Column<T>[];
  data: T[];
  className?: string;
}

export function DataTable<T extends Record<string, any>>({ columns, data, className }: DataTableProps<T>) {
  return (
    <div className={cn("card overflow-hidden", className)}>
      <table className="w-full text-sm text-left">
        <thead className="bg-slate-800 text-slate-400">
          <tr>
            {columns.map((col) => (
              <th key={col.key} className="px-6 py-3 font-medium">
                {col.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-700">
          {data.map((item, idx) => (
            <tr key={idx} className="hover:bg-slate-800/50 transition-colors">
              {columns.map((col) => (
                <td key={col.key} className={cn("px-6 py-4", col.className)}>
                  {col.render ? col.render(item) : item[col.key]}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
