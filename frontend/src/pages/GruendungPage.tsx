import React, { useCallback, useEffect, useRef, useState } from 'react';
import { ArrowLeft, FileDown, FilePlus2, Paperclip } from 'lucide-react';
import { Api, COMPANY_SETTINGS_CHANGED } from '../services/api';
import { formatCents, formatDate } from '../utils/formatters';
import { downloadBlob } from '../utils/download';
import type {
  CompanyDocument,
  FoundationDuty,
  FoundationState,
  FragebogenSheet,
  OpeningBalanceSheet,
} from '../types';
import type { NavigateFn } from '../components/Sidebar';
import { GruendungHelpDialog } from '../components/GruendungHelp';
import { DocumentAttachDialog } from '../components/DocumentAttachDialog';
import { SourceLink } from '../components/ui/LegalText';
import { useWriteLock } from '../components/WriteLock';
import {
  Button,
  Checkbox,
  Help,
  Dialog,
  Notice,
  PageHeader,
  Progress,
  SkeletonRows,
  Stat,
  StatRow,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  cn,
  toast,
} from '../components/ui';

interface GruendungPageProps {
  onNavigate: NavigateFn;
}

/** Der Stand eines Schrittes, aus seinen Feldern abgeleitet. */
type StepState = 'done' | 'waiting' | 'open';

function stateOf(duty: FoundationDuty): StepState {
  if (duty.isDone) return 'done';
  if (duty.isPending) return 'waiting';
  return 'open';
}

export const GruendungPage: React.FC<GruendungPageProps> = ({ onNavigate }) => {
  const writeLock = useWriteLock();
  const [state, setState] = useState<FoundationState | null>(null);
  const [loading, setLoading] = useState(true);
  const [help, setHelp] = useState(false);
  const [busy, setBusy] = useState('');

  const [opening, setOpening] = useState<OpeningBalanceSheet | null>(null);
  const [fragebogen, setFragebogen] = useState<FragebogenSheet | null>(null);
  const [attachFor, setAttachFor] = useState<FoundationDuty | null>(null);
  const loadVersion = useRef(0);

  const load = useCallback(async () => {
    const version = ++loadVersion.current;
    try {
      const next = await Api.getFoundationState();
      if (version !== loadVersion.current) return;
      setState(next);
      setLoading(false);
      if (next?.hasFoundation) {
        const [balance, sheet] = await Promise.allSettled([Api.getOpeningBalance(), Api.getFragebogenSheet()]);
        if (version !== loadVersion.current) return;
        setOpening(balance.status === 'fulfilled' ? balance.value : null);
        setFragebogen(sheet.status === 'fulfilled' ? sheet.value : null);
      } else {
        setOpening(null);
        setFragebogen(null);
      }
    } catch (e) {
      if (version !== loadVersion.current) return;
      setState(null);
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      if (version === loadVersion.current) setLoading(false);
    }
  }, []);

  useEffect(() => {
    const refresh = () => { void load(); };
    const onVisible = () => { if (document.visibilityState === 'visible') refresh(); };
    refresh();
    window.addEventListener(COMPANY_SETTINGS_CHANGED, refresh);
    window.addEventListener('focus', refresh);
    document.addEventListener('visibilitychange', onVisible);
    return () => {
      loadVersion.current++;
      window.removeEventListener(COMPANY_SETTINGS_CHANGED, refresh);
      window.removeEventListener('focus', refresh);
      document.removeEventListener('visibilitychange', onVisible);
    };
  }, [load]);

  async function run(key: string, action: () => Promise<void>) {
    setBusy(key);
    try {
      await action();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy('');
    }
  }

  const setTaskStatus = (duty: FoundationDuty, status: string) =>
    run(duty.key, async () => {
      await Api.setFoundationDutyStatus(duty.key, status);
      await load();
    });

  const fileOpening = () =>
    run('eroeffnungsbilanz', async () => {
      const doc = await Api.fileOpeningBalance();
      toast.success(`${doc.fileName} liegt in den Unterlagen.`);
      await load();
    });

  const exportOpeningXBRL = () =>
    run('ebilanz', async () => {
      const xbrl = await Api.exportOpeningBalanceXBRL();
      downloadBlob(
        `eroeffnungsbilanz-${opening?.asOf ?? ''}.xml`,
        new Blob([xbrl], { type: 'application/xml' })
      );
      toast.success('Die XBRL-Instanz ist gespeichert. Übermittelt wird über geeignete E-Bilanz-Software.');
    });

  const fileFragebogen = () =>
    run('fragebogen', async () => {
      const doc = await Api.fileFragebogenSheet();
      toast.success(`${doc.fileName} liegt in den Unterlagen.`);
      await load();
    });

  const openDocument = async (doc: CompanyDocument) => {
    try {
      const preview = await Api.getDocumentContent(doc.id);
      window.open(preview.dataUrl, '_blank');
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    }
  };

  if (loading) {
    return (
      <div className="max-w-[1200px] mx-auto px-8 py-8">
        <SkeletonRows rows={8} />
      </div>
    );
  }

  if (!state?.applies || !state.hasFoundation || !state.foundation) {
    return (
      <div className="max-w-[1200px] mx-auto px-8 py-8">
        <PageHeader title="Gründung" context="Für diesen Mandanten ist keine Gründung erfasst" />
        <div className="mt-8">
          <Notice
            text="Der Gründungsweg gilt für Kapitalgesellschaften, deren Gründung in Buchfink erfasst ist."
            action={
              <Button variant="secondary" size="sm" onClick={() => onNavigate('tasks')}>
                Zur Aufgabenliste
              </Button>
            }
          />
        </div>
      </div>
    );
  }

  const { guide, duties } = state;
  const percent = guide.total > 0 ? Math.round((guide.done / guide.total) * 100) : 0;

  return (
    <div className="max-w-[1200px] mx-auto px-8 py-8">
      <PageHeader helpSummary="Halten Sie die Schritte Ihrer Unternehmensgründung und die zugehörigen Nachweise fest."
        title="Gründung"
        context={`${guide.done} von ${guide.total} Aufgaben abgeschlossen`}
        explain={
          <>
            Notartermin, Kapitaleinzahlung und Handelsregisteranmeldung sind bereits erledigt.
            Die Checkliste führt durch die anschließenden Meldungen und Unterlagen.
            Haken Sie erledigte Aufgaben ab und legen Sie die Nachweise dazu ab.
          </>
        }
        onMore={() => setHelp(true)}
        action={
          <Button
            variant="secondary"
            icon={<ArrowLeft className="w-4 h-4" strokeWidth={1.5} />}
            onClick={() => onNavigate('tasks')}
          >
            <span>Zurück<span className="hidden sm:inline"> zu den Aufgaben</span></span>
          </Button>
        }
      />

      <div className="mt-8">
        <Progress label="Gründungsaufgaben" value={percent} />
      </div>

      <ul aria-label="Gründungsaufgaben" className="mt-6 divide-y divide-line border-y border-line">
        {duties.map((duty) => (
          <FoundationTask
            key={duty.key}
            duty={duty}
            waitingReason={duty.isPending ? duty.waitingFor || 'Nach der Handelsregistereintragung' : undefined}
            opening={duty.key === 'eroeffnungsbilanz' ? opening : null}
            fragebogen={duty.key === 'fragebogen' ? fragebogen : null}
            busy={busy !== ''}
            locked={writeLock.locked}
            lockHint={writeLock.hint}
            onMarkDone={() => void setTaskStatus(duty, duty.isDone ? 'open' : 'done')}
            onStatus={(status) => void setTaskStatus(duty, status)}
            onAttach={() => setAttachFor(duty)}
            onFileOpening={() => void fileOpening()}
            onExportXBRL={() => void exportOpeningXBRL()}
            onFileFragebogen={() => void fileFragebogen()}
            onNavigate={onNavigate}
          />
        ))}
      </ul>
      {!state.foundation.registeredOn && (
        <Button variant="quiet" size="sm" className="mt-4" onClick={() => onNavigate('deadlines')}>
          Handelsregistereintragung erfassen
        </Button>
      )}

      <GruendungHelpDialog open={help} onClose={() => setHelp(false)} />

      <DocumentAttachDialog
        open={attachFor !== null}
        dutyKey={attachFor?.key ?? ''}
        dutyTitle={attachFor?.title ?? ''}
        documents={duties.find((duty) => duty.key === attachFor?.key)?.proof ?? []}
        onOpenDocument={openDocument}
        readOnly={writeLock.locked || !attachFor?.acceptsProof}
        onOpenChange={(next) => !next && setAttachFor(null)}
        onAttached={() => {
          setAttachFor(null);
          void load();
        }}
      />
    </div>
  );
};

interface FoundationTaskProps {
  duty: FoundationDuty;
  waitingReason?: string;
  opening: OpeningBalanceSheet | null;
  fragebogen: FragebogenSheet | null;
  busy: boolean;
  locked: boolean;
  lockHint?: string;
  onMarkDone: () => void;
  onStatus: (status: string) => void;
  onAttach: () => void;
  onFileOpening: () => void;
  onExportXBRL: () => void;
  onFileFragebogen: () => void;
  onNavigate: NavigateFn;
}

const FoundationTask: React.FC<FoundationTaskProps> = ({
  duty,
  waitingReason,
  opening,
  fragebogen,
  busy,
  locked,
  lockHint,
  onMarkDone,
  onStatus,
  onAttach,
  onFileOpening,
  onExportXBRL,
  onFileFragebogen,
  onNavigate,
}) => {
  const done = stateOf(duty) === 'done';
  const waiting = !done && Boolean(waitingReason);
  const [workOpen, setWorkOpen] = useState(false);
  const [sheetOpen, setSheetOpen] = useState(false);
  const proof = duty.proof ?? [];
  const deadline = duty.excludedBy || (duty.isNotApplicable ? 'Nicht zutreffend' : done
    ? duty.doneOn ? `Erledigt am ${formatDate(duty.doneOn)}` : 'Erledigt'
    : waiting ? waitingReason
    : duty.dueDate ? `Bis ${formatDate(duty.dueDate)}` : duty.deadline);

  const controlsDisabled = waiting || locked || busy || Boolean(duty.excludedBy) || Boolean(duty.missingFields?.length);
  const statusActions = duty.condition && (!done || duty.isNotApplicable) && !duty.excludedBy ? (
    <button type="button" disabled={controlsDisabled} title={waitingReason || lockHint}
      aria-label={`${duty.isNotApplicable ? 'Wieder öffnen' : 'Nicht zutreffend'}: ${duty.title}`}
      onClick={() => onStatus(duty.isNotApplicable ? 'open' : 'skipped')}
      className="text-caption text-ink-subtle hover:text-ink hover:underline disabled:text-ink-faint disabled:cursor-not-allowed disabled:no-underline">
      {duty.isNotApplicable ? 'Wieder öffnen' : 'Nicht zutreffend'}
    </button>
  ) : undefined;

  return (
    <li className={cn('py-4', waiting && 'text-ink-subtle')}>
      <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
        <div className="min-w-0 flex-1 basis-full sm:basis-0">
          <div className="flex items-start gap-1">
          <Checkbox
            checked={done && !duty.isNotApplicable}
            indeterminate={duty.isNotApplicable}
            disabled={controlsDisabled}
            onCheckedChange={onMarkDone}
            label={duty.title}
            hint={done ? undefined : duty.condition}
            className={cn('min-w-0', done && 'text-ink-subtle line-through', waiting && 'text-ink-subtle')}
          />
          <Help label={`Erklärung: ${duty.title}`} summary={duty.description}>
            <p>{duty.where}</p>
            <p>Frist: {duty.deadline}. {duty.reference}</p>
            {duty.condition && <p>{duty.condition}. Falls dies aktuell nicht zutrifft, wählen Sie „Nicht zutreffend“. Sie können sie später wieder öffnen.</p>}
            {waitingReason && <p>{waitingReason}.</p>}
            {duty.excludedBy && <p>{duty.excludedBy}. Diese Angabe können Sie in den Stammdaten ändern.</p>}
            {duty.todo.length > 0 && <ol className="list-decimal pl-5 space-y-2">
              {duty.todo.map((step) => <li key={step}>{step}</li>)}
            </ol>}
            {duty.provides && <p>{duty.provides}</p>}
          </Help>
          </div>
          {!!duty.missingFields?.length && <p className="ml-[26px] mt-1 text-caption text-ink-muted">Noch offen: {duty.missingFields.join(', ')}</p>}
          {statusActions && <div className="ml-[26px] mt-1 w-fit max-w-full">{statusActions}</div>}
        </div>
        <span className={cn('text-caption sm:text-right sm:max-w-52', done || waiting ? 'text-ink-subtle' : 'text-ink-muted')}>
          {deadline}
          {waiting && duty.dueDate && <span className="block">Frist: {formatDate(duty.dueDate)}</span>}
        </span>
        <div className="ml-auto flex shrink-0 items-center gap-2 text-caption">
          {duty.actionUrl && (waiting
            ? <span className="text-ink-subtle">{duty.actionLabel}</span>
            : <SourceLink href={duty.actionUrl}>{duty.actionLabel || 'Portal öffnen'}</SourceLink>)}
          {(['stammdaten', 'geschaeftsbriefe'].includes(duty.key) || waitingReason === 'Beschäftigte in den Stammdaten angeben') && (
            <Button variant="quiet" size="sm" onClick={() => onNavigate('settings')}>{duty.key === 'stammdaten' ? 'Zu den Einstellungen' : 'Stammdaten'}</Button>
          )}
          {(opening || fragebogen) && (
            <Button variant="quiet" size="sm" disabled={waiting || busy} onClick={() => setWorkOpen(true)}>
              {opening ? 'Bilanz' : 'Datenblatt'}
            </Button>
          )}
          {(duty.acceptsProof || proof.length > 0) && <Button variant="quiet" size="sm" iconOnly
            aria-label={`Nachweise: ${duty.title}`} title={proof.length ? `${proof.length} ${proof.length === 1 ? 'Nachweis' : 'Nachweise'}` : 'Nachweis ablegen'}
            aria-description={`${proof.length} ${proof.length === 1 ? 'Nachweis' : 'Nachweise'}`}
            disabled={controlsDisabled}
            onClick={onAttach}>
            <Paperclip className="w-4 h-4" strokeWidth={1.5} />
            {proof.length > 0 && <span aria-hidden="true" className={cn('absolute -right-1 -top-1 min-w-4 h-4 rounded-full px-1 text-[10px] leading-4 tabular-nums text-center', controlsDisabled ? 'bg-sunken text-ink-faint' : 'bg-accent-soft text-accent-text')}>{proof.length}</span>}
          </Button>}
        </div>
      </div>

      <Dialog open={workOpen} onOpenChange={setWorkOpen} title={duty.title}>
        {opening && <OpeningBalancePanel sheet={opening} busy={busy} locked={locked} lockHint={lockHint}
          onFile={onFileOpening} onExportXBRL={onExportXBRL} onShow={() => setSheetOpen(true)}
          showing={sheetOpen} onClose={() => setSheetOpen(false)} onNavigate={onNavigate} />}
        {fragebogen && <FragebogenPanel sheet={fragebogen} busy={busy} locked={locked} lockHint={lockHint}
          onFile={onFileFragebogen} onShow={() => setSheetOpen(true)} showing={sheetOpen}
          onClose={() => setSheetOpen(false)} onSettings={() => onNavigate('settings')} />}
      </Dialog>
    </li>
  );
};

/** Die Eröffnungsbilanz mit ihren beiden Ausgaben: PDF und XBRL. */
const OpeningBalancePanel: React.FC<{
  sheet: OpeningBalanceSheet;
  busy: boolean;
  locked: boolean;
  lockHint?: string;
  showing: boolean;
  onFile: () => void;
  onExportXBRL: () => void;
  onShow: () => void;
  onClose: () => void;
  onNavigate: NavigateFn;
}> = ({ sheet, busy, locked, lockHint, showing, onFile, onExportXBRL, onShow, onClose, onNavigate }) => (
  <div className="mt-6">
    <StatRow>
      <Stat label="Stichtag" value={formatDate(sheet.asOf)} context="Tag der Beurkundung" />
      <Stat label="Aktiva" value={formatCents(sheet.assets)} />
      <Stat
        label="Passiva"
        value={formatCents(sheet.equity)}
        context={sheet.balances ? 'geht auf' : 'geht nicht auf'}
        tone={sheet.balances ? 'positive' : 'negative'}
      />
      <Stat
        label="Abgelegt"
        value={sheet.filedOn ? formatDate(sheet.filedOn) : '—'}
        context={sheet.filedOn ? 'in den Unterlagen' : 'noch nicht aufgestellt'}
      />
    </StatRow>

    {sheet.findings.map((finding) => (
      <Notice
        key={finding}
        className="mt-5"
        text={finding}
        action={
          ['Zeichnung des Stammkapitals', 'Gründungsbuchungen vervollständigen', 'Einzahlung vervollständigen'].some((text) => finding.includes(text)) ? (
            <Button variant="secondary" size="sm" onClick={() => onNavigate('deadlines')}>
              Gründung buchen
            </Button>
          ) : (
            <Button variant="secondary" size="sm" onClick={() => onNavigate('settings')}>
              Zu den Stammdaten
            </Button>
          )
        }
      />
    ))}

    <div className="mt-5 flex flex-wrap gap-2">
      <Button variant="secondary" size="sm" onClick={onShow}>
        Ansehen
      </Button>
      <Button
        variant="primary"
        size="sm"
        icon={<FilePlus2 className="w-4 h-4" strokeWidth={1.5} />}
        loading={busy}
        disabled={locked || !sheet.balances || sheet.findings.length > 0}
        title={
          locked
            ? lockHint
            : !sheet.balances || sheet.findings.length > 0
              ? 'Bitte zuerst die oben genannten fehlenden Angaben und Buchungen ergänzen.'
              : undefined
        }
        onClick={onFile}
      >
        Aufstellen und ablegen
      </Button>
      <Button
        variant="secondary"
        size="sm"
        icon={<FileDown className="w-4 h-4" strokeWidth={1.5} />}
        disabled={!sheet.balances}
        onClick={onExportXBRL}
      >
        E-Bilanz erzeugen
      </Button>
    </div>

    <Dialog
      open={showing}
      onOpenChange={(next) => !next && onClose()}
      title={`Eröffnungsbilanz zum ${formatDate(sheet.asOf)}`}
      footer={
        <Button variant="secondary" onClick={onClose}>
          Schließen
        </Button>
      }
    >
      <Table>
        <Thead>
          <Tr>
            <Th>Position</Th>
            <Th numeric className="w-40">
              Betrag
            </Th>
          </Tr>
        </Thead>
        <Tbody>
          {[
            ...sheet.statement.assets.map((l) => ({ ...l, side: 'Aktiva' })),
            ...sheet.statement.liabilities.map((l) => ({ ...l, side: 'Passiva' })),
          ]
            .filter((line) => line.amount !== 0 && !line.isSubtotal)
            .map((line) => (
              <Tr key={line.side + line.key}>
                <Td>
                  <span className={cn(line.level > 1 && 'pl-4 text-ink-muted')}>
                    {line.ordinal} {line.label}
                  </span>
                </Td>
                <Td numeric>{formatCents(line.amount)}</Td>
              </Tr>
            ))}
          <Tr variant="sum">
            <Td>Summe Aktiva</Td>
            <Td numeric>{formatCents(sheet.assets)}</Td>
          </Tr>
          <Tr variant="sum">
            <Td>Summe Passiva</Td>
            <Td numeric>{formatCents(sheet.equity)}</Td>
          </Tr>
        </Tbody>
      </Table>
    </Dialog>
  </div>
);

/** Das Datenblatt zum Fragebogen: was Buchfink weiß und was es nicht weiß. */
const FragebogenPanel: React.FC<{
  sheet: FragebogenSheet;
  busy: boolean;
  locked: boolean;
  lockHint?: string;
  showing: boolean;
  onFile: () => void;
  onShow: () => void;
  onClose: () => void;
  onSettings: () => void;
}> = ({ sheet, busy, locked, lockHint, showing, onFile, onShow, onClose, onSettings }) => {
  const missing = sheet.rows.filter((row) => row.missing).length;
  return (
    <div className="mt-6">
      {missing > 0 && (
        <Notice
          className="mb-5"
          text={`${missing} Angabe${missing === 1 ? '' : 'n'} für den Fragebogen ${
            missing === 1 ? 'fehlt' : 'fehlen'
          } in den Stammdaten. Ergänzen Sie diese vor der Verwendung des Datenblatts.`}
          action={<Button variant="secondary" size="sm" onClick={onSettings}>Stammdaten ergänzen</Button>}
        />
      )}
      <div className="flex flex-wrap gap-2">
        <Button variant="secondary" size="sm" onClick={onShow}>
          Ansehen
        </Button>
        <Button
          variant="primary"
          size="sm"
          icon={<FilePlus2 className="w-4 h-4" strokeWidth={1.5} />}
          loading={busy}
          disabled={locked}
          title={lockHint}
          onClick={onFile}
        >
          Datenblatt ablegen
        </Button>
      </div>

      <Dialog
        open={showing}
        onOpenChange={(next) => !next && onClose()}
        title="Datenblatt zum Fragebogen"
        width="max-w-4xl"
        footer={
          <Button variant="secondary" onClick={onClose}>
            Schließen
          </Button>
        }
      >
        <Table>
          <Thead>
            <Tr>
              <Th className="w-40">Abschnitt</Th>
              <Th>Angabe</Th>
              <Th>Inhalt</Th>
            </Tr>
          </Thead>
          <Tbody>
            {sheet.rows.map((row) => (
              <Tr key={row.section + row.label}>
                <Td className="text-ink-subtle">{row.section}</Td>
                <Td>{row.label}</Td>
                <Td className={cn('whitespace-normal break-words', row.missing && 'text-attention-text')}>
                  {row.missing ? 'nicht erfasst' : row.value}
                </Td>
              </Tr>
            ))}
          </Tbody>
        </Table>

        <div className="mt-6">
          <p className="text-label text-ink-subtle mb-2">Was der Fragebogen außerdem verlangt</p>
          <ul className="space-y-1.5 text-body text-ink-muted">
            {sheet.open.map((item) => (
              <li key={item} className="flex gap-2">
                <span className="text-ink-faint">·</span>
                <span>{item}</span>
              </li>
            ))}
          </ul>
        </div>
      </Dialog>
    </div>
  );
};

export default GruendungPage;
