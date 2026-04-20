import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import ListingFormPage from '../../frontend/src/pages/lostfound/ListingFormPage';

jest.mock('../../frontend/src/services/api', () => ({ get: jest.fn(), post: jest.fn(), defaults: { headers: { common: {} } } }));

describe('ListingFormPage', () => {
  test('category dropdown renders all options', () => {
    render(<MemoryRouter><ListingFormPage /></MemoryRouter>);
    const select = screen.getByTestId('category-select');
    expect(select).toBeInTheDocument();
    expect(select.querySelectorAll('option').length).toBeGreaterThan(1);
  });

  test('location field shows validation error for short input', async () => {
    render(<MemoryRouter><ListingFormPage /></MemoryRouter>);
    fireEvent.change(screen.getByTestId('title-input'), { target: { value: 'Lost item' } });
    fireEvent.change(screen.getByTestId('category-select'), { target: { value: 'Electronics' } });
    fireEvent.change(screen.getByTestId('location-input'), { target: { value: 'A' } }); // too short
    fireEvent.submit(screen.getByTestId('title-input').closest('form'));
    expect(screen.getByTestId('location-error')).toBeInTheDocument();
  });

  test('time window end before start shows validation error', () => {
    render(<MemoryRouter><ListingFormPage /></MemoryRouter>);
    fireEvent.change(screen.getByTestId('title-input'), { target: { value: 'Lost wallet' } });
    fireEvent.change(screen.getByTestId('category-select'), { target: { value: 'Keys' } });
    fireEvent.change(screen.getByTestId('location-input'), { target: { value: 'Chicago, IL' } });
    fireEvent.change(screen.getByTestId('start-time'), { target: { value: '2024-06-15T14:00' } });
    fireEvent.change(screen.getByTestId('end-time'), { target: { value: '2024-06-15T12:00' } }); // before start
    fireEvent.submit(screen.getByTestId('title-input').closest('form'));
    expect(screen.getByTestId('time-error')).toBeInTheDocument();
  });
});
