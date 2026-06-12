# Agent Knowledge Base - FinOps Platform

## Regras de Negócio Críticas

- **Integração Huawei OBS é mandatória**: o `ingestion-service` falha na inicialização se `HUAWEI_ACCESS_KEY` e `HUAWEI_SECRET_KEY` não estiverem configurados. Não trate a integração OBS como opcional.
- **Huawei OBS é compatível com S3**: o `ingestion-service` utiliza `minio-go/v7` (cliente S3) para se comunicar com OBS.
- **Endpoint OBS**: deve ser informado sem protocolo (ex: `obs.sa-brazil-1.myhuaweicloud.com`). O cliente usa HTTPS por padrão.
- **Conversão USD→BRL**: todos os valores de custo nos serviços Go e Python devem ser convertidos usando `USD_TO_BRL_RATE`.

## Decisões Técnicas

- Backend em Go usa `go mod tidy` nos Dockerfiles para resolver dependências.
- Frontend Next.js usa `output: 'standalone'` e `NEXT_PUBLIC_API_URL` deve ser passado como `build.arg` no docker-compose.
- ML services usam dados mockados/hardcoded para demonstração, mas aplicam conversão de moeda.
- Docker Compose na raiz é a forma principal de deploy local.

## Arquivos Importantes

- `docker-compose.yml`: stack completa local.
- `backend/ingestion-service/main.go`: integração OBS (mandatória).
- `.env.example`: variáveis de ambiente obrigatórias.
- `deploy/helm/finops-platform/`: Helm chart completo.
- `deploy/kubernetes/`: manifests Kubernetes com infraestrutura.

## Comandos Úteis

```bash
docker compose up -d --build
make test
./scripts/diagnose.sh
```
