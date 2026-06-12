'use client';

import * as echarts from 'echarts';
import { BaseChart } from './BaseChart';
import type { AnomalyPoint } from '@/types';

interface AnomalyChartProps {
  data: number[];
  dates: string[];
  anomalies: AnomalyPoint[];
  className?: string;
}

export function AnomalyChart({ data, dates, anomalies, className }: AnomalyChartProps) {
  const anomalyMap = new Map(anomalies.map(a => [a.date, a]));
  const scatterData = dates.map((date, i) => {
    const anomaly = anomalyMap.get(date);
    return anomaly ? [i, data[i]] : null;
  }).filter((item): item is number[] => item !== null);

  const option: echarts.EChartsOption = {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis',
      backgroundColor: '#1e293b',
      borderColor: '#334155',
      textStyle: { color: '#e2e8f0' },
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      data: dates,
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
        data: data,
        type: 'line',
        smooth: true,
        lineStyle: { color: '#38bdf8', width: 2 },
        itemStyle: { color: '#38bdf8' },
      },
      {
        data: scatterData,
        type: 'scatter',
        symbolSize: 15,
        itemStyle: { color: '#ef4444' },
      },
    ],
  };

  return <BaseChart option={option} className={className} />;
}
