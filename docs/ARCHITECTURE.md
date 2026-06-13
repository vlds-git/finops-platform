# Arquitetura FinOps Enterprise

## Visão Geral

Plataforma modular para gestão financeira de cloud, atualmente operando em ambiente **Huawei-only** via exportações FOCUS 1.0 armazenadas no OBS.

## Diagrama de Arquitetura

```mermaid
graph TB
    subgraph "Frontend"
        FE[Next.js 14 - 6 páginas]
    end

    subgraph "API Gateway"
        GW[Gin + JWT + RBAC + Rate Limit]
    end

    subgraph "Core Services"
        ING[Ingestion Service]
        CA[Cost Analytics]
        AL[Alert Manager]
    end

    subgraph "ML Engines (futuro)"
        FC[Forecast Engine Python]
        AN[Anomaly Detection Python]
        REC[Recommendation Engine Python]
    end

    subgraph "Data Layer"
        CH[(ClickHouse)]
        PG[(PostgreSQL)]
        RD[(Redis)]
        KF[(Kafka)]
    end

    subgraph "External"
        OBS[Huawei OBS]
    end

    FE --> GW
    GW --> ING
    GW --> CA
    GW --> AL

    ING --> OBS
    ING --> KF
    KF --> CA

    CA --> CH
    CA --> PG
    CA -.->|forecast / anomalies / recommendations| CH
    AL --> PG
    AL --> KF
    GW --> PG
    GW --> RD

    FC -.->|em standby| CH
    AN -.->|em standby| CH
    REC -.->|em standby| CH
```

## Fluxo FOCUS

```mermaid
sequenceDiagram
    participant OBS as Huawei OBS
    participant ING as Ingestion
    participant KF as Kafka
    participant CA as Cost Analytics
    participant CH as ClickHouse

    OBS->>ING: ZIP/CSV FOCUS 1.0 Export
    ING->>ING: Download Incremental
    ING->>ING: Parse CSV (PascalCase -> snake_case)
    ING->>ING: Normalização e conversão USD -> BRL
    ING->>KF: cost.raw
    KF->>CA: Consume
    CA->>CH: Insert costs_raw
    CA->>CH: Aggregate costs_daily
```

## Fluxo OBS

```mermaid
sequenceDiagram
    participant CRON as Scheduler
    participant ING as Ingestion
    participant OBS as Huawei OBS
    participant CHK as Checkpoint DB
    participant DLQ as Dead Letter Queue

    CRON->>ING: Trigger Sync
    ING->>CHK: Load Last Checkpoint
    ING->>OBS: List Objects (since checkpoint)
    OBS-->>ING: Object List
    loop Parallel Download
        ING->>OBS: Get Object
        OBS-->>ING: File Data
        ING->>ING: Validate & Parse
        alt Success
            ING->>CHK: Update Checkpoint
        else Failure
            ING->>DLQ: Send to Retry
        end
    end
```

## Fluxo Kafka

```mermaid
graph LR
    P1[Ingestion] --> T1[cost.raw]
    T1 --> C1[Cost Analytics]
    T1 --> C2[Forecast]
    T1 --> C3[Anomaly]
    T1 --> C4[Recommendation]
    C1 --> T2[cost.aggregated]
    C2 --> T3[forecast.results]
    C3 --> T4[anomaly.alerts]
    C4 --> T5[recommendations]
```

## Fluxo Forecast (atual)

```mermaid
graph TB
    subgraph "Input"
        CH[(ClickHouse)]
    end
    subgraph "Cost Analytics"
        Q[Query custos diários]
        L[Tendência linear]
    end
    subgraph "Output"
        FE[Frontend / API]
    end
    CH --> Q
    Q --> L
    L --> FE
```

> A previsão utiliza regressão linear sobre a série temporal de custos reais. Modelos avançados (Prophet, ARIMA) estão no roadmap para integração futura via serviços Python.

## Fluxo Alertas (futuro)

```mermaid
graph TB
    B[Budget Monitor] --> K[Kafka budget.alerts]
    K --> AM[Alert Manager]
    AM --> E[Email]
    AM --> S[Slack]
    AM --> W[Webhook]
    AM --> PG[(PostgreSQL)]
```

> A página de Alertas foi removida do frontend. A funcionalidade será reintroduzida vinculada aos budgets: quando `spent > amount * alert_threshold`, um evento será publicado e notificado.

## Justificativa de Componentes

| Componente | Justificativa |
|------------|---------------|
| API Gateway | Ponto único de entrada. JWT, RBAC, Rate Limit, Auditoria, proxy para serviços |
| Ingestion Service | Isolamento do processamento de dados brutos FOCUS. Resiliência com checkpoint e DLQ |
| Cost Analytics | Separação de concerns. Processamento pesado em ClickHouse; consome Kafka; expõe forecast, anomalies e recommendations |
| Forecast Engine (Python) | Reservado para modelos avançados (Prophet, ARIMA). Em standby até integração com ClickHouse |
| Anomaly Detection (Python) | Reservado para algoritmos avançados (Isolation Forest). Em standby |
| Recommendation Engine (Python) | Reservado para análise de padrões de uso. Em standby |
| Alert Manager | Desacoplamento de notificações. Múltiplos canais. Será integrado aos budgets |
| ClickHouse | OLAP otimizado para analytics financeiro. Compressão, partições, TTL |
| PostgreSQL | ACID para metadata, configs, usuários, budgets, accounts |
| Kafka | Buffer entre ingestão e processamento. Backpressure handling |
| Redis | Cache de sessões, rate limiting, configs |

## Decisões de Arquitetura

1. **Microserviços controlados**: 7 serviços definidos, mas a lógica de forecast/anomalies/recommendations foi consolidada no `cost-analytics` para a pré-produção, reduzindo complexidade operacional enquanto os modelos Python não estão integrados.
2. **Huawei-first**: remoção deliberada de páginas e lógicas multi-cloud (Azure/AWS) para refletir o ambiente real de dados.
3. **Polyglot persistence**: ClickHouse para analytics, PostgreSQL para transacional.
4. **Event-driven**: Kafka como backbone entre ingestão e cost-analytics.
5. **Observability-first**: health checks, logs estruturados e endpoints `/metrics` em todos os serviços. Stack Prometheus/Grafana/Loki disponível no Docker Compose.
6. **Security by default**: JWT, RBAC, input validation, security headers. Em andamento: hash de senhas, login contra PostgreSQL e gestão segura de secrets (ver `ROADMAP.md`).
