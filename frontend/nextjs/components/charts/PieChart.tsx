'use client';

import { BaseChart } from './BaseChart';

interface PieChartProps {
  data: { name: string; value: number }[];
  className?: string;
  colors?: string[];
}

export function PieChart({ data, className, colors = ['#38bdf8', '#818cf8', '#c084fc', '#f472b6', '#94a3b8'] }: PieChartProps) {
  const option = {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'item',
      backgroundColor: '#1e293b',
      borderColor: '#334155',
      textStyle: { color: '#e2e8f0' },
      formatter: '{b}: {c} ({d}%)',
    },
    series: [
      {
        type: 'pie',
        radius: ['40%', '70%'],
        data: data.map((d, i) => ({
          value: d.value,
          name: d.name,
          itemStyle: { color: colors[i % colors.length] },
        })),
        label: {
          color: '#e2e8f0',
          formatter: '{b}
{d}%',
        },
        emphasis: {
          itemStyle: {
            shadowBlur: 10,
            shadowOffsetX: 0,
            shadowColor: 'rgba(0, 0, 0, 0.5)',
          },
        },
      },
    ],
  };

  return <BaseChart option={option} className={className} />;
}
