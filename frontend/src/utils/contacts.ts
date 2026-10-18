import type { Contact, Task } from '../types';

/**
 * Die Standard-Buchungsgruppe eines Lieferanten. Erst eine Gruppe, die der
 * Mensch in der App bestätigt hat, erlaubt Claude, E-Rechnungen des
 * Lieferanten ohne Rückfrage zu buchen; Claude schlägt sie nur vor.
 */

export type PostingGroupState = 'confirmed' | 'proposed' | 'none';

/** Wartet ein Vorschlag von Claude auf die Bestätigung? */
export function hasPendingProposal(contact: Contact): boolean {
  return (
    contact.type === 'vendor' &&
    !contact.blocked &&
    Boolean(contact.proposedPostingGroup) &&
    contact.proposedPostingGroup !== contact.defaultPostingGroup
  );
}

/** Ein offener Vorschlag geht vor: er verlangt eine Entscheidung. */
export function postingGroupState(contact: Contact): PostingGroupState {
  if (hasPendingProposal(contact)) return 'proposed';
  return contact.defaultPostingGroup ? 'confirmed' : 'none';
}

/** Der Schlüssel der Aufgabe in der Übersicht. Sie entsteht hier, nicht im Backend. */
export const POSTING_GROUP_TASK_KEY = 'standardgruppen';

/**
 * Die Aufgabe „Standardgruppen bestätigen“, solange Claude Gruppen
 * vorgeschlagen hat. Bei einem Lieferanten führt sie zu ihm, bei mehreren zur
 * Liste der Kontakte.
 */
export function postingGroupTask(contacts: Contact[]): Task | null {
  const pending = contacts.filter(hasPendingProposal);
  if (pending.length === 0) return null;
  const one = pending.length === 1 ? pending[0] : null;
  return {
    key: POSTING_GROUP_TASK_KEY,
    group: 'open',
    title: one
      ? `Standardgruppe für ${one.company || one.name} bestätigen`
      : `${pending.length} Standardgruppen bestätigen`,
    why: 'Claude hat sie vorgeschlagen. Erst nach Ihrer Bestätigung bucht Claude E-Rechnungen dieser Lieferanten ohne Rückfrage.',
    target: { area: 'company', tab: 'contacts', id: one ? String(one.id) : undefined },
    count: pending.length > 1 ? pending.length : 0,
    amount: 0,
  };
}
