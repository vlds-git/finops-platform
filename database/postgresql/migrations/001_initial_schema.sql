-- PostgreSQL Schema for FinOps Enterprise Platform
-- Migration: 001_initial_schema.sql

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    roles VARCHAR(50)[] DEFAULT ARRAY['viewer'],
    provider VARCHAR(50) DEFAULT 'local',
    external_id VARCHAR(255),
    active BOOLEAN DEFAULT true,
    last_login TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Audit log
CREATE TABLE IF NOT EXISTS audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(50) NOT NULL,
    resource VARCHAR(100) NOT NULL,
    resource_id VARCHAR(255),
    details JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Cloud accounts
CREATE TABLE IF NOT EXISTS cloud_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider VARCHAR(50) NOT NULL,
    account_id VARCHAR(255) NOT NULL,
    account_name VARCHAR(255),
    access_key VARCHAR(255),
    secret_key VARCHAR(255),
    endpoint VARCHAR(255),
    bucket VARCHAR(255),
    prefix VARCHAR(255),
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(provider, account_id)
);

-- Budgets
CREATE TABLE IF NOT EXISTS budgets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    amount DECIMAL(18,4) NOT NULL,
    currency VARCHAR(3) DEFAULT 'BRL',
    spent DECIMAL(18,4) DEFAULT 0,
    period VARCHAR(20) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    alert_threshold DECIMAL(5,2) DEFAULT 0.80,
    provider VARCHAR(50),
    account_id VARCHAR(255),
    service VARCHAR(100),
    tags JSONB,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Alert rules
CREATE TABLE IF NOT EXISTS alert_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    condition VARCHAR(100) NOT NULL,
    threshold DECIMAL(18,4) NOT NULL,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    channel VARCHAR(50) NOT NULL,
    destination VARCHAR(500) NOT NULL,
    enabled BOOLEAN DEFAULT true,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Alerts (instances)
CREATE TABLE IF NOT EXISTS alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID REFERENCES alert_rules(id) ON DELETE CASCADE,
    rule_name VARCHAR(255),
    severity VARCHAR(20) NOT NULL,
    message TEXT NOT NULL,
    value DECIMAL(18,4),
    threshold DECIMAL(18,4),
    status VARCHAR(20) DEFAULT 'firing' CHECK (status IN ('firing', 'resolved', 'acknowledged')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP,
    resolved_by UUID REFERENCES users(id)
);

-- Recommendations
CREATE TABLE IF NOT EXISTS recommendations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    resource_id VARCHAR(255),
    resource_type VARCHAR(100),
    service VARCHAR(100),
    region VARCHAR(100),
    current_cost DECIMAL(18,4),
    projected_cost DECIMAL(18,4),
    savings DECIMAL(18,4),
    savings_percentage DECIMAL(5,2),
    confidence DECIMAL(5,2),
    priority VARCHAR(20),
    justification TEXT,
    action TEXT,
    risk TEXT,
    implementation TEXT,
    status VARCHAR(20) DEFAULT 'open' CHECK (status IN ('open', 'applied', 'ignored', 'expired')),
    applied_at TIMESTAMP,
    applied_by UUID REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Forecasts
CREATE TABLE IF NOT EXISTS forecasts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider VARCHAR(50) NOT NULL,
    account_id VARCHAR(255),
    service VARCHAR(100),
    period VARCHAR(20) NOT NULL,
    model VARCHAR(50) NOT NULL,
    total_forecast DECIMAL(18,4),
    trend DECIMAL(10,4),
    confidence DECIMAL(5,2),
    data JSONB,
    generated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Anomalies
CREATE TABLE IF NOT EXISTS anomalies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider VARCHAR(50) NOT NULL,
    account_id VARCHAR(255),
    service VARCHAR(100),
    date DATE NOT NULL,
    value DECIMAL(18,4) NOT NULL,
    expected DECIMAL(18,4) NOT NULL,
    deviation DECIMAL(18,4) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    score DECIMAL(10,4),
    method VARCHAR(50),
    status VARCHAR(20) DEFAULT 'open',
    investigated_at TIMESTAMP,
    investigated_by UUID REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Ingestion checkpoints
CREATE TABLE IF NOT EXISTS ingestion_checkpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider VARCHAR(50) NOT NULL,
    account_id VARCHAR(255) NOT NULL,
    last_file VARCHAR(500),
    last_offset BIGINT DEFAULT 0,
    last_processed_at TIMESTAMP,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(provider, account_id)
);

-- Ingestion dead letter queue
CREATE TABLE IF NOT EXISTS ingestion_dlq (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider VARCHAR(50) NOT NULL,
    account_id VARCHAR(255) NOT NULL,
    file_name VARCHAR(500),
    error_message TEXT,
    retry_count INT DEFAULT 0,
    last_retry_at TIMESTAMP,
    resolved BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Dashboards
CREATE TABLE IF NOT EXISTS dashboards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    config JSONB,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_audit_user_id ON audit(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_created_at ON audit(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_status ON alerts(status);
CREATE INDEX IF NOT EXISTS idx_alerts_created_at ON alerts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_recommendations_status ON recommendations(status);
CREATE INDEX IF NOT EXISTS idx_anomalies_date ON anomalies(date DESC);
CREATE INDEX IF NOT EXISTS idx_anomalies_status ON anomalies(status);
CREATE INDEX IF NOT EXISTS idx_forecasts_provider ON forecasts(provider, account_id);
CREATE INDEX IF NOT EXISTS idx_budgets_period ON budgets(start_date, end_date);

-- Functions
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_budgets_updated_at BEFORE UPDATE ON budgets FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_alert_rules_updated_at BEFORE UPDATE ON alert_rules FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_cloud_accounts_updated_at BEFORE UPDATE ON cloud_accounts FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_ingestion_checkpoints_updated_at BEFORE UPDATE ON ingestion_checkpoints FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_dashboards_updated_at BEFORE UPDATE ON dashboards FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Seed data
INSERT INTO users (email, name, password_hash, roles) VALUES
('admin@finops.local', 'Admin User', 'admin123', ARRAY['admin']),
('analyst@finops.local', 'FinOps Analyst', 'analyst123', ARRAY['analyst']),
('viewer@finops.local', 'Viewer', 'viewer123', ARRAY['viewer'])
ON CONFLICT DO NOTHING;

INSERT INTO cloud_accounts (provider, account_id, account_name, bucket, prefix, active) VALUES
('huawei', 'hw-account-001', 'Produção Huawei', 'focusfinops', 'daily-exports/Daily_Cost_Export_Focus1-0/', true),
('azure', 'az-sub-001', 'Produção Azure', 'finops-focus-az', 'billing/', false),
('aws', 'aws-account-001', 'Produção AWS', 'finops-focus-aws', 'cur/', false)
ON CONFLICT DO NOTHING;

INSERT INTO budgets (name, amount, period, start_date, end_date, alert_threshold, provider, account_id) VALUES
('Produção - Q2 2026', 150000, 'quarterly', '2026-04-01', '2026-06-30', 0.90, 'all', 'all'),
('Desenvolvimento', 50000, 'quarterly', '2026-04-01', '2026-06-30', 0.85, 'all', 'all'),
('Staging', 30000, 'quarterly', '2026-04-01', '2026-06-30', 0.90, 'all', 'all')
ON CONFLICT DO NOTHING;

INSERT INTO alert_rules (name, description, condition, threshold, severity, channel, destination) VALUES
('Budget Threshold', 'Alert when budget exceeds threshold', 'budget_utilization', 0.90, 'critical', 'slack', 'https://hooks.slack.com/finops'),
('Cost Anomaly', 'Alert on cost anomalies', 'cost_spike', 2.0, 'high', 'email', 'finops@company.com'),
('Forecast Alert', 'Alert when forecast exceeds budget', 'forecast_exceed', 1.05, 'medium', 'webhook', 'https://api.company.com/alerts')
ON CONFLICT DO NOTHING;
