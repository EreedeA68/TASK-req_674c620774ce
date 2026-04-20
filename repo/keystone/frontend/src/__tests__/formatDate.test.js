import { formatDate, formatDateOnly } from '../utils/formatDate';

describe('formatDate', () => {
  test('returns dash for null', () => {
    expect(formatDate(null)).toBe('—');
  });

  test('returns dash for empty string', () => {
    expect(formatDate('')).toBe('—');
  });

  test('returns dash for invalid date string', () => {
    expect(formatDate('not-a-date')).toBe('—');
  });

  test('formats valid ISO date with AM/PM', () => {
    const result = formatDate('2024-06-15T14:30:00Z');
    expect(result).toMatch(/\d{2}\/\d{2}\/\d{4}/);
    expect(result).toMatch(/AM|PM/);
  });

  test('formats valid ISO date consistently', () => {
    const result = formatDate('2024-01-01T00:00:00Z');
    expect(typeof result).toBe('string');
    expect(result).not.toBe('—');
  });
});

describe('formatDateOnly', () => {
  test('returns dash for null', () => {
    expect(formatDateOnly(null)).toBe('—');
  });

  test('returns dash for empty string', () => {
    expect(formatDateOnly('')).toBe('—');
  });

  test('returns dash for invalid date', () => {
    expect(formatDateOnly('bad')).toBe('—');
  });

  test('formats valid date as MM/DD/YYYY', () => {
    const result = formatDateOnly('2024-06-15T00:00:00Z');
    expect(result).toMatch(/\d{2}\/\d{2}\/\d{4}/);
    expect(result).not.toMatch(/AM|PM/);
  });
});
