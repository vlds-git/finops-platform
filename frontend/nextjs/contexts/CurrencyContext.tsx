'use client';

import { createContext, useContext, useEffect, useState, ReactNode } from 'react';

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
