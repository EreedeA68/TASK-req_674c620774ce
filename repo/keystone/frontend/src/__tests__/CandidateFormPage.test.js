import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import CandidateFormPage from '../pages/candidates/CandidateFormPage';

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useParams: () => ({ id: undefined }),
  useNavigate: () => jest.fn(),
}));

jest.mock('../services/api', () => ({ get: jest.fn(), post: jest.fn(), put: jest.fn(), defaults: { headers: { common: {} } } }));

const api = require('../services/api');

beforeEach(() => jest.clearAllMocks());

describe('CandidateFormPage — new', () => {
  test('renders all form fields', () => {
    render(<MemoryRouter><CandidateFormPage /></MemoryRouter>);
    expect(screen.getByTestId('field-firstName')).toBeInTheDocument();
    expect(screen.getByTestId('field-lastName')).toBeInTheDocument();
    expect(screen.getByTestId('field-dob')).toBeInTheDocument();
    expect(screen.getByTestId('field-examScore')).toBeInTheDocument();
    expect(screen.getByTestId('field-applicationDate')).toBeInTheDocument();
    expect(screen.getByTestId('field-position')).toBeInTheDocument();
  });

  test('renders Save and Cancel buttons', () => {
    render(<MemoryRouter><CandidateFormPage /></MemoryRouter>);
    expect(screen.getByRole('button', { name: 'Save' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument();
  });

  test('shows heading "New Candidate"', () => {
    render(<MemoryRouter><CandidateFormPage /></MemoryRouter>);
    expect(screen.getByText('New Candidate')).toBeInTheDocument();
  });

  test('submits form and calls API post', async () => {
    api.post.mockResolvedValue({ data: { data: { id: 'new-id' } } });
    render(<MemoryRouter><CandidateFormPage /></MemoryRouter>);
    fireEvent.change(screen.getByTestId('field-firstName'), { target: { value: 'Jane' } });
    fireEvent.change(screen.getByTestId('field-lastName'), { target: { value: 'Smith' } });
    fireEvent.submit(screen.getByRole('button', { name: 'Save' }).closest('form'));
    await waitFor(() => expect(api.post).toHaveBeenCalledWith('/candidates', expect.any(Object)));
  });

  test('shows error message when API fails', async () => {
    api.post.mockRejectedValue({ response: { data: { errorMessage: 'Duplicate entry' } } });
    render(<MemoryRouter><CandidateFormPage /></MemoryRouter>);
    fireEvent.submit(screen.getByRole('button', { name: 'Save' }).closest('form'));
    await waitFor(() => expect(screen.getByText('Duplicate entry')).toBeInTheDocument());
  });

  test('save button shows Saving... while in flight', async () => {
    let resolve;
    api.post.mockReturnValue(new Promise(r => { resolve = r; }));
    render(<MemoryRouter><CandidateFormPage /></MemoryRouter>);
    fireEvent.submit(screen.getByRole('button', { name: 'Save' }).closest('form'));
    await waitFor(() => expect(screen.getByText('Saving...')).toBeInTheDocument());
    resolve({ data: { data: { id: 'x' } } });
  });
});
