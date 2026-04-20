import React, { useState, useEffect } from 'react';
import api from '../../services/api';
import { useAuth } from '../../context/AuthContext';
import { Navigate } from 'react-router-dom';
import DataTable from '../../components/DataTable';
import TimestampDisplay from '../../components/TimestampDisplay';

export default function AuditLogPage() {
  const { user } = useAuth();
  const [data, setData] = useState([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);

  if (!['ADMIN','AUDITOR'].includes(user?.role)) return <Navigate to="/unauthorized" replace />;

  useEffect(() => {
    setLoading(true);
    api.get('/audit-logs', { params: { page, limit: 50 } })
      .then(r => { setData(r.data.data?.items || []); setTotal(r.data.data?.total || 0); })
      .finally(() => setLoading(false));
  }, [page]);

  const columns = [
    { key: 'actorId', label: 'Actor', render: v => <span className="font-mono text-xs">{v?.slice(0,8)}...</span> },
    { key: 'action', label: 'Action' },
    { key: 'resourceType', label: 'Resource Type' },
    { key: 'resourceId', label: 'Resource', render: v => v ? <span className="font-mono text-xs">{v?.slice(0,8)}...</span> : '—' },
    { key: 'ipAddress', label: 'IP' },
    { key: 'createdAt', label: 'Timestamp', render: v => <TimestampDisplay value={v} /> },
  ];

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-900 mb-6">Audit Logs</h1>
      <DataTable columns={columns} data={data} loading={loading}
        pagination={{ page, total, limit: 50 }} onPageChange={p => setPage(p)} />
    </div>
  );
}
