import React from 'react';

export default function MaskedField({ value }) {
  if (!value) return <span>—</span>;
  const masked = '••••' + String(value).slice(-4);
  return <span data-testid="masked-field" aria-label="masked value">{masked}</span>;
}
