# Security Policy - FinOps Enterprise Platform

## Versões Suportadas

| Versão | Suportada |
|--------|-----------|
| 1.0.x  | ✅ Sim |
| 1.1.x-pré | ⚠️ Em desenvolvimento |

## Reportando Vulnerabilidades

**NÃO abra issues públicas para vulnerabilidades de segurança.**

Envie um email para: security@company.com

Inclua:
- Descrição da vulnerabilidade
- Passos para reproduzir
- Impacto potencial
- Sugestão de mitigação (opcional)

Resposta em até 48 horas.

## Medidas de Segurança Implementadas

### Autenticação e Autorização
- JWT com expiração de 24h
- RBAC com 3 perfis (Admin, Analyst, Viewer)
- Rate limiting por usuário (local por instância)

### Proteção de Dados
- Prepared statements nas queries PostgreSQL
- Queries ClickHouse usam placeholders para filtros de provider/account
- XSS protection via security headers
- CORS configurável (atualmente permissivo em desenvolvimento)

### Infraestrutura
- Security headers (CSP, HSTS, X-Frame-Options)
- Health checks (`/health`, `/ready`, `/live`, `/metrics`)
- Kubernetes Secrets para credenciais

### Observabilidade
- Audit logging de requisições no gateway
- Structured logs com user_id, IP, action
- Prometheus metrics básicos

## Riscos Conhecidos e Correções em Andamento

| Risco | Severidade | Status | Ação |
|-------|------------|--------|------|
| Senhas armazenadas em plaintext no PostgreSQL | 🔴 Alta | Em aberto | Implementar bcrypt/Argon2 (ver `ROADMAP.md` Fase 1) |
| Login hardcoded no gateway (`admin@finops.local` / `admin123`) | 🔴 Alta | Em aberto | Validar contra PostgreSQL (ver `ROADMAP.md` Fase 1) |
| Roles e permissões hardcoded no código | 🟡 Média | Em aberto | Mover para PostgreSQL (ver `ROADMAP.md` Fase 1) |
| Queries ClickHouse dinâmicas (`fmt.Sprintf`) | 🟡 Média | Parcial | `start_date`/`end_date` validados; melhorias pendentes |
| CORS permissivo (`*`) | 🟡 Média | Em aberto | Tornar configurável (ver `ROADMAP.md` Fase 1) |
| Credenciais em `.env` / `values.yaml` | 🟡 Média | Em aberto | Adotar External Secrets/Vault (ver `ROADMAP.md` Fase 1) |
| Sem NetworkPolicies no Helm | 🟡 Média | Em aberto | Adicionar (ver `ROADMAP.md` Fase 2) |
| Sem TLS no Ingress do Helm | 🟡 Média | Em aberto | Adicionar cert-manager (ver `ROADMAP.md` Fase 2) |

## Checklist de Segurança para Deploy

- [ ] JWT_SECRET é único e forte (64+ bytes)
- [ ] PostgreSQL password é forte e rotacionada
- [ ] ClickHouse password é forte e rotacionada
- [ ] OBS credentials são rotacionadas a cada 90 dias
- [ ] Senhas dos usuários estão hasheadas (bcrypt/Argon2)
- [ ] Login valida credenciais no PostgreSQL
- [ ] TLS/HTTPS está configurado
- [ ] Network policies estão ativas
- [ ] Rate limiting está configurado
- [ ] CORS restringe origens permitidas
- [ ] Audit logging está funcionando
- [ ] Backups estão criptografados
- [ ] Grafana/Loki não expostos publicamente
- [ ] Secrets não estão commitados no repositório
