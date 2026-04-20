import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AuthContext } from '../context/AuthContext';
import CandidateDetailPage from '../pages/candidates/CandidateDetailPage';

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useParams: () => ({ id: 'cand-001' }),
}));

jest.mock('../services/api', () => ({ get: jest.fn(), post: jest.fn(), defaults: { headers: { common: {} } } }));

const api = require('../services/api');

const mockCandidate = (overrides = {}) => ({
  id: 'cand-001', status: 'DRAFT', completenessStatus: 'complete',
  createdAt: '2024-01-01T10:00:00Z', submittedAt: null, reviewedAt: null,
  reviewerComments: null, demographics: { firstName: 'John', lastName: 'Doe' },
  examScores: { score: 85 }, applicationDetails: { position: 'Officer' },
  transferPreferences: {}, ...overrides
});

const ctx = (role) => ({ user: { role, username: 'tester' } });

const wrap = (role, candidate) => {
  api.get.mockResolvedValue({ data: { data: candidate || mockCandidate() } });
  return render(
    <AuthContext.Provider value={ctx(role)}>
      <MemoryRouter><CandidateDetailPage /></MemoryRouter>
    </AuthContext.Provider>
  );
};

beforeEach(() => jest.clearAllMocks());

describe('CandidateDetailPage', () => {
  test('shows loading initially', () => {
    api.get.mockReturnValue(new Promise(() => {}));
    render(<AuthContext.Provider value={ctx('REVIEWER')}><MemoryRouter><CandidateDetailPage /></MemoryRouter></AuthContext.Provider>);
    expect(screen.getByText('Loading...')).toBeInTheDocument();
  });

  test('shows error when API fails', async () => {
    api.get.mockRejectedValue(new Error('network'));
    render(<AuthContext.Provider value={ctx('REVIEWER')}><MemoryRouter><CandidateDetailPage /></MemoryRouter></AuthContext.Provider>);
    await waitFor(() => expect(screen.getByText(/Failed to load/i)).toBeInTheDocument());
  });

  test('renders candidate details and status badge', async () => {
    wrap('REVIEWER');
    await waitFor(() => expect(screen.getByTestId('status-badge')).toBeInTheDocument());
    expect(screen.getByText('Candidate Detail')).toBeInTheDocument();
  });

  test('INTAKE_SPECIALIST sees submit button for DRAFT candidate', async () => {
    wrap('INTAKE_SPECIALIST');
    await waitFor(() => expect(screen.getByTestId('submit-btn')).toBeInTheDocument());
  });

  test('REVIEWER does not see submit button', async () => {
    wrap('REVIEWER');
    await waitFor(() => expect(screen.queryByTestId('submit-btn')).not.toBeInTheDocument());
  });

  test('REVIEWER sees approve and reject buttons for SUBMITTED candidate', async () => {
    wrap('REVIEWER', mockCandidate({ status: 'SUBMITTED' }));
    await waitFor(() => {
      expect(screen.getByTestId('approve-btn')).toBeInTheDocument();
      expect(screen.getByTestId('reject-btn')).toBeInTheDocument();
    });
  });

  test('clicking approve btn shows approve form', async () => {
    wrap('REVIEWER', mockCandidate({ status: 'SUBMITTED' }));
    await waitFor(() => screen.getByTestId('approve-btn'));
    fireEvent.click(screen.getByTestId('approve-btn'));
    expect(screen.getByTestId('approve-comments')).toBeInTheDocument();
    expect(screen.getByTestId('approve-submit')).toBeInTheDocument();
  });

  test('clicking reject btn shows reject form', async () => {
    wrap('REVIEWER', mockCandidate({ status: 'SUBMITTED' }));
    await waitFor(() => screen.getByTestId('reject-btn'));
    fireEvent.click(screen.getByTestId('reject-btn'));
    expect(screen.getByTestId('reject-comments')).toBeInTheDocument();
    expect(screen.getByTestId('reject-submit')).toBeInTheDocument();
  });

  test('reject submit is disabled when comments empty', async () => {
    wrap('REVIEWER', mockCandidate({ status: 'SUBMITTED' }));
    await waitFor(() => screen.getByTestId('reject-btn'));
    fireEvent.click(screen.getByTestId('reject-btn'));
    expect(screen.getByTestId('reject-submit')).toBeDisabled();
  });

  test('AUDITOR sees no approve/reject/submit buttons', async () => {
    wrap('AUDITOR');
    await waitFor(() => expect(screen.getByTestId('status-badge')).toBeInTheDocument());
    expect(screen.queryByTestId('approve-btn')).not.toBeInTheDocument();
    expect(screen.queryByTestId('reject-btn')).not.toBeInTheDocument();
    expect(screen.queryByTestId('submit-btn')).not.toBeInTheDocument();
  });

  test('demographics are masked for non-admin/reviewer roles', async () => {
    wrap('INTAKE_SPECIALIST', mockCandidate({ demographics: { firstName: 'John', dob: '1990-01-01' } }));
    await waitFor(() => expect(screen.getByTestId('status-badge')).toBeInTheDocument());
    expect(screen.queryByText('1990-01-01')).not.toBeInTheDocument();
  });
});
