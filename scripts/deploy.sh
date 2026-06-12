#!/bin/bash
# deploy.sh - Deploy da plataforma FinOps

set -e

ENVIRONMENT="${ENVIRONMENT:-docker}"
NAMESPACE="${NAMESPACE:-finops}"

echo "========================================="
echo "FinOps Enterprise Platform - Deploy"
echo "Environment: $ENVIRONMENT"
echo "========================================="

case $ENVIRONMENT in
  docker)
    echo "Deploying with Docker Compose..."
    docker compose -f docker-compose.yml up -d --build
    echo ""
    echo "Waiting for services to be healthy..."
    sleep 10
    docker compose -f docker-compose.yml ps
    echo ""
    echo "✓ Deployed!"
    echo "Frontend: http://localhost:3000"
    echo "API: http://localhost:8080"
    echo "Grafana: http://localhost:3001 (admin/admin)"
    ;;

  k8s|kubernetes)
    echo "Deploying to Kubernetes..."
    kubectl apply -f deploy/kubernetes/namespace.yaml
    kubectl apply -f deploy/kubernetes/configmap.yaml
    kubectl apply -f deploy/kubernetes/secret.yaml
    kubectl apply -f deploy/kubernetes/
    echo ""
    echo "Waiting for pods..."
    kubectl wait --for=condition=ready pod -l app=api-gateway -n $NAMESPACE --timeout=300s
    echo ""
    echo "✓ Deployed to Kubernetes!"
    kubectl get pods -n $NAMESPACE
    ;;

  helm)
    echo "Deploying with Helm..."
    helm dependency build deploy/helm/finops-platform
    helm install finops deploy/helm/finops-platform \
      --namespace $NAMESPACE \
      --create-namespace \
      --wait \
      --timeout 600s
    echo ""
    echo "✓ Deployed with Helm!"
    helm list -n $NAMESPACE
    kubectl get pods -n $NAMESPACE
    ;;

  *)
    echo "Unknown environment: $ENVIRONMENT"
    echo "Usage: ./deploy.sh [docker|k8s|helm]"
    exit 1
    ;;
esac

echo "========================================="
