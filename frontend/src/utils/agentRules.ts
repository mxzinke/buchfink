/**
 * Prüft die Eingaben unter „Buchen ohne Rückfrage“. Ein leeres Feld steht für
 * die Voreinstellung und geht als 0 an die App; deshalb ist 0 als Eingabe
 * kein gültiger Wert: wer „0“ tippt, meint nicht die Voreinstellung.
 */
export interface RulesDraft {
  maxGross: string;
  minBookings: string;
}

export interface RulesDefaults {
  /** Voreingestellte Betragsgrenze, so wie das Feld sie zeigt. */
  maxGrossLabel: string;
  minBookings: number;
}

export interface CheckedRules {
  directMaxGross: number;
  directMinBookings: number;
  errors: Partial<Record<keyof RulesDraft, string>>;
}

export function checkRules(
  draft: RulesDraft,
  defaults: RulesDefaults,
  parseCents: (input: string) => number | null,
): CheckedRules {
  const errors: CheckedRules['errors'] = {};
  let directMaxGross = 0;
  if (draft.maxGross.trim() !== '') {
    const cents = parseCents(draft.maxGross);
    if (cents === null || cents < 0) {
      errors.maxGross = 'Ein Betrag wie 1.000,00.';
    } else if (cents === 0) {
      errors.maxGross = `Mindestens 0,01 €. Ein leeres Feld steht für die Voreinstellung von ${defaults.maxGrossLabel}.`;
    } else {
      directMaxGross = cents;
    }
  }
  let directMinBookings = 0;
  if (draft.minBookings.trim() !== '') {
    const n = Number(draft.minBookings);
    if (!Number.isInteger(n) || n < 0) {
      errors.minBookings = 'Eine ganze Zahl wie 3.';
    } else if (n === 0) {
      errors.minBookings = `Mindestens 1. Ein leeres Feld steht für die Voreinstellung von ${defaults.minBookings}.`;
    } else {
      directMinBookings = n;
    }
  }
  return { directMaxGross, directMinBookings, errors };
}
