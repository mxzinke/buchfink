import { useEffect, useState } from 'react';
import { Events } from '@wailsio/runtime';
import { Api } from '../services/api';
import type { AppConsentRequest } from '../types';
import { formatDateOfTimestamp, formatTime } from '../utils/formatters';
import { Button, DetailList, Dialog, Help, Progress, toast } from './ui';

/** Ereignisse aus internal/wailsbridge/mcp_service.go. */
const EVENT_REQUEST = 'buchfink:app-anfrage';
const EVENT_DONE = 'buchfink:app-anfrage-erledigt';

/** „Zulassen" wird erst nach dieser Zeit klickbar, gegen untergeschobene Klicks. */
const ARM_DELAY_MS = 2000;

/** Was ein Recht in der Sprache des Anwenders bedeutet. */
const SCOPE_RIGHTS: Record<string, string> = {
  'buchfink:lesen': 'Bücher, Belege und Auswertungen lesen',
  'buchfink:buchen': 'Routinefälle buchen, Rückfragen und Stammdaten für Sie vorbereiten',
};

/** Der Abgleichcode in Vierergruppen, damit er sich laut vergleichen lässt. */
function groupCode(code: string): string {
  const plain = code.replace(/[\s-]/g, '');
  return plain.length > 4 ? plain.replace(/(.{4})(?=.)/g, '$1 ') : code;
}

/**
 * Eine App bittet um Zugriff auf Buchfink. Zugestimmt wird nur hier, im
 * Fenster der App; der Browser zeigt denselben Abgleichcode und wartet.
 * Schließen heißt ablehnen.
 */
export function AppConsentDialog() {
  const [requests, setRequests] = useState<AppConsentRequest[]>([]);
  // Die Anfrage, für die „Zulassen" scharf ist. An die Kennung gebunden,
  // damit eine nachrückende Anfrage nicht schon im ersten Bild klickbar ist.
  const [armedFor, setArmedFor] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [progress, setProgress] = useState(0);

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
  const currentId = current?.id;
  const armed = currentId !== undefined && armedFor === currentId;

  // Die Wartezeit läuft sichtbar ab: ein gesperrter Knopf ohne Grund sähe aus
  // wie ein Fehler (§10.4).
  useEffect(() => {
    if (currentId === undefined) return;
    const started = Date.now();
    setProgress(0);
    const tick = window.setInterval(() => {
      setProgress(Math.min(100, ((Date.now() - started) / ARM_DELAY_MS) * 100));
    }, 100);
    const timer = window.setTimeout(() => {
      window.clearInterval(tick);
      setProgress(100);
      setArmedFor(currentId);
    }, ARM_DELAY_MS);
    return () => {
      window.clearTimeout(timer);
      window.clearInterval(tick);
    };
  }, [currentId]);

  const decide = async (approve: boolean) => {
    if (!current || busy || (approve && !armed)) return;
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

  const rights = current.scopes.map((scope) => SCOPE_RIGHTS[scope] ?? scope);
  const expires = current.expiresAt ? formatTime(current.expiresAt) : '';
  const waiting = requests.length - 1;

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
          <Button
            variant="primary"
            disabled={!armed || busy}
            title={armed ? undefined : 'Vergleichen Sie zuerst den Code mit dem im Browser.'}
            onClick={() => void decide(true)}
          >
            Zulassen
          </Button>
        </>
      }
    >
      <p className="text-heading text-ink break-words">{current.clientName}</p>
      <p className="text-caption text-ink-subtle mt-0.5">
        {current.knownSince
          ? `Name laut App · zugelassen seit ${formatDateOfTimestamp(current.knownSince)}`
          : 'Name laut App · zum ersten Mal hier'}
      </p>

      <div className="mt-6">
        <p className="flex items-center text-label text-ink-muted">
          Abgleichcode
          <Help
            summary="Derselbe Code steht im Browser. Stimmen beide überein, kommt die Anfrage von Ihrer Anmeldung."
            label="Erklärung zum Abgleichcode"
          >
            Jedes Programm auf diesem Rechner kann eine Anmeldung beginnen. Der Code verbindet diese
            Anfrage mit dem Browserfenster, das Ihre App geöffnet hat. Lassen Sie nur zu, wenn Sie die
            Anmeldung gerade selbst gestartet haben und beide Codes gleich sind.
          </Help>
        </p>
        <p className="mt-1 code-num text-display tracking-[0.2em] text-ink" aria-live="polite">
          {groupCode(current.userCode)}
        </p>
      </div>

      <DetailList
        className="mt-6"
        items={[
          {
            label: 'Unternehmen',
            value: current.tenantName || 'Kein Unternehmen geöffnet',
          },
          {
            label: 'Rücksprung',
            value: <span className="code-num break-all">{current.redirectUri}</span>,
          },
          {
            label: 'Die App darf',
            value: (
              <ul className="flex flex-col gap-1">
                {rights.map((right) => (
                  <li key={right}>{right}</li>
                ))}
              </ul>
            ),
          },
          {
            label: 'Bleibt bei Ihnen',
            value: 'Festschreiben, Übermitteln, Stammdaten übernehmen, Prüfermodus',
          },
        ]}
      />

      <div className="mt-6">
        {armed ? (
          <p className="text-caption text-ink-subtle">
            {expires ? `Die Anfrage verfällt um ${expires}.` : 'Die Anfrage verfällt nach fünf Minuten.'}
            {waiting > 0 && ` Danach ${waiting === 1 ? 'wartet eine weitere' : `warten ${waiting} weitere`}.`}
          </p>
        ) : (
          <Progress value={progress} label="Code vergleichen" />
        )}
      </div>
    </Dialog>
  );
}
