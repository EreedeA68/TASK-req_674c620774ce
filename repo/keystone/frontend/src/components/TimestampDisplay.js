import React from 'react';
import { formatDate } from '../utils/formatDate';

export default function TimestampDisplay({ value }) {
  return <span data-testid="timestamp-display">{formatDate(value)}</span>;
}
