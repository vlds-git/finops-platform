'use client';

import { createContext, useContext, useEffect, useState, ReactNode } from 'react';
import { useQueryClient } from 'react-query';

type Currency = 'USD' | 'BRL';

interface CurrencyContextType {
  currency: Currency;
  setCurrency: (currency: Currency) => void;
  rate: number;
}

const CurrencyContext = createContext<CurrencyContextType>({
  currency: 'USD',
  setCurrency: () => {},
  rate: 5.15,
});

export function CurrencyProvider({ children, defaultRate = 5.15 }: { children: ReactNode; defaultRate?: number }) {
  const queryClient = useQueryClient();
  const [currency, setCurrencyState] = useState<Currency>('USD');
  const [rate, setRate] = useState<number>(defaultRate);

  useEffect(() => {
    const saved = typeof window !== 'undefined' ? localStorage.getItem('finops-currency') : null;
    if (saved === 'USD' || saved === 'BRL') {
      setCurrencyState(saved);
    }
  }, []);

  useEffect(() => {
    fetch('/api/v1/ingestion/status')
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => {
        if (data?.usd_to_brl) {
          setRate(Number(data.usd_to_brl));
        }
      })
      .catch(() => {
        // ignore
      });
  }, []);

  const setCurrency = (value: Currency) => {
    setCurrencyState(value);
    if (typeof window !== 'undefined') {
      localStorage.setItem('finops-currency', value);
    }
    // Invalidate all cost/dashboard queries so they refetch with the new currency
    queryClient.invalidateQueries({ queryKey: ['costs'] });
    queryClient.invalidateQueries({ queryKey: ['costs-trends'] });
    queryClient.invalidateQueries({ queryKey: ['services'] });
    queryClient.invalidateQueries({ queryKey: ['regions'] });
    queryClient.invalidateQueries({ queryKey: ['providers'] });
    queryClient.invalidateQueries({ queryKey: ['kpis'] });
    queryClient.invalidateQueries({ queryKey: ['executive-dashboard'] });
    queryClient.invalidateQueries({ queryKey: ['operational-dashboard'] });
    queryClient.invalidateQueries({ queryKey: ['forecast'] });
    queryClient.invalidateQueries({ queryKey: ['anomalies'] });
    queryClient.invalidateQueries({ queryKey: ['recommendations'] });
    queryClient.invalidateQueries({ queryKey: ['budgets'] });
  };

  return (
    <CurrencyContext.Provider value={{ currency, setCurrency, rate }}>
      {children}
    </CurrencyContext.Provider>
  );
}

export function useCurrency() {
  return useContext(CurrencyContext);
}
