# Guia do Administrador - FinOps Enterprise Platform

## Como Criar Usuários

### Via API

```bash
# 1. Obtenha token de admin
curl -X POST http://localhost:8080/api/v1/auth/login   -H "Content-Type: application/json"   -d '{"email":"admin@finops.local","password":"admin123"}'

# 2. Crie usuário
curl -X POST http://localhost:8080/api/v1/admin/users   -H "Authorization: Bearer <TOKEN>"   -H "Content-Type: application/json"   -d '{
    "email": "novo.usuario@company.com",
    "name": "Novo Usuário",
    "password": "senhaSegura123",
    "roles": ["analyst"]
  }'
```

### Via PostgreSQL

```bash
psql -h postgres -U finops -d finops

INSERT INTO users (email, name, password_hash, roles) VALUES
('novo.usuario@company.com', 'Novo Usuário', 'senhaSegura123', ARRAY['analyst']);
```

### Perfis Disponíveis

| Perfil | Permissões |
|--------|-----------|
| **Admin** | Acesso total. Criar usuários, budgets, alertas, contas cloud. Configurar OBS. |
| **Analyst** | Ver custos, forecast, anomalias, recomendações. Criar budgets e alertas. |
| **Viewer** | Apenas visualização. Dashboards, relatórios, budgets. Sem ação. |

### Integração Futura com IdP

Para Microsoft Entra ID, LDAP ou Keycloak:

1. Configure o IdP no `api-gateway` via variáveis de ambiente:
```yaml
env:
  - name: IDP_PROVIDER
    value: "azure"  # ou "ldap", "keycloak"
  - name: IDP_CLIENT_ID
    value: "your-client-id"
  - name: IDP_CLIENT_SECRET
    valueFrom:
      secretKeyRef:
        name: finops-secrets
        key: idp-client-secret
  - name: IDP_TENANT_ID
    value: "your-tenant-id"
  - name: IDP_REDIRECT_URL
    value: "https://finops.local/auth/callback"
```

2. Mapeie grupos do IdP para roles:
```yaml
env:
  - name: IDP_GROUP_ADMIN
    value: "FinOps-Admins"
  - name: IDP_GROUP_ANALYST
    value: "FinOps-Analysts"
  - name: IDP_GROUP_VIEWER
    value: "FinOps-Viewers"
```

---

## Como Criar Budgets

### Via API

```bash
curl -X POST http://localhost:8080/api/v1/budgets   -H "Authorization: Bearer <TOKEN>"   -H "Content-Type: application/json"   -d '{
    "name": "Produção - Q2 2024",
    "amount": 200000.00,
    "currency": "BRL",
    "period": "quarterly",
    "start_date": "2024-04-01",
    "end_date": "2024-06-30",
    "alert_threshold": 0.85,
    "provider": "all",
    "account_id": "all",
    "service": "all",
    "tags": {"environment": "production"}
  }'
```

### Via PostgreSQL

```bash
psql -h postgres -U finops -d finops

INSERT INTO budgets (name, amount, period, start_date, end_date, alert_threshold, provider, account_id) VALUES
('Produção - Q2 2024', 200000.00, 'quarterly', '2024-04-01', '2024-06-30', 0.85, 'all', 'all');
```

### Campos do Budget

| Campo | Descrição | Exemplo |
|-------|-----------|---------|
| `name` | Nome descritivo | "Produção - Q2 2024" |
| `amount` | Valor orçado | 200000.00 |
| `currency` | Moeda | "BRL", "USD" |
| `period` | Período | "monthly", "quarterly", "yearly" |
| `start_date` | Início | "2024-04-01" |
| `end_date` | Fim | "2024-06-30" |
| `alert_threshold` | % para alerta | 0.85 = 85% |
| `provider` | Provedor filtro | "all", "huawei", "azure", "aws" |
| `account_id` | Conta filtro | "all", "hw-001" |
| `service` | Serviço filtro | "all", "Compute" |
| `tags` | Tags filtro | `{"environment": "production"}` |

### Alertas de Budget

Quando `spent/amount >= alert_threshold`, o sistema gera alerta automaticamente.

Configure notificações no Alert Manager para receber avisos.

---

## Como Criar Alertas

### Via API

```bash
curl -X POST http://localhost:8080/api/v1/alerts   -H "Authorization: Bearer <TOKEN>"   -H "Content-Type: application/json"   -d '{
    "name": "Cost Spike Alert",
    "description": "Alerta quando custo diário excede 150% da média",
    "condition": "cost_spike",
    "threshold": 1.5,
    "severity": "high",
    "channel": "slack",
    "destination": "https://hooks.slack.com/services/YOUR/WEBHOOK/URL",
    "enabled": true
  }'
```

### Condições Disponíveis

| Condição | Descrição | Threshold |
|----------|-----------|-----------|
| `budget_utilization` | % do budget utilizado | 0.90 = 90% |
| `cost_spike` | Múltiplo da média | 2.0 = 200% |
| `forecast_exceed` | Forecast vs budget | 1.05 = 105% |
| `anomaly_detected` | Anomalia detectada | Qualquer severidade |
| `idle_resource` | Recurso ocioso > N dias | 30 dias |

### Canais de Notificação

| Canal | Destination | Configuração |
|-------|-------------|--------------|
| `slack` | Webhook URL | Crie em Slack Apps > Incoming Webhooks |
| `email` | Endereço de email | Configure SMTP no Alert Manager |
| `webhook` | URL customizada | Endpoint que recebe POST JSON |
| `teams` | Webhook URL | Similar ao Slack |

### Via PostgreSQL

```bash
psql -h postgres -U finops -d finops

INSERT INTO alert_rules (name, description, condition, threshold, severity, channel, destination, enabled) VALUES
('Budget Alert', 'Alerta de budget', 'budget_utilization', 0.90, 'critical', 'slack', 'https://hooks.slack.com/...', true);
```

---

## Como Cadastrar Contas Cloud

### Via API

```bash
curl -X POST http://localhost:8080/api/v1/admin/cloud-accounts   -H "Authorization: Bearer <TOKEN>"   -H "Content-Type: application/json"   -d '{
    "provider": "huawei",
    "account_id": "hw-account-002",
    "account_name": "Desenvolvimento Huawei",
    "access_key": "AKIA...",
    "secret_key": "secret...",
    "endpoint": "obs.sa-brazil-1.myhuaweicloud.com",
    "bucket": "finops-focus-hw-dev",
    "prefix": "exports/",
    "active": true
  }'
```

### Provedores Suportados

| Provedor | `provider` | Campos Específicos |
|----------|-----------|-------------------|
| Huawei Cloud | `huawei` | `endpoint`, `bucket`, `prefix` |
| Azure | `azure` | `subscription_id`, `storage_account`, `container` |
| AWS | `aws` | `region`, `bucket`, `prefix`, `role_arn` |

### Via PostgreSQL

```bash
psql -h postgres -U finops -d finops

INSERT INTO cloud_accounts (provider, account_id, account_name, access_key, secret_key, endpoint, bucket, prefix, active) VALUES
('huawei', 'hw-account-002', 'Desenvolvimento Huawei', 'AKIA...', 'secret...', 'obs.sa-brazil-1.myhuaweicloud.com', 'finops-focus-hw-dev', 'exports/', true);
```

---

## Como Configurar OBS

### Pré-requisitos

1. Conta Huawei Cloud ativa
2. Bucket OBS criado
3. IAM User com permissões de leitura
4. FOCUS exports configurados no provedor

### Passo a Passo

1. **Crie o bucket no console Huawei Cloud:**
   - Navegue para Object Storage Service
   - Crie bucket: `finops-focus-hw-prod`
   - Região: `sa-brazil-1`
   - Storage Class: `Standard`
   - Versioning: `Enabled`

2. **Configure lifecycle:**
   - Transition to Warm: 90 dias
   - Transition to Cold: 365 dias
   - Expire: 2555 dias (7 anos)

3. **Crie IAM User:**
   - Nome: `finops-ingestion`
   - Política: `OBS ReadOnly`
   - Guarde Access Key e Secret Key

4. **Configure FOCUS Export:**
   - No console de billing do provedor, configure export para OBS
   - Formato: CSV ou Parquet
   - Frequência: Diária
   - Prefixo: `exports/YYYY/MM/DD/`

5. **Cadastre na plataforma:**
   - Use API ou PostgreSQL (seção acima)
   - Preencha: `endpoint`, `bucket`, `prefix`, `access_key`, `secret_key`

6. **Teste a conexão:**
   ```bash
   curl -X POST http://localhost:8080/api/v1/ingestion/trigger      -H "Authorization: Bearer <TOKEN>"      -H "Content-Type: application/json"      -d '{"provider":"huawei","bucket":"finops-focus-hw-prod","prefix":"exports/","account_id":"hw-account-001"}'
   ```

7. **Verifique status:**
   ```bash
   curl -H "Authorization: Bearer <TOKEN>"      http://localhost:8080/api/v1/ingestion/status
   ```

---

## Como Configurar Forecast

### Via API

```bash
# Configure parâmetros do forecast engine
curl -X POST http://localhost:8001/api/v1/forecast   -H "Content-Type: application/json"   -d '{
    "provider": "huawei",
    "account_id": "hw-001",
    "period": "30d",
    "model": "ensemble"
  }'
```

### Modelos Disponíveis

| Modelo | Descrição | Quando Usar |
|--------|-----------|-------------|
| `prophet` | Facebook Prophet. Tendência + sazonalidade. | Dados com sazonalidade clara |
| `arima` | AutoRegressive Integrated Moving Average. | Dados estacionários |
| `holt-winters` | Exponential Smoothing. Tendência + sazonalidade. | Dados com tendência |
| `ensemble` | Média dos 3 modelos. | Padrão. Melhor accuracy |

### Períodos

| Período | Uso | Precisão |
|---------|-----|----------|
| `30d` | Operação diária | Alta (MAPE ~4%) |
| `90d` | Planejamento trimestral | Média (MAPE ~8%) |
| `12m` | Planejamento anual | Baixa (MAPE ~15%) |

### Agendamento Automático

Configure cron job para gerar forecast diariamente:

```bash
# Kubernetes CronJob
cat <<EOF | kubectl apply -f -
apiVersion: batch/v1
kind: CronJob
metadata:
  name: forecast-daily
  namespace: finops
spec:
  schedule: "0 6 * * *"  # 6 AM daily
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: forecast
            image: curlimages/curl:latest
            command:
            - sh
            - -c
            - |
              curl -X POST http://forecast-engine:8001/api/v1/forecast                 -H "Content-Type: application/json"                 -d '{"provider":"all","period":"30d","model":"ensemble"}'
          restartPolicy: OnFailure
EOF
```

---

## Como Configurar Anomalias

### Via API

```bash
# Configure detecção de anomalias
curl -X POST http://localhost:8002/api/v1/anomalies   -H "Content-Type: application/json"   -d '{
    "provider": "huawei",
    "account_id": "hw-001",
    "method": "ensemble",
    "sensitivity": 0.05,
    "window": 7
  }'
```

### Métodos Disponíveis

| Método | Descrição | Quando Usar |
|--------|-----------|-------------|
| `isolation_forest` | ML não supervisionado. Detecta outliers multivariados. | Dados complexos, múltiplas dimensões |
| `zscore` | Estatístico. Desvio padrão da média. | Dados normais, simples |
| `rolling_average` | Média móvel com bandas. | Dados com tendência suave |
| `ensemble` | Combinação dos 3. | Padrão. Melhor coverage |

### Parâmetros

| Parâmetro | Descrição | Range | Padrão |
|-----------|-----------|-------|--------|
| `sensitivity` | Sensibilidade da detecção | 0.01 - 0.20 | 0.05 |
| `window` | Janela para rolling average | 3 - 30 dias | 7 |

- **Sensitivity baixa (0.01)**: Menos falsos positivos, pode perder anomalias sutis
- **Sensitivity alta (0.20)**: Mais sensível, mais falsos positivos

### Agendamento Automático

```bash
# Kubernetes CronJob
cat <<EOF | kubectl apply -f -
apiVersion: batch/v1
kind: CronJob
metadata:
  name: anomaly-daily
  namespace: finops
spec:
  schedule: "0 7 * * *"  # 7 AM daily
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: anomaly
            image: curlimages/curl:latest
            command:
            - sh
            - -c
            - |
              curl -X POST http://anomaly-detection:8002/api/v1/anomalies                 -H "Content-Type: application/json"                 -d '{"provider":"all","method":"ensemble","sensitivity":0.05}'
          restartPolicy: OnFailure
EOF
```

---

## Checklist de Configuração Inicial

- [ ] Criar usuário admin
- [ ] Cadastrar contas cloud (Huawei, Azure, AWS)
- [ ] Configurar OBS (bucket, IAM, lifecycle)
- [ ] Configurar FOCUS exports nos provedores
- [ ] Criar budgets para cada ambiente
- [ ] Criar alertas de budget e anomalias
- [ ] Configurar canais de notificação (Slack/Email)
- [ ] Testar ingestão manual
- [ ] Verificar dados no ClickHouse
- [ ] Configurar forecast automático (CronJob)
- [ ] Configurar anomaly detection automático (CronJob)
- [ ] Configurar dashboards Grafana
- [ ] Configurar alertas Prometheus
- [ ] Documentar equipe e responsáveis
- [ ] Treinar usuários (Analysts e Viewers)

---

## Comandos Úteis para Administradores

```bash
# Ver todos os usuários
psql -h postgres -U finops -c "SELECT id, email, name, roles, active, last_login FROM users ORDER BY created_at"

# Ver todos os budgets
psql -h postgres -U finops -c "SELECT name, amount, spent, (spent/amount)*100 as pct, alert_threshold FROM budgets"

# Ver todas as regras de alerta
psql -h postgres -U finops -c "SELECT name, condition, threshold, severity, channel, enabled FROM alert_rules"

# Ver todas as contas cloud
psql -h postgres -U finops -c "SELECT provider, account_id, account_name, bucket, active FROM cloud_accounts"

# Ver audit trail
psql -h postgres -U finops -c "SELECT user_id, action, resource, details, created_at FROM audit ORDER BY created_at DESC LIMIT 50"

# Resetar senha de usuário
psql -h postgres -U finops -c "UPDATE users SET password_hash='novaSenha' WHERE email='usuario@company.com'"

# Desativar usuário
psql -h postgres -U finops -c "UPDATE users SET active=false WHERE email='usuario@company.com'"

# Verificar tamanho do banco
psql -h postgres -U finops -c "SELECT pg_size_pretty(pg_database_size('finops'))"

# Verificar conexões ativas
psql -h postgres -U finops -c "SELECT count(*), state FROM pg_stat_activity WHERE datname='finops' GROUP BY state"
```
