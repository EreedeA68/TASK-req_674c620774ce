import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AuthContext } from '../context/AuthContext';
import CandidateListPage from '../pages/candidates/CandidateListPage';

jest.mock('../services/api', () => ({
  get: jest.fn(),
  defaults: { headers: { common: {} } }
}));

const api = require('../services/api');

const mockCandidates = [
  { id: 'uuid-1', status: 'DRAFT', completenessStatus: 'incomplete', createdAt: '2024-01-01T10:00:00Z', submittedAt: null },
  { id: 'uuid-2', status: 'APPROVED', completenessStatus: 'complete', createdAt: '2024-02-01T12:00:00Z', submittedAt: '2024-02-02T09:00:00Z' },
];

const ctx = { user: { role: 'REVIEWER', username: 'test' } };

beforeEach(() => {
  api.get.mockResolvedValue({ data: { data: { items: mockCandidates, total: 2 } } });
});

describe('CandidateListPage', () => {
  test('renders DataTable with candidate rows', async () => {
    render(<AuthContext.Provider value={ctx}><MemoryRouter><CandidateListPage /></MemoryRouter></AuthContext.Provider>);
    await waitFor(() => expect(screen.getByTestId('data-table')).toBeInTheDocument());
  });

  test('StatusBadge renders for each status', async () => {
    render(<AuthContext.Provider value={ctx}><MemoryRouter><CandidateListPage /></MemoryRouter></AuthContext.Provider>);
    await waitFor(() => {
      const badges = screen.getAllByTestId('status-badge');
      expect(badges.length).toBeGreaterThan(0);
    });
  });

  test('timestamps rendered via TimestampDisplay', async () => {
    render(<AuthContext.Provider value={ctx}><MemoryRouter><CandidateListPage /></MemoryRouter></AuthContext.Provider>);
    await waitFor(() => {
      const timestamps = screen.getAllByTestId('timestamp-display');
      expect(timestamps.length).toBeGreaterThan(0);
    });
  });

  test('status filter updates API call', async () => {
    render(<AuthContext.Provider value={ctx}><MemoryRouter><CandidateListPage /></MemoryRouter></AuthContext.Provider>);
    await waitFor(() => screen.getByTestId('status-filter'));
    fireEvent.change(screen.getByTestId('status-filter'), { target: { value: 'APPROVED' } });
    await waitFor(() => expect(api.get).toHaveBeenCalledWith('/candidates', expect.objectContaining({ params: expect.objectContaining({ status: 'APPROVED' }) })));
  });
});
