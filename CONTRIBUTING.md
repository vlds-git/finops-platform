# Contributing to FinOps Enterprise Platform

## Desenvolvimento Local

### Pré-requisitos
- Docker 24.0+
- Docker Compose 2.20+
- Go 1.22+
- Python 3.11+
- kubectl 1.28+ (opcional)
- Helm 3.12+ (opcional)

### Setup

```bash
# Clone
git clone <repo>
cd finops-platform

# Copie env
cp .env.example .env

# Suba infraestrutura
cd deploy/docker
docker-compose up -d postgres clickhouse redis kafka

# Instale dependências Go
cd backend/api-gateway && go mod download
cd backend/ingestion-service && go mod download
cd backend/cost-analytics && go mod download
cd backend/alert-manager && go mod download

# Instale dependências Python
cd ml/forecast-engine && pip install -r requirements.txt
cd ml/anomaly-detection && pip install -r requirements.txt
cd ml/recommendation-engine && pip install -r requirements.txt
```

### Executando Testes

```bash
# Go tests
make test
# ou individualmente:
cd backend/api-gateway && go test -v ./...
cd backend/ingestion-service && go test -v ./...
cd backend/cost-analytics && go test -v ./...
cd backend/alert-manager && go test -v ./...

# Python tests
cd ml/forecast-engine && pytest -v
cd ml/anomaly-detection && pytest -v
cd ml/recommendation-engine && pytest -v
```

### Padrões de Código

**Go:**
- Siga `gofmt` e `golint`
- Use `context.Context` para timeouts
- Trate todos os erros explicitamente
- Adicione health checks em novos serviços

**Python:**
- Siga PEP 8
- Use type hints (FastAPI)
- Adicione docstrings
- Use logging estruturado

**Frontend:**
- TailwindCSS para estilização
- ECharts para visualizações
- Componentes reutilizáveis

### Pull Request Process

1. Crie uma branch: `git checkout -b feature/nome-descritivo`
2. Faça commits com mensagens claras
3. Execute todos os testes
4. Atualize documentação se necessário
5. Abra PR com descrição detalhada

### Reportando Bugs

Use o template:
- **Serviço afetado:**
- **Versão:**
- **Ambiente:** (Docker/K8s/Helm)
- **Passos para reproduzir:**
- **Comportamento esperado:**
- **Comportamento atual:**
- **Logs:**
