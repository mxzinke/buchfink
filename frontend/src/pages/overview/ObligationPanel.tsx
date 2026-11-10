import React, { useEffect, useState } from 'react';
import { Api } from '../../services/api';
import type { ConnectedApp, EvidenceKind, Obligation } from '../../types';
import { formatDate, formatDateOfTimestamp, formatDateRange } from '../../utils/formatters';
import {
  appName,
  CHANNEL_LABELS,
  CONFIRMABLE_EVIDENCE,
  defaultEvidence,
  EVIDENCE_LABELS,
  EVIDENCE_SOURCE_LABELS,
  isConfirmable,
  needsReference,
  RESPONSIBILITY_LABELS,
  todayISO,
} from '../../utils/overview';
import { targetLabel } from '../../utils/findings';
import { toRoute, useNavigation } from '../../navigation';
import { useWriteLock } from '../../components/WriteLock';
import {
  Button,
  DetailList,
  DetailPanel,
  Dialog,
  Disclosure,
  Field,
  FormGrid,
  Input,
  Notice,
  Select,
  SkeletonRows,
  StatusBadge,
  Textarea,
  toast,
  type Status,
} from '../../components/ui';

function message(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

/** Der Stand als Abzeichen. Eine entfallene Pflicht hat keins; sie nennt den Grund. */
function obligationBadge(o: Obligation): Status | null {
  switch (o.status) {
    case 'overdue':
      return 'ueberfaellig';
    case 'done':
      return 'erledigt';
    case 'waived':
      return null;
    default:
      return 'offen';
  }
}

/** Was der Mensch jetzt tut, in einem Satz. */
function nextStep(o: Obligation, preparedBy: string | null): string {
  if (o.status === 'done') return 'Erledigt.';
  if (o.status === 'waived') return `Entfällt: ${o.waivedReason ?? ''}`;
  if (o.evidenceSource === 'data') {
    switch (o.kind) {
      case 'ustva':
      case 'zm':
        return 'Tragen Sie nach der Abgabe das Transferticket im Monatsabschluss ein.';
      case 'festschreibung':
        return 'Schreiben Sie den Monat im Monatsabschluss fest.';
      default:
        return 'Den Stand setzen Sie im Jahresabschluss.';
    }
  }
  if (preparedBy) {
    return `${preparedBy} hat das vorbereitet. Bestätigen Sie es hier mit dem Nachweis, sobald es abgegeben ist.`;
  }
  if (o.kind === 'kst-vorauszahlung' || o.kind === 'gewst-vorauszahlung') {
    return 'Zahlen Sie den Betrag aus dem Bescheid. Erledigt ist das mit der gebuchten Zahlung oder Ihrer Bestätigung hier.';
  }
  switch (o.responsibility) {
    case 'agent_prepares':
      return 'Claude bereitet das vor. Danach geben Sie es ab und bestätigen es hier mit dem Nachweis.';
    case 'tax_advisor':
      return 'Ihr Steuerberater gibt ab. Bestätigen Sie es hier mit dem Nachweis, den er Ihnen nennt.';
    default:
      return 'Erledigen Sie das selbst und bestätigen Sie es hier mit dem Nachweis.';
  }
}

function dueText(o: Obligation): string {
  if (o.immediate) return `unverzüglich, ab ${formatDate(o.dueDate)}`;
  if (o.statutoryDate && o.statutoryDate !== o.dueDate) {
    const reason = o.shiftReason ? `, ${o.shiftReason}` : '';
    return `am ${formatDate(o.dueDate)} (statt ${formatDate(o.statutoryDate)}${reason})`;
  }
  return `am ${formatDate(o.dueDate)}`;
}

export interface ObligationPanelProps {
  /** Schlüssel der Pflicht aus dem Ziel; ohne ist die Seitenleiste zu. */
  obligationKey: string | undefined;
  onClose: () => void;
  /** Nach Bestätigen, Entfallen oder Zurücknehmen: die Übersicht lädt neu. */
  onChanged: () => void;
  apps: ConnectedApp[];
}

/**
 * Die Einzelheiten einer Pflicht in der Seitenleiste: zuerst, was jetzt zu tun
 * ist, darunter Frist, Weg und Zuständigkeit, eingeklappt der Hintergrund.
 * Was Buchfink nicht aus den Daten erkennt, bestätigt der Mensch hier mit
 * Nachweis; das lässt sich acht Sekunden lang rückgängig machen.
 */
export const ObligationPanel: React.FC<ObligationPanelProps> = ({
  obligationKey,
  onClose,
  onChanged,
  apps,
}) => {
  const { navigate } = useNavigation();
  const writeLock = useWriteLock();
  const [obligation, setObligation] = useState<Obligation | null>(null);
  const [error, setError] = useState('');
  const [confirming, setConfirming] = useState(false);
  const [waiving, setWaiving] = useState(false);

  useEffect(() => {
    if (!obligationKey) return;
    let current = true;
    setObligation(null);
    setError('');
    Api.getObligation(obligationKey)
      .then((o) => current && setObligation(o))
      .catch((e) => current && setError(message(e)));
    return () => {
      current = false;
    };
  }, [obligationKey]);

  const o = obligation?.key === obligationKey ? obligation : null;
  const route = o ? toRoute(o.target.area, o.target.tab, o.target.id) : null;
  const preparedBy = o?.preparation ? appName(o.preparation.channel, apps) : null;
  const badge = o ? obligationBadge(o) : null;
  const confirmable = o ? isConfirmable(o) : false;

  async function reopen(key: string, title: string) {
    try {
      await Api.reopenObligation(key, 'In der Übersicht rückgängig gemacht');
      toast.info(`${title} steht wieder offen.`);
      onChanged();
    } catch (e) {
      toast.error(message(e));
    }
  }

  function decided(done: Obligation, text: string) {
    setConfirming(false);
    setWaiving(false);
    onClose();
    onChanged();
    toast.undo(text, () => void reopen(done.key, done.title));
  }

  const items = o
    ? [
        ...(o.periodFrom && o.periodTo
          ? [{ label: 'Zeitraum', value: formatDateRange(o.periodFrom, o.periodTo), numeric: true }]
          : []),
        ...(CHANNEL_LABELS[o.channel] ? [{ label: 'Weg', value: CHANNEL_LABELS[o.channel] }] : []),
        { label: 'Zuständig', value: RESPONSIBILITY_LABELS[o.responsibility] ?? o.responsibility },
        { label: 'Erledigt durch', value: EVIDENCE_SOURCE_LABELS[o.evidenceSource] ?? o.evidenceSource },
        ...(o.preparation
          ? [
              {
                label: 'Vorbereitet',
                value: `${preparedBy} am ${formatDateOfTimestamp(o.preparation.at)}`,
              },
              ...(o.preparation.note ? [{ label: 'Notiz', value: o.preparation.note }] : []),
            ]
          : []),
        ...(o.proof
          ? [
              {
                label: 'Nachweis',
                value: [EVIDENCE_LABELS[o.proof.kind], o.proof.reference, `vom ${formatDate(o.proof.date)}`]
                  .filter(Boolean)
                  .join(' '),
              },
            ]
          : []),
      ]
    : [];

  return (
    <DetailPanel
      open={Boolean(obligationKey)}
      onOpenChange={(open) => !open && onClose()}
      title={o?.title ?? 'Pflicht'}
      meta={badge && <StatusBadge status={badge} />}
      context={o ? `Fällig ${dueText(o)}` : undefined}
      footer={
        o && (
          <>
            {route && (
              <Button
                variant={confirmable ? 'quiet' : 'secondary'}
                className="mr-auto"
                onClick={() => navigate(route)}
              >
                {targetLabel(route)}
              </Button>
            )}
            {confirmable && (
              <>
                <Button
                  variant="secondary"
                  disabled={writeLock.locked}
                  title={writeLock.hint}
                  onClick={() => setWaiving(true)}
                >
                  Entfällt
                </Button>
                <Button
                  variant="primary"
                  disabled={writeLock.locked}
                  title={writeLock.hint}
                  onClick={() => setConfirming(true)}
                >
                  Als erledigt bestätigen
                </Button>
              </>
            )}
          </>
        )
      }
    >
      {error ? (
        <Notice tone="negative" text={error} />
      ) : !o ? (
        <SkeletonRows rows={5} />
      ) : (
        <>
          <p className="text-body text-ink">{nextStep(o, preparedBy)}</p>
          <DetailList className="mt-6" items={items} />
          <Disclosure variant="row" title="Hintergrund" className="mt-6 border-t">
            <div className="flex flex-col gap-2 text-body text-ink-muted">
              <p>{o.description}</p>
              {o.hints.map((hint) => (
                <p key={hint}>{hint}</p>
              ))}
              <p>Rechtsgrundlage: {o.basis}</p>
            </div>
          </Disclosure>

          {/* Die Dialoge stehen in der Seitenleiste, damit Base UI sie als
              verschachtelt erkennt und der Fokusfang der Leiste sie nicht
              sperrt. */}
          <ConfirmObligationDialog
            open={confirming}
            obligation={o}
            onClose={() => setConfirming(false)}
            onDone={(done) => decided(done, `${done.title} ist als erledigt bestätigt.`)}
          />
          <WaiveObligationDialog
            open={waiving}
            obligation={o}
            onClose={() => setWaiving(false)}
            onDone={(done) => decided(done, `${done.title} entfällt.`)}
          />
        </>
      )}
    </DetailPanel>
  );
};

const REFERENCE_LABELS: Partial<Record<EvidenceKind, string>> = {
  transfer_ticket: 'Transferticket',
  file_number: 'Aktenzeichen',
  order_number: 'Auftragsnummer',
};

interface DecisionDialogProps {
  open: boolean;
  obligation: Obligation;
  onClose: () => void;
  onDone: (obligation: Obligation) => void;
}

/**
 * Der Nachweis zu einer Bestätigung: Art, Nummer, Tag und eine Notiz. Er steht
 * danach an der Pflicht und im Änderungsprotokoll.
 */
const ConfirmObligationDialog: React.FC<DecisionDialogProps> = ({
  open,
  obligation,
  onClose,
  onDone,
}) => {
  const writeLock = useWriteLock();
  const [kind, setKind] = useState<EvidenceKind>(() => defaultEvidence(obligation.channel));
  const [reference, setReference] = useState('');
  const [date, setDate] = useState(todayISO);
  const [note, setNote] = useState('');
  const [errors, setErrors] = useState<{ reference?: string; date?: string }>({});
  const [failure, setFailure] = useState('');
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!open) return;
    setKind(defaultEvidence(obligation.channel));
    setReference('');
    setDate(todayISO());
    setNote('');
    setErrors({});
    setFailure('');
  }, [open, obligation.key, obligation.channel]);

  const dirty = reference.trim() !== '' || note.trim() !== '';

  async function submit() {
    const next: typeof errors = {};
    if (needsReference(kind) && !reference.trim()) {
      next.reference = `Tragen Sie das ${REFERENCE_LABELS[kind]} ein.`;
    }
    if (!date) next.date = 'Tragen Sie den Tag ein.';
    else if (date > todayISO()) next.date = 'Der Tag liegt in der Zukunft.';
    setErrors(next);
    if (next.reference || next.date) return;
    setBusy(true);
    setFailure('');
    try {
      const done = await Api.confirmObligation(obligation.key, {
        date,
        kind,
        reference: needsReference(kind) ? reference.trim() : '',
        note: note.trim(),
      });
      onDone(done);
    } catch (e) {
      setFailure(message(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => !next && onClose()}
      title="Als erledigt bestätigen"
      width="max-w-lg"
      dirty={dirty}
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            loading={busy}
            disabled={writeLock.locked || busy}
            title={writeLock.hint}
            onClick={() => void submit()}
          >
            Bestätigen
          </Button>
        </>
      }
    >
      <p className="text-body text-ink-muted mb-5">{obligation.title}</p>
      {failure && <Notice tone="negative" className="mb-5" text={failure} />}
      <FormGrid>
        <Field
          label="Nachweis"
          helpSummary="Die Nummer, mit der Sie die Abgabe belegen."
          explain={
            <>
              Mein ELSTER nennt nach dem Senden ein Transferticket im Übermittlungsprotokoll.
              Unternehmensregister und Transparenzregister nennen eine Auftragsnummer, Notar und
              Gewerbeamt ein Aktenzeichen. Ohne Nummer genügt die Bestätigung mit Tag und Notiz.
            </>
          }
        >
          <Select<EvidenceKind>
            items={CONFIRMABLE_EVIDENCE.map((value) => ({ value, label: EVIDENCE_LABELS[value] }))}
            value={kind}
            onValueChange={setKind}
          />
        </Field>
        {needsReference(kind) ? (
          <Field label={REFERENCE_LABELS[kind] ?? 'Nummer'} error={errors.reference}>
            <Input value={reference} onChange={(e) => setReference(e.target.value)} />
          </Field>
        ) : (
          <div />
        )}
        <Field label="Erledigt am" error={errors.date}>
          <Input type="date" value={date} max={todayISO()} onChange={(e) => setDate(e.target.value)} />
        </Field>
      </FormGrid>
      <Field label="Notiz" optional className="mt-5">
        <Textarea value={note} rows={2} onChange={(e) => setNote(e.target.value)} />
      </Field>
    </Dialog>
  );
};

/** Eine Pflicht, die für diese Gesellschaft nicht gilt, entfällt mit Grund. */
const WaiveObligationDialog: React.FC<DecisionDialogProps> = ({ open, obligation, onClose, onDone }) => {
  const writeLock = useWriteLock();
  const [reason, setReason] = useState('');
  const [error, setError] = useState('');
  const [failure, setFailure] = useState('');
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!open) return;
    setReason('');
    setError('');
    setFailure('');
  }, [open, obligation.key]);

  async function submit() {
    if (!reason.trim()) {
      setError('Nennen Sie den Grund.');
      return;
    }
    setBusy(true);
    setFailure('');
    try {
      onDone(await Api.waiveObligation(obligation.key, reason.trim()));
    } catch (e) {
      setFailure(message(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => !next && onClose()}
      title="Pflicht entfällt"
      width="max-w-lg"
      dirty={reason.trim() !== ''}
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            loading={busy}
            disabled={writeLock.locked || busy}
            title={writeLock.hint}
            onClick={() => void submit()}
          >
            Als entfallen kennzeichnen
          </Button>
        </>
      }
    >
      <p className="text-body text-ink-muted mb-5">{obligation.title}</p>
      {failure && <Notice tone="negative" className="mb-5" text={failure} />}
      <Field label="Grund" error={error} hint="Steht im Änderungsprotokoll">
        <Textarea value={reason} rows={3} onChange={(e) => setReason(e.target.value)} />
      </Field>
    </Dialog>
  );
};
