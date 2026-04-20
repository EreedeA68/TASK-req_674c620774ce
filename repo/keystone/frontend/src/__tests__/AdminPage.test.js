import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import AdminPage from '../pages/admin/AdminPage';

jest.mock('../services/api', () => ({ get: jest.fn(), post: jest.fn(), defaults: { headers: { common: {} } } }));

const api = require('../services/api');

const mockUsers = [
  { id: '1', username: 'alice', email: 'alice@test.com', role: 'REVIEWER', isLocked: false, createdAt: '2024-01-01T10:00:00Z' },
  { id: '2', username: 'bob', email: 'bob@test.com', role: 'INTAKE_SPECIALIST', isLocked: false, createdAt: '2024-02-01T10:00:00Z' },
];

beforeEach(() => {
  api.get.mockResolvedValue({ data: { data: { users: mockUsers } } });
  api.post.mockResolvedValue({ data: { data: { id: 'new-user' } } });
});

describe('AdminPage', () => {
  test('renders heading and user table', async () => {
    render(<MemoryRouter><AdminPage /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('data-table')).toBeInTheDocument());
    expect(screen.getByText('Admin — User Management')).toBeInTheDocument();
  });

  test('create user form fields render', () => {
    render(<MemoryRouter><AdminPage /></MemoryRouter>);
    expect(screen.getByTestId('new-username')).toBeInTheDocument();
    expect(screen.getByTestId('new-email')).toBeInTheDocument();
    expect(screen.getByTestId('new-password')).toBeInTheDocument();
    expect(screen.getByTestId('new-role')).toBeInTheDocument();
    expect(screen.getByTestId('create-user-btn')).toBeInTheDocument();
  });

  test('role selector contains all roles', () => {
    render(<MemoryRouter><AdminPage /></MemoryRouter>);
    const select = screen.getByTestId('new-role');
    const options = Array.from(select.querySelectorAll('option')).map(o => o.value);
    expect(options).toContain('ADMIN');
    expect(options).toContain('REVIEWER');
    expect(options).toContain('INTAKE_SPECIALIST');
    expect(options).toContain('INVENTORY_CLERK');
    expect(options).toContain('AUDITOR');
  });

  test('successful user creation shows success message', async () => {
    render(<MemoryRouter><AdminPage /></MemoryRouter>);
    fireEvent.change(screen.getByTestId('new-username'), { target: { value: 'newuser' } });
    fireEvent.change(screen.getByTestId('new-email'), { target: { value: 'new@test.com' } });
    fireEvent.change(screen.getByTestId('new-password'), { target: { value: 'Password@1234!' } });
    fireEvent.submit(screen.getByTestId('create-user-btn').closest('form'));
    await waitFor(() => expect(screen.getByText(/User created successfully/i)).toBeInTheDocument());
  });

  test('shows error when user creation fails', async () => {
    api.post.mockRejectedValue({ response: { data: { errorMessage: 'Username already taken' } } });
    render(<MemoryRouter><AdminPage /></MemoryRouter>);
    fireEvent.submit(screen.getByTestId('create-user-btn').closest('form'));
    await waitFor(() => expect(screen.getByText('Username already taken')).toBeInTheDocument());
  });
});
