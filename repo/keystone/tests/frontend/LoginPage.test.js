import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { AuthContext } from '../../frontend/src/context/AuthContext';
import LoginPage from '../../frontend/src/pages/auth/LoginPage';

// Mock navigate
jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useNavigate: () => jest.fn(),
}));

const renderLogin = (loginFn = jest.fn()) =>
  render(
    <AuthContext.Provider value={{ login: loginFn }}>
      <MemoryRouter><LoginPage /></MemoryRouter>
    </AuthContext.Provider>
  );

describe('LoginPage', () => {
  test('renders username field, password field, submit button', () => {
    renderLogin();
    expect(screen.getByTestId('username-input')).toBeInTheDocument();
    expect(screen.getByTestId('password-input')).toBeInTheDocument();
    expect(screen.getByTestId('submit-button')).toBeInTheDocument();
  });

  test('shows error message on failed login', async () => {
    const loginFn = jest.fn().mockRejectedValue({ response: { status: 401, data: { errorMessage: 'invalid username or password' } } });
    renderLogin(loginFn);
    fireEvent.change(screen.getByTestId('username-input'), { target: { value: 'bad' } });
    fireEvent.change(screen.getByTestId('password-input'), { target: { value: 'bad' } });
    fireEvent.click(screen.getByTestId('submit-button'));
    await waitFor(() => expect(screen.getByTestId('error-message')).toBeInTheDocument());
  });

  test('shows MFA input when API indicates MFA required', async () => {
    const loginFn = jest.fn().mockRejectedValue({ response: { status: 401, data: { errorMessage: 'MFA required', details: 'mfa_required' } } });
    renderLogin(loginFn);
    fireEvent.change(screen.getByTestId('username-input'), { target: { value: 'u' } });
    fireEvent.change(screen.getByTestId('password-input'), { target: { value: 'p' } });
    fireEvent.click(screen.getByTestId('submit-button'));
    await waitFor(() => expect(screen.getByTestId('mfa-input')).toBeInTheDocument());
  });

  test('shows lockout message when API returns 423', async () => {
    const loginFn = jest.fn().mockRejectedValue({ response: { status: 423, data: { errorMessage: 'locked', data: {} } } });
    renderLogin(loginFn);
    fireEvent.change(screen.getByTestId('username-input'), { target: { value: 'u' } });
    fireEvent.change(screen.getByTestId('password-input'), { target: { value: 'p' } });
    fireEvent.click(screen.getByTestId('submit-button'));
    await waitFor(() => expect(screen.getByTestId('lockout-message')).toBeInTheDocument());
  });

  test('submit button disabled while request in flight', async () => {
    let resolve;
    const loginFn = jest.fn().mockReturnValue(new Promise(r => { resolve = r; }));
    renderLogin(loginFn);
    fireEvent.change(screen.getByTestId('username-input'), { target: { value: 'u' } });
    fireEvent.change(screen.getByTestId('password-input'), { target: { value: 'p' } });
    fireEvent.click(screen.getByTestId('submit-button'));
    await waitFor(() => expect(screen.getByTestId('submit-button')).toBeDisabled());
    resolve({});
  });
});
