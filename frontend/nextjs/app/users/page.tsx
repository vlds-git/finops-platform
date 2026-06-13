'use client';

import { useState } from 'react';
import { MainLayout } from '@/components/layout/MainLayout';
import { DataTable } from '@/components/ui/DataTable';
import { Badge } from '@/components/ui/Badge';
import { useUsers, useCreateUser, useUpdateUser, useDeleteUser, type CreateUserInput } from '@/hooks/useUsers';
import type { User } from '@/types';
import { Plus, Edit, Trash2, X } from 'lucide-react';

interface UserFormData {
  id?: string;
  email: string;
  name: string;
  password: string;
  roles: string[];
  active: boolean;
}

const emptyForm: UserFormData = {
  email: '',
  name: '',
  password: '',
  roles: ['viewer'],
  active: true,
};

const roleOptions = [
  { value: 'admin', label: 'Administrador' },
  { value: 'analyst', label: 'Analista' },
  { value: 'viewer', label: 'Visualizador' },
];

export default function UsersPage() {
  const { data: users = [], isLoading } = useUsers();
  const createUser = useCreateUser();
  const updateUser = useUpdateUser();
  const deleteUser = useDeleteUser();
  const [isOpen, setIsOpen] = useState(false);
  const [form, setForm] = useState<UserFormData>(emptyForm);

  const openCreate = () => {
    setForm(emptyForm);
    setIsOpen(true);
  };

  const openEdit = (u: User) => {
    setForm({
      id: u.id,
      email: u.email,
      name: u.name,
      password: '',
      roles: u.roles,
      active: u.active,
    });
    setIsOpen(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (form.id) {
      const payload: Partial<User> = {
        email: form.email,
        name: form.name,
        roles: form.roles,
        active: form.active,
      };
      if (form.password) (payload as any).password = form.password;
      await updateUser.mutateAsync({ id: form.id, ...payload });
    } else {
      const payload: CreateUserInput = {
        email: form.email,
        name: form.name,
        password: form.password,
        roles: form.roles,
      };
      await createUser.mutateAsync(payload);
    }
    setIsOpen(false);
    setForm(emptyForm);
  };

  const handleDelete = async (id: string) => {
    if (confirm('Deseja realmente excluir este usuário?')) {
      await deleteUser.mutateAsync(id);
    }
  };

  const toggleRole = (role: string) => {
    setForm((prev) => ({
      ...prev,
      roles: prev.roles.includes(role)
        ? prev.roles.filter((r) => r !== role)
        : [...prev.roles, role],
    }));
  };

  const columns = [
    { key: 'name', header: 'Nome' },
    { key: 'email', header: 'Email' },
    { key: 'roles', header: 'Perfil', render: (u: User) => (
      <div className="flex gap-1">
        {u.roles.map((r) => (
          <Badge key={r} variant={r === 'admin' ? 'danger' : r === 'analyst' ? 'warning' : 'info'}>
            {r}
          </Badge>
        ))}
      </div>
    )},
    { key: 'active', header: 'Status', render: (u: User) => (
      <span className={u.active ? 'text-emerald-400' : 'text-red-400'}>
        {u.active ? 'Ativo' : 'Inativo'}
      </span>
    )},
    { key: 'last_login', header: 'Último Acesso', render: (u: User) => (
      <span className="text-slate-400">{u.last_login ? new Date(u.last_login).toLocaleString('pt-BR') : 'Nunca'}</span>
    )},
    { key: 'actions', header: 'Ações', render: (u: User) => (
      <div className="flex items-center gap-2">
        <button onClick={() => openEdit(u)} className="text-sky-400 hover:text-sky-300 flex items-center gap-1">
          <Edit className="w-4 h-4" />
        </button>
        <button onClick={() => handleDelete(u.id)} className="text-red-400 hover:text-red-300 flex items-center gap-1">
          <Trash2 className="w-4 h-4" />
        </button>
      </div>
    )},
  ];

  return (
    <MainLayout title="Usuários">
      <div className="space-y-6">
        <div className="flex justify-end">
          <button
            onClick={openCreate}
            className="px-4 py-2 bg-sky-600 hover:bg-sky-500 text-white rounded-lg text-sm font-medium transition-colors flex items-center gap-2"
          >
            <Plus className="w-4 h-4" />
            Novo Usuário
          </button>
        </div>

        <DataTable columns={columns} data={users} isLoading={isLoading} />

        {isOpen && (
          <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
            <div className="bg-slate-800 border border-slate-700 rounded-lg p-6 w-full max-w-lg">
              <div className="flex justify-between items-center mb-4">
                <h2 className="text-lg font-semibold text-white">{form.id ? 'Editar Usuário' : 'Novo Usuário'}</h2>
                <button onClick={() => setIsOpen(false)} className="text-slate-400 hover:text-white">
                  <X className="w-5 h-5" />
                </button>
              </div>
              <form onSubmit={handleSubmit} className="space-y-4">
                <div>
                  <label className="block text-sm text-slate-400 mb-1">Nome</label>
                  <input
                    required
                    value={form.name}
                    onChange={(e) => setForm({ ...form, name: e.target.value })}
                    className="w-full bg-slate-900 border border-slate-700 rounded-lg px-3 py-2 text-white text-sm"
                  />
                </div>
                <div>
                  <label className="block text-sm text-slate-400 mb-1">Email</label>
                  <input
                    required
                    type="email"
                    value={form.email}
                    onChange={(e) => setForm({ ...form, email: e.target.value })}
                    className="w-full bg-slate-900 border border-slate-700 rounded-lg px-3 py-2 text-white text-sm"
                  />
                </div>
                <div>
                  <label className="block text-sm text-slate-400 mb-1">Senha {form.id && '(deixe em branco para não alterar)'}</label>
                  <input
                    required={!form.id}
                    type="password"
                    value={form.password}
                    onChange={(e) => setForm({ ...form, password: e.target.value })}
                    className="w-full bg-slate-900 border border-slate-700 rounded-lg px-3 py-2 text-white text-sm"
                  />
                </div>
                <div>
                  <label className="block text-sm text-slate-400 mb-1">Perfis</label>
                  <div className="flex gap-3">
                    {roleOptions.map((role) => (
                      <label key={role.value} className="flex items-center gap-2 text-sm text-slate-300 cursor-pointer">
                        <input
                          type="checkbox"
                          checked={form.roles.includes(role.value)}
                          onChange={() => toggleRole(role.value)}
                          className="rounded border-slate-600 bg-slate-900 text-sky-500"
                        />
                        {role.label}
                      </label>
                    ))}
                  </div>
                </div>
                {form.id && (
                  <div className="flex items-center gap-2">
                    <input
                      id="active"
                      type="checkbox"
                      checked={form.active}
                      onChange={(e) => setForm({ ...form, active: e.target.checked })}
                      className="rounded border-slate-600 bg-slate-900 text-sky-500"
                    />
                    <label htmlFor="active" className="text-sm text-slate-300 cursor-pointer">Ativo</label>
                  </div>
                )}
                <div className="flex justify-end gap-3 pt-2">
                  <button
                    type="button"
                    onClick={() => setIsOpen(false)}
                    className="px-4 py-2 rounded-lg border border-slate-600 text-slate-300 text-sm hover:bg-slate-700"
                  >
                    Cancelar
                  </button>
                  <button
                    type="submit"
                    className="px-4 py-2 rounded-lg bg-sky-600 hover:bg-sky-500 text-white text-sm font-medium"
                  >
                    Salvar
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}
      </div>
    </MainLayout>
  );
}
