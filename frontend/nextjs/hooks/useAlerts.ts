'use client';

import { useQuery } from 'react-query';
import api from '@/lib/api';
import type { Alert, AlertRule } from '@/types';

export function useAlerts(status = 'firing') {
  return useQuery<Alert[]>(
    ['alerts', status],
    () => api.get('/alerts', { params: { status } }).then(r => r.data)
  );
}

export function useAlertRules() {
  return useQuery<AlertRule[]>(
    'alert-rules',
    () => api.get('/alerts/rules').then(r => r.data)
  );
}
