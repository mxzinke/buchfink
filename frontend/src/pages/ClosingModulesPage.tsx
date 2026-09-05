import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { AlertCircle, Download, Plus } from 'lucide-react';
import type {
  Account,
  Accrual,
  AccrualKind,
  AccrualPreview,
  AccrualProposal,
  AccrualReport,
  AccrualRequest,
  Appropriation,
  AppropriationPreview,
  ClosingStepKey,
  ClosingStepView,
  ClosingSteps,
  DiscountRate,
  InventoryAccount,
  InventoryOverview,
  InventoryPreview,
  JournalLine,
  LegacySpecialDepreciationNotice,
  NotesSectionText,
  Provision,
  ProvisionKind,
  ProvisionMirror,
  ProvisionMovementKind,
  ProvisionPreview,
  Receipt,
  Reconciliation,
  TaxElectionRegister,
  TaxProvisionPreview,
  VatSettlement,
} from '../types';
import { Api } from '../services/api';
import { useWriteLock } from '../components/WriteLock';
import { downloadCSV } from '../utils/download';
import {
  formatCents,
  formatCentsPlain,
  formatDate,
  formatRateMicros,
  parseCents,
} from '../utils/formatters';
import {
  Button,
  Combobox,
  Dialog,
  EmptyState,
  Field,
  HelpPopover,
  HelpTooltip,
  Input,
  Menu,
  MenuGroup,
  MenuItem,
  MenuSeparator,
  Notice,
  PageHeader,
  Section,
  Select,
  SkeletonRows,
  Stat,
  StatRow,
  StatusBadge,
  TabPanel,
  Table,
  Tabs,
  Tbody,
  Td,
  Textarea,
  Th,
  Thead,
  Tr,
  cn,
  toast,
} from '../components/ui';

/**
 * Die Abschlussbausteine: Rechnungsabgrenzung, Rückstellungen, Inventurwert,
 * Umsatzsteuer-Verrechnung, Steuerrückstellung, Ergebnisverwendung, Anhang und
 * das Verzeichnis nach § 5 Abs. 1 Satz 2 EStG.
 *
 * Gerechnet wird ausschließlich im Backend. Jeder Baustein zeigt vor der
 * Freigabe denselben Buchungssatz, den er buchen würde — die Ansicht stellt ihn
 * dar und rechnet ihn nicht nach. Eine zweite Rechnung im Frontend wäre eine
 * zweite Wahrheit, und die ungetestete von beiden.
 *
 * Der Weg vom offenen Geschäftsjahr über Aufstellung und Feststellung bis zur
 * Offenlegung bleibt auf der Seite „Jahresabschluss": das ist der Zustand des
 * Abschlusses, hier steht die Arbeit, die zu ihm führt.
 */

type ModuleTab =
  | 'schritte'
  | 'abgrenzung'
  | 'rueckstellungen'
  | 'vorraete'
  | 'umsatzsteuer'
  | 'steuern'
  | 'ergebnis'
  | 'anhang';

/** Die Arten der Abgrenzung im Klartext; die Norm steht in der Erklärung. */
const ACCRUAL_KIND_LABELS: Record<AccrualKind, string> = {
  active: 'Ausgabe für spätere Jahre',
  passive: 'Einnahme für spätere Jahre',
  disagio: 'Damnum oder Disagio aus einem Darlehen',
};

/**
 * Die Rückstellungsarten in der Reihenfolge, in der die Maske sie anbietet:
 * die häufigen zuerst. Dieselbe Reihenfolge wie `domain.AllProvisionKinds`.
 */
const PROVISION_KIND_ORDER: ProvisionKind[] = [
  'uncertain_liability',
  'closing_costs',
  'personnel',
  'warranty_without_obligation',
  'deferred_maintenance',
  'pending_loss',
  'retention_costs',
  'tax_income',
  'tax_trade',
  'pension',
];

const PROVISION_KIND_LABELS: Record<ProvisionKind, string> = {
  uncertain_liability: 'Offene Verpflichtung, Höhe noch unklar',
  pending_loss: 'Drohender Verlust aus einem laufenden Geschäft',
  deferred_maintenance: 'Aufgeschobene Instandhaltung (Nachholung bis 31. März)',
  warranty_without_obligation: 'Gewährleistung aus Kulanz',
  tax_income: 'Körperschaftsteuer und Solidaritätszuschlag',
  tax_trade: 'Gewerbesteuer',
  closing_costs: 'Jahresabschluss- und Prüfungskosten',
  retention_costs: 'Aufbewahrung der Geschäftsunterlagen',
  personnel: 'Personalkosten (Urlaub, Tantiemen, Überstunden)',
  pension: 'Pensionszusage (nur Erfassung, keine Bewertung)',
};

/**
 * Die Steuerrückstellungen fehlen in der Auswahl mit Absicht: sie entstehen im
 * Baustein „Steuerrückstellung" aus dem Ergebnis des Jahres, und der Dienst
 * weist eine von Hand gebildete ab.
 */
const MANUAL_PROVISION_KINDS = PROVISION_KIND_ORDER.filter(
  (kind) => kind !== 'tax_income' && kind !== 'tax_trade',
);

const MOVEMENT_KIND_LABELS: Record<ProvisionMovementKind, string> = {
  formation: 'Bildung',
  increase: 'Zuführung',
  consumption: 'Verbrauch',
  release: 'Auflösung',
  unwinding: 'Aufzinsung',
};

function message(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}

function todayISO(): string {
  const now = new Date();
  const pad = (value: number) => String(value).padStart(2, '0');
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`;
}

/** Der Fehler aus dem Backend steht als Hinweisfläche über den Aktionen (§10.4). */
const BackendError: React.FC<{ message?: string }> = ({ message: text }) =>
  text ? (
    <div className="mb-5 flex items-start gap-2.5 rounded-control border border-negative-line bg-negative-soft px-4 py-3">
      <AlertCircle className="w-4 h-4 mt-0.5 shrink-0 text-negative" strokeWidth={1.5} />
      <p className="text-body text-negative-text">{text}</p>
    </div>
  ) : null;

/**
 * Der Buchungssatz in zwei Spalten, Soll links und Haben rechts (§11.1). Die
 * beiden Summen müssen sichtbar gleich sein; das ist die Kontrolle, die ein
 * Buchhalter als Erstes sucht.
 */
const PostingLines: React.FC<{ lines: JournalLine[] }> = ({ lines }) => {
  const rows = lines ?? [];
  const debit = rows.filter((line) => line.side === 'S').reduce((sum, line) => sum + line.amount, 0);
  const credit = rows.filter((line) => line.side === 'H').reduce((sum, line) => sum + line.amount, 0);

  if (rows.length === 0) {
    return <p className="text-body text-ink-muted">Aus den Angaben entsteht keine Buchung.</p>;
  }

  return (
    <Table density="kompakt">
      <Thead>
        <Tr>
          <Th className="w-28">Konto</Th>
          <Th>Text</Th>
          <Th numeric className="w-40">
            Soll
          </Th>
          <Th numeric className="w-40">
            Haben
          </Th>
        </Tr>
      </Thead>
      <Tbody>
        {rows.map((line, index) => (
          <Tr key={`${line.account}-${index}`}>
            <Td code>{line.account}</Td>
            <Td className="whitespace-normal">{line.accountName || line.text || '—'}</Td>
            <Td numeric>{line.side === 'S' ? formatCents(line.amount) : ''}</Td>
            <Td numeric>{line.side === 'H' ? formatCents(line.amount) : ''}</Td>
          </Tr>
        ))}
        <Tr variant="sum">
          <Td>Summe</Td>
          <Td />
          <Td numeric>{formatCents(debit)}</Td>
          <Td numeric>{formatCents(credit)}</Td>
        </Tr>
      </Tbody>
    </Table>
  );
};

/** Ein Abschnitt, solange er lädt: Skelettzeilen statt eines Spinners (§8.4). */
const Loading: React.FC = () => (
  <div className="mt-2">
    <SkeletonRows rows={5} />
  </div>
);

interface TabProps {
  year: number;
}

/** Die Kontoauswahl teilen sich mehrere Bausteine; geladen wird sie einmal. */
interface AccountsProps {
  accounts: Account[];
}

// -------------------------------------------------------------------------
// Schritte
// -------------------------------------------------------------------------

/**
 * Das Vokabular der Statusanzeige (§11.3) kennt kein Wort für „übersprungen":
 * es beschreibt Zustände von Buchungen, Rechnungen und Abschlüssen, nicht die
 * Entscheidung, eine Arbeit bewusst nicht zu tun. Sie bekommt deshalb kein
 * erfundenes Abzeichen, sondern die neutrale Form desselben Musters — Raute und
 * Wort, ohne Farbe, weil hier nichts gut und nichts schlecht ist.
 */
const SkippedMark: React.FC = () => (
  <span
    className={cn(
      'inline-flex items-center gap-1.5 h-5 px-2 rounded-control border border-line-strong',
      'text-caption font-medium whitespace-nowrap text-ink-subtle',
    )}
  >
    <span className="mark-diamond bg-ink-faint" aria-hidden="true" />
    Übersprungen
  </span>
);

/**
 * Welcher Reiter einen Schritt bearbeitet.
 *
 * Nur die Bausteine dieser Seite stehen darin. Abschreibung, Prüfbericht,
 * Bilanz, Feststellung und Offenlegung wohnen auf anderen Seiten; ein Verweis,
 * der dorthin führte, verließe den Reitersatz und käme nicht zurück.
 */
const STEP_TABS: Partial<Record<ClosingStepKey, ModuleTab>> = {
  accruals: 'abgrenzung',
  provisions: 'rueckstellungen',
  inventory: 'vorraete',
  vat_settlement: 'umsatzsteuer',
  tax_provision: 'steuern',
  appropriation: 'ergebnis',
};

const StepsTab: React.FC<TabProps & { onOpenTab: (tab: ModuleTab) => void }> = ({
  year,
  onOpenTab,
}) => {
  const writeLock = useWriteLock();
  const [steps, setSteps] = useState<ClosingSteps | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [skipStep, setSkipStep] = useState<ClosingStepView | null>(null);
  const [reason, setReason] = useState('');
  const [fieldError, setFieldError] = useState('');
  const [dialogError, setDialogError] = useState('');
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      setSteps(await Api.getClosingSteps(year));
    } catch (e) {
      setSteps(null);
      setError(message(e));
    } finally {
      setLoading(false);
    }
  }, [year]);

  useEffect(() => {
    void load();
  }, [load]);

  async function markDone(step: ClosingStepView) {
    setBusy(true);
    try {
      setSteps(await Api.markClosingStepDone(year, step.key));
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  async function submitSkip() {
    if (!skipStep) return;
    if (!reason.trim()) {
      setFieldError('Der Grund fehlt. Ein Abschluss ohne diesen Schritt ist eine Aussage.');
      return;
    }
    setBusy(true);
    try {
      setSteps(await Api.skipClosingStep(year, skipStep.key, reason));
      setSkipStep(null);
      setReason('');
    } catch (e) {
      setDialogError(message(e));
    } finally {
      setBusy(false);
    }
  }

  if (loading) return <Loading />;
  if (!steps) {
    return (
      <EmptyState
        title="Die Schritte konnten nicht gelesen werden"
        description={error || 'Das Geschäftsjahr ist nicht verfügbar.'}
        action={
          <Button variant="secondary" onClick={() => void load()}>
            Erneut laden
          </Button>
        }
      />
    );
  }

  return (
    <>
      <StatRow>
        <Stat
          label="Offene Schritte"
          value={String(steps.openCount)}
          context={`von ${steps.steps.length} Bausteinen`}
          tone={steps.openCount === 0 ? 'positive' : 'neutral'}
        />
        <Stat label="Bilanzstichtag" value={formatDate(steps.cutoff)} context="Datum aller Abschlussbuchungen" />
        <Stat label="Geschäftsjahr" value={String(steps.fiscalYear)} context="Folgt der Kopfzeile" />
      </StatRow>

      <Section
        title="Bausteine des Abschlusses"
        context="In fachlicher Reihenfolge: jeder Schritt setzt den vorigen voraus"
        action={
          <HelpPopover label="Erklärung zur Reihenfolge">
            Die Reihenfolge ist fachlich und nicht kosmetisch: Die Abgrenzung nimmt Aufwand aus dem
            Jahr heraus, die Rückstellung bringt welchen hinein, und erst danach steht das Ergebnis,
            aus dem die Steuerrückstellung gerechnet wird. Der Zustand folgt, wo möglich, aus den
            Daten; nur das Überspringen ist eine Angabe des Anwenders und verlangt einen Grund.
          </HelpPopover>
        }
      >
        <BackendError message={error} />
        <Table>
          <Thead>
            <Tr>
              <Th className="w-12" numeric>
                Nr.
              </Th>
              <Th>Baustein</Th>
              <Th className="w-44">Stand</Th>
              <Th>Woran der Stand liegt</Th>
              <Th className="w-56" aria-label="Aktionen" />
            </Tr>
          </Thead>
          <Tbody>
            {steps.steps.map((step) => (
              <Tr key={step.key}>
                <Td numeric className="text-ink-subtle">
                  {step.order}
                </Td>
                <Td>
                  <span className="inline-flex items-center gap-1.5">
                    {step.label}
                    <HelpTooltip label={`Erklärung zu ${step.label}`} content={step.hint} />
                  </span>
                </Td>
                <Td>
                  {step.state === 'skipped' ? (
                    <SkippedMark />
                  ) : (
                    <StatusBadge status={step.state === 'done' ? 'gebucht' : 'offen'} />
                  )}
                </Td>
                <Td className="text-ink-muted whitespace-normal">
                  {step.state === 'skipped' ? step.reason || '—' : step.detail || '—'}
                </Td>
                <Td className="pl-0">
                  <div className="flex justify-end gap-1">
                    {/* Ohne diesen Verweis führt aus der Zeile kein Weg zu der
                        Arbeit, die sie benennt — der Reiter wäre zu erraten. */}
                    {STEP_TABS[step.key] && (
                      <Button
                        variant="quiet"
                        size="sm"
                        onClick={() => onOpenTab(STEP_TABS[step.key] as ModuleTab)}
                      >
                        Bearbeiten
                      </Button>
                    )}
                    {/* Ein Schritt, dessen Zustand aus den Daten folgt, braucht
                        keinen Haken: er entsteht mit der Buchung. */}
                    {!step.automatic && (
                      <>
                        {step.state !== 'done' && (
                          <Button
                            variant="quiet"
                            size="sm"
                            disabled={busy || writeLock.locked}
                            title={writeLock.hint}
                            onClick={() => void markDone(step)}
                          >
                            Abhaken
                          </Button>
                        )}
                        {step.state !== 'skipped' && (
                          <Button
                            variant="quiet"
                            size="sm"
                            disabled={busy || writeLock.locked}
                            title={writeLock.hint}
                            onClick={() => {
                              setSkipStep(step);
                              setReason('');
                              setFieldError('');
                              setDialogError('');
                            }}
                          >
                            Überspringen
                          </Button>
                        )}
                      </>
                    )}
                  </div>
                </Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      </Section>

      <Dialog
        open={skipStep !== null}
        onOpenChange={(next) => !next && setSkipStep(null)}
        title={skipStep ? `${skipStep.label} überspringen` : ''}
        width="max-w-lg"
        footer={
          <>
            <Button variant="secondary" onClick={() => setSkipStep(null)}>
              Abbrechen
            </Button>
            <Button
              variant="primary"
              loading={busy}
              disabled={writeLock.locked}
              title={writeLock.hint}
              onClick={() => void submitSkip()}
            >
              Schritt überspringen
            </Button>
          </>
        }
      >
        <Field
          label="Grund"
          error={fieldError || undefined}
          help="Bleibt am Schritt festgehalten und steht neben ihm in der Liste."
        >
          <Textarea
            rows={3}
            value={reason}
            onChange={(e) => {
              setReason(e.target.value);
              setFieldError('');
            }}
          />
        </Field>
        <div className="mt-5">
          <BackendError message={dialogError} />
        </div>
      </Dialog>
    </>
  );
};

// -------------------------------------------------------------------------
// Rechnungsabgrenzung
// -------------------------------------------------------------------------

interface AccrualDraft {
  kind: AccrualKind;
  text: string;
  total: string;
  start: string;
  end: string;
  account: string;
  deferred: string;
  sourceEntryId: number;
}

const EMPTY_ACCRUAL: AccrualDraft = {
  kind: 'active',
  text: '',
  total: '',
  start: '',
  end: '',
  account: '',
  deferred: '',
  sourceEntryId: 0,
};

/**
 * Anlegen einer Abgrenzung. Der abgegrenzte Betrag und der Auflösungsplan
 * kommen aus der Vorschau des Backends und werden hier nicht gerechnet: Zwölftel
 * oder Kalendertage entscheidet die Einstellung des Mandanten, und ein zweiter
 * Rechenweg in der Maske ergäbe je nach Einstellung eine andere Zahl als die
 * gebuchte.
 */
const AccrualDialog: React.FC<
  AccountsProps & {
    open: boolean;
    year: number;
    initial: AccrualDraft;
    onOpenChange: (open: boolean) => void;
    onBooked: () => Promise<void>;
  }
> = ({ accounts, open, year, initial, onOpenChange, onBooked }) => {
  const writeLock = useWriteLock();
  const [draft, setDraft] = useState<AccrualDraft>(initial);
  const [preview, setPreview] = useState<AccrualPreview | null>(null);
  const [previewError, setPreviewError] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (open) {
      setDraft(initial);
      setPreview(null);
      setPreviewError('');
      setError('');
    }
  }, [open, initial]);

  const request = useMemo<AccrualRequest | null>(() => {
    const total = parseCents(draft.total);
    if (!draft.text.trim() || total === null || total <= 0) return null;
    if (!draft.start || !draft.end || !draft.account) return null;
    const deferred = parseCents(draft.deferred);
    return {
      fiscalYear: year,
      kind: draft.kind,
      sourceEntryId: draft.sourceEntryId || undefined,
      text: draft.text,
      totalAmount: total,
      startDate: draft.start,
      endDate: draft.end,
      account: draft.account,
      deferredAmount: deferred && deferred > 0 ? deferred : undefined,
    };
  }, [draft, year]);

  // Die Vorschau folgt der Eingabe mit kurzer Verzögerung: sie ist ein
  // Backend-Aufruf je Tastendruck wert, aber nicht je Buchstabe.
  useEffect(() => {
    if (!open || !request) {
      setPreview(null);
      setPreviewError('');
      return;
    }
    let cancelled = false;
    const timer = window.setTimeout(() => {
      Api.previewAccrual(request)
        .then((result) => {
          if (cancelled) return;
          setPreview(result);
          setPreviewError('');
        })
        .catch((e) => {
          if (cancelled) return;
          setPreview(null);
          setPreviewError(message(e));
        });
    }, 300);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [open, request]);

  async function submit() {
    if (!request) return;
    setBusy(true);
    setError('');
    try {
      await Api.bookAccrual(request);
      onOpenChange(false);
      await onBooked();
      toast.success('Abgrenzungsposten gebildet.');
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  const accountOptions = useMemo(
    () =>
      accounts
        .filter((account) => !account.isRange && !account.isReserved)
        .map((account) => ({
          value: account.number,
          label: `${account.number} ${account.name}`,
          meta: account.kontenklasseName,
        })),
    [accounts],
  );

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title="Rechnungsabgrenzung bilden"
      width="max-w-3xl"
      footer={
        <>
          <Button variant="secondary" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            loading={busy}
            disabled={!preview || writeLock.locked}
            title={
              writeLock.hint ??
              (preview ? undefined : 'Erst mit vollständigen Angaben entsteht ein Buchungssatz')
            }
            onClick={() => void submit()}
          >
            Abgrenzung buchen
          </Button>
        </>
      }
    >
      <div className="grid grid-cols-2 gap-4">
        <Field
          label="Art"
          explain={
            <>
              § 250 Abs. 1 HGB verlangt den aktiven Posten für Ausgaben vor dem Stichtag, die
              Aufwand einer bestimmten Zeit danach sind; Abs. 2 den passiven für Einnahmen. Das
              Damnum eines Darlehens darf nach Abs. 3 aktiviert und über die Laufzeit verteilt
              werden — dort ist der Aufwand Zinsaufwand.
            </>
          }
        >
          <Select
            items={(Object.keys(ACCRUAL_KIND_LABELS) as AccrualKind[]).map((kind) => ({
              value: kind,
              label: ACCRUAL_KIND_LABELS[kind],
            }))}
            value={draft.kind}
            onValueChange={(kind) => setDraft((prev) => ({ ...prev, kind: kind as AccrualKind }))}
          />
        </Field>
        <Field label="Text" hint="sagt, worum es geht">
          <Input
            value={draft.text}
            onChange={(e) => setDraft((prev) => ({ ...prev, text: e.target.value }))}
            placeholder="Versicherungsprämie 2027"
          />
        </Field>
        <Field label="Gesamtbetrag">
          <Input
            align="right"
            inputMode="decimal"
            placeholder="0,00"
            value={draft.total}
            onChange={(e) => setDraft((prev) => ({ ...prev, total: e.target.value }))}
          />
        </Field>
        <Field
          label={draft.kind === 'passive' ? 'Ertragskonto' : 'Aufwandskonto'}
          help="Das Konto, das der Posten entlastet und im Folgejahr wieder belastet."
        >
          <Combobox
            items={accountOptions}
            value={draft.account || null}
            onValueChange={(account) =>
              setDraft((prev) => ({ ...prev, account: account ?? '' }))
            }
            placeholder="Konto suchen …"
            emptyText="Kein Konto gefunden."
          />
        </Field>
        <Field label="Leistung von">
          <Input
            type="date"
            value={draft.start}
            onChange={(e) => setDraft((prev) => ({ ...prev, start: e.target.value }))}
          />
        </Field>
        <Field label="Leistung bis">
          <Input
            type="date"
            value={draft.end}
            onChange={(e) => setDraft((prev) => ({ ...prev, end: e.target.value }))}
          />
        </Field>
        <Field
          label="Abzugrenzender Betrag"
          optional
          hint="leer heißt: nach dem Verfahren rechnen"
          help="Nur nötig, wenn sich die Leistung nicht gleichmäßig über die Zeit verteilt."
        >
          <Input
            align="right"
            inputMode="decimal"
            placeholder="0,00"
            value={draft.deferred}
            onChange={(e) => setDraft((prev) => ({ ...prev, deferred: e.target.value }))}
          />
        </Field>
      </div>

      <div className="mt-6">
        {previewError ? (
          <BackendError message={previewError} />
        ) : preview ? (
          <>
            <StatRow className="mb-5">
              <Stat
                label="Abzugrenzen"
                value={formatCents(preview.accrual.deferredAmount)}
                context={`Buchung zum ${formatDate(preview.bookingDate)}`}
              />
              <Stat
                label="Gesamtbetrag"
                value={formatCents(preview.accrual.totalAmount)}
                context="Was die Rechnung insgesamt trägt"
              />
              <Stat
                label="Auflösungen"
                value={String(preview.releases.length)}
                context="Buchungen in den Folgejahren"
              />
            </StatRow>

            {preview.warnings.map((warning) => (
              <Notice key={warning} text={warning} className="mb-5" />
            ))}

            <PostingLines lines={preview.lines} />

            {preview.releases.length > 0 && (
              <div className="mt-5">
                <h3 className="text-label text-ink-muted mb-2">Auflösungsplan</h3>
                <Table density="kompakt">
                  <Thead>
                    <Tr>
                      <Th className="w-28">Jahr</Th>
                      <Th className="w-40">Datum</Th>
                      <Th numeric>Betrag</Th>
                    </Tr>
                  </Thead>
                  <Tbody>
                    {preview.releases.map((release) => (
                      <Tr key={`${release.fiscalYear}-${release.date}`}>
                        <Td className="num">{release.fiscalYear}</Td>
                        <Td className="num">{formatDate(release.date)}</Td>
                        <Td numeric>{formatCents(release.amount)}</Td>
                      </Tr>
                    ))}
                  </Tbody>
                </Table>
              </div>
            )}
          </>
        ) : (
          <p className="text-body text-ink-muted">
            Art, Text, Betrag, Zeitraum und Konto ergeben den Buchungssatz.
          </p>
        )}
      </div>

      <div className="mt-5">
        <BackendError message={error} />
      </div>
    </Dialog>
  );
};

const AccrualsTab: React.FC<TabProps & AccountsProps> = ({ year, accounts }) => {
  const writeLock = useWriteLock();
  const [proposal, setProposal] = useState<AccrualProposal | null>(null);
  const [accruals, setAccruals] = useState<Accrual[]>([]);
  const [report, setReport] = useState<AccrualReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [dialogOpen, setDialogOpen] = useState(false);
  const [draft, setDraft] = useState<AccrualDraft>(EMPTY_ACCRUAL);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [nextProposal, nextAccruals, nextReport] = await Promise.all([
        Api.proposeAccruals(year),
        Api.getAccruals(year),
        Api.getAccrualReport(''),
      ]);
      setProposal(nextProposal);
      setAccruals(nextAccruals);
      setReport(nextReport);
    } catch (e) {
      setError(message(e));
    } finally {
      setLoading(false);
    }
  }, [year]);

  useEffect(() => {
    void load();
  }, [load]);

  if (loading) return <Loading />;

  const items = proposal?.items ?? [];

  return (
    <>
      <StatRow>
        <Stat
          label="Vorschläge"
          value={String(items.filter((item) => !item.alreadyBooked).length)}
          context="Buchungen mit Leistung nach dem Stichtag"
        />
        <Stat
          label="Verfahren"
          value={proposal?.method === 'daily' ? 'Taggenau' : 'Monatsgenau'}
          context="Einstellung des Mandanten"
        />
        <Stat
          label="Vorschlagsschwelle"
          value={formatCents(proposal?.threshold ?? 0)}
          context="Steuerliches Wahlrecht, keine HGB-Grenze"
        />
        <Stat
          label="Bestand am Stichtag"
          value={formatCents((report?.totalActive ?? 0) + (report?.totalPassive ?? 0))}
          context={report ? `Stichtag ${formatDate(report.cutoff)}` : 'Noch kein Bestand'}
        />
      </StatRow>

      <Section
        title="Vorschläge"
        context={proposal ? `Stichtag ${formatDate(proposal.cutoff)}` : undefined}
        action={
          <div className="flex items-center gap-3">
            {/* Der Erklärtext kommt aus dem Vorschlag: dieselbe Schwelle, die
                der Dienst angewendet hat, und keine zweite Fassung, die
                auseinanderläuft. */}
            <HelpPopover label="Erklärung zur Rechnungsabgrenzung">
              {proposal?.note ||
                '§ 250 Abs. 1 HGB verlangt die Abgrenzung ohne Rücksicht auf die Höhe; die Vorschlagsschwelle ist allein ein steuerliches Wahlrecht.'}
            </HelpPopover>
            <Button
              variant="primary"
              icon={<Plus className="w-4 h-4" strokeWidth={1.5} />}
              disabled={writeLock.locked}
              title={writeLock.hint}
              onClick={() => {
                setDraft(EMPTY_ACCRUAL);
                setDialogOpen(true);
              }}
            >
              Abgrenzung bilden
            </Button>
          </div>
        }
      >
        <BackendError message={error} />

        {items.length === 0 ? (
          <EmptyState
            title="Keine Buchung reicht über den Stichtag hinaus"
            description="Buchfink schlägt eine Abgrenzung vor, sobald ein Leistungszeitraum nach dem Bilanzstichtag endet."
          />
        ) : (
          <Table>
            <Thead>
              <Tr>
                <Th className="w-28">Buchung</Th>
                <Th>Beschreibung</Th>
                <Th className="w-24">Konto</Th>
                <Th className="w-44">Leistung</Th>
                <Th numeric className="w-36">
                  Gesamt
                </Th>
                <Th numeric className="w-36">
                  Abzugrenzen
                </Th>
                <Th className="w-32" aria-label="Aktionen" />
              </Tr>
            </Thead>
            <Tbody>
              {items.map((item) => (
                <Tr key={item.entryId}>
                  <Td code>{item.entryNumber}</Td>
                  <Td className="whitespace-normal">
                    {item.description}
                    <span className="text-ink-muted">{` · ${ACCRUAL_KIND_LABELS[item.kind]}`}</span>
                    {item.belowThreshold && (
                      <span className="text-ink-muted"> · unter der Vorschlagsschwelle</span>
                    )}
                  </Td>
                  <Td code>{item.account}</Td>
                  <Td className="num text-ink-subtle">
                    {`${formatDate(item.serviceFrom)} – ${formatDate(item.serviceTo)}`}
                  </Td>
                  <Td numeric>{formatCents(item.totalAmount)}</Td>
                  <Td numeric>{formatCents(item.deferredAmount)}</Td>
                  <Td className="pl-0">
                    {item.alreadyBooked ? (
                      <StatusBadge status="gebucht" />
                    ) : (
                      <Button
                        variant="quiet"
                        size="sm"
                        disabled={writeLock.locked}
                        title={writeLock.hint}
                        onClick={() => {
                          setDraft({
                            kind: item.kind,
                            text: item.description,
                            total: formatCentsPlain(item.totalAmount),
                            start: item.serviceFrom,
                            end: item.serviceTo,
                            account: item.account,
                            deferred: '',
                            sourceEntryId: item.entryId,
                          });
                          setDialogOpen(true);
                        }}
                      >
                        Abgrenzen
                      </Button>
                    )}
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>

      <Section title="Gebildete Posten" context={`Geschäftsjahr ${year}`}>
        {accruals.length === 0 ? (
          <EmptyState
            title="Noch kein Abgrenzungsposten gebildet"
            description="Ein gebildeter Posten steht hier mit seinem Auflösungsplan."
          />
        ) : (
          <Table>
            <Thead>
              <Tr>
                <Th>Text</Th>
                <Th className="w-56">Art</Th>
                <Th className="w-24">Konto</Th>
                <Th className="w-44">Zeitraum</Th>
                <Th numeric className="w-36">
                  Abgegrenzt
                </Th>
                <Th className="w-32">Stand</Th>
              </Tr>
            </Thead>
            <Tbody>
              {accruals.map((accrual) => (
                <Tr key={accrual.id}>
                  <Td className="whitespace-normal">{accrual.text}</Td>
                  <Td className="whitespace-normal">{ACCRUAL_KIND_LABELS[accrual.kind]}</Td>
                  <Td code>{accrual.account}</Td>
                  <Td className="num text-ink-subtle">
                    {`${formatDate(accrual.startDate)} – ${formatDate(accrual.endDate)}`}
                  </Td>
                  <Td numeric>{formatCents(accrual.deferredAmount)}</Td>
                  <Td>
                    <StatusBadge status={accrual.formationEntryId ? 'gebucht' : 'entwurf'} />
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>

      <Section
        title="Bestand zum Stichtag"
        context={report ? `Restlaufzeit und Restbetrag zum ${formatDate(report.cutoff)}` : undefined}
      >
        {!report || report.rows.length === 0 ? (
          <EmptyState
            title="Kein Bestand an Abgrenzungen"
            description="Ausgewiesen wird, was gebildet und noch nicht aufgelöst ist."
          />
        ) : (
          <Table>
            <Thead>
              <Tr>
                <Th>Text</Th>
                <Th className="w-56">Art</Th>
                <Th className="w-24">Konto</Th>
                <Th numeric className="w-36">
                  Gebildet
                </Th>
                <Th numeric className="w-36">
                  Aufgelöst
                </Th>
                <Th numeric className="w-36">
                  Rest
                </Th>
                <Th numeric className="w-32">
                  Resttage
                </Th>
              </Tr>
            </Thead>
            <Tbody>
              {report.rows.map((row) => (
                <Tr key={row.accrualId}>
                  <Td className="whitespace-normal">{row.text}</Td>
                  <Td className="whitespace-normal">{row.kindLabel}</Td>
                  <Td code>{row.account}</Td>
                  <Td numeric>{formatCents(row.deferredAmount)}</Td>
                  <Td numeric className="text-ink-muted">
                    {formatCents(row.released)}
                  </Td>
                  <Td numeric>{formatCents(row.remaining)}</Td>
                  <Td numeric className="text-ink-subtle">
                    {row.remainingDays}
                  </Td>
                </Tr>
              ))}
              <Tr variant="sum">
                <Td>Summe</Td>
                <Td />
                <Td />
                <Td />
                <Td />
                <Td numeric>{formatCents(report.totalActive + report.totalPassive)}</Td>
                <Td />
              </Tr>
            </Tbody>
          </Table>
        )}
      </Section>

      <AccrualDialog
        accounts={accounts}
        open={dialogOpen}
        year={year}
        initial={draft}
        onOpenChange={setDialogOpen}
        onBooked={load}
      />
    </>
  );
};

// -------------------------------------------------------------------------
// Rückstellungen
// -------------------------------------------------------------------------

interface ProvisionDraft {
  kind: ProvisionKind;
  text: string;
  amount: string;
  expectedOn: string;
  reason: string;
  balanceAccount: string;
  expenseAccount: string;
}

const EMPTY_PROVISION: ProvisionDraft = {
  kind: 'uncertain_liability',
  text: '',
  amount: '',
  expectedOn: '',
  reason: '',
  balanceAccount: '',
  expenseAccount: '',
};

function accountOptionsOf(accounts: Account[]) {
  return accounts
    .filter((account) => !account.isRange && !account.isReserved)
    .map((account) => ({
      value: account.number,
      label: `${account.number} ${account.name}`,
      meta: account.kontenklasseName,
    }));
}

/**
 * Bildung einer Rückstellung. Abzinsung, Konten und Buchungssatz kommen aus der
 * Vorschau: Ob abgezinst wird, hängt an der Restlaufzeit und am Satz der
 * Deutschen Bundesbank, und beides gehört nicht in eine Maske.
 */
const ProvisionFormDialog: React.FC<
  AccountsProps & {
    open: boolean;
    year: number;
    onOpenChange: (open: boolean) => void;
    onBooked: () => Promise<void>;
  }
> = ({ accounts, open, year, onOpenChange, onBooked }) => {
  const writeLock = useWriteLock();
  const [draft, setDraft] = useState<ProvisionDraft>(EMPTY_PROVISION);
  const [preview, setPreview] = useState<ProvisionPreview | null>(null);
  const [previewError, setPreviewError] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (open) {
      setDraft(EMPTY_PROVISION);
      setPreview(null);
      setPreviewError('');
      setError('');
    }
  }, [open]);

  const request = useMemo(() => {
    const amount = parseCents(draft.amount);
    if (!draft.text.trim() || amount === null || amount <= 0) return null;
    if (!draft.expectedOn || !draft.reason.trim()) return null;
    return {
      fiscalYear: year,
      kind: draft.kind,
      text: draft.text,
      amount,
      expectedOn: draft.expectedOn,
      reason: draft.reason,
      balanceAccount: draft.balanceAccount || undefined,
      expenseAccount: draft.expenseAccount || undefined,
    };
  }, [draft, year]);

  useEffect(() => {
    if (!open || !request) {
      setPreview(null);
      setPreviewError('');
      return;
    }
    let cancelled = false;
    const timer = window.setTimeout(() => {
      Api.previewProvision(request)
        .then((result) => {
          if (cancelled) return;
          setPreview(result);
          setPreviewError('');
        })
        .catch((e) => {
          if (cancelled) return;
          setPreview(null);
          setPreviewError(message(e));
        });
    }, 300);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [open, request]);

  async function submit() {
    if (!request) return;
    setBusy(true);
    setError('');
    try {
      await Api.bookProvisionFormation(request);
      onOpenChange(false);
      await onBooked();
      toast.success('Rückstellung gebildet.');
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  const options = useMemo(() => accountOptionsOf(accounts), [accounts]);

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title="Rückstellung bilden"
      width="max-w-3xl"
      footer={
        <>
          <Button variant="secondary" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            loading={busy}
            disabled={!preview || writeLock.locked}
            title={
              writeLock.hint ??
              (preview ? undefined : 'Erst mit vollständigen Angaben entsteht ein Buchungssatz')
            }
            onClick={() => void submit()}
          >
            Rückstellung buchen
          </Button>
        </>
      }
    >
      <div className="grid grid-cols-2 gap-4">
        <Field
          label="Wofür wird zurückgestellt"
          explain={
            <>
              § 249 Abs. 1 HGB zählt abschließend auf, wofür eine Rückstellung gebildet werden darf;
              Abs. 2 verbietet alles Übrige. Aus der Art folgen das Bilanz- und das Aufwandskonto.
              Steuerrückstellungen entstehen im eigenen Baustein aus dem Ergebnis des Jahres.
            </>
          }
        >
          <Select
            items={MANUAL_PROVISION_KINDS.map((kind) => ({
              value: kind,
              label: PROVISION_KIND_LABELS[kind],
            }))}
            value={draft.kind}
            onValueChange={(kind) => setDraft((prev) => ({ ...prev, kind: kind as ProvisionKind }))}
          />
        </Field>
        <Field label="Text" hint="wofür die Rückstellung gebildet wird">
          <Input
            value={draft.text}
            onChange={(e) => setDraft((prev) => ({ ...prev, text: e.target.value }))}
            placeholder="Jahresabschluss 2026"
          />
        </Field>
        <Field
          label="Erfüllungsbetrag"
          help="Der Betrag, der nach vernünftiger kaufmännischer Beurteilung nötig ist."
        >
          <Input
            align="right"
            inputMode="decimal"
            placeholder="0,00"
            value={draft.amount}
            onChange={(e) => setDraft((prev) => ({ ...prev, amount: e.target.value }))}
          />
        </Field>
        <Field
          label="Erwartete Erfüllung"
          help="Ab mehr als einem Jahr Restlaufzeit wird abgezinst (§ 253 Abs. 2 HGB)."
        >
          <Input
            type="date"
            value={draft.expectedOn}
            onChange={(e) => setDraft((prev) => ({ ...prev, expectedOn: e.target.value }))}
          />
        </Field>
        <Field label="Rückstellungskonto" optional hint="leer heißt: Vorschlag aus der Art">
          <Combobox
            items={options}
            value={draft.balanceAccount || null}
            onValueChange={(account) =>
              setDraft((prev) => ({ ...prev, balanceAccount: account ?? '' }))
            }
            placeholder="Konto suchen …"
            emptyText="Kein Konto gefunden."
          />
        </Field>
        <Field label="Aufwandskonto" optional hint="leer heißt: Vorschlag aus der Art">
          <Combobox
            items={options}
            value={draft.expenseAccount || null}
            onValueChange={(account) =>
              setDraft((prev) => ({ ...prev, expenseAccount: account ?? '' }))
            }
            placeholder="Konto suchen …"
            emptyText="Kein Konto gefunden."
          />
        </Field>
      </div>

      <Field
        label="Begründung"
        className="mt-4"
        help="Eine Rückstellung ist eine Schätzung; ohne Grundlage ist sie eine Zahl ohne Herkunft."
      >
        <Textarea
          rows={3}
          value={draft.reason}
          onChange={(e) => setDraft((prev) => ({ ...prev, reason: e.target.value }))}
        />
      </Field>

      <div className="mt-6">
        {previewError ? (
          <BackendError message={previewError} />
        ) : preview ? (
          <>
            <StatRow className="mb-5">
              <Stat
                label="Erfüllungsbetrag"
                value={formatCents(preview.settlementAmount)}
                context={`Fällig ${formatDate(preview.provision.expectedDate)}`}
              />
              <Stat
                label="Bilanzansatz"
                value={formatCents(preview.amount)}
                context={
                  preview.discounted
                    ? `Abgezinst über ${preview.discountYears} Jahre mit ${preview.discountRate}`
                    : 'Ohne Abzinsung: Restlaufzeit bis zu einem Jahr'
                }
              />
              <Stat
                label="Steuerlicher Wert"
                value={formatCents(preview.taxAmount)}
                context="5,5 % nach § 6 Abs. 1 Nr. 3a EStG, nicht gebucht"
              />
            </StatRow>

            {preview.findings.map((finding) => (
              <Notice key={finding} text={finding} className="mb-5" />
            ))}

            <PostingLines lines={preview.lines} />
          </>
        ) : (
          <p className="text-body text-ink-muted">
            Art, Text, Betrag, Erfüllungszeitpunkt und Begründung ergeben den Buchungssatz.
          </p>
        )}
      </div>

      <div className="mt-5">
        <BackendError message={error} />
      </div>
    </Dialog>
  );
};

type ProvisionAction = 'increase' | 'unwinding' | 'consumption' | 'release' | 'settle';

const ACTION_TITLES: Record<ProvisionAction, string> = {
  increase: 'Rückstellung erhöhen',
  unwinding: 'Aufzinsung buchen',
  consumption: 'Verbrauch buchen',
  release: 'Rückstellung auflösen',
  settle: 'Rückstellung erledigen',
};

const ACTION_HINTS: Record<ProvisionAction, string> = {
  increase: 'Eine Zuführung ist eine geänderte Schätzung und braucht ihre eigene Begründung.',
  unwinding: 'Der Barwert wächst, weil die Fälligkeit näher rückt; Gegenposten ist Zinsaufwand.',
  consumption: 'Die Verpflichtung wird erfüllt; was die Rückstellung nicht deckt, bleibt Aufwand.',
  release: 'Aufgelöst wird nur, soweit der Grund entfallen ist (§ 249 Abs. 2 Satz 2 HGB).',
  settle: 'Ein offener Rest wird mit dieser Begründung aufgelöst.',
};

/** Zuführung, Aufzinsung, Verbrauch, Auflösung und Erledigung in einer Maske. */
const ProvisionChangeDialog: React.FC<
  AccountsProps & {
    provision: Provision | null;
    action: ProvisionAction;
    year: number;
    onOpenChange: (open: boolean) => void;
    onDone: () => Promise<void>;
  }
> = ({ accounts, provision, action, year, onOpenChange, onDone }) => {
  const writeLock = useWriteLock();
  const [amount, setAmount] = useState('');
  const [date, setDate] = useState(todayISO());
  const [reason, setReason] = useState('');
  const [paymentAccount, setPaymentAccount] = useState('');
  const [error, setError] = useState('');
  // Ein unleserlicher Betrag ist ein Fehler der Eingabe und keine Ablehnung des
  // Backends: er gehört an das Feld, nicht in die Hinweisfläche (§10.4).
  const [amountError, setAmountError] = useState('');
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (provision) {
      setAmount('');
      setDate(todayISO());
      setReason('');
      setPaymentAccount('');
      setError('');
      setAmountError('');
    }
  }, [provision, action]);

  const options = useMemo(() => accountOptionsOf(accounts), [accounts]);
  const needsAmount = action !== 'settle';

  async function submit() {
    if (!provision) return;
    const value = parseCents(amount);
    if (needsAmount && (value === null || value <= 0)) {
      setAmountError('Der Betrag fehlt oder ist nicht lesbar.');
      return;
    }
    setBusy(true);
    setError('');
    setAmountError('');
    try {
      const change = {
        provisionId: provision.id,
        amount: value ?? 0,
        date,
        reason,
        paymentAccount: paymentAccount || undefined,
      };
      switch (action) {
        case 'increase':
          await Api.bookProvisionIncrease({
            provisionId: provision.id,
            fiscalYear: year,
            kind: provision.kind,
            text: provision.text,
            amount: value ?? 0,
            expectedOn: provision.expectedDate,
            reason,
            date,
          });
          break;
        case 'unwinding':
          await Api.bookProvisionUnwinding(change);
          break;
        case 'consumption':
          await Api.bookProvisionConsumption(change);
          break;
        case 'release':
          await Api.bookProvisionRelease(change);
          break;
        case 'settle':
          await Api.settleProvision(provision.id, date, reason);
          break;
      }
      onOpenChange(false);
      await onDone();
      toast.success(`${ACTION_TITLES[action]}: gebucht.`);
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open={provision !== null}
      onOpenChange={onOpenChange}
      title={ACTION_TITLES[action]}
      width="max-w-lg"
      footer={
        <>
          <Button variant="secondary" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            loading={busy}
            disabled={writeLock.locked}
            title={writeLock.hint}
            onClick={() => void submit()}
          >
            {ACTION_TITLES[action]}
          </Button>
        </>
      }
    >
      <p className="text-body text-ink-muted mb-4">
        {provision ? `${provision.text} · ${formatCents(provision.settlementAmount)}` : ''}
      </p>

      <div className="grid grid-cols-2 gap-4">
        {needsAmount && (
          <Field label="Betrag" error={amountError || undefined}>
            <Input
              align="right"
              inputMode="decimal"
              placeholder="0,00"
              value={amount}
              onChange={(e) => {
                setAmount(e.target.value);
                setAmountError('');
              }}
            />
          </Field>
        )}
        <Field label="Datum">
          <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
        </Field>
        {action === 'consumption' && (
          <Field label="Gezahlt von" className="col-span-2">
            <Combobox
              items={options}
              value={paymentAccount || null}
              onValueChange={(account) => setPaymentAccount(account ?? '')}
              placeholder="Konto suchen …"
              emptyText="Kein Konto gefunden."
            />
          </Field>
        )}
      </div>

      <Field label="Begründung" className="mt-4" help={ACTION_HINTS[action]}>
        <Textarea rows={3} value={reason} onChange={(e) => setReason(e.target.value)} />
      </Field>

      <div className="mt-5">
        <BackendError message={error} />
      </div>
    </Dialog>
  );
};

/** Die Bewegungen einer Rückstellung — der Weg vom Erfüllungsbetrag zum Rest. */
const ProvisionMovementsDialog: React.FC<{
  provision: Provision | null;
  onOpenChange: (open: boolean) => void;
}> = ({ provision, onOpenChange }) => (
  <Dialog
    open={provision !== null}
    onOpenChange={onOpenChange}
    title={provision ? `Bewegungen: ${provision.text}` : ''}
    width="max-w-2xl"
  >
    {provision && provision.movements.length === 0 ? (
      <p className="text-body text-ink-muted">Zu dieser Rückstellung ist nichts gebucht.</p>
    ) : (
      <Table density="kompakt">
        <Thead>
          <Tr>
            <Th className="w-32">Art</Th>
            <Th className="w-32">Datum</Th>
            <Th numeric className="w-36">
              Betrag
            </Th>
            <Th className="w-28">Buchung</Th>
            <Th>Begründung</Th>
          </Tr>
        </Thead>
        <Tbody>
          {(provision?.movements ?? []).map((movement) => (
            <Tr key={movement.id}>
              <Td>{MOVEMENT_KIND_LABELS[movement.kind]}</Td>
              <Td className="num">{formatDate(movement.date)}</Td>
              <Td numeric>{formatCents(movement.amount)}</Td>
              <Td code>{movement.entryNumber || '—'}</Td>
              <Td className="text-ink-muted whitespace-normal">{movement.reason || '—'}</Td>
            </Tr>
          ))}
        </Tbody>
      </Table>
    )}
  </Dialog>
);

/**
 * Die Abzinsungssätze der Deutschen Bundesbank. Sie stehen in einer pflegbaren
 * Tabelle und nicht im Code: sie ändern sich monatlich, und ein Programm, das
 * sie mitbringt, wäre einen Monat nach seiner Auslieferung falsch — ohne es zu
 * sagen.
 */
const DiscountRatesSection: React.FC = () => {
  const writeLock = useWriteLock();
  const [months, setMonths] = useState<string[]>([]);
  const [month, setMonth] = useState('');
  const [rates, setRates] = useState<DiscountRate[]>([]);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  const [importPath, setImportPath] = useState('');
  const [importMonth, setImportMonth] = useState('');
  const [importAverage, setImportAverage] = useState('7');

  const [newYears, setNewYears] = useState('');
  const [newRate, setNewRate] = useState('');

  // Fehlende oder unleserliche Eingaben stehen am Feld; die Hinweisfläche
  // darüber bleibt den Ablehnungen des Backends vorbehalten (§10.4).
  const [yearsError, setYearsError] = useState('');
  const [rateError, setRateError] = useState('');
  const [pathError, setPathError] = useState('');
  const [importMonthError, setImportMonthError] = useState('');

  const load = useCallback(async (selected: string) => {
    try {
      const [availableMonths, rows] = await Promise.all([
        Api.getDiscountRateMonths(),
        Api.getDiscountRates(selected),
      ]);
      setMonths(availableMonths);
      setRates(rows);
      setError('');
    } catch (e) {
      setRates([]);
      setError(message(e));
    }
  }, []);

  useEffect(() => {
    void load(month);
  }, [load, month]);

  async function runImport() {
    const missingPath = !importPath.trim();
    const missingMonth = !importMonth.trim();
    if (missingPath || missingMonth) {
      setPathError(missingPath ? 'Der Pfad der Datei fehlt.' : '');
      setImportMonthError(
        missingMonth ? 'Ohne Monat ließe sich kein Satz einem Stichtag zuordnen.' : '',
      );
      return;
    }
    setPathError('');
    setImportMonthError('');
    setBusy(true);
    try {
      const count = await Api.importDiscountRatesCSV(
        importPath,
        importMonth,
        Number.parseInt(importAverage, 10) || 7,
      );
      setImportPath('');
      setMonth(importMonth);
      await load(importMonth);
      toast.success(`${count} Zinssätze übernommen.`);
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  async function addRate() {
    const years = Number.parseInt(newYears, 10);
    const percent = Number.parseFloat(newRate.replace(',', '.'));
    const target = month || importMonth;
    if (!target) {
      // Der Monat wird oben gewählt oder unten eingetragen; gemeldet wird er
      // dort, wo er einzutragen ist.
      setImportMonthError('Zum Satz gehört der Monat seiner Veröffentlichung (JJJJ-MM).');
      return;
    }
    if (!Number.isFinite(years) || !Number.isFinite(percent)) {
      setYearsError(Number.isFinite(years) ? '' : 'Die Restlaufzeit fehlt oder ist nicht lesbar.');
      setRateError(Number.isFinite(percent) ? '' : 'Der Satz fehlt oder ist nicht lesbar.');
      return;
    }
    setImportMonthError('');
    setYearsError('');
    setRateError('');
    setBusy(true);
    try {
      await Api.saveDiscountRates([
        {
          month: target,
          years,
          rateMicros: Math.round(percent * 10000),
          average: Number.parseInt(importAverage, 10) || 7,
        },
      ]);
      setNewYears('');
      setNewRate('');
      await load(target);
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Section
      title="Abzinsungssätze der Deutschen Bundesbank"
      context="Ohne Satz zinst Buchfink nicht ab und erzeugt einen Befund"
      action={
        <HelpPopover label="Erklärung zu den Abzinsungssätzen">
          § 253 Abs. 2 HGB verlangt die Abzinsung mit dem durchschnittlichen Marktzinssatz der
          vergangenen sieben Geschäftsjahre; für Altersversorgungsverpflichtungen sind es zehn. Die
          Sätze veröffentlicht die Deutsche Bundesbank monatlich. Erwartet werden zwei Spalten:
          Restlaufzeit in Jahren und Satz in Prozent.
        </HelpPopover>
      }
    >
      <BackendError message={error} />

      <div className="flex flex-wrap items-end gap-4 mb-5">
        <Field label="Monat" className="w-48">
          <Select
            items={[
              { value: '', label: 'Jüngste Sätze' },
              ...months.map((entry) => ({ value: entry, label: entry })),
            ]}
            value={month}
            onValueChange={(next) => setMonth(next as string)}
            placeholder="Jüngste Sätze"
          />
        </Field>
        <Field label="Mittelung" className="w-48" hint="sieben Jahre, Pensionen zehn">
          <Select
            items={[
              { value: '7', label: 'Sieben Jahre' },
              { value: '10', label: 'Zehn Jahre' },
            ]}
            value={importAverage}
            onValueChange={(next) => setImportAverage(next as string)}
          />
        </Field>
        <Field
          label="Restlaufzeit"
          className="w-32"
          hint="in Jahren"
          error={yearsError || undefined}
        >
          <Input
            align="right"
            inputMode="numeric"
            value={newYears}
            onChange={(e) => {
              setNewYears(e.target.value);
              setYearsError('');
            }}
          />
        </Field>
        <Field label="Satz in Prozent" className="w-40" error={rateError || undefined}>
          <Input
            align="right"
            inputMode="decimal"
            placeholder="1,50"
            value={newRate}
            onChange={(e) => {
              setNewRate(e.target.value);
              setRateError('');
            }}
          />
        </Field>
        <Button
          variant="secondary"
          loading={busy}
          disabled={writeLock.locked}
          title={writeLock.hint}
          onClick={() => void addRate()}
        >
          Satz übernehmen
        </Button>
      </div>

      <div className="flex flex-wrap items-end gap-4 mb-5">
        <Field
          label="Pfad der CSV-Datei"
          className="flex-1 min-w-64"
          help="Die Veröffentlichung der Bundesbank mit Restlaufzeit und Satz."
          error={pathError || undefined}
        >
          <Input
            value={importPath}
            onChange={(e) => {
              setImportPath(e.target.value);
              setPathError('');
            }}
            placeholder="/Pfad/zur/abzinsungssaetze.csv"
          />
        </Field>
        <Field
          label="Monat der Veröffentlichung"
          className="w-56"
          hint="JJJJ-MM"
          error={importMonthError || undefined}
        >
          <Input
            value={importMonth}
            onChange={(e) => {
              setImportMonth(e.target.value);
              setImportMonthError('');
            }}
            placeholder="2026-12"
          />
        </Field>
        <Button
          variant="secondary"
          loading={busy}
          disabled={writeLock.locked}
          title={writeLock.hint}
          onClick={() => void runImport()}
        >
          Datei einlesen
        </Button>
      </div>

      {rates.length === 0 ? (
        <EmptyState
          title="Keine Zinssätze hinterlegt"
          description="Ohne Satz bleibt eine Rückstellung mit mehr als einem Jahr Restlaufzeit unabgezinst."
        />
      ) : (
        <Table density="kompakt">
          <Thead>
            <Tr>
              <Th className="w-32">Monat</Th>
              <Th numeric className="w-32">
                Restlaufzeit
              </Th>
              <Th numeric className="w-32">
                Satz
              </Th>
              <Th className="w-40">Mittelung</Th>
            </Tr>
          </Thead>
          <Tbody>
            {rates.map((rate) => (
              <Tr key={`${rate.month}-${rate.years}-${rate.average}`}>
                <Td className="num">{rate.month}</Td>
                <Td numeric>{`${rate.years} Jahre`}</Td>
                <Td numeric>{formatRateMicros(rate.rateMicros)}</Td>
                <Td className="text-ink-muted">{`${rate.average} Jahre`}</Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      )}
    </Section>
  );
};

const ProvisionsTab: React.FC<TabProps & AccountsProps> = ({ year, accounts }) => {
  const writeLock = useWriteLock();
  const [provisions, setProvisions] = useState<Provision[]>([]);
  const [mirror, setMirror] = useState<ProvisionMirror | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [formOpen, setFormOpen] = useState(false);
  const [changeTarget, setChangeTarget] = useState<Provision | null>(null);
  const [changeAction, setChangeAction] = useState<ProvisionAction>('increase');
  const [movementsOf, setMovementsOf] = useState<Provision | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [rows, nextMirror] = await Promise.all([
        Api.getProvisions(year),
        Api.getProvisionMirror(year),
      ]);
      setProvisions(rows);
      setMirror(nextMirror);
    } catch (e) {
      setError(message(e));
    } finally {
      setLoading(false);
    }
  }, [year]);

  useEffect(() => {
    void load();
  }, [load]);

  if (loading) return <Loading />;

  function openAction(provision: Provision, action: ProvisionAction) {
    setChangeAction(action);
    setChangeTarget(provision);
  }

  return (
    <>
      <Section
        title="Rückstellungen"
        context={`Geschäftsjahr ${year}`}
        divider={false}
        action={
          <div className="flex items-center gap-3">
            <HelpPopover label="Erklärung zu den Rückstellungen">
              § 249 Abs. 1 HGB verlangt Rückstellungen für ungewisse Verbindlichkeiten und drohende
              Verluste; Abs. 2 verbietet alle übrigen. Bewertet wird mit dem Erfüllungsbetrag
              (§ 253 Abs. 1 Satz 2 HGB), bei mehr als einem Jahr Restlaufzeit abgezinst
              (§ 253 Abs. 2 HGB). Aufgelöst wird nur, soweit der Grund entfällt.
            </HelpPopover>
            <Button
              variant="primary"
              icon={<Plus className="w-4 h-4" strokeWidth={1.5} />}
              disabled={writeLock.locked}
              title={writeLock.hint}
              onClick={() => setFormOpen(true)}
            >
              Rückstellung bilden
            </Button>
          </div>
        }
      >
        <BackendError message={error} />

        {provisions.length === 0 ? (
          <EmptyState
            title="Keine Rückstellung erfasst"
            description="Erfasst wird, was am Stichtag dem Grunde nach besteht und der Höhe oder Fälligkeit nach offen ist."
          />
        ) : (
          <Table>
            <Thead>
              <Tr>
                <Th>Text</Th>
                <Th className="w-64">Art</Th>
                <Th className="w-32">Fällig</Th>
                <Th numeric className="w-36">
                  Erfüllungsbetrag
                </Th>
                <Th numeric className="w-36">
                  Bilanzansatz
                </Th>
                <Th className="w-24">Konto</Th>
                <Th className="w-32">Stand</Th>
                <Th className="w-24" aria-label="Aktionen" />
              </Tr>
            </Thead>
            <Tbody>
              {provisions.map((provision) => (
                <Tr key={provision.id}>
                  <Td className="whitespace-normal">{provision.text}</Td>
                  <Td className="whitespace-normal">{PROVISION_KIND_LABELS[provision.kind]}</Td>
                  <Td className="num text-ink-subtle">{formatDate(provision.expectedDate)}</Td>
                  <Td numeric>{formatCents(provision.settlementAmount)}</Td>
                  <Td numeric>
                    {formatCents(provision.discountedAmount)}
                    {provision.discountRateMicros ? (
                      <span className="text-ink-muted">
                        {` · ${formatRateMicros(provision.discountRateMicros)}`}
                      </span>
                    ) : null}
                  </Td>
                  <Td code>{provision.balanceAccount}</Td>
                  <Td>
                    <StatusBadge status={provision.settledOn ? 'ausgeglichen' : 'offen'} />
                  </Td>
                  <Td className="pl-0">
                    <div className="flex justify-end">
                      <Menu
                        trigger={
                          <Button variant="quiet" size="sm">
                            Bearbeiten
                          </Button>
                        }
                      >
                        <MenuGroup label="Bewegungen">
                          <MenuItem onClick={() => openAction(provision, 'increase')}>
                            Zuführen
                          </MenuItem>
                          <MenuItem onClick={() => openAction(provision, 'unwinding')}>
                            Aufzinsen
                          </MenuItem>
                          <MenuItem onClick={() => openAction(provision, 'consumption')}>
                            Verbrauchen
                          </MenuItem>
                          <MenuItem onClick={() => openAction(provision, 'release')}>
                            Auflösen
                          </MenuItem>
                          <MenuItem onClick={() => openAction(provision, 'settle')}>
                            Erledigen
                          </MenuItem>
                        </MenuGroup>
                        <MenuSeparator />
                        <MenuItem onClick={() => setMovementsOf(provision)}>
                          Bewegungen ansehen
                        </MenuItem>
                      </Menu>
                    </div>
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>

      <Section
        title="Rückstellungsspiegel"
        context="Anfangsbestand, Zuführung, Verbrauch, Auflösung, Aufzinsung, Endbestand"
        action={
          <HelpPopover label="Erklärung zum Rückstellungsspiegel">
            Der Spiegel ist Bestandteil des Anhangs. Er geht per Definition auf: Endbestand ist
            Anfangsbestand zuzüglich Zuführung und Aufzinsung, abzüglich Verbrauch und Auflösung.
            Ohne die einzelnen Spalten ließe sich die Entwicklung aus dem Endbestand nicht ablesen.
          </HelpPopover>
        }
      >
        {!mirror || mirror.rows.length === 0 ? (
          <EmptyState
            title="Kein Bestand an Rückstellungen"
            description="Der Spiegel entsteht aus den Bewegungen des Geschäftsjahres."
          />
        ) : (
          <Table>
            <Thead>
              <Tr>
                <Th>Art</Th>
                <Th className="w-24">Konto</Th>
                <Th numeric>Anfang</Th>
                <Th numeric>Zuführung</Th>
                <Th numeric>Verbrauch</Th>
                <Th numeric>Auflösung</Th>
                <Th numeric>Aufzinsung</Th>
                <Th numeric>Ende</Th>
              </Tr>
            </Thead>
            <Tbody>
              {mirror.rows.map((row) => (
                <Tr key={`${row.kind}-${row.account}`}>
                  <Td className="whitespace-normal">{row.label}</Td>
                  <Td code>{row.account}</Td>
                  <Td numeric>{formatCents(row.opening)}</Td>
                  <Td numeric>{formatCents(row.additions)}</Td>
                  <Td numeric>{formatCents(row.used)}</Td>
                  <Td numeric>{formatCents(row.released)}</Td>
                  <Td numeric>{formatCents(row.unwinding)}</Td>
                  <Td numeric>{formatCents(row.closing)}</Td>
                </Tr>
              ))}
              <Tr variant="sum">
                <Td>Summe</Td>
                <Td />
                <Td numeric>{formatCents(mirror.total.opening)}</Td>
                <Td numeric>{formatCents(mirror.total.additions)}</Td>
                <Td numeric>{formatCents(mirror.total.used)}</Td>
                <Td numeric>{formatCents(mirror.total.released)}</Td>
                <Td numeric>{formatCents(mirror.total.unwinding)}</Td>
                <Td numeric>{formatCents(mirror.total.closing)}</Td>
              </Tr>
            </Tbody>
          </Table>
        )}
      </Section>

      <DiscountRatesSection />

      <ProvisionFormDialog
        accounts={accounts}
        open={formOpen}
        year={year}
        onOpenChange={setFormOpen}
        onBooked={load}
      />
      <ProvisionChangeDialog
        accounts={accounts}
        provision={changeTarget}
        action={changeAction}
        year={year}
        onOpenChange={(open) => !open && setChangeTarget(null)}
        onDone={load}
      />
      <ProvisionMovementsDialog
        provision={movementsOf}
        onOpenChange={(open) => !open && setMovementsOf(null)}
      />
    </>
  );
};

// -------------------------------------------------------------------------
// Vorräte
// -------------------------------------------------------------------------

/**
 * Der Inventurwert wird erfasst und nicht gerechnet: § 240 HGB verlangt die
 * körperliche Bestandsaufnahme, und was auf dem Konto steht, ist der Wert der
 * gebuchten Zu- und Abgänge. Die Differenz zwischen beiden ist die
 * Bestandsveränderung, die der Abschluss bucht.
 */
const InventoryDialog: React.FC<{
  account: InventoryAccount | null;
  year: number;
  onOpenChange: (open: boolean) => void;
  onBooked: () => Promise<void>;
}> = ({ account, year, onOpenChange, onBooked }) => {
  const writeLock = useWriteLock();
  const [amount, setAmount] = useState('');
  const [countedOn, setCountedOn] = useState('');
  const [method, setMethod] = useState('');
  const [receiptId, setReceiptId] = useState('');
  const [receipts, setReceipts] = useState<Receipt[]>([]);
  const [preview, setPreview] = useState<InventoryPreview | null>(null);
  const [previewError, setPreviewError] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!account) return;
    setAmount('');
    setCountedOn('');
    setMethod('');
    setReceiptId('');
    setPreview(null);
    setPreviewError('');
    setError('');
    Api.getReceipts('')
      .then((rows) => setReceipts(rows.filter((receipt) => receipt.status !== 'discarded')))
      .catch((e) => setError(message(e)));
  }, [account]);

  const request = useMemo(() => {
    if (!account) return null;
    const value = parseCents(amount);
    if (value === null || value < 0) return null;
    return {
      fiscalYear: year,
      account: account.account,
      amount: value,
      countedOn,
      method,
      receiptId: Number.parseInt(receiptId, 10) || 0,
    };
  }, [account, amount, countedOn, method, receiptId, year]);

  useEffect(() => {
    if (!request) {
      setPreview(null);
      return;
    }
    let cancelled = false;
    const timer = window.setTimeout(() => {
      Api.previewInventory(request)
        .then((result) => {
          if (cancelled) return;
          setPreview(result);
          setPreviewError('');
        })
        .catch((e) => {
          if (cancelled) return;
          setPreview(null);
          setPreviewError(message(e));
        });
    }, 300);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [request]);

  async function submit() {
    if (!request) return;
    setBusy(true);
    setError('');
    try {
      await Api.bookInventory(request);
      onOpenChange(false);
      await onBooked();
      toast.success('Bestandsveränderung gebucht.');
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open={account !== null}
      onOpenChange={onOpenChange}
      title={account ? `Inventurwert ${account.account} ${account.accountName}` : ''}
      width="max-w-2xl"
      footer={
        <>
          <Button variant="secondary" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            loading={busy}
            disabled={writeLock.locked}
            title={writeLock.hint}
            onClick={() => void submit()}
          >
            Bestandsveränderung buchen
          </Button>
        </>
      }
    >
      <div className="grid grid-cols-2 gap-4">
        <Field
          label="Inventurwert zum Stichtag"
          help="Der bewertete Wert: Verbrauchsfolge und Niederstwert berücksichtigt der Anwender."
        >
          <Input
            align="right"
            inputMode="decimal"
            placeholder="0,00"
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
          />
        </Field>
        <Field label="Tag der Aufnahme">
          <Input type="date" value={countedOn} onChange={(e) => setCountedOn(e.target.value)} />
        </Field>
        <Field
          label="Aufnahmeverfahren"
          help="§ 241 HGB lässt mehrere zu: Stichtags-, permanente oder Stichprobeninventur."
        >
          <Input
            value={method}
            onChange={(e) => setMethod(e.target.value)}
            placeholder="Stichtagsinventur"
          />
        </Field>
        <Field label="Inventurliste" help="Die Aufnahme selbst ist der Beleg und deshalb Pflicht.">
          {/* Der Belegspeicher wächst über die Jahre; eine Liste, die man
              durchsuchen muss, ist eine Combobox und kein Auswahlfeld (§10.4).
              Gesucht wird über Belegnummer und Datum — die Beschriftung ist der
              Suchtext. */}
          <Combobox
            items={receipts.map((receipt) => ({
              value: String(receipt.id),
              label: `${receipt.receiptNumber} · ${formatDate(receipt.createdAt)}`,
            }))}
            value={receiptId || null}
            onValueChange={(next) => setReceiptId(next ?? '')}
            placeholder="Beleg suchen …"
            emptyText="Kein Beleg gefunden."
          />
        </Field>
      </div>

      <div className="mt-6">
        {previewError ? (
          <BackendError message={previewError} />
        ) : preview ? (
          <>
            <StatRow className="mb-5">
              <Stat label="Buchwert" value={formatCents(preview.bookValue)} context="Vor der Abschlussbuchung" />
              <Stat label="Inventurwert" value={formatCents(preview.counted)} context="Aus der Aufnahme" />
              <Stat
                label="Bestandsveränderung"
                value={formatCents(preview.change)}
                context={`Gegenkonto ${preview.changeAccount}`}
                tone={preview.change === 0 ? 'neutral' : preview.change > 0 ? 'positive' : 'negative'}
              />
            </StatRow>
            <PostingLines lines={preview.lines} />
          </>
        ) : (
          <p className="text-body text-ink-muted">
            Betrag, Tag der Aufnahme, Verfahren und Beleg ergeben die Bestandsveränderung.
          </p>
        )}
      </div>

      <div className="mt-5">
        <BackendError message={error} />
      </div>
    </Dialog>
  );
};

const InventoryTab: React.FC<TabProps> = ({ year }) => {
  const writeLock = useWriteLock();
  const [overview, setOverview] = useState<InventoryOverview | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [target, setTarget] = useState<InventoryAccount | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      setOverview(await Api.getInventoryAccounts(year));
    } catch (e) {
      setOverview(null);
      setError(message(e));
    } finally {
      setLoading(false);
    }
  }, [year]);

  useEffect(() => {
    void load();
  }, [load]);

  if (loading) return <Loading />;

  const accountRows = overview?.accounts ?? [];

  return (
    <>
      <Section
        title="Vorräte"
        context={overview ? `Stichtag ${formatDate(overview.cutoff)}` : `Geschäftsjahr ${year}`}
        divider={false}
        action={
          <HelpPopover label="Erklärung zur Inventur">
            {overview?.note ||
              '§ 240 HGB verlangt zum Ende jedes Geschäftsjahres eine Bestandsaufnahme. Buchfink bewertet nicht; der erfasste Wert ist der bewertete Wert.'}
          </HelpPopover>
        }
      >
        <BackendError message={error} />

        {accountRows.length === 0 ? (
          <EmptyState
            title="Keine Vorratskonten mit Bestand"
            description="Erfasst wird der Inventurwert je Vorratskonto der Kontenklasse 1."
          />
        ) : (
          <Table>
            <Thead>
              <Tr>
                <Th className="w-24">Konto</Th>
                <Th>Bezeichnung</Th>
                <Th className="w-48">Gruppe</Th>
                <Th className="w-24">Gegenkonto</Th>
                <Th numeric className="w-36">
                  Buchwert
                </Th>
                <Th numeric className="w-36">
                  Inventurwert
                </Th>
                <Th className="w-44" aria-label="Aktionen" />
              </Tr>
            </Thead>
            <Tbody>
              {accountRows.map((row) => (
                <Tr key={row.account}>
                  <Td code>{row.account}</Td>
                  <Td className="whitespace-normal">{row.accountName}</Td>
                  <Td className="text-ink-muted whitespace-normal">{row.group}</Td>
                  <Td code>{row.changeAccount}</Td>
                  <Td numeric>{formatCents(row.bookValue)}</Td>
                  <Td numeric>
                    {row.booked ? formatCents(row.counted) : '—'}
                    {row.countedAt && (
                      <span className="text-ink-muted">{` · ${formatDate(row.countedAt)}`}</span>
                    )}
                  </Td>
                  <Td className="pl-0">
                    <div className="flex justify-end">
                      <Button
                        variant="quiet"
                        size="sm"
                        disabled={writeLock.locked}
                        title={writeLock.hint}
                        onClick={() => setTarget(row)}
                      >
                        {row.booked ? 'Erneut erfassen' : 'Inventurwert erfassen'}
                      </Button>
                    </div>
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>

      <InventoryDialog
        account={target}
        year={year}
        onOpenChange={(open) => !open && setTarget(null)}
        onBooked={load}
      />
    </>
  );
};

// -------------------------------------------------------------------------
// Umsatzsteuer-Verrechnung
// -------------------------------------------------------------------------

const VatSettlementTab: React.FC<TabProps> = ({ year }) => {
  const writeLock = useWriteLock();
  const [settlement, setSettlement] = useState<VatSettlement | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      setSettlement(await Api.previewVatSettlement(year));
    } catch (e) {
      setSettlement(null);
      setError(message(e));
    } finally {
      setLoading(false);
    }
  }, [year]);

  useEffect(() => {
    void load();
  }, [load]);

  async function book() {
    setBusy(true);
    try {
      const entry = await Api.bookVatSettlement(year);
      await load();
      toast.success(`Umsatzsteuer verrechnet: Buchung ${entry.entryNumber}.`);
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  if (loading) return <Loading />;

  return (
    <>
      <Section
        title="Umsatzsteuer-Verrechnung"
        context={settlement ? `Stichtag ${formatDate(settlement.cutoff)}` : `Geschäftsjahr ${year}`}
        divider={false}
        action={
          <div className="flex items-center gap-3">
            <HelpPopover label="Erklärung zur Umsatzsteuer-Verrechnung">
              Zum Bilanzstichtag stehen Vorsteuer, Umsatzsteuer und die geleisteten Vorauszahlungen
              nebeneinander. Die Verrechnung führt sie zu einem Saldo zusammen: einer Zahllast auf
              dem Konto der Umsatzsteuer des Vorjahres oder einer Forderung. Das Ergebnis entspricht
              der Umsatzsteuer-Jahreserklärung.
            </HelpPopover>
            {/* Kein Rückfragedialog: die Verrechnung ist eine gewöhnliche
                Buchung und per Generalumkehr zurückzunehmen. Die Vorschau
                darüber ist die Ankündigung, dieser Knopf die Freigabe (§8.2);
                ein zweiter Klick auf „Wirklich?" fragte nach nichts. */}
            <Button
              variant="primary"
              loading={busy}
              disabled={!settlement || settlement.lines.length === 0 || busy || writeLock.locked}
              title={
                writeLock.hint ??
                (settlement && settlement.lines.length === 0
                  ? 'Die Steuerkonten stehen bereits auf null'
                  : undefined)
              }
              onClick={() => void book()}
            >
              Verrechnung buchen
            </Button>
          </div>
        }
      >
        <BackendError message={error} />

        {!settlement ? (
          <EmptyState
            title="Die Verrechnung konnte nicht gerechnet werden"
            description="Ohne Geschäftsjahr gibt es keinen Stichtag, zu dem verrechnet würde."
          />
        ) : (
          <>
            {/* Der Satz, der bisher im Rückfragedialog stand: er gehört in die
                Ansicht, weil er auch dann gilt, wenn niemand den Knopf drückt. */}
            <Notice
              className="mb-6"
              text={`Gebucht wird zum ${formatDate(settlement.bookingDate)}. Ein zweiter Lauf ist gesperrt; zurück geht es nur über eine Generalumkehr.`}
            />

            <StatRow className="mb-6">
              <Stat label="Vorsteuer" value={formatCents(settlement.inputTax)} context="Kontenklasse 1400 ff." />
              <Stat label="Umsatzsteuer" value={formatCents(settlement.outputTax)} context="Kontenklasse 3800 ff." />
              <Stat
                label="Vorauszahlungen"
                value={formatCents(settlement.prepaid)}
                context="Geleistet im Geschäftsjahr"
              />
              <Stat
                label={settlement.refund > 0 ? 'Erstattung' : 'Zahllast'}
                value={formatCents(settlement.refund > 0 ? settlement.refund : settlement.payable)}
                context={settlement.refund > 0 ? 'Forderung an das Finanzamt' : 'Verbindlichkeit'}
                tone={settlement.refund > 0 ? 'positive' : 'neutral'}
              />
            </StatRow>

            {settlement.rows.length > 0 && (
              <div className="mb-6">
                <h3 className="text-label text-ink-muted mb-2">Salden der Steuerkonten</h3>
                <Table density="kompakt">
                  <Thead>
                    <Tr>
                      <Th className="w-24">Konto</Th>
                      <Th>Bezeichnung</Th>
                      <Th numeric className="w-40">
                        Saldo
                      </Th>
                    </Tr>
                  </Thead>
                  <Tbody>
                    {settlement.rows.map((row) => (
                      <Tr key={row.account}>
                        <Td code>{row.account}</Td>
                        <Td className="whitespace-normal">{row.accountName}</Td>
                        <Td numeric>{formatCents(row.balance)}</Td>
                      </Tr>
                    ))}
                  </Tbody>
                </Table>
              </div>
            )}

            <h3 className="text-label text-ink-muted mb-2">Buchungssatz zum Stichtag</h3>
            <PostingLines lines={settlement.lines} />
          </>
        )}
      </Section>
    </>
  );
};

// -------------------------------------------------------------------------
// Steuerrückstellung
// -------------------------------------------------------------------------

const TaxProvisionTab: React.FC<TabProps> = ({ year }) => {
  const writeLock = useWriteLock();
  const [preview, setPreview] = useState<TaxProvisionPreview | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [income, setIncome] = useState('');
  const [trade, setTrade] = useState('');
  const [reason, setReason] = useState('');
  const [busy, setBusy] = useState(false);

  // Die Beträge der Felder — sie und nicht der Vorschlag werden gebucht.
  const chosenIncome = parseCents(income) ?? 0;
  const chosenTrade = parseCents(trade) ?? 0;
  /**
   * Der Buchungssatz unter der Vorschau stammt aus `PreviewTaxProvision(year)`
   * und kennt die geänderten Beträge nicht. Solange sie vom Vorschlag
   * abweichen, zeigt er also nicht, was gebucht würde — das muss dabeistehen,
   * sonst gäbe die Ansicht eine Vorschau, die keine ist.
   */
  const deviates =
    preview !== null &&
    (chosenIncome !== preview.incomeProvision || chosenTrade !== preview.tradeProvision);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const result = await Api.previewTaxProvision(year);
      setPreview(result);
      setIncome(result ? formatCentsPlain(result.incomeProvision) : '');
      setTrade(result ? formatCentsPlain(result.tradeProvision) : '');
    } catch (e) {
      setPreview(null);
      setError(message(e));
    } finally {
      setLoading(false);
    }
  }, [year]);

  useEffect(() => {
    void load();
  }, [load]);

  async function book() {
    setBusy(true);
    try {
      await Api.bookTaxProvision({
        fiscalYear: year,
        incomeProvision: chosenIncome,
        tradeProvision: chosenTrade,
        reason,
      });
      await load();
      toast.success('Steuerrückstellung gebildet.');
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  if (loading) return <Loading />;

  return (
    <>
      <Section
        title="Steuerrückstellung"
        context={preview ? `Stichtag ${formatDate(preview.cutoff)}` : `Geschäftsjahr ${year}`}
        divider={false}
        action={
          <div className="flex items-center gap-3">
            <HelpPopover label="Erklärung zur Steuerrückstellung">
              Körperschaftsteuer 15 % (§ 23 Abs. 1 KStG), Solidaritätszuschlag 5,5 % darauf (§ 4
              SolZG) und Gewerbesteuer aus Messzahl 3,5 % (§ 11 Abs. 2 GewStG) mal dem Hebesatz der
              Gemeinde. Verlustvorträge, Hinzurechnungen und Kürzungen kennt Buchfink nicht — die
              Beträge sind deshalb änderbar.
            </HelpPopover>
            {/* Wie bei der Verrechnung: die Rückstellung ist umkehrbar, die
                Vorschau darüber ist die Ankündigung und dieser Knopf die
                Freigabe (§8.2). */}
            <Button
              variant="primary"
              loading={busy}
              disabled={!preview || busy || writeLock.locked}
              title={writeLock.hint}
              onClick={() => void book()}
            >
              Rückstellung buchen
            </Button>
          </div>
        }
      >
        <BackendError message={error} />

        {!preview ? (
          <EmptyState
            title="Die Steuerrückstellung konnte nicht gerechnet werden"
            description="Sie setzt ein Ergebnis voraus, und das steht erst nach den übrigen Abschlussbuchungen."
          />
        ) : (
          <>
            {preview.warning && <Notice text={preview.warning} className="mb-6" />}

            <StatRow className="mb-6">
              <Stat
                label="Ergebnis vor Steuern"
                value={formatCents(preview.input.profitBeforeTax)}
                context="Aus der Gewinn- und Verlustrechnung"
              />
              <Stat
                label="Zu versteuern"
                value={formatCents(preview.taxableIncome)}
                context="Ergebnis zuzüglich nicht abziehbarer Aufwendungen"
              />
              <Stat
                label="Körperschaftsteuer und Soli"
                value={formatCents(preview.incomeProvision)}
                context={`Vorauszahlungen ${formatCents(preview.input.prepaidCorporate)}`}
              />
              <Stat
                label="Gewerbesteuer"
                value={formatCents(preview.tradeProvision)}
                context={`Hebesatz ${preview.input.tradeTaxRatePercent} %`}
              />
            </StatRow>

            <h3 className="text-label text-ink-muted mb-2">Rechenweg</h3>
            <Table density="kompakt">
              <Thead>
                <Tr>
                  <Th>Schritt</Th>
                  <Th numeric className="w-44">
                    Betrag
                  </Th>
                  <Th className="w-72">Grundlage</Th>
                </Tr>
              </Thead>
              <Tbody>
                <Tr>
                  <Td>Ergebnis vor Steuern</Td>
                  <Td numeric>{formatCents(preview.input.profitBeforeTax)}</Td>
                  <Td className="text-ink-muted">§ 275 HGB</Td>
                </Tr>
                <Tr>
                  <Td>Nicht abziehbare Betriebsausgaben</Td>
                  <Td numeric>{formatCents(preview.input.nonDeductible)}</Td>
                  <Td className="text-ink-muted">§ 4 Abs. 5 EStG, § 10 KStG</Td>
                </Tr>
                <Tr>
                  <Td>Zu versteuerndes Einkommen</Td>
                  <Td numeric>{formatCents(preview.taxableIncome)}</Td>
                  <Td className="text-ink-muted">§ 7 Abs. 1 KStG</Td>
                </Tr>
                <Tr>
                  <Td>Körperschaftsteuer</Td>
                  <Td numeric>{formatCents(preview.corporateTax)}</Td>
                  <Td className="text-ink-muted">§ 23 Abs. 1 KStG</Td>
                </Tr>
                <Tr>
                  <Td>Solidaritätszuschlag</Td>
                  <Td numeric>{formatCents(preview.solidarity)}</Td>
                  <Td className="text-ink-muted">§ 4 SolZG</Td>
                </Tr>
                <Tr>
                  <Td>Gewerbeertrag, abgerundet</Td>
                  <Td numeric>{formatCents(preview.tradeIncome)}</Td>
                  <Td className="text-ink-muted">§ 11 Abs. 1 Satz 3 GewStG</Td>
                </Tr>
                <Tr>
                  <Td>Steuermessbetrag</Td>
                  <Td numeric>{formatCents(preview.tradeBase)}</Td>
                  <Td className="text-ink-muted">§ 11 Abs. 2 GewStG</Td>
                </Tr>
                <Tr>
                  <Td>Gewerbesteuer</Td>
                  <Td numeric>{formatCents(preview.tradeTax)}</Td>
                  <Td className="text-ink-muted">§ 16 GewStG</Td>
                </Tr>
                <Tr>
                  <Td>Vorauszahlungen Körperschaftsteuer</Td>
                  <Td numeric>{formatCents(preview.input.prepaidCorporate)}</Td>
                  <Td className="text-ink-muted">Gebucht im Geschäftsjahr</Td>
                </Tr>
                <Tr>
                  <Td>Vorauszahlungen Gewerbesteuer</Td>
                  <Td numeric>{formatCents(preview.input.prepaidTrade)}</Td>
                  <Td className="text-ink-muted">Gebucht im Geschäftsjahr</Td>
                </Tr>
                <Tr variant="sum">
                  <Td>Rückstellung</Td>
                  <Td numeric>{formatCents(preview.incomeProvision + preview.tradeProvision)}</Td>
                  <Td className="text-ink-muted">{preview.ratesUsed}</Td>
                </Tr>
              </Tbody>
            </Table>

            {(preview.incomeRefund > 0 || preview.tradeRefund > 0) && (
              <Notice
                className="mt-6"
                text={`Überzahlung: ${formatCents(preview.incomeRefund + preview.tradeRefund)} sind eine Forderung und keine Rückstellung; Buchfink bucht sie nicht.`}
              />
            )}

            <div className="mt-6 grid grid-cols-2 gap-4">
              <Field
                label="Körperschaftsteuer und Solidaritätszuschlag"
                hint="änderbar, der Vorschlag kennt keine Verlustvorträge"
              >
                <Input
                  align="right"
                  inputMode="decimal"
                  value={income}
                  onChange={(e) => setIncome(e.target.value)}
                />
              </Field>
              <Field label="Gewerbesteuer" hint="änderbar, ohne Hinzurechnungen und Kürzungen">
                <Input
                  align="right"
                  inputMode="decimal"
                  value={trade}
                  onChange={(e) => setTrade(e.target.value)}
                />
              </Field>
            </div>

            <Field
              label="Begründung"
              optional
              className="mt-4"
              help="Leer heißt: die Erklärung der Rechnung wird festgehalten."
            >
              <Textarea rows={2} value={reason} onChange={(e) => setReason(e.target.value)} />
            </Field>

            <div className="mt-6">
              <h3 className="text-label text-ink-muted mb-2">Buchungssatz zum Stichtag</h3>
              {deviates ? (
                <Notice
                  className="mb-3"
                  text={
                    `Der Buchungssatz zeigt den Vorschlag über ${formatCents(preview.incomeProvision + preview.tradeProvision)}. ` +
                    `Gebucht werden die Beträge aus den Feldern: ${formatCents(chosenIncome + chosenTrade)} ` +
                    `zum ${formatDate(preview.cutoff)}, aufgeteilt in Körperschaftsteuer und Soli ` +
                    `(${formatCents(chosenIncome)}) und Gewerbesteuer (${formatCents(chosenTrade)}).`
                  }
                />
              ) : (
                <Notice
                  className="mb-3"
                  text={`Gebucht wird zum ${formatDate(preview.cutoff)}. Ein zweiter Lauf ist gesperrt; zurück geht es nur über eine Generalumkehr.`}
                />
              )}
              <PostingLines lines={preview.lines} />
            </div>
          </>
        )}
      </Section>
    </>
  );
};

// -------------------------------------------------------------------------
// Ergebnisverwendung
// -------------------------------------------------------------------------

/**
 * Der Beschluss gehört zum Jahr, dessen Ergebnis verwendet wird — gebucht wird
 * er im Folgejahr. Deshalb zeigt diese Ansicht im Geschäftsjahr der Kopfzeile
 * die Verwendung des Vorjahres: § 252 Abs. 1 Nr. 1 HGB verbietet, die
 * Eröffnungsbilanz nachträglich zu ändern, und der Beschluss fällt nach dem
 * Stichtag.
 */
const AppropriationTab: React.FC<TabProps> = ({ year }) => {
  const writeLock = useWriteLock();
  const usedYear = year - 1;
  const [stored, setStored] = useState<Appropriation | null>(null);
  const [preview, setPreview] = useState<AppropriationPreview | null>(null);
  const [previewError, setPreviewError] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [decisionDate, setDecisionDate] = useState('');
  const [text, setText] = useState('');
  const [legalReserve, setLegalReserve] = useState('');
  const [otherReserves, setOtherReserves] = useState('');
  const [distribution, setDistribution] = useState('');
  // Das Beschlussdokument ist freiwillig (§ 42a GmbHG verlangt keine Form),
  // aber ohne Auswahlfeld wäre es gar nicht zu hinterlegen.
  const [receiptId, setReceiptId] = useState('');
  const [receipts, setReceipts] = useState<Receipt[]>([]);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const existing = await Api.getAppropriation(usedYear);
      setStored(existing);
      if (existing) {
        setDecisionDate(existing.decisionDate);
        setText(existing.text ?? '');
        setLegalReserve(formatCentsPlain(existing.legalReserve));
        setOtherReserves(formatCentsPlain(existing.otherReserves));
        setDistribution(formatCentsPlain(existing.distribution));
        setReceiptId(existing.receiptId ? String(existing.receiptId) : '');
      }
      // Belege für das Beschlussdokument: ein verworfener Beleg ist keiner.
      try {
        const rows = await Api.getReceipts('');
        setReceipts(rows.filter((receipt) => receipt.status !== 'discarded'));
      } catch {
        setReceipts([]);
      }
    } catch (e) {
      setError(message(e));
    } finally {
      setLoading(false);
    }
  }, [usedYear]);

  useEffect(() => {
    void load();
  }, [load]);

  const request = useMemo(
    () => ({
      decisionDate,
      text,
      legalReserve: parseCents(legalReserve) ?? 0,
      otherReserves: parseCents(otherReserves) ?? 0,
      distribution: parseCents(distribution) ?? 0,
      receiptId: Number.parseInt(receiptId, 10) || undefined,
    }),
    [decisionDate, text, legalReserve, otherReserves, distribution, receiptId],
  );

  useEffect(() => {
    let cancelled = false;
    const timer = window.setTimeout(() => {
      Api.previewAppropriation(usedYear, request)
        .then((result) => {
          if (cancelled) return;
          setPreview(result);
          setPreviewError('');
        })
        .catch((e) => {
          if (cancelled) return;
          setPreview(null);
          setPreviewError(message(e));
        });
    }, 300);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [usedYear, request]);

  async function book() {
    setBusy(true);
    try {
      await Api.bookAppropriation(usedYear, request);
      await load();
      toast.success(`Ergebnisverwendung ${usedYear} festgehalten.`);
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  if (loading) return <Loading />;

  return (
    <>
      <Section
        title={`Ergebnisverwendung ${usedYear}`}
        context={`Beschlossen und gebucht im Geschäftsjahr ${year}`}
        divider={false}
        action={
          <div className="flex items-center gap-3">
            <HelpPopover label="Erklärung zur Ergebnisverwendung">
              Über die Verwendung des Ergebnisses beschließen die Gesellschafter (§ 29 GmbHG, § 42a
              Abs. 2 GmbHG). Der Saldenvortrag bringt das Ergebnis zunächst unverwendet auf das
              Vortragskonto; erst der Beschluss verteilt es. Die Unternehmergesellschaft muss ein
              Viertel des Jahresüberschusses in die gesetzliche Rücklage einstellen, solange das
              Stammkapital unter 25.000 Euro liegt (§ 5a Abs. 3 GmbHG).
            </HelpPopover>
            {/* Der Beschluss ist umkehrbar; die Vorschau des Buchungssatzes
                unter den Feldern ist die Ankündigung (§8.2). */}
            <Button
              variant="primary"
              loading={busy}
              disabled={!preview || busy || writeLock.locked}
              title={writeLock.hint}
              onClick={() => void book()}
            >
              Beschluss buchen
            </Button>
          </div>
        }
      >
        <BackendError message={error} />

        {/* Welches Jahr hier verwendet wird, folgt aus der Kopfzeile — das
            sieht man dem Reiter nicht an. Ohne diesen Satz sucht, wer den
            Schritt „Saldenvortrag und Ergebnisverwendung" des Jahres X offen
            findet, hier vergeblich nach dem Beschluss für X. */}
        <Notice
          className="mb-6"
          text={
            `Dieser Reiter zeigt die Verwendung des Ergebnisses ${usedYear}, weil der Beschluss im ` +
            `Geschäftsjahr ${year} der Kopfzeile gefasst und gebucht wird. Für das Ergebnis ${year} ` +
            `stellen Sie die Kopfzeile auf ${year + 1}.`
          }
        />

        {stored && (
          <Notice
            className="mb-6"
            text={`Für ${usedYear} ist am ${formatDate(stored.decisionDate)} bereits ein Beschluss erfasst; ein zweiter verlangt zuvor die Generalumkehr seiner Buchung.`}
          />
        )}

        <StatRow className="mb-6">
          <Stat
            label="Verwendbar"
            value={formatCents(preview?.netIncome ?? 0)}
            context="Saldo des Vortragskontos"
          />
          <Stat
            label={`Jahresüberschuss ${usedYear}`}
            value={formatCents(preview?.yearResult ?? 0)}
            context="Ohne Vorträge früherer Jahre"
          />
          <Stat
            label="Pflichtrücklage"
            value={formatCents(preview?.requiredLegalReserve ?? 0)}
            context="Ein Viertel, § 5a Abs. 3 GmbHG"
          />
          <Stat
            label="Vortrag auf neue Rechnung"
            value={formatCents(preview?.appropriation.carryForward ?? 0)}
            context="Bleibt ohne Buchung stehen"
          />
        </StatRow>

        <div className="grid grid-cols-2 gap-4">
          <Field label="Datum des Beschlusses" help="Er wird an seinem eigenen Datum gebucht.">
            <Input
              type="date"
              value={decisionDate}
              onChange={(e) => setDecisionDate(e.target.value)}
            />
          </Field>
          <Field label="Beschlusstext" optional>
            <Input
              value={text}
              onChange={(e) => setText(e.target.value)}
              placeholder="Gesellschafterbeschluss vom …"
            />
          </Field>
          <Field label="Gesetzliche Rücklage">
            <Input
              align="right"
              inputMode="decimal"
              placeholder="0,00"
              value={legalReserve}
              onChange={(e) => setLegalReserve(e.target.value)}
            />
          </Field>
          <Field label="Andere Gewinnrücklagen">
            <Input
              align="right"
              inputMode="decimal"
              placeholder="0,00"
              value={otherReserves}
              onChange={(e) => setOtherReserves(e.target.value)}
            />
          </Field>
          <Field
            label="Ausschüttung"
            help="Brutto; Kapitalertragsteuer und Solidaritätszuschlag werden einbehalten."
          >
            <Input
              align="right"
              inputMode="decimal"
              placeholder="0,00"
              value={distribution}
              onChange={(e) => setDistribution(e.target.value)}
            />
          </Field>
          <Field
            label="Beschlussdokument"
            optional
            help="Das Protokoll der Gesellschafterversammlung, falls es als Beleg abgelegt ist. § 42a GmbHG verlangt keine Form; der Beschluss gilt auch ohne Dokument."
          >
            <Combobox
              items={receipts.map((receipt) => ({
                value: String(receipt.id),
                label: `${receipt.receiptNumber} · ${formatDate(receipt.createdAt)}`,
              }))}
              value={receiptId || null}
              onValueChange={(next) => setReceiptId(next ?? '')}
              placeholder="Beleg suchen …"
              emptyText="Kein Beleg gefunden."
            />
          </Field>
        </div>

        <div className="mt-6">
          {previewError ? (
            <BackendError message={previewError} />
          ) : preview ? (
            <>
              {preview.warnings.map((warning) => (
                <Notice key={warning} text={warning} className="mb-5" />
              ))}

              {preview.appropriation.withholdingTax > 0 && (
                <div className="mb-6">
                  <StatRow>
                    <Stat
                      label="Kapitalertragsteuer"
                      value={formatCents(preview.appropriation.withholdingTax)}
                      context="25 %, § 43a Abs. 1 Satz 1 Nr. 1 EStG"
                    />
                    <Stat
                      label="Solidaritätszuschlag darauf"
                      value={formatCents(preview.appropriation.solidarityOnWithholding)}
                      context="5,5 %, anzumelden bis zum 10. des Folgemonats"
                    />
                  </StatRow>
                </div>
              )}

              <h3 className="text-label text-ink-muted mb-2">
                {`Buchungssatz zum ${formatDate(preview.bookingDate)}`}
              </h3>
              <PostingLines lines={preview.lines} />
            </>
          ) : (
            <p className="text-body text-ink-muted">
              Das Beschlussdatum entscheidet über das Buchungsjahr.
            </p>
          )}
        </div>
      </Section>
    </>
  );
};

// -------------------------------------------------------------------------
// Anhang, Verzeichnis und Überleitung
// -------------------------------------------------------------------------

/** Ein Abschnitt des Anhangs: Freitext, den kein Programm errechnen kann. */
const NotesSectionEditor: React.FC<{
  entry: NotesSectionText;
  year: number;
  onSaved: (texts: NotesSectionText[]) => void;
}> = ({ entry, year, onSaved }) => {
  const writeLock = useWriteLock();
  const [text, setText] = useState(entry.text);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    setText(entry.text);
  }, [entry.text]);

  async function save() {
    setBusy(true);
    setError('');
    try {
      onSaved(await Api.saveNotesText(year, entry.section, text));
      toast.success(`${entry.label} gespeichert.`);
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mb-6">
      <Field label={entry.label} hint={entry.basis} help={entry.hint}>
        <Textarea rows={4} value={text} onChange={(e) => setText(e.target.value)} />
      </Field>
      <div className="mt-2 flex justify-end">
        <Button
          variant="secondary"
          size="sm"
          loading={busy}
          disabled={text === entry.text || writeLock.locked}
          title={writeLock.hint ?? (text === entry.text ? 'Nichts geändert' : undefined)}
          onClick={() => void save()}
        >
          Übernehmen
        </Button>
      </div>
      <BackendError message={error} />
    </div>
  );
};

const NotesTab: React.FC<TabProps> = ({ year }) => {
  const [texts, setTexts] = useState<NotesSectionText[]>([]);
  const [register, setRegister] = useState<TaxElectionRegister | null>(null);
  const [reconciliation, setReconciliation] = useState<Reconciliation | null>(null);
  const [legacy, setLegacy] = useState<LegacySpecialDepreciationNotice | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [exporting, setExporting] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [nextTexts, nextRegister, nextReconciliation, nextLegacy] = await Promise.all([
        Api.getNotesTexts(year),
        Api.getTaxElectionRegister(year),
        Api.getReconciliation(year),
        Api.getLegacySpecialDepreciations(),
      ]);
      setTexts(nextTexts);
      setRegister(nextRegister);
      setReconciliation(nextReconciliation);
      setLegacy(nextLegacy);
    } catch (e) {
      setError(message(e));
    } finally {
      setLoading(false);
    }
  }, [year]);

  useEffect(() => {
    void load();
  }, [load]);

  async function exportRegister() {
    setExporting(true);
    try {
      downloadCSV(
        `verzeichnis-wahlrechte-${year}.csv`,
        await Api.exportTaxElectionRegisterCSV(year),
      );
    } catch (e) {
      setError(message(e));
    } finally {
      setExporting(false);
    }
  }

  if (loading) return <Loading />;

  return (
    <>
      <Section
        title="Anhang"
        context="Was kein Programm errechnen kann, steht als Freitext"
        divider={false}
        action={
          <HelpPopover label="Erklärung zum Anhang">
            Der Anhang erläutert Bilanz und Gewinn- und Verlustrechnung (§ 284 HGB). Die Texte
            werden beim Anlegen des Folgejahres als Vorlage übernommen: Bilanzierungs- und
            Bewertungsmethoden ändern sich selten, und ein leeres Feld verführt dazu, sie zu
            vergessen. Kleinstkapitalgesellschaften dürfen ihn nach § 264 Abs. 1 Satz 5 HGB weglassen.
          </HelpPopover>
        }
      >
        <BackendError message={error} />
        {texts.length === 0 ? (
          <EmptyState
            title="Keine Abschnitte verfügbar"
            description="Der Anhang ist eine Gliederung; ohne Geschäftsjahr gibt es sie nicht."
          />
        ) : (
          <div className="max-w-3xl">
            {texts.map((entry) => (
              <NotesSectionEditor
                key={entry.section}
                entry={entry}
                year={year}
                onSaved={setTexts}
              />
            ))}
          </div>
        )}
      </Section>

      <Section
        title="Verzeichnis nach § 5 Abs. 1 Satz 2 EStG"
        context={register ? `Geschäftsjahr ${register.fiscalYear}` : undefined}
        action={
          <div className="flex items-center gap-3">
            <HelpPopover label="Erklärung zum Verzeichnis">
              Wer ein steuerliches Wahlrecht abweichend von der Handelsbilanz ausübt, muss die
              betroffenen Wirtschaftsgüter in ein laufend zu führendes Verzeichnis aufnehmen. In
              Buchfink entsteht ein solcher Wert allein aus der Sonderabschreibung nach § 7g Abs. 5
              EStG; sie wird seit dem Wegfall der umgekehrten Maßgeblichkeit nicht mehr gebucht,
              sondern am Anlagegut geführt.
            </HelpPopover>
            <Button
              variant="secondary"
              icon={<Download className="w-4 h-4" strokeWidth={1.5} />}
              loading={exporting}
              disabled={!register || register.rows.length === 0}
              title={register && register.rows.length === 0 ? 'Das Verzeichnis ist leer' : undefined}
              onClick={() => void exportRegister()}
            >
              Als CSV speichern
            </Button>
          </div>
        }
      >
        {legacy && legacy.rows.length > 0 && (
          <Notice
            className="mb-6"
            text={`${legacy.rows.length} Sonderabschreibungen über ${formatCents(legacy.total)} stehen noch als Buchung im Journal; sie bleiben stehen und werden nicht doppelt im Verzeichnis geführt.`}
          />
        )}

        {!register || register.rows.length === 0 ? (
          <EmptyState
            title="Kein steuerliches Wahlrecht ausgeübt"
            description="Ohne Sonderabschreibung nach § 7g Abs. 5 EStG bleibt Handels- gleich Steuerbilanz."
          />
        ) : (
          <Table>
            <Thead>
              <Tr>
                <Th className="w-28">Inventar-Nr.</Th>
                <Th>Wirtschaftsgut</Th>
                <Th className="w-32">Anschaffung</Th>
                <Th numeric className="w-36">
                  Kosten
                </Th>
                <Th className="w-44">Vorschrift</Th>
                <Th numeric className="w-36">
                  Buchwert HB
                </Th>
                <Th numeric className="w-36">
                  Buchwert StB
                </Th>
                <Th numeric className="w-36">
                  Differenz
                </Th>
              </Tr>
            </Thead>
            <Tbody>
              {register.rows.map((row) => (
                <Tr key={row.assetId}>
                  <Td code>{row.inventoryNumber}</Td>
                  <Td className="whitespace-normal">{row.name}</Td>
                  <Td className="num text-ink-subtle">{formatDate(row.acquisitionDate)}</Td>
                  <Td numeric>{formatCents(row.cost)}</Td>
                  <Td className="text-ink-muted whitespace-normal">{row.provision}</Td>
                  <Td numeric>{formatCents(row.bookValue)}</Td>
                  <Td numeric>{formatCents(row.taxBookValue)}</Td>
                  <Td numeric>{formatCents(row.totalDifference)}</Td>
                </Tr>
              ))}
              <Tr variant="sum">
                <Td>Summe</Td>
                <Td />
                <Td />
                <Td />
                <Td />
                <Td numeric>{formatCents(register.totalBookValue)}</Td>
                <Td numeric>{formatCents(register.totalTaxBookValue)}</Td>
                <Td numeric>{formatCents(register.totalDifference)}</Td>
              </Tr>
            </Tbody>
          </Table>
        )}
      </Section>

      <Section
        title="Überleitung zur Steuerbilanz"
        context={reconciliation ? `Stichtag ${formatDate(reconciliation.cutoff)}` : undefined}
        action={
          <HelpPopover label="Erklärung zur Überleitung">
            § 60 Abs. 2 EStDV verlangt, die Handelsbilanz durch Zusätze oder Anmerkungen an die
            steuerlichen Vorschriften anzupassen, wo beide auseinanderfallen. In Buchfink sind das
            die Sonderabschreibung nach § 7g Abs. 5 EStG und die abweichende Abzinsung der
            Rückstellungen mit 5,5 % (§ 6 Abs. 1 Nr. 3a Buchst. e EStG).
          </HelpPopover>
        }
      >
        {!reconciliation || reconciliation.rows.length === 0 ? (
          <EmptyState
            title="Keine Abweichung zwischen Handels- und Steuerbilanz"
            description="Ohne steuerliches Wahlrecht und ohne abgezinste Rückstellung stimmen beide überein."
          />
        ) : (
          <Table>
            <Thead>
              <Tr>
                <Th>Position</Th>
                <Th numeric className="w-40">
                  Handelsbilanz
                </Th>
                <Th numeric className="w-40">
                  Steuerbilanz
                </Th>
                <Th numeric className="w-40">
                  Differenz
                </Th>
                <Th className="w-64">Rechtsgrundlage</Th>
              </Tr>
            </Thead>
            <Tbody>
              {reconciliation.rows.map((row) => (
                <Tr key={row.position}>
                  <Td className="whitespace-normal">
                    <span className="inline-flex items-center gap-1.5">
                      {row.position}
                      {row.explanation && (
                        <HelpTooltip
                          label={`Erklärung zu ${row.position}`}
                          content={row.explanation}
                        />
                      )}
                    </span>
                  </Td>
                  <Td numeric>{formatCents(row.commercial)}</Td>
                  <Td numeric>{formatCents(row.tax)}</Td>
                  <Td numeric>{formatCents(row.difference)}</Td>
                  <Td className="text-ink-muted whitespace-normal">{row.basis}</Td>
                </Tr>
              ))}
              <Tr variant="sum">
                <Td>Wirkung auf das Eigenkapital</Td>
                <Td />
                <Td />
                <Td numeric>{formatCents(reconciliation.equityEffect)}</Td>
                <Td />
              </Tr>
            </Tbody>
          </Table>
        )}
      </Section>
    </>
  );
};

// -------------------------------------------------------------------------
// Die Seite
// -------------------------------------------------------------------------

const TABS: { value: ModuleTab; label: string }[] = [
  { value: 'schritte', label: 'Schritte' },
  { value: 'abgrenzung', label: 'Abgrenzung' },
  { value: 'rueckstellungen', label: 'Rückstellungen' },
  { value: 'vorraete', label: 'Vorräte' },
  { value: 'umsatzsteuer', label: 'Umsatzsteuer' },
  { value: 'steuern', label: 'Steuerrückstellung' },
  { value: 'ergebnis', label: 'Ergebnisverwendung' },
  { value: 'anhang', label: 'Anhang' },
];

export interface ClosingModulesPageProps {
  /** Das Geschäftsjahr aus der Kopfzeile; die Ansicht folgt ihm. */
  year: number;
}

export const ClosingModulesPage: React.FC<ClosingModulesPageProps> = ({ year }) => {
  const [tab, setTab] = useState<ModuleTab>('schritte');
  const [accounts, setAccounts] = useState<Account[]>([]);
  // Offene und gesamte Bausteine kommen beide aus der Schrittliste. Die Zahl
  // der Schritte hier als Konstante zu schreiben wäre eine zweite Wahrheit:
  // sie stünde falsch da, sobald das Backend einen Baustein ergänzt.
  const [stepCounts, setStepCounts] = useState<{ open: number; total: number } | null>(null);

  // Der Kontenrahmen wird einmal geladen und an die Masken durchgereicht: Er
  // ändert sich während der Arbeit nicht, und jede Maske, die ihn selbst holt,
  // holt ihn noch einmal.
  useEffect(() => {
    Api.getAccounts()
      .then(setAccounts)
      .catch(() => setAccounts([]));
  }, []);

  // Die Zahl der offenen Bausteine steht im Seitenkopf. Sie kommt aus derselben
  // Quelle wie die Schrittliste und wird hier nur gelesen.
  useEffect(() => {
    Api.getClosingSteps(year)
      .then((steps) => setStepCounts({ open: steps.openCount, total: steps.steps.length }))
      .catch(() => setStepCounts(null));
  }, [year, tab]);

  return (
    <div className="max-w-[1200px] mx-auto px-8 py-8">
      <PageHeader
        title="Abschlussbausteine"
        context={
          stepCounts === null
            ? `Geschäftsjahr ${year}`
            : `Geschäftsjahr ${year} · ${stepCounts.open} von ${stepCounts.total} Schritten offen`
        }
      />

      <div className="mt-6">
        <Tabs items={TABS} value={tab} onValueChange={(next) => setTab(next as ModuleTab)}>
          <TabPanel value="schritte">
            {tab === 'schritte' && <StepsTab year={year} onOpenTab={setTab} />}
          </TabPanel>
          <TabPanel value="abgrenzung">
            {tab === 'abgrenzung' && <AccrualsTab year={year} accounts={accounts} />}
          </TabPanel>
          <TabPanel value="rueckstellungen">
            {tab === 'rueckstellungen' && <ProvisionsTab year={year} accounts={accounts} />}
          </TabPanel>
          <TabPanel value="vorraete">{tab === 'vorraete' && <InventoryTab year={year} />}</TabPanel>
          <TabPanel value="umsatzsteuer">
            {tab === 'umsatzsteuer' && <VatSettlementTab year={year} />}
          </TabPanel>
          <TabPanel value="steuern">{tab === 'steuern' && <TaxProvisionTab year={year} />}</TabPanel>
          <TabPanel value="ergebnis">
            {tab === 'ergebnis' && <AppropriationTab year={year} />}
          </TabPanel>
          <TabPanel value="anhang">{tab === 'anhang' && <NotesTab year={year} />}</TabPanel>
        </Tabs>
      </div>
    </div>
  );
};
