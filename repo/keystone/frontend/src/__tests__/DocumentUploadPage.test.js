import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import DocumentUploadPage from '../pages/candidates/DocumentUploadPage';

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useParams: () => ({ id: 'cand-001' }),
}));

jest.mock('../services/api', () => ({ post: jest.fn(), defaults: { headers: { common: {} } } }));

const api = require('../services/api');

beforeEach(() => jest.clearAllMocks());

describe('DocumentUploadPage', () => {
  test('renders heading and file input', () => {
    render(<MemoryRouter><DocumentUploadPage /></MemoryRouter>);
    expect(screen.getByText('Upload Document')).toBeInTheDocument();
    expect(screen.getByTestId('file-input')).toBeInTheDocument();
  });

  test('upload button not shown before file selected', () => {
    render(<MemoryRouter><DocumentUploadPage /></MemoryRouter>);
    expect(screen.queryByRole('button', { name: /upload/i })).not.toBeInTheDocument();
  });

  test('upload button appears after valid file selected', async () => {
    render(<MemoryRouter><DocumentUploadPage /></MemoryRouter>);
    const input = screen.getByTestId('file-input');
    const file = new File(['content'], 'doc.pdf', { type: 'application/pdf' });
    Object.defineProperty(file, 'size', { value: 1024 });
    fireEvent.change(input, { target: { files: [file] } });
    await waitFor(() => expect(screen.getByRole('button', { name: 'Upload' })).toBeInTheDocument());
  });

  test('shows success result after upload', async () => {
    api.post.mockResolvedValue({ data: { data: { fileName: 'doc.pdf', sha256Hash: 'abc123' } } });
    render(<MemoryRouter><DocumentUploadPage /></MemoryRouter>);
    const input = screen.getByTestId('file-input');
    const file = new File(['content'], 'doc.pdf', { type: 'application/pdf' });
    Object.defineProperty(file, 'size', { value: 1024 });
    fireEvent.change(input, { target: { files: [file] } });
    await waitFor(() => screen.getByRole('button', { name: 'Upload' }));
    fireEvent.click(screen.getByRole('button', { name: 'Upload' }));
    await waitFor(() => expect(screen.getByText(/Upload successful/i)).toBeInTheDocument());
    expect(screen.getByTestId('sha256-hash').textContent).toBe('abc123');
  });

  test('shows error message when upload fails', async () => {
    api.post.mockRejectedValue({ response: { data: { errorMessage: 'File type not allowed' } } });
    render(<MemoryRouter><DocumentUploadPage /></MemoryRouter>);
    const input = screen.getByTestId('file-input');
    const file = new File(['content'], 'doc.pdf', { type: 'application/pdf' });
    Object.defineProperty(file, 'size', { value: 1024 });
    fireEvent.change(input, { target: { files: [file] } });
    await waitFor(() => screen.getByRole('button', { name: 'Upload' }));
    fireEvent.click(screen.getByRole('button', { name: 'Upload' }));
    await waitFor(() => expect(screen.getByText('File type not allowed')).toBeInTheDocument());
  });
});
