-- ============================================
-- FINOPS PLATFORM - INITIAL DATABASE SETUP
-- ============================================

-- Criação do schema principal
CREATE SCHEMA IF NOT EXISTS finops;

-- Tabela de usuários
CREATE TABLE IF NOT EXISTS finops.users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) DEFAULT 'user',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabela de tenants/cloud accounts
CREATE TABLE IF NOT EXISTS finops.cloud_accounts (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    account_id VARCHAR(255) NOT NULL,
    credentials JSONB,
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Tabela de alertas
CREATE TABLE IF NOT EXISTS finops.alerts (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    severity VARCHAR(50) DEFAULT 'medium',
    status VARCHAR(50) DEFAULT 'open',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP
);

-- Tabela de configurações
CREATE TABLE IF NOT EXISTS finops.configurations (
    id SERIAL PRIMARY KEY,
    key VARCHAR(255) UNIQUE NOT NULL,
    value TEXT,
    description TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Índices
CREATE INDEX IF NOT EXISTS idx_users_email ON finops.users(email);
CREATE INDEX IF NOT EXISTS idx_alerts_status ON finops.alerts(status);
CREATE INDEX IF NOT EXISTS idx_cloud_accounts_provider ON finops.cloud_accounts(provider);

-- Dados iniciais
INSERT INTO finops.configurations (key, value, description) VALUES
    ('platform_version', '1.0.0', 'Versão da plataforma'),
    ('currency_default', 'USD', 'Moeda padrão para exibição de custos'),
    ('anomaly_threshold', '2.5', 'Multiplicador do desvio padrão para detecção de anomalias')
ON CONFLICT (key) DO NOTHING;
