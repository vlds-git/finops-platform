# Guia de Deploy - FinOps Enterprise Platform

## Requisitos de Infraestrutura

### Mínimo (PoC / Desenvolvimento)
- **CPU**: 8 cores
- **RAM**: 16 GB
- **Disco**: 100 GB SSD
- **Rede**: Acesso à internet para pulls de imagem

### Recomendado (Produção)
- **CPU**: 16+ cores
- **RAM**: 32+ GB
- **Disco**: 500 GB SSD (ClickHouse) + 100 GB (PostgreSQL)
- **Rede**: VPC isolada, load balancer, DNS

### Componentes Pré-existentes (não provisionados)
- VPC / Subnets
- Load Balancer (opcional, para produção)
- DNS / Route53
- IAM / RBAC base

## Deploy com Docker Compose (Desenvolvimento / PoC)

### Passo 1: Preparação

```bash
# Clone o repositório
git clone <repo-url> finops-platform
cd finops-platform

# Verifique Docker e Docker Compose
docker --version
docker-compose --version

# Configure credenciais (opcional para desenvolvimento)
cp .env.example .env
# Edite .env com suas credenciais Huawei OBS (opcional para demo)
```

### Passo 2: Build e Deploy

```bash
cd deploy/docker

# Build das imagens
docker-compose build

# Deploy completo
docker-compose up -d

# Verifique status
docker-compose ps

# Aguarde todos healthy (pode levar 2-3 minutos)
docker-compose ps | grep -c "Up (healthy)"
```

### Passo 3: Validação

```bash
# Teste health checks
for port in 8080 8081 8082 8083 8001 8002 8003; do
  echo "Port $port:"
  curl -s http://localhost:$port/health | jq .
done

# Teste login
curl -X POST http://localhost:8080/api/v1/auth/login   -H "Content-Type: application/json"   -d '{"email":"admin@finops.local","password":"admin123"}'

# Acesse frontend
open http://localhost:3000
```

### Passo 4: Dados Iniciais

```bash
# PostgreSQL já vem com seed data (usuários, budgets, alert rules)
# ClickHouse schema é aplicado automaticamente

# Verifique dados iniciais
docker exec finops-postgres psql -U finops -c "SELECT COUNT(*) FROM users"
docker exec finops-clickhouse clickhouse-client -q "SHOW TABLES"
```

### Passo 5: Configuração OBS (Opcional)

```bash
# Configure variáveis no .env
HUAWEI_ACCESS_KEY=your-access-key
HUAWEI_SECRET_KEY=your-secret-key
HUAWEI_OBS_ENDPOINT=obs.sa-brazil-1.myhuaweicloud.com

# Reinicie ingestion service
docker-compose restart ingestion-service

# Trigger ingestão manual
curl -X POST http://localhost:8081/ingestion/trigger   -H "Content-Type: application/json"   -d '{"provider":"huawei","bucket":"your-bucket","prefix":"exports/","account_id":"hw-001"}'
```

### Paradas e Updates

```bash
# Parar tudo
docker-compose down

# Parar mantendo volumes
docker-compose stop

# Restart
docker-compose restart

# Update imagens
docker-compose pull
docker-compose up -d

# Ver logs
docker-compose logs -f api-gateway
docker-compose logs -f cost-analytics

# Limpar tudo (CUIDADO: perde dados)
docker-compose down -v
```

## Deploy com Kubernetes (Produção)

### Pré-requisitos

- Cluster Kubernetes 1.25+
- kubectl configurado
- Helm 3.12+ (opcional)
- Ingress Controller (nginx ou traefik)
- Storage Class para PVCs

### Passo 1: Namespace e Configurações

```bash
# Crie namespace
kubectl apply -f deploy/kubernetes/namespace.yaml

# Aplique ConfigMap e Secret
kubectl apply -f deploy/kubernetes/configmap.yaml
kubectl apply -f deploy/kubernetes/secret.yaml

# Verifique
kubectl get namespace finops
kubectl get configmaps -n finops
kubectl get secrets -n finops
```

### Passo 2: Infraestrutura (via Helm Charts)

```bash
# Adicione repositórios
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

# PostgreSQL
helm install finops-postgres bitnami/postgresql   --namespace finops   --set auth.username=finops   --set auth.password=finops   --set auth.database=finops   --set primary.persistence.size=20Gi

# Redis
helm install finops-redis bitnami/redis   --namespace finops   --set auth.enabled=false   --set master.persistence.size=10Gi

# Kafka
helm install finops-kafka bitnami/kafka   --namespace finops   --set replicaCount=3   --set persistence.size=50Gi   --set zookeeper.persistence.size=10Gi

# ClickHouse (use chart oficial ou custom)
# helm install finops-clickhouse bitnami/clickhouse ...
# Ou use StatefulSet customizado
kubectl apply -f deploy/kubernetes/clickhouse.yaml  # se disponível

# Aguarde prontidão
kubectl wait --for=condition=ready pod -l app=postgresql -n finops --timeout=300s
kubectl wait --for=condition=ready pod -l app=redis -n finops --timeout=300s
kubectl wait --for=condition=ready pod -l app=kafka -n finops --timeout=300s
```

### Passo 3: Aplicação

```bash
# Aplique serviços na ordem correta
kubectl apply -f deploy/kubernetes/gateway.yaml
kubectl apply -f deploy/kubernetes/ingestion.yaml
kubectl apply -f deploy/kubernetes/cost-analytics.yaml
kubectl apply -f deploy/kubernetes/alert-manager.yaml
kubectl apply -f deploy/kubernetes/forecast-engine.yaml
kubectl apply -f deploy/kubernetes/anomaly-detection.yaml
kubectl apply -f deploy/kubernetes/recommendation-engine.yaml

# Verifique
kubectl get pods -n finops
kubectl get svc -n finops
```

### Passo 4: Ingress e Acesso

```bash
# Aplique ingress
kubectl apply -f deploy/kubernetes/ingress.yaml

# Verifique
kubectl get ingress -n finops

# Configure DNS apontando para o IP do Load Balancer
# ou use /etc/hosts para teste
kubectl get svc -n ingress-nginx  # ou seu ingress controller

# Teste acesso
curl -H "Host: finops.local" http://<ingress-ip>/health
```

### Passo 5: HPA (Horizontal Pod Autoscaler)

```bash
# Aplique HPA para serviços críticos
kubectl apply -f deploy/kubernetes/hpa.yaml

# Verifique
kubectl get hpa -n finops
```

### Passo 6: Validação Pós-Deploy

```bash
# Verifique todos os pods
kubectl get pods -n finops

# Verifique logs de inicialização
kubectl logs -n finops -l app=api-gateway --tail=50
kubectl logs -n finops -l app=cost-analytics --tail=50

# Teste health
curl http://finops.local/health
curl http://finops.local/ready

# Teste login
curl -X POST http://finops.local/api/v1/auth/login   -H "Content-Type: application/json"   -d '{"email":"admin@finops.local","password":"admin123"}'

# Teste forecast
curl -X POST http://finops.local/api/v1/forecast   -H "Content-Type: application/json"   -d '{"provider":"huawei","period":"30d","model":"ensemble"}'
```

## Deploy com Helm (Recomendado para Produção)

### Passo 1: Preparação

```bash
# Adicione repositórios
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

# Build dependências
helm dependency build deploy/helm/finops-platform
```

### Passo 2: Custom Values

```bash
# Crie arquivo de valores customizados
cat > production-values.yaml <<EOF
replicaCount: 2

gateway:
  ingress:
    enabled: true
    className: nginx
    hosts:
      - host: finops.company.com
        paths:
          - path: /
            pathType: Prefix
    annotations:
      cert-manager.io/cluster-issuer: "letsencrypt-prod"
      nginx.ingress.kubernetes.io/rate-limit: "200"
  resources:
    limits:
      cpu: 1000m
      memory: 512Mi
    requests:
      cpu: 250m
      memory: 256Mi

costAnalytics:
  replicaCount: 3
  resources:
    limits:
      cpu: 2000m
      memory: 1Gi

postgresql:
  auth:
    username: finops
    password: "$(openssl rand -base64 32)"
    database: finops
  primary:
    persistence:
      enabled: true
      size: 50Gi
    resources:
      limits:
        cpu: 2000m
        memory: 2Gi

clickhouse:
  auth:
    username: default
    password: "$(openssl rand -base64 32)"
  persistence:
    enabled: true
    size: 200Gi

kafka:
  replicaCount: 3
  persistence:
    enabled: true
    size: 100Gi
EOF
```

### Passo 3: Instalação

```bash
# Instale
helm install finops deploy/helm/finops-platform   --namespace finops   --create-namespace   -f production-values.yaml   --wait   --timeout 600s

# Verifique
helm list -n finops
kubectl get pods -n finops
kubectl get ingress -n finops
```

### Passo 4: Upgrade

```bash
# Edite production-values.yaml
# Execute upgrade
helm upgrade finops deploy/helm/finops-platform   --namespace finops   -f production-values.yaml   --wait   --timeout 600s

# Verifique rollout
kubectl rollout status deployment/finops-api-gateway -n finops
```

### Passo 5: Rollback

```bash
# Verifique histórico
helm history finops -n finops

# Rollback para revisão anterior
helm rollback finops <REVISION> -n finops

# Verifique
helm list -n finops
kubectl get pods -n finops
```

## Configurações de Produção

### Segurança

```yaml
# Secret (não commitar em repositório!)
apiVersion: v1
kind: Secret
metadata:
  name: finops-secrets
  namespace: finops
type: Opaque
stringData:
  jwt-secret: "$(openssl rand -base64 64)"  # 64 bytes random
  postgres-password: "$(openssl rand -base64 32)"
  huawei-access-key: ""
  huawei-secret-key: ""
```

### Network Policies

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: finops-default
  namespace: finops
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: finops
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: finops
```

### Resource Quotas

```yaml
apiVersion: v1
kind: ResourceQuota
metadata:
  name: finops-quota
  namespace: finops
spec:
  hard:
    requests.cpu: "10"
    requests.memory: 20Gi
    limits.cpu: "20"
    limits.memory: 40Gi
    persistentvolumeclaims: "10"
```

### Pod Security Standards

```yaml
# Restricted pod security
apiVersion: v1
kind: Namespace
metadata:
  name: finops
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
```

## Troubleshooting de Deploy

### Problema: Pods em CrashLoopBackOff

```bash
# Verifique logs
kubectl logs -n finops <pod-name> --previous

# Verifique events
kubectl describe pod -n finops <pod-name>

# Causas comuns:
# 1. Variáveis de ambiente faltando
kubectl get pod -n finops <pod-name> -o yaml | grep -A 20 env

# 2. Secret não encontrado
kubectl get secrets -n finops

# 3. ConfigMap não encontrado
kubectl get configmaps -n finops

# 4. Resource limits muito baixos
kubectl top pod -n finops <pod-name>
```

### Problema: Ingress não funciona

```bash
# Verifique ingress
kubectl get ingress -n finops
kubectl describe ingress -n finops finops-ingress

# Verifique ingress controller
kubectl get pods -n ingress-nginx
kubectl logs -n ingress-nginx -l app.kubernetes.io/name=ingress-nginx

# Verifique DNS
nslookup finops.company.com

# Teste direto no service
kubectl port-forward -n finops svc/api-gateway 8080:8080
curl localhost:8080/health
```

### Problema: PostgreSQL não inicia

```bash
# Verifique PVC
kubectl get pvc -n finops
kubectl describe pvc -n finops data-postgres-0

# Verifique storage class
kubectl get storageclass

# Verifique logs
kubectl logs -n finops -l app=postgresql

# Se PVC preso em Pending: verifique provisioner
kubectl get events -n finops | grep -i pvc
```

### Problema: Kafka não conecta

```bash
# Verifique zookeeper
kubectl get pods -n finops -l app=zookeeper
kubectl logs -n finops -l app=zookeeper

# Verifique kafka
kubectl get pods -n finops -l app=kafka
kubectl logs -n finops -l app=kafka

# Teste conectividade
kubectl exec -n finops -it kafka-0 -- kafka-broker-api-versions --bootstrap-server localhost:9092
```

## Checklist de Deploy

### Pré-Deploy
- [ ] Infraestrutura base pronta (VPC, subnets, LB)
- [ ] Kubernetes cluster configurado
- [ ] Storage Class disponível
- [ ] Ingress Controller instalado
- [ ] DNS configurado
- [ ] Credenciais OBS prontas
- [ ] TLS certificates (Let's Encrypt ou custom)

### Deploy
- [ ] Namespace criado
- [ ] Secrets configurados
- [ ] ConfigMaps aplicados
- [ ] Infraestrutura (PostgreSQL, ClickHouse, Redis, Kafka) pronta
- [ ] Aplicação deployada
- [ ] Ingress configurado
- [ ] HPA aplicado
- [ ] Network Policies aplicadas

### Pós-Deploy
- [ ] Health checks passando
- [ ] Login funcionando
- [ ] Dashboards carregando
- [ ] Forecast gerando
- [ ] Anomaly detection funcionando
- [ ] Alertas configurados
- [ ] Backups configurados
- [ ] Monitoramento (Prometheus/Grafana) funcionando
- [ ] Documentação entregue à equipe
- [ ] Treinamento realizado
