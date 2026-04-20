import React, { useState, useEffect } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import api from '../../services/api';
import StatusBadge from '../../components/StatusBadge';
import TimestampDisplay from '../../components/TimestampDisplay';

export default function PartDetailPage() {
  const { id } = useParams();
  const navigate = useNavigate();
  const [part, setPart] = useState(null);
  const [versions, setVersions] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([
      api.get(`/parts/${id}`),
      api.get(`/parts/${id}/versions`)
    ]).then(([pRes, vRes]) => {
      setPart(pRes.data.data);
      setVersions(vRes.data.data || []);
    }).finally(() => setLoading(false));
  }, [id]);

  if (loading) return <div className="text-center py-10 text-gray-500">Loading...</div>;
  if (!part) return null;

  return (
    <div className="max-w-3xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">{part.name}</h1>
          <p className="text-sm text-gray-500">#{part.partNumber}</p>
        </div>
        <StatusBadge status={part.status} />
      </div>

      <div className="bg-white rounded-lg shadow p-6 mb-6">
        <h2 className="text-lg font-semibold mb-3">Current Version ({part.versionNumber})</h2>
        <p className="text-sm text-gray-600 mb-4">{part.description}</p>
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div><span className="font-medium">Current Version ID:</span> <span className="font-mono text-xs">{part.currentVersionID}</span></div>
          <div><span className="font-medium">Created:</span> <TimestampDisplay value={part.createdAt} /></div>
        </div>
      </div>

      <div className="bg-white rounded-lg shadow p-6">
        <h2 className="text-lg font-semibold mb-3">Version History</h2>
        <div className="space-y-3" data-testid="version-history">
          {versions.map(v => (
            <div key={v.id} className="flex items-center justify-between border rounded-md p-3">
              <div>
                <span className="font-medium text-sm">v{v.versionNumber}</span>
                <span className="text-xs text-gray-500 ml-2">{v.changeSummary || 'No summary'}</span>
                <div className="text-xs text-gray-400"><TimestampDisplay value={v.createdAt} /></div>
              </div>
              <div className="flex gap-2">
                {versions.length > 1 && v.versionNumber > 1 && (
                  <Link to={`/parts/${id}/compare/${v.versionNumber - 1}/${v.versionNumber}`}
                    className="text-xs text-blue-600 hover:underline" data-testid={`compare-btn-${v.versionNumber}`}>Compare</Link>
                )}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
