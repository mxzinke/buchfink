import React, { useState } from 'react';
import { Lock, Unlock } from 'lucide-react';
import { Api } from '../../../services/api';
import type { AppConfig } from '../../../types';
import { Button, Field, FieldRow, Input, Notice, toast } from '../../../components/ui';

function message(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

/**
 * Der Prüfermodus für die Betriebsprüfung: niemand kann buchen oder etwas
 * ändern, auch keine verbundene App. Auswerten und herausgeben geht weiter.
 * Er endet an dem Tag, der angegeben ist, von selbst.
 */
export const ExaminerPanel: React.FC<{
  appConfig: AppConfig;
  onAppConfigChange: (config: AppConfig) => void;
}> = ({ appConfig, onAppConfigChange }) => {
  const [until, setUntil] = useState('');
  const [reason, setReason] = useState('');
  const [endReason, setEndReason] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  async function enable() {
    setBusy(true);
    setError('');
    try {
      onAppConfigChange(await Api.enableReadOnly(until, reason));
      setUntil('');
      setReason('');
      toast.success('Prüfermodus eingeschaltet.');
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  async function disable() {
    setBusy(true);
    setError('');
    try {
      onAppConfigChange(await Api.disableReadOnly(endReason));
      setEndReason('');
      toast.success('Prüfermodus beendet.');
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="max-w-3xl">
      {error && <Notice tone="negative" className="mb-4" text={error} />}
      {appConfig.readOnly ? (
        <FieldRow>
          <Field label="Grund für das Beenden" hint="Steht im Änderungsprotokoll" className="flex-1 min-w-64">
            <Input
              value={endReason}
              placeholder="Prüfung abgeschlossen"
              onChange={(e) => setEndReason(e.target.value)}
            />
          </Field>
          <Button
            variant="secondary"
            loading={busy}
            className="mt-6"
            icon={<Unlock className="w-4 h-4" strokeWidth={1.5} />}
            onClick={() => void disable()}
          >
            Prüfermodus beenden
          </Button>
        </FieldRow>
      ) : (
        <FieldRow>
          <Field label="Gilt bis einschließlich" className="w-52">
            <Input type="date" value={until} onChange={(e) => setUntil(e.target.value)} />
          </Field>
          <Field label="Grund" hint="Etwa: Außenprüfung 2023 bis 2025" className="flex-1 min-w-64">
            <Input value={reason} placeholder="Außenprüfung" onChange={(e) => setReason(e.target.value)} />
          </Field>
          <Button
            variant="secondary"
            loading={busy}
            className="mt-6"
            icon={<Lock className="w-4 h-4" strokeWidth={1.5} />}
            onClick={() => void enable()}
          >
            Prüfermodus einschalten
          </Button>
        </FieldRow>
      )}
    </div>
  );
};
