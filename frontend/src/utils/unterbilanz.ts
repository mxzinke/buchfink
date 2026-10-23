import type { Unterbilanz } from '../types';

export interface UnterbilanzRow {
  label: string;
  cents: number;
}

/**
 * Die Zeilen der Unterbilanz im Block „Gesellschafter & Kapital“: Stammkapital
 * gegen Reinvermögen und, wenn das Reinvermögen nicht reicht, wie viel der
 * Gründungsaufwand laut Satzung deckt. Die Haftung steht immer da, auch als
 * Null.
 */
export function unterbilanzRows(u: Unterbilanz): UnterbilanzRow[] {
  const rows: UnterbilanzRow[] = [
    { label: 'Stammkapital', cents: u.shareCapital },
    { label: 'Reinvermögen', cents: u.netAssets },
  ];
  if (u.shortfall > 0) {
    rows.push({ label: 'Unterdeckung', cents: u.shortfall });
    if (u.covered > 0) rows.push({ label: 'Gedeckt durch Gründungsaufwand', cents: u.covered });
  }
  rows.push({ label: 'Haftung der Gesellschafter', cents: u.amount });
  return rows;
}
