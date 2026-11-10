import type { CompanySettings, Task } from '../types';

/**
 * Was in den Stammdaten fehlt. Stammdaten trägt nur der Mensch ein; keine
 * verbundene App ändert sie. Die Blöcke unter Unternehmen › Stammdaten und die
 * Aufgabe in der Übersicht lesen dieselben Listen, damit beide dasselbe sagen.
 */

export function firmGaps(s: CompanySettings): string[] {
  const inFounding = Boolean(s.inGruendung);
  return [
    !s.seat && 'Sitz',
    !s.managingDirectors && 'Geschäftsführung',
    !inFounding && (!s.registerCourt || !s.registerNumber) && 'Registereintrag',
  ].filter(Boolean) as string[];
}

export function addressGaps(s: CompanySettings): string[] {
  return [!s.street && 'Straße', !s.postalCode && 'Postleitzahl', !s.city && 'Ort'].filter(
    Boolean,
  ) as string[];
}

export function taxGaps(s: CompanySettings): string[] {
  return [
    !s.taxNumber && 'Steuernummer',
    !s.federalState && 'Bundesland',
    (!s.vatPeriod || s.vatPeriod === 'unknown') && 'Voranmeldungszeitraum',
  ].filter(Boolean) as string[];
}

export function bankGaps(bankAccounts: number): string[] {
  return bankAccounts === 0 ? ['Bankkonto'] : [];
}

export interface CapitalFacts {
  /** Ist die Gründung erfasst? */
  hasFoundation: boolean;
  /** Das früheste Geschäftsjahr der Bücher. Liegt die Gründung darin, gehört sie erfasst. */
  firstYear: number;
}

export function capitalGaps(s: CompanySettings, facts: CapitalFacts): string[] {
  const foundedInBooks = s.foundedOn.length === 10 && Number(s.foundedOn.slice(0, 4)) >= facts.firstYear;
  return [
    !s.shareCapital && 'Stammkapital',
    (s.shareholders ?? []).length === 0 && 'Gesellschafter',
    foundedInBooks && !facts.hasFoundation && 'Gründung',
  ].filter(Boolean) as string[];
}

export interface ProfileFacts extends CapitalFacts {
  bankAccounts: number;
}

/** Alle fehlenden Angaben in der Reihenfolge der Blöcke. */
export function profileGaps(s: CompanySettings, facts: ProfileFacts): string[] {
  return [
    ...firmGaps(s),
    ...addressGaps(s),
    ...taxGaps(s),
    ...bankGaps(facts.bankAccounts),
    ...capitalGaps(s, facts),
  ];
}

/** Der Schlüssel der Aufgabe in der Übersicht. Sie entsteht hier, nicht im Backend. */
export const PROFILE_TASK_KEY = 'stammdaten';

/** In der Übersicht steht die Anschrift als eine Angabe. */
const SHORT: Record<string, string> = { Straße: 'Anschrift', Postleitzahl: 'Anschrift', Ort: 'Anschrift' };

/**
 * Die Aufgabe „Stammdaten vervollständigen“, solange etwas fehlt. Ohne
 * Steuernummer und Voranmeldungszeitraum fehlen Fristen, ohne Anschrift und
 * Bankkonto lässt sich keine Rechnung ausstellen. Was fehlt, steht im Titel.
 */
export function profileTask(gaps: string[]): Task | null {
  if (gaps.length === 0) return null;
  const short = Array.from(new Set(gaps.map((gap) => SHORT[gap] ?? gap)));
  return {
    key: PROFILE_TASK_KEY,
    group: 'open',
    title: `Stammdaten vervollständigen: ${short.join(', ')}`,
    why: 'Stammdaten tragen nur Sie ein. Ohne sie fehlen Fristen und Angaben auf Rechnungen.',
    target: { area: 'company', tab: 'profile' },
    count: 0,
    amount: 0,
  };
}
