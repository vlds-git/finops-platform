#!/bin/bash
# ================================================================================
# SCRIPT DE MIGRAÇÃO - FINOPS PLATFORM DEPLOY FIX
# ================================================================================
# Execute este script na RAIZ do repositório finops-platform
# ================================================================================

set -e

echo "🚀 Iniciando correções de deploy..."
echo ""

# ================================================================================
# PASSO 1: Verificar se estamos na raiz do repositório
# ================================================================================
if [ ! -d ".git" ] && [ ! -d "backend" ] && [ ! -d "frontend" ]; then
    echo "❌ ERRO: Execute este script na RAIZ do repositório finops-platform"
    echo "   Você deve estar na pasta que contém: backend/, frontend/, deploy/, etc."
    exit 1
fi

echo "✅ Localização correta detectada"
echo ""

# ================================================================================
# PASSO 2: Mover docker-compose.yml da pasta deploy/docker/ para a RAIZ
# ================================================================================
echo "📦 PASSO 2: Movendo docker-compose.yml para a raiz..."

if [ -f "deploy/docker/docker-compose.yml" ]; then
    # Fazer backup do antigo
    cp deploy/docker/docker-compose.yml deploy/docker/docker-compose.yml.backup
    echo "   💾 Backup criado: deploy/docker/docker-compose.yml.backup"

    # Mover para raiz
    cp deploy/docker/docker-compose.yml docker-compose.yml
    echo "   ✅ docker-compose.yml copiado para a raiz"

    # Remover o antigo (opcional - comentado para segurança)
    # rm deploy/docker/docker-compose.yml
    # echo "   🗑️  Antigo removido de deploy/docker/"
else
    echo "   ⚠️  docker-compose.yml não encontrado em deploy/docker/"
    echo "   ℹ️  Verifique se o arquivo já foi movido manualmente"
fi
echo ""

# ================================================================================
# PASSO 3: Corrigir o next.config.js do frontend
# ================================================================================
echo "📦 PASSO 3: Corrigindo next.config.js..."

if [ -f "frontend/nextjs/next.config.js" ]; then
    # Fazer backup
    cp frontend/nextjs/next.config.js frontend/nextjs/next.config.js.backup
    echo "   💾 Backup criado: frontend/nextjs/next.config.js.backup"

    # Sobrescrever com a versão corrigida
    cat > frontend/nextjs/next.config.js << 'EOF'
/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  output: 'standalone',
  images: {
    remotePatterns: [
      {
        protocol: 'https',
        hostname: '**',
      },
    ],
  },
}

module.exports = nextConfig
EOF
    echo "   ✅ next.config.js atualizado com output: 'standalone'"
else
    echo "   ❌ next.config.js não encontrado em frontend/nextjs/"
fi
echo ""

# ================================================================================
# PASSO 4: Criar schema inicial do PostgreSQL
# ================================================================================
echo "📦 PASSO 4: Criando schema inicial do PostgreSQL..."

mkdir -p database/postgresql/migrations

cat > database/postgresql/migrations/001_init.sql << 'EOF'
-- ============================================
-- FINOPS PLATFORM - INITIAL DATABASE SETUP
-- ============================================

CREATE SCHEMA IF NOT EXISTS finops;

CREATE TABLE IF NOT EXISTS finops.users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) DEFAULT 'user',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS finops.cloud_accounts (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    account_id VARCHAR(255) NOT NULL,
    credentials JSONB,
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS finops.alerts (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    severity VARCHAR(50) DEFAULT 'medium',
    status VARCHAR(50) DEFAULT 'open',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS finops.configurations (
    id SERIAL PRIMARY KEY,
    key VARCHAR(255) UNIQUE NOT NULL,
    value TEXT,
    description TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON finops.users(email);
CREATE INDEX IF NOT EXISTS idx_alerts_status ON finops.alerts(status);
CREATE INDEX IF NOT EXISTS idx_cloud_accounts_provider ON finops.cloud_accounts(provider);

INSERT INTO finops.configurations (key, value, description) VALUES
    ('platform_version', '1.0.0', 'Versão da plataforma'),
    ('currency_default', 'USD', 'Moeda padrão para exibição de custos'),
    ('anomaly_threshold', '2.5', 'Multiplicador do desvio padrão para detecção de anomalias')
ON CONFLICT (key) DO NOTHING;
EOF

echo "   ✅ Schema criado: database/postgresql/migrations/001_init.sql"
echo ""

# ================================================================================
# PASSO 5: Criar provisioning do Grafana
# ================================================================================
echo "📦 PASSO 5: Criando provisioning do Grafana..."

mkdir -p deploy/docker/grafana/datasources
mkdir -p deploy/docker/grafana/dashboards

cat > deploy/docker/grafana/datasources/datasources.yml << 'EOF'
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true
    editable: false

  - name: Loki
    type: loki
    access: proxy
    url: http://loki:3100
    editable: false

  - name: PostgreSQL
    type: postgres
    access: proxy
    url: postgres:5432
    database: finops
    user: finops
    secureJsonData:
      password: finops
    jsonData:
      sslmode: disable
    editable: false
EOF

cat > deploy/docker/grafana/dashboards/dashboards.yml << 'EOF'
apiVersion: 1

providers:
  - name: 'default'
    orgId: 1
    folder: ''
    type: file
    disableDeletion: false
    updateIntervalSeconds: 10
    allowUiUpdates: true
    options:
      path: /etc/grafana/provisioning/dashboards
EOF

echo "   ✅ Datasources criados: deploy/docker/grafana/datasources/datasources.yml"
echo "   ✅ Dashboards provider criado: deploy/docker/grafana/dashboards/dashboards.yml"
echo ""

# ================================================================================
# PASSO 6: Criar .env.example
# ================================================================================
echo "📦 PASSO 6: Criando .env.example..."

cat > .env.example << 'EOF'
# ============================================
# FINOPS PLATFORM - ENVIRONMENT VARIABLES
# ============================================

# Huawei OBS (opcional para DEV - deixe vazio se não usar)
HUAWEI_ACCESS_KEY=
HUAWEI_SECRET_KEY=
HUAWEI_OBS_ENDPOINT=

# JWT Secret (altere em produção)
JWT_SECRET=finops-enterprise-secret-key

# Database
POSTGRES_USER=finops
POSTGRES_PASSWORD=finops
POSTGRES_DB=finops

# ClickHouse
CLICKHOUSE_USER=default
CLICKHOUSE_PASSWORD=
CLICKHOUSE_DB=finops

# Grafana
GF_SECURITY_ADMIN_USER=admin
GF_SECURITY_ADMIN_PASSWORD=admin
EOF

if [ ! -f ".env" ]; then
    cp .env.example .env
    echo "   ✅ .env criado a partir do .env.example"
else
    echo "   ⚠️  .env já existe, não sobrescrito"
fi
echo ""

# ================================================================================
# PASSO 7: Verificar Dockerfiles dos serviços
# ================================================================================
echo "📦 PASSO 7: Verificando Dockerfiles dos serviços..."

for service in backend/api-gateway backend/ingestion-service backend/cost-analytics backend/alert-manager ml/forecast-engine ml/anomaly-detection ml/recommendation-engine; do
    dockerfile="$service/Dockerfile"
    if [ -f "$dockerfile" ]; then
        if grep -q "wget\|curl" "$dockerfile" 2>/dev/null; then
            echo "   ✅ $dockerfile - wget/curl encontrado"
        else
            echo "   ⚠️  $dockerfile - ATENÇÃO: wget/curl NÃO encontrado!"
            echo "      Healthchecks podem falhar. Use o template em templates/"
        fi
    else
        echo "   ❌ $dockerfile - NÃO ENCONTRADO"
    fi
done
echo ""

# ================================================================================
# PASSO 8: Resumo final
# ================================================================================
echo "================================================================================"
echo "✅ MIGRAÇÃO CONCLUÍDA!"
echo "================================================================================"
echo ""
echo "📁 Estrutura final na raiz:"
echo "   docker-compose.yml         ← Execute 'docker compose up -d --build' aqui"
echo "   .env                       ← Variáveis de ambiente"
echo ""
echo "🚀 Próximo passo:"
echo "   cd $(pwd)"
echo "   docker compose up -d --build"
echo ""
echo "📊 Acesse:"
echo "   Frontend:  http://localhost:3000"
echo "   API:       http://localhost:8080"
echo "   Grafana:   http://localhost:3001 (admin/admin)"
echo ""
