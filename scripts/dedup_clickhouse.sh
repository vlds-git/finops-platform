#!/bin/bash
# Deduplica costs_raw e reseta offsets do Kafka para evitar reprocessamento.
# Uso: ./dedup_clickhouse.sh [clickhouse_container] [kafka_container]
set -e

CH_CONTAINER="${1:-finops-clickhouse}"
KAFKA_CONTAINER="${2:-finops-kafka}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "[DEDUP] Parando ingestion-service e cost-analytics para evitar novas duplicatas..."
docker compose stop ingestion-service cost-analytics || true

echo "[DEDUP] Executando deduplicacao no ClickHouse (container: $CH_CONTAINER)..."
docker exec -i "$CH_CONTAINER" clickhouse-client --database=finops --multiquery < "$SCRIPT_DIR/dedup_clickhouse.sql"

echo "[DEDUP] Resetando offsets do consumer group 'cost-analytics' para latest..."
reset_done=false
for cmd in kafka-consumer-groups.sh kafka-consumer-groups; do
  if docker exec "$KAFKA_CONTAINER" bash -c "command -v $cmd" >/dev/null 2>&1; then
    if docker exec "$KAFKA_CONTAINER" $cmd --bootstrap-server localhost:9092 --group cost-analytics --reset-offsets --to-latest --execute --topic cost.raw; then
      reset_done=true
      break
    fi
  fi
done

if [ "$reset_done" != true ]; then
  echo "[DEDUP] AVISO: Nao foi possivel resetar offsets via kafka-consumer-groups."
  echo "[DEDUP] Tentativa alternativa: deletando consumer group 'cost-analytics'..."
  for cmd in kafka-consumer-groups.sh kafka-consumer-groups; do
    if docker exec "$KAFKA_CONTAINER" bash -c "command -v $cmd" >/dev/null 2>&1; then
      docker exec "$KAFKA_CONTAINER" $cmd --bootstrap-server localhost:9092 --group cost-analytics --delete || true
    fi
  done
  echo "[DEDUP] O grupo sera recriado pelo cost-analytics ao iniciar e comecara do latest."
fi

echo "[DEDUP] Reiniciando servicos..."
docker compose start ingestion-service cost-analytics

echo "[DEDUP] Concluido. Verifique a contagem:"
echo "  docker exec $CH_CONTAINER clickhouse-client --database=finops -q 'SELECT count(), uniqExact(*) FROM costs_raw'"
echo "[DEDUP] Verifique o consumer group:"
echo "  docker exec $KAFKA_CONTAINER kafka-consumer-groups.sh --bootstrap-server localhost:9092 --describe --group cost-analytics"
