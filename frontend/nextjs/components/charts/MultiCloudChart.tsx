'use client';

import * as echarts from 'echarts';
import { BaseChart } from './BaseChart';

interface MultiCloudChartProps {
  data: {
    months: string[];
    huawei: number[];
    azure: number[];
    aws: number[];
  };
  className?: string;
}

export function MultiCloudChart({ data, className }: MultiCloudChartProps) {
  const option: echarts.EChartsOption = {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'axis',
      backgroundColor: '#1e293b',
      borderColor: '#334155',
      textStyle: { color: '#e2e8f0' },
    },
    legend: {
      data: ['Huawei Cloud', 'Azure', 'AWS'],
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
      data: data.months,
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
        name: 'Huawei Cloud',
        type: 'line',
        data: data.huawei,
        lineStyle: { color: '#ef4444' },
        itemStyle: { color: '#ef4444' },
      },
      {
        name: 'Azure',
        type: 'line',
        data: data.azure,
        lineStyle: { color: '#3b82f6' },
        itemStyle: { color: '#3b82f6' },
      },
      {
        name: 'AWS',
        type: 'line',
        data: data.aws,
        lineStyle: { color: '#f97316' },
        itemStyle: { color: '#f97316' },
      },
    ],
  };

  return <BaseChart option={option} className={className} />;
}
