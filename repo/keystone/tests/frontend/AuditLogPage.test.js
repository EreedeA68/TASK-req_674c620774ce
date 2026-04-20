import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { AuthContext } from '../../frontend/src/context/AuthContext';
import AuditLogPage from '../../frontend/src/pages/reports/AuditLogPage';

jest.mock('../../frontend/src/services/api', () => ({
  get: jest.fn().mockResolvedValue({ data: { data: { items: [
    { id: '1', actorId: 'actor-1', action: 'LOGIN', resourceType: 'user', resourceId: null, ipAddress: '127.0.0.1', createdAt: '2024-01-01T10:00:00Z' }
  ], total: 1 } } }),
  defaults: { headers: { common: {} } }
}));

const wrap = (role) => render(
  <AuthContext.Provider value={{ user: { role, username: 'test' } }}>
    <MemoryRouter initialEntries={['/audit-logs']}>
      <Routes>
        <Route path="/audit-logs" element={<AuditLogPage />} />
        <Route path="/unauthorized" element={<div data-testid="unauth">Unauth</div>} />
      </Routes>
    </MemoryRouter>
  </AuthContext.Provider>
);

describe('AuditLogPage', () => {
  test('renders correctly for AUDITOR', async () => {
    wrap('AUDITOR');
    await waitFor(() => expect(screen.getByTestId('data-table')).toBeInTheDocument());
  });

  test('renders correctly for ADMIN', async () => {
    wrap('ADMIN');
    await waitFor(() => expect(screen.getByTestId('data-table')).toBeInTheDocument());
  });

  test('redirects INTAKE_SPECIALIST to /unauthorized', () => {
    wrap('INTAKE_SPECIALIST');
    expect(screen.getByTestId('unauth')).toBeInTheDocument();
  });
});
