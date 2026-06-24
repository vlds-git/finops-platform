# Guia do Administrador - FinOps Enterprise Platform

## Como Criar Usuários

### Via Frontend

1. Acesse **Usuários** no menu lateral.
2. Clique em **Novo Usuário**.
3. Preencha nome, email, senha e perfis (admin, analyst, viewer).
4. Clique em **Salvar**.

### Via API

```bash
# 1. Obtenha token de admin
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@finops.local","password":"admin123"}'

# 2. Crie usuário
curl -X POST http://localhost:8080/api/v1/admin/users \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "novo.usuario@company.com",
    "name": "Novo Usuário",
    "password": "senhaSegura123",
    "roles": ["analyst"],
    "active": true
  }'

# 3. Atualizar usuário
curl -X PUT http://localhost:8080/api/v1/admin/users/<ID> \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "novo.usuario@company.com",
    "name": "Novo Usuário",
    "roles": ["analyst", "viewer"],
    "active": true
  }'

# 4. Remover usuário
curl -X DELETE http://localhost:8080/api/v1/admin/users/<ID> \
  -H "Authorization: Bearer <TOKEN>"
```

> **Atenção:** no ciclo atual, senhas ainda são armazenadas em plaintext no PostgreSQL. A substituição por hash (bcrypt/Argon2) está no `ROADMAP.md` como item P0.

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

### Via Frontend

1. Acesse **Budgets** no menu lateral.
2. Clique em **Novo Budget**.
3. Preencha nome, valor orçado, período, datas de início/fim e threshold de alerta.
4. Selecione a **account** obrigatoriamente no dropdown (dados vindos de `/api/v1/accounts`).
5. Clique em **Salvar**.

O campo `spent` é calculado automaticamente pelo `cost-analytics` a partir do ClickHouse, filtrando por `provider` e `billing_account_id` dentro do período do budget.

### Via API

```bash
# 1. Listar accounts disponíveis
curl http://localhost:8080/api/v1/accounts \
  -H "Authorization: Bearer <TOKEN>"

# 2. Criar budget vinculado a uma account
curl -X POST http://localhost:8080/api/v1/budgets \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Produção - Q2 2024",
    "amount": 200000.00,
    "period": "quarterly",
    "start_date": "2024-04-01",
    "end_date": "2024-06-30",
    "alert_threshold": 0.85,
    "provider": "huawei",
    "account_id": "hw-account-001"
  }'

# 3. Atualizar budget
curl -X PUT http://localhost:8080/api/v1/budgets/<ID> \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Produção - Q2 2024",
    "amount": 250000.00,
    "alert_threshold": 0.80
  }'

# 4. Remover budget
curl -X DELETE http://localhost:8080/api/v1/budgets/<ID> \
  -H "Authorization: Bearer <TOKEN>"
```

### Campos do Budget

| Campo | Descrição | Exemplo |
|-------|-----------|---------|
| `name` | Nome descritivo | "Produção - Q2 2024" |
| `amount` | Valor orçado | 200000.00 |
| `period` | Período | "monthly", "quarterly", "yearly", "custom" |
| `start_date` | Início | "2024-04-01" |
| `end_date` | Fim | "2024-06-30" |
| `alert_threshold` | % para alerta | 0.85 = 85% |
| `provider` | Provedor | "huawei" |
| `account_id` | Conta obrigatória | "hw-account-001" |

### Status do Budget

| Status | Condição |
|--------|----------|
| On Track | `spent/amount < alert_threshold` |
| Warning | `spent/amount >= alert_threshold` e `<= 100%` |
| Over Budget | `spent/amount > 100%` |

> Alertas serão reintroduzidos vinculados aos budgets (ver `ROADMAP.md`). A página de Alertas foi removida do frontend.

---

## Como Criar Alertas

> **A página de Alertas foi removida do frontend.** A funcionalidade será reintroduzida de forma integrada aos budgets: quando `spent/amount >= alert_threshold`, o sistema publicará um evento e notificará via Alert Manager.
>
> Enquanto isso, o backend `alert-manager` continua disponível para criação manual de regras via API, se necessário.

### API do Alert Manager (backend)

```bash
curl -X POST http://localhost:8080/api/v1/alerts \
  -H "Authorization: Bearer <TOKEN>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Budget Threshold",
    "description": "Alerta quando budget atinge 85%",
    "condition": "budget_utilization",
    "threshold": 0.85,
    "severity": "high",
    "channel": "slack",
    "destination": "https://hooks.slack.com/services/YOUR/WEBHOOK/URL",
    "enabled": true
  }'
```

### Canais de Notificação

| Canal | Destination | Configuração |
|-------|-------------|--------------|
| `slack` | Webhook URL | Crie em Slack Apps > Incoming Webhooks |
| `email` | Endereço de email | Configure SMTP no Alert Manager |
| `webhook` | URL customizada | Endpoint que recebe POST JSON |
| `teams` | Webhook URL | Similar ao Slack |

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
    "bucket": "focusfinops",
    "prefix": "daily-exports/Daily_Cost_Export_Focus1-0/",
    "active": true
  }'
```

### Provedores Suportados

| Provedor | `provider` | Campos Específicos |
|----------|-----------|-------------------|
| Huawei Cloud | `huawei` | `endpoint`, `bucket`, `prefix` |

> Azure e AWS foram removidos do escopo atual. O ambiente é Huawei-only.

### Via PostgreSQL

```bash
psql -h postgres -U finops -d finops

INSERT INTO cloud_accounts (provider, account_id, account_name, access_key, secret_key, endpoint, bucket, prefix, active) VALUES
('huawei', 'hw-account-002', 'Desenvolvimento Huawei', 'AKIA...', 'secret...', 'obs.sa-brazil-1.myhuaweicloud.com', 'focusfinops', 'daily-exports/Daily_Cost_Export_Focus1-0/', true);
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
   - Crie bucket: `focusfinops`
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
   - Prefixo: `daily-exports/Daily_Cost_Export_Focus1-0/`

5. **Cadastre na plataforma:**
   - Use API ou PostgreSQL (seção acima)
   - Preencha: `endpoint`, `bucket`, `prefix`, `access_key`, `secret_key`

6. **Teste a conexão:**
   ```bash
   curl -X POST http://localhost:8080/api/v1/ingestion/trigger      -H "Authorization: Bearer <TOKEN>"      -H "Content-Type: application/json"      -d '{"provider":"huawei","bucket":"focusfinops","prefix":"daily-exports/Daily_Cost_Export_Focus1-0/","account_id":"hw-account-001"}'
   ```

7. **Verifique status:**
   ```bash
   curl -H "Authorization: Bearer <TOKEN>"      http://localhost:8080/api/v1/ingestion/status
   ```

---

## Como Configurar Forecast

### Via Frontend

1. Acesse **Forecast** no menu lateral.
2. Selecione o período de histórico no header (ex: 30d).
3. Escolha o horizonte de previsão: 30, 60 ou 90 dias.
4. Visualize gráfico histórico + projeção e métricas resumidas.

### Via API

O forecast é servido pelo `cost-analytics` via API Gateway:

```bash
# Forecast com base nos últimos 30 dias
curl "http://localhost:8080/api/v1/forecast?start_date=2024-05-01&end_date=2024-05-30&forecast_days=30&currency=BRL" \
  -H "Authorization: Bearer <TOKEN>"
```

### Modelo Atual

| Modelo | Descrição | Status |
|--------|-----------|--------|
| Linear Trend | Regressão linear sobre custos diários do ClickHouse | **Ativo** |
| Prophet / ARIMA / Holt-Winters | Modelos estatísticos avançados | Em standby (ver `ROADMAP.md`) |

---

## Como Configurar Anomalias

### Via Frontend

1. Acesse **Anomalias** no menu lateral.
2. Selecione o período no header.
3. Visualize timeline, resumo e tabela de anomalias detectadas.

### Via API

As anomalias são calculadas pelo `cost-analytics` via API Gateway:

```bash
curl "http://localhost:8080/api/v1/anomalies?start_date=2024-05-01&end_date=2024-05-30&currency=BRL" \
  -H "Authorization: Bearer <TOKEN>"
```

### Método Atual

| Método | Descrição | Status |
|--------|-----------|--------|
| Z-Score | Desvio padrão da média dos custos diários | **Ativo** |
| Isolation Forest / Rolling Average | Algoritmos avançados | Em standby (ver `ROADMAP.md`) |

---

## Checklist de Configuração Inicial

- [ ] Criar usuário admin
- [ ] Cadastrar contas cloud (Huawei)
- [ ] Configurar OBS (bucket, IAM, lifecycle)
- [ ] Configurar FOCUS exports diários
- [ ] Criar budgets vinculados a accounts
- [ ] Testar ingestão manual via `/api/v1/ingestion/trigger`
- [ ] Verificar dados no ClickHouse (`finops.costs_raw`)
- [ ] Validar dashboards (períodos e moeda USD/BRL)
- [ ] Configurar dashboards Grafana
- [ ] Configurar alertas Prometheus
- [ ] Documentar equipe e responsáveis
- [ ] Treinar usuários (Analysts e Viewers)

> Forecast e anomalias já funcionam online a partir dos dados do ClickHouse. Não é necessário CronJob inicial.

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

# ClickHouse - contagem total e distinta (detectar duplicatas)
clickhouse-client --database=finops -q "SELECT count(), uniqExact(*) FROM costs_raw"

# ClickHouse - deduplicar tabela costs_raw (execute com cuidado; para o ambiente Docker Compose use scripts/dedup_clickhouse.sh)
clickhouse-client --database=finops --multiquery < scripts/dedup_clickhouse.sql
```
