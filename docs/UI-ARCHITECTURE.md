# UI-ARCHITECTURE.md

## Arquitetura de UX/UI - FinOps Enterprise Platform

### Design System

#### Paleta de Cores (Dark Mode Enterprise)
- **Background Primary**: `#0f172a` (slate-900)
- **Background Secondary**: `#1e293b` (slate-800)
- **Background Card**: `#334155` (slate-700)
- **Accent Primary**: `#3b82f6` (blue-500)
- **Accent Success**: `#10b981` (emerald-500)
- **Accent Warning**: `#f59e0b` (amber-500)
- **Accent Danger**: `#ef4444` (red-500)
- **Text Primary**: `#f8fafc` (slate-50)
- **Text Secondary**: `#94a3b8` (slate-400)
- **Border**: `#475569` (slate-600)

#### Tipografia
- **Fonte Principal**: Inter, system-ui, sans-serif
- **Títulos**: 24px/32px/48px (semibold)
- **Corpo**: 14px/16px (regular)
- **Dados Numéricos**: 28px/36px (tabular-nums)

#### Componentes Base
- **Cards**: rounded-xl, border border-slate-600, bg-slate-800, shadow-lg
- **Botões Primários**: bg-blue-600, hover:bg-blue-500, rounded-lg
- **Botões Secundários**: bg-slate-700, hover:bg-slate-600, rounded-lg
- **Inputs**: bg-slate-900, border-slate-600, rounded-lg, focus:ring-blue-500
- **Tabelas**: bg-slate-800, border-slate-600, hover:bg-slate-700
- **Badges**: rounded-full, px-3 py-1, text-xs font-semibold

### Navegação

#### Sidebar (Fixa, 240px)
```
┌─────────────────────────────────────┐
│  [Logo] FinOps Enterprise           │
├─────────────────────────────────────┤
│  🏠 Dashboard Executivo             │
│  🔮 Forecast                        │
│  ⚠️ Anomalias                       │
│  💡 Recomendações                   │
│  💰 Budgets                         │
│  👥 Usuários                        │
├─────────────────────────────────────┤
│  [Avatar] Usuário ▼                 │
└─────────────────────────────────────┘
```

> Foram removidos do menu: Dashboard Operacional, Alertas, Multi-Cloud (Azure/AWS) e Configurações. O ambiente atual é **Huawei-only**.

#### Topbar (Fixa, 64px)
- Título da página atual
- Seletor de período (24h, 48h, 7d, 30d, 90d, custom)
- Botão de moeda (USD/BRL)
- Botão de notificações (badge com contador)
- Botão de busca

> O seletor de provider foi removido; o ambiente é Huawei Cloud. Filtros adicionais por account podem ser adicionados futuramente.

### Fluxos de Navegação

#### Fluxo Principal: Dashboard Executivo → Detalhamento
1. Usuário acessa Dashboard Executivo
2. Visualiza KPIs reais (custo total, forecast, serviços, recursos)
3. Clica em "Custos por Serviço" → visualiza ranking no próprio dashboard
4. Clica em "Custos por Região" → visualiza distribuição regional
5. Seleciona outro período no header → todos os widgets recarregam

#### Fluxo: Anomalias → Recomendações
1. Usuário acessa Anomalias
2. Visualiza timeline de anomalias baseadas em Z-Score
3. Clica em anomalia específica → visualiza detalhes na tabela
4. Navega para Recomendações para ver oportunidades de economia

#### Fluxo: Budgets
1. Usuário acessa Budgets
2. Visualiza orçado vs realizado por account
3. Cria/editou um budget vinculado a uma account
4. Quando `spent > threshold`, o status muda para Warning/Over Budget

> Alertas serão reintroduzidos de forma integrada aos budgets (ver `ROADMAP.md`). Não há página de Alertas no momento.

### Estrutura das Telas

A aplicação possui **6 telas principais** após a simplificação para ambiente Huawei-only.

#### 1. Dashboard Executivo
```
┌─────────────────────────────────────────────────────────────┐
│ [Sidebar] │ [Topbar: Dashboard Executivo | Período | USD/BRL]│
│           ├───────────────────────────────────────────────┤
│           │ [KPI Cards: 4 colunas]                        │
│           │ ┌────────┐┌────────┐┌────────┐┌────────┐      │
│           │ │Custo   ││Forecast││Serviços││Recursos│      │
│           │ │Total   ││30d     ││Ativos  ││Rastread│      │
│           │ │$1.2M   ││$1.3M   ││142     ││12K     │      │
│           │ └────────┘└────────┘└────────┘└────────┘      │
│           │ [Tendência de Custos]  [Custos por Serviço]   │
│           │ ┌─────────────┐  ┌──────────────────────┐     │
│           │ │ Line Chart  │  │ Horizontal Bar Chart │     │
│           │ └─────────────┘  └──────────────────────┘     │
│           │ [Custos por Região]  [KPIs Estratégicos]      │
│           │ ┌─────────────┐  ┌──────────────────────┐     │
│           │ │ Bar Chart   │  │ Progress Bars        │     │
│           │ └─────────────┘  └──────────────────────┘     │
└───────────┴─────────────────────────────────────────────────┘
```

#### 2. Forecast
- Seletor de horizonte: 30 / 60 / 90 dias
- Gráfico de linha com valores históricos + previsão
- Resumo: modelo (Linear Trend), confiança, previsão total
- Dados reais do ClickHouse via `cost-analytics`

#### 3. Anomalias
- Timeline de anomalias baseadas em Z-Score
- Resumo: total, impacto, baseline, desvio padrão
- Tabela com data, valor, esperado, desvio, severidade

#### 4. Recomendações
- Cards de oportunidade geradas a partir dos serviços de maior custo
- Resumo: economia potencial, oportunidades, prioridade alta, confiança média
- Botões "Aplicar" / "Ignorar"

#### 5. Budgets
- Cards de total orçado, realizado e variação
- Tabela com nome, provider, account, período, valores e status
- Modal para criar/editar budget vinculado a uma account
- Status: On Track / Warning / Over Budget

#### 6. Usuários
- Tabela de usuários com nome, email, perfis, status, último acesso
- Modal para criar/editar usuário
- Perfis: admin, analyst, viewer

> Telas removidas: Dashboard Operacional, Alertas, Multi-Cloud. Podem ser reintroduzidas futuramente conforme `ROADMAP.md`.

### Componentes Reutilizáveis

#### 1. KPI Card
```html
<div class="bg-slate-800 rounded-xl p-6 border border-slate-600">
  <div class="text-slate-400 text-sm mb-1">{label}</div>
  <div class="text-3xl font-semibold text-white">{value}</div>
  <div class="text-sm mt-2 {trendColor}">{trend} {percentage}</div>
</div>
```

#### 2. Chart Container
```html
<div class="bg-slate-800 rounded-xl p-6 border border-slate-600">
  <div class="flex justify-between items-center mb-4">
    <h3 class="text-lg font-semibold text-white">{title}</h3>
    <div class="flex gap-2">[Filtros do gráfico]</div>
  </div>
  <div id="{chartId}" class="h-80"></div>
</div>
```

#### 3. Data Table
```html
<div class="bg-slate-800 rounded-xl border border-slate-600 overflow-hidden">
  <table class="w-full text-left">
    <thead class="bg-slate-700 text-slate-300">
      <tr>[Headers]</tr>
    </thead>
    <tbody class="divide-y divide-slate-600">
      <tr class="hover:bg-slate-700">[Data]</tr>
    </tbody>
  </table>
</div>
```

#### 4. Filter Bar
```html
<div class="flex gap-4 p-4 bg-slate-800 rounded-xl border border-slate-600">
  <select>[Período]</select>
  <select>[Cloud]</select>
  <select>[Ambiente]</select>
  <select>[Business Unit]</select>
  <button class="bg-blue-600 text-white px-4 py-2 rounded-lg">Aplicar</button>
</div>
```

### Responsividade
- **Desktop (>1280px)**: Sidebar fixa + conteúdo fluido
- **Tablet (768-1280px)**: Sidebar colapsável + conteúdo adaptativo
- **Mobile (<768px)**: Bottom navigation + cards empilhados

### Estados e Feedback
- **Loading**: Skeleton screens com pulse animation
- **Empty**: Ilustração + mensagem + CTA
- **Error**: Toast notification vermelho com retry
- **Success**: Toast notification verde
- **Confirm**: Modal de confirmação

### Acessibilidade
- ARIA labels em todos os elementos interativos
- Contraste mínimo 4.5:1
- Navegação por teclado (Tab, Enter, Escape)
- Screen reader friendly tables
- Focus indicators visíveis
