import React, { useRef, useState } from 'react';

const ALLOWED_TYPES = ['application/pdf', 'image/jpeg', 'image/png'];
const MAX_SIZE = 20 * 1024 * 1024; // 20MB

export default function FileUpload({ onFile, accept = '.pdf,.jpg,.jpeg,.png' }) {
  const inputRef = useRef();
  const [error, setError] = useState('');
  const [fileName, setFileName] = useState('');

  const validate = (file) => {
    if (!ALLOWED_TYPES.includes(file.type)) {
      setError('Only PDF, JPG, and PNG files are allowed');
      return false;
    }
    if (file.size > MAX_SIZE) {
      setError('File size must not exceed 20 MB');
      return false;
    }
    return true;
  };

  const handleChange = (e) => {
    const file = e.target.files[0];
    if (!file) return;
    setError('');
    if (!validate(file)) { setFileName(''); return; }
    setFileName(file.name);
    onFile(file);
  };

  const handleDrop = (e) => {
    e.preventDefault();
    const file = e.dataTransfer.files[0];
    if (!file) return;
    setError('');
    if (!validate(file)) { setFileName(''); return; }
    setFileName(file.name);
    onFile(file);
  };

  return (
    <div>
      <div onDrop={handleDrop} onDragOver={e => e.preventDefault()}
        className="border-2 border-dashed border-gray-300 rounded-lg p-8 text-center cursor-pointer hover:border-blue-400"
        onClick={() => inputRef.current.click()} data-testid="file-upload-zone">
        <input ref={inputRef} type="file" accept={accept} onChange={handleChange} className="hidden" data-testid="file-input" />
        {fileName ? (
          <p className="text-sm text-green-700 font-medium">{fileName}</p>
        ) : (
          <p className="text-sm text-gray-500">Drag &amp; drop or click to upload (PDF, JPG, PNG — max 20 MB)</p>
        )}
      </div>
      {error && <p className="mt-1 text-sm text-red-600" data-testid="file-error">{error}</p>}
    </div>
  );
}
