'use client';

import { BaseChart } from './BaseChart';
import type { ForecastPoint } from '@/types';

interface ForecastChartProps {
  historical: { date: string; value: number }[];
  forecast: ForecastPoint[];
  className?: string;
}

export function ForecastChart({ historical, forecast, className }: ForecastChartProps) {
  const allDates = [
    ...historical.map(d => d.date),
    ...forecast.map(d => d.date),
  ];

  const historicalData = [
    ...historical.map(d => d.value),
    ...Array(forecast.length).fill(null),
  ];

  const forecastData = [
    ...Array(historical.length).fill(null),
    ...forecast.map(d => d.value),
  ];

  const upperBound = [
    ...Array(historical.length).fill(null),
    ...forecast.map(d => d.upper),
  ];

  const lowerBound = [
    ...Array(historical.length).fill(null),
    ...forecast.map(d => d.lower),
  ];

  const option = {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis',
      backgroundColor: '#1e293b',
      borderColor: '#334155',
      textStyle: { color: '#e2e8f0' },
    },
    legend: {
      data: ['Histórico', 'Forecast', 'Intervalo'],
      textStyle: { color: '#94a3b8' },
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      data: allDates,
      axisLine: { lineStyle: { color: '#334155' } },
      axisLabel: { color: '#94a3b8' },
    },
    yAxis: {
      type: 'value',
      axisLine: { lineStyle: { color: '#334155' } },
      splitLine: { lineStyle: { color: '#1e293b' } },
      axisLabel: { color: '#94a3b8' },
    },
    series: [
      {
        name: 'Histórico',
        data: historicalData,
        type: 'line',
        smooth: true,
        lineStyle: { color: '#38bdf8', width: 2 },
        itemStyle: { color: '#38bdf8' },
      },
      {
        name: 'Forecast',
        data: forecastData,
        type: 'line',
        smooth: true,
        lineStyle: { color: '#a78bfa', width: 2, type: 'dashed' },
        itemStyle: { color: '#a78bfa' },
      },
      {
        name: 'Intervalo Superior',
        data: upperBound,
        type: 'line',
        lineStyle: { opacity: 0 },
        itemStyle: { opacity: 0 },
        areaStyle: {
          color: 'rgba(167, 139, 250, 0.1)',
        },
        stack: 'confidence',
        symbol: 'none',
      },
      {
        name: 'Intervalo Inferior',
        data: lowerBound.map((v, i) => v !== null ? upperBound[i] - v : null),
        type: 'line',
        lineStyle: { opacity: 0 },
        itemStyle: { opacity: 0 },
        areaStyle: {
          color: 'rgba(167, 139, 250, 0.1)',
        },
        stack: 'confidence',
        symbol: 'none',
      },
    ],
  };

  return <BaseChart option={option} className={className} />;
}
