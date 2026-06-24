#!/bin/bash
# Full redeploy da plataforma FinOps a partir do zero.
# ATENCAO: apaga todos os volumes (PostgreSQL, ClickHouse, Kafka, Redis, Prometheus, Grafana).
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_DIR"

echo "============================================================"
echo " FinOps Platform - Redeploy do zero"
echo "============================================================"
echo ""
echo "ATENCAO: este script ira:"
echo "  1. Parar todos os containers"
echo "  2. REMOVER TODOS OS VOLUMES (dados serao apagados)"
echo "  3. Rebuildar todas as imagens"
echo "  4. Subir a stack do zero"
echo "  5. Aplicar migrations do PostgreSQL"
echo ""
read -p "Tem certeza? Digite 'sim' para continuar: " confirm
if [ "$confirm" != "sim" ]; then
  echo "Abortado."
  exit 1
fi

echo ""
echo "[REDEPLOY] 1/8 - Parando e removendo containers/volumes..."
docker compose down -v --remove-orphans

echo ""
echo "[REDEPLOY] 2/8 - Removendo imagens locais antigas..."
docker compose rm -f || true

echo ""
echo "[REDEPLOY] 3/8 - Subindo stack com rebuild..."
docker compose up -d --build

echo ""
echo "[REDEPLOY] 4/8 - Aguardando servicos iniciarem (60s)..."
sleep 60

echo ""
echo "[REDEPLOY] 5/8 - Verificando health dos servicos principais..."
for svc in finops-api-gateway finops-ingestion finops-cost-analytics finops-postgres finops-clickhouse finops-kafka; do
  echo "  - $svc"
  docker inspect --format='{{.State.Status}}' "$svc" || echo "    AVISO: $svc nao encontrado"
done

echo ""
echo "[REDEPLOY] 6/8 - Aplicando migrations do PostgreSQL..."
docker compose cp "$SCRIPT_DIR/../database/postgresql/migrations/001_initial_schema.sql" postgres:/tmp/schema.sql
docker compose exec -T postgres psql -U finops -d finops -f /tmp/schema.sql

echo ""
echo "[REDEPLOY] 7/8 - Verificando conta Huawei configurada..."
docker compose exec -T postgres psql -U finops -d finops -c \
  "SELECT provider, account_id, account_name, bucket, prefix, active FROM cloud_accounts;"

echo ""
echo "[REDEPLOY] 8/8 - Verificando schema ClickHouse..."
docker exec finops-clickhouse clickhouse-client --database=finops -q "SHOW TABLES" || true

echo ""
echo "============================================================"
echo " Redeploy concluido!"
echo "============================================================"
echo ""
echo "Proximos passos:"
echo ""
echo "1. Disparar ingestao inicial:"
echo "   curl -X POST http://localhost:8081/api/v1/ingestion/trigger \\"
echo "     -H 'Content-Type: application/json' \\"
echo "     -d '{\"provider\":\"huawei\",\"bucket\":\"focusfinops\",\"prefix\":\"daily-exports/Daily_Cost_Export_Focus1-0/\",\"account_id\":\"hw-account-001\"}'"
echo ""
echo "2. Acompanhar a ingestao:"
echo "   docker logs -f finops-ingestion"
echo "   docker logs -f finops-cost-analytics"
echo ""
echo "3. Verificar dados no ClickHouse:"
echo "   docker exec finops-clickhouse clickhouse-client --database=finops -q 'SELECT count(), uniqExact(*) FROM costs_raw'"
echo ""
echo "4. Acessar o frontend:"
echo "   http://10.140.12.32:3000"
echo "   Login: admin@finops.local / admin123"
echo ""
