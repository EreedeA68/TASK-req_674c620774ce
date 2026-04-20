import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import MaskedField from '../../frontend/src/components/MaskedField';
import TimestampDisplay from '../../frontend/src/components/TimestampDisplay';
import StatusBadge from '../../frontend/src/components/StatusBadge';
import FileUpload from '../../frontend/src/components/FileUpload';
import ConfirmDialog from '../../frontend/src/components/ConfirmDialog';

describe('MaskedField', () => {
  test('renders last 4 characters only', () => {
    render(<MaskedField value="123456789" />);
    const el = screen.getByTestId('masked-field');
    expect(el.textContent).toContain('6789');
    expect(el.textContent).not.toContain('12345');
  });

  test('full value absent from DOM', () => {
    const { container } = render(<MaskedField value="SECRETVALUE" />);
    expect(container.innerHTML).not.toContain('SECRETVALU');
    expect(container.innerHTML).toContain('ALUE');
  });
});

describe('TimestampDisplay', () => {
  test('renders MM/DD/YYYY h:mm A format', () => {
    render(<TimestampDisplay value="2024-06-15T14:30:00Z" />);
    const el = screen.getByTestId('timestamp-display');
    expect(el.textContent).toMatch(/\d{2}\/\d{2}\/\d{4}/);
    expect(el.textContent).toMatch(/AM|PM/);
  });

  test('renders dash for null value', () => {
    render(<TimestampDisplay value={null} />);
    expect(screen.getByTestId('timestamp-display').textContent).toBe('—');
  });
});

describe('StatusBadge', () => {
  test('renders correct CSS class for APPROVED', () => {
    render(<StatusBadge status="APPROVED" />);
    const el = screen.getByTestId('status-badge');
    expect(el.className).toContain('green');
  });

  test('renders correct CSS class for REJECTED', () => {
    render(<StatusBadge status="REJECTED" />);
    const el = screen.getByTestId('status-badge');
    expect(el.className).toContain('red');
  });

  test('renders correct CSS class for DRAFT', () => {
    render(<StatusBadge status="DRAFT" />);
    const el = screen.getByTestId('status-badge');
    expect(el.className).toContain('gray');
  });
});

describe('FileUpload', () => {
  test('rejects files over 20MB', () => {
    const onFile = jest.fn();
    render(<FileUpload onFile={onFile} />);
    const input = screen.getByTestId('file-input');
    const file = new File(['x'.repeat(21 * 1024 * 1024)], 'big.pdf', { type: 'application/pdf' });
    Object.defineProperty(file, 'size', { value: 21 * 1024 * 1024 });
    fireEvent.change(input, { target: { files: [file] } });
    expect(screen.getByTestId('file-error')).toBeInTheDocument();
    expect(onFile).not.toHaveBeenCalled();
  });

  test('rejects non PDF/JPG/PNG files', () => {
    const onFile = jest.fn();
    render(<FileUpload onFile={onFile} />);
    const input = screen.getByTestId('file-input');
    const file = new File(['data'], 'doc.docx', { type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document' });
    fireEvent.change(input, { target: { files: [file] } });
    expect(screen.getByTestId('file-error')).toBeInTheDocument();
    expect(onFile).not.toHaveBeenCalled();
  });
});

describe('ConfirmDialog', () => {
  test('renders confirm and cancel buttons when open', () => {
    const onConfirm = jest.fn(), onCancel = jest.fn();
    render(<ConfirmDialog open={true} title="Test" message="Are you sure?" onConfirm={onConfirm} onCancel={onCancel} />);
    expect(screen.getByTestId('confirm-btn')).toBeInTheDocument();
    expect(screen.getByTestId('cancel-btn')).toBeInTheDocument();
  });

  test('calls onConfirm when confirm clicked', () => {
    const onConfirm = jest.fn(), onCancel = jest.fn();
    render(<ConfirmDialog open={true} title="T" message="M" onConfirm={onConfirm} onCancel={onCancel} />);
    fireEvent.click(screen.getByTestId('confirm-btn'));
    expect(onConfirm).toHaveBeenCalled();
  });

  test('calls onCancel when cancel clicked', () => {
    const onConfirm = jest.fn(), onCancel = jest.fn();
    render(<ConfirmDialog open={true} title="T" message="M" onConfirm={onConfirm} onCancel={onCancel} />);
    fireEvent.click(screen.getByTestId('cancel-btn'));
    expect(onCancel).toHaveBeenCalled();
  });
});
