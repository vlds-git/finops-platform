export interface User {
  id: string;
  email: string;
  name: string;
  roles: string[];
  active: boolean;
  last_login?: string;
  created_at: string;
}

export interface CostSummary {
  total_cost: number;
  amortized_cost: number;
  list_cost: number;
  service_count: number;
  resource_count: number;
}

export interface CostTrend {
  date: string;
  cost: number;
  usage: number;
}

export interface ServiceCost {
  service: string;
  cost: number;
  usage: number;
}

export interface KPIData {
  name: string;
  value: number;
  unit: string;
  trend: number;
  target?: number;
  status: 'good' | 'warning' | 'danger';
}

export interface Budget {
  id: string;
  name: string;
  amount: number;
  spent: number;
  remaining: number;
  period: string;
  start_date: string;
  end_date: string;
  alert_threshold: number;
  provider: string;
  account_id: string;
  created_at: string;
}

export interface ForecastPoint {
  date: string;
  value: number;
  lower: number;
  upper: number;
}

export interface ForecastData {
  provider: string;
  account_id: string;
  period: string;
  model: string;
  forecast: ForecastPoint[];
  total_forecast: number;
  trend: number;
  confidence: number;
  generated_at: string;
}

export interface AnomalyPoint {
  date: string;
  value: number;
  expected: number;
  deviation: number;
  severity: 'low' | 'medium' | 'high' | 'critical';
  score: number;
  method: string;
}

export interface AnomalyData {
  provider: string;
  account_id: string;
  anomalies: AnomalyPoint[];
  total_anomalies: number;
  total_impact: number;
  baseline_mean: number;
  baseline_std: number;
  generated_at: string;
}

export interface RecommendationItem {
  id: string;
  category: string;
  title: string;
  description: string;
  resource_id: string;
  resource_type: string;
  service: string;
  region: string;
  current_cost: number;
  projected_cost: number;
  savings: number;
  savings_percentage: number;
  confidence: number;
  priority: 'low' | 'medium' | 'high';
  justification: string;
  action: string;
  risk: string;
  implementation: string;
  created_at: string;
}

export interface RecommendationData {
  provider: string;
  account_id: string;
  total_savings: number;
  total_opportunities: number;
  recommendations: RecommendationItem[];
  generated_at: string;
}

export interface AlertRule {
  id: string;
  name: string;
  description: string;
  condition: string;
  threshold: number;
  severity: 'low' | 'medium' | 'high' | 'critical';
  channel: string;
  destination: string;
  enabled: boolean;
  created_at: string;
}

export interface Alert {
  id: string;
  rule_id: string;
  rule_name: string;
  severity: string;
  message: string;
  value: number;
  threshold: number;
  status: 'firing' | 'resolved' | 'acknowledged';
  created_at: string;
  resolved_at?: string;
}

export interface CloudAccount {
  id: string;
  provider: string;
  account_id: string;
  account_name: string;
  bucket: string;
  prefix: string;
  active: boolean;
  created_at: string;
}

export interface ExecutiveDashboard {
  total_cost: number;
  forecast_30d: number;
  potential_savings: number;
  top_services: { name: string; cost: number }[];
  top_applications: { name: string; cost: number }[];
  kpis: { name: string; value: number }[];
}

export interface OperationalDashboard {
  by_service: { name: string; cost: number }[];
  by_project: { name: string; cost: number }[];
  by_environment: { name: string; cost: number }[];
  by_region: { name: string; cost: number }[];
  by_business_unit: { name: string; cost: number }[];
  by_tags: { key: string; value: string; cost: number }[];
}

export interface MultiCloudData {
  provider: string;
  account_count: number;
  cost: number;
  percentage: number;
}
