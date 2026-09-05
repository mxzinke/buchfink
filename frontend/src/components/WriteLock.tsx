import React, { createContext, useContext, useMemo } from 'react';
import { formatDate } from '../utils/formatters';

/**
 * Der Prüfermodus in der Oberfläche.
 *
 * Die Bridge weist im Prüfermodus jede schreibende Bedienung ab. Ein Knopf, der
 * bis dahin aktiv bleibt, verspricht etwas, das die Anwendung nicht hält: der
 * Anwender füllt ein Formular aus und erfährt den Grund erst in der
 * Fehlermeldung des Absendens. Deshalb sind die schreibenden Knöpfe schon in
 * der Oberfläche gesperrt und tragen die Erklärung im `title` (§10.4).
 *
 * Ein Kontext statt einer Kette von Eigenschaften: schreibende Knöpfe stehen
 * bis in die Formulardialoge der Anlagenverwaltung hinein, und jede
 * Zwischenstation, die die Angabe nur durchreicht, wäre eine Stelle, an der sie
 * beim nächsten Dialog vergessen wird. Die Sperre gilt der ganzen Anwendung und
 * nicht einer Ansicht — genau das bildet der Kontext ab.
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
