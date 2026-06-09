'use client';

import { BaseChart } from './BaseChart';

interface BarChartProps {
  data: { name: string; value: number }[];
  className?: string;
  horizontal?: boolean;
  color?: string;
}

export function BarChart({ data, className, horizontal = false, color = '#38bdf8' }: BarChartProps) {
  const option = {
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
      type: horizontal ? 'value' : 'category',
      data: horizontal ? undefined : data.map(d => d.name),
      axisLine: { lineStyle: { color: '#334155' } },
      axisLabel: { color: '#94a3b8' },
      splitLine: horizontal ? { lineStyle: { color: '#1e293b' } } : undefined,
    },
    yAxis: {
      type: horizontal ? 'category' : 'value',
      data: horizontal ? data.map(d => d.name) : undefined,
      axisLine: { lineStyle: { color: '#334155' } },
      axisLabel: { color: '#94a3b8' },
      splitLine: !horizontal ? { lineStyle: { color: '#1e293b' } } : undefined,
    },
    series: [
      {
        data: data.map(d => d.value),
        type: 'bar',
        itemStyle: {
          color: {
            type: 'linear',
            x: horizontal ? 1 : 0,
            y: 0,
            x2: 0,
            y2: horizontal ? 0 : 1,
            colorStops: [
              { offset: 0, color },
              { offset: 1, color: color + '80' },
            ],
          },
        },
        barWidth: '60%',
      },
    ],
  };

  return <BaseChart option={option} className={className} />;
}
