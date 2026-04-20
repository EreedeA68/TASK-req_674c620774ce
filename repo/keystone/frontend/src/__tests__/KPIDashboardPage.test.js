import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import KPIDashboardPage from '../pages/reports/KPIDashboardPage';

jest.mock('../services/api', () => ({
  get: jest.fn(),
  defaults: { headers: { common: {} } }
}));

const api = require('../services/api');

beforeEach(() => {
  api.get.mockResolvedValue({ data: { data: { conversionRate: 0.65, reviewCycleTime: 24.5, quotaUtilization: 0.8 } } });
});

describe('KPIDashboardPage', () => {
  test('conversion rate metric rendered', async () => {
    render(<MemoryRouter><KPIDashboardPage /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('kpi-conversion')).toBeInTheDocument());
  });

  test('review cycle time metric rendered', async () => {
    render(<MemoryRouter><KPIDashboardPage /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('kpi-cycle-time')).toBeInTheDocument());
  });

  test('quota utilization metric rendered', async () => {
    render(<MemoryRouter><KPIDashboardPage /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('kpi-quota')).toBeInTheDocument());
  });

  test('export preview section renders', async () => {
    render(<MemoryRouter><KPIDashboardPage /></MemoryRouter>);
    await waitFor(() => expect(screen.getByTestId('kpi-conversion')).toBeInTheDocument());
    expect(screen.getByText(/Export Preview/i)).toBeInTheDocument();
  });
});
