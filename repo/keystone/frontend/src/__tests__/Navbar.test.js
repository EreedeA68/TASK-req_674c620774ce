import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AuthContext } from '../context/AuthContext';
import Navbar from '../components/Navbar';

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useNavigate: () => jest.fn(),
}));

const wrap = (role) => {
  const logout = jest.fn().mockResolvedValue();
  render(
    <AuthContext.Provider value={{ user: { role, username: 'tester' }, logout }}>
      <MemoryRouter><Navbar /></MemoryRouter>
    </AuthContext.Provider>
  );
  return { logout };
};

describe('Navbar', () => {
  test('renders navbar element', () => {
    wrap('ADMIN');
    expect(screen.getByTestId('navbar')).toBeInTheDocument();
  });

  test('shows username and role', () => {
    wrap('ADMIN');
    expect(screen.getByText('tester (ADMIN)')).toBeInTheDocument();
  });

  test('ADMIN sees all nav links', () => {
    wrap('ADMIN');
    expect(screen.getByRole('link', { name: 'Candidates' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Parts' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Audit Logs' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Admin' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Reports' })).toBeInTheDocument();
  });

  test('INVENTORY_CLERK sees Parts and Bulk Import links', () => {
    wrap('INVENTORY_CLERK');
    expect(screen.getByRole('link', { name: 'Parts' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Bulk Import' })).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Admin' })).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Audit Logs' })).not.toBeInTheDocument();
  });

  test('AUDITOR sees Audit Logs and Reports links', () => {
    wrap('AUDITOR');
    expect(screen.getByRole('link', { name: 'Audit Logs' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Reports' })).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Admin' })).not.toBeInTheDocument();
  });

  test('INTAKE_SPECIALIST sees Candidates and Lost & Found only', () => {
    wrap('INTAKE_SPECIALIST');
    expect(screen.getByRole('link', { name: 'Candidates' })).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Parts' })).not.toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Admin' })).not.toBeInTheDocument();
  });

  test('Logout button triggers logout', async () => {
    const { logout } = wrap('ADMIN');
    fireEvent.click(screen.getByRole('button', { name: 'Logout' }));
    expect(logout).toHaveBeenCalled();
  });
});
