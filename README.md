# FinOps Enterprise Platform

Plataforma FinOps para gestão financeira de cloud **Huawei-only**, consumindo exportações FOCUS 1.0 do OBS e exibindo dashboards dinâmicos em tempo real.

> **Status:** Pré-produção. Veja `ROADMAP.md` para itens pendentes.

## Stack

- **Frontend**: Next.js 14 + TailwindCSS + Apache ECharts
- **Backend**: Golang + Gin
- **ML**: Python + FastAPI (em standby)
- **Analytics**: ClickHouse
- **Metadata**: PostgreSQL
- **Cache**: Redis
- **Messaging**: Kafka
- **Observability**: Prometheus + Grafana + Loki
- **Deploy**: Docker + Kubernetes + Helm

## Quick Start

```bash
# Clone
git clone <repo-url> finops-platform
cd finops-platform

# Configure credenciais (obrigatório para ingestão real)
cp .env.example .env
# Edite .env com HUAWEI_ACCESS_KEY, HUAWEI_SECRET_KEY e HUAWEI_OBS_ENDPOINT

# Docker Compose (desenvolvimento)
docker compose up -d --build
```

Após subir a stack:
- Frontend: http://localhost:3000
- API Gateway: http://localhost:8080
- Grafana: http://localhost:3001 (admin/admin)
- Prometheus: http://localhost:9090

Login de demo: `admin@finops.local` / `admin123`

## Documentação

- [Arquitetura](docs/ARCHITECTURE.md)
- [Runbook Operacional](docs/RUNBOOK.md)
- [Guia do Administrador](docs/ADMIN_GUIDE.md)
- [Guia de Deploy](docs/DEPLOY_GUIDE.md)
- [Troubleshooting](docs/TROUBLESHOOTING.md)
- [UI Architecture](docs/UI-ARCHITECTURE.md)
- [Roadmap](ROADMAP.md)
- [Changelog](CHANGELOG.md)

## Serviços

| Serviço | Porta | Descrição | Status |
|---------|-------|-----------|--------|
| API Gateway | 8080 | Auth, Rate Limit, Routing, Proxy | ✅ Ativo |
| Ingestion | 8081 | OBS Huawei, FOCUS processing | ✅ Ativo |
| Cost Analytics | 8082 | KPIs, Forecast, Anomalies, Recommendations | ✅ Ativo |
| Alert Manager | 8083 | Notifications backend | ⚠️ Integração futura |
| Forecast Engine | 8001 | Prophet, ARIMA, Holt-Winters | ⏸️ Standby |
| Anomaly Detection | 8002 | Isolation Forest, Z-Score | ⏸️ Standby |
| Recommendation | 8003 | Rightsizing, Savings | ⏸️ Standby |
| Frontend | 3000 | Dashboards | ✅ Ativo |

> Forecast, anomalias e recomendações são calculados pelo `cost-analytics` a partir de dados reais do ClickHouse.

## Health Checks

Todos os serviços expõem:
- `/health` - Health check (GET e HEAD)
- `/ready` - Readiness probe
- `/live` - Liveness probe
- `/metrics` - Prometheus metrics

## Variáveis de Ambiente

Copie `.env.example` para `.env` e **preencha obrigatoriamente** as credenciais Huawei OBS:

```bash
cp .env.example .env
```

> ⚠️ **A integração com Huawei OBS é mandatória.** O `ingestion-service` não sobe sem `HUAWEI_ACCESS_KEY` e `HUAWEI_SECRET_KEY` configurados. A OBS é acessada via cliente S3 compatível (`minio-go/v7`).

## Licença

Enterprise Internal Use
