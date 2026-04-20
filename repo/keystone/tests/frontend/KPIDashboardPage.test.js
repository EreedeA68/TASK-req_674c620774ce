import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import KPIDashboardPage from '../../frontend/src/pages/reports/KPIDashboardPage';

jest.mock('../../frontend/src/services/api', () => ({
  get: jest.fn().mockResolvedValue({ data: { data: { conversionRate: 0.65, reviewCycleTime: 24.5, quotaUtilization: 0.8 } } }),
  defaults: { headers: { common: {} } }
}));

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

  test('sensitive fields show masked value in export preview', async () => {
    render(<MemoryRouter><KPIDashboardPage /></MemoryRouter>);
    await waitFor(() => {
      const masked = screen.getAllByTestId('masked-field');
      expect(masked.length).toBeGreaterThan(0);
    });
  });
});
