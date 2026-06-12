# FinOps Enterprise Platform

Plataforma FinOps moderna, escalável e pronta para produção.

## Stack

- **Frontend**: Next.js 14 + TailwindCSS + Apache ECharts
- **Backend**: Golang + Gin
- **ML**: Python + FastAPI
- **Analytics**: ClickHouse
- **Metadata**: PostgreSQL
- **Cache**: Redis
- **Messaging**: Kafka
- **Observability**: OpenTelemetry + Prometheus + Grafana + Loki
- **Deploy**: Docker + Kubernetes + Helm

## Quick Start

```bash
# Docker Compose (recomendado para desenvolvimento/local)
docker compose up -d --build

# Ou Kubernetes
make deploy-k8s

# Ou Helm
make deploy-helm
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

## Serviços

| Serviço | Porta | Descrição |
|---------|-------|-----------|
| API Gateway | 8080 | Auth, Rate Limit, Routing |
| Ingestion | 8081 | OBS, FOCUS processing |
| Cost Analytics | 8082 | KPIs, Aggregations |
| Forecast Engine | 8001 | Prophet, ARIMA, Holt-Winters |
| Anomaly Detection | 8002 | Isolation Forest, Z-Score |
| Recommendation | 8003 | Rightsizing, Savings |
| Alert Manager | 8083 | Notifications |
| Frontend | 3000 | Dashboards |

## Health Checks

Todos os serviços expõem:
- `/health` - Health check
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
