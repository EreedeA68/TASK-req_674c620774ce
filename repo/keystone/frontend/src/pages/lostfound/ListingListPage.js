import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import api from '../../services/api';
import { useAuth } from '../../context/AuthContext';
import StatusBadge from '../../components/StatusBadge';
import TimestampDisplay from '../../components/TimestampDisplay';

export default function ListingListPage() {
  const { user } = useAuth();
  const [listings, setListings] = useState([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);

  useEffect(() => {
    setLoading(true);
    api.get('/listings', { params: { page, limit: 20 } })
      .then(r => { setListings(r.data.data?.items || []); setTotal(r.data.data?.total || 0); })
      .finally(() => setLoading(false));
  }, [page]);

  const canCreate = ['ADMIN','INVENTORY_CLERK'].includes(user?.role);

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Lost &amp; Found</h1>
        {canCreate && <Link to="/listings/new" className="px-4 py-2 bg-blue-600 text-white text-sm rounded-md hover:bg-blue-700">New Listing</Link>}
      </div>
      {loading ? <div className="text-center py-10 text-gray-500">Loading...</div> : (
        <div className="grid gap-4" data-testid="listings-grid">
          {listings.map(l => (
            <Link key={l.id} to={`/listings/${l.id}`} className="bg-white rounded-lg shadow p-4 hover:shadow-md transition flex items-start justify-between">
              <div>
                <div className="flex items-center gap-2 mb-1">
                  <span className="text-sm font-medium text-gray-900">{l.title}</span>
                  {l.isDuplicateFlagged && (
                    <span className="px-2 py-0.5 bg-yellow-100 text-yellow-700 text-xs rounded-full" data-testid="duplicate-flag">Possible Duplicate</span>
                  )}
                </div>
                <div className="flex items-center gap-2 text-xs text-gray-500">
                  <span className="bg-gray-100 px-2 py-0.5 rounded">{l.category}</span>
                  <span>{l.locationDescription}</span>
                  <TimestampDisplay value={l.createdAt} />
                </div>
              </div>
              <StatusBadge status={l.status} />
            </Link>
          ))}
        </div>
      )}
      <div className="mt-4 flex gap-2">
        <button onClick={() => setPage(p => Math.max(1, p-1))} disabled={page <= 1} className="px-3 py-1 text-sm border rounded disabled:opacity-40" data-testid="prev-page">Prev</button>
        <button onClick={() => setPage(p => p+1)} disabled={page * 20 >= total} className="px-3 py-1 text-sm border rounded disabled:opacity-40" data-testid="next-page">Next</button>
      </div>
    </div>
  );
}
