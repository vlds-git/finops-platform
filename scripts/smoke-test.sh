#!/usr/bin/env bash
set -euo pipefail

# FinOps Platform — Smoke Test
# Uso: ./scripts/smoke-test.sh [HOST]
# Exemplo: ./scripts/smoke-test.sh http://localhost:8080

HOST="${1:-http://localhost:8080}"

echo "=== FinOps Platform Smoke Test ==="
echo "Host: $HOST"
echo ""

# 1. Healthchecks públicos
echo "[1/6] Healthchecks públicos..."
for path in /health /ready /live; do
  status=$(curl -s -o /dev/null -w "%{http_code}" "${HOST}${path}")
  echo "  ${path} -> ${status}"
  if [ "$status" != "200" ]; then
    echo "  ERRO: ${path} retornou ${status}"
    exit 1
  fi
done

# 2. Login
echo ""
echo "[2/6] Login admin..."
login_resp=$(curl -s -X POST "${HOST}/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@finops.local","password":"admin123"}')
echo "  resposta: ${login_resp}"
TOKEN=$(echo "$login_resp" | python3 -c "import sys,json; print(json.load(sys.stdin).get('token',''))" 2>/dev/null || echo "")
if [ -z "$TOKEN" ]; then
  echo "  ERRO: não foi possível extrair token"
  exit 1
fi
echo "  token obtido com sucesso"

# 3. Rotas de custos
echo ""
echo "[3/6] Rotas de custos..."
for path in /api/v1/costs /api/v1/costs/trends /api/v1/costs/services /api/v1/dashboard/executive /api/v1/dashboard/operational; do
  status=$(curl -s -o /dev/null -w "%{http_code}" "${HOST}${path}" \
    -H "Authorization: Bearer ${TOKEN}")
  echo "  ${path} -> ${status}"
done

# 4. Budgets
echo ""
echo "[4/6] Budgets..."
status=$(curl -s -o /dev/null -w "%{http_code}" "${HOST}/api/v1/budgets" \
  -H "Authorization: Bearer ${TOKEN}")
echo "  GET /api/v1/budgets -> ${status}"

# 5. Alerts
echo ""
echo "[5/6] Alerts..."
status=$(curl -s -o /dev/null -w "%{http_code}" "${HOST}/api/v1/alerts" \
  -H "Authorization: Bearer ${TOKEN}")
echo "  GET /api/v1/alerts -> ${status}"

# 6. Ingestion status
echo ""
echo "[6/6] Ingestion status..."
status=$(curl -s -o /dev/null -w "%{http_code}" "${HOST}/api/v1/ingestion/status" \
  -H "Authorization: Bearer ${TOKEN}")
echo "  GET /api/v1/ingestion/status -> ${status}"

echo ""
echo "=== Smoke Test concluído ==="
