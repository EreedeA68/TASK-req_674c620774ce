import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import MFASetupPage from '../pages/auth/MFASetupPage';

jest.mock('../services/api', () => ({ get: jest.fn(), post: jest.fn(), defaults: { headers: { common: {} } } }));

const api = require('../services/api');

const setupData = { secret: 'JBSWY3DPEHPK3PXP', qrImageData: null };

beforeEach(() => {
  jest.clearAllMocks();
  api.post.mockResolvedValue({ data: { data: setupData } });
});

describe('MFASetupPage', () => {
  test('loads and shows TOTP secret', async () => {
    render(<MemoryRouter><MFASetupPage /></MemoryRouter>);
    await waitFor(() => expect(screen.getByText(setupData.secret)).toBeInTheDocument());
  });

  test('renders code input and verify button', async () => {
    render(<MemoryRouter><MFASetupPage /></MemoryRouter>);
    await waitFor(() => screen.getByTestId('totp-code-input'));
    expect(screen.getByTestId('confirm-mfa-btn')).toBeInTheDocument();
  });

  test('verify button disabled when input is empty', async () => {
    render(<MemoryRouter><MFASetupPage /></MemoryRouter>);
    await waitFor(() => screen.getByTestId('confirm-mfa-btn'));
    expect(screen.getByTestId('confirm-mfa-btn')).toBeDisabled();
  });

  test('verify button enabled when code is entered', async () => {
    render(<MemoryRouter><MFASetupPage /></MemoryRouter>);
    await waitFor(() => screen.getByTestId('totp-code-input'));
    fireEvent.change(screen.getByTestId('totp-code-input'), { target: { value: '123456' } });
    expect(screen.getByTestId('confirm-mfa-btn')).not.toBeDisabled();
  });

  test('shows success message after successful verification', async () => {
    api.post
      .mockResolvedValueOnce({ data: { data: setupData } })
      .mockResolvedValueOnce({});
    render(<MemoryRouter><MFASetupPage /></MemoryRouter>);
    await waitFor(() => screen.getByTestId('totp-code-input'));
    fireEvent.change(screen.getByTestId('totp-code-input'), { target: { value: '123456' } });
    fireEvent.click(screen.getByTestId('confirm-mfa-btn'));
    await waitFor(() => expect(screen.getByText(/MFA enabled successfully/i)).toBeInTheDocument());
  });

  test('shows error when verification fails', async () => {
    api.post
      .mockResolvedValueOnce({ data: { data: setupData } })
      .mockRejectedValueOnce({ response: { data: { errorMessage: 'Invalid code' } } });
    render(<MemoryRouter><MFASetupPage /></MemoryRouter>);
    await waitFor(() => screen.getByTestId('totp-code-input'));
    fireEvent.change(screen.getByTestId('totp-code-input'), { target: { value: '000000' } });
    fireEvent.click(screen.getByTestId('confirm-mfa-btn'));
    await waitFor(() => expect(screen.getByText('Invalid code')).toBeInTheDocument());
  });

  test('shows error when setup fails to initialize', async () => {
    api.post.mockRejectedValueOnce(new Error('network'));
    render(<MemoryRouter><MFASetupPage /></MemoryRouter>);
    await waitFor(() => expect(screen.getByText(/Failed to initialize MFA setup/i)).toBeInTheDocument());
  });
});
