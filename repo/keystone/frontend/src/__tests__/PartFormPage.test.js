import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import PartFormPage from '../pages/parts/PartFormPage';

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useParams: () => ({ id: undefined }),
  useNavigate: () => jest.fn(),
}));

jest.mock('../services/api', () => ({ get: jest.fn(), post: jest.fn(), put: jest.fn(), defaults: { headers: { common: {} } } }));

const api = require('../services/api');

beforeEach(() => jest.clearAllMocks());

describe('PartFormPage — new', () => {
  test('renders heading "New Part"', () => {
    render(<MemoryRouter><PartFormPage /></MemoryRouter>);
    expect(screen.getByText('New Part')).toBeInTheDocument();
  });

  test('renders part number field for new part', () => {
    render(<MemoryRouter><PartFormPage /></MemoryRouter>);
    expect(screen.getByTestId('field-partNumber')).toBeInTheDocument();
  });

  test('renders name, description, fitment fields', () => {
    render(<MemoryRouter><PartFormPage /></MemoryRouter>);
    expect(screen.getByTestId('field-name')).toBeInTheDocument();
    expect(screen.getByTestId('field-description')).toBeInTheDocument();
    expect(screen.getByTestId('field-fitmentMake')).toBeInTheDocument();
    expect(screen.getByTestId('field-fitmentModel')).toBeInTheDocument();
    expect(screen.getByTestId('field-fitmentYear')).toBeInTheDocument();
  });

  test('Save and Cancel buttons render', () => {
    render(<MemoryRouter><PartFormPage /></MemoryRouter>);
    expect(screen.getByRole('button', { name: 'Save' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument();
  });

  test('form submit calls API post', async () => {
    api.post.mockResolvedValue({ data: { data: { id: 'part-new' } } });
    render(<MemoryRouter><PartFormPage /></MemoryRouter>);
    fireEvent.change(screen.getByTestId('field-name'), { target: { value: 'Brake Pad' } });
    fireEvent.submit(screen.getByRole('button', { name: 'Save' }).closest('form'));
    await waitFor(() => expect(api.post).toHaveBeenCalledWith('/parts', expect.any(Object)));
  });

  test('shows error message on API failure', async () => {
    api.post.mockRejectedValue({ response: { data: { errorMessage: 'Part number exists' } } });
    render(<MemoryRouter><PartFormPage /></MemoryRouter>);
    fireEvent.submit(screen.getByRole('button', { name: 'Save' }).closest('form'));
    await waitFor(() => expect(screen.getByText('Part number exists')).toBeInTheDocument());
  });
});
