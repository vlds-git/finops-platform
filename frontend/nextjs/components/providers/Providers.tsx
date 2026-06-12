'use client';

import { QueryClient, QueryClientProvider } from 'react-query';
import { ReactNode } from 'react';
import { CurrencyProvider } from '@/contexts/CurrencyContext';

const queryClient = new QueryClient();

export function Providers({ children }: { children: ReactNode }) {
  return (
    <QueryClientProvider client={queryClient}>
      <CurrencyProvider>{children}</CurrencyProvider>
    </QueryClientProvider>
  );
}
