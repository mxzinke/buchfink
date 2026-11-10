import React, { useState } from 'react';
import { RefreshCw } from 'lucide-react';
import { Updates, useUpdateStatus } from '../../update';
import { formatDateTime } from '../../utils/formatters';
import { Button, Field, FieldValue, Notice, Switch } from '../../components/ui';

function message(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

/**
 * Programmfassung mit der Suche nach einer neueren. Installiert wird über den
 * Hinweis in der Seitenleiste, der nach einem Fund erscheint.
 */
export const UpdateSettings: React.FC<{ programVersion: string }> = ({ programVersion }) => {
  const status = useUpdateStatus();
  const [checking, setChecking] = useState(false);
  const [error, setError] = useState('');

  async function run(action: () => Promise<unknown>) {
    setError('');
    try {
      await action();
    } catch (e) {
      setError(message(e));
    }
  }

  async function check() {
    setChecking(true);
    await run(Updates.check);
    setChecking(false);
  }

  const version = programVersion || 'dev';
  if (!status?.supported) {
    return (
      <Field label="Programmfassung" hint={status ? 'Arbeitsstand, aktualisiert sich nicht' : undefined}>
        <FieldValue className="code-num">{version}</FieldValue>
      </Field>
    );
  }

  const found = status.release && status.phase !== 'idle';
  const result = found
    ? status.phase === 'installed'
      ? `Fassung ${status.release?.version} läuft nach dem Neustart`
      : `Fassung ${status.release?.version} ist verfügbar`
    : status.checkedAt
      ? `Aktuell, gesucht ${formatDateTime(status.checkedAt)}`
      : 'Noch nicht gesucht';

  return (
    <>
      <Field
        label="Programmfassung"
        hint={result}
        helpSummary="Eine neue Fassung installieren Sie über den Hinweis unten in der Seitenleiste."
      >
        <div className="flex flex-wrap items-center gap-3">
          <FieldValue className="code-num">{version}</FieldValue>
          <Button
            variant="secondary"
            size="sm"
            loading={checking}
            icon={<RefreshCw className="w-3.5 h-3.5" strokeWidth={1.5} />}
            onClick={() => void check()}
          >
            Nach neuer Fassung suchen
          </Button>
        </div>
      </Field>
      {(error || (status.error && !found)) && (
        <Notice tone="negative" text={error || `Die Suche ist gescheitert: ${status.error}`} />
      )}
      <Field
        label="Selbst nach neuen Fassungen suchen"
        hint="Beim Start und danach täglich"
        helpSummary="Buchfink fragt GitHub nach dem neuesten Release und überträgt dabei die Programmfassung."
      >
        <Switch
          checked={status.autoCheck}
          onCheckedChange={(enabled) => void run(() => Updates.setAutoCheck(enabled))}
          aria-label="Selbst nach neuen Fassungen suchen"
        />
      </Field>
    </>
  );
};
