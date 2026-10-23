/**
 * Auswahllisten der Stammdaten, gemeinsam für Einrichtung und Unternehmen.
 * Die Werte müssen zu internal/domain passen: Rechtsformen zu
 * `domain.LegalFormGmbH`/`LegalFormUG`, Länder zu `calendar.ValidState`,
 * Erklärungen zu `domain.AllDeclarations`.
 */

export const LEGAL_FORMS = [
  { value: 'GmbH', label: 'GmbH' },
  { value: 'UG (haftungsbeschränkt)', label: 'UG (haftungsbeschränkt)' },
];

/** Die Bundesländer als Kürzel; leer heißt: nur bundesweite Feiertage. */
export const FEDERAL_STATES = [
  { value: 'BW', label: 'Baden-Württemberg' },
  { value: 'BY', label: 'Bayern' },
  { value: 'BE', label: 'Berlin' },
  { value: 'BB', label: 'Brandenburg' },
  { value: 'HB', label: 'Bremen' },
  { value: 'HH', label: 'Hamburg' },
  { value: 'HE', label: 'Hessen' },
  { value: 'MV', label: 'Mecklenburg-Vorpommern' },
  { value: 'NI', label: 'Niedersachsen' },
  { value: 'NW', label: 'Nordrhein-Westfalen' },
  { value: 'RP', label: 'Rheinland-Pfalz' },
  { value: 'SL', label: 'Saarland' },
  { value: 'SN', label: 'Sachsen' },
  { value: 'ST', label: 'Sachsen-Anhalt' },
  { value: 'SH', label: 'Schleswig-Holstein' },
  { value: 'TH', label: 'Thüringen' },
];

export function federalStateLabel(code: string | undefined): string {
  return FEDERAL_STATES.find((s) => s.value === code)?.label ?? '';
}

export const VAT_PERIODS = [
  { value: 'unknown', label: 'Noch nicht geklärt' },
  { value: 'month', label: 'Monatlich' },
  { value: 'quarter', label: 'Vierteljährlich' },
  { value: 'year', label: 'Nur die Jahreserklärung' },
  { value: 'none', label: 'Keine Umsatzsteuererklärungen' },
];

export function vatPeriodLabel(period: string): string {
  return VAT_PERIODS.find((p) => p.value === period)?.label ?? period;
}

/** Die Jahreserklärungen, die ein Steuerberater übernehmen kann. */
export const DECLARATIONS = [
  { value: 'kst', label: 'Körperschaftsteuer' },
  { value: 'gewst', label: 'Gewerbesteuer' },
  { value: 'ust_jahr', label: 'Umsatzsteuer-Jahreserklärung' },
  { value: 'ebilanz', label: 'E-Bilanz' },
];

const MONTH_NAMES = [
  'Januar',
  'Februar',
  'März',
  'April',
  'Mai',
  'Juni',
  'Juli',
  'August',
  'September',
  'Oktober',
  'November',
  'Dezember',
];

/** „Kalenderjahr" oder „1. Juli bis 30. Juni". */
export function fiscalYearLabel(startMonth: number): string {
  const month = startMonth >= 1 && startMonth <= 12 ? startMonth : 1;
  if (month === 1) return 'Kalenderjahr';
  const end = month - 2 < 0 ? 11 : month - 2;
  const lastDay = new Date(2027, end + 1, 0).getDate();
  return `1. ${MONTH_NAMES[month - 1]} bis ${lastDay}. ${MONTH_NAMES[end]}`;
}
