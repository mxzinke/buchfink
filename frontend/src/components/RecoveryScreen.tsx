import React, { useState } from 'react';
import { AlertTriangle, KeyRound, ShieldCheck } from 'lucide-react';
import { Api } from '../services/api';
import { TenantConfig } from '../types';
import { SHELL_BUTTON, SHELL_PANEL, cn } from './ui';

interface RecoveryScreenProps {
  activeTenant: TenantConfig | null;
  onRecovered: () => void;
}

/**
 * Erscheint, wenn die Daten des Mandanten verschlüsselt sind, der Schlüssel im
 * Schlüsselbund dieses Rechners aber fehlt — neuer Computer, verlorener
 * Schlüsselbund, übertragene Daten. Entsperrt wird mit der Recovery-Datei, die
 * getrennt vom Datenbackup liegen sollte.
 *
 * Der Schirm gehört zur Schale (§16) und trägt genau eine Aktion.
 */
export const RecoveryScreen: React.FC<RecoveryScreenProps> = ({ activeTenant, onRecovered }) => {
  const [recovering, setRecovering] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function recover() {
    setError(null);
    try {
      const path = await Api.selectRecoveryFile('Recovery-Schlüsseldatei auswählen');
      if (!path) return; // Auswahl abgebrochen.
      setRecovering(true);
      await Api.recoverFromFile(path);
      onRecovered();
    } catch (e) {
      const message = e instanceof Error ? e.message : String(e);
      setError(
        message.includes('recovery key does not match')
          ? 'Diese Recovery-Datei gehört zu einem anderen Mandanten.'
          : message.includes('not a Buchfink recovery file')
            ? 'Die gewählte Datei ist keine Buchfink-Recovery-Datei.'
            : message || 'Die Wiederherstellung ist fehlgeschlagen.',
      );
    } finally {
      setRecovering(false);
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center p-6 bg-shell-deep text-shell-text">
      <div className={cn(SHELL_PANEL, 'w-full max-w-lg p-8')}>
        <div className="flex items-center gap-3">
          <span className="grid place-items-center w-10 h-10 shrink-0 rounded-control border border-shell-line bg-shell-raised text-accent-light">
            <KeyRound className="w-5 h-5" strokeWidth={1.5} />
          </span>
          <span className="min-w-0">
            <h1 className="text-heading text-white">Wiederherstellung nötig</h1>
            <p className="text-caption text-shell-text-muted mt-0.5 truncate">
              Mandant {activeTenant?.name || 'unbekannt'}
            </p>
          </span>
        </div>

        <p className="text-body text-shell-text-muted mt-5">
          Die Daten sind verschlüsselt, der Zugriffsschlüssel liegt aber nicht im Schlüsselbund
          dieses Rechners. Mit der extern gesicherten Recovery-Datei wird der Zugriff hier wieder
          eingerichtet.
        </p>

        {error && (
          <p className="mt-4 flex items-start gap-2 rounded-control border border-negative/50 bg-negative/15 px-4 py-3 text-body text-shell-negative">
            <AlertTriangle className="w-4 h-4 mt-0.5 shrink-0" strokeWidth={1.5} />
            {error}
          </p>
        )}

        <button
          type="button"
          onClick={() => void recover()}
          disabled={recovering}
          className={cn(SHELL_BUTTON.primary, 'w-full mt-5')}
        >
          <ShieldCheck className="w-4 h-4" strokeWidth={1.5} />
          {recovering ? 'Wird wiederhergestellt …' : 'Recovery-Datei auswählen'}
        </button>

        <p className="text-caption text-shell-text-muted mt-4">
          Ohne diese Datei lassen sich die verschlüsselten Inhalte auf diesem Rechner nicht
          entschlüsseln. Sie liegt dort, wo sie getrennt vom Datenbackup gesichert wurde.
        </p>
      </div>
    </div>
  );
};
