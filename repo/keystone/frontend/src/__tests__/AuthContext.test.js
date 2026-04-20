import React from 'react';
import { render, screen, waitFor, act } from '@testing-library/react';
import { AuthProvider, AuthContext, useAuth } from '../context/AuthContext';

jest.mock('../services/api', () => ({
  get: jest.fn(),
  post: jest.fn(),
  defaults: { headers: { common: {} } }
}));

const api = require('../services/api');

const Consumer = () => {
  const { user, loading } = useAuth();
  if (loading) return <div data-testid="loading">Loading</div>;
  return <div data-testid="user">{user ? user.username : 'none'}</div>;
};

beforeEach(() => {
  jest.clearAllMocks();
  localStorage.clear();
  delete api.defaults.headers.common['Authorization'];
});

describe('AuthProvider', () => {
  test('renders children without a token (not loading)', async () => {
    api.get.mockResolvedValue({ data: { data: null } });
    render(<AuthProvider><Consumer /></AuthProvider>);
    await waitFor(() => expect(screen.getByTestId('user')).toBeInTheDocument());
    expect(screen.getByTestId('user').textContent).toBe('none');
  });

  test('fetches user when token exists in localStorage', async () => {
    localStorage.setItem('ks_token', 'test-token');
    api.get.mockResolvedValue({ data: { data: { username: 'alice', role: 'ADMIN' } } });
    render(<AuthProvider><Consumer /></AuthProvider>);
    await waitFor(() => expect(screen.getByTestId('user').textContent).toBe('alice'));
  });

  test('clears token on /auth/me failure', async () => {
    localStorage.setItem('ks_token', 'bad-token');
    api.get.mockRejectedValue(new Error('401'));
    render(<AuthProvider><Consumer /></AuthProvider>);
    await waitFor(() => expect(screen.getByTestId('user').textContent).toBe('none'));
    expect(localStorage.getItem('ks_token')).toBeNull();
  });
});

describe('login / logout', () => {
  test('login sets user and token', async () => {
    api.get.mockResolvedValue({ data: { data: { username: 'bob', role: 'REVIEWER' } } });
    api.post.mockResolvedValue({ data: { data: { token: 'tok123', user: { username: 'bob', role: 'REVIEWER' } } } });

    let loginFn;
    const Capture = () => {
      const ctx = useAuth();
      loginFn = ctx.login;
      return <div data-testid="u">{ctx.user ? ctx.user.username : 'none'}</div>;
    };
    render(<AuthProvider><Capture /></AuthProvider>);
    await waitFor(() => expect(screen.getByTestId('u')).toBeInTheDocument());

    await act(async () => { await loginFn('bob', 'pass'); });
    expect(screen.getByTestId('u').textContent).toBe('bob');
    expect(localStorage.getItem('ks_token')).toBe('tok123');
  });

  test('logout clears user and token', async () => {
    localStorage.setItem('ks_token', 'tok');
    api.get.mockResolvedValue({ data: { data: { username: 'alice', role: 'ADMIN' } } });
    api.post.mockResolvedValue({});

    let logoutFn;
    const Capture = () => {
      const ctx = useAuth();
      logoutFn = ctx.logout;
      return <div data-testid="u">{ctx.user ? ctx.user.username : 'none'}</div>;
    };
    render(<AuthProvider><Capture /></AuthProvider>);
    await waitFor(() => expect(screen.getByTestId('u').textContent).toBe('alice'));

    await act(async () => { await logoutFn(); });
    expect(screen.getByTestId('u').textContent).toBe('none');
    expect(localStorage.getItem('ks_token')).toBeNull();
  });
});
