import React from 'react';
import { render, screen } from '@testing-library/react';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { AuthContext } from '../../frontend/src/context/AuthContext';
import RoleGuard from '../../frontend/src/components/RoleGuard';

const makeCtx = (role) => ({ user: role ? { role, username: 'test' } : null });

const wrap = (role, roles) => render(
  <AuthContext.Provider value={makeCtx(role)}>
    <MemoryRouter initialEntries={['/protected']}>
      <Routes>
        <Route path="/protected" element={<RoleGuard roles={roles}><div data-testid="content">Content</div></RoleGuard>} />
        <Route path="/login" element={<div data-testid="login">Login</div>} />
        <Route path="/unauthorized" element={<div data-testid="unauth">Unauthorized</div>} />
      </Routes>
    </MemoryRouter>
  </AuthContext.Provider>
);

describe('RoleGuard', () => {
  test('allows render for permitted role', () => {
    wrap('ADMIN', ['ADMIN']);
    expect(screen.getByTestId('content')).toBeInTheDocument();
  });

  test('redirects to /unauthorized for wrong role', () => {
    wrap('AUDITOR', ['ADMIN']);
    expect(screen.getByTestId('unauth')).toBeInTheDocument();
  });

  test('redirects to /login when not authenticated', () => {
    wrap(null, ['ADMIN']);
    expect(screen.getByTestId('login')).toBeInTheDocument();
  });
});
