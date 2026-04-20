import React, { useState } from 'react';

export default function DataTable({ columns, data, pagination, onPageChange, loading }) {
  const [sortKey, setSortKey] = useState(null);
  const [sortDir, setSortDir] = useState('asc');

  const handleSort = (key) => {
    if (sortKey === key) setSortDir(d => d === 'asc' ? 'desc' : 'asc');
    else { setSortKey(key); setSortDir('asc'); }
  };

  const sorted = sortKey ? [...(data || [])].sort((a, b) => {
    const av = a[sortKey], bv = b[sortKey];
    if (av < bv) return sortDir === 'asc' ? -1 : 1;
    if (av > bv) return sortDir === 'asc' ? 1 : -1;
    return 0;
  }) : (data || []);

  return (
    <div className="overflow-hidden shadow ring-1 ring-black ring-opacity-5 rounded-lg">
      <table className="min-w-full divide-y divide-gray-300" data-testid="data-table">
        <thead className="bg-gray-50">
          <tr>
            {columns.map(col => (
              <th key={col.key} onClick={() => col.sortable !== false && handleSort(col.key)}
                className={`px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider ${col.sortable !== false ? 'cursor-pointer hover:bg-gray-100' : ''}`}>
                {col.label} {sortKey === col.key ? (sortDir === 'asc' ? '↑' : '↓') : ''}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="bg-white divide-y divide-gray-200">
          {loading ? (
            <tr><td colSpan={columns.length} className="px-4 py-8 text-center text-gray-500">Loading...</td></tr>
          ) : sorted.length === 0 ? (
            <tr><td colSpan={columns.length} className="px-4 py-8 text-center text-gray-500">No records found</td></tr>
          ) : sorted.map((row, i) => (
            <tr key={row.id || i} className="hover:bg-gray-50">
              {columns.map(col => (
                <td key={col.key} className="px-4 py-3 text-sm text-gray-900 whitespace-nowrap">
                  {col.render ? col.render(row[col.key], row) : row[col.key]}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
      {pagination && (
        <div className="bg-white px-4 py-3 flex items-center justify-between border-t border-gray-200">
          <span className="text-sm text-gray-700">Page {pagination.page} of {Math.ceil(pagination.total / pagination.limit) || 1}</span>
          <div className="flex gap-2">
            <button onClick={() => onPageChange(pagination.page - 1)} disabled={pagination.page <= 1}
              className="px-3 py-1 text-sm border rounded disabled:opacity-40" data-testid="prev-page">Previous</button>
            <button onClick={() => onPageChange(pagination.page + 1)} disabled={pagination.page * pagination.limit >= pagination.total}
              className="px-3 py-1 text-sm border rounded disabled:opacity-40" data-testid="next-page">Next</button>
          </div>
        </div>
      )}
    </div>
  );
}
