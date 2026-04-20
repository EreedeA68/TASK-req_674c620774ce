import React, { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import api from '../../services/api';
import { useAuth } from '../../context/AuthContext';
import StatusBadge from '../../components/StatusBadge';
import TimestampDisplay from '../../components/TimestampDisplay';
import ConfirmDialog from '../../components/ConfirmDialog';

export default function CandidateDetailPage() {
  const { id } = useParams();
  const { user } = useAuth();
  const [candidate, setCandidate] = useState(null);
  const [loading, setLoading] = useState(true);
  const [rejectComments, setRejectComments] = useState('');
  const [showReject, setShowReject] = useState(false);
  const [approveComments, setApproveComments] = useState('');
  const [showApprove, setShowApprove] = useState(false);
  const [submitDialog, setSubmitDialog] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    api.get(`/candidates/${id}`)
      .then(r => setCandidate(r.data.data))
      .catch(() => setError('Failed to load candidate'))
      .finally(() => setLoading(false));
  }, [id]);

  const reload = async () => {
    const r = await api.get(`/candidates/${id}`);
    setCandidate(r.data.data);
  };

  const handleApprove = async () => {
    await api.post(`/candidates/${id}/approve`, { comments: approveComments.trim() || 'Approved' });
    setShowApprove(false);
    setApproveComments('');
    await reload();
  };

  const handleReject = async () => {
    if (!rejectComments.trim()) { setError('Comments required for rejection'); return; }
    await api.post(`/candidates/${id}/reject`, { comments: rejectComments });
    setShowReject(false);
    await reload();
  };

  const handleSubmit = async () => {
    await api.post(`/candidates/${id}/submit`);
    setSubmitDialog(false);
    await reload();
  };

  const canReview = ['ADMIN', 'REVIEWER'].includes(user?.role);
  const canSubmit = ['ADMIN', 'INTAKE_SPECIALIST'].includes(user?.role);
  const canSeeFullData = ['ADMIN', 'REVIEWER'].includes(user?.role);

  const renderDemographics = (demo) => {
    if (!demo) return null;
    const d = typeof demo === 'string' ? JSON.parse(demo) : demo;
    if (!canSeeFullData) {
      const masked = {
        ...d,
        dob: d.dob ? '••••-••-••' : undefined,
        gender: d.gender ? '••••' : undefined,
        ethnicity: d.ethnicity ? '••••' : undefined,
      };
      return <pre className="bg-gray-50 p-3 rounded text-xs overflow-auto">{JSON.stringify(masked, null, 2)}</pre>;
    }
    return <pre className="bg-gray-50 p-3 rounded text-xs overflow-auto">{JSON.stringify(d, null, 2)}</pre>;
  };

  const renderSensitiveField = (data, label) => {
    if (!data) return null;
    return (
      <div>
        <h3 className="font-semibold text-gray-800 mb-1">{label}</h3>
        {canSeeFullData
          ? <pre className="bg-gray-50 p-3 rounded text-xs overflow-auto">{JSON.stringify(data, null, 2)}</pre>
          : <p className="bg-gray-50 p-3 rounded text-xs text-gray-400 italic">••• {label} restricted (REVIEWER or ADMIN only) •••</p>
        }
      </div>
    );
  };

  if (loading) return <div className="text-center py-10 text-gray-500">Loading...</div>;
  if (error) return <div className="text-red-600">{error}</div>;
  if (!candidate) return null;

  return (
    <div className="max-w-3xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Candidate Detail</h1>
        <StatusBadge status={candidate.status} />
      </div>

      <div className="bg-white rounded-lg shadow p-6 space-y-4 mb-6">
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div><span className="font-medium text-gray-600">ID:</span> <span>{candidate.id}</span></div>
          <div><span className="font-medium text-gray-600">Completeness:</span> <span>{candidate.completenessStatus}</span></div>
          <div><span className="font-medium text-gray-600">Created:</span> <TimestampDisplay value={candidate.createdAt} /></div>
          <div><span className="font-medium text-gray-600">Submitted:</span> <TimestampDisplay value={candidate.submittedAt} /></div>
          <div><span className="font-medium text-gray-600">Reviewed:</span> <TimestampDisplay value={candidate.reviewedAt} /></div>
          <div><span className="font-medium text-gray-600">Reviewer Comments:</span> <span>{candidate.reviewerComments || '—'}</span></div>
        </div>

        {candidate.demographics && (
          <div><h3 className="font-semibold text-gray-800 mb-1">Demographics</h3>
            {renderDemographics(candidate.demographics)}
          </div>
        )}
        {renderSensitiveField(candidate.examScores, 'Exam Scores')}
        {renderSensitiveField(candidate.applicationDetails, 'Application Details')}
        {renderSensitiveField(candidate.transferPreferences, 'Transfer Preferences')}
      </div>

      {canSubmit && candidate.status === 'DRAFT' && (
        <div className="flex gap-3 mb-6" data-testid="submit-actions">
          <button onClick={() => setSubmitDialog(true)}
            className="px-4 py-2 bg-blue-600 text-white text-sm rounded-md hover:bg-blue-700"
            data-testid="submit-btn">
            Submit for Review
          </button>
        </div>
      )}

      {canReview && candidate.status === 'SUBMITTED' && (
        <div className="flex gap-3 mb-6" data-testid="review-actions">
          <button onClick={() => setShowApprove(!showApprove)} className="px-4 py-2 bg-green-600 text-white text-sm rounded-md hover:bg-green-700" data-testid="approve-btn">Approve</button>
          <button onClick={() => setShowReject(!showReject)} className="px-4 py-2 bg-red-600 text-white text-sm rounded-md hover:bg-red-700" data-testid="reject-btn">Reject</button>
        </div>
      )}

      {showApprove && (
        <div className="bg-white rounded-lg shadow p-4 mb-4">
          <label className="block text-sm font-medium text-gray-700 mb-1">Approval Comments (optional)</label>
          <textarea value={approveComments} onChange={e => setApproveComments(e.target.value)} rows={3}
            className="w-full border rounded-md px-3 py-2 text-sm" data-testid="approve-comments" />
          <button onClick={handleApprove}
            className="mt-2 px-4 py-2 bg-green-600 text-white text-sm rounded-md hover:bg-green-700" data-testid="approve-submit">
            Confirm Approval
          </button>
        </div>
      )}

      {showReject && (
        <div className="bg-white rounded-lg shadow p-4 mb-4">
          <label className="block text-sm font-medium text-gray-700 mb-1">Rejection Comments (required)</label>
          <textarea value={rejectComments} onChange={e => setRejectComments(e.target.value)} rows={3}
            className="w-full border rounded-md px-3 py-2 text-sm" data-testid="reject-comments" />
          {error && <p className="text-red-600 text-xs mt-1">{error}</p>}
          <button onClick={handleReject} disabled={!rejectComments.trim()}
            className="mt-2 px-4 py-2 bg-red-600 text-white text-sm rounded-md disabled:opacity-50" data-testid="reject-submit">
            Submit Rejection
          </button>
        </div>
      )}

      <ConfirmDialog open={submitDialog} title="Submit Candidate" message="Submit this candidate for review?"
        onConfirm={handleSubmit} onCancel={() => setSubmitDialog(false)} />
    </div>
  );
}
