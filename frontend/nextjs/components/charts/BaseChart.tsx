'use client';

import { useEffect, useRef } from 'react';
import * as echarts from 'echarts';

interface BaseChartProps {
  option: echarts.EChartsOption;
  className?: string;
}

export function BaseChart({ option, className }: BaseChartProps) {
  const chartRef = useRef<HTMLDivElement>(null);
  const chartInstance = useRef<echarts.ECharts | null>(null);

  useEffect(() => {
    if (!chartRef.current) return;

    chartInstance.current = echarts.init(chartRef.current, 'dark', {
      renderer: 'canvas',
    });

    chartInstance.current.setOption(option);

    const handleResize = () => {
      chartInstance.current?.resize();
    };

    window.addEventListener('resize', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
      chartInstance.current?.dispose();
    };
  }, [option]);

  return <div ref={chartRef} className={className || 'w-full h-[300px]'} />;
}
