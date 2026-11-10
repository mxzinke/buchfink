import type {
  DepreciationMethod,
  Side,
  TakeoverAsset,
  TakeoverLine,
  TakeoverOpenItem,
  TakeoverRequest,
} from '../types';

/** Der im Vorsystem erklärte Steuerfall eines offenen Postens; leer heißt unbekannt. */
export type TakeoverTax = '' | '19' | '7' | '0';

export interface TakeoverAccountDraft {
  account: string;
  side: Side;
  amount: string;
}

export interface TakeoverItemDraft {
  contactId: number;
  documentNumber: string;
  documentDate: string;
  dueDate: string;
  amount: string;
  tax: TakeoverTax;
}

export interface TakeoverAssetDraft {
  name: string;
  account: string;
  acquisitionDate: string;
  acquisitionCost: string;
  accumulatedDepreciation: string;
  usefulLifeMonths: string;
  method: DepreciationMethod | '';
}

export interface TakeoverDraft {
  receiptId: number;
  legacySystem: string;
  accounts: TakeoverAccountDraft[];
  receivables: TakeoverItemDraft[];
  payables: TakeoverItemDraft[];
  assets: TakeoverAssetDraft[];
}

type ParseCents = (input: string) => number | null;

// Die Schleifen zählen mit Index: der Test übersetzt diese Datei ohne
// Iteratoren.

const TAX: Record<Exclude<TakeoverTax, ''>, Pick<TakeoverOpenItem, 'taxRate' | 'taxTreatment'>> = {
  '19': { taxRate: 1900, taxTreatment: 'domestic' },
  '7': { taxRate: 700, taxTreatment: 'domestic' },
  '0': { taxRate: 0, taxTreatment: 'not_taxable' },
};

function positive(text: string, parse: ParseCents): number | null {
  const cents = parse(text);
  return cents !== null && cents > 0 ? cents : null;
}

function items(label: string, drafts: TakeoverItemDraft[], parse: ParseCents): TakeoverOpenItem[] | string {
  const out: TakeoverOpenItem[] = [];
  for (let index = 0; index < drafts.length; index++) {
    const d = drafts[index];
    if (!d.contactId && !d.documentNumber.trim() && !d.amount.trim()) continue;
    const where = `${label}, Zeile ${index + 1}`;
    const amount = positive(d.amount, parse);
    if (!d.contactId) return `${where}: Der Kontakt fehlt.`;
    if (!d.documentNumber.trim()) return `${where}: Die Rechnungsnummer fehlt.`;
    if (!d.documentDate) return `${where}: Das Rechnungsdatum fehlt.`;
    if (amount === null) return `${where}: Der offene Betrag fehlt.`;
    out.push({
      contactId: d.contactId,
      amount,
      documentNumber: d.documentNumber.trim(),
      documentDate: d.documentDate,
      dueDate: d.dueDate || undefined,
      ...(d.tax ? TAX[d.tax] : {}),
    });
  }
  return out;
}

/**
 * Baut aus den Zeilen des Dialogs die Schlussbilanz des Vorsystems. Leere
 * Zeilen entfallen; eine unvollständige nennt der Fehler mit ihrer Stelle.
 */
export function buildTakeoverRequest(
  draft: TakeoverDraft,
  parse: ParseCents,
): { request: TakeoverRequest; error?: undefined } | { request?: undefined; error: string } {
  if (!draft.receiptId) return { error: 'Wählen Sie den Beleg mit der Schlussbilanz des Vorsystems.' };

  const accounts: TakeoverLine[] = [];
  for (let index = 0; index < draft.accounts.length; index++) {
    const d = draft.accounts[index];
    if (!d.account.trim() && !d.amount.trim()) continue;
    const amount = positive(d.amount, parse);
    if (!/^\d{4}$/.test(d.account.trim())) return { error: `Bestände, Zeile ${index + 1}: Das Konto fehlt.` };
    if (amount === null) return { error: `Bestände, Zeile ${index + 1}: Der Betrag fehlt.` };
    accounts.push({ account: d.account.trim(), side: d.side, amount });
  }
  if (accounts.length === 0) return { error: 'Erfassen Sie die Bestände der Schlussbilanz.' };

  const receivables = items('Forderungen', draft.receivables, parse);
  if (typeof receivables === 'string') return { error: receivables };
  const payables = items('Verbindlichkeiten', draft.payables, parse);
  if (typeof payables === 'string') return { error: payables };

  const assets: TakeoverAsset[] = [];
  for (let index = 0; index < draft.assets.length; index++) {
    const d = draft.assets[index];
    if (!d.name.trim() && !d.account.trim() && !d.acquisitionCost.trim()) continue;
    const where = `Anlagen, Zeile ${index + 1}`;
    const cost = positive(d.acquisitionCost, parse);
    const accumulated = d.accumulatedDepreciation.trim() ? parse(d.accumulatedDepreciation) : 0;
    const months = d.usefulLifeMonths.trim() ? Number(d.usefulLifeMonths) : 0;
    if (!d.name.trim()) return { error: `${where}: Die Bezeichnung fehlt.` };
    if (!/^\d{4}$/.test(d.account.trim())) return { error: `${where}: Das Anlagekonto fehlt.` };
    if (!d.acquisitionDate) return { error: `${where}: Das Anschaffungsdatum fehlt.` };
    if (cost === null) return { error: `${where}: Die Anschaffungskosten fehlen.` };
    if (accumulated === null || accumulated < 0 || accumulated > cost) {
      return { error: `${where}: Die bisherige Abschreibung liegt zwischen null und den Anschaffungskosten.` };
    }
    if (!Number.isInteger(months) || months < 0) return { error: `${where}: Die Nutzungsdauer steht in Monaten.` };
    assets.push({
      name: d.name.trim(),
      account: d.account.trim(),
      acquisitionDate: d.acquisitionDate,
      acquisitionCost: cost,
      accumulatedDepreciation: accumulated,
      usefulLifeMonths: months || undefined,
      method: d.method || undefined,
    });
  }

  return {
    request: {
      receiptId: draft.receiptId,
      legacySystem: draft.legacySystem.trim() || undefined,
      accounts,
      receivables,
      payables,
      assets,
    },
  };
}

/** Aktiva und Passiva der erfassten Bestände, für die Summenzeile des Dialogs. */
export function takeoverTotals(accounts: TakeoverAccountDraft[], parse: ParseCents): { debit: number; credit: number } {
  let debit = 0;
  let credit = 0;
  for (const d of accounts) {
    const amount = positive(d.amount, parse) ?? 0;
    if (d.side === 'H') credit += amount;
    else debit += amount;
  }
  return { debit, credit };
}
