/**
 * Die Monatsauswahl des Monatsabschlusses.
 *
 * Sie steht hier und nicht zweimal in den Seiten, die den Dialog öffnen
 * (Aufgaben und Fristen): zwei Listen von Monatsnamen laufen auseinander, und
 * die zweite hatte den Fall am Jahreswechsel schon nicht mehr richtig — im
 * Januar steht der Dezember des Vorjahres zur Frage, und eine Liste, die nur
 * das laufende Jahr kennt, zeigt dann einen leeren Auswahlknopf.
 */

/** Ein Monat als Wert („2026-08") und als Name („August 2026"). */
export interface MonthOption {
  value: string;
  label: string;
}

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

/** Die zwölf Monate eines Jahres. */
export function monthsOfYear(year: number): MonthOption[] {
  return MONTH_NAMES.map((name, index) => ({
    value: `${year}-${String(index + 1).padStart(2, '0')}`,
    label: `${name} ${year}`,
  }));
}

/**
 * Die wählbaren Monate zu einem gewählten Monat: sein Jahr und das Vorjahr.
 *
 * Aus dem gewählten Monat und nicht aus dem Kalender von heute: am
 * Jahreswechsel steht der Dezember zur Frage, und eine Liste, die dann nur
 * Januar bis Dezember des neuen Jahres kennt, ließe ihn nicht mehr wählen. Das
 * Vorjahr steht dabei, weil ein Monatsabschluss nachgeholt wird — im Januar ist
 * der offene Monat der Dezember davor.
 */
export function monthOptions(current: string): MonthOption[] {
  const year = Number(current.slice(0, 4)) || new Date().getFullYear();
  return [...monthsOfYear(year - 1), ...monthsOfYear(year)];
}

/** Der Monat vor dem übergebenen Tag als „JJJJ-MM": der, dessen Abschluss ansteht. */
export function previousMonth(today: Date): string {
  const date = new Date(today.getFullYear(), today.getMonth() - 1, 1);
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`;
}
