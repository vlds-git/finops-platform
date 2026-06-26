-- Deduplica costs_raw removendo registros exatamente iguais.
-- Normaliza a coluna tags (Map) para ordem fixa, pois maps com mesmas chaves
-- em ordem diferente sao considerados valores distintos pelo ClickHouse.
-- Executar dentro do container clickhouse via:
-- clickhouse-client --database=finops --multiquery < dedup_clickhouse.sql

CREATE TABLE IF NOT EXISTS finops.costs_raw_dedup (
    provider String,
    billing_account_id String,
    service_name String,
    resource_type String,
    resource_id String,
    region String,
    usage_quantity Float64,
    usage_unit String,
    effective_cost Float64,
    effective_cost_brl Float64,
    list_cost Float64,
    list_cost_brl Float64,
    contracted_cost Float64,
    contracted_cost_brl Float64,
    amortized_cost Float64,
    amortized_cost_brl Float64,
    date Date,
    environment String,
    application String,
    business_unit String,
    tags Map(String, String)
) ENGINE = MergeTree()
ORDER BY (provider, date, service_name)
PARTITION BY toYYYYMM(date);

-- Inserir apenas registros distintos, ignorando a ordem dos tags e
-- reconstruindo os tags com ordem fixa (application, business_unit, environment).
INSERT INTO finops.costs_raw_dedup
SELECT DISTINCT
    provider,
    billing_account_id,
    service_name,
    resource_type,
    resource_id,
    region,
    usage_quantity,
    usage_unit,
    effective_cost,
    effective_cost_brl,
    list_cost,
    list_cost_brl,
    contracted_cost,
    contracted_cost_brl,
    amortized_cost,
    amortized_cost_brl,
    date,
    environment,
    application,
    business_unit,
    map('application', application, 'business_unit', business_unit, 'environment', environment)
FROM finops.costs_raw;

-- Trocar as tabelas atomicamente
RENAME TABLE finops.costs_raw TO finops.costs_raw_old, finops.costs_raw_dedup TO finops.costs_raw;

-- Remover tabela antiga
DROP TABLE finops.costs_raw_old;
