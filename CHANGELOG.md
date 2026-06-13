# Changelog - FinOps Enterprise Platform

Todas as mudanças notáveis neste projeto serão documentadas neste arquivo.

O formato é baseado em [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
e este projeto adere ao [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Período customizável** no header: 24h, 48h, 7d, 30d, 90d e intervalo personalizado.
- **Dashboard Executivo dinâmico**: KPIs reais, variação vs período anterior, custos por serviço e por região.
- **Forecast real** via `cost-analytics` usando tendência linear sobre dados do ClickHouse.
- **Anomalias reais** via `cost-analytics` usando Z-Score sobre custos diários.
- **Recomendações reais** via `cost-analytics` baseadas nos serviços de maior custo.
- **Budgets vinculados a accounts** com cálculo de spent no período do budget e alertas por threshold.
- **CRUD completo de usuários** (PUT/DELETE) e integração frontend com backend.
- **Endpoint `/accounts`** para listar contas cloud do PostgreSQL.
- **HEAD /health** e `/metrics` básicos nos serviços Go para eliminar 404s de healthchecks.
- **Helm Chart aprimorado**: credenciais ClickHouse e Huawei injetadas via Secret, job de migrations.

### Changed
- **Sidebar simplificada**: removidos Alertas, Multi-Cloud e Dashboard Operacional.
- **Filtro de provider** substituído por título "Huawei Cloud" (ambiente Huawei-only).
- **Conversão USD/BRL** via query param `currency` respeitada pelos endpoints de custos.
- **API Gateway**: `/forecast`, `/anomalies` e `/recommendations` roteados para `cost-analytics` (dados reais).
- **Backend `cost-analytics`**: todos os endpoints de custos respeitam `start_date`/`end_date` e `currency`.

### Removed
- Páginas frontend removidas: Alertas, Multi-Cloud, Dashboard Operacional.
- Gráficos estáticos do Dashboard Executivo: Top Aplicações e Custos por Provedor.
- Dependência dos serviços ML mockados para forecast, anomalias e recomendações.

### Fixed
- Erro `ClickHouse insert error: code: 16` resolvido com colunas BRL no schema.
- 404s de `/metrics` e `HEAD /health` nos logs dos serviços.

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
