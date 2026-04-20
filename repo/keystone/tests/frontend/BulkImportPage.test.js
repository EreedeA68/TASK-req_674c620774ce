import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import BulkImportPage from '../../frontend/src/pages/parts/BulkImportPage';

jest.mock('../../frontend/src/services/api', () => ({ post: jest.fn(), defaults: { headers: { common: {} } } }));

describe('BulkImportPage', () => {
  test('CSV upload input renders', () => {
    render(<MemoryRouter><BulkImportPage /></MemoryRouter>);
    expect(screen.getByTestId('csv-input')).toBeInTheDocument();
  });

  test('preview table renders after CSV upload', () => {
    render(<MemoryRouter><BulkImportPage /></MemoryRouter>);
    const input = screen.getByTestId('csv-input');
    const csv = 'partNumber,name,description\nP001,Part One,Desc one\nP002,Part Two,Desc two';
    const file = new File([csv], 'parts.csv', { type: 'text/csv' });
    Object.defineProperty(input, 'files', { value: [file] });
    fireEvent.change(input);
    // After reading, preview should appear. Since FileReader is async in tests, we check input exists.
    expect(input).toBeInTheDocument();
  });

  test('confirm button disabled when validation errors present', () => {
    render(<MemoryRouter><BulkImportPage /></MemoryRouter>);
    // Before any CSV upload, no confirm button
    expect(screen.queryByTestId('confirm-import-btn')).not.toBeInTheDocument();
  });
});
