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
│  🏠 Dashboard                       │
│    ├─ Executivo                     │
│    └─ Operacional                   │
│  📊 Analytics                       │
│    ├─ Custos por Serviço            │
│    ├─ Custos por Projeto            │
│    ├─ Custos por Ambiente           │
│    ├─ Custos por Região             │
│    └─ Tags                          │
│  🔮 Forecast                        │
│  ⚠️ Anomalias                       │
│  💡 Recomendações                   │
│  💰 Budgets                         │
│  🔔 Alertas                         │
│  ☁️ Multi-Cloud                     │
│    ├─ Huawei Cloud                  │
│    ├─ Azure                         │
│    └─ AWS                           │
│  👥 Usuários                        │
│  ⚙️ Configurações                   │
├─────────────────────────────────────┤
│  [Avatar] Usuário ▼                 │
└─────────────────────────────────────┘
```

#### Topbar (Fixa, 64px)
- Título da página atual
- Filtros globais (Período, Cloud, Ambiente, BU)
- Botão de notificações (badge com contador)
- Botão de tema (dark/light)
- Avatar do usuário com dropdown

### Fluxos de Navegação

#### Fluxo Principal: Dashboard Executivo → Detalhamento
1. Usuário acessa Dashboard Executivo
2. Visualiza KPIs estratégicos (cards superiores)
3. Clica em "Top Serviços" → navega para Analytics por Serviço
4. Clica em "Top Aplicações" → navega para Analytics por Projeto
5. Clica em "Forecast" → navega para Forecast com o período pré-selecionado

#### Fluxo: Anomalias → Recomendações
1. Usuário acessa Anomalias
2. Visualiza timeline de anomalias
3. Clica em anomalia específica → modal com detalhes
4. Botão "Ver Recomendações" → navega para Recomendações filtradas

#### Fluxo: Budgets → Alertas
1. Usuário acessa Budgets
2. Visualiza orçado vs realizado
3. Clica em budget com alerta → navega para Alertas filtrados

### Estrutura das Telas

#### 1. Dashboard Executivo
```
┌─────────────────────────────────────────────────────────────┐
│ [Sidebar] │ [Topbar: Dashboard Executivo | Filtros Globais]│
│           ├───────────────────────────────────────────────┤
│           │ [KPI Cards: 4 colunas]                        │
│           │ ┌────────┐┌────────┐┌────────┐┌────────┐      │
│           │ │Custo   ││Forecast││Economia││Efficien│      │
│           │ │Total   ││30d     ││Potencial││cy      │      │
│           │ │$1.2M   ││$1.3M   ││$180K   ││85%     │      │
│           │ └────────┘└────────┘└────────┘└────────┘      │
│           │ [Gráfico Principal: Trend de Custos]          │
│           │ ┌─────────────────────────────────────┐       │
│           │ │  Line Chart (12 meses)              │       │
│           │ └─────────────────────────────────────┘       │
│           │ [Top Serviços]    [Top Aplicações]            │
│           │ ┌─────────────┐  ┌─────────────┐              │
│           │ │ Bar Chart   │  │ Bar Chart   │              │
│           │ └─────────────┘  └─────────────┘              │
│           │ [Distribuição por Cloud]                      │
│           │ ┌─────────────────────────────────────┐       │
│           │ │ Pie/Donut Chart                     │       │
│           │ └─────────────────────────────────────┘       │
└───────────┴─────────────────────────────────────────────────┘
```

#### 2. Dashboard Operacional
- Tabela detalhada de custos
- Filtros avançados (Serviço, Projeto, Ambiente, Região, Tags)
- Drill-down por hierarquia
- Exportação CSV/Excel

#### 3. Forecast
- Tabs: 30 dias / 90 dias / 12 meses
- Gráfico de linha com banda de confiança
- Tabela com valores previstos vs histórico
- Indicadores de tendência (↑/↓)

#### 4. Anomalias
- Timeline vertical de anomalias detectadas
- Cards com severidade (Critical/Warning/Info)
- Impacto financeiro estimado
- Filtros por período, serviço, cloud

#### 5. Recomendações
- Cards de oportunidade com:
  - Tipo (Rightsizing, Idle, Storage, Savings)
  - Justificativa técnica
  - Estimativa financeira
  - Botão "Aplicar" / "Ignorar"

#### 6. Budgets
- Cards de budgets ativos
- Gráfico de progresso (orçado vs realizado)
- Indicador de tendência (on-track / at-risk / over-budget)
- Tabela de histórico

#### 7. Alertas
- Tabela de alertas com:
  - Status (Active/Resolved/Acknowledged)
  - Severidade
  - Regra
  - Destinatários
  - Timestamp
- Botão para criar nova regra

#### 8. Multi-Cloud
- Tabs: Huawei Cloud / Azure / AWS
- Cards de contas conectadas
- Status de sincronização
- Última ingestão
- Ações: Sincronizar, Configurar, Desconectar

#### 9. Usuários
- Tabela de usuários
- Colunas: Nome, Email, Perfil, Status, Último Acesso
- Botão: Adicionar, Editar, Desativar
- Modal de permissões granulares

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
