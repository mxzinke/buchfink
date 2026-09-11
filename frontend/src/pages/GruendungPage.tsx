import React, { useCallback, useEffect, useState } from 'react';
import { ArrowLeft, Check, FileDown, FilePlus2, Paperclip } from 'lucide-react';
import { Api } from '../services/api';
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
import { GruendungHelpDialog, GruendungHelpMark } from '../components/GruendungHelp';
import { DocumentAttachDialog } from '../components/DocumentAttachDialog';
import { useWriteLock } from '../components/WriteLock';
import {
  Button,
  Dialog,
  Notice,
  PageHeader,
  Progress,
  Section,
  SkeletonRows,
  Stat,
  StatRow,
  StatusBadge,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  cn,
  toast,
} from '../components/ui';

/**
 * Der Gründungsweg.
 *
 * Er steht neben dem geführten Weg des Jahresabschlusses und folgt ihm: die
 * Schritte kommen aus dem Backend, der Fortschritt wird dort gerechnet, und
 * jeder Schritt öffnet seine Arbeit dort, wo sie wohnt.
 *
 * Was ihn vom Abschlussweg unterscheidet, ist der Anlass. Wer zum ersten Mal
 * gründet, weiß nicht, wohin er sich wenden soll — deshalb sagt jeder Schritt
 * nicht nur, was zu tun ist, sondern auch wo, mit welchen Handgriffen, und was
 * Buchfink dafür beisteuert.
 *
 * Die Seite hat keinen Eintrag in der Navigation. Sie ist eine Phase und kein
 * Ort: erreicht wird sie über den Hinweis auf der Startseite und über den
 * Gründungsabschnitt der Fristenseite, und mit der Eintragung führt dorthin
 * nichts mehr.
 */

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

  const load = useCallback(async () => {
    setLoading(true);
    try {
      setState(await Api.getFoundationState());
    } catch (e) {
      setState(null);
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  /** Die beiden Vorschauen sind Nebenauskunft: fehlen sie, bleibt der Weg der Weg. */
  const loadSheets = useCallback(async () => {
    Api.getOpeningBalance()
      .then(setOpening)
      .catch(() => setOpening(null));
    Api.getFragebogenSheet()
      .then(setFragebogen)
      .catch(() => setFragebogen(null));
  }, []);

  useEffect(() => {
    if (state?.hasFoundation) void loadSheets();
  }, [state?.hasFoundation, loadSheets]);

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

  const markDone = (duty: FoundationDuty) =>
    run(duty.key, async () => {
      const today = new Date().toISOString().slice(0, 10);
      await Api.completeFoundationDuty(duty.key, duty.isDone ? '' : today);
      await load();
    });

  const fileOpening = () =>
    run('eroeffnungsbilanz', async () => {
      const doc = await Api.fileOpeningBalance();
      toast.success(`${doc.fileName} liegt in den Unterlagen.`);
      await load();
      await loadSheets();
    });

  const exportOpeningXBRL = () =>
    run('ebilanz', async () => {
      const xbrl = await Api.exportOpeningBalanceXBRL();
      downloadBlob(
        `eroeffnungsbilanz-${opening?.asOf ?? ''}.xml`,
        new Blob([xbrl], { type: 'application/xml' })
      );
      toast.success('Die XBRL-Instanz ist gespeichert. Übermittelt wird über Mein ELSTER.');
    });

  const fileFragebogen = () =>
    run('fragebogen', async () => {
      const doc = await Api.fileFragebogenSheet();
      toast.success(`${doc.fileName} liegt in den Unterlagen.`);
      await load();
      await loadSheets();
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

  const { foundation, guide, duties } = state;
  const registered = state.stage === 'eingetragen';
  const percent = guide.total > 0 ? Math.round((guide.done / guide.total) * 100) : 0;

  return (
    <div className="max-w-[1200px] mx-auto px-8 py-8">
      <PageHeader helpSummary="Halten Sie die Schritte Ihrer Unternehmensgründung und die zugehörigen Nachweise fest."
        title="Gründung"
        context={
          registered
            ? `Eingetragen am ${formatDate(foundation.registeredOn)}`
            : `In Gründung seit ${formatDate(foundation.notarizedOn)}`
        }
        explain={
          <>
            Der Weg von der Beurkundung bis zur ersten Offenlegung. Jeder Schritt sagt, wo er zu
            erledigen ist und was dazugehört; erledigt wird er mit seinem Datum, und der Nachweis
            bleibt in den Unterlagen des Unternehmens.
          </>
        }
        onMore={() => setHelp(true)}
        action={
          <Button
            variant="secondary"
            icon={<ArrowLeft className="w-4 h-4" strokeWidth={1.5} />}
            onClick={() => onNavigate('tasks')}
          >
            Zurück zu den Aufgaben
          </Button>
        }
      />

      <Section
        title="Ihr Stand"
        context={`${guide.done} von ${guide.total} Schritten erledigt`}
        className="mt-8"
      >
        <Progress
          label="Weg durch die Gründung"
          value={percent}
          detail={`${guide.done} von ${guide.total} Schritten`}
        />
        <div className="mt-6">
          <StatRow>
            <Stat
              label="Stammkapital"
              value={formatCents(foundation.shareCapital)}
              context="laut Gesellschaftsvertrag"
            />
            <Stat
              label="Davon geleistet"
              value={formatCents(
                (foundation.shareholders ?? []).reduce((sum, s) => sum + s.paidIn, 0)
              )}
              context={
                state.anmeldung?.isSatisfied
                  ? 'Anmeldung möglich'
                  : `${formatCents(state.anmeldung?.requiredPaidIn ?? 0)} nötig`
              }
              tone={state.anmeldung?.isSatisfied ? 'positive' : 'neutral'}
            />
            <Stat
              label={
                <span className="flex items-center">
                  Unterbilanz
                  <GruendungHelpMark onMore={() => setHelp(true)} />
                </span>
              }
              value={formatCents(state.unterbilanz?.amount ?? 0)}
              context={
                state.unterbilanz?.isFinal ? 'zum Tag der Eintragung' : 'vorläufig, wächst weiter'
              }
              tone={(state.unterbilanz?.amount ?? 0) > 0 ? 'negative' : 'positive'}
            />
            <Stat
              label="Noch offen"
              value={String(guide.open)}
              context={guide.waiting > 0 ? `${guide.waiting} warten auf die Eintragung` : 'nichts wartet'}
            />
          </StatRow>
        </div>
        {guide.nextTitle && (
          <p className="mt-6 text-body text-ink-muted">
            Als Nächstes: <span className="text-ink font-semibold">{guide.nextTitle}</span>
          </p>
        )}
      </Section>

      {duties.map((duty, index) => (
        <StepSection
          key={duty.key}
          duty={duty}
          index={index}
          opening={duty.key === 'eroeffnungsbilanz' ? opening : null}
          fragebogen={duty.key === 'fragebogen' ? fragebogen : null}
          busy={busy === duty.key}
          locked={writeLock.locked}
          lockHint={writeLock.hint}
          onMarkDone={() => void markDone(duty)}
          onAttach={() => setAttachFor(duty)}
          onOpenDocument={openDocument}
          onFileOpening={() => void fileOpening()}
          onExportXBRL={() => void exportOpeningXBRL()}
          onFileFragebogen={() => void fileFragebogen()}
          onNavigate={onNavigate}
        />
      ))}

      <GruendungHelpDialog open={help} onClose={() => setHelp(false)} />

      <DocumentAttachDialog
        open={attachFor !== null}
        dutyKey={attachFor?.key ?? ''}
        dutyTitle={attachFor?.title ?? ''}
        onOpenChange={(next) => !next && setAttachFor(null)}
        onAttached={() => {
          setAttachFor(null);
          void load();
        }}
      />
    </div>
  );
};

interface StepSectionProps {
  duty: FoundationDuty;
  index: number;
  opening: OpeningBalanceSheet | null;
  fragebogen: FragebogenSheet | null;
  busy: boolean;
  locked: boolean;
  lockHint?: string;
  onMarkDone: () => void;
  onAttach: () => void;
  onOpenDocument: (doc: CompanyDocument) => void;
  onFileOpening: () => void;
  onExportXBRL: () => void;
  onFileFragebogen: () => void;
  onNavigate: NavigateFn;
}

/** Ein Schritt des Weges: was, wo, wie — und was Buchfink beisteuert. */
const StepSection: React.FC<StepSectionProps> = ({
  duty,
  index,
  opening,
  fragebogen,
  busy,
  locked,
  lockHint,
  onMarkDone,
  onAttach,
  onOpenDocument,
  onFileOpening,
  onExportXBRL,
  onFileFragebogen,
  onNavigate,
}) => {
  const state = stateOf(duty);
  const [sheetOpen, setSheetOpen] = useState(false);

  return (
    <Section helpSummary="Hier erfahren Sie, was für diesen Gründungsschritt zu erledigen ist."
      title={`${duty.order}. ${duty.title}`}
      context={duty.where}
      divider={index > 0}
      className={index === 0 ? 'mt-8' : undefined}
      explain={
        <>
          {duty.description} Frist: {duty.deadline} ({duty.reference}).
        </>
      }
      action={
        <div className="flex items-center gap-2">
          {state === 'done' ? (
            <span className="flex items-center gap-2 text-caption text-ink-subtle">
              <Check className="w-4 h-4 text-positive-text" strokeWidth={1.5} />
              {duty.doneOn ? `Erledigt am ${formatDate(duty.doneOn)}` : 'Erledigt'}
            </span>
          ) : state === 'waiting' ? (
            <span className="text-caption text-ink-subtle">Wartet auf die Eintragung</span>
          ) : duty.dueDate ? (
            <span className="text-caption text-ink-subtle num">
              Fällig {formatDate(duty.dueDate)}
            </span>
          ) : (
            <StatusBadge status="offen" />
          )}
        </div>
      }
    >
      {state !== 'done' && duty.todo.length > 0 && (
        <ol className="space-y-2">
          {duty.todo.map((step, i) => (
            <li key={step} className="flex gap-3 text-body text-ink-muted">
              <span className="shrink-0 num text-ink-subtle tabular-nums">{i + 1}.</span>
              <span>{step}</span>
            </li>
          ))}
        </ol>
      )}

      {duty.provides && state !== 'done' && (
        <p className="mt-5 rounded-control border border-line bg-surface px-4 py-3 text-body text-ink-muted">
          {duty.provides}
        </p>
      )}

      {/* Die Eröffnungsbilanz: der eine Schritt, den Buchfink vollständig kann. */}
      {duty.key === 'eroeffnungsbilanz' && opening && (
        <OpeningBalancePanel
          sheet={opening}
          busy={busy}
          locked={locked}
          lockHint={lockHint}
          onFile={onFileOpening}
          onExportXBRL={onExportXBRL}
          onShow={() => setSheetOpen(true)}
          showing={sheetOpen}
          onClose={() => setSheetOpen(false)}
          onNavigate={onNavigate}
        />
      )}

      {duty.key === 'fragebogen' && fragebogen && (
        <FragebogenPanel
          sheet={fragebogen}
          busy={busy}
          locked={locked}
          lockHint={lockHint}
          onFile={onFileFragebogen}
          onShow={() => setSheetOpen(true)}
          showing={sheetOpen}
          onClose={() => setSheetOpen(false)}
        />
      )}

      {(duty.proof ?? []).length > 0 && (
        <div className="mt-6">
          <p className="text-label text-ink-subtle mb-2">Nachweise</p>
          <ul className="space-y-1.5">
            {(duty.proof ?? []).map((doc) => (
              <li key={doc.id}>
                <button
                  type="button"
                  onClick={() => onOpenDocument(doc)}
                  className="flex items-center gap-2 text-body text-accent-text hover:text-accent
                             transition-colors duration-120 ease-quiet"
                >
                  <Paperclip className="w-3.5 h-3.5 shrink-0" strokeWidth={1.5} />
                  <span className="truncate">{doc.title || doc.fileName}</span>
                  {doc.documentDate && (
                    <span className="text-caption text-ink-subtle num">
                      {formatDate(doc.documentDate)}
                    </span>
                  )}
                </button>
              </li>
            ))}
          </ul>
        </div>
      )}

      <div className="mt-6 flex flex-wrap items-center gap-2">
        <Button
          variant="secondary"
          size="sm"
          icon={<Paperclip className="w-4 h-4" strokeWidth={1.5} />}
          disabled={locked}
          title={lockHint}
          onClick={onAttach}
        >
          Nachweis ablegen
        </Button>
        <Button
          variant="quiet"
          size="sm"
          loading={busy}
          disabled={locked}
          title={lockHint}
          onClick={onMarkDone}
        >
          {state === 'done' ? 'Als offen führen' : 'Erledigt vermerken'}
        </Button>
      </div>
    </Section>
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
          finding.includes('Zeichnung des Stammkapitals') ? (
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
        disabled={locked || !sheet.balances}
        title={
          locked
            ? lockHint
            : !sheet.balances
              ? 'Eine Bilanz, die nicht aufgeht, wird nicht abgelegt.'
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
}> = ({ sheet, busy, locked, lockHint, showing, onFile, onShow, onClose }) => {
  const missing = sheet.rows.filter((row) => row.missing).length;
  return (
    <div className="mt-6">
      {missing > 0 && (
        <Notice
          className="mb-5"
          text={`${missing} Angabe${missing === 1 ? '' : 'n'} für den Fragebogen ${
            missing === 1 ? 'fehlt' : 'fehlen'
          } in den Stammdaten. Das Datenblatt weist sie als Lücke aus.`}
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
                <Td className={cn(row.missing && 'text-attention-text')}>
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
