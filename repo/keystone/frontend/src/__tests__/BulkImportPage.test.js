import React from 'react';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import BulkImportPage from '../pages/parts/BulkImportPage';

jest.mock('../services/api', () => ({ post: jest.fn(), defaults: { headers: { common: {} } } }));

const api = require('../services/api');

function simulateCsvUpload(input, csvContent) {
  const file = new File([csvContent], 'parts.csv', { type: 'text/csv' });
  const readAsText = jest.fn();
  let onloadCallback;
  const mockReader = {
    readAsText,
    set onload(fn) { onloadCallback = fn; },
    get onload() { return onloadCallback; }
  };
  jest.spyOn(global, 'FileReader').mockImplementation(() => mockReader);
  fireEvent.change(input, { target: { files: [file] } });
  act(() => { onloadCallback({ target: { result: csvContent } }); });
}

beforeEach(() => {
  jest.clearAllMocks();
  jest.restoreAllMocks();
});

describe('BulkImportPage', () => {
  test('CSV upload input renders', () => {
    render(<MemoryRouter><BulkImportPage /></MemoryRouter>);
    expect(screen.getByTestId('csv-input')).toBeInTheDocument();
  });

  test('page heading renders', () => {
    render(<MemoryRouter><BulkImportPage /></MemoryRouter>);
    expect(screen.getByText('Bulk Import Parts')).toBeInTheDocument();
  });

  test('no confirm button before CSV upload', () => {
    render(<MemoryRouter><BulkImportPage /></MemoryRouter>);
    expect(screen.queryByTestId('confirm-import-btn')).not.toBeInTheDocument();
  });

  test('preview table renders after valid CSV upload', async () => {
    render(<MemoryRouter><BulkImportPage /></MemoryRouter>);
    const csv = 'part_number,name,description\nP001,Part One,Desc one\nP002,Part Two,Desc two';
    simulateCsvUpload(screen.getByTestId('csv-input'), csv);
    await waitFor(() => expect(screen.getByTestId('preview-table')).toBeInTheDocument());
  });

  test('valid rows show OK status', async () => {
    render(<MemoryRouter><BulkImportPage /></MemoryRouter>);
    const csv = 'part_number,name,description\nP001,Part One,Desc';
    simulateCsvUpload(screen.getByTestId('csv-input'), csv);
    await waitFor(() => expect(screen.getByTestId('valid-row')).toBeInTheDocument());
  });

  test('row missing part_number shows error', async () => {
    render(<MemoryRouter><BulkImportPage /></MemoryRouter>);
    const csv = 'part_number,name,description\n,Part One,Desc';
    simulateCsvUpload(screen.getByTestId('csv-input'), csv);
    await waitFor(() => expect(screen.getByTestId('error-row')).toBeInTheDocument());
    expect(screen.getByText(/part_number required/i)).toBeInTheDocument();
  });

  test('confirm button disabled when rows have errors', async () => {
    render(<MemoryRouter><BulkImportPage /></MemoryRouter>);
    const csv = 'part_number,name,description\n,Bad Row,Desc';
    simulateCsvUpload(screen.getByTestId('csv-input'), csv);
    await waitFor(() => screen.getByTestId('confirm-import-btn'));
    expect(screen.getByTestId('confirm-import-btn')).toBeDisabled();
  });

  test('confirm button enabled when all rows valid', async () => {
    render(<MemoryRouter><BulkImportPage /></MemoryRouter>);
    const csv = 'part_number,name,description\nP001,Part One,Desc';
    simulateCsvUpload(screen.getByTestId('csv-input'), csv);
    await waitFor(() => screen.getByTestId('confirm-import-btn'));
    expect(screen.getByTestId('confirm-import-btn')).not.toBeDisabled();
  });

  test('successful import shows result message', async () => {
    api.post.mockResolvedValue({ data: { data: { imported: 2 } } });
    render(<MemoryRouter><BulkImportPage /></MemoryRouter>);
    const csv = 'part_number,name,description\nP001,Part One,Desc\nP002,Part Two,Desc2';
    simulateCsvUpload(screen.getByTestId('csv-input'), csv);
    await waitFor(() => screen.getByTestId('confirm-import-btn'));
    fireEvent.click(screen.getByTestId('confirm-import-btn'));
    await waitFor(() => expect(screen.getByText(/2 parts imported/i)).toBeInTheDocument());
  });
});
