# Handoff Context - FinOps Platform

> Arquivo gerado para continuidade entre sessões/máquinas.
> Última atualização: 2026-06-12

## Objetivo Atual

Corrigir dashboard funcional com dados reais, eliminando problemas de qualidade e acuracidade:

1. **Duplicatas no ClickHouse** — `costs_raw` chegou a ~5.5M registros, mas há ~535k únicos.
2. **Conversão USD/BRL no frontend** — backend já converte com taxa 5.15, mas queries precisam refetch ao trocar moeda.
3. **Crashes em Budgets e Usuários** quando não há registros.
4. **KPIs/Forecast/Anomalias/Recomendações** baseados em dados reais, não hardcoded.
5. **Causa raiz identificada**: `cost-analytics` consumia mensagens do Kafka sem comitar offsets, reprocessando o tópico `cost.raw` indefinidamente.

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
- Diretório local Windows: `c:\Users\Victor Laranjeira\GitHub\finops-platform`

## Alterações Realizadas (ainda não testadas na VM alvo)

### Backend `backend/cost-analytics/main.go`
- Adicionado `CommitMessages(ctx, msg)` após inserção no ClickHouse.
- Configurado `kafka.Reader` com `StartOffset: kafka.LastOffset` para evitar reprocessar tópico inteiro.
- Null-safety em todas as agregações (`sum`, `uniqExact`) usando ponteiros e `derefFloat`/`derefUint64`.
- Retorno de arrays vazios em vez de `nil`/erro 500 quando não há dados.
- KPIs reais: `Cost Efficiency = effective_cost / list_cost`; `Budget Utilization` baseado em dados reais.
- `potential_savings` no executive dashboard calculado como `list_cost - effective_cost`.
- `total_savings` em recommendations = soma real das economias das recomendações.
- `getBudgets` e `getUsers` tratam datas nulas e `last_login` nulo; retornam `[]` quando vazio.

### Backend `backend/ingestion-service/main.go`
- Adicionado lock global (`processMu`) em `processIngestion` para evitar ingestões concorrentes.

### Frontend
- `frontend/nextjs/contexts/CurrencyContext.tsx`: troca de moeda invalida queries de custo/dashboard.
- `frontend/nextjs/app/dashboard/page.tsx`: `useCostTrends` usa `startDate/endDate`; KPIs estratégicos calculados de dados reais.
- `frontend/nextjs/app/forecast/page.tsx`: `useCostTrends` corrigido para usar `startDate/endDate`.

### Scripts
- `scripts/dedup_clickhouse.sql`: deduplica `costs_raw` via `SELECT DISTINCT *` com troca atômica de tabelas.
- `scripts/dedup_clickhouse.sh`: para serviços, deduplica, reseta offsets do Kafka para latest, reinicia.
- `scripts/full-redeploy.sh`: redeploy completo do zero (apaga volumes, rebuilda, aplica migrations).

### Documentação
- `docs/RUNBOOK.md`: seção de redeploy do zero e troubleshooting de duplicatas.
- `docs/ADMIN_GUIDE.md`: comandos úteis de ClickHouse e deduplicação.

## Estado Atual na VM Alvo

Última verificação (2026-06-24):
- `costs_raw`: `count() = 5.489.961`, `uniqExact(*) = 534.884`
- Ingestion-service: checkpoint correto, `processed=0` na última execução.
- Cost-analytics: continuava inserindo registros antigos do Kafka (falta de commit de offsets).
- Ingestion interval: `5m`.

## Próximos Passos Recomendados

### Opção A: Redeploy do zero (recomendada para garantir limpeza)
1. Na VM alvo (`/finops-platform`):
   ```bash
   git pull origin Teste-Docker-Compose
   chmod +x scripts/full-redeploy.sh
   ./scripts/full-redeploy.sh
   ```
2. O script apaga volumes, rebuilda imagens, aplica migrations.
3. Disparar ingestão:
   ```bash
   curl -X POST http://localhost:8081/api/v1/ingestion/trigger \
     -H 'Content-Type: application/json' \
     -d '{"provider":"huawei","bucket":"focusfinops","prefix":"daily-exports/Daily_Cost_Export_Focus1-0/","account_id":"hw-account-001"}'
   ```
4. Monitorar até estabilizar:
   ```bash
   watch -n 5 'docker exec finops-clickhouse clickhouse-client --database=finops -q "SELECT count(), uniqExact(*) FROM costs_raw"'
   ```

### Opção B: Deduplicar sem apagar volumes
1. Na VM alvo (`/finops-platform`):
   ```bash
   git pull origin Teste-Docker-Compose
   docker compose up -d --build cost-analytics ingestion-service
   ./scripts/dedup_clickhouse.sh
   ```
2. Verificar estabilização da contagem.

## Validação Esperada

- `count()` ≈ `uniqExact(*)` e ambos param de subir quando não há novos arquivos.
- Dashboard mostra dados reais, sem valores hardcoded.
- USD/BRL alterna corretamente e refetcha dados.
- Budgets e Usuários não crasham quando vazios.
- Forecast/Anomalias/Recomendações baseados em dados reais.

## Comandos de Diagnóstico Úteis

```bash
# Contagem no ClickHouse
docker exec finops-clickhouse clickhouse-client --database=finops -q "SELECT count(), uniqExact(*) FROM costs_raw"

# Logs
docker logs -f finops-ingestion
docker logs -f finops-cost-analytics

# Consumer group Kafka
docker exec finops-kafka kafka-consumer-groups.sh --bootstrap-server localhost:9092 --describe --group cost-analytics

# Checkpoints
docker compose exec -T postgres psql -U finops -d finops -c "SELECT * FROM ingestion_checkpoints"

# Health
curl http://localhost:8080/health
curl http://localhost:8081/health
curl http://localhost:8082/health
```

## Contato / Decisões Pendentes

- Usuário acessa `az-vm-ubu-finops-platform-v2` através de jumphost.
- Próxima sessão provavelmente será na jumphost.
- Decidir entre **Opção A (redeploy do zero)** ou **Opção B (deduplicação sem apagar volumes)**.
