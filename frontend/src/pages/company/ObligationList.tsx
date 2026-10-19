import { todayISO } from '../../utils/calendar';
import React, { useState } from 'react';
import { Api } from '../../services/api';
import type { Deadline } from '../../types';
import { useWriteLock } from '../../components/WriteLock';
import { formatDate } from '../../utils/formatters';
import { Button, Dialog, Field, Help, Input, Notice, StatusBadge, toast } from '../../components/ui';

export function obligationsOf(deadlines: Deadline[], kinds: readonly string[]): Deadline[] {
  return deadlines.filter((d) => kinds.some((kind) => d.key.startsWith(`${kind}.`)));
}



/**
 * Pflichten, die Buchfink nicht aus den Daten erkennt: Anmeldungen bei Behörden
 * und Registern. Der Mensch bestätigt sie mit dem Tag der Erledigung.
 * `action` setzt vor die Bestätigung, was eine Pflicht zusätzlich braucht,
 * etwa das Dokument zum Unterzeichnen.
 */
export const ObligationList: React.FC<{
  deadlines: Deadline[];
  onConfirmed: () => void;
  action?: (deadline: Deadline) => React.ReactNode;
}> = ({ deadlines, onConfirmed, action }) => {
  const writeLock = useWriteLock();
  const [confirming, setConfirming] = useState<Deadline | null>(null);
  const [date, setDate] = useState(todayISO());
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  if (deadlines.length === 0) return null;

  async function confirm() {
    if (!confirming) return;
    setBusy(true);
    setError('');
    try {
      await Api.confirmObligation(confirming.key, {
        date,
        kind: 'confirmation',
        note: 'in der App als erledigt bestätigt',
      });
      toast.success(`${confirming.title}: als erledigt bestätigt.`);
      setConfirming(null);
      onConfirmed();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mt-6">
      <h3 className="text-label text-ink-muted mb-2">Pflichten</h3>
      <ul className="border-t border-line">
        {deadlines.map((d) => {
          const done = d.isDone || d.status === 'done' || d.status === 'waived';
          const overdue = !done && (d.status === 'overdue' || (d.dueDate !== '' && d.dueDate < todayISO()));
          return (
            <li
              key={d.key}
              id={d.key}
              className="flex flex-wrap items-center justify-between gap-x-4 gap-y-1 py-2.5 border-b border-line"
            >
              <span className="min-w-0">
                <span className="flex items-center text-body text-ink">
                  {d.title}
                  {d.description && (
                    <Help summary={d.description} label={`Erklärung zu ${d.title}`}>
                      {d.reference ? `${d.description} Grundlage: ${d.reference}.` : d.description}
                    </Help>
                  )}
                </span>
                <span className="block text-caption text-ink-subtle num">
                  {done
                    ? d.doneOn
                      ? `Erledigt am ${formatDate(d.doneOn)}`
                      : 'Erledigt'
                    : d.dueDate
                      ? `Fällig am ${formatDate(d.dueDate)}`
                      : 'Ohne festen Termin'}
                </span>
              </span>
              <span className="flex items-center gap-3">
                <StatusBadge status={done ? 'erledigt' : overdue ? 'ueberfaellig' : 'offen'} />
                {action?.(d)}
                {!done && d.confirmable && (
                  <Button
                    variant="secondary"
                    size="sm"
                    disabled={writeLock.locked}
                    title={writeLock.hint}
                    onClick={() => {
                      setDate(todayISO());
                      setError('');
                      setConfirming(d);
                    }}
                  >
                    Als erledigt bestätigen
                  </Button>
                )}
              </span>
            </li>
          );
        })}
      </ul>

      <Dialog
        open={confirming !== null}
        onOpenChange={(open) => !open && setConfirming(null)}
        title={confirming?.title ?? ''}
        width="max-w-md"
        footer={
          <>
            <Button variant="secondary" onClick={() => setConfirming(null)}>
              Abbrechen
            </Button>
            <Button variant="primary" loading={busy} onClick={() => void confirm()}>
              Als erledigt bestätigen
            </Button>
          </>
        }
      >
        <p className="text-body text-ink-muted mb-5">
          Die Bestätigung steht mit dem Tag im Pflichtenkalender und im Änderungsprotokoll.
        </p>
        <Field label="Erledigt am">
          <Input type="date" value={date} max={todayISO()} onChange={(e) => setDate(e.target.value)} />
        </Field>
        {error && <Notice tone="negative" className="mt-5" text={error} />}
      </Dialog>
    </div>
  );
};
