import { useEffect, useState } from 'react';
import { Events } from '@wailsio/runtime';
import { Api } from '../services/api';
import type { AppConsentRequest } from '../types';
import { Button, Dialog, toast } from './ui';

/** Ereignisse aus internal/wailsbridge/mcp_service.go. */
const EVENT_REQUEST = 'buchfink:app-anfrage';
const EVENT_DONE = 'buchfink:app-anfrage-erledigt';

/** „Zulassen" wird erst nach dieser Zeit klickbar, gegen untergeschobene Klicks. */
const ARM_DELAY_MS = 1500;

const SCOPE_LABELS: Record<string, string> = {
  'buchfink:lesen': 'Lesen',
  'buchfink:buchen': 'Buchen',
};

/**
 * Eine App bittet um Zugriff auf Buchfink. Zugestimmt wird nur hier, im
 * Fenster der App; der Browser zeigt denselben Abgleichcode und wartet.
 * Schließen heißt ablehnen.
 */
export function AppConsentDialog() {
  const [requests, setRequests] = useState<AppConsentRequest[]>([]);
  const [armed, setArmed] = useState(false);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let active = true;
    Api.getPendingAppRequests()
      .then((pending) => active && setRequests(pending))
      .catch(() => undefined);
    const offRequest = Events.On(EVENT_REQUEST, (event) => {
      const request = event.data as AppConsentRequest;
      setRequests((prev) => [
        ...prev.filter((r) => r.id !== request.id),
        { ...request, scopes: request.scopes ?? [] },
      ]);
    });
    const offDone = Events.On(EVENT_DONE, (event) => {
      const id = event.data as string;
      setRequests((prev) => prev.filter((r) => r.id !== id));
    });
    return () => {
      active = false;
      offRequest();
      offDone();
    };
  }, []);

  const current = requests[0];

  useEffect(() => {
    setArmed(false);
    if (!current) return;
    const timer = window.setTimeout(() => setArmed(true), ARM_DELAY_MS);
    return () => window.clearTimeout(timer);
  }, [current?.id]);

  const decide = async (approve: boolean) => {
    if (!current || busy) return;
    setBusy(true);
    try {
      await Api.resolveAppRequest(current.id, current.nonce, approve);
      if (approve) toast.success(`„${current.clientName}“ ist zugelassen.`);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setRequests((prev) => prev.filter((r) => r.id !== current.id));
      setBusy(false);
    }
  };

  if (!current) return null;

  const rights = current.scopes.map((s) => SCOPE_LABELS[s] ?? s).join(' und ');

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) void decide(false);
      }}
      title="Eine App möchte auf Buchfink zugreifen"
      width="max-w-lg"
      footer={
        <>
          <Button variant="secondary" disabled={busy} onClick={() => void decide(false)}>
            Ablehnen
          </Button>
          <Button disabled={!armed || busy} onClick={() => void decide(true)}>
            Zulassen
          </Button>
        </>
      }
    >
      <dl className="grid grid-cols-[auto_1fr] gap-x-6 gap-y-2 text-body">
        <dt className="text-ink-muted">App</dt>
        <dd className="text-ink">
          {current.clientName} <span className="text-ink-muted">(Angabe der App)</span>
        </dd>
        <dt className="text-ink-muted">Abgleichcode</dt>
        <dd className="text-ink font-semibold tabular-nums tracking-widest">{current.userCode}</dd>
        <dt className="text-ink-muted">Rücksprung</dt>
        <dd className="text-ink break-all">{current.redirectUri}</dd>
        <dt className="text-ink-muted">Mandant</dt>
        <dd className="text-ink">{current.tenantName || 'kein Mandant geöffnet'}</dd>
        <dt className="text-ink-muted">Rechte</dt>
        <dd className="text-ink">{rights}</dd>
      </dl>
      <p className="text-body text-ink-muted mt-4">
        Bitte nur zulassen, wenn der Code mit dem im Browser übereinstimmt und Sie die Anmeldung
        selbst begonnen haben. Jede Buchung der App trägt ihren Erfassungsweg.
      </p>
    </Dialog>
  );
}
