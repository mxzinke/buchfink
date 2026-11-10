import { useCallback, useEffect, useState } from 'react';
import { Api } from '../services/api';
import { useLiveReload } from '../live';
import type { MasterDataChangeLine, MasterDataProposal } from '../types';
import { agentName } from '../utils/actor';
import { formatDateOfTimestamp } from '../utils/formatters';
import { useWorkspace } from '../workspace';
import { useWriteLock } from './WriteLock';
import {
  Button,
  Dialog,
  Field,
  Help,
  Notice,
  Progress,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Textarea,
  Tr,
  toast,
} from './ui';

/** Öffnet den Dialog wieder, auch für Vorschläge, die auf „Später" stehen. */
export const OPEN_MASTER_DATA_PROPOSALS = 'buchfink:stammdaten-vorschlaege-oeffnen';

/** „Übernehmen" wird erst nach dieser Zeit klickbar: erst lesen, dann entscheiden. */
const ARM_DELAY_MS = 2000;

function byArea(lines: MasterDataChangeLine[]): [string, MasterDataChangeLine[]][] {
  const groups = new Map<string, MasterDataChangeLine[]>();
  for (const line of lines) groups.set(line.area, [...(groups.get(line.area) ?? []), line]);
  return Array.from(groups.entries());
}

/**
 * Ein verbundener Agent schlägt Unternehmensdaten, Gründung oder Bankkonten
 * vor. Der Dialog zeigt die Vorschau gegen den heutigen Stand; übernommen wird
 * genau sie. Schließen heißt „Später", nicht ablehnen.
 */
export function MasterDataProposalDialog() {
  const { activeTenant, reloadSettings } = useWorkspace();
  const writeLock = useWriteLock();
  const [proposals, setProposals] = useState<MasterDataProposal[]>([]);
  const [deferred, setDeferred] = useState<number[]>([]);
  // Die Vorschau, für die „Übernehmen" scharf ist. An den Fingerabdruck
  // gebunden: ändert sich die Vorschau, läuft die Wartezeit neu.
  const [armedFor, setArmedFor] = useState<string | null>(null);
  const [progress, setProgress] = useState(0);
  const [busy, setBusy] = useState(false);
  const [rejecting, setRejecting] = useState(false);
  const [note, setNote] = useState('');

  const load = useCallback(async () => {
    try {
      setProposals(await Api.getMasterDataProposals());
    } catch {
      setProposals([]);
    }
  }, []);

  useEffect(() => {
    setDeferred([]);
    void load();
  }, [load, activeTenant?.id]);

  useLiveReload(load, ['stammdaten']);

  useEffect(() => {
    const reopen = () => setDeferred([]);
    window.addEventListener(OPEN_MASTER_DATA_PROPOSALS, reopen);
    return () => window.removeEventListener(OPEN_MASTER_DATA_PROPOSALS, reopen);
  }, []);

  const shown = proposals.filter((p) => !deferred.includes(p.id));
  const current = shown[0];
  const armKey = current ? `${current.id}:${current.fingerprint}` : undefined;
  const armed = armKey !== undefined && armedFor === armKey;

  useEffect(() => {
    if (armKey === undefined) return;
    const started = Date.now();
    setProgress(0);
    const tick = window.setInterval(() => {
      setProgress(Math.min(100, ((Date.now() - started) / ARM_DELAY_MS) * 100));
    }, 100);
    const timer = window.setTimeout(() => {
      window.clearInterval(tick);
      setProgress(100);
      setArmedFor(armKey);
    }, ARM_DELAY_MS);
    return () => {
      window.clearTimeout(timer);
      window.clearInterval(tick);
    };
  }, [armKey]);

  useEffect(() => {
    setRejecting(false);
    setNote('');
  }, [current?.id]);

  if (!current) return null;

  const later = () => setDeferred((prev) => [...prev, current.id]);

  const accept = async () => {
    if (busy || !armed) return;
    setBusy(true);
    try {
      await Api.acceptMasterDataProposal(current.id, current.fingerprint);
      toast.success('Die Stammdaten sind übernommen.');
      await reloadSettings();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      await load();
      setBusy(false);
    }
  };

  const reject = async () => {
    if (busy) return;
    setBusy(true);
    try {
      await Api.rejectMasterDataProposal(current.id, note.trim());
      toast.success(`Abgelehnt. ${agentName(current.channel)} sieht die Entscheidung.`);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      await load();
      setBusy(false);
    }
  };

  const agent = agentName(current.channel);
  const waiting = shown.length - 1;
  const blocked = Boolean(current.problem) || writeLock.locked;

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) later();
      }}
      title={`${agent} schlägt Änderungen an den Stammdaten vor`}
      width="max-w-2xl"
      footer={
        rejecting ? (
          <>
            <Button variant="secondary" disabled={busy} onClick={() => setRejecting(false)}>
              Zurück
            </Button>
            <Button variant="danger" disabled={busy || writeLock.locked} onClick={() => void reject()}>
              Ablehnen
            </Button>
          </>
        ) : (
          <>
            <Button variant="quiet" disabled={busy} onClick={later}>
              Später
            </Button>
            <Button
              variant="secondary"
              disabled={busy || writeLock.locked}
              title={writeLock.hint}
              onClick={() => setRejecting(true)}
            >
              Ablehnen …
            </Button>
            <Button
              variant="primary"
              disabled={!armed || busy || blocked}
              title={writeLock.locked ? writeLock.hint : armed ? undefined : 'Prüfen Sie zuerst die Vorschau.'}
              onClick={() => void accept()}
            >
              Übernehmen
            </Button>
          </>
        )
      }
    >
      <p className="flex items-center text-body text-ink">
        {current.reason}
        <Help
          summary="Nichts davon gilt, bevor Sie übernehmen. Übernommen wird genau diese Vorschau."
          label="Erklärung zum Vorschlag"
        >
          {agent} hat die Angaben gelesen, etwa aus einem Schreiben, und darf Stammdaten nicht selbst ändern.
          Ändert sich der Stand, bevor Sie übernehmen, zeigt Buchfink die neue Vorschau. Lehnen Sie ab,
          sieht {agent} die Ablehnung und Ihren Grund.
        </Help>
      </p>
      <p className="text-caption text-ink-subtle mt-0.5">
        Vorgeschlagen am {formatDateOfTimestamp(current.createdAt)}
        {current.receiptNumber ? ` · aus Beleg ${current.receiptNumber}` : ''}
      </p>

      {current.problem && <Notice className="mt-4" text={current.problem} />}

      {byArea(current.lines).map(([area, lines]) => (
        <section key={area} className="mt-6">
          <h3 className="text-label text-ink-muted mb-2">{area}</h3>
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th>Feld</Th>
                <Th>Bisher</Th>
                <Th>Neu</Th>
              </Tr>
            </Thead>
            <Tbody>
              {lines.map((line) => (
                <Tr key={`${line.area}:${line.field}`}>
                  <Td className="text-ink-muted">{line.field}</Td>
                  <Td className="whitespace-normal text-ink-subtle">{line.before || '—'}</Td>
                  <Td className="whitespace-normal text-ink">{line.after || '—'}</Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        </section>
      ))}

      {rejecting ? (
        <Field label={`Grund für ${agent}`} optional className="mt-6">
          <Textarea
            value={note}
            onChange={(e) => setNote(e.target.value)}
            placeholder="Etwa: Die Nummer gehört zu einem anderen Unternehmen."
          />
        </Field>
      ) : (
        <div className="mt-6">
          {armed || blocked ? (
            waiting > 0 && (
              <p className="text-caption text-ink-subtle">
                Danach {waiting === 1 ? 'wartet ein weiterer Vorschlag' : `warten ${waiting} weitere Vorschläge`}.
              </p>
            )
          ) : (
            <Progress value={progress} label="Vorschau prüfen" />
          )}
        </div>
      )}
    </Dialog>
  );
}
