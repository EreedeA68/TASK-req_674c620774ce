import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import api from '../../services/api';
import DataTable from '../../components/DataTable';
import StatusBadge from '../../components/StatusBadge';
import { useAuth } from '../../context/AuthContext';

export default function PartListPage() {
  const { user } = useAuth();
  const [data, setData] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(false);

  const load = (p = page, q = search) => {
    setLoading(true);
    api.get('/parts', { params: { page: p, limit: 20, search: q || undefined } })
      .then(r => { setData(r.data.data?.items || []); setTotal(r.data.data?.total || 0); })
      .finally(() => setLoading(false));
  };

  useEffect(() => { load(); }, [page]);

  const canCreate = ['ADMIN','INVENTORY_CLERK'].includes(user?.role);

  const columns = [
    { key: 'partNumber', label: 'Part #' },
    { key: 'name', label: 'Name', render: (v, row) => <Link to={`/parts/${row.id}`} className="text-blue-600 hover:underline">{v}</Link> },
    { key: 'status', label: 'Status', render: v => <StatusBadge status={v} /> },
    { key: 'versionNumber', label: 'Version' },
  ];

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Automotive Parts</h1>
        <div className="flex gap-2">
          {canCreate && <Link to="/parts/import" className="px-4 py-2 border text-sm rounded-md hover:bg-gray-50">Bulk Import</Link>}
          {canCreate && <Link to="/parts/new" className="px-4 py-2 bg-blue-600 text-white text-sm rounded-md hover:bg-blue-700">New Part</Link>}
        </div>
      </div>
      <div className="mb-4 flex gap-2">
        <input type="text" value={search} onChange={e => setSearch(e.target.value)} placeholder="Search parts..."
          className="border border-gray-300 rounded-md px-3 py-1.5 text-sm" data-testid="search-input" />
        <button onClick={() => { setPage(1); load(1, search); }} className="px-4 py-1.5 bg-gray-100 text-sm rounded-md border">Search</button>
      </div>
      <DataTable columns={columns} data={data} loading={loading}
        pagination={{ page, total, limit: 20 }} onPageChange={p => setPage(p)} />
    </div>
  );
}
