import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import PartDetailPage from '../pages/parts/PartDetailPage';

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useParams: () => ({ id: 'part-001' }),
  useNavigate: () => jest.fn(),
}));

jest.mock('../services/api', () => ({ get: jest.fn(), defaults: { headers: { common: {} } } }));

const api = require('../services/api');

const mockPart = {
  id: 'part-001', partNumber: 'P-001', name: 'Brake Pad', description: 'Front brake pad',
  status: 'ACTIVE', versionNumber: 2, currentVersionID: 'v-002', createdAt: '2024-01-01T10:00:00Z'
};

const mockVersions = [
  { id: 'v-001', versionNumber: 1, changeSummary: 'Initial', createdAt: '2024-01-01T10:00:00Z' },
  { id: 'v-002', versionNumber: 2, changeSummary: 'Updated specs', createdAt: '2024-02-01T10:00:00Z' },
];

beforeEach(() => {
  api.get.mockImplementation((url) => {
    if (url.includes('/versions')) return Promise.resolve({ data: { data: mockVersions } });
    return Promise.resolve({ data: { data: mockPart } });
  });
});

describe('PartDetailPage', () => {
  test('shows loading state', () => {
    api.get.mockReturnValue(new Promise(() => {}));
    render(<MemoryRouter><PartDetailPage /></MemoryRouter>);
    expect(screen.getByText('Loading...')).toBeInTheDocument();
  });

  test('renders part name and part number', async () => {
    render(<MemoryRouter><PartDetailPage /></MemoryRouter>);
    await waitFor(() => expect(screen.getByText('Brake Pad')).toBeInTheDocument());
    expect(screen.getByText('#P-001')).toBeInTheDocument();
  });

  test('renders status badge', async () => {
    render(<MemoryRouter><PartDetailPage /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('status-badge')).toBeInTheDocument());
  });

  test('renders version history section', async () => {
    render(<MemoryRouter><PartDetailPage /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('version-history')).toBeInTheDocument());
  });

  test('shows version entries', async () => {
    render(<MemoryRouter><PartDetailPage /></MemoryRouter>);
    await waitFor(() => expect(screen.getByText('Initial')).toBeInTheDocument());
    expect(screen.getByText('Updated specs')).toBeInTheDocument();
  });

  test('compare link shown for version > 1', async () => {
    render(<MemoryRouter><PartDetailPage /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('compare-btn-2')).toBeInTheDocument());
  });
});
