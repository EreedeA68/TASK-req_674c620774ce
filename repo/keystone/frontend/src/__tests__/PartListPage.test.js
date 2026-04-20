import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AuthContext } from '../context/AuthContext';
import PartListPage from '../pages/parts/PartListPage';

jest.mock('../services/api', () => ({ get: jest.fn(), defaults: { headers: { common: {} } } }));

const api = require('../services/api');

const mockParts = [
  { id: 'p1', partNumber: 'P-001', name: 'Brake Pad', status: 'ACTIVE', versionNumber: 1 },
  { id: 'p2', partNumber: 'P-002', name: 'Oil Filter', status: 'ACTIVE', versionNumber: 2 },
];

const wrap = (role) => render(
  <AuthContext.Provider value={{ user: { role, username: 'tester' } }}>
    <MemoryRouter><PartListPage /></MemoryRouter>
  </AuthContext.Provider>
);

beforeEach(() => {
  api.get.mockResolvedValue({ data: { data: { items: mockParts, total: 2 } } });
});

describe('PartListPage', () => {
  test('renders data table with parts', async () => {
    wrap('INVENTORY_CLERK');
    await waitFor(() => expect(screen.getByTestId('data-table')).toBeInTheDocument());
  });

  test('renders search input', () => {
    wrap('INVENTORY_CLERK');
    expect(screen.getByTestId('search-input')).toBeInTheDocument();
  });

  test('INVENTORY_CLERK sees New Part and Bulk Import links', async () => {
    wrap('INVENTORY_CLERK');
    await waitFor(() => expect(screen.getByRole('link', { name: 'New Part' })).toBeInTheDocument());
    expect(screen.getByRole('link', { name: 'Bulk Import' })).toBeInTheDocument();
  });

  test('AUDITOR does not see New Part link', async () => {
    wrap('AUDITOR');
    await waitFor(() => screen.getByTestId('data-table'));
    expect(screen.queryByRole('link', { name: 'New Part' })).not.toBeInTheDocument();
  });

  test('search button triggers API call', async () => {
    wrap('INVENTORY_CLERK');
    await waitFor(() => screen.getByTestId('search-input'));
    fireEvent.change(screen.getByTestId('search-input'), { target: { value: 'brake' } });
    fireEvent.click(screen.getByRole('button', { name: 'Search' }));
    await waitFor(() => expect(api.get).toHaveBeenCalledWith('/parts', expect.objectContaining({
      params: expect.objectContaining({ search: 'brake' })
    })));
  });
});
