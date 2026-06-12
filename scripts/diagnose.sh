#!/bin/bash
# diagnose.sh - Diagnóstico rápido da plataforma

echo "=== FinOps Platform Diagnostics ==="
echo ""

# 1. Docker Compose status
if docker compose -f docker-compose.yml ps &>/dev/null; then
  echo "1. Docker Compose Status:"
  docker compose -f docker-compose.yml ps
fi

# 2. Kubernetes status
if kubectl get namespace finops &>/dev/null; then
  echo ""
  echo "2. Kubernetes Pods:"
  kubectl get pods -n finops
fi

# 3. Health checks
echo ""
echo "3. Health Checks:"
for endpoint in \
  "http://localhost:8080/health:API Gateway" \
  "http://localhost:8081/health:Ingestion" \
  "http://localhost:8082/health:Cost Analytics" \
  "http://localhost:8083/health:Alert Manager" \
  "http://localhost:8001/health:Forecast" \
  "http://localhost:8002/health:Anomaly" \
  "http://localhost:8003/health:Recommendation"; do
  IFS=':' read -r url name <<< "$endpoint"
  status=$(curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null || echo "000")
  if [ "$status" = "200" ]; then
    echo "  ✓ $name: OK"
  else
    echo "  ✗ $name: FAIL ($status)"
  fi
done

# 4. Databases
echo ""
echo "4. Databases:"
echo -n "  PostgreSQL: "
pg_isready -h localhost -p 5432 &>/dev/null && echo "✓ OK" || echo "✗ FAIL"

echo -n "  ClickHouse: "
curl -s -o /dev/null -w "%{http_code}" http://localhost:8124/ping &>/dev/null && echo "✓ OK" || echo "✗ FAIL"

echo -n "  Redis: "
redis-cli -h localhost ping &>/dev/null && echo "✓ OK" || echo "✗ FAIL"

echo -n "  Kafka: "
nc -z localhost 9092 &>/dev/null && echo "✓ OK" || echo "✗ FAIL"

echo ""
echo "=== Diagnostics Complete ==="
