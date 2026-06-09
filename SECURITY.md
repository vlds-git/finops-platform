# Security Policy - FinOps Enterprise Platform

## Versões Suportadas

| Versão | Suportada |
|--------|-----------|
| 1.0.x  | ✅ Sim |

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
- Rate limiting por usuário e IP
- Proteção contra brute force

### Proteção de Dados
- Prepared statements em todas as queries
- Input validation em todos os endpoints
- XSS protection via security headers
- CSRF protection
- SSRF protection

### Infraestrutura
- Security headers (CSP, HSTS, X-Frame-Options)
- Network policies no Kubernetes
- Secrets encriptados (Kubernetes Secrets)
- TLS/HTTPS via Ingress
- Pod Security Standards (restricted)

### Observabilidade
- Audit logging de todas as requisições
- Structured logs com user_id, IP, action
- Prometheus metrics para detecção de anomalias

## Checklist de Segurança para Deploy

- [ ] JWT_SECRET é único e forte (64+ bytes)
- [ ] PostgreSQL password é forte e rotacionada
- [ ] OBS credentials são rotacionadas a cada 90 dias
- [ ] TLS/HTTPS está configurado
- [ ] Network policies estão ativas
- [ ] Rate limiting está configurado
- [ ] Audit logging está funcionando
- [ ] Backups estão criptografados
- [ ] Grafana/Loki não expostos publicamente
