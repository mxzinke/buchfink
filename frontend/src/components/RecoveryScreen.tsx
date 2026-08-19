import React, { useState } from 'react';
import { KeyRound, AlertTriangle, ShieldCheck, Loader2 } from 'lucide-react';
import { Api } from '../services/api';
import { TenantConfig } from '../types';

interface RecoveryScreenProps {
  activeTenant: TenantConfig | null;
  onRecovered: () => void;
}

/**
 * Shown when the active tenant is encrypted but its OS-keychain secret is
 * missing on this machine (new computer, lost keychain, migrated data). The user
 * unlocks the data with the recovery file they exported and stored externally.
 */
export const RecoveryScreen: React.FC<RecoveryScreenProps> = ({ activeTenant, onRecovered }) => {
  const [isRecovering, setIsRecovering] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleRecover = async () => {
    setError(null);
    try {
      const path = await Api.selectRecoveryFile();
      if (!path) return; // user cancelled the picker
      setIsRecovering(true);
      await Api.recoverFromFile(path);
      onRecovered();
    } catch (e: any) {
      const msg = typeof e?.message === 'string' ? e.message : String(e);
      setError(
        msg.includes('recovery key does not match')
          ? 'Diese Recovery-Datei passt nicht zu diesem Mandanten.'
          : msg.includes('not a Buchfink recovery file')
          ? 'Die gewählte Datei ist keine gültige Buchfink-Recovery-Datei.'
          : msg || 'Wiederherstellung fehlgeschlagen.'
      );
    } finally {
      setIsRecovering(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-stone-900 text-stone-100 p-6 font-sans">
      <div className="max-w-lg w-full bg-[#24211E]/90 border border-white/15 rounded-2xl p-8 shadow-2xl space-y-5">
        <div className="flex items-center gap-3">
          <div className="w-11 h-11 rounded-xl bg-amber-500/15 border border-amber-500/30 flex items-center justify-center shrink-0">
            <KeyRound className="w-5 h-5 text-amber-300" />
          </div>
          <div>
            <h1 className="text-lg font-bold text-white">Wiederherstellung erforderlich</h1>
            <p className="text-xs text-stone-300 mt-0.5">
              Mandant: <span className="font-semibold">{activeTenant?.name || 'Unbekannt'}</span>
            </p>
          </div>
        </div>

        <div className="p-4 bg-[#1D1B19]/70 rounded-xl border border-white/10 text-xs text-stone-300 leading-relaxed space-y-2">
          <p>
            Die Daten dieses Mandanten sind verschlüsselt, aber der Zugriffsschlüssel wurde im
            Schlüsselbund dieses Rechners nicht gefunden – etwa, weil die Daten auf einen neuen
            Computer übertragen wurden.
          </p>
          <p>
            Bitte wählen Sie die <strong>Recovery-Schlüsseldatei</strong>, die Sie zuvor extern
            gesichert haben (z. B. auf einem USB-Stick). Danach wird der Zugriff auf diesem Rechner
            automatisch wiederhergestellt.
          </p>
        </div>

        {error && (
          <div className="p-3 bg-rose-500/10 border border-rose-500/30 rounded-xl text-rose-300 text-xs flex items-start gap-2">
            <AlertTriangle className="w-4 h-4 text-rose-400 shrink-0 mt-0.5" />
            <span>{error}</span>
          </div>
        )}

        <button
          type="button"
          onClick={handleRecover}
          disabled={isRecovering}
          className="w-full inline-flex items-center justify-center gap-2 px-5 py-3 rounded-xl bg-amber-600 hover:bg-amber-500 text-white font-semibold text-sm shadow-md transition-colors disabled:opacity-70"
        >
          {isRecovering ? (
            <>
              <Loader2 className="w-4 h-4 animate-spin" /> Wird wiederhergestellt...
            </>
          ) : (
            <>
              <ShieldCheck className="w-4 h-4" /> Recovery-Schlüsseldatei auswählen...
            </>
          )}
        </button>

        <p className="text-[11px] text-stone-400 leading-relaxed">
          Keine Recovery-Datei? Ohne sie können die verschlüsselten Inhalte auf diesem Rechner
          nicht entschlüsselt werden. Prüfen Sie Ihre externen Sicherungen (USB-Stick, Tresor,
          Passwortmanager).
        </p>
      </div>
    </div>
  );
};
