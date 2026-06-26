# Handoff Context - FinOps Platform

> Arquivo gerado para continuidade entre sessões/máquinas.
> Última atualização: 2026-06-26

## Objetivo Atual

Dashboard FinOps funcional com dados reais, sem duplicatas e com conversão USD/BRL correta.

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
- **Batch insert no Kafka consumer**: `consumeKafka` acumula 500 registros e insere via `PrepareBatch` no ClickHouse, aumentando o throughput de ~600 msg/s para ~3.000 msg/s.
- **Commit de offsets em batch**: offsets comitados apenas após o batch ser inserido com sucesso.
- **Correção de parsing de datas**: `getTrends`, `getAnomalies` e `getForecast` escaneiam a coluna `date` como `time.Time` e formatam para `YYYY-MM-DD` (antes retornavam `date: ""`).
- **Conversão USD/BRL completa**: `getCosts` agora converte `effective_cost`, `amortized_cost` e `list_cost` para BRL quando `currency=BRL`.
- Null-safety em todas as agregações (`sum`, `uniqExact`) usando ponteiros e `derefFloat`/`derefUint64`.
- Retorno de arrays vazios em vez de `nil`/erro 500 quando não há dados.
- KPIs reais: `Cost Efficiency = effective_cost / list_cost`; `Budget Utilization` baseado em dados reais.
- `potential_savings` no executive dashboard calculado como `list_cost - effective_cost`.
- `total_savings` em recommendations = soma real das economias das recomendações.
- `getBudgets` e `getUsers` tratam datas nulas e `last_login` nulo; retornam `[]` quando vazio.

### Backend `backend/ingestion-service/main.go`
- Lock global (`processMu`) em `processIngestion` para evitar ingestões concorrentes.

### Frontend
- `frontend/nextjs/contexts/CurrencyContext.tsx`: troca de moeda invalida queries de custo/dashboard.
- `frontend/nextjs/app/dashboard/page.tsx`: `useCostTrends` usa `startDate/endDate`; KPIs estratégicos calculados de dados reais.
- `frontend/nextjs/app/forecast/page.tsx`: `useCostTrends` corrigido para usar `startDate/endDate`.
- Rebuild do frontend com `NEXT_PUBLIC_API_URL=http://10.140.12.32:8080`.

### Scripts
- `scripts/dedup_clickhouse.sql`: deduplica `costs_raw` via `SELECT DISTINCT *` com troca atômica de tabelas.
- `scripts/dedup_clickhouse.sh`: para serviços, deduplica, reseta offsets do Kafka, reinicia.
- `scripts/full-redeploy.sh`: redeploy completo do zero (apaga volumes, rebuilda, aplica migrations).

### Documentação
- `docs/RUNBOOK.md`: seção de redeploy do zero e troubleshooting de duplicatas.
- `docs/ADMIN_GUIDE.md`: comandos úteis de ClickHouse e deduplicação.

## Estado Atual na VM Alvo

Última verificação (2026-06-26 02:04 UTC):
- `costs_raw`: `count() = 556.223`, `uniqExact(*) = 556.223` (sem duplicatas).
- Kafka consumer group `cost-analytics`: `CURRENT-OFFSET = LOG-END-OFFSET = 3.878.151`, `LAG = 0`.
- Ingestion-service: scheduler a cada 5m; últimos arquivos do bucket já processados.
- Todos os containers healthy.

## Validação Final

- ✅ Redeploy do zero concluído com sucesso.
- ✅ Dados reais ingeridos do bucket Huawei OBS.
- ✅ Duplicatas eliminadas no ClickHouse.
- ✅ Dashboard mostra dados reais (costs, trends, KPIs, anomalies, recommendations, budgets, users).
- ✅ USD/BRL alterna corretamente e refetcha dados.
- ✅ Budgets e Usuários não crasham quando vazios.
- ✅ Forecast/Anomalias/Recomendações baseados em dados reais.

## Comandos de Diagnóstico Úteis

```bash
# Contagem no ClickHouse
docker exec finops-clickhouse clickhouse-client --database=finops -q "SELECT count(), uniqExact(*) FROM costs_raw"

# Logs
sudo docker logs -f finops-ingestion
sudo docker logs -f finops-cost-analytics

# Consumer group Kafka
sudo docker exec finops-kafka /bin/kafka-consumer-groups --bootstrap-server localhost:9092 --describe --group cost-analytics

# Checkpoints
sudo docker compose exec -T postgres psql -U finops -d finops -c "SELECT * FROM ingestion_checkpoints"

# Health
curl http://localhost:8080/health
curl http://localhost:8081/health
```

## Próximos Passos Recomendados

1. **Monitorar estabilidade**: verificar se `count()` ≈ `uniqExact(*)` nas próximas horas.
2. **Resolver duplicatas de origem**: os arquivos diários da Huawei são cumulativos. Considerar:
   - Processar apenas o arquivo mais recente de cada mês no ingestion-service; ou
   - Alterar `costs_raw` para `ReplacingMergeTree` com chave de deduplicação.
3. **Melhorar forecast**: a lógica atual compara períodos de mesma duração e explode o trend quando o período anterior tem poucos dados.
4. **Commitar mudanças**: `backend/cost-analytics/main.go` foi modificado; fazer push para o branch `Teste-Docker-Compose`.

## Contato / Decisões Pendentes

- Dashboard acessível em `http://10.140.12.32:3000`.
- Login demo funcional.
- Decisão futura sobre estratégia de deduplicação permanente.
