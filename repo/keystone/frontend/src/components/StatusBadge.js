import React from 'react';

const STATUS_COLORS = {
  DRAFT: 'bg-gray-100 text-gray-700',
  SUBMITTED: 'bg-blue-100 text-blue-700',
  APPROVED: 'bg-green-100 text-green-700',
  REJECTED: 'bg-red-100 text-red-700',
  PUBLISHED: 'bg-green-100 text-green-700',
  UNLISTED: 'bg-yellow-100 text-yellow-700',
  DELETED: 'bg-red-100 text-red-700',
  ACTIVE: 'bg-green-100 text-green-700',
  DEPRECATED: 'bg-orange-100 text-orange-700',
};

export default function StatusBadge({ status }) {
  const cls = STATUS_COLORS[status] || 'bg-gray-100 text-gray-600';
  return (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${cls}`} data-testid="status-badge">
      {status}
    </span>
  );
}
