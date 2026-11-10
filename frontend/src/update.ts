import { useEffect, useSyncExternalStore } from 'react';
import { Api } from './services/api';
import type { UpdateStatus } from './types';

/**
 * Der Stand der Aktualisierung, einmal für die ganze Oberfläche: der Hinweis
 * in der Seitenleiste und die Einstellungen zeigen dasselbe, gleich wo gesucht
 * oder installiert wurde.
 */
let status: UpdateStatus | null = null;
const listeners = new Set<() => void>();

function publish(next: UpdateStatus): UpdateStatus {
  status = next;
  listeners.forEach((listener) => listener());
  return next;
}

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => listeners.delete(listener);
}

/** Der Hintergrundlauf sucht täglich; so oft liest die Oberfläche nach. */
const REFRESH_MS = 30 * 60 * 1000;
const DOWNLOAD_REFRESH_MS = 500;

export const Updates = {
  refresh: (): Promise<UpdateStatus> => Api.getUpdateStatus().then(publish),
  check: (): Promise<UpdateStatus> => Api.checkForUpdate().then(publish),
  setAutoCheck: (enabled: boolean): Promise<UpdateStatus> => Api.setUpdateCheck(enabled).then(publish),
  /**
   * Installiert die gefundene Fassung. Der Download meldet seinen Fortschritt
   * nicht von selbst; solange er läuft, liest die Oberfläche den Stand nach.
   */
  install: async (): Promise<UpdateStatus> => {
    const poll = window.setInterval(() => void Updates.refresh().catch(() => undefined), DOWNLOAD_REFRESH_MS);
    try {
      return publish(await Api.installUpdate());
    } catch (error) {
      await Updates.refresh().catch(() => undefined);
      throw error;
    } finally {
      window.clearInterval(poll);
    }
  },
  restart: (): Promise<void> => Api.restartAfterUpdate(),
};

/** Der Stand der Aktualisierung; `null`, solange er nicht gelesen ist. */
export function useUpdateStatus(): UpdateStatus | null {
  const current = useSyncExternalStore(subscribe, () => status);
  useEffect(() => {
    void Updates.refresh().catch(() => undefined);
    const timer = window.setInterval(() => void Updates.refresh().catch(() => undefined), REFRESH_MS);
    return () => window.clearInterval(timer);
  }, []);
  return current;
}
