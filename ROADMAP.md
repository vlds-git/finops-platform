# Roadmap - FinOps Enterprise Platform

Este documento consolida as melhorias identificadas durante o ciclo de pré-produção e organiza o trabalho futuro em fases. A ordem reflete prioridade para deixar a plataforma pronta para produção em ambiente Huawei-only.

## Legenda

- **P0** — Bloqueante para produção (segurança, estabilidade, dados).
- **P1** — Alto valor operacional, deve ser feito antes do go-live amplo.
- **P2** — Melhorias significativas de UX/performance, podem vir em release seguinte.
- **P3** — Estratégico/técnico, para evolução contínua.

---

## Fase 1 — Segurança & Compliance (P0)

| # | Item | Motivação | Escopo Técnico |
|---|------|-----------|----------------|
| 1.1 | **Hash de senhas com bcrypt/Argon2** | Hoje senhas são armazenadas em plaintext no PostgreSQL (`password_hash`). | Alterar `createUser`/`updateUser` no `cost-analytics` para hashear senha; manter compatibilidade com login do gateway migrando para validação no banco. |
| 1.2 | **Login do gateway contra PostgreSQL** | Login hardcoded (`admin@finops.local` / `admin123`) não escala. | `api-gateway` deve consultar tabela `users`, validar hash e gerar JWT com roles do banco. |
| 1.3 | **RBAC persistente** | Roles e permissões estão hardcoded no `api-gateway`. | Mover definição de roles/permissões para PostgreSQL; gateway carrega ao iniciar ou consulta por requisição. |
| 1.4 | **Validação de entrada e queries parametrizadas** | `cost-analytics` monta queries com `fmt.Sprintf` e `dateWhereClause`. | Validar/formatar `start_date`/`end_date`; usar placeholders do driver ClickHouse para todas as variáveis de filtro. |
| 1.5 | **CORS configurável para produção** | Atualmente `Access-Control-Allow-Origin: *`. | Tornar origens permitidas uma variável de ambiente; rejeitar origens não autorizadas. |
| 1.6 | **Rotação e gerenciamento seguro de secrets** | Credenciais Huawei OBS e JWT estão em `.env` / `values.yaml`. | Adotar External Secrets Operator, Vault ou Sealed Secrets; nunca commitar credenciais. |

---

## Fase 2 — Infraestrutura & Deploy (P0/P1)

| # | Item | Motivação | Escopo Técnico |
|---|------|-----------|----------------|
| 2.1 | **TLS no Ingress via cert-manager** | Helm expõe apenas HTTP. | Adicionar annotations do cert-manager e seção `tls` no `ingress.yaml`; documentar domínios de produção. |
| 2.2 | **ServiceAccount, SecurityContext e runAsNonRoot** | Containers rodam com privilégios amplos. | Criar SA por serviço, definir `runAsNonRoot: true`, `readOnlyRootFilesystem` onde possível, limitar capabilities. |
| 2.3 | **NetworkPolicies** | Ausência de isolamento de rede entre pods. | Permitir tráfego apenas entre serviços necessários; bloquear egress/ingress padrão. |
| 2.4 | **PodDisruptionBudgets** | Garantir disponibilidade durante atualizações/nós. | Adicionar PDB com `minAvailable: 1` para gateway, cost-analytics, ingestion. |
| 2.5 | **Versionamento de imagens** | Todas as imagens usam `latest`. | Trocar tags por versão semver (`v1.2.3`) e atualizar CI para taggear builds. |
| 2.6 | **Job de backups** | Sem estratégia de backup para PostgreSQL e ClickHouse. | Criar CronJobs para dump do PostgreSQL e export de partições ClickHouse para storage object (OBS/S3). |
| 2.7 | **CI/CD e GitOps** | Deploy manual sujeito a erros. | Criar pipelines (GitHub Actions/GitLab CI) para build, teste, scan e deploy; adotar ArgoCD/Flux para GitOps. |
| 2.8 | **Health checks e observabilidade no K8s** | Prometheus/Grafana/Loki só existem no Docker Compose. | Adicionar ServiceMonitors, PodLogs e dashboards no Helm; integrar com stack observabilidade do cluster. |
| 2.9 | **Migrações idempotentes** | Job atual roda schemas inteiros a cada upgrade. | Adicionar controle de versão de migrations (ex: `golang-migrate` ou `flyway`) para rodar apenas scripts pendentes. |

---

## Fase 3 — Funcionalidades de Negócio (P1)

| # | Item | Motivação | Escopo Técnico |
|---|------|-----------|----------------|
| 3.1 | **Alertas vinculados a budgets** | Botão de alertas foi removido; a funcionalidade deve emergir dos budgets. | Quando `spent > amount * alert_threshold`, publicar evento no Kafka/alert-manager e exibir badge/notificação no frontend. |
| 3.2 | **Tags reais a partir do CSV Huawei** | Hoje tags são padrão (`environment: production`, etc.) porque o campo `Tags` do CSV está vazio. | Investigar colunas de tags no FOCUS 1.0 da Huawei; mapear corretamente ou extrair de `ResourceName`/`ResourceId`. |
| 3.3 | **Custos por account na sidebar/dashboard** | Usuário precisa navegar/visualizar custos filtrados por conta. | Adicionar seletor de account no Header (opcional) e filtrar todos os endpoints por `account_id`. |
| 3.4 | **Detalhamento drill-down** | Dashboard mostra agregações, mas não permite explorar detalhes. | Criar página de detalhe por serviço/região/recurso com tabela de custos brutos. |
| 3.5 | **Export de relatórios** | Não há como exportar dados. | Adicionar botão "Exportar CSV/PDF" nos dashboards e tabelas. |
| 3.6 | **Parquet ingestion** | FOCUS pode vir em Parquet; hoje há fallback limitado. | Implementar parser Parquet real no `ingestion-service`. |
| 3.7 | **Reprocessamento seletivo** | Reprocessar apenas um período/account específico. | Melhorar endpoint `/ingestion/reprocess` para aceitar filtros de data/account. |

---

## Fase 4 — ML & Analytics Avançado (P2/P3)

| # | Item | Motivação | Escopo Técnico |
|---|------|-----------|----------------|
| 4.1 | **Reintroduzir Forecast Engine Python integrado** | Modelo linear é simples; modelos estatísticos (Prophet/ARIMA) trazem mais precisão. | Conectar `ml/forecast-engine` ao ClickHouse, treinar periodicamente e persistir na tabela `forecasts`. |
| 4.2 | **Reintroduzir Anomaly Detection Python integrado** | Z-Score é básico; Isolation Forest captura padrões mais complexos. | Consumir série temporal do ClickHouse, detectar anomalias e persistir na tabela `anomalies`. |
| 4.3 | **Recommendation Engine baseado em regras avançadas** | Recomendações atuais são heurísticas simples. | Identificar recursos ociosos, padrões de uso, Savings Plans/Reserved Instances viáveis. |
| 4.4 | **Materialized Views no ClickHouse** | Agregações são calculadas on-the-fly. | Criar MVs para custos diários, mensais e por dimensão; reduzir latência dos dashboards. |
| 4.5 | **Budget forecast e what-if** | Permitir simular impacto de budgets. | Adicionar endpoint que cruza forecast com budgets e projeta estouro. |

---

## Fase 5 — Qualidade & Manutenção (P1/P2)

| # | Item | Motivação | Escopo Técnico |
|---|------|-----------|----------------|
| 5.1 | **Testes automatizados** | Plataforma sem testes aumenta risco de regressão. | Adicionar testes unitários no backend Go, testes de API (Postman/newman), testes de componente no frontend. |
| 5.2 | **Lint e formatação padronizada** | Código mistura estilos. | Adotar `gofmt`, `golangci-lint`, ESLint/Prettier; adicionar ao CI. |
| 5.3 | **Documentação viva** | Docs desatualizam rápido. | Sincronizar `ARCHITECTURE.md`, `ADMIN_GUIDE.md`, `DEPLOY_GUIDE.md`, `UI-ARCHITECTURE.md`, `MANIFEST.md` a cada release. |
| 5.4 | **Métricas de negócio** | Saber quanto a plataforma economizou. | Criar KPIs reais de savings, budget health, tag coverage a partir dos dados. |
| 5.5 | **Rate limiting distribuído** | Hoje é local por instância. | Mover para Redis para rate limit consistente em múltiplas réplicas. |

---

## Critérios de Saída por Fase

| Fase | Critério |
|------|----------|
| 1 | Nenhuma senha em plaintext; login via DB; CORS restrito; secrets fora do repo. |
| 2 | Helm roda em cluster de produção com TLS, SA, NetworkPolicy, PDB, backup e observabilidade. |
| 3 | Usuário consegue criar budgets com alertas, filtrar por account e exportar relatórios. |
| 4 | ML Python lê do ClickHouse e complementa (não substitui) as análises do cost-analytics. |
| 5 | CI bloqueia merge com falhas de lint/teste; docs atualizados automaticamente. |

---

## Notas

- Itens marcados como **P0** devem ser concluídos antes de qualquer deploy em produção real.
- O ambiente atual é **Huawei-only**, então Multi-Cloud foi deliberadamente removido do escopo.
- Alertas serão reintroduzidos de forma integrada aos budgets, não como página isolada.
