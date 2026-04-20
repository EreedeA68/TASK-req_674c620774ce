import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../../context/AuthContext';

export default function LoginPage() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [totpCode, setTotpCode] = useState('');
  const [mfaRequired, setMfaRequired] = useState(false);
  const [error, setError] = useState('');
  const [lockoutMsg, setLockoutMsg] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError(''); setLockoutMsg('');
    setLoading(true);
    try {
      await login(username, password, mfaRequired ? totpCode : undefined);
      navigate('/');
    } catch (err) {
      const status = err.response?.status;
      const msg = err.response?.data?.errorMessage || 'Login failed';
      if (status === 423) setLockoutMsg(err.response?.data?.data?.lockoutUntil ? `Account locked until ${err.response.data.data.lockoutUntil}` : 'Account locked. Try again in 15 minutes.');
      else if (err.response?.data?.details === 'mfa_required') setMfaRequired(true);
      else setError(msg);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-100">
      <div className="bg-white rounded-xl shadow-lg p-8 w-full max-w-sm">
        <h1 className="text-2xl font-bold text-gray-900 mb-6 text-center">Keystone</h1>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Username</label>
            <input type="text" value={username} onChange={e => setUsername(e.target.value)}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              data-testid="username-input" autoComplete="username" required />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Password</label>
            <input type="password" value={password} onChange={e => setPassword(e.target.value)}
              className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              data-testid="password-input" autoComplete="current-password" required />
          </div>
          {mfaRequired && (
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">MFA Code</label>
              <input type="text" value={totpCode} onChange={e => setTotpCode(e.target.value)}
                className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                data-testid="mfa-input" placeholder="6-digit code" maxLength={6} />
            </div>
          )}
          {error && <p className="text-sm text-red-600" data-testid="error-message">{error}</p>}
          {lockoutMsg && <p className="text-sm text-orange-600" data-testid="lockout-message">{lockoutMsg}</p>}
          <button type="submit" disabled={loading}
            className="w-full py-2 px-4 bg-blue-600 text-white text-sm font-medium rounded-md hover:bg-blue-700 disabled:opacity-50"
            data-testid="submit-button">
            {loading ? 'Signing in...' : 'Sign In'}
          </button>
        </form>
      </div>
    </div>
  );
}
