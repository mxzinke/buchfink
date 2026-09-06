import React, { createContext, useContext, useMemo } from 'react';
import { formatDate } from '../utils/formatters';

/**
 * Der Prüfermodus in der Oberfläche.
 *
 * Die Bridge weist im Prüfermodus jede schreibende Bedienung ab. Ein Knopf, der
 * bis dahin aktiv bleibt, verspricht etwas, das die Anwendung nicht hält: der
 * Anwender füllt ein Formular aus und erfährt den Grund erst in der
 * Fehlermeldung des Absendens. Deshalb sind die schreibenden Knöpfe schon in
 * der Oberfläche gesperrt und zeigen die Erklärung im `title` (§10.4).
 *
 * Ein Kontext statt einer Kette von Eigenschaften: schreibende Knöpfe stehen
 * bis in die Formulardialoge der Anlagenverwaltung hinein, und jede
 * Zwischenstation, die die Angabe nur durchreicht, wäre eine Stelle, an der sie
 * beim nächsten Dialog vergessen wird. Die Sperre gilt der ganzen Anwendung und
 * nicht einer Ansicht — das bildet der Kontext ab.
 */

export interface WriteLock {
  /** Gilt der Prüfermodus gerade? */
  locked: boolean;
  /**
   * Der Grund für den `title` eines gesperrten Knopfes. Leer, solange nichts
   * gesperrt ist — ein `title` an einem bedienbaren Knopf wäre nur Rauschen.
   */
  hint?: string;
}

const OPEN: WriteLock = { locked: false, hint: undefined };

const WriteLockContext = createContext<WriteLock>(OPEN);

export interface WriteLockProviderProps {
  /** Der Zustand aus `GetAppConfig`. */
  readOnly: boolean;
  /** Tag, bis zu dem der Modus gilt (einschließlich). */
  until?: string;
  /** Der Grund, mit dem der Modus eingeschaltet wurde. */
  reason?: string;
  children: React.ReactNode;
}

export const WriteLockProvider: React.FC<WriteLockProviderProps> = ({
  readOnly,
  until,
  reason,
  children,
}) => {
  const value = useMemo<WriteLock>(() => {
    if (!readOnly) return OPEN;
    const period = until ? ` bis ${formatDate(until)}` : '';
    const because = reason ? ` · ${reason}` : '';
    return {
      locked: true,
      hint: `Prüfermodus${period}: die Buchführung nimmt keine Änderung auf${because}`,
    };
  }, [readOnly, until, reason]);

  return <WriteLockContext.Provider value={value}>{children}</WriteLockContext.Provider>;
};

/**
 * Der Zustand für eine schreibende Bedienung.
 *
 * Muster: `disabled={locked || busy}` und `title={hint}`.
 */
export function useWriteLock(): WriteLock {
  return useContext(WriteLockContext);
}

/**
 * Die zweite Sperre mit demselben Ergebnis: das festgestellte oder offengelegte
 * Geschäftsjahr (§11.5).
 *
 * Sie steht neben dem Prüfermodus und nicht in ihm, weil sie einen anderen
 * Geltungsbereich hat: der Prüfermodus sperrt jede Änderung der Anwendung, das
 * festgestellte Jahr nur die Erfassung in diesem Jahr. Getrennt gehalten,
 * gemeinsam gelesen — die Buchungsansichten fragen über `usePostingLock` beide
 * zugleich ab und zeigen den Grund im `title` des gesperrten Knopfes (§10.4).
 */
const PostingLockContext = createContext<number | undefined>(undefined);

export interface PostingLockProviderProps {
  /** Das angezeigte Geschäftsjahr, sofern es festgestellt oder offengelegt ist. */
  closedYear?: number;
  children: React.ReactNode;
}

export const PostingLockProvider: React.FC<PostingLockProviderProps> = ({
  closedYear,
  children,
}) => (
  <PostingLockContext.Provider value={closedYear}>{children}</PostingLockContext.Provider>
);

/**
 * Der Zustand für eine erfassende Bedienung in einer Buchungsansicht.
 *
 * Gleiche Form wie `useWriteLock`, damit die Ansichten dasselbe Muster
 * behalten: `disabled={locked || busy}` und `title={hint}`. Der Prüfermodus
 * geht vor, weil er die weitere Sperre ist.
 */
export function usePostingLock(): WriteLock {
  const writeLock = useContext(WriteLockContext);
  const closedYear = useContext(PostingLockContext);
  return useMemo<WriteLock>(() => {
    if (writeLock.locked) return writeLock;
    if (closedYear === undefined) return OPEN;
    return {
      locked: true,
      hint:
        `Das Geschäftsjahr ${closedYear} ist festgestellt. Buchungen nimmt es erst wieder an, ` +
        `wenn die Feststellung mit Grund zurückgesetzt wird.`,
    };
  }, [writeLock, closedYear]);
}
