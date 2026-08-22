// SPDX-FileCopyrightText: 2026 Maximilian P.
// SPDX-License-Identifier: EUPL-1.2

/**
 * Formatierung für de-DE. Beträge kommen aus dem Backend als ganze Cent.
 */

import type { Cents, TaxRate } from '../types';

/**
 * Formatiert einen Cent-Betrag, z. B. 119000 → "1.190,00 €".
 *
 * Die Umrechnung passiert über Integer-Division statt über eine Division durch
 * 100, damit auch große Beträge exakt bleiben und nicht über den Umweg einer
 * Fließkommazahl gerundet werden.
 */
export function formatCents(amount: Cents, currency = 'EUR'): string {
  if (!Number.isFinite(amount)) return '—';

  const negative = amount < 0;
  const absolute = Math.abs(Math.trunc(amount));
  const whole = Math.floor(absolute / 100);
  const fraction = absolute % 100;

  const grouped = whole.toLocaleString('de-DE');
  const symbol = currency === 'EUR' ? ' €' : ` ${currency}`;

  return `${negative ? '−' : ''}${grouped},${String(fraction).padStart(2, '0')}${symbol}`;
}

/** Wie formatCents, aber ohne Währungszeichen — für Tabellenspalten. */
export function formatCentsPlain(amount: Cents): string {
  return formatCents(amount, '').trimEnd();
}

/** Wandelt eine Nutzereingabe ("1.234,56", "1234.56") in Cent um. */
export function parseCents(input: string): Cents | null {
  const raw = input.trim().replace(/[€\s]/g, '');
  if (!raw) return null;

  let sign = 1;
  let body = raw;
  if (body.startsWith('-')) {
    sign = -1;
    body = body.slice(1);
  } else if (body.startsWith('+')) {
    body = body.slice(1);
  }

  // Deutsche Schreibweise: Punkt gruppiert, Komma trennt die Nachkommastellen.
  if (body.includes(',')) {
    body = body.replace(/\./g, '').replace(',', '.');
  }
  if (!/^\d*(\.\d{0,2})?$/.test(body)) return null;

  const [whole, fraction = ''] = body.split('.');
  const cents = Number(whole || '0') * 100 + Number(fraction.padEnd(2, '0') || '0');
  return Number.isFinite(cents) ? sign * cents : null;
}

/** Formatiert einen Steuersatz in Basispunkten: 1900 → "19 %". */
export function formatTaxRate(rate: TaxRate): string {
  if (rate % 100 === 0) return `${rate / 100} %`;
  return `${(rate / 100).toFixed(2).replace('.', ',')} %`;
}

/** Formatiert eine Menge mit drei Nachkommastellen: 1500 → "1,5". */
export function formatQuantity(quantityMilli: number): string {
  const value = quantityMilli / 1000;
  return value.toLocaleString('de-DE', { maximumFractionDigits: 3 });
}

export function formatDate(dateStr: string): string {
  if (!dateStr) return '—';
  const parts = dateStr.split('T')[0].split('-');
  if (parts.length === 3) return `${parts[2]}.${parts[1]}.${parts[0]}`;
  const parsed = new Date(dateStr);
  return Number.isNaN(parsed.getTime()) ? dateStr : new Intl.DateTimeFormat('de-DE').format(parsed);
}

/** Zeigt einen Leistungszeitraum an; bei Zeitpunktleistung nur ein Datum. */
export function formatDateRange(from: string, to: string): string {
  if (!from) return '—';
  if (!to || from === to) return formatDate(from);
  return `${formatDate(from)} – ${formatDate(to)}`;
}

export function formatShortHash(hash: string): string {
  if (!hash) return '—';
  if (hash.length <= 12) return hash;
  return `${hash.substring(0, 6)}…${hash.substring(hash.length - 6)}`;
}

/**
 * Beschriftet eine Buchungsseite ausgeschrieben. „S" und „H" sind die
 * Kürzel der Datenhaltung, in der Oberfläche steht Soll und Haben.
 */
export function formatSide(side: 'S' | 'H'): string {
  return side === 'S' ? 'Soll' : 'Haben';
}
