import React, { createContext, useContext, useEffect, useRef, useState } from 'react';
import { Events } from '@wailsio/runtime';
import type { ChangeEvent, ChangeKind } from './types';
import { concerns, forTenant, reloadDelay } from './utils/changes';

/**
 * Live-Aktualisierung: Was ein Agent, ein Hintergrundlauf oder die App selbst
 * im geöffneten Mandanten ändert, meldet das Backend als Ereignis. Die
 * Übersicht und die offene Ansicht laden dann neu, ohne Skelett und ohne
 * offene Formulare oder Dialoge anzutasten: jede Ansicht lädt über
 * `useLiveReload` dieselben Daten wie beim Öffnen und ersetzt nur ihren Stand.
 */

/** Nach jeder Änderung in einem Mandanten (internal/wailsbridge/changes.go). */
export const EVENT_CHANGED = 'buchfink:geaendert';

/** Nach Dateien, die ins Fenster gezogen wurden (internal/wailsbridge/inbox_service.go). */
export const EVENT_INBOX_ADDED = 'buchfink:eingang-abgelegt';

/** Ruhe nach dem letzten Ereignis, bevor neu geladen wird. */
const SETTLE_MS = 400;
/** Spätestens so lange nach dem ersten Ereignis einer Folge wird neu geladen. */
const MAX_WAIT_MS = 1500;

interface Changes {
  /** Zählt die Bündel von Änderungen im geöffneten Mandanten. */
  revision: number;
  /** Die Arten des letzten Bündels. */
  kinds: ChangeKind[];
}

const ChangesContext = createContext<Changes>({ revision: 0, kinds: [] });

/**
 * Hört auf die Ereignisse des Backends und fasst die des geöffneten Mandanten
 * zu Bündeln zusammen. Ereignisse anderer Mandanten bleiben ohne Wirkung.
 */
export const ChangesProvider: React.FC<{ tenantId: string | undefined; children: React.ReactNode }> = ({
  tenantId,
  children,
}) => {
  const [changes, setChanges] = useState<Changes>({ revision: 0, kinds: [] });

  useEffect(() => {
    if (!tenantId) return;
    let pending = new Set<ChangeKind>();
    let firstAt = 0;
    let timer: number | undefined;
    const flush = () => {
      const kinds = [...pending];
      pending = new Set();
      firstAt = 0;
      timer = undefined;
      setChanges((prev) => ({ revision: prev.revision + 1, kinds }));
    };
    const off = Events.On(EVENT_CHANGED, (event) => {
      const change = event.data as ChangeEvent;
      if (!forTenant(change, tenantId)) return;
      for (const kind of change.kinds ?? []) pending.add(kind);
      const now = Date.now();
      if (!firstAt) firstAt = now;
      window.clearTimeout(timer);
      timer = window.setTimeout(flush, reloadDelay(firstAt, now, SETTLE_MS, MAX_WAIT_MS));
    });
    return () => {
      off();
      window.clearTimeout(timer);
    };
  }, [tenantId]);

  return <ChangesContext.Provider value={changes}>{children}</ChangesContext.Provider>;
};

/**
 * Lädt eine Ansicht neu, sobald sich im geöffneten Mandanten etwas geändert
 * hat, auf Wunsch nur bei bestimmten Arten. Beim ersten Rendern lädt die
 * Ansicht selbst; der Hook ruft `reload` erst beim nächsten Bündel. `reload`
 * darf kein Skelett zeigen: die Ansicht behält ihren Stand, bis der neue da
 * ist.
 */
export function useLiveReload(reload: () => unknown, kinds?: readonly ChangeKind[]): void {
  const changes = useContext(ChangesContext);
  const seen = useRef(changes.revision);
  const latest = useRef({ reload, kinds });
  latest.current = { reload, kinds };

  useEffect(() => {
    if (changes.revision === seen.current) return;
    seen.current = changes.revision;
    if (!concerns(changes.kinds, latest.current.kinds)) return;
    void latest.current.reload();
  }, [changes]);
}
