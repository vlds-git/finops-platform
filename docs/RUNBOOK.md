# RUNBOOK - FinOps Enterprise Platform

## Visão Geral

A FinOps Enterprise Platform é uma solução de gestão financeira de cloud composta por 7 microserviços, 4 camadas de dados e 4 ferramentas de observabilidade.

### Arquitetura da Solução

```
┌─────────────────────────────────────────────────────────────┐
│                        API Gateway (8080)                    │
│              JWT, RBAC, Rate Limit, Audit                    │
└─────────────────────────────────────────────────────────────┘
                              │
        ┌──────────┬──────────┼──────────┬──────────┐
        ▼          ▼          ▼          ▼          ▼
┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐
│Ingestion │ │  Cost    │ │  Alert   │ │ Forecast │ │ Anomaly  │
│(8081)    │ │Analytics │ │ Manager  │ │ (8001)   │ │ (8002)   │
│          │ │(8082)    │ │(8083)    │ │          │ │          │
└──────────┘ └──────────┘ └──────────┘ └──────────┘ └──────────┘
        │          │          │          │          │
        └──────────┴──────────┼──────────┴──────────┘
                              ▼
                    ┌─────────────────┐
                    │     Kafka       │
                    │   (9092)        │
                    └─────────────────┘
                              │
        ┌──────────┬──────────┼──────────┬──────────┐
        ▼          ▼          ▼          ▼          ▼
┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐
│ClickHouse│ │PostgreSQL│ │  Redis   │ │  Grafana │
│(8123/9000)│ │ (5432)   │ │ (6379)   │ │ (3001)   │
│ Analytics │ │ Metadata │ │  Cache   │ │Dashboards│
└──────────┘ └──────────┘ └──────────┘ └──────────┘
```

### Fluxo de Dados

1. **FOCUS Export** → Huawei OBS → Ingestion Service → Kafka → Cost Analytics → ClickHouse
2. **ClickHouse** → Forecast Engine → PostgreSQL + Kafka
3. **ClickHouse** → Anomaly Detection → PostgreSQL + Kafka
4. **ClickHouse** → Recommendation Engine → PostgreSQL
5. **Kafka** → Alert Manager → Slack/Email/Webhook

### Dependências entre Serviços

| Serviço | Depende de |
|---------|-----------|
| API Gateway | PostgreSQL, Redis |
| Ingestion | Kafka, Huawei OBS |
| Cost Analytics | PostgreSQL, ClickHouse, Kafka |
| Alert Manager | PostgreSQL, Kafka |
| Forecast | ClickHouse |
| Anomaly | ClickHouse |
| Recommendation | ClickHouse |

---

## Primeiro Deploy

### Docker Compose

```bash
# 1. Clone o repositório
cd finops-platform

# 2. Configure variáveis de ambiente (opcional)
cp .env.example .env
# Edite .env com suas credenciais Huawei OBS

# 3. Suba a stack
cd deploy/docker
docker-compose up -d

# 4. Verifique saúde
docker-compose ps
docker-compose logs -f api-gateway

# 5. Acesse
# Frontend: http://localhost:3000
# API: http://localhost:8080
# Grafana: http://localhost:3001 (admin/admin)
# Prometheus: http://localhost:9090
```

### Kubernetes

```bash
# 1. Aplique namespace e configs
kubectl apply -f deploy/kubernetes/namespace.yaml
kubectl apply -f deploy/kubernetes/configmap.yaml
kubectl apply -f deploy/kubernetes/secret.yaml

# 2. Aplique infraestrutura
kubectl apply -f deploy/kubernetes/postgres.yaml  # ou use Helm chart
kubectl apply -f deploy/kubernetes/clickhouse.yaml
kubectl apply -f deploy/kubernetes/redis.yaml
kubectl apply -f deploy/kubernetes/kafka.yaml

# 3. Aguarde infraestrutura pronta
kubectl wait --for=condition=ready pod -l app=postgres -n finops --timeout=300s
kubectl wait --for=condition=ready pod -l app=clickhouse -n finops --timeout=300s

# 4. Aplique serviços
kubectl apply -f deploy/kubernetes/gateway.yaml
kubectl apply -f deploy/kubernetes/ingestion.yaml
kubectl apply -f deploy/kubernetes/cost-analytics.yaml
kubectl apply -f deploy/kubernetes/alert-manager.yaml
kubectl apply -f deploy/kubernetes/forecast-engine.yaml
kubectl apply -f deploy/kubernetes/anomaly-detection.yaml
kubectl apply -f deploy/kubernetes/recommendation-engine.yaml

# 5. Aplique ingress
kubectl apply -f deploy/kubernetes/ingress.yaml

# 6. Verifique
kubectl get pods -n finops
kubectl get svc -n finops
kubectl get ingress -n finops
```

### Helm

```bash
# 1. Adicione repositórios
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

# 2. Instale dependências
helm dependency build deploy/helm/finops-platform

# 3. Instale a plataforma
helm install finops deploy/helm/finops-platform   --namespace finops   --create-namespace   --set gateway.ingress.hosts[0].host=finops.yourdomain.com

# 4. Verifique
helm list -n finops
kubectl get pods -n finops
```

---

## Operação Diária

### Checklist Diário (5 minutos)

```bash
# 1. Verifique saúde de todos os serviços
for svc in api-gateway ingestion-service cost-analytics alert-manager forecast-engine anomaly-detection recommendation-engine; do
  echo "=== $svc ==="
  curl -s http://$svc:8080/health 2>/dev/null || curl -s http://$svc:8001/health 2>/dev/null || echo "CHECK PORT"
done

# 2. Verifique Kafka
curl -s http://kafka:9092 2>/dev/null || echo "Kafka connectivity check"

# 3. Verifique bancos de dados
psql -h postgres -U finops -c "SELECT 1" 2>/dev/null || echo "PostgreSQL check"
curl -s http://clickhouse:8123/ping 2>/dev/null || echo "ClickHouse check"

# 4. Verifique ingestão
kubectl logs -n finops -l app=ingestion-service --tail=20

# 5. Verifique alertas pendentes
psql -h postgres -U finops -c "SELECT COUNT(*) FROM alerts WHERE status='firing'"
```

### Validar Ingestão FOCUS

```bash
# Verifique último checkpoint
psql -h postgres -U finops -c "SELECT provider, account_id, last_file, last_processed_at FROM ingestion_checkpoints ORDER BY last_processed_at DESC"

# Verifique DLQ (Dead Letter Queue)
psql -h postgres -U finops -c "SELECT provider, file_name, error_message, retry_count FROM ingestion_dlq WHERE resolved=false"

# Verifique volume de dados no ClickHouse
clickhouse-client -q "SELECT count(), sum(effective_cost) FROM costs WHERE date = today()"

# Verifique Kafka topics
kafka-topics --bootstrap-server kafka:9092 --list
kafka-consumer-groups --bootstrap-server kafka:9092 --describe --group cost-analytics
```

### Validar Integração OBS

```bash
# Verifique logs do ingestion
kubectl logs -n finops -l app=ingestion-service --since=1h | grep -i "obs\|huawei\|error"

# Verifique conectividade OBS (dentro do pod)
kubectl exec -n finops -it deployment/ingestion-service -- sh
# Dentro do pod:
# wget -q --spider http://obs.myhwclouds.com || echo "OBS connectivity issue"
```

### Validar Kafka

```bash
# Verifique brokers
kafka-broker-api-versions --bootstrap-server kafka:9092

# Verifique consumer groups
kafka-consumer-groups --bootstrap-server kafka:9092 --list
kafka-consumer-groups --bootstrap-server kafka:9092 --describe --group cost-analytics
kafka-consumer-groups --bootstrap-server kafka:9092 --describe --group forecast-engine
kafka-consumer-groups --bootstrap-server kafka:9092 --describe --group anomaly-detection

# Verifique lag
kafka-consumer-groups --bootstrap-server kafka:9092 --describe --group cost-analytics | grep -i lag

# Verifique tópicos
kafka-topics --bootstrap-server kafka:9092 --describe --topic cost.raw
kafka-topics --bootstrap-server kafka:9092 --describe --topic anomaly.alerts
```

### Validar PostgreSQL

```bash
# Conecte-se
psql -h postgres -U finops -d finops

# Verifique tabelas principais
\dt

# Verifique usuários
SELECT id, email, name, roles, active, last_login FROM users;

# Verifique budgets
SELECT name, amount, spent, (spent/amount)*100 as utilization, alert_threshold FROM budgets;

# Verifique alertas
SELECT status, severity, COUNT(*) FROM alerts GROUP BY status, severity;

# Verifique tamanho do banco
SELECT pg_size_pretty(pg_database_size('finops'));

# Verifique conexões ativas
SELECT count(*) FROM pg_stat_activity WHERE datname = 'finops';
```

### Validar ClickHouse

```bash
# Conecte-se
clickhouse-client

# Verifique tabelas
SHOW TABLES;

# Verifique volume de dados
SELECT 
  table,
  formatReadableSize(sum(bytes)) as size,
  sum(rows) as rows,
  max(modification_time) as latest
FROM system.parts
WHERE active
GROUP BY table
ORDER BY sum(bytes) DESC;

# Verifique partições
SELECT 
  partition,
  formatReadableSize(sum(bytes)) as size,
  sum(rows) as rows
FROM system.parts
WHERE active AND table = 'costs'
GROUP BY partition
ORDER BY partition DESC;

# Verifique queries lentas
SELECT query, query_duration_ms, read_rows, read_bytes
FROM system.query_log
WHERE event_date = today()
ORDER BY query_duration_ms DESC
LIMIT 10;

# Verifique saúde do cluster
SELECT * FROM system.replicas WHERE is_readonly OR is_session_expired OR future_parts > 100 OR parts_to_check > 100;
```

### Validar Forecast Engine

```bash
# Verifique saúde
curl -s http://forecast-engine:8001/health | jq .

# Verifique últimos forecasts
psql -h postgres -U finops -c "SELECT provider, period, model, total_forecast, generated_at FROM forecasts ORDER BY generated_at DESC LIMIT 5"

# Verifique logs
kubectl logs -n finops -l app=forecast-engine --tail=50

# Teste manual
curl -X POST http://forecast-engine:8001/api/v1/forecast   -H "Content-Type: application/json"   -d '{"provider":"huawei","account_id":"hw-001","period":"30d","model":"ensemble"}' | jq .
```

### Validar Anomaly Engine

```bash
# Verifique saúde
curl -s http://anomaly-detection:8002/health | jq .

# Verifique anomalias recentes
psql -h postgres -U finops -c "SELECT date, value, expected, deviation, severity, method FROM anomalies WHERE status='open' ORDER BY date DESC LIMIT 10"

# Verifique no ClickHouse
clickhouse-client -q "SELECT count(), severity FROM anomaly_results WHERE date >= today() - 7 GROUP BY severity"

# Teste manual
curl -X POST http://anomaly-detection:8002/api/v1/anomalies   -H "Content-Type: application/json"   -d '{"provider":"huawei","account_id":"hw-001","method":"ensemble"}' | jq .
```

---

## Troubleshooting

### API Gateway

**Sintoma:** 502 Bad Gateway, timeouts
**Causas:** Serviço downstream indisponível, rate limit atingido, JWT inválido
**Diagnóstico:**
```bash
# Verifique saúde
curl -v http://api-gateway:8080/health
curl -v http://api-gateway:8080/ready

# Verifique logs
kubectl logs -n finops -l app=api-gateway --tail=100

# Verifique rate limit (Redis)
redis-cli -h redis KEYS "rate*"

# Verifique conectividade com downstream
for svc in cost-analytics:8082 alert-manager:8083 forecast-engine:8001 anomaly-detection:8002 recommendation-engine:8003; do
  kubectl exec -n finops -it deployment/api-gateway -- wget -q --spider http://$svc || echo "DOWN: $svc"
done
```
**Correção:**
```bash
# Reinicie o gateway
kubectl rollout restart deployment/api-gateway -n finops

# Se rate limit: limpe Redis
redis-cli -h redis FLUSHDB

# Se downstream indisponível: verifique serviço específico
```

### Ingestion Service

**Sintoma:** Sem dados novos, DLQ crescendo, erros OBS
**Causas:** Credenciais OBS expiradas, checkpoint corrompido, Kafka indisponível, arquivo inválido
**Diagnóstico:**
```bash
# Verifique logs
kubectl logs -n finops -l app=ingestion-service --tail=200 | grep -i "error\|fail\|obs"

# Verifique checkpoints
psql -h postgres -U finops -c "SELECT * FROM ingestion_checkpoints"

# Verifique DLQ
psql -h postgres -U finops -c "SELECT * FROM ingestion_dlq WHERE resolved=false"

# Verifique conectividade Kafka
kubectl exec -n finops -it deployment/ingestion-service -- nc -z kafka 9092

# Verifique credenciais OBS (dentro do pod)
kubectl exec -n finops -it deployment/ingestion-service -- env | grep HUAWEI
```
**Correção:**
```bash
# Reset checkpoint para reprocessamento
curl -X POST http://ingestion-service:8081/ingestion/reprocess   -H "Content-Type: application/json"   -d '{"provider":"huawei","account_id":"hw-001"}'

# Resolva DLQ
psql -h postgres -U finops -c "UPDATE ingestion_dlq SET resolved=true WHERE id='uuid'"

# Reinicie serviço
kubectl rollout restart deployment/ingestion-service -n finops
```

### Cost Analytics

**Sintoma:** Dashboards vazios, queries lentas, erros 500
**Causas:** ClickHouse indisponível, schema desatualizado, Kafka consumer parado
**Diagnóstico:**
```bash
# Verifique ClickHouse
clickhouse-client -q "SELECT 1"

# Verifique tabelas
clickhouse-client -q "SHOW TABLES LIKE 'costs%'"

# Verifique consumer Kafka
kafka-consumer-groups --bootstrap-server kafka:9092 --describe --group cost-analytics

# Verifique logs
kubectl logs -n finops -l app=cost-analytics --tail=100
```
**Correção:**
```bash
# Se ClickHouse indisponível: verifique pod
kubectl get pods -n finops -l app=clickhouse
kubectl describe pod -n finops -l app=clickhouse

# Se consumer parado: reinicie
kubectl rollout restart deployment/cost-analytics -n finops

# Se schema desatualizado: aplique migrations
clickhouse-client < database/clickhouse/schema.sql
```

### PostgreSQL

**Sintoma:** Erros de conexão, queries lentas, disco cheio
**Causas:** Muitas conexões, índices corrompidos, WAL crescendo, disco cheio
**Diagnóstico:**
```bash
# Verifique conexões
psql -h postgres -U finops -c "SELECT count(*), state FROM pg_stat_activity GROUP BY state"

# Verifique locks
psql -h postgres -U finops -c "SELECT * FROM pg_locks WHERE NOT granted"

# Verifique tamanho
psql -h postgres -U finops -c "SELECT pg_size_pretty(pg_database_size('finops'))"
psql -h postgres -U finops -c "SELECT schemaname, tablename, pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) FROM pg_tables WHERE schemaname='public' ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC"

# Verifique WAL
psql -h postgres -U finops -c "SELECT pg_size_pretty(pg_wal_lsn_diff(pg_current_wal_lsn(),'0/0'))"

# Verifique logs do pod
kubectl logs -n finops -l app=postgres --tail=100
```
**Correção:**
```bash
# Se conexões esgotadas: mate idle connections
psql -h postgres -U finops -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE state = 'idle' AND state_change < now() - interval '1 hour'"

# Se disco cheio: expanda volume ou limpe dados antigos
# Kubernetes: edite PVC
kubectl patch pvc postgres-data -n finops -p '{"spec":{"resources":{"requests":{"storage":"20Gi"}}}}'

# Se WAL crescendo: force checkpoint
psql -h postgres -U finops -c "CHECKPOINT"

# Reinicie se necessário
kubectl rollout restart statefulset/postgres -n finops
```

### ClickHouse

**Sintoma:** Queries lentas, partições faltando, memória alta
**Causas:** Muitas partições, índices faltando, memória insuficiente, replicação quebrada
**Diagnóstico:**
```bash
# Verifique memória
clickhouse-client -q "SELECT formatReadableSize(memory_usage) FROM system.processes"

# Verifique partições
clickhouse-client -q "SELECT database, table, partition, formatReadableSize(sum(bytes)) FROM system.parts WHERE active GROUP BY database, table, partition ORDER BY sum(bytes) DESC LIMIT 20"

# Verifique merges
clickhouse-client -q "SELECT * FROM system.merges"

# Verifique replicas
clickhouse-client -q "SELECT database, table, is_readonly, is_session_expired, future_parts, parts_to_check FROM system.replicas"

# Verifique logs
kubectl logs -n finops -l app=clickhouse --tail=100
```
**Correção:**
```bash
# Se memória alta: mate queries pesadas
clickhouse-client -q "KILL QUERY WHERE query_duration_ms > 600000"

# Se partições antigas: otimize
clickhouse-client -q "OPTIMIZE TABLE costs FINAL"

# Se replicação quebrada: ressincronize
clickhouse-client -q "SYSTEM RESTART REPLICA costs"

# Reinicie se necessário
kubectl rollout restart statefulset/clickhouse -n finops
```

### Kafka

**Sintoma:** Consumer lag alto, mensagens não processadas, brokers offline
**Causas:** Disco cheio, zookeeper desconectado, consumer parado, retenção agressiva
**Diagnóstico:**
```bash
# Verifique brokers
kafka-broker-api-versions --bootstrap-server kafka:9092

# Verifique tópicos
kafka-topics --bootstrap-server kafka:9092 --describe

# Verifique consumer lag
kafka-consumer-groups --bootstrap-server kafka:9092 --describe --all-groups

# Verifique disco
kubectl exec -n finops -it kafka-0 -- df -h /bitnami/kafka

# Verifique zookeeper
kubectl logs -n finops -l app=zookeeper --tail=50
```
**Correção:**
```bash
# Se lag alto: escale consumers
kubectl scale deployment/cost-analytics -n finops --replicas=4

# Se disco cheio: expanda PVC ou reduza retenção
kubectl patch pvc kafka-data -n finops -p '{"spec":{"resources":{"requests":{"storage":"50Gi"}}}}'

# Se consumer parado: reinicie
kubectl rollout restart deployment/cost-analytics -n finops

# Se zookeeper desconectado: reinicie
kubectl rollout restart statefulset/zookeeper -n finops
```

### Forecast Engine

**Sintoma:** Forecasts desatualizados, erros de modelo, timeouts
**Causas:** ClickHouse indisponível, modelo quebrado, memória insuficiente
**Diagnóstico:**
```bash
# Verifique saúde
curl -s http://forecast-engine:8001/health

# Verifique logs
kubectl logs -n finops -l app=forecast-engine --tail=100

# Verifique memória
kubectl top pod -n finops -l app=forecast-engine

# Teste manual
curl -X POST http://forecast-engine:8001/api/v1/forecast -d '{"provider":"huawei","period":"30d"}'
```
**Correção:**
```bash
# Reinicie
kubectl rollout restart deployment/forecast-engine -n finops

# Se memória: aumente limites
kubectl patch deployment forecast-engine -n finops -p '{"spec":{"template":{"spec":{"containers":[{"name":"forecast","resources":{"limits":{"memory":"2Gi"}}}]}}}}'
```

### Anomaly Detection

**Sintoma:** Sem anomalias detectadas, falsos positivos, performance ruim
**Causas:** Baseline desatualizada, parâmetros incorretos, modelo desatualizado
**Diagnóstico:**
```bash
# Verifique saúde
curl -s http://anomaly-detection:8002/health

# Verifique anomalias no ClickHouse
clickhouse-client -q "SELECT count(), severity FROM anomaly_results WHERE date >= today() - 7 GROUP BY severity"

# Verifique logs
kubectl logs -n finops -l app=anomaly-detection --tail=100
```
**Correção:**
```bash
# Reinicie
kubectl rollout restart deployment/anomaly-detection -n finops

# Ajuste sensibilidade via API
curl -X POST http://anomaly-detection:8002/api/v1/anomalies -d '{"sensitivity":0.03}'
```

---

## Comandos Prontos

### kubectl

```bash
# Ver pods
kubectl get pods -n finops

# Ver logs
kubectl logs -n finops -l app=api-gateway --tail=100 -f

# Executar comando no pod
kubectl exec -n finops -it deployment/api-gateway -- sh

# Port forward
kubectl port-forward -n finops svc/api-gateway 8080:8080

# Reiniciar deployment
kubectl rollout restart deployment/api-gateway -n finops

# Ver status rollout
kubectl rollout status deployment/api-gateway -n finops

# Descrever pod
kubectl describe pod -n finops <pod-name>

# Ver events
kubectl get events -n finops --sort-by='.lastTimestamp'

# Ver resource usage
kubectl top pods -n finops
kubectl top nodes

# Escalar
kubectl scale deployment cost-analytics -n finops --replicas=4

# Deletar pod (force restart)
kubectl delete pod -n finops -l app=api-gateway

# Ver secrets
kubectl get secrets -n finops
kubectl get secret finops-secrets -n finops -o yaml

# Ver configmaps
kubectl get configmaps -n finops
kubectl get configmap finops-config -n finops -o yaml
```

### helm

```bash
# Instalar
helm install finops deploy/helm/finops-platform -n finops --create-namespace

# Upgrade
helm upgrade finops deploy/helm/finops-platform -n finops

# Rollback
helm rollback finops 1 -n finops

# Ver releases
helm list -n finops

# Ver histórico
helm history finops -n finops

# Ver valores
helm get values finops -n finops

# Desinstalar
helm uninstall finops -n finops

# Debug template
helm template finops deploy/helm/finops-platform -n finops
```

### docker

```bash
# Ver containers
docker ps

# Ver logs
docker logs finops-api-gateway --tail=100 -f

# Executar no container
docker exec -it finops-api-gateway sh

# Reiniciar
docker restart finops-api-gateway

# Ver stats
docker stats

# Limpar volumes não usados
docker volume prune

# Limpar redes não usadas
docker network prune
```

### curl

```bash
# Health checks
curl -s http://localhost:8080/health | jq .
curl -s http://localhost:8081/health | jq .
curl -s http://localhost:8001/health | jq .

# Login
curl -X POST http://localhost:8080/api/v1/auth/login   -H "Content-Type: application/json"   -d '{"email":"admin@finops.local","password":"admin123"}' | jq .

# Forecast
curl -X POST http://localhost:8001/api/v1/forecast   -H "Content-Type: application/json"   -d '{"provider":"huawei","account_id":"hw-001","period":"30d","model":"ensemble"}' | jq .

# Anomalies
curl -X POST http://localhost:8002/api/v1/anomalies   -H "Content-Type: application/json"   -d '{"provider":"huawei","account_id":"hw-001","method":"ensemble"}' | jq .

# Recommendations
curl -X POST http://localhost:8003/api/v1/recommendations   -H "Content-Type: application/json"   -d '{"provider":"huawei","account_id":"hw-001","category":"all"}' | jq .
```

### SQL PostgreSQL

```bash
# Conectar
psql -h postgres -U finops -d finops

# Ver tabelas
\dt

# Ver usuários
SELECT id, email, name, roles, active FROM users;

# Ver budgets
SELECT name, amount, spent, (spent/amount)*100 as utilization FROM budgets;

# Ver alertas
SELECT * FROM alerts WHERE status='firing' ORDER BY created_at DESC;

# Ver checkpoints
SELECT * FROM ingestion_checkpoints;

# Ver DLQ
SELECT * FROM ingestion_dlq WHERE resolved=false;

# Ver recomendações
SELECT category, title, savings, status FROM recommendations WHERE status='open';

# Ver forecasts
SELECT provider, period, model, total_forecast, generated_at FROM forecasts ORDER BY generated_at DESC;

# Ver anomalias
SELECT date, value, expected, deviation, severity FROM anomalies WHERE status='open';

# Ver audit
SELECT * FROM audit ORDER BY created_at DESC LIMIT 50;

# Contar registros
SELECT COUNT(*) FROM users;
SELECT COUNT(*) FROM budgets;
SELECT COUNT(*) FROM alerts;
SELECT COUNT(*) FROM recommendations;

# Backup
pg_dump -h postgres -U finops -d finops > finops_backup_$(date +%Y%m%d).sql

# Restore
psql -h postgres -U finops -d finops < finops_backup_20240101.sql
```

### SQL ClickHouse

```bash
# Conectar
clickhouse-client

# Ver tabelas
SHOW TABLES;

# Ver volume
SELECT table, formatReadableSize(sum(bytes)) as size, sum(rows) as rows 
FROM system.parts WHERE active GROUP BY table ORDER BY size DESC;

# Ver custos hoje
SELECT provider, service_name, sum(effective_cost) as cost 
FROM costs WHERE date = today() GROUP BY provider, service_name;

# Ver custos por mês
SELECT toStartOfMonth(date) as month, provider, sum(effective_cost) as cost 
FROM costs WHERE date >= today() - 365 GROUP BY month, provider ORDER BY month;

# Ver top serviços
SELECT service_name, sum(effective_cost) as cost 
FROM costs WHERE date >= today() - 30 GROUP BY service_name ORDER BY cost DESC LIMIT 10;

# Ver custos por aplicação
SELECT application, sum(effective_cost) as cost 
FROM costs WHERE date >= today() - 30 GROUP BY application ORDER BY cost DESC;

# Ver anomalias
SELECT * FROM anomaly_results WHERE date >= today() - 7 ORDER BY score DESC;

# Ver forecasts
SELECT * FROM forecast_results WHERE forecast_date >= today() ORDER BY forecast_date;

# Otimizar tabela
OPTIMIZE TABLE costs FINAL;

# Ver partições
SELECT partition, formatReadableSize(sum(bytes)) as size 
FROM system.parts WHERE active AND table='costs' GROUP BY partition ORDER BY partition DESC;

# Deletar dados antigos (use com cuidado)
ALTER TABLE costs DELETE WHERE date < '2023-01-01';
```

---

## Disaster Recovery

### Falha do PostgreSQL

**Impacto:** Metadata indisponível, autenticação quebrada, budgets e alertas não acessíveis
**RTO:** 15 minutos
**RPO:** 5 minutos (com backup contínuo)

**Procedimento:**
```bash
# 1. Verifique status do pod
kubectl get pods -n finops -l app=postgres
kubectl describe pod -n finops -l app=postgres

# 2. Se pod crashando, verifique logs
kubectl logs -n finops -l app=postgres --previous

# 3. Se PVC corrompido, restaure do backup
# 3.1. Pare o pod atual
kubectl scale statefulset postgres -n finops --replicas=0

# 3.2. Delete PVC antigo (CUIDADO: dados serão perdidos se não houver backup)
kubectl delete pvc postgres-data -n finops

# 3.3. Recrie PVC e pod
kubectl scale statefulset postgres -n finops --replicas=1

# 3.4. Restaure backup
kubectl cp finops_backup_latest.sql finops/postgres-0:/tmp/
kubectl exec -n finops postgres-0 -- psql -U finops -d finops -f /tmp/finops_backup_latest.sql

# 4. Verifique integridade
kubectl exec -n finops postgres-0 -- psql -U finops -c "SELECT COUNT(*) FROM users"
kubectl exec -n finops postgres-0 -- psql -U finops -c "SELECT COUNT(*) FROM budgets"

# 5. Reinicie serviços dependentes
kubectl rollout restart deployment/api-gateway -n finops
kubectl rollout restart deployment/cost-analytics -n finops
kubectl rollout restart deployment/alert-manager -n finops
```

**Prevenção:**
- Backup automático diário via `pg_dump`
- WAL archiving para PITR
- Replicação streaming para standby
- Monitoramento de disco e conexões

### Falha do ClickHouse

**Impacto:** Analytics indisponíveis, dashboards vazios, forecast/anomaly quebrados
**RTO:** 10 minutos
**RPO:** 0 (dados re-ingestíveis do OBS)

**Procedimento:**
```bash
# 1. Verifique status
kubectl get pods -n finops -l app=clickhouse

# 2. Se pod crashando
kubectl logs -n finops -l app=clickhouse --previous

# 3. Se problema de memória/disco
kubectl top pod -n finops -l app=clickhouse
kubectl exec -n finops -it clickhouse-0 -- df -h

# 4. Reinicie
kubectl rollout restart statefulset/clickhouse -n finops

# 5. Se dados perdidos: re-ingestão
# 5.1. Reset checkpoints
psql -h postgres -U finops -c "UPDATE ingestion_checkpoints SET last_file='', last_offset=0"

# 5.2. Trigger reprocessamento
curl -X POST http://ingestion-service:8081/ingestion/trigger   -d '{"provider":"huawei","bucket":"finops-focus-hw","account_id":"hw-001"}'

# 6. Verifique
clickhouse-client -q "SELECT count() FROM costs"
```

**Prevenção:**
- TTL configurado para retenção automática
- Particionamento mensal para fácil manutenção
- Backup de schema (dados são re-ingestíveis)
- Monitoramento de memória e disco

### Falha do Kafka

**Impacto:** Pipeline de eventos parado, dados não fluem entre serviços
**RTO:** 10 minutos
**RPO:** Dependendo da retenção (padrão: 7 dias)

**Procedimento:**
```bash
# 1. Verifique brokers
kafka-broker-api-versions --bootstrap-server kafka:9092

# 2. Verifique zookeeper
kubectl get pods -n finops -l app=zookeeper
kubectl logs -n finops -l app=zookeeper

# 3. Se zookeeper problemático
kubectl rollout restart statefulset/zookeeper -n finops

# 4. Se kafka problemático
kubectl rollout restart statefulset/kafka -n finops

# 5. Verifique tópicos
kafka-topics --bootstrap-server kafka:9092 --list

# 6. Se tópicos perdidos, recrie
kafka-topics --bootstrap-server kafka:9092 --create --topic cost.raw --partitions 3 --replication-factor 1
kafka-topics --bootstrap-server kafka:9092 --create --topic anomaly.alerts --partitions 3 --replication-factor 1
kafka-topics --bootstrap-server kafka:9092 --create --topic forecast.results --partitions 3 --replication-factor 1
kafka-topics --bootstrap-server kafka:9092 --create --topic recommendations --partitions 3 --replication-factor 1

# 7. Reinicie consumers
kubectl rollout restart deployment/cost-analytics -n finops
kubectl rollout restart deployment/alert-manager -n finops
```

**Prevenção:**
- Múltiplos brokers (3+) em produção
- Zookeeper ensemble (3+)
- Monitoramento de lag
- Alertas para consumer offline

### Falha da Ingestão

**Impacto:** Dados não chegam ao ClickHouse, dashboards desatualizados
**RTO:** 5 minutos
**RPO:** 0 (dados persistem no OBS)

**Procedimento:**
```bash
# 1. Verifique logs
kubectl logs -n finops -l app=ingestion-service --tail=200

# 2. Verifique conectividade OBS
kubectl exec -n finops -it deployment/ingestion-service -- wget -q --spider http://obs.myhwclouds.com

# 3. Verifique credenciais
kubectl exec -n finops -it deployment/ingestion-service -- env | grep HUAWEI

# 4. Verifique checkpoints
psql -h postgres -U finops -c "SELECT * FROM ingestion_checkpoints"

# 5. Se checkpoint corrompido, reset
psql -h postgres -U finops -c "UPDATE ingestion_checkpoints SET last_file='', last_offset=0 WHERE provider='huawei'"

# 6. Reinicie
kubectl rollout restart deployment/ingestion-service -n finops

# 7. Trigger manual
curl -X POST http://ingestion-service:8081/ingestion/trigger   -d '{"provider":"huawei","bucket":"finops-focus-hw","prefix":"exports/","account_id":"hw-001"}'
```

**Prevenção:**
- DLQ para falhas de processamento
- Retry automático com backoff exponencial
- Monitoramento de lag de ingestão
- Alertas para ingestão parada > 1h

### Falha da Integração OBS

**Impacto:** Não é possível obter novos exports FOCUS
**RTO:** 15 minutos
**RPO:** 0 (dados no OBS são persistentes)

**Procedimento:**
```bash
# 1. Verifique credenciais
kubectl get secret finops-secrets -n finops -o yaml | grep huawei

# 2. Atualize credenciais se expiradas
kubectl patch secret finops-secrets -n finops --type='json' -p='[{"op": "replace", "path": "/data/huawei-access-key", "value":"'$(echo -n 'NEW_KEY' | base64)'"}]'

# 3. Verifique endpoint OBS
kubectl exec -n finops -it deployment/ingestion-service -- nslookup obs.myhwclouds.com

# 4. Verifique bucket
# Acesse console Huawei Cloud e verifique:
# - Bucket existe
# - Permissões corretas
# - Arquivos presentes

# 5. Reinicie ingestion
kubectl rollout restart deployment/ingestion-service -n finops
```

**Prevenção:**
- Rotate credentials a cada 90 dias
- Monitoramento de acesso ao bucket
- Alertas para falhas de autenticação
- Backup de credenciais em vault

---

## Backup e Restore

### Backup Completo

```bash
#!/bin/bash
# backup.sh - Executar diariamente via cron

BACKUP_DIR="/backups/finops/$(date +%Y%m%d)"
mkdir -p $BACKUP_DIR

# PostgreSQL
pg_dump -h postgres -U finops -d finops > $BACKUP_DIR/postgres.sql

# ClickHouse Schema
clickhouse-client -q "SHOW CREATE TABLE costs" > $BACKUP_DIR/clickhouse_schema.sql

# Configs Kubernetes
kubectl get configmaps -n finops -o yaml > $BACKUP_DIR/configmaps.yaml
kubectl get secrets -n finops -o yaml > $BACKUP_DIR/secrets.yaml

# Helm values
helm get values finops -n finops > $BACKUP_DIR/helm-values.yaml

# Compress
 tar -czf $BACKUP_DIR.tar.gz $BACKUP_DIR
rm -rf $BACKUP_DIR

# Upload para storage remoto
# aws s3 cp $BACKUP_DIR.tar.gz s3://finops-backups/
# ou
# obsutil cp $BACKUP_DIR.tar.gz obs://finops-backups/

echo "Backup completed: $BACKUP_DIR.tar.gz"
```

### Restore Completo

```bash
#!/bin/bash
# restore.sh - Executar em caso de disaster recovery

BACKUP_FILE="$1"  # Passar arquivo de backup como argumento

# 1. Extrair
tar -xzf $BACKUP_FILE
BACKUP_DIR=$(tar -tzf $BACKUP_FILE | head -1)

# 2. Restore PostgreSQL
psql -h postgres -U finops -c "DROP DATABASE IF EXISTS finops"
psql -h postgres -U finops -c "CREATE DATABASE finops"
psql -h postgres -U finops -d finops < $BACKUP_DIR/postgres.sql

# 3. Restore ClickHouse Schema
clickhouse-client < $BACKUP_DIR/clickhouse_schema.sql

# 4. Re-ingest dados (dados brutos estão no OBS)
# Trigger reprocessamento para todos os providers
curl -X POST http://ingestion-service:8081/ingestion/reprocess

# 5. Restore configs
kubectl apply -f $BACKUP_DIR/configmaps.yaml
kubectl apply -f $BACKUP_DIR/secrets.yaml

echo "Restore completed from: $BACKUP_FILE"
```

---

## Upgrade

### Procedimento de Upgrade sem Perda de Dados

```bash
# 1. Backup completo
./backup.sh

# 2. Verifique saúde atual
kubectl get pods -n finops
for svc in api-gateway cost-analytics alert-manager; do
  curl -s http://$svc:8080/health | jq .
done

# 3. Atualize imagens
# Edite values.yaml ou use set
helm upgrade finops deploy/helm/finops-platform -n finops   --set gateway.image.tag=v1.1.0   --set costAnalytics.image.tag=v1.1.0

# 4. Aguarde rollout
kubectl rollout status deployment/api-gateway -n finops
kubectl rollout status deployment/cost-analytics -n finops

# 5. Verifique saúde pós-upgrade
for svc in api-gateway cost-analytics alert-manager; do
  curl -s http://$svc:8080/health | jq .
done

# 6. Verifique dados
psql -h postgres -U finops -c "SELECT COUNT(*) FROM users"
clickhouse-client -q "SELECT count() FROM costs"

# 7. Se problema, rollback
helm rollback finops -n finops

# 8. Verifique após rollback
kubectl get pods -n finops
```

---

## Observabilidade

### Logs

**Localização:**
- Docker: `docker logs <container>`
- Kubernetes: `kubectl logs -n finops <pod>`
- Loki: http://localhost:3100 (via Grafana)

**Estrutura:**
Todos os serviços logam em JSON:
```json
{"time":"2024-01-15T10:30:00Z","level":"info","service":"api-gateway","message":"request processed","method":"GET","path":"/api/v1/costs","status":200,"duration_ms":45,"user_id":"uuid"}
```

**Queries Loki:**
```logql
# Logs do gateway
{app="api-gateway"}

# Erros do gateway
{app="api-gateway"} |= "error"

# Logs de ingestão
{app="ingestion-service"}

# Erros de ingestão
{app="ingestion-service"} |= "error" or "fail"

# Logs por usuário
{app="api-gateway"} |= "user_id="uuid""

# Logs por path
{app="api-gateway"} |= "path=/api/v1/costs"
```

### Métricas

**Prometheus:** http://localhost:9090

**Métricas principais:**
```promql
# Requisições por segundo
rate(http_requests_total[5m])

# Latência p95
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# Erros 5xx
rate(http_requests_total{status=~"5.."}[5m])

# CPU usage
rate(container_cpu_usage_seconds_total[5m])

# Memória
container_memory_usage_bytes

# Kafka consumer lag
kafka_consumer_group_lag

# PostgreSQL connections
pg_stat_activity_count

# ClickHouse queries
clickhouse_query_duration_ms
```

**Alertas Prometheus:**
```yaml
# Exemplo de regra
- alert: HighErrorRate
  expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.1
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "High error rate detected"

- alert: KafkaConsumerLag
  expr: kafka_consumer_group_lag > 1000
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "Kafka consumer lag is high"

- alert: PostgreSQLConnectionsHigh
  expr: pg_stat_activity_count > 80
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "PostgreSQL connections approaching limit"
```

### Traces

**OpenTelemetry:** Todos os serviços exportam traces para Jaeger/Tempo (configurar collector)

**Ver traces:**
```bash
# Via Jaeger UI (se configurado)
# http://jaeger:16686

# Busque por service name, operation, ou trace ID
```

### Dashboards Grafana

**Dashboards incluídos:**
1. **FinOps - Overview**: KPIs, custos totais, forecast
2. **FinOps - Services**: Custos por serviço, região, ambiente
3. **FinOps - Infrastructure**: CPU, memória, disco dos pods
4. **FinOps - Kafka**: Topics, consumer lag, throughput
5. **FinOps - PostgreSQL**: Connections, queries, locks
6. **FinOps - ClickHouse**: Queries, partições, memória

---

## Checklist Operacional

### Checklist Diário (5 min)
- [ ] Todos os pods em Running: `kubectl get pods -n finops`
- [ ] Health checks passando: `curl */health`
- [ ] Kafka consumer lag < 1000
- [ ] PostgreSQL connections < 80
- [ ] ClickHouse responding
- [ ] Alertas firing < 5
- [ ] Ingestão processando (último checkpoint < 1h)
- [ ] DLQ vazia

### Checklist Semanal (15 min)
- [ ] Revisar anomalias detectadas
- [ ] Revisar recomendações pendentes
- [ ] Verificar budgets próximos do limite
- [ ] Revisar logs de erro (Loki)
- [ ] Verificar uso de recursos (CPU/memória/disco)
- [ ] Revisar métricas de performance (latência p95, p99)
- [ ] Verificar backups automáticos
- [ ] Revisar alertas de infraestrutura

### Checklist Mensal (30 min)
- [ ] Revisar relatório de custos multi-cloud
- [ ] Validar forecast vs realizado
- [ ] Revisar savings opportunities aplicadas
- [ ] Auditar usuários e permissões
- [ ] Revisar e atualizar budgets
- [ ] Verificar retenção de dados (ClickHouse TTL)
- [ ] Revisar e rotacionar credenciais (OBS, APIs)
- [ ] Atualizar documentação operacional
- [ ] Revisar SLAs e metas FinOps
- [ ] Planejar capacity para próximo trimestre
