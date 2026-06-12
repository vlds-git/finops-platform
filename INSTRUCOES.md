# 🚀 FinOps Platform - Aplicação de Patches

## ⚠️ IMPORTANTE: Não substitua os arquivos originais!

Este pacote contém **apenas correções e novos arquivos**. O código original da aplicação (frontend, backend, ML) deve permanecer intacto.

## 📁 O que está neste pacote

### Scripts de Patch (aplicam correções nos arquivos EXISTENTES)
- `apply-patches.py` + `apply-patches.sh` — Corrige Dockerfiles e erros de sintaxe Go
- `patch-ingestion-zip-brl.py` + `patch-ingestion-zip-brl.sh` — Adiciona ZIP unzip + USD→BRL no ingestion-service
- `patch-python-usd-brl.py` + `patch-python-usd-brl.sh` — Adiciona USD→BRL nos serviços Python

### Novos arquivos (copiar para o repositório)
- `docker-compose.yml` — Na raiz do projeto
- `.env.example` — Na raiz do projeto
- `database/postgresql/migrations/001_init.sql` — Schema PostgreSQL
- `database/clickhouse/schema.sql` — Schema ClickHouse (sintaxe corrigida)
- `deploy/docker/prometheus.yml` — Config Prometheus
- `deploy/docker/loki.yml` — Config Loki
- `deploy/docker/grafana/datasources/datasources.yml` — Grafana datasources
- `deploy/docker/grafana/dashboards/dashboards.yml` — Grafana dashboards
- `frontend-next.config.js` — Next.js com `output: 'standalone'`

## 🚀 Como usar (3 passos)

### Passo 1: Clone o repositório original
```bash
git clone https://github.com/vlds-git/finops-platform.git
cd finops-platform
```

### Passo 2: Execute os scripts de patch
```bash
# Extrair o ZIP e entrar na pasta
# Copiar os scripts para a raiz do repositório
cp finops-patches-final/*.py finops-patches-final/*.sh ./

# Dar permissão
chmod +x *.sh

# Aplicar todas as correções
./apply-patches.sh           # Corrige Dockerfiles + erros Go
./patch-ingestion-zip-brl.sh # Adiciona ZIP + USD→BRL
./patch-python-usd-brl.sh    # Adiciona USD→BRL nos Python
```

### Passo 3: Copiar novos arquivos
```bash
# Copiar para a raiz do repositório
cp finops-patches-final/docker-compose.yml finops-patches-final/.env.example ./

# Copiar schemas de banco
mkdir -p database/postgresql/migrations database/clickhouse
cp finops-patches-final/database/postgresql/migrations/001_init.sql database/postgresql/migrations/
cp finops-patches-final/database/clickhouse/schema.sql database/clickhouse/

# Copiar configs de observability
mkdir -p deploy/docker/grafana/datasources deploy/docker/grafana/dashboards
cp finops-patches-final/deploy/docker/prometheus.yml deploy/docker/
cp finops-patches-final/deploy/docker/loki.yml deploy/docker/
cp finops-patches-final/deploy/docker/grafana/datasources/datasources.yml deploy/docker/grafana/datasources/
cp finops-patches-final/deploy/docker/grafana/dashboards/dashboards.yml deploy/docker/grafana/dashboards/

# Substituir next.config.js (apenas este arquivo!)
cp finops-patches-final/frontend-next.config.js frontend/nextjs/next.config.js
```

### Passo 4: Commit e deploy
```bash
git add .
git commit -m "fix: deploy docker-compose + feat: USD-BRL + ZIP unzip"
git push origin main

# Na VM
docker compose up -d --build
```

## 🆕 Features adicionadas

| Feature | Onde está implementada |
|---------|------------------------|
| 💱 USD → BRL | `docker-compose.yml`, `ingestion-service`, `cost-analytics`, `forecast-engine`, `anomaly-detection`, `recommendation-engine` |
| 📦 ZIP Unzip | `ingestion-service/main.go` + `Dockerfile` (adicionado `unzip`) |
| 🔧 Correções Go | `api-gateway`, `alert-manager`, `cost-analytics`, `ingestion-service` (strings multilinhas, imports) |
| 🐳 Docker Compose | `docker-compose.yml` na raiz + configs observability |

## ⚠️ Arquivos que NÃO foram modificados

Todo o código original permanece intacto:
- ✅ Frontend Next.js (app/, components/, hooks/, lib/, types/)
- ✅ Testes (main_test.go, test_main.py)
- ✅ Lógica de negócio completa dos serviços
- ✅ Integrações com ClickHouse, Kafka, etc.
