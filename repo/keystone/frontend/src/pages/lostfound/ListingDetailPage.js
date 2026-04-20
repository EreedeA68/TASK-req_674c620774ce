import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import api from '../../services/api';
import { useAuth } from '../../context/AuthContext';
import StatusBadge from '../../components/StatusBadge';
import TimestampDisplay from '../../components/TimestampDisplay';
import ConfirmDialog from '../../components/ConfirmDialog';

export default function ListingDetailPage() {
  const { id } = useParams();
  const { user } = useAuth();
  const navigate = useNavigate();
  const [listing, setListing] = useState(null);
  const [loading, setLoading] = useState(true);
  const [overrideDialog, setOverrideDialog] = useState(false);

  const load = () => api.get(`/listings/${id}`).then(r => setListing(r.data.data)).finally(() => setLoading(false));
  useEffect(() => { load(); }, [id]);

  const handleUnlist = async () => { await api.post(`/listings/${id}/unlist`); load(); };
  const handleDelete = async () => { await api.delete(`/listings/${id}`); navigate('/listings'); };
  const handleOverride = async () => { await api.post(`/listings/${id}/override-duplicate`); setOverrideDialog(false); load(); };

  const canOverride = ['ADMIN', 'REVIEWER'].includes(user?.role);
  const canUnlist = ['ADMIN', 'INVENTORY_CLERK', 'REVIEWER'].includes(user?.role);
  const canDelete = ['ADMIN'].includes(user?.role);

  if (loading) return <div className="text-center py-10 text-gray-500">Loading...</div>;
  if (!listing) return null;

  return (
    <div className="max-w-2xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900">{listing.title}</h1>
        <StatusBadge status={listing.status} />
      </div>
      <div className="bg-white rounded-lg shadow p-6 space-y-3 text-sm">
        {listing.isDuplicateFlagged && (
          <div className="bg-yellow-50 border border-yellow-200 rounded p-3">
            <p className="text-yellow-700 font-medium">Flagged as possible duplicate</p>
            {canOverride && (
              <button onClick={() => setOverrideDialog(true)} className="mt-2 px-3 py-1 bg-yellow-600 text-white text-xs rounded" data-testid="override-btn">Override Duplicate Flag</button>
            )}
          </div>
        )}
        <div><span className="font-medium text-gray-600">Category:</span> {listing.category}</div>
        <div><span className="font-medium text-gray-600">Location:</span> {listing.locationDescription}</div>
        <div><span className="font-medium text-gray-600">Time Window:</span> <TimestampDisplay value={listing.timeWindowStart} /> — <TimestampDisplay value={listing.timeWindowEnd} /></div>
        <div><span className="font-medium text-gray-600">Created:</span> <TimestampDisplay value={listing.createdAt} /></div>
        <div><span className="font-medium text-gray-600">Updated:</span> <TimestampDisplay value={listing.updatedAt} /></div>
      </div>
      <div className="mt-4 flex gap-3">
        {listing.status === 'PUBLISHED' && canUnlist && <button onClick={handleUnlist} className="px-4 py-2 bg-yellow-600 text-white text-sm rounded-md" data-testid="unlist-btn">Unlist</button>}
        {canDelete && <button onClick={handleDelete} className="px-4 py-2 bg-red-600 text-white text-sm rounded-md" data-testid="delete-btn">Delete</button>}
      </div>
      <ConfirmDialog open={overrideDialog} title="Override Duplicate Flag" message="This will clear the duplicate flag and allow the listing to remain active."
        onConfirm={handleOverride} onCancel={() => setOverrideDialog(false)} />
    </div>
  );
}
