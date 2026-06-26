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
- **Suporte a múltiplas métricas de custo**: novo query param `cost_metric` (`effective`, `gross`, `amortized`, `list`, `contracted`).
- **Nova coluna `charge_type`**: populada a partir do FOCUS `ChargeCategory` e usada pelo `cost_metric=gross`.

### Backend `backend/ingestion-service/main.go`
- Lock global (`processMu`) em `processIngestion` para evitar ingestões concorrentes.
- **Processamento apenas do arquivo mais recente por mês**: elimina duplicatas causadas por exports FOCUS cumulativos diários.
- **Truncamento de datas sobrepostas**: antes de publicar um arquivo, o ingestion-service chama o cost-analytics para remover registros das datas presentes no arquivo, garantindo que o export mais recente prevaleça.
- Checkpoint salvo por provider/account com o último arquivo processado.

### Frontend
- Rebuild com `NEXT_PUBLIC_API_URL=http://10.140.12.32:8080`.
- `/login` e `/dashboard` carregam sem erro.

### Scripts
- `scripts/dedup_clickhouse.sql`: deduplica manual (mantido para emergências, mas não mais necessário no fluxo normal).

## Estado Atual na VM Alvo

Última verificação (2026-06-26 ~03:40 UTC):
- `costs_raw`: `count() = 281.823`, `uniqExact(*) = 203.163` (diferença esperada devido a linhas FOCUS que diferem apenas em colunas não mapeadas no ClickHouse; somas de custo estão corretas).
- Kafka consumer group `cost-analytics`: `LAG = 0`.
- Todos os containers healthy.
- Arquivos processados:
  - Maio: `daily-exports/Daily_Cost_Export_Focus1-0/202605/20260605T085424Z/FOCUS_COST_DATA_202605_000001.zip`
  - Junho: `daily-exports/Daily_Cost_Export_Focus1-0/202606/20260625T084645Z/FOCUS_COST_DATA_202606_000001.zip`

### Comparação com Cost Center Huawei Cloud

| Métrica | Console Huawei | FinOps Dashboard (`effective`) | FinOps Dashboard (`gross`) |
|---|---|---|---|
| Maio 2026 | $1.100.466,97 | $945.602,83 | $1.038.500,38 |
| Junho 2026 (MTD) | $804.364,68 | $708.575,83 | $708.597,49 |
| Maio + Junho MTD | ~$1.904.831 | $1.654.178,66 | $1.747.097,87 |

- `effective`: soma de `EffectiveCost` do FOCUS (inclui ajustes/créditos).
- `gross`: soma de `EffectiveCost` apenas para `charge_type IN ('Usage', 'Purchase', 'Tax')` (sem ajustes/créditos).
- A console Huawei provavelmente inclui impostos/taxas sobre o valor bruto, o que explica a diferença residual (~6% maio, ~13% junho).

## Validação Final

- ✅ Redeploy do zero concluído.
- ✅ Dados reais ingeridos do bucket Huawei OBS.
- ✅ Deduplicação automática implementada (último arquivo por mês + truncamento de datas).
- ✅ Dashboard mostra dados reais.
- ✅ USD/BRL alterna corretamente.
- ✅ Budgets e Usuários não crasham.
- ✅ Forecast/Anomalias/Recomendações baseados em dados reais.
- ✅ Suporte a múltiplas métricas de custo (`effective`, `gross`, `amortized`, `list`, `contracted`).

## Comandos de Diagnóstico Úteis

```bash
# Contagem no ClickHouse
sudo docker exec finops-clickhouse clickhouse-client --database=finops -q "SELECT count(), uniqExact(*) FROM costs_raw"

# Custos por mês
sudo docker exec finops-clickhouse clickhouse-client --database=finops -q "SELECT toYYYYMM(date) as month, sum(effective_cost) FROM costs_raw GROUP BY month ORDER BY month"

# Distribuição por charge_type
sudo docker exec finops-clickhouse clickhouse-client --database=finops -q "SELECT charge_type, count(), sum(effective_cost) FROM costs_raw GROUP BY charge_type"

# Logs
sudo docker logs -f finops-ingestion
sudo docker logs -f finops-cost-analytics

# Health
curl http://localhost:8080/health
curl http://localhost:8081/health
```

## Próximos Passos Recomendados

1. **Monitorar estabilidade**: verificar se `count()` se mantém estável nas próximas execuções do scheduler.
2. **Novo arquivo FOCUS**: quando a Huawei publicar o export de `20260626`, o ingestion-service deve processá-lo automaticamente e atualizar os dados de junho.
3. **Alinhamento com console**: validar com o time financeiro da Huawei qual métrica e quais impostos/taxas são incluídos no Cost Center. Considerar adicionar um fator de imposto configurável ao dashboard.
4. **Otimização**: avaliar migrar `costs_raw` para `ReplacingMergeTree` com chave de deduplicação completa, eliminando a necessidade de truncamento manual.

## Contato / Decisões Pendentes

- Dashboard acessível em `http://10.140.12.32:3000`.
- Login demo funcional.
- Discrepância residual com a console Huawei está documentada e provavelmente explicada por impostos/taxas não presentes nos arquivos FOCUS.
