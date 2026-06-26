# Handoff Context - FinOps Platform

> Arquivo gerado para continuidade entre sessões/máquinas.
> Última atualização: 2026-06-26

## Objetivo Atual

Dashboard FinOps funcional com dados reais, sem duplicatas e alinhado ao máximo possível com o Cost Center da Huawei Cloud.

## Ambiente

- **VM alvo**: `az-vm-ubu-finops-platform-v2` (acessível apenas via jumphost)
- **IP frontend**: `http://10.140.12.32:3000`
- **Stack**: Docker Compose (PostgreSQL, ClickHouse, Redis, Kafka, Zookeeper, Prometheus, Grafana, Loki)
- **Serviços principais**: `finops-api-gateway` (8080), `finops-cost-analytics` (8082), `finops-ingestion` (8081), `finops-frontend` (3000)
- **Bucket Huawei OBS real**: `focusfinops`, prefixo `daily-exports/Daily_Cost_Export_Focus1-0/`
- **Login demo**: `admin@finops.local` / `admin123`
- **Taxa USD/BRL**: `USD_TO_BRL_RATE=5.15`

## Código-fonte

- Repositório clonado em `/finops-platform` na VM alvo.
- Branch: `Teste-Docker-Compose`
- Diretório local Windows: `c:\finops-platform`

## Alterações Realizadas e Validadas

### Backend `backend/cost-analytics/main.go`
- **Batch insert no Kafka consumer**: `consumeKafka` acumula 500 registros e insere via `PrepareBatch` no ClickHouse.
- **Commit de offsets em batch**: offsets comitados apenas após o batch inserido com sucesso.
- **Correção de parsing de datas**: `getTrends`, `getAnomalies` e `getForecast` escaneiam `date` como `time.Time`.
- **Conversão USD/BRL completa**: `getCosts` converte `effective_cost`, `amortized_cost` e `list_cost` para BRL.
- **Normalização de tags**: tags inseridos no ClickHouse com ordem fixa (`application`, `business_unit`, `environment`).
- **Arredondamento BRL determinístico**: colunas `*_brl` arredondadas para 2 casas decimais no insert, evitando diferenças de ponto flutuante que quebravam a deduplicação.
- Null-safety em agregações com ponteiros e `derefFloat`/`derefUint64`.
- Retorno de arrays vazios em vez de `nil`/erro 500 quando não há dados.
- KPIs, forecast, anomalies e recommendations baseados em dados reais.

### Backend `backend/ingestion-service/main.go`
- Lock global (`processMu`) em `processIngestion` para evitar ingestões concorrentes.

### Frontend
- Rebuild com `NEXT_PUBLIC_API_URL=http://10.140.12.32:8080`.
- `/login` e `/dashboard` carregam sem erro.

### Scripts
- `scripts/dedup_clickhouse.sql`: deduplica ignorando `tags` e colunas `*_brl`, reconstruindo ambas de forma determinística.

## Estado Atual na VM Alvo

Última verificação (2026-06-26 ~02:20 UTC):
- `costs_raw`: `count() = 233.854`, `uniqExact(*) = 233.854` (sem duplicatas).
- Kafka consumer group `cost-analytics`: `LAG = 0`.
- Todos os containers healthy.

### Comparação com Cost Center Huawei Cloud

| Métrica | Console Huawei | FinOps Dashboard | Diferença |
|---|---|---|---|
| Maio 2026 | $1.100.466,97 | $1.082.512,05 | ~1,6% |
| Junho 2026 (MTD) | $804.364,68 | $689.144,69 | ~14,3% |
| Maio + Junho MTD | ~$1.904.831 | $1.771.656,74 | ~7,0% |

A maior parte da diferença em junho parece ser dados incompletos nos arquivos FOCUS do bucket (especialmente a partir do dia 22/06, onde os valores caem drasticamente). Maio está muito próximo da console.

## Validação Final

- ✅ Redeploy do zero concluído.
- ✅ Dados reais ingeridos do bucket Huawei OBS.
- ✅ Duplicatas eliminadas (de ~3M para 233.854 registros).
- ✅ Dashboard mostra dados reais.
- ✅ USD/BRL alterna corretamente.
- ✅ Budgets e Usuários não crasham.
- ✅ Forecast/Anomalias/Recomendações baseados em dados reais.

## Comandos de Diagnóstico Úteis

```bash
# Contagem no ClickHouse
sudo docker exec finops-clickhouse clickhouse-client --database=finops -q "SELECT count(), uniqExact(*) FROM costs_raw"

# Custos por mês
sudo docker exec finops-clickhouse clickhouse-client --database=finops -q "SELECT toYYYYMM(date) as month, sum(effective_cost) FROM costs_raw GROUP BY month ORDER BY month"

# Logs
sudo docker logs -f finops-ingestion
sudo docker logs -f finops-cost-analytics

# Consumer group Kafka
sudo docker exec finops-kafka /bin/kafka-consumer-groups --bootstrap-server localhost:9092 --describe --group cost-analytics

# Health
curl http://localhost:8080/health
curl http://localhost:8081/health
```

## Próximos Passos Recomendados

1. **Monitorar estabilidade**: verificar se `count()` continua igual a `uniqExact(*)` nas próximas horas/dias.
2. **Dados incompletos de junho**: investigar por que os arquivos FOCUS do bucket têm dados parciais a partir de ~22/06. Possíveis causas:
   - Arquivos ainda não foram gerados/enviados pela Huawei.
   - Ingestion-service parou de processar antes do último arquivo.
   - Prefixo do bucket precisa ser ajustado.
3. **Deduplicação automática**: implementar no ingestion-service o processamento apenas do arquivo mais recente de cada mês, ou migrar `costs_raw` para `ReplacingMergeTree`.
4. **Alinhamento com console**: verificar se a console usa `effective_cost`, `list_cost` ou outra métrica; confirmar se impostos/taxas estão incluídos nos arquivos FOCUS.

## Contato / Decisões Pendentes

- Dashboard acessível em `http://10.140.12.32:3000`.
- Login demo funcional.
- Discrepância de ~7% em maio+junho precisa de validação com o time de dados/financeiro da Huawei.
