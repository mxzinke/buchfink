import React, { useState } from 'react';
import { ArrowUpCircle, ExternalLink } from 'lucide-react';
import { Api } from '../services/api';
import type { UpdateStatus } from '../types';
import { Updates, useUpdateStatus } from '../update';
import { formatDate } from '../utils/formatters';
import { Button, Dialog, Notice, Progress } from './ui';

function message(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

/** Der Hinweis erscheint, sobald eine neuere Fassung gefunden oder installiert ist. */
function hasNews(status: UpdateStatus | null): status is UpdateStatus {
  return !!status?.release && status.phase !== 'idle' && status.phase !== 'checking';
}

/**
 * Hinweis auf eine neue Programmfassung in der Seitenleiste. Installiert wird
 * erst auf einen Klick im Dialog: die Fassung steht an jeder Buchung, ihr
 * Wechsel ist eine Entscheidung des Menschen.
 */
export const UpdateNotice: React.FC = () => {
  const status = useUpdateStatus();
  const [open, setOpen] = useState(false);
  const [restartError, setRestartError] = useState('');

  if (!hasNews(status)) return null;
  const { release, releaseNotes, phase } = status;
  const version = release?.version ?? '';
  const installed = phase === 'installed';
  const downloading = phase === 'downloading';

  async function restart() {
    setRestartError('');
    try {
      await Updates.restart();
    } catch (error) {
      setRestartError(message(error));
    }
  }

  async function installAndRestart() {
    setRestartError('');
    try {
      await Updates.install();
    } catch {
      // Den Grund zeigt der Dialog aus dem Stand (status.error).
      return;
    }
    await restart();
  }

  const footer = installed ? (
    <Button variant="primary" onClick={() => void restart()}>
      Jetzt neu starten
    </Button>
  ) : status.manualReason ? (
    <Button
      variant="primary"
      icon={<ExternalLink className="w-4 h-4" strokeWidth={1.5} />}
      onClick={() => void Api.openReleasesPage().catch((error) => setRestartError(message(error)))}
    >
      Fassungen im Netz
    </Button>
  ) : (
    <>
      <Button variant="secondary" onClick={() => setOpen(false)} disabled={downloading}>
        Später
      </Button>
      <Button variant="primary" loading={downloading} onClick={() => void installAndRestart()}>
        Installieren und neu starten
      </Button>
    </>
  );

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="w-full mb-1 flex items-start gap-2.5 px-2 py-1.5 rounded-control text-left
                   transition-colors duration-120 ease-quiet hover:bg-shell-raised"
      >
        <ArrowUpCircle className="mt-0.5 w-3.5 h-3.5 shrink-0 text-accent-light" strokeWidth={1.5} />
        <span className="min-w-0">
          <span className="block text-caption text-shell-text">
            Buchfink <span className="num">{version}</span> {installed ? 'ist installiert' : 'ist verfügbar'}
          </span>
          <span className="block text-caption text-shell-text-muted">
            {installed ? 'Läuft nach dem Neustart' : 'Ansehen und installieren'}
          </span>
        </span>
      </button>

      <Dialog open={open} onOpenChange={setOpen} title={`Buchfink ${version}`} width="max-w-2xl" footer={footer}>
        <div className="space-y-4">
          <p className="text-body text-ink-muted">
            {installed
              ? `Buchfink ${version} ist installiert und läuft nach dem Neustart. Bis dahin arbeitet die App mit Fassung ${status.current} weiter.`
              : `Sie arbeiten mit Fassung ${status.current}.${
                  release?.publishedAt ? ` Fassung ${version} ist am ${formatDate(release.publishedAt)} erschienen.` : ''
                }`}
          </p>

          {status.manualReason && !installed && <Notice text={status.manualReason} />}
          {status.error && !downloading && <Notice tone="negative" text={status.error} />}
          {restartError && <Notice tone="negative" text={restartError} />}
          {downloading && (
            <Progress value={status.progress} label="Neue Fassung wird geladen" detail={`${status.progress} %`} />
          )}

          {releaseNotes && (releaseNotes.summary || releaseNotes.changes.length > 0) && (
            <div className="max-h-80 overflow-y-auto pr-2">
              {releaseNotes.summary && <p className="text-body text-ink">{releaseNotes.summary}</p>}
              <ul className="mt-2 space-y-1">
                {releaseNotes.changes.map((change) => (
                  <li key={change} className="flex gap-2.5 text-body text-ink-muted">
                    <span className="mark-diamond bg-line-strong mt-2 shrink-0" />
                    <span>{change}</span>
                  </li>
                ))}
              </ul>
            </div>
          )}

          {!installed && !status.manualReason && (
            <p className="text-caption text-ink-subtle">
              Buchfink lädt die Datei von GitHub, prüft sie gegen die Prüfsumme des Release und startet neu. Beim
              Beenden entsteht wie sonst eine Sicherung, wenn ein Sicherungsordner eingerichtet ist.
            </p>
          )}
        </div>
      </Dialog>
    </>
  );
};
