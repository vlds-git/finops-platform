# Changelog - FinOps Enterprise Platform

Todas as mudanças notáveis neste projeto serão documentadas neste arquivo.

O formato é baseado em [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
e este projeto adere ao [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2024-01-15

### Added
- **API Gateway** com JWT, RBAC, Rate Limiting, CORS, Security Headers e Audit Logging
- **Ingestion Service** com integração Huawei OBS, download incremental, checkpoints, DLQ, reprocessamento
- **Cost Analytics Engine** com agregações ClickHouse, KPIs FinOps, dashboards executivo e operacional
- **Forecast Engine** com Prophet, ARIMA, Holt-Winters e Ensemble (30d, 90d, 12m)
- **Anomaly Detection Engine** com Isolation Forest, Z-Score, Rolling Average e Ensemble
- **Recommendation Engine** com Rightsizing, Idle Resources, Storage Optimization e Savings Opportunities
- **Alert Manager** com Slack, Email, Webhook e histórico de alertas
- **Frontend Protótipo** com 9 telas navegáveis, Dark Mode, Apache ECharts
- **PostgreSQL Schema** com 12 tabelas, índices, triggers, seed data
- **ClickHouse Schema** com partições, Materialized Views, TTL, índices bloom_filter
- **Docker Compose** stack completa com 11 serviços
- **Helm Chart** com templates, values, secrets, ingress
- **Kubernetes Manifests** com deployments, services, ingress, HPA, configmaps, secrets
- **Terraform** para OBS, IAM, Load Balancer
- **Observabilidade** com Prometheus, Grafana, Loki
- **Documentação** completa: Architecture, Runbook, Admin Guide, Deploy Guide, Troubleshooting, UI Architecture
- **Testes** unitários para todos os 7 serviços
- **Scripts** de build, deploy e diagnose
- **Makefile** com comandos padronizados

### Security
- JWT authentication com expiração de 24h
- RBAC com 3 perfis (Admin, Analyst, Viewer)
- Rate limiting por usuário/IP
- Security headers (CSP, HSTS, X-Frame-Options, etc.)
- Input validation em todos os endpoints
- SQL Injection protection via prepared statements
- XSS protection via headers
- SSRF protection
- Audit logging de todas as requisições

### Infrastructure
- Health checks (/health, /ready, /live, /metrics) em todos os serviços
- OpenTelemetry tracing support
- Structured JSON logging
- Kafka topics: cost.raw, anomaly.alerts, forecast.results, recommendations
- Redis para cache de sessões e rate limiting
- PostgreSQL connection pooling
- ClickHouse query optimization

## [Unreleased]

### Fixed
- Correção de erros de compilação nos serviços Go (`ingestion-service`, `alert-manager`)
- Recriação dos arquivos `go.mod` com encoding limpo
- Implementação de proxy reverso real no `api-gateway`
- Correção do build do frontend Next.js (`QueryClientProvider`, rota raiz, Dockerfile, TS)
- Aplicação da conversão USD→BRL nos serviços ML
- Alinhamento do `docker-compose.yml`, `Makefile` e scripts de deploy
- Completude do Helm chart com todos os serviços
- Adição de manifests de infraestrutura no Kubernetes
- Configuração de healthchecks e observability

### Planned
- Integração Microsoft Entra ID / LDAP / Keycloak
- Multi-tenant support
- Cost allocation tagging automático
- ML model retraining pipeline
- Custom dashboard builder
- API versioning v2
- Mobile app companion
