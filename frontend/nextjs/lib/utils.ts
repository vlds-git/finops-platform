import { type ClassValue, clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function formatCurrency(value: number, currency = 'BRL'): string {
  return new Intl.NumberFormat('pt-BR', {
    style: 'currency',
    currency,
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(value);
}

export function formatPercentage(value: number): string {
  return `${(value * 100).toFixed(1)}%`;
}

export function formatNumber(value: number): string {
  return new Intl.NumberFormat('pt-BR').format(value);
}

export function getStatusColor(status: string): string {
  switch (status) {
    case 'good':
    case 'resolved':
      return 'text-emerald-400';
    case 'warning':
    case 'firing':
      return 'text-amber-400';
    case 'danger':
    case 'critical':
      return 'text-red-400';
    default:
      return 'text-slate-400';
  }
}

export function getSeverityBadge(severity: string): string {
  switch (severity) {
    case 'critical':
      return 'badge-danger';
    case 'high':
      return 'badge-warning';
    case 'medium':
      return 'badge-info';
    case 'low':
      return 'badge-success';
    default:
      return 'badge-info';
  }
}
