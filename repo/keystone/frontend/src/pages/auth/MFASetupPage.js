import React, { useState, useEffect } from 'react';
import api from '../../services/api';

export default function MFASetupPage() {
  const [setupData, setSetupData] = useState(null);
  const [verified, setVerified] = useState(false);
  const [totpCode, setTotpCode] = useState('');
  const [error, setError] = useState('');
  const [verifying, setVerifying] = useState(false);

  useEffect(() => {
    api.post('/auth/mfa/setup').then(r => setSetupData(r.data.data)).catch(() => setError('Failed to initialize MFA setup'));
  }, []);

  const handleVerify = async () => {
    if (!totpCode.trim()) { setError('Enter the 6-digit code from your authenticator app'); return; }
    setVerifying(true);
    setError('');
    try {
      await api.post('/auth/mfa/verify', { code: totpCode });
      setVerified(true);
    } catch (e) {
      setError(e.response?.data?.errorMessage || 'Verification failed — check the code and try again');
    } finally {
      setVerifying(false);
    }
  };

  return (
    <div className="max-w-md mx-auto mt-10 bg-white rounded-xl shadow p-6">
      <h2 className="text-xl font-semibold mb-4">Set Up Two-Factor Authentication</h2>
      {error && <p className="text-red-600 text-sm mb-4">{error}</p>}
      {!verified && setupData && (
        <div className="space-y-4">
          <p className="text-sm text-gray-600">Scan the QR code with your authenticator app, or enter the secret manually.</p>
          <div className="bg-gray-50 rounded p-4 text-xs font-mono break-all">{setupData.secret}</div>
          {setupData.qrImageData && (
            <img src={setupData.qrImageData} alt="QR Code" className="mx-auto" />
          )}
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Enter the 6-digit code to confirm setup</label>
            <input
              type="text"
              value={totpCode}
              onChange={e => setTotpCode(e.target.value)}
              maxLength={6}
              placeholder="000000"
              className="w-full border rounded-md px-3 py-2 text-sm"
              data-testid="totp-code-input"
            />
          </div>
          <button
            onClick={handleVerify}
            disabled={verifying || !totpCode.trim()}
            className="w-full py-2 bg-blue-600 text-white rounded-md text-sm font-medium hover:bg-blue-700 disabled:opacity-50"
            data-testid="confirm-mfa-btn"
          >
            {verifying ? 'Verifying...' : 'Verify and Enable MFA'}
          </button>
        </div>
      )}
      {verified && <p className="text-green-600 font-medium">MFA enabled successfully. You will be prompted for a code on next login.</p>}
    </div>
  );
}
