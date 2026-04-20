import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import api from '../../services/api';
import { useAuth } from '../../context/AuthContext';

export default function PartVersionComparePage() {
  const { id, v1, v2 } = useParams();
  const { user } = useAuth();
  const navigate = useNavigate();
  const [diff, setDiff] = useState(null);
  const [loading, setLoading] = useState(true);
  const [promoting, setPromoting] = useState(false);

  useEffect(() => {
    api.get(`/parts/${id}/versions/${v1}/compare/${v2}`)
      .then(r => setDiff(r.data.data))
      .finally(() => setLoading(false));
  }, [id, v1, v2]);

  const handlePromote = async () => {
    setPromoting(true);
    await api.post(`/parts/${id}/promote`, { versionId: diff?.versionB?.id });
    navigate(`/parts/${id}`);
  };

  const canPromote = ['ADMIN','INVENTORY_CLERK'].includes(user?.role);

  if (loading) return <div className="text-center py-10 text-gray-500">Loading...</div>;

  const diffEntries = diff?.diff ? Object.entries(diff.diff) : [];

  return (
    <div className="max-w-3xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">Version Compare: v{v1} &rarr; v{v2}</h1>
      <div className="bg-white rounded-lg shadow overflow-hidden mb-6">
        <table className="min-w-full divide-y divide-gray-200" data-testid="compare-table">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">Field</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">v{v1}</th>
              <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">v{v2}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200">
            {diffEntries.map(([field, d]) => (
              <tr key={field} className={d.changed ? 'bg-yellow-50' : ''} data-testid={d.changed ? 'changed-field' : 'unchanged-field'}>
                <td className="px-4 py-3 text-sm font-medium text-gray-700">{field}</td>
                <td className="px-4 py-3 text-sm text-gray-600 font-mono text-xs">{JSON.stringify(d.oldValue)}</td>
                <td className={`px-4 py-3 text-sm font-mono text-xs ${d.changed ? 'text-blue-700 font-semibold' : 'text-gray-600'}`}>{JSON.stringify(d.newValue)}</td>
              </tr>
            ))}
            {diffEntries.length === 0 && (
              <tr><td colSpan={3} className="px-4 py-6 text-center text-gray-500">No differences found</td></tr>
            )}
          </tbody>
        </table>
      </div>
      {canPromote && (
        <button onClick={handlePromote} disabled={promoting}
          className="px-6 py-2 bg-green-600 text-white text-sm rounded-md hover:bg-green-700 disabled:opacity-50" data-testid="promote-btn">
          {promoting ? 'Promoting...' : `Promote v${v2} to Active`}
        </button>
      )}
    </div>
  );
}
