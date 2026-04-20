import React from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';

const ROLE_LINKS = {
  ADMIN: [
    { to: '/candidates', label: 'Candidates' },
    { to: '/listings', label: 'Lost & Found' },
    { to: '/parts', label: 'Parts' },
    { to: '/reports/kpi', label: 'Reports' },
    { to: '/audit-logs', label: 'Audit Logs' },
    { to: '/admin', label: 'Admin' },
  ],
  INTAKE_SPECIALIST: [
    { to: '/candidates', label: 'Candidates' },
    { to: '/listings', label: 'Lost & Found' },
  ],
  REVIEWER: [
    { to: '/candidates', label: 'Candidates' },
    { to: '/listings', label: 'Lost & Found' },
  ],
  INVENTORY_CLERK: [
    { to: '/listings', label: 'Lost & Found' },
    { to: '/parts', label: 'Parts' },
    { to: '/parts/import', label: 'Bulk Import' },
  ],
  AUDITOR: [
    { to: '/candidates', label: 'Candidates' },
    { to: '/listings', label: 'Lost & Found' },
    { to: '/parts', label: 'Parts' },
    { to: '/audit-logs', label: 'Audit Logs' },
    { to: '/reports/kpi', label: 'Reports' },
  ],
};

export default function Navbar() {
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const links = ROLE_LINKS[user?.role] || [];

  const handleLogout = async () => { await logout(); navigate('/login'); };

  return (
    <nav className="bg-gray-900 text-white shadow" data-testid="navbar">
      <div className="max-w-7xl mx-auto px-4 flex items-center justify-between h-14">
        <div className="flex items-center gap-6">
          <span className="font-bold text-lg tracking-tight">Keystone</span>
          {links.map(l => (
            <Link key={l.to} to={l.to} className="text-sm text-gray-300 hover:text-white transition">{l.label}</Link>
          ))}
        </div>
        <div className="flex items-center gap-4">
          <span className="text-xs text-gray-400">{user?.username} ({user?.role})</span>
          <button onClick={handleLogout} className="text-sm text-gray-300 hover:text-white">Logout</button>
        </div>
      </div>
    </nav>
  );
}
