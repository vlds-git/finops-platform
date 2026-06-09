#!/bin/bash
# build.sh - Build todas as imagens Docker da plataforma FinOps

set -e

REGISTRY="${REGISTRY:-finops}"
TAG="${TAG:-latest}"

echo "========================================="
echo "FinOps Enterprise Platform - Build"
echo "Registry: $REGISTRY"
echo "Tag: $TAG"
echo "========================================="

SERVICES=(
  "backend/api-gateway:api-gateway"
  "backend/ingestion-service:ingestion-service"
  "backend/cost-analytics:cost-analytics"
  "backend/alert-manager:alert-manager"
  "ml/forecast-engine:forecast-engine"
  "ml/anomaly-detection:anomaly-detection"
  "ml/recommendation-engine:recommendation-engine"
)

for service in "${SERVICES[@]}"; do
  IFS=':' read -r path image <<< "$service"
  echo ""
  echo "Building $REGISTRY/$image:$TAG..."
  docker build -t "$REGISTRY/$image:$TAG" "$path"
  echo "✓ $image built successfully"
done

echo ""
echo "========================================="
echo "Build complete! All images:"
docker images | grep "$REGISTRY"
echo "========================================="
