import React, { useState, useEffect } from 'react';
import api from '../../services/api';
import DataTable from '../../components/DataTable';
import TimestampDisplay from '../../components/TimestampDisplay';

const ROLES = ['ADMIN','INTAKE_SPECIALIST','REVIEWER','INVENTORY_CLERK','AUDITOR'];

export default function AdminPage() {
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(false);
  const [form, setForm] = useState({ username: '', email: '', password: '', role: 'INTAKE_SPECIALIST' });
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  const loadUsers = () => {
    setLoading(true);
    api.get('/admin/users').then(r => setUsers(r.data.data?.users || [])).finally(() => setLoading(false));
  };
  useEffect(() => { loadUsers(); }, []);

  const set = k => e => setForm(f => ({ ...f, [k]: e.target.value }));

  const handleCreate = async (e) => {
    e.preventDefault();
    setError(''); setSuccess('');
    try {
      await api.post('/admin/users', form);
      setSuccess('User created successfully');
      setForm({ username: '', email: '', password: '', role: 'INTAKE_SPECIALIST' });
      loadUsers();
    } catch (err) { setError(err.response?.data?.errorMessage || 'Failed to create user'); }
  };

  const columns = [
    { key: 'username', label: 'Username' },
    { key: 'email', label: 'Email' },
    { key: 'role', label: 'Role' },
    { key: 'isLocked', label: 'Locked', render: v => v ? 'Yes' : 'No' },
    { key: 'createdAt', label: 'Created', render: v => <TimestampDisplay value={v} /> },
  ];

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-900 mb-6">Admin — User Management</h1>
      <div className="bg-white rounded-lg shadow p-6 mb-6">
        <h2 className="text-lg font-semibold mb-4">Create User</h2>
        <form onSubmit={handleCreate} className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Username</label>
            <input value={form.username} onChange={set('username')} required className="w-full border rounded-md px-3 py-2 text-sm" data-testid="new-username" />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Email</label>
            <input type="email" value={form.email} onChange={set('email')} required className="w-full border rounded-md px-3 py-2 text-sm" data-testid="new-email" />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Password</label>
            <input type="password" value={form.password} onChange={set('password')} required minLength={12} className="w-full border rounded-md px-3 py-2 text-sm" data-testid="new-password" />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Role</label>
            <select value={form.role} onChange={set('role')} className="w-full border rounded-md px-3 py-2 text-sm" data-testid="new-role">
              {ROLES.map(r => <option key={r} value={r}>{r}</option>)}
            </select>
          </div>
          {error && <p className="col-span-2 text-red-600 text-sm">{error}</p>}
          {success && <p className="col-span-2 text-green-600 text-sm">{success}</p>}
          <div className="col-span-2">
            <button type="submit" className="px-6 py-2 bg-blue-600 text-white text-sm rounded-md hover:bg-blue-700" data-testid="create-user-btn">Create User</button>
          </div>
        </form>
      </div>
      <div className="bg-white rounded-lg shadow">
        <div className="px-6 py-4 border-b"><h2 className="text-lg font-semibold">Users</h2></div>
        <DataTable columns={columns} data={users} loading={loading} />
      </div>
    </div>
  );
}
