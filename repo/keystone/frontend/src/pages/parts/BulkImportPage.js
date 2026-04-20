import React, { useState } from 'react';
import api from '../../services/api';

function parseCsvPreview(text) {
  const lines = text.trim().split('\n');
  if (lines.length < 2) return { headers: [], rows: [] };
  const headers = lines[0].split(',').map(h => h.trim());
  const rows = lines.slice(1).map((line, i) => {
    const vals = line.split(',').map(v => v.trim());
    const row = { _index: i };
    headers.forEach((h, j) => { row[h] = vals[j] || ''; });
    row._errors = [];
    if (!row['part_number']) row._errors.push('part_number required');
    if (!row['name']) row._errors.push('name required');
    return row;
  });
  return { headers, rows };
}

export default function BulkImportPage() {
  const [csvText, setCsvText] = useState('');
  const [preview, setPreview] = useState(null);
  const [result, setResult] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleFile = (e) => {
    const file = e.target.files[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = (ev) => {
      const text = ev.target.result;
      setCsvText(text);
      setPreview(parseCsvPreview(text));
    };
    reader.readAsText(file);
  };

  const hasErrors = preview?.rows?.some(r => r._errors?.length > 0);

  const handleImport = async () => {
    setLoading(true); setError('');
    const blob = new Blob([csvText], { type: 'text/csv' });
    const fd = new FormData();
    fd.append('file', blob, 'import.csv');
    try {
      const r = await api.post('/parts/import', fd, { headers: { 'Content-Type': 'multipart/form-data' } });
      setResult(r.data.data);
      setPreview(null);
    } catch (err) {
      setError(err.response?.data?.errorMessage || 'Import failed');
    } finally { setLoading(false); }
  };

  return (
    <div className="max-w-4xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">Bulk Import Parts</h1>
      <div className="bg-white rounded-lg shadow p-6 space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">Upload CSV File</label>
          <p className="text-xs text-gray-500 mb-2">Required columns: part_number, name, description</p>
          <input type="file" accept=".csv" onChange={handleFile} className="block text-sm" data-testid="csv-input" />
        </div>

        {preview && (
          <div>
            <h2 className="text-lg font-semibold mb-2">Preview ({preview.rows.length} rows)</h2>
            <div className="overflow-auto max-h-96">
              <table className="min-w-full divide-y divide-gray-200 text-xs" data-testid="preview-table">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="px-2 py-2 text-left">Status</th>
                    {preview.headers.map(h => <th key={h} className="px-2 py-2 text-left">{h}</th>)}
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-200">
                  {preview.rows.map(row => (
                    <tr key={row._index} className={row._errors?.length > 0 ? 'bg-red-50' : ''} data-testid={row._errors?.length > 0 ? 'error-row' : 'valid-row'}>
                      <td className="px-2 py-2">{row._errors?.length > 0 ? <span className="text-red-600">{row._errors.join(', ')}</span> : <span className="text-green-600">OK</span>}</td>
                      {preview.headers.map(h => <td key={h} className="px-2 py-2">{row[h]}</td>)}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {error && <p className="text-red-600 text-sm mt-2">{error}</p>}
            <button onClick={handleImport} disabled={hasErrors || loading}
              className="mt-4 px-6 py-2 bg-blue-600 text-white text-sm rounded-md disabled:opacity-50" data-testid="confirm-import-btn">
              {loading ? 'Importing...' : 'Confirm Import'}
            </button>
            {hasErrors && <p className="text-sm text-red-600 mt-1">Fix validation errors before importing</p>}
          </div>
        )}

        {result && (
          <div className="bg-green-50 rounded p-4">
            <p className="text-green-700 font-medium">Import successful: {result.imported} parts imported</p>
          </div>
        )}
      </div>
    </div>
  );
}
