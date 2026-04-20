import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AuthContext } from '../context/AuthContext';
import ListingDetailPage from '../pages/lostfound/ListingDetailPage';

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useParams: () => ({ id: 'listing-001' }),
  useNavigate: () => jest.fn(),
}));

jest.mock('../services/api', () => ({ get: jest.fn(), post: jest.fn(), delete: jest.fn(), defaults: { headers: { common: {} } } }));

const api = require('../services/api');

const mockListing = (overrides = {}) => ({
  id: 'listing-001', title: 'Lost Wallet', category: 'Wallet',
  locationDescription: 'Denver, CO', status: 'PUBLISHED',
  isDuplicateFlagged: false,
  timeWindowStart: '2024-06-01T09:00:00Z', timeWindowEnd: '2024-07-01T09:00:00Z',
  createdAt: '2024-01-01T10:00:00Z', updatedAt: '2024-01-02T10:00:00Z',
  ...overrides
});

const wrap = (role, listing) => {
  api.get.mockResolvedValue({ data: { data: listing || mockListing() } });
  return render(
    <AuthContext.Provider value={{ user: { role, username: 'tester' } }}>
      <MemoryRouter><ListingDetailPage /></MemoryRouter>
    </AuthContext.Provider>
  );
};

beforeEach(() => jest.clearAllMocks());

describe('ListingDetailPage', () => {
  test('shows loading state initially', () => {
    api.get.mockReturnValue(new Promise(() => {}));
    render(<AuthContext.Provider value={{ user: { role: 'REVIEWER', username: 't' } }}>
      <MemoryRouter><ListingDetailPage /></MemoryRouter>
    </AuthContext.Provider>);
    expect(screen.getByText('Loading...')).toBeInTheDocument();
  });

  test('renders listing title and status badge', async () => {
    wrap('REVIEWER');
    await waitFor(() => expect(screen.getByText('Lost Wallet')).toBeInTheDocument());
    expect(screen.getByTestId('status-badge')).toBeInTheDocument();
  });

  test('renders category and location', async () => {
    wrap('REVIEWER');
    await waitFor(() => expect(screen.getByText('Denver, CO')).toBeInTheDocument());
    expect(screen.getByText('Wallet')).toBeInTheDocument();
  });

  test('ADMIN sees delete button', async () => {
    wrap('ADMIN');
    await waitFor(() => expect(screen.getByTestId('delete-btn')).toBeInTheDocument());
  });

  test('REVIEWER does not see delete button', async () => {
    wrap('REVIEWER');
    await waitFor(() => screen.getByText('Lost Wallet'));
    expect(screen.queryByTestId('delete-btn')).not.toBeInTheDocument();
  });

  test('PUBLISHED listing shows unlist button for ADMIN', async () => {
    wrap('ADMIN');
    await waitFor(() => expect(screen.getByTestId('unlist-btn')).toBeInTheDocument());
  });

  test('duplicate flagged listing shows warning and override button for REVIEWER', async () => {
    wrap('REVIEWER', mockListing({ isDuplicateFlagged: true }));
    await waitFor(() => expect(screen.getByText(/Flagged as possible duplicate/i)).toBeInTheDocument());
    expect(screen.getByTestId('override-btn')).toBeInTheDocument();
  });

  test('override button not shown to INVENTORY_CLERK', async () => {
    wrap('INVENTORY_CLERK', mockListing({ isDuplicateFlagged: true }));
    await waitFor(() => screen.getByText(/Flagged as possible duplicate/i));
    expect(screen.queryByTestId('override-btn')).not.toBeInTheDocument();
  });

  test('clicking override opens confirm dialog', async () => {
    wrap('REVIEWER', mockListing({ isDuplicateFlagged: true }));
    await waitFor(() => screen.getByTestId('override-btn'));
    fireEvent.click(screen.getByTestId('override-btn'));
    expect(screen.getByTestId('confirm-btn')).toBeInTheDocument();
    expect(screen.getByTestId('cancel-btn')).toBeInTheDocument();
  });
});
