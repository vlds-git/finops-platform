# UI Architecture - FinOps Enterprise Platform

## Design System

### Cores
- **Background**: `#0f172a` (slate-900)
- **Card**: `#1e293b` (slate-800)
- **Border**: `#334155` (slate-700)
- **Text Primary**: `#e2e8f0` (slate-200)
- **Text Secondary**: `#94a3b8` (slate-400)
- **Accent**: `#38bdf8` (sky-400)
- **Success**: `#4ade80` (green-400)
- **Warning**: `#fbbf24` (amber-400)
- **Danger**: `#f87171` (red-400)

### Tipografia
- Fonte: Inter (Google Fonts)
- Títulos: 600 weight, 1.25rem
- Body: 400 weight, 0.875rem
- Labels: 400 weight, 0.75rem

### Componentes
- **Cards**: rounded-xl, border slate-700, hover border slate-600
- **Buttons**: rounded-lg, sky-600 bg, white text
- **Badges**: rounded-full, color-coded by severity
- **Tables**: divide-y slate-700, hover slate-800
- **Sidebar**: fixed 256px, slate-800 bg

## Navegação

```
Sidebar
├── Dashboard Executivo
├── Dashboard Operacional
├── Forecast
│   ├── 30 Dias
│   ├── 90 Dias
│   └── 12 Meses
├── Anomalias
│   ├── Timeline
│   └── Lista
├── Recomendações
│   ├── Rightsizing
│   ├── Idle Resources
│   ├── Storage
│   └── Savings
├── Budgets
│   ├── Lista
│   └── Novo Budget
├── Alertas
│   ├── Regras
│   ├── Histórico
│   └── Destinatários
├── Usuários
│   ├── Lista
│   └── Permissões
└── Multi-Cloud
    ├── Huawei Cloud
    ├── Azure
    └── AWS
```

## Fluxos

### 1. Login → Dashboard
1. Usuário acessa `/`
2. Redirecionado para login (se não autenticado)
3. JWT armazenado no localStorage
4. Redirecionado para Dashboard Executivo

### 2. Dashboard → Drill-down
1. Usuário clica em KPI "Custo Total"
2. Navega para Dashboard Operacional com filtro aplicado
3. Pode filtrar por: Serviço, Ambiente, Região, Tag

### 3. Forecast → Análise
1. Usuário seleciona período (30/90/365d)
2. Sistema carrega ensemble forecast
3. Exibe Prophet + ARIMA + Holt-Winters comparativo
4. Usuário pode exportar CSV

### 4. Anomalia → Investigação
1. Sistema detecta anomalia via Kafka
2. Alerta aparece no header (badge vermelho)
3. Usuário clica → navega para Anomalias
4. Clica "Investigar" → abre detalhes com contexto

### 5. Recomendação → Aplicação
1. Engine gera recomendação
2. Aparece na lista com: savings, risco, justificativa
3. Usuário clica "Aplicar"
4. Confirmação modal → API call → status update

## Estrutura de Telas

### Dashboard Executivo
- 4 KPI cards (topo)
- Gráfico de tendência (esquerda)
- Gráfico de serviços (direita)
- Top aplicações (esquerda)
- Provedores (centro)
- KPIs estratégicos (direita)

### Dashboard Operacional
- Custos por ambiente
- Custos por região
- Custos por business unit
- Filtros globais (período, provider, conta)

### Forecast
- Seletor de período (tabs)
- Gráfico principal (histórico + forecast + intervalo)
- Métricas do modelo (MAPE, RMSE, Confiança)
- Tabela comparativa por modelo

### Anomalias
- Timeline com scatter plot
- Resumo (total, impacto, baseline)
- Tabela com: data, valor, esperado, desvio, severidade
- Ação: Investigar, Ignorar, Criar Regra

### Recomendações
- Cards expansíveis
- Cada card: título, descrição, savings, justificativa, ação, risco
- Botões: Aplicar, Ignorar, Detalhes
- Filtros por categoria e prioridade

### Budgets
- Cards resumo (orçado, realizado, variação)
- Tabela: nome, período, valores, %, status, alerta
- Botão: Novo Budget
- Alerta visual quando > threshold

### Alertas
- Lista com: indicador de status, título, descrição, severidade, tempo
- Filtros: Todas, Firing, Resolved
- Botão: Nova Regra
- Ação: Resolver

### Usuários
- Tabela: nome, email, perfil, status, último acesso
- Botão: Novo Usuário
- Perfis: Admin, Analyst, Viewer

### Multi-Cloud
- Cards por provedor (logo, contas, custo, %)
- Gráfico comparativo temporal
- Drill-down por provedor

## Responsividade
- Desktop: sidebar fixa + conteúdo fluido
- Tablet: sidebar colapsável
- Mobile: bottom navigation

## Interatividade
- Tooltips em todos os gráficos
- Hover states em cards e tabelas
- Loading skeletons
- Toast notifications para ações
- Modal de confirmação para ações destrutivas
