import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AuthContext } from '../context/AuthContext';
import ListingListPage from '../pages/lostfound/ListingListPage';

jest.mock('../services/api', () => ({ get: jest.fn(), defaults: { headers: { common: {} } } }));

const api = require('../services/api');

const mockListings = [
  { id: 'l1', title: 'Lost Keys', category: 'Keys', locationDescription: 'Denver, CO', status: 'PUBLISHED', isDuplicateFlagged: false, createdAt: '2024-01-01T10:00:00Z' },
  { id: 'l2', title: 'Blue Wallet', category: 'Wallet', locationDescription: 'Austin, TX', status: 'PUBLISHED', isDuplicateFlagged: true, createdAt: '2024-02-01T10:00:00Z' },
];

const wrap = (role, listings = mockListings) => {
  api.get.mockResolvedValue({ data: { data: { items: listings, total: listings.length } } });
  return render(
    <AuthContext.Provider value={{ user: { role, username: 'tester' } }}>
      <MemoryRouter><ListingListPage /></MemoryRouter>
    </AuthContext.Provider>
  );
};

describe('ListingListPage', () => {
  test('renders listings grid', async () => {
    wrap('REVIEWER');
    await waitFor(() => expect(screen.getByTestId('listings-grid')).toBeInTheDocument());
  });

  test('shows listing titles', async () => {
    wrap('REVIEWER');
    await waitFor(() => expect(screen.getByText('Lost Keys')).toBeInTheDocument());
    expect(screen.getByText('Blue Wallet')).toBeInTheDocument();
  });

  test('shows duplicate flag badge for flagged listing', async () => {
    wrap('REVIEWER');
    await waitFor(() => expect(screen.getByTestId('duplicate-flag')).toBeInTheDocument());
  });

  test('INVENTORY_CLERK sees New Listing link', async () => {
    wrap('INVENTORY_CLERK');
    await waitFor(() => expect(screen.getByRole('link', { name: 'New Listing' })).toBeInTheDocument());
  });

  test('REVIEWER does not see New Listing link', async () => {
    wrap('REVIEWER');
    await waitFor(() => screen.getByTestId('listings-grid'));
    expect(screen.queryByRole('link', { name: 'New Listing' })).not.toBeInTheDocument();
  });

  test('empty state shows no items', async () => {
    wrap('REVIEWER', []);
    await waitFor(() => expect(screen.getByTestId('listings-grid')).toBeInTheDocument());
  });

  test('prev/next pagination buttons render', async () => {
    wrap('REVIEWER');
    await waitFor(() => screen.getByTestId('listings-grid'));
    expect(screen.getByTestId('prev-page')).toBeInTheDocument();
    expect(screen.getByTestId('next-page')).toBeInTheDocument();
  });
});
