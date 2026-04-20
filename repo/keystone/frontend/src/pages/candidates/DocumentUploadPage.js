import React, { useState } from 'react';
import { useParams } from 'react-router-dom';
import api from '../../services/api';
import FileUpload from '../../components/FileUpload';

export default function DocumentUploadPage() {
  const { id } = useParams();
  const [file, setFile] = useState(null);
  const [result, setResult] = useState(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleUpload = async () => {
    if (!file) return;
    setError(''); setLoading(true);
    const fd = new FormData();
    fd.append('file', file);
    try {
      const r = await api.post(`/candidates/${id}/documents`, fd, { headers: { 'Content-Type': 'multipart/form-data' } });
      setResult(r.data.data);
    } catch (err) {
      setError(err.response?.data?.errorMessage || 'Upload failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-xl mx-auto">
      <h1 className="text-2xl font-bold mb-6">Upload Document</h1>
      <div className="bg-white rounded-lg shadow p-6 space-y-4">
        <FileUpload onFile={setFile} />
        {file && (
          <button onClick={handleUpload} disabled={loading}
            className="px-4 py-2 bg-blue-600 text-white text-sm rounded-md hover:bg-blue-700 disabled:opacity-50">
            {loading ? 'Uploading...' : 'Upload'}
          </button>
        )}
        {error && <p className="text-red-600 text-sm">{error}</p>}
        {result && (
          <div className="bg-green-50 rounded p-4 text-sm space-y-1">
            <p className="font-medium text-green-700">Upload successful</p>
            <p><span className="font-medium">File:</span> {result.fileName}</p>
            <p><span className="font-medium">SHA-256:</span> <span className="font-mono text-xs break-all" data-testid="sha256-hash">{result.sha256Hash}</span></p>
          </div>
        )}
      </div>
    </div>
  );
}
