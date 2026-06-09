# FinOps Enterprise Platform

Plataforma FinOps moderna, escalável e pronta para produção.

## Stack

- **Frontend**: HTML5 + TailwindCSS + Apache ECharts (Preview)
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
# Docker Compose
cd deploy/docker
docker-compose up -d

# Ou Kubernetes + Helm
cd deploy/helm/finops-platform
helm install finops .
```

## Documentação

- [Arquitetura](docs/ARCHITECTURE.md)
- [Runbook Operacional](docs/RUNBOOK.md)
- [Guia do Administrador](docs/ADMIN_GUIDE.md)
- [UI Architecture](frontend/preview/UI-ARCHITECTURE.md)

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

## Licença

Enterprise Internal Use
