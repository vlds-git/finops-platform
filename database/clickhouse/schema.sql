-- ClickHouse Schema for FinOps Enterprise Platform

-- Main costs table (raw data from FOCUS)
CREATE TABLE IF NOT EXISTS costs (
    id UUID DEFAULT generateUUIDv4(),
    date Date,
    datetime DateTime,
    provider LowCardinality(String),
    account_id LowCardinality(String),
    billing_account_id LowCardinality(String),
    billing_account_name String,
    charge_type LowCardinality(String),
    charge_subcategory LowCardinality(String),
    service_name LowCardinality(String),
    service_category LowCardinality(String),
    resource_type LowCardinality(String),
    resource_id String,
    resource_name String,
    region LowCardinality(String),
    availability_zone LowCardinality(String),
    usage_unit LowCardinality(String),
    usage_quantity Decimal(18,6),
    effective_cost Decimal(18,4),
    list_cost Decimal(18,4),
    contracted_cost Decimal(18,4),
    amortized_cost Decimal(18,4),
    tags Map(String, String),
    environment LowCardinality(String) DEFAULT tags['environment'],
    application LowCardinality(String) DEFAULT tags['application'],
    business_unit LowCardinality(String) DEFAULT tags['business_unit'],
    project LowCardinality(String) DEFAULT tags['project'],
    team LowCardinality(String) DEFAULT tags['team'],
    invoice_issuer LowCardinality(String),
    invoice_id String,
    billing_period_start Date,
    billing_period_end Date,
    charge_period_start Date,
    charge_period_end Date,

    PRIMARY KEY (date, provider, account_id, service_name),
    ORDER BY (date, provider, account_id, service_name, resource_id),
    PARTITION BY toYYYYMM(date),
    TTL date + INTERVAL 36 MONTH
) ENGINE = MergeTree();

-- Materialized View: Daily costs by service
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_daily_costs_by_service
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (date, provider, account_id, service_name)
AS SELECT
    date,
    provider,
    account_id,
    service_name,
    service_category,
    sum(effective_cost) as total_cost,
    sum(amortized_cost) as total_amortized,
    sum(list_cost) as total_list,
    sum(usage_quantity) as total_usage,
    count() as record_count,
    uniqExact(resource_id) as resource_count
FROM costs
GROUP BY date, provider, account_id, service_name, service_category;

-- Materialized View: Daily costs by application
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_daily_costs_by_app
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (date, provider, application)
AS SELECT
    date,
    provider,
    application,
    environment,
    business_unit,
    sum(effective_cost) as total_cost,
    sum(usage_quantity) as total_usage,
    count() as record_count
FROM costs
GROUP BY date, provider, application, environment, business_unit;

-- Materialized View: Monthly costs by provider
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_monthly_costs_by_provider
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(month)
ORDER BY (month, provider, account_id)
AS SELECT
    toStartOfMonth(date) as month,
    provider,
    account_id,
    sum(effective_cost) as total_cost,
    sum(amortized_cost) as total_amortized,
    sum(list_cost) as total_list,
    sum(usage_quantity) as total_usage,
    count() as record_count,
    uniqExact(service_name) as service_count
FROM costs
GROUP BY month, provider, account_id;

-- Materialized View: Costs by region
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_costs_by_region
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMM(date)
ORDER BY (date, provider, region)
AS SELECT
    date,
    provider,
    region,
    sum(effective_cost) as total_cost,
    sum(usage_quantity) as total_usage,
    count() as record_count
FROM costs
GROUP BY date, provider, region;

-- Aggregated table for fast dashboard queries
CREATE TABLE IF NOT EXISTS costs_aggregated (
    date Date,
    provider LowCardinality(String),
    account_id LowCardinality(String),
    service_name LowCardinality(String),
    application LowCardinality(String),
    environment LowCardinality(String),
    business_unit LowCardinality(String),
    region LowCardinality(String),
    total_cost Decimal(18,4),
    total_usage Decimal(18,6),
    resource_count UInt32,

    PRIMARY KEY (date, provider, service_name),
    ORDER BY (date, provider, service_name, application, environment),
    PARTITION BY toYYYYMM(date)
) ENGINE = SummingMergeTree();

-- Populate aggregated table from MV
CREATE MATERIALIZED VIEW IF NOT EXISTS mv_costs_aggregated_populator
TO costs_aggregated
AS SELECT
    date,
    provider,
    account_id,
    service_name,
    application,
    environment,
    business_unit,
    region,
    sum(effective_cost) as total_cost,
    sum(usage_quantity) as total_usage,
    uniqExact(resource_id) as resource_count
FROM costs
GROUP BY date, provider, account_id, service_name, application, environment, business_unit, region;

-- Table for forecast storage
CREATE TABLE IF NOT EXISTS forecast_results (
    provider LowCardinality(String),
    account_id LowCardinality(String),
    service LowCardinality(String),
    forecast_date Date,
    model LowCardinality(String),
    value Decimal(18,4),
    lower_bound Decimal(18,4),
    upper_bound Decimal(18,4),
    confidence Decimal(5,2),
    generated_at DateTime DEFAULT now(),

    PRIMARY KEY (forecast_date, provider, model),
    ORDER BY (forecast_date, provider, account_id, service, model),
    PARTITION BY toYYYYMM(forecast_date)
) ENGINE = MergeTree();

-- Table for anomaly storage
CREATE TABLE IF NOT EXISTS anomaly_results (
    provider LowCardinality(String),
    account_id LowCardinality(String),
    service LowCardinality(String),
    date Date,
    value Decimal(18,4),
    expected Decimal(18,4),
    deviation Decimal(18,4),
    severity LowCardinality(String),
    score Decimal(10,4),
    method LowCardinality(String),
    status LowCardinality(String) DEFAULT 'open',
    created_at DateTime DEFAULT now(),

    PRIMARY KEY (date, provider, service),
    ORDER BY (date, provider, account_id, service),
    PARTITION BY toYYYYMM(date)
) ENGINE = MergeTree();

-- Retention policy: drop partitions older than 36 months
ALTER TABLE costs MODIFY TTL date + INTERVAL 36 MONTH;
ALTER TABLE mv_daily_costs_by_service MODIFY TTL date + INTERVAL 36 MONTH;
ALTER TABLE mv_daily_costs_by_app MODIFY TTL date + INTERVAL 36 MONTH;
ALTER TABLE mv_costs_by_region MODIFY TTL date + INTERVAL 36 MONTH;
ALTER TABLE costs_aggregated MODIFY TTL date + INTERVAL 36 MONTH;

-- Indexes for performance
ALTER TABLE costs ADD INDEX IF NOT EXISTS idx_resource_id resource_id TYPE bloom_filter GRANULARITY 3;
ALTER TABLE costs ADD INDEX IF NOT EXISTS idx_tags tags TYPE bloom_filter GRANULARITY 3;
