# 🚀 FinOps Platform - Deploy com Docker Compose

## Pré-requisitos

- Docker Engine 24.0+
- Docker Compose v2.20+
- 8GB+ RAM disponível
- 20GB+ espaço em disco

## Instruções de Deploy

### 1. Clone o repositório

```bash
git clone https://github.com/vlds-git/finops-platform.git
cd finops-platform
```

### 2. Configure as variáveis de ambiente

```bash
cp .env.example .env
# Edite o .env conforme necessário
```

### 3. Inicie a plataforma

```bash
docker compose up -d --build
```

### 4. Verifique o status

```bash
docker compose ps
docker compose logs -f
```

### 5. Acesse a plataforma

| Serviço | URL |
|---------|-----|
| Frontend | http://localhost:3000 |
| API Gateway | http://localhost:8080 |
| Grafana | http://localhost:3001 (admin/admin) |
| Prometheus | http://localhost:9090 |
| Loki | http://localhost:3100 |

## Comandos úteis

```bash
# Rebuild completo
docker compose down -v
docker compose up -d --build

# Logs de um serviço específico
docker compose logs -f api-gateway

# Reiniciar um serviço
docker compose restart cost-analytics

# Escalar (se necessário)
docker compose up -d --scale api-gateway=2
```

## Troubleshooting

### Problema: Kafka não inicia
- Verifique se o Zookeeper está healthy: `docker compose ps zookeeper`
- Aumente o `start_period` do healthcheck do Kafka se necessário

### Problema: ClickHouse healthcheck falha
- O healthcheck agora usa fallback wget/curl
- Verifique: `docker compose logs clickhouse`

### Problema: Frontend não carrega
- Verifique se o build foi bem-sucedido: `docker compose logs frontend`
- O `next.config.js` deve ter `output: 'standalone'`

### Problema: Serviços Go não respondem ao healthcheck
- Verifique se o Dockerfile instala `wget` ou `curl`
- Os templates de Dockerfile incluem ambos

## Alterações realizadas nesta correção

1. ✅ `docker-compose.yml` movido para raiz do projeto (paths corrigidos)
2. ✅ `next.config.js` - adicionado `output: 'standalone'`
3. ✅ `.env.example` - criado com todas as variáveis necessárias
4. ✅ `init.sql` - schema inicial do PostgreSQL
5. ✅ Zookeeper - adicionado healthcheck
6. ✅ ClickHouse - healthcheck com fallback wget/curl
7. ✅ Healthchecks de todos os serviços - fallback wget/curl
8. ✅ Variáveis Huawei - valores padrão vazios para DEV
9. ✅ Grafana provisioning - datasources e dashboards configurados
10. ✅ Templates de Dockerfile - Go e Python com healthcheck tools
