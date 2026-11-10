import type { BankTransaction, BankSuggestion } from '../types';
import { formatCents } from './formatters';

/** Was vom Umsatz noch keinem Posten zugeordnet ist. */
const remainingOf = (tx: BankTransaction) => Math.max(0, Math.abs(tx.amount) - (tx.matchedAmount || 0));

/**
 * Was ein Vorschlag beim Bestätigen bucht, oder warum er sich so nicht
 * bestätigen lässt. Ein Vorschlag aus offenen Posten muss den Umsatz genau
 * decken; bei Teilzahlungen bleibt der Rest offen.
 */
export function bankMatchPlan(
  tx: BankTransaction,
  suggestion: BankSuggestion,
):
  | { kind: 'direct'; counterAccount: string }
  | { kind: 'advance' | 'refund'; advanceId: number }
  | { kind: 'settle'; allocations: { openItemEntryId: number; settledAmount: number }[] }
  | { kind: 'blocked'; reason: string } {
  if (suggestion.kind === 'rule' || suggestion.kind === 'credit_card') {
    return suggestion.counterAccount
      ? { kind: 'direct', counterAccount: suggestion.counterAccount }
      : { kind: 'blocked', reason: 'Dem Vorschlag fehlt das Gegenkonto.' };
  }
  const items = suggestion.items ?? [];
  const advance = items.find((item) => item.source === 'advance' || item.source === 'advance_refund');
  if (advance) {
    return items.length === 1 && Math.abs(tx.amount) <= advance.openAmount
      ? { kind: advance.source === 'advance_refund' ? 'refund' : 'advance', advanceId: advance.advanceInvoiceId ?? 0 }
      : { kind: 'blocked', reason: 'Diese Aufteilung ordnet Claude nach Rückfrage zu.' };
  }
  // Einen Fremdwährungsposten gleicht die Zahlung ganz aus; den Unterschied zum
  // gebuchten Euro-Betrag bucht das Backend als Kursdifferenz.
  if (items.length === 1 && items[0].currency) {
    return { kind: 'settle', allocations: [{ openItemEntryId: items[0].entryId, settledAmount: items[0].openAmount }] };
  }
  let remaining = remainingOf(tx);
  const allocations = items.map((item) => {
    const amount = Math.min(Math.abs(item.openAmount), remaining);
    remaining -= amount;
    return { openItemEntryId: item.entryId, settledAmount: amount };
  });
  if (remaining !== 0 || allocations.length === 0) {
    return {
      kind: 'blocked',
      reason: `Die Posten decken ${formatCents(remainingOf(tx) - remaining, tx.currency)} von ${formatCents(
        remainingOf(tx),
        tx.currency,
      )}. Den Rest ordnet Claude zu.`,
    };
  }
  return { kind: 'settle', allocations: allocations.filter((a) => a.settledAmount > 0) };
}
