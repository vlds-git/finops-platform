-- FinOps Platform - ClickHouse Schema

CREATE DATABASE IF NOT EXISTS finops;

-- Raw cost data from ingestion
CREATE TABLE IF NOT EXISTS finops.costs_raw (
    provider String,
    billing_account_id String,
    service_name String,
    resource_type String,
    region String,
    usage_quantity Float64,
    usage_unit String,
    effective_cost Float64,
    effective_cost_brl Float64,
    list_cost Float64,
    contracted_cost Float64,
    amortized_cost Float64,
    date Date,
    environment String,
    application String,
    business_unit String,
    tags Map(String, String)
) ENGINE = MergeTree()
ORDER BY (provider, date, service_name)
PARTITION BY toYYYYMM(date)
TTL date + INTERVAL 36 MONTH;

-- Daily aggregated costs
CREATE TABLE IF NOT EXISTS finops.costs_daily (
    provider String,
    service_name String,
    region String,
    date Date,
    total_cost Float64,
    total_cost_brl Float64,
    total_usage Float64,
    environment String,
    application String,
    business_unit String
) ENGINE = MergeTree()
ORDER BY (provider, date, service_name)
PARTITION BY toYYYYMM(date);

-- Anomalies detected
CREATE TABLE IF NOT EXISTS finops.anomalies (
    id UUID,
    service_name String,
    region String,
    date Date,
    expected_value Float64,
    actual_value Float64,
    deviation Float64,
    severity String,
    detected_at DateTime
) ENGINE = MergeTree()
ORDER BY (date, service_name);

-- Forecasts
CREATE TABLE IF NOT EXISTS finops.forecasts (
    service_name String,
    region String,
    forecast_date Date,
    predicted_cost Float64,
    predicted_cost_brl Float64,
    confidence_lower Float64,
    confidence_upper Float64,
    model_version String,
    created_at DateTime
) ENGINE = MergeTree()
ORDER BY (forecast_date, service_name);

-- Recommendations
CREATE TABLE IF NOT EXISTS finops.recommendations (
    id UUID,
    category String,
    title String,
    description String,
    potential_savings Float64,
    potential_savings_brl Float64,
    priority String,
    status String,
    created_at DateTime
) ENGINE = MergeTree()
ORDER BY (created_at, category);
