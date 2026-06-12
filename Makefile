# FinOps Enterprise Platform - Makefile

.PHONY: help build up down logs test clean deploy-k8s deploy-helm

help: ## Mostra esta ajuda
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build todas as imagens Docker
	@echo "Building all services..."
	cd backend/api-gateway && docker build -t finops/api-gateway:latest .
	cd backend/ingestion-service && docker build -t finops/ingestion-service:latest .
	cd backend/cost-analytics && docker build -t finops/cost-analytics:latest .
	cd backend/alert-manager && docker build -t finops/alert-manager:latest .
	cd ml/forecast-engine && docker build -t finops/forecast-engine:latest .
	cd ml/anomaly-detection && docker build -t finops/anomaly-detection:latest .
	cd ml/recommendation-engine && docker build -t finops/recommendation-engine:latest .
	cd frontend/nextjs && docker build -t finops/frontend:latest .
	@echo "Build complete!"

up: ## Sobe a stack com Docker Compose
	docker compose -f docker-compose.yml up -d

down: ## Para a stack com Docker Compose
	docker compose -f docker-compose.yml down

down-volumes: ## Para e remove volumes (CUIDADO: perde dados)
	docker compose -f docker-compose.yml down -v

logs: ## Mostra logs de todos os serviços
	docker compose -f docker-compose.yml logs -f

logs-gateway: ## Logs do API Gateway
	docker compose -f docker-compose.yml logs -f api-gateway

logs-cost: ## Logs do Cost Analytics
	docker compose -f docker-compose.yml logs -f cost-analytics

logs-ml: ## Logs dos serviços ML
	docker compose -f docker-compose.yml logs -f forecast-engine anomaly-detection recommendation-engine

test: ## Executa health checks em todos os serviços
	@echo "Testing health endpoints..."
	@curl -s http://localhost:8080/health | jq . || echo "Gateway FAIL"
	@curl -s http://localhost:8081/health | jq . || echo "Ingestion FAIL"
	@curl -s http://localhost:8082/health | jq . || echo "Cost Analytics FAIL"
	@curl -s http://localhost:8083/health | jq . || echo "Alert Manager FAIL"
	@curl -s http://localhost:8001/health | jq . || echo "Forecast FAIL"
	@curl -s http://localhost:8002/health | jq . || echo "Anomaly FAIL"
	@curl -s http://localhost:8003/health | jq . || echo "Recommendation FAIL"

clean: ## Remove imagens, containers e volumes
	docker system prune -af
	docker volume prune -f

# Kubernetes
deploy-k8s: ## Deploy no Kubernetes
	kubectl apply -f deploy/kubernetes/namespace.yaml
	kubectl apply -f deploy/kubernetes/configmap.yaml
	kubectl apply -f deploy/kubernetes/secret.yaml
	kubectl apply -f deploy/kubernetes/
	kubectl wait --for=condition=ready pod -l app=api-gateway -n finops --timeout=300s

delete-k8s: ## Remove do Kubernetes
	kubectl delete -f deploy/kubernetes/ --ignore-not-found=true

# Helm
deploy-helm: ## Deploy com Helm
	helm dependency build deploy/helm/finops-platform
	hem install finops deploy/helm/finops-platform \
		--namespace finops \
		--create-namespace \
		--wait \
		--timeout 600s

upgrade-helm: ## Upgrade com Helm
	helm upgrade finops deploy/helm/finops-platform \
		--namespace finops \
		--wait \
		--timeout 600s

rollback-helm: ## Rollback Helm
	helm rollback finops -n finops

delete-helm: ## Remove Helm release
	helm uninstall finops -n finops

# Database
migrate-pg: ## Aplica migrations PostgreSQL
	psql -h localhost -U finops -d finops -f database/postgresql/migrations/001_initial_schema.sql

migrate-ch: ## Aplica schema ClickHouse
	clickhouse-client -h localhost < database/clickhouse/schema.sql

# Scripts
seed: ## Popula dados de exemplo
	psql -h localhost -U finops -d finops -c "\COPY (SELECT * FROM users) TO stdout"

# Frontend
serve-frontend: ## Serve frontend localmente (requer Python)
	cd frontend/preview && python3 -m http.server 3000

# Backup
backup: ## Backup completo
	@mkdir -p backups/$(shell date +%Y%m%d)
	pg_dump -h localhost -U finops -d finops > backups/$(shell date +%Y%m%d)/postgres.sql
	clickhouse-client -h localhost -q "SHOW CREATE TABLE costs" > backups/$(shell date +%Y%m%d)/clickhouse_schema.sql
	@echo "Backup saved to backups/$(shell date +%Y%m%d)/"
