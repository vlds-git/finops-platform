# Guia de Troubleshooting - FinOps Enterprise Platform

## Índice de Problemas

1. [API Gateway](#api-gateway)
2. [Ingestion Service](#ingestion-service)
3. [Cost Analytics](#cost-analytics)
4. [Budgets e Accounts](#budgets-e-accounts)
5. [Usuários](#usuários)
6. [Alert Manager](#alert-manager)
7. [Forecast Engine / Anomaly / Recommendation](#forecast-engine--anomaly--recommendation)
8. [PostgreSQL](#postgresql)
9. [ClickHouse](#clickhouse)
10. [Kafka](#kafka)
11. [Redis](#redis)
12. [Huawei OBS](#huawei-obs)
13. [Frontend](#frontend)
14. [Performance](#performance)
15. [Segurança](#seguranca)

---

## API Gateway

### 502 Bad Gateway
**Sintoma:** Requisições retornam 502, serviço parece indisponível.

**Diagnóstico:**
```bash
# Verifique saúde do gateway
curl -v http://localhost:8080/health
curl -v http://localhost:8080/ready

# Verifique se está rodando
docker ps | grep api-gateway
kubectl get pods -n finops -l app=api-gateway

# Verifique logs
kubectl logs -n finops -l app=api-gateway --tail=100

# Verifique downstream services
for svc in cost-analytics:8082 alert-manager:8083; do
  kubectl exec -n finops -it deployment/api-gateway -- wget -q --spider http://$svc || echo "DOWN: $svc"
done
```

**Causas e Correções:**

| Causa | Identificação | Correção |
|-------|--------------|----------|
| Serviço downstream offline | `wget` falha para serviço | Reinicie o serviço downstream |
| Rate limit atingido | Logs mostram "rate limit exceeded" | Limpe Redis: `redis-cli FLUSHDB` ou aguarde |
| JWT inválido | Logs mostram "invalid token" | Verifique secret ou renove token |
| Memória esgotada | `kubectl top pod` mostra alta | Aumente limites ou reinicie |

**Comando rápido:**
```bash
kubectl rollout restart deployment/api-gateway -n finops
```

### 401 Unauthorized
**Sintoma:** Todas as requisições retornam 401.

**Diagnóstico:**
```bash
# Verifique se o token está expirado
curl -v -H "Authorization: Bearer <token>" http://localhost:8080/api/v1/costs

# Verifique JWT secret
kubectl get secret finops-secrets -n finops -o jsonpath='{.data.jwt-secret}' | base64 -d
# Compare com o secret usado para gerar o token
```

**Correção:**
```bash
# Renove o token
# 1. Faça login novamente
curl -X POST http://localhost:8080/api/v1/auth/login   -d '{"email":"admin@finops.local","password":"admin123"}'

# 2. Ou verifique se o secret mudou e reinicie o gateway
kubectl rollout restart deployment/api-gateway -n finops
```

### 403 Forbidden
**Sintoma:** Token válido, mas acesso negado.

**Diagnóstico:**
```bash
# Verifique roles do usuário
psql -h postgres -U finops -c "SELECT email, roles FROM users WHERE email='usuario@company.com'"

# Verifique logs de audit
psql -h postgres -U finops -c "SELECT action, resource, details FROM audit WHERE user_id='uuid' ORDER BY created_at DESC"
```

**Correção:**
```bash
# Atualize roles
psql -h postgres -U finops -c "UPDATE users SET roles=ARRAY['analyst'] WHERE email='usuario@company.com'"
```

---

## Ingestion Service

### Sem dados novos no ClickHouse
**Sintoma:** Dashboards mostram dados desatualizados, última data antiga.

**Diagnóstico:**
```bash
# Verifique último checkpoint
psql -h postgres -U finops -c "SELECT provider, account_id, last_file, last_processed_at FROM ingestion_checkpoints ORDER BY last_processed_at DESC"

# Verifique DLQ
psql -h postgres -U finops -c "SELECT COUNT(*) FROM ingestion_dlq WHERE resolved=false"

# Verifique logs do ingestion
kubectl logs -n finops -l app=ingestion-service --tail=200 | grep -i "error\|fail\|obs"

# Verifique Kafka topic
kafka-topics --bootstrap-server kafka:9092 --describe --topic cost.raw

# Verifique consumer groups
kafka-consumer-groups --bootstrap-server kafka:9092 --describe --group cost-analytics
```

**Causas e Correções:**

| Causa | Identificação | Correção |
|-------|--------------|----------|
| Credenciais OBS expiradas | Logs: "missing OBS credentials" | Atualize secret e reinicie |
| Checkpoint corrompido | Last file inválido | Reset checkpoint e reprocesse |
| Kafka indisponível | Consumer lag alto | Reinicie Kafka ou consumers |
| Arquivo inválido | DLQ com entries | Resolva DLQ ou ignore arquivo |
| Bucket vazio | OBS sem novos exports | Verifique export no console do provedor |

**Comandos de correção:**
```bash
# Reset checkpoint
psql -h postgres -U finops -c "UPDATE ingestion_checkpoints SET last_file='', last_offset=0 WHERE provider='huawei'"

# Trigger reprocessamento
curl -X POST http://localhost:8081/ingestion/reprocess   -d '{"provider":"huawei","account_id":"hw-001"}'

# Reinicie serviço
kubectl rollout restart deployment/ingestion-service -n finops
```

### DLQ crescendo
**Sintoma:** Tabela ingestion_dlq tem muitos registros não resolvidos.

**Diagnóstico:**
```bash
psql -h postgres -U finops -c "SELECT provider, file_name, error_message, retry_count FROM ingestion_dlq WHERE resolved=false ORDER BY retry_count DESC"
```

**Correção:**
```bash
# Resolva manualmente após investigação
psql -h postgres -U finops -c "UPDATE ingestion_dlq SET resolved=true WHERE id='uuid'"

# Ou limpe todos resolvidos
psql -h postgres -U finops -c "DELETE FROM ingestion_dlq WHERE resolved=true AND created_at < now() - interval '7 days'"
```

---

## Cost Analytics

### Dashboards vazios
**Sintoma:** Gráficos mostram "No data" ou valores zerados.

**Diagnóstico:**
```bash
# Verifique ClickHouse
clickhouse-client -q "SELECT count() FROM costs"
clickhouse-client -q "SELECT max(date) FROM costs"

# Verifique tabelas existentes
clickhouse-client -q "SHOW TABLES"

# Verifique consumer Kafka
kafka-consumer-groups --bootstrap-server kafka:9092 --describe --group cost-analytics

# Verifique logs
kubectl logs -n finops -l app=cost-analytics --tail=100
```

**Causas e Correções:**

| Causa | Identificação | Correção |
|-------|--------------|----------|
| ClickHouse vazio | `count() = 0` | Verifique ingestão, reprocesse |
| Schema desatualizado | Tabela `costs` não existe | Aplique schema.sql |
| Consumer parado | Lag alto no Kafka | Reinicie cost-analytics |
| Query timeout | Logs: "timeout" | Otimize query ou aumente recursos |

**Comandos:**
```bash
# Aplique schema ClickHouse
clickhouse-client < database/clickhouse/schema.sql

# Reinicie consumer
kubectl rollout restart deployment/cost-analytics -n finops

# Verifique dados brutos
clickhouse-client -q "SELECT date, provider, service_name, effective_cost FROM costs LIMIT 10"
```

### Queries lentas
**Sintoma:** Dashboards demoram > 10s para carregar.

**Diagnóstico:**
```bash
# Verifique queries lentas no ClickHouse
clickhouse-client -q "SELECT query, query_duration_ms, read_rows, read_bytes FROM system.query_log WHERE event_date = today() ORDER BY query_duration_ms DESC LIMIT 10"

# Verifique uso de recursos
kubectl top pod -n finops -l app=cost-analytics
kubectl top pod -n finops -l app=clickhouse

# Verifique partições
clickhouse-client -q "SELECT partition, formatReadableSize(sum(bytes)) as size, sum(rows) as rows FROM system.parts WHERE active AND table='costs' GROUP BY partition ORDER BY partition DESC"
```

**Correção:**
```bash
# Otimize tabela
clickhouse-client -q "OPTIMIZE TABLE costs FINAL"

# Verifique índices
clickhouse-client -q "SELECT name, type, expr FROM system.data_skipping_indices WHERE table='costs'"

# Se necessário, adicione índices
ALTER TABLE costs ADD INDEX idx_resource_id resource_id TYPE bloom_filter GRANULARITY 3;
ALTER TABLE costs ADD INDEX idx_tags tags TYPE bloom_filter GRANULARITY 3;

# Aumente recursos se CPU/memória limitados
kubectl patch deployment cost-analytics -n finops -p '{"spec":{"template":{"spec":{"containers":[{"name":"cost-analytics","resources":{"limits":{"cpu":"2000m","memory":"1Gi"}}}]}}}}'
```

---

## Budgets e Accounts

### Budget mostra spent zerado ou incorreto
**Sintoma:** Coluna "Realizado" do budget não reflete custos reais.

**Diagnóstico:**
```bash
# Verifique se a account existe
psql -h postgres -U finops -c "SELECT account_id, account_name FROM cloud_accounts"

# Verifique custos no período do budget
clickhouse-client -q "SELECT sum(effective_cost) FROM costs_raw WHERE date BETWEEN '2024-04-01' AND '2024-06-30' AND provider='huawei' AND billing_account_id='hw-account-001'"

# Verifique logs do cost-analytics
kubectl logs -n finops -l app.kubernetes.io/component=cost-analytics --tail=100
```

**Causas comuns:**
- `account_id` do budget não existe em `cloud_accounts`.
- Não há dados no ClickHouse para o período/provider/account selecionado.
- Colunas BRL ausentes (`effective_cost_brl`, `list_cost_brl`, etc.).

**Correção:**
```bash
# Aplique schema ClickHouse se necessário
clickhouse-client -q "ALTER TABLE costs_raw ADD COLUMN IF NOT EXISTS effective_cost_brl Float64"

# Reprocesse a ingestão
curl -X POST http://localhost:8080/api/v1/ingestion/reprocess -H "Authorization: Bearer <TOKEN>"
```

---

## Usuários

### Não é possível criar/editar usuário
**Sintoma:** Modal de usuário retorna erro ou não salva.

**Diagnóstico:**
```bash
# Verifique logs do cost-analytics
kubectl logs -n finops -l app.kubernetes.io/component=cost-analytics --tail=100

# Teste API manualmente
curl -X POST http://localhost:8080/api/v1/admin/users \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{"email":"teste@company.com","name":"Teste","password":"senha123","roles":["viewer"]}'
```

**Causas comuns:**
- Email duplicado (coluna `email` tem unique constraint).
- Senha em branco na criação.
- Roles inválidas (deve ser um array de strings).

> **Nota:** senhas são armazenadas em plaintext no ciclo atual. A correção está no `ROADMAP.md`.

---

## Alert Manager

> A página de Alertas foi removida do frontend. O backend continua disponível, mas a funcionalidade principal será reintroduzida vinculada aos budgets.

### Alertas não sendo enviados
**Sintoma:** Alertas aparecem no banco mas notificações não chegam.

**Diagnóstico:**
```bash
# Verifique alertas no banco
psql -h postgres -U finops -c "SELECT id, rule_name, severity, status, created_at FROM alerts WHERE status='firing' ORDER BY created_at DESC"

# Verifique logs do alert-manager
kubectl logs -n finops -l app.kubernetes.io/component=alert-manager --tail=100

# Verifique configuração das regras
psql -h postgres -U finops -c "SELECT name, channel, destination, enabled FROM alert_rules"

# Teste webhook manualmente
curl -X POST <webhook-url> -d '{"test":"message"}' -v
```

---

## Forecast Engine / Anomaly / Recommendation

> Forecast, anomalias e recomendações são calculados pelo `cost-analytics` a partir de dados reais do ClickHouse. Os serviços Python estão em standby.

### Forecast vazio ou valores fixos
**Sintoma:** Página de Forecast não mostra dados ou valores não mudam.

**Diagnóstico:**
```bash
# Teste API diretamente
curl "http://localhost:8080/api/v1/forecast?start_date=2024-05-01&end_date=2024-05-30&forecast_days=30" \
  -H "Authorization: Bearer <TOKEN>"

# Verifique se há dados no período
clickhouse-client -q "SELECT date, sum(effective_cost) FROM costs_raw WHERE date BETWEEN '2024-05-01' AND '2024-05-30' GROUP BY date ORDER BY date"

# Logs do cost-analytics
kubectl logs -n finops -l app.kubernetes.io/component=cost-analytics --tail=100
```

### Sem anomalias detectadas
**Sintoma:** Página de Anomalias retorna lista vazia.

**Diagnóstico:**
```bash
curl "http://localhost:8080/api/v1/anomalies?start_date=2024-05-01&end_date=2024-05-30" \
  -H "Authorization: Bearer <TOKEN>"

# Verifique variância dos dados
clickhouse-client -q "SELECT date, sum(effective_cost) FROM costs_raw WHERE date BETWEEN '2024-05-01' AND '2024-05-30' GROUP BY date ORDER BY date"
```

> Anomalias são detectadas apenas quando o desvio padrão é significativo (`|z| > 2`). Se os custos forem muito estáveis, nenhuma anomalia será reportada.

---

## PostgreSQL

### Conexões esgotadas
**Sintoma:** Erros "too many connections", serviços não conseguem conectar.

**Diagnóstico:**
```bash
psql -h postgres -U finops -c "SELECT count(*), state FROM pg_stat_activity WHERE datname='finops' GROUP BY state"
psql -h postgres -U finops -c "SELECT count(*) FROM pg_stat_activity WHERE state='idle' AND state_change < now() - interval '1 hour'"
```

**Correção:**
```bash
# Mate conexões idle antigas
psql -h postgres -U finops -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE state='idle' AND state_change < now() - interval '1 hour'"

# Verifique max_connections
psql -h postgres -U finops -c "SHOW max_connections"

# Aumente se necessário (editar postgresql.conf ou via Helm values)
# postgresql.primary.configuration: "max_connections = 200"
```

### Disco cheio
**Sintoma:** PostgreSQL para de escrever, erros "no space left".

**Diagnóstico:**
```bash
# Verifique uso de disco
kubectl exec -n finops -it postgres-0 -- df -h

# Verifique tamanho do banco
psql -h postgres -U finops -c "SELECT pg_size_pretty(pg_database_size('finops'))"

# Verifique tabelas grandes
psql -h postgres -U finops -c "SELECT schemaname, tablename, pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) FROM pg_tables WHERE schemaname='public' ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC"
```

**Correção:**
```bash
# Expanda PVC (Kubernetes)
kubectl patch pvc data-postgres-0 -n finops -p '{"spec":{"resources":{"requests":{"storage":"30Gi"}}}}'

# Limpe dados antigos (CUIDADO)
psql -h postgres -U finops -c "DELETE FROM audit WHERE created_at < now() - interval '1 year'"
psql -h postgres -U finops -c "DELETE FROM alerts WHERE status='resolved' AND created_at < now() - interval '6 months'"

# Vacuum para recuperar espaço
psql -h postgres -U finops -c "VACUUM FULL"
```

### WAL crescendo infinitamente
**Sintoma:** Diretório pg_wal consome muito espaço.

**Correção:**
```bash
# Force checkpoint
psql -h postgres -U finops -c "CHECKPOINT"

# Verifique replicação
psql -h postgres -U finops -c "SELECT * FROM pg_replication_slots"

# Se slot inativo, remova
psql -h postgres -U finops -c "SELECT pg_drop_replication_slot('slot_name')"
```

---

## ClickHouse

### Memória alta / OOMKilled
**Sintoma:** Pod reinicia frequentemente, status OOMKilled.

**Diagnóstico:**
```bash
kubectl get pod -n finops -l app=clickhouse
kubectl describe pod -n finops -l app=clickhouse | grep -A 5 "Last State"

# Verifique queries pesadas
clickhouse-client -q "SELECT query, query_duration_ms, memory_usage FROM system.processes ORDER BY memory_usage DESC"

# Verifique configuração de memória
clickhouse-client -q "SELECT name, value FROM system.settings WHERE name LIKE '%memory%'"
```

**Correção:**
```bash
# Mate queries pesadas
clickhouse-client -q "KILL QUERY WHERE query_duration_ms > 300000"

# Aumente limites de memória (config.xml ou via env)
# max_memory_usage = 4GB
# max_bytes_before_external_group_by = 2GB

# Aumente recursos do pod
kubectl patch statefulset clickhouse -n finops -p '{"spec":{"template":{"spec":{"containers":[{"name":"clickhouse","resources":{"limits":{"memory":"4Gi"}}}]}}}}'
```

### Partições faltando / Dados não inserindo
**Sintoma:** Erros "Part ... doesn't exist" ou inserts falham.

**Diagnóstico:**
```bash
# Verifique partições
clickhouse-client -q "SELECT partition, name, active FROM system.parts WHERE table='costs' ORDER BY partition DESC"

# Verifique erros de inserção
clickhouse-client -q "SELECT * FROM system.errors ORDER BY last_error_time DESC LIMIT 10"
```

**Correção:**
```bash
# Se tabela corrompida, recrie (dados re-ingestíveis do OBS)
clickhouse-client -q "DROP TABLE IF EXISTS costs"
clickhouse-client < database/clickhouse/schema.sql

# Re-ingest dados
# Reset checkpoints e trigger reprocessamento
```

### Replicação quebrada (se usar replicas)
**Sintoma:** Dados inconsistentes entre réplicas.

**Correção:**
```bash
# Verifique status das réplicas
clickhouse-client -q "SELECT database, table, is_readonly, is_session_expired, future_parts, parts_to_check FROM system.replicas"

# Ressincronize
clickhouse-client -q "SYSTEM RESTART REPLICA costs"
clickhouse-client -q "SYSTEM RESTORE REPLICA costs"

# Se necessário, force resync
clickhouse-client -q "SYSTEM SYNC REPLICA costs"
```

---

## Kafka

### Consumer lag alto
**Sintoma:** Mensagens acumulam, não são processadas.

**Diagnóstico:**
```bash
# Verifique lag
kafka-consumer-groups --bootstrap-server kafka:9092 --describe --all-groups

# Verifique se consumers estão rodando
kubectl get pods -n finops -l app=cost-analytics
kubectl get pods -n finops -l app=forecast-engine
kubectl get pods -n finops -l app=anomaly-detection
```

**Correção:**
```bash
# Escale consumers
kubectl scale deployment cost-analytics -n finops --replicas=4

# Se lag persistir, verifique se consumers compartilham group ID correto
# Cada serviço deve ter group ID único:
# cost-analytics: group_id = "cost-analytics"
# forecast-engine: group_id = "forecast-engine"

# Reinicie consumers
kubectl rollout restart deployment/cost-analytics -n finops
```

### Disco cheio no Kafka
**Sintoma:** Brokers não aceitam novas mensagens.

**Diagnóstico:**
```bash
kubectl exec -n finops -it kafka-0 -- df -h /bitnami/kafka
kubectl exec -n finops -it kafka-0 -- du -sh /bitnami/kafka/data/*
```

**Correção:**
```bash
# Expanda PVC
kubectl patch pvc data-kafka-0 -n finops -p '{"spec":{"resources":{"requests":{"storage":"100Gi"}}}}'

# Ou reduza retenção (temporário)
kafka-configs --bootstrap-server kafka:9092 --entity-type topics --entity-name cost.raw --alter --add-config retention.ms=86400000

# Delete tópicos antigos se não necessários
# CUIDADO: perde dados
```

### Zookeeper desconectado
**Sintoma:** Kafka brokers não formam cluster.

**Correção:**
```bash
# Verifique zookeeper
kubectl get pods -n finops -l app=zookeeper
kubectl logs -n finops -l app=zookeeper

# Reinicie zookeeper
kubectl rollout restart statefulset/zookeeper -n finops

# Aguarde e reinicie kafka
kubectl rollout restart statefulset/kafka -n finops
```

---

## Redis

### Conexões recusadas
**Sintoma:** Serviços não conseguem conectar ao Redis.

**Correção:**
```bash
# Verifique Redis
kubectl get pods -n finops -l app=redis
kubectl logs -n finops -l app=redis

# Teste conexão
kubectl exec -n finops -it redis-master-0 -- redis-cli ping

# Reinicie
kubectl rollout restart statefulset/redis-master -n finops
```

### Memória esgotada
**Sintoma:** Redis evictando keys, cache misses altos.

**Correção:**
```bash
# Verifique uso
kubectl exec -n finops -it redis-master-0 -- redis-cli info memory

# Limpe cache antigo
kubectl exec -n finops -it redis-master-0 -- redis-cli EVAL "return redis.call('del', unpack(redis.call('keys', 'rate:*')))" 0

# Aumente memória
kubectl patch statefulset redis-master -n finops -p '{"spec":{"template":{"spec":{"containers":[{"name":"redis","resources":{"limits":{"memory":"1Gi"}}}]}}}}'
```

---

## Huawei OBS

### 403 Forbidden
**Sintoma:** Ingestion não consegue acessar bucket.

**Diagnóstico:**
```bash
# Verifique credenciais
kubectl get secret finops-secrets -n finops -o jsonpath='{.data.huawei-access-key}' | base64 -d
kubectl get secret finops-secrets -n finops -o jsonpath='{.data.huawei-secret-key}' | base64 -d

# Teste via OBS CLI (se disponível)
# obsutil ls obs://bucket-name/prefix/
```

**Correção:**
```bash
# 1. Verifique no console Huawei Cloud:
#    - IAM User existe
#    - Access Key ativa
#    - Política OBS ReadOnly anexada
#    - Bucket existe na região correta

# 2. Atualize secret
kubectl patch secret finops-secrets -n finops --type='json' -p='[{"op":"replace","path":"/data/huawei-access-key","value":"'$(echo -n 'NEW_KEY' | base64)'"}]'
kubectl patch secret finops-secrets -n finops --type='json' -p='[{"op":"replace","path":"/data/huawei-secret-key","value":"'$(echo -n 'NEW_SECRET' | base64)'"}]'

# 3. Reinicie ingestion
kubectl rollout restart deployment/ingestion-service -n finops
```

### 404 Not Found
**Sintoma:** Bucket ou prefixo não existe.

**Correção:**
```bash
# Verifique no console:
# - Bucket existe
# - Prefixo correto (ex: exports/ vs billing/)
# - Arquivos presentes

# Atualize configuração da conta cloud
psql -h postgres -U finops -c "UPDATE cloud_accounts SET prefix='correct-prefix/' WHERE provider='huawei'"
```

---

## Frontend

### Página em branco
**Sintoma:** Acesso a http://localhost:3000 mostra tela branca.

**Diagnóstico:**
```bash
# Verifique console do navegador (F12)
# Erros comuns: CORS, CDN blocked, JS errors

# Verifique se ECharts carregou
# Verifique se Tailwind carregou
```

**Correção:**
```bash
# 1. Verifique conectividade com internet (CDN)
# 2. Se offline, baixe assets localmente:
#    - echarts.min.js
#    - tailwindcss CDN
# 3. Verifique se index.html está no volume correto
kubectl exec -n finops -it deployment/frontend -- ls /usr/share/nginx/html

# 4. Reinicie nginx
kubectl rollout restart deployment/frontend -n finops
```

### Gráficos não renderizam
**Sintoma:** Cards aparecem mas gráficos estão vazios.

**Correção:**
```bash
# 1. Verifique se API está respondendo
curl http://localhost:8080/api/v1/costs

# 2. Se API retorna dados, problema é no frontend
#    - Verifique console do navegador por erros JS
#    - Verifique se ECharts está inicializado após DOM ready

# 3. Se API retorna vazio, problema é no backend
#    - Verifique Cost Analytics
#    - Verifique ClickHouse
```

---

## Performance

### Latência alta na API
**Sintoma:** Respostas demoram > 2s.

**Diagnóstico:**
```bash
# Teste latência
curl -o /dev/null -s -w "%{time_total}
" http://localhost:8080/api/v1/costs

# Verifique p95 no Prometheus
# histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# Verifique recursos
kubectl top pods -n finops
```

**Correção:**
```bash
# 1. Escale serviços críticos
kubectl scale deployment api-gateway -n finops --replicas=4
kubectl scale deployment cost-analytics -n finops --replicas=4

# 2. Otimize ClickHouse queries
# Adicione índices, otimize partições

# 3. Ative caching no Redis
# Verifique se responses estão cacheadas

# 4. Verifique network latency entre pods
kubectl exec -n finops -it deployment/api-gateway -- ping cost-analytics
```

### ClickHouse queries lentas
**Sintoma:** Analytics demoram > 30s.

**Correção:**
```bash
# 1. Verifique se está usando Materialized Views
clickhouse-client -q "SELECT * FROM system.views WHERE database='finops'"

# 2. Se não, crie MVs para queries comuns
# (já incluído no schema.sql)

# 3. Verifique se partições estão otimizadas
clickhouse-client -q "OPTIMIZE TABLE costs FINAL"

# 4. Aumente max_threads
clickhouse-client -q "SET max_threads = 8"

# 5. Considere adicionar mais réplicas para leitura
```

---

## Segurança

### Suspeita de acesso não autorizado
**Sintoma:** Logs mostram acessos de IPs desconhecidos.

**Diagnóstico:**
```bash
# Verifique audit log
psql -h postgres -U finops -c "SELECT user_id, action, resource, ip_address, created_at FROM audit WHERE created_at > now() - interval '1 hour' ORDER BY created_at DESC"

# Verifique tentativas de login falhas
kubectl logs -n finops -l app=api-gateway | grep -i "unauthorized\|invalid credentials"
```

**Correção:**
```bash
# 1. Revogue tokens suspeitos
# JWT não tem revogação nativa, mas pode:
# - Mudar JWT_SECRET (força re-login de todos)
kubectl patch secret finops-secrets -n finops --type='json' -p='[{"op":"replace","path":"/data/jwt-secret","value":"'$(openssl rand -base64 64 | base64)'"}]'
kubectl rollout restart deployment/api-gateway -n finops

# 2. Desative usuário suspeito
psql -h postgres -U finops -c "UPDATE users SET active=false WHERE email='suspeito@company.com'"

# 3. Verifique e ajuste RBAC
psql -h postgres -U finops -c "SELECT email, roles FROM users WHERE active=true"

# 4. Adicione rate limiting mais agressivo
# Edite ConfigMap e reinicie
```

### SQL Injection detectada
**Sintoma:** Logs mostram queries maliciosas.

**Diagnóstico:**
```bash
# Verifique logs de queries
kubectl logs -n finops -l app=cost-analytics | grep -i "SELECT\|DROP\|DELETE"

# Verifique audit
psql -h postgres -U finops -c "SELECT * FROM audit WHERE action='query' AND details LIKE '%DROP%'"
```

**Correção:**
```bash
# O código já usa prepared statements, mas verifique:
# 1. Se há queries dinâmicas no código
# 2. Valide input nos endpoints
# 3. Adicione WAF se necessário
# 4. Monitore padrões suspeitos
```

---

## Ferramentas de Diagnóstico Rápido

### Script de diagnóstico completo

```bash
#!/bin/bash
# diagnose.sh - Executar quando algo está errado

echo "=== FinOps Platform Diagnostics ==="
echo ""

# 1. Pods status
echo "1. Pod Status:"
kubectl get pods -n finops

# 2. Health checks
echo ""
echo "2. Health Checks:"
for svc in api-gateway:8080 ingestion-service:8081 cost-analytics:8082 alert-manager:8083; do
  name=$(echo $svc | cut -d: -f1)
  port=$(echo $svc | cut -d: -f2)
  echo -n "$name: "
  curl -s -o /dev/null -w "%{http_code}" http://localhost:$port/health || echo "FAIL"
done

# 3. ML Health (opcional - serviços em standby)
echo ""
echo "3. ML Services (optional):"
for svc in forecast-engine:8001 anomaly-detection:8002 recommendation-engine:8003; do
  name=$(echo $svc | cut -d: -f1)
  port=$(echo $svc | cut -d: -f2)
  echo -n "$name: "
  curl -s -o /dev/null -w "%{http_code}" http://localhost:$port/health || echo "STANDBY/FAIL"
done

# 4. Database connectivity
echo ""
echo "4. Databases:"
echo -n "PostgreSQL: "
psql -h postgres -U finops -c "SELECT 1" > /dev/null 2>&1 && echo "OK" || echo "FAIL"

echo -n "ClickHouse: "
curl -s -o /dev/null -w "%{http_code}" http://localhost:8123/ping || echo "FAIL"

echo -n "Redis: "
redis-cli -h redis ping || echo "FAIL"

echo -n "Kafka: "
kafka-broker-api-versions --bootstrap-server kafka:9092 > /dev/null 2>&1 && echo "OK" || echo "FAIL"

# 5. Resource usage
echo ""
echo "5. Resource Usage:"
kubectl top pods -n finops

# 6. Recent errors
echo ""
echo "6. Recent Errors (last 10):"
for pod in $(kubectl get pods -n finops -o name | head -5); do
  echo "--- $pod ---"
  kubectl logs -n finops $pod --tail=10 | grep -i "error\|fail\|panic" || echo "No errors"
done

echo ""
echo "=== Diagnostics Complete ==="
```

---

## Contatos e Escalonamento

| Problema | Responsável | Escalonamento |
|----------|-------------|---------------|
| Infraestrutura (K8s, rede) | SRE / DevOps | Infrastructure Team |
| Banco de dados (PG, CH) | DBA | Database Team |
| Aplicação (bugs, crashes) | Development | Engineering Lead |
| Segurança (breach, IAM) | Security | CISO |
| Cloud Provider (OBS, API) | Cloud Team | Huawei Support |
| ML Models (accuracy) | Data Science | ML Lead |
