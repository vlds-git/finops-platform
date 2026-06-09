# Arquitetura FinOps Enterprise

## Visão Geral

Plataforma modular para gestão financeira de cloud multi-provedor.

## Diagrama de Arquitetura

```mermaid
graph TB
    subgraph "Frontend"
        FE[Next.js / HTML Preview]
    end

    subgraph "API Gateway"
        GW[Gin + JWT + RBAC]
    end

    subgraph "Core Services"
        ING[Ingestion Service]
        CA[Cost Analytics]
        AL[Alert Manager]
    end

    subgraph "ML Engines"
        FC[Forecast Engine]
        AN[Anomaly Detection]
        REC[Recommendation Engine]
    end

    subgraph "Data Layer"
        CH[(ClickHouse)]
        PG[(PostgreSQL)]
        RD[(Redis)]
        KF[(Kafka)]
    end

    subgraph "External"
        OBS[Huawei OBS]
        AZ[Azure]
        AWS[AWS]
    end

    FE --> GW
    GW --> ING
    GW --> CA
    GW --> AL
    GW --> FC
    GW --> AN
    GW --> REC

    ING --> OBS
    ING --> KF
    KF --> CA
    KF --> FC
    KF --> AN
    KF --> REC

    CA --> CH
    CA --> PG
    FC --> CH
    AN --> CH
    REC --> CH
    AL --> PG
    AL --> RD
    GW --> PG
    GW --> RD
```

## Fluxo FOCUS

```mermaid
sequenceDiagram
    participant OBS as Huawei OBS
    participant ING as Ingestion
    participant KF as Kafka
    participant CA as Cost Analytics
    participant CH as ClickHouse

    OBS->>ING: CSV/Parquet Export
    ING->>ING: Download Incremental
    ING->>ING: Parse FOCUS
    ING->>ING: Normalização Multi-Cloud
    ING->>KF: cost.events
    KF->>CA: Consume
    CA->>CH: Insert Partitions
    CA->>CH: Materialized Views
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

## Fluxo Forecast

```mermaid
graph TB
    subgraph "Input"
        CH[(ClickHouse)]
    end
    subgraph "Forecast Engine"
        Q[Query Historical Data]
        P[Prophet]
        A[ARIMA]
        H[Holt-Winters]
        E[Ensemble]
    end
    subgraph "Output"
        PG[(PostgreSQL)]
        KF[Kafka]
    end
    CH --> Q
    Q --> P
    Q --> A
    Q --> H
    P --> E
    A --> E
    H --> E
    E --> PG
    E --> KF
```

## Fluxo Alertas

```mermaid
graph TB
    A[Anomaly Detection] --> K[Kafka anomaly.alerts]
    B[Budget Monitor] --> K
    C[Recommendation] --> K
    K --> AM[Alert Manager]
    AM --> E[Email]
    AM --> S[Slack]
    AM --> W[Webhook]
    AM --> PG[(PostgreSQL)]
```

## Justificativa de Componentes

| Componente | Justificativa |
|------------|---------------|
| API Gateway | Ponto único de entrada. JWT, RBAC, Rate Limit, Auditoria |
| Ingestion Service | Isolamento do processamento de dados brutos. Resiliência com checkpoint e DLQ |
| Cost Analytics | Separação de concerns. Processamento pesado em ClickHouse |
| Forecast Engine | Python é padrão de mercado para séries temporais (Prophet, statsmodels) |
| Anomaly Detection | ML especializado. Isolation Forest para multivariado |
| Recommendation Engine | Regras + ML para rightsizing |
| Alert Manager | Desacoplamento de notificações. Múltiplos canais |
| ClickHouse | OLAP otimizado para analytics financeiro. Compressão, partições, MVs |
| PostgreSQL | ACID para metadata, configs, usuários |
| Kafka | Buffer entre ingestão e processamento. Backpressure handling |
| Redis | Cache de sessões, rate limiting, configs |

## Decisões de Arquitetura

1. **Microserviços controlados**: 7 serviços, cada um com responsabilidade única e justificável
2. **Polyglot persistence**: ClickHouse para analytics, PostgreSQL para transacional
3. **Event-driven**: Kafka como backbone para desacoplamento
4. **Observability-first**: OpenTelemetry em todos os serviços
5. **Security by default**: JWT, RBAC, input validation, security headers
