'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';
import { cn } from '@/lib/utils';
import {
  LayoutDashboard,
  BarChart3,
  TrendingUp,
  AlertTriangle,
  Zap,
  Wallet,
  Bell,
  Users,
  Cloud,
  LogOut,
} from 'lucide-react';

const navItems = [
  { href: '/dashboard', label: 'Dashboard Executivo', icon: LayoutDashboard },
  { href: '/operational', label: 'Dashboard Operacional', icon: BarChart3 },
  { href: '/forecast', label: 'Forecast', icon: TrendingUp },
  { href: '/anomalies', label: 'Anomalias', icon: AlertTriangle },
  { href: '/recommendations', label: 'Recomendações', icon: Zap },
  { href: '/budgets', label: 'Budgets', icon: Wallet },
  { href: '/alerts', label: 'Alertas', icon: Bell },
  { href: '/users', label: 'Usuários', icon: Users },
  { href: '/multicloud', label: 'Multi-Cloud', icon: Cloud },
];

export function Sidebar() {
  const pathname = usePathname();

  return (
    <aside className="w-64 bg-slate-850 border-r border-slate-700 flex flex-col h-full">
      <div className="p-6 border-b border-slate-700">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 bg-sky-500 rounded-lg flex items-center justify-center">
            <BarChart3 className="w-5 h-5 text-white" />
          </div>
          <span className="font-bold text-lg text-white">FinOps Enterprise</span>
        </div>
      </div>

      <nav className="flex-1 p-4 space-y-1 overflow-y-auto">
        {navItems.map((item) => {
          const Icon = item.icon;
          const isActive = pathname === item.href || pathname.startsWith(item.href + '/');
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                'nav-item',
                isActive && 'active'
              )}
            >
              <Icon className="w-5 h-5" />
              <span>{item.label}</span>
            </Link>
          );
        })}
      </nav>

      <div className="p-4 border-t border-slate-700">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 rounded-full bg-sky-500 flex items-center justify-center text-sm font-bold text-white">
            AD
          </div>
          <div>
            <p className="text-sm font-medium text-white">Admin User</p>
            <p className="text-xs text-slate-400">admin@finops.local</p>
          </div>
        </div>
        <button className="mt-3 flex items-center gap-2 text-sm text-slate-400 hover:text-slate-200 transition-colors">
          <LogOut className="w-4 h-4" />
          Sair
        </button>
      </div>
    </aside>
  );
}
