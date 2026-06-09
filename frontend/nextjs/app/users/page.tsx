'use client';

import { MainLayout } from '@/components/layout/MainLayout';
import { DataTable } from '@/components/ui/DataTable';
import { Badge } from '@/components/ui/Badge';
import type { User } from '@/types';
import { Plus, Edit } from 'lucide-react';

const mockUsers: User[] = [
  { id: '1', email: 'admin@finops.local', name: 'Admin User', roles: ['admin'], active: true, last_login: '2024-01-15T14:30:00Z', created_at: '2024-01-01' },
  { id: '2', email: 'ana.silva@company.com', name: 'Ana Silva', roles: ['analyst'], active: true, last_login: '2024-01-15T10:15:00Z', created_at: '2024-01-02' },
  { id: '3', email: 'carlos.mendes@company.com', name: 'Carlos Mendes', roles: ['viewer'], active: true, last_login: '2024-01-14T16:45:00Z', created_at: '2024-01-03' },
];

export default function UsersPage() {
  const columns = [
    { key: 'name', header: 'Nome' },
    { key: 'email', header: 'Email' },
    { key: 'roles', header: 'Perfil', render: (u: User) => (
      <Badge variant={u.roles.includes('admin') ? 'danger' : u.roles.includes('analyst') ? 'warning' : 'info'}>
        {u.roles[0]}
      </Badge>
    )},
    { key: 'active', header: 'Status', render: (u: User) => (
      <span className={u.active ? 'text-emerald-400' : 'text-red-400'}>
        {u.active ? 'Ativo' : 'Inativo'}
      </span>
    )},
    { key: 'last_login', header: 'Último Acesso', render: (u: User) => (
      <span className="text-slate-400">{u.last_login ? new Date(u.last_login).toLocaleString('pt-BR') : 'Nunca'}</span>
    )},
    { key: 'action', header: 'Ações', render: () => (
      <button className="text-sky-400 hover:text-sky-300 flex items-center gap-1">
        <Edit className="w-4 h-4" />
        Editar
      </button>
    )},
  ];

  return (
    <MainLayout title="Usuários">
      <div className="space-y-6">
        <div className="flex justify-end">
          <button className="px-4 py-2 bg-sky-600 hover:bg-sky-500 text-white rounded-lg text-sm font-medium transition-colors flex items-center gap-2">
            <Plus className="w-4 h-4" />
            Novo Usuário
          </button>
        </div>

        <DataTable columns={columns} data={mockUsers} />
      </div>
    </MainLayout>
  );
}
