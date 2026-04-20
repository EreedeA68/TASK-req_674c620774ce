import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import api from '../../services/api';
import DataTable from '../../components/DataTable';
import StatusBadge from '../../components/StatusBadge';
import TimestampDisplay from '../../components/TimestampDisplay';

const STATUSES = ['', 'DRAFT', 'SUBMITTED', 'APPROVED', 'REJECTED'];

export default function CandidateListPage() {
  const [data, setData] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState('');
  const [loading, setLoading] = useState(false);

  const load = async (p = page, s = status) => {
    setLoading(true);
    try {
      const r = await api.get('/candidates', { params: { page: p, limit: 20, status: s || undefined } });
      setData(r.data.data?.items || []);
      setTotal(r.data.data?.total || 0);
    } catch {}
    setLoading(false);
  };

  useEffect(() => { load(); }, [page, status]);

  const columns = [
    { key: 'id', label: 'ID', render: (v) => <Link to={`/candidates/${v}`} className="text-blue-600 hover:underline text-xs">{v?.slice(0,8)}...</Link> },
    { key: 'status', label: 'Status', render: (v) => <StatusBadge status={v} /> },
    { key: 'completenessStatus', label: 'Completeness' },
    { key: 'createdAt', label: 'Created', render: (v) => <TimestampDisplay value={v} /> },
    { key: 'submittedAt', label: 'Submitted', render: (v) => <TimestampDisplay value={v} /> },
  ];

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Candidates</h1>
        <Link to="/candidates/new" className="px-4 py-2 bg-blue-600 text-white text-sm rounded-md hover:bg-blue-700">New Candidate</Link>
      </div>
      <div className="mb-4 flex gap-3">
        <select value={status} onChange={e => { setStatus(e.target.value); setPage(1); load(1, e.target.value); }}
          className="border border-gray-300 rounded-md px-3 py-1.5 text-sm" data-testid="status-filter">
          {STATUSES.map(s => <option key={s} value={s}>{s || 'All Statuses'}</option>)}
        </select>
      </div>
      <DataTable columns={columns} data={data} loading={loading}
        pagination={{ page, total, limit: 20 }} onPageChange={p => setPage(p)} />
    </div>
  );
}
