import React, { useCallback, useEffect, useState } from 'react';
import { AlertCircle, Layers, Undo2 } from 'lucide-react';
import { CarryForwardPreview, ClosingState, FiscalYearStatus, SizeClass } from '../types';
import { Api } from '../services/api';
import { formatCents, formatDate, formatDateTime } from '../utils/formatters';
import {
  Button,
  ConfirmDialog,
  Dialog,
  EmptyState,
  Field,
  HelpPopover,
  HelpTooltip,
  Input,
  PageHeader,
  Section,
  SkeletonRows,
  Stat,
  StatRow,
  StatusBadge,
  Table,
  Tbody,
  Td,
  Textarea,
  Th,
  Thead,
  Tr,
  type Status,
  toast,
} from '../components/ui';

/**
 * Der Jahresabschluss: der Weg vom offenen Geschäftsjahr über die Aufstellung
 * und die Feststellung bis zur Offenlegung, und der Saldenvortrag ins
 * Folgejahr.
 *
 * Gerechnet und geprüft wird alles im Backend (internal/service/closing_service.go).
 * Diese Ansicht zeigt nur, was dort steht — insbesondere rechnet sie die
 * Vortragsdifferenz nicht selbst nach, sonst gäbe es zwei Wahrheiten über die
 * Bilanzidentität.
 */

/** Der Abschlussstand im Vokabular der Statusanzeige (§11.3). */
const STATUS_BADGE: Record<FiscalYearStatus, Status> = {
  open: 'offen',
  prepared: 'aufgestellt',
  adopted: 'festgestellt',
  disclosed: 'offengelegt',
};

/**
 * Die Beschriftung eines Schrittes — als Knopf wie als Dialogtitel, damit der
 * Nutzer im Dialog dasselbe Verb liest, das ihn dorthin gebracht hat (§15.3).
 *
 * `open` fehlt bewusst: dorthin führt kein Schritt, sondern nur die Rücksetzung.
 */
const STEP_ACTION: Partial<Record<FiscalYearStatus, string>> = {
  prepared: 'Abschluss aufstellen',
  adopted: 'Abschluss feststellen',
  disclosed: 'Offenlegung eintragen',
};

/** Die Größenklasse in der Sprache der Oberfläche. */
const SIZE_LABELS: Record<string, string> = {
  micro: 'Kleinstkapitalgesellschaft',
  small: 'Kleine Kapitalgesellschaft',
  medium: 'Mittelgroße Kapitalgesellschaft',
  large: 'Große Kapitalgesellschaft',
};

const SIZE_DEPTH_LABELS: Record<string, string> = {
  full: 'Vollgliederung',
  short: 'Verkürzte Gliederung',
  letters: 'Buchstabengliederung',
};

function todayISO(): string {
  const now = new Date();
  const pad = (value: number) => String(value).padStart(2, '0');
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`;
}

/**
 * Gefärbt wird nur das Ergebnis, und die Null ist keins: ein ausgeglichenes
 * Jahr in Salbei zu zeigen wäre eine Aussage, die niemand getroffen hat (§3.4).
 */
function resultTone(amount: number): 'neutral' | 'positive' | 'negative' {
  if (amount === 0) return 'neutral';
  return amount < 0 ? 'negative' : 'positive';
}

function message(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}

/** Ein Hinweis in Rosé: Ursache und nächster Schritt, je in einem Satz (§15.3). */
interface Hint {
  key: string;
  cause: string;
  next: string;
}

/**
 * Warum ein Vortrag jetzt nicht zu buchen ist.
 *
 * Der Dienst lehnt in genau diesen Fällen ab und nennt dabei Ursache und
 * nächsten Schritt. Ein Knopf, der ohne Begründung grau ist, verschwiege beides
 * (§8.3), deshalb steht der Grund hier in einer Hinweisfläche über den Aktionen
 * (§6.2 Nr. 4) — mit derselben Diagnose, die auch die Ablehnung tragen würde.
 */
function carryForwardHints(preview: CarryForwardPreview): Hint[] {
  const hints: Hint[] = [];

  if (!preview.isBalanced) {
    hints.push({
      key: 'balance',
      cause:
        `Die Salden des Geschäftsjahres ${preview.fromYear} gehen um ` +
        `${formatCents(preview.balanceDifference)} nicht auf.`,
      // Die häufigste Ursache ist kein Datenfehler, sondern eine Lücke in der
      // Kette: die offenen Posten laufen jahresübergreifend, die
      // Sachkontensalden nicht.
      next: preview.priorYearNotCarried
        ? `Buchen Sie zuerst den Saldenvortrag ${preview.fromYear - 1} → ${preview.fromYear}.`
        : `Prüfen Sie die Summen- und Saldenliste des Geschäftsjahres ${preview.fromYear}.`,
    });
  }

  if (preview.irreversible) {
    hints.push({
      key: 'irreversible',
      cause:
        `Ein Teil der Vortragswerte im Geschäftsjahr ${preview.toYear} gehört zu keiner ` +
        `zurücknehmbaren Buchung mehr.`,
      next: `Gleichen Sie den stornierten Vortrag innerhalb des Geschäftsjahres ${preview.toYear} aus.`,
    });
  }

  return hints;
}

/**
 * Die ausführliche Fassung desselben Grundes.
 *
 * Ein Hinweisstreifen trägt einen Satz Ursache und einen Satz nächsten Schritt
 * (§15.1); die Kette der Vorträge und die Liste der betroffenen Konten gehören
 * deshalb nicht dorthin, sondern in den `title` des gesperrten Knopfes, wo sie
 * abrufbar bleiben, ohne dauerhaft in der Arbeitsansicht zu stehen (§15.2).
 */
function carryForwardDetail(preview: CarryForwardPreview | null): string {
  if (!preview) return '';

  if (!preview.isBalanced) {
    return (
      `Aktiva, Passiva und das Jahresergebnis von ${formatCents(preview.netIncome)} stimmen nicht ` +
      `zusammen; ein Vortrag würde die Differenz ins Geschäftsjahr ${preview.toYear} tragen.` +
      (preview.priorYearNotCarried
        ? ` Im Geschäftsjahr ${preview.fromYear} steht kein Saldenvortrag, obwohl es Buchungen aus ` +
          `früheren Jahren gibt.`
        : '')
    );
  }

  if (preview.irreversible) {
    const carried = preview.rows.filter((row) => row.carried !== 0).map((row) => row.account);
    const accounts =
      carried.length > 8 ? `${carried.slice(0, 8).join(', ')} und ${carried.length - 8} weitere` : carried.join(', ');
    return (
      `Die bestehende Vortragsbuchung wurde außerhalb des Geschäftsjahres ${preview.toYear} ` +
      `storniert; ein Korrekturvortrag würde diese Werte verdoppeln.` +
      (accounts ? ` Zu prüfen sind die Konten mit einem Wert in der Spalte „Vorgetragen": ${accounts}.` : '')
    );
  }

  return '';
}

interface Step {
  key: string;
  title: string;
  status: Status;
  date: string;
  note: string;
  action?: React.ReactNode;
}

export interface ClosingPageProps {
  /** Das Geschäftsjahr aus der Kopfzeile; die Ansicht folgt ihm. */
  year: number;
  /**
   * Meldet einen geänderten Abschlussstand nach oben: die Kopfzeile zeigt das
   * Schloss eines festgestellten Jahres, und das soll nicht erst nach einem
   * Neustart stimmen.
   */
  onFiscalYearChanged?: () => void | Promise<void>;
}

export const ClosingPage: React.FC<ClosingPageProps> = ({ year, onFiscalYearChanged }) => {
  const [state, setState] = useState<ClosingState | null>(null);
  const [preview, setPreview] = useState<CarryForwardPreview | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [previewError, setPreviewError] = useState('');

  const [stepStatus, setStepStatus] = useState<FiscalYearStatus | null>(null);
  const [stepDate, setStepDate] = useState(todayISO());
  const [stepNote, setStepNote] = useState('');
  // Zwei Arten von Fehler, zwei Orte: der fachliche Feldfehler steht am Feld,
  // die Ablehnung aus dem Backend in einer Hinweisfläche über dem Fuß (§10.4).
  const [stepFieldError, setStepFieldError] = useState('');
  const [stepBackendError, setStepBackendError] = useState('');

  const [reopenOpen, setReopenOpen] = useState(false);
  const [reopenReason, setReopenReason] = useState('');
  const [reopenFieldError, setReopenFieldError] = useState('');
  const [reopenBackendError, setReopenBackendError] = useState('');

  const [confirmCarry, setConfirmCarry] = useState(false);
  const [busy, setBusy] = useState(false);

  // Die Größenklasse hängt an der Bilanzsumme und damit an einem Abschluss, der
  // aufgeht. Sie bekommt deshalb einen eigenen Fehlerpfad: schlägt sie fehl,
  // bleiben Schritte und Saldenvortrag benutzbar.
  const [sizeClass, setSizeClass] = useState<SizeClass | null>(null);
  const [sizeClassError, setSizeClassError] = useState('');
  const [employees, setEmployees] = useState('0');
  const [savingEmployees, setSavingEmployees] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    setPreviewError('');
    try {
      const closing = await Api.getClosingState(year);
      setState(closing);
      setEmployees(String(closing.fiscalYear.averageEmployees ?? 0));
      try {
        setSizeClass(await Api.getSizeClass(year));
        setSizeClassError('');
      } catch (e) {
        setSizeClass(null);
        setSizeClassError(message(e));
      }
      try {
        setPreview(await Api.getCarryForwardPreview(closing.nextYear));
      } catch (e) {
        // Der Vortragsstand kann fehlschlagen, ohne dass der Abschlussstand
        // dadurch unbrauchbar wäre. Beides gemeinsam zu verwerfen würde die
        // Ansicht wegen einer Teilfrage leeren.
        setPreview(null);
        setPreviewError(message(e));
      }
    } catch (e) {
      setState(null);
      setPreview(null);
      setError(message(e));
    } finally {
      setLoading(false);
    }
  }, [year]);

  useEffect(() => {
    void load();
  }, [load]);

  /**
   * Die Arbeitnehmerzahl ist das dritte Merkmal des § 267 Abs. 1 HGB. Sie steht
   * am Geschäftsjahr und nicht in den Einstellungen: sie gilt für dieses Jahr.
   */
  async function saveEmployees() {
    const count = Number.parseInt(employees, 10);
    if (!Number.isFinite(count) || count < 0) {
      toast.error('Die Arbeitnehmerzahl ist eine Zahl ab null. Bitte korrigieren Sie die Eingabe.');
      return;
    }
    setSavingEmployees(true);
    try {
      await Api.setAverageEmployees(year, count);
      await load();
    } catch (e) {
      toast.error(message(e));
    } finally {
      setSavingEmployees(false);
    }
  }

  function openStepDialog(next: FiscalYearStatus) {
    setStepStatus(next);
    setStepDate(todayISO());
    setStepNote('');
    setStepFieldError('');
    setStepBackendError('');
  }

  async function submitStep() {
    if (!state || !stepStatus) return;
    if (!stepDate) {
      setStepFieldError('Das Datum fehlt. Ohne Tag lässt sich der Schritt nicht belegen.');
      return;
    }
    setBusy(true);
    try {
      await Api.setFiscalYearStatus(state.year, stepStatus, stepDate, stepNote);
      setStepStatus(null);
      // Kein Toast: der sich schließende Dialog ist die Rückmeldung, und der
      // neue Stand steht danach in der Schritte-Tabelle (§8.5).
      await load();
      await onFiscalYearChanged?.();
    } catch (e) {
      setStepBackendError(message(e));
    } finally {
      setBusy(false);
    }
  }

  async function submitReopen() {
    if (!state) return;
    if (!reopenReason.trim()) {
      setReopenFieldError('Der Grund fehlt. Er wird im Änderungsprotokoll festgehalten.');
      return;
    }
    setBusy(true);
    try {
      await Api.reopenFiscalYear(state.year, reopenReason);
      setReopenOpen(false);
      setReopenReason('');
      await load();
      await onFiscalYearChanged?.();
    } catch (e) {
      setReopenBackendError(message(e));
    } finally {
      setBusy(false);
    }
  }

  async function runCarryForward() {
    if (!preview) return;
    setBusy(true);
    try {
      const entries = await Api.carryForward(preview.toYear);
      await load();
      // Der Vortrag legt das Zieljahr an und bucht hinein: es ist ab jetzt ein
      // Geschäftsjahr und gehört in die Auswahl der Kopfzeile, nicht erst nach
      // einem Neustart.
      await onFiscalYearChanged?.();
      toast.success(
        `Saldenvortrag ${preview.fromYear} → ${preview.toYear}: ${entries.length} Buchungen.`
      );
    } catch (e) {
      setPreviewError(message(e));
    } finally {
      setBusy(false);
    }
  }

  if (loading) {
    return (
      <div className="max-w-[1200px] mx-auto px-8 py-8">
        <PageHeader title="Jahresabschluss" context={`Geschäftsjahr ${year}`} />
        <div className="mt-8">
          <SkeletonRows rows={6} />
        </div>
      </div>
    );
  }

  if (!state) {
    return (
      <div className="max-w-[1200px] mx-auto px-8 py-8">
        <PageHeader title="Jahresabschluss" context={`Geschäftsjahr ${year}`} />
        <div className="mt-8">
          <EmptyState
            title="Der Abschlussstand konnte nicht gelesen werden"
            description={error || 'Das Geschäftsjahr ist nicht verfügbar.'}
            action={
              <Button variant="secondary" onClick={() => void load()}>
                Erneut laden
              </Button>
            }
          />
        </div>
      </div>
    );
  }

  const fy = state.fiscalYear;
  const adopted = fy.status === 'adopted' || fy.status === 'disclosed';

  const steps: Step[] = [
    {
      key: 'commitment',
      title: 'Jahres-Festschreibung',
      status: state.hasYearCommitment ? 'festgeschrieben' : 'offen',
      date: state.hasYearCommitment ? state.committedUntil ?? '' : '',
      note: '',
    },
    {
      key: 'prepared',
      title: 'Aufgestellt',
      status: fy.preparedOn ? 'aufgestellt' : 'offen',
      date: fy.preparedOn ?? '',
      note: '',
    },
    {
      key: 'adopted',
      title: 'Festgestellt',
      status: fy.adoptedOn ? 'festgestellt' : 'offen',
      date: fy.adoptedOn ?? '',
      note: fy.adoptionNote ?? '',
      action: adopted ? (
        <Button
          variant="quiet"
          size="sm"
          icon={<Undo2 className="w-3.5 h-3.5" strokeWidth={1.5} />}
          onClick={() => {
            setReopenFieldError('');
            setReopenBackendError('');
            setReopenOpen(true);
          }}
        >
          Zurücksetzen
        </Button>
      ) : undefined,
    },
    {
      key: 'disclosed',
      title: 'Offengelegt',
      status: fy.disclosedOn ? 'offengelegt' : 'offen',
      date: fy.disclosedOn ?? '',
      note: '',
    },
  ];

  const carryTone = state.carryForwardCurrent
    ? 'positive'
    : state.carriedForward
      ? 'negative'
      : 'neutral';
  const carryValue = state.carryForwardCurrent
    ? 'Aktuell'
    : state.carriedForward
      ? 'Überholt'
      : 'Offen';

  const hints = preview ? carryForwardHints(preview) : [];

  const carryBlocked =
    !preview ||
    !preview.isBalanced ||
    Boolean(preview.irreversible) ||
    (preview.alreadyCarried && !preview.needsCorrection) ||
    (preview.entries === 0 && !preview.alreadyCarried);

  const carryLabel = preview?.alreadyCarried ? 'Korrekturvortrag buchen' : 'Vortrag buchen';
  const carryReason = !preview
    ? 'Der Vortragsstand konnte nicht gelesen werden'
    : !preview.isBalanced
      ? `Die Salden des Geschäftsjahres ${preview.fromYear} gehen nicht auf`
      : preview.irreversible
        ? 'Der bestehende Vortrag lässt sich nicht zurücknehmen'
        : preview.alreadyCarried && !preview.needsCorrection
          ? 'Der Vortrag ist auf dem aktuellen Stand'
          : preview.entries === 0
            ? 'Es gibt keine Salden, die vorzutragen wären'
            : '';
  // Der Grund allein sagt, dass gesperrt ist; die Erklärung sagt, warum.
  const carryTitle = carryBlocked
    ? [carryReason ? `${carryReason}.` : '', carryForwardDetail(preview)].filter(Boolean).join(' ')
    : undefined;

  return (
    <div className="max-w-[1200px] mx-auto px-8 py-8">
      <PageHeader
        title="Jahresabschluss"
        context={`Geschäftsjahr ${fy.year} · ${formatDate(fy.startDate)} – ${formatDate(fy.endDate)}`}
        action={
          <div className="flex items-center gap-3">
            <StatusBadge status={STATUS_BADGE[fy.status]} />
            {state.nextStatus && (
              <Button
                variant="primary"
                disabled={!state.canAdopt}
                title={state.canAdopt ? undefined : state.blocker}
                onClick={() => openStepDialog(state.nextStatus as FiscalYearStatus)}
              >
                {STEP_ACTION[state.nextStatus]}
              </Button>
            )}
          </div>
        }
      />

      <div className="mt-6">
        <StatRow>
          <Stat
            label="Jahresergebnis"
            value={formatCents(state.netIncome)}
            context="Erträge minus Aufwendungen der GuV-Konten"
            tone={resultTone(state.netIncome)}
          />
          <Stat
            label="Jahres-Festschreibung"
            value={state.hasYearCommitment ? 'Vorhanden' : 'Fehlt'}
            context={
              state.committedUntil
                ? `Festgeschrieben bis ${formatDate(state.committedUntil)}`
                : 'Voraussetzung der Feststellung'
            }
            tone={state.hasYearCommitment ? 'positive' : 'neutral'}
          />
          <Stat
            label={`Saldenvortrag nach ${state.nextYear}`}
            value={carryValue}
            context={
              state.carriedForwardAt
                ? `Zuletzt am ${formatDateTime(state.carriedForwardAt)}`
                : 'Noch nicht gebucht'
            }
            tone={carryTone}
          />
          <Stat
            label="Zeitraum"
            value={fy.isShort ? 'Rumpfjahr' : 'Volles Jahr'}
            context={`${formatDate(fy.startDate)} – ${formatDate(fy.endDate)}`}
          />
        </StatRow>
      </div>

      <Section
        title="Schritte"
        context="Von der Festschreibung bis zur Offenlegung"
        action={
          <HelpPopover label="Erklärung zum Jahresabschluss">
            § 242 HGB verlangt zum Ende jedes Geschäftsjahres einen Abschluss; unterzeichnet wird er
            unter Angabe des Datums (§ 245 HGB). Festgestellt wird er von den Gesellschaftern
            (§ 42a Abs. 2 GmbHG). Ab der Feststellung nimmt das Geschäftsjahr keine Buchung mehr an;
            zurück geht es nur über die Rücksetzung, und die verlangt einen Grund.
          </HelpPopover>
        }
      >
        <Table>
          <Thead>
            <Tr>
              <Th>Schritt</Th>
              <Th className="w-44">Stand</Th>
              <Th className="w-32">Datum</Th>
              <Th>Beschlussbezug</Th>
              <Th className="w-36" aria-label="Aktionen" />
            </Tr>
          </Thead>
          <Tbody>
            {steps.map((step) => (
              <Tr key={step.key}>
                <Td>{step.title}</Td>
                <Td>
                  <StatusBadge status={step.status} />
                </Td>
                <Td className="text-ink-subtle num">{step.date ? formatDate(step.date) : '—'}</Td>
                <Td className="text-ink-muted">{step.note || '—'}</Td>
                <Td className="pl-0">{step.action}</Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      </Section>

      <Section
        title="Größenklasse"
        context={
          sizeClass
            ? `${SIZE_LABELS[sizeClass.class] ?? sizeClass.class} · ${sizeClass.reason}`
            : 'Bilanzsumme, Umsatzerlöse und Arbeitnehmerzahl entscheiden'
        }
        action={
          <HelpPopover label="Erklärung zur Größenklasse">
            Die §§ 267, 267a HGB ordnen eine Kapitalgesellschaft nach Bilanzsumme, Umsatzerlösen und
            Arbeitnehmerzahl ein; zwei der drei Merkmale entscheiden. Die Rechtsfolge tritt erst
            ein, wenn zwei aufeinander folgende Stichtage dieselbe Klasse ergeben (§ 267 Abs. 4
            HGB).
          </HelpPopover>
        }
      >
        {/* Ab der Feststellung bleibt das Feld an seinem Platz und wird
            deaktiviert (§11.5): an der Zahl hängt über die Größenklasse die
            Gliederungstiefe, und die eines festgestellten Abschlusses steht.
            Der Service weist die Änderung ebenfalls ab — die Sperre gehört
            nicht allein in die Oberfläche. */}
        <Field
          label="Arbeitnehmer im Jahresdurchschnitt"
          help="Durchschnitt der an den vier Quartalsstichtagen Beschäftigten (§ 267 Abs. 5 HGB); Auszubildende bleiben außer Betracht."
          hint={adopted ? 'Änderbar erst nach Rücksetzung der Feststellung' : undefined}
          disabled={adopted}
          className="max-w-sm"
        >
          <div className="flex items-center gap-2">
            <Input
              value={employees}
              inputMode="numeric"
              disabled={adopted}
              onChange={(e) => setEmployees(e.target.value)}
              className="num"
            />
            <Button
              variant="secondary"
              onClick={saveEmployees}
              loading={savingEmployees}
              disabled={adopted}
            >
              Übernehmen
            </Button>
          </div>
        </Field>

        {sizeClassError ? (
          <p className="mt-4 text-body text-ink-muted">{sizeClassError}</p>
        ) : (
          sizeClass && (
            <div className="mt-6">
              <Table density="kompakt">
                <Thead>
                  <Tr>
                    <Th>Merkmal oder Folge</Th>
                    <Th className="w-80">Wert</Th>
                    <Th className="w-64">Norm</Th>
                  </Tr>
                </Thead>
                <Tbody>
                  <Tr>
                    <Td>Bilanzsumme</Td>
                    <Td className="num">{formatCents(sizeClass.criteria.balanceSheetTotal)}</Td>
                    <Td className="text-ink-muted">§ 267 Abs. 4a HGB</Td>
                  </Tr>
                  <Tr>
                    <Td>Umsatzerlöse</Td>
                    <Td className="num">{formatCents(sizeClass.criteria.revenue)}</Td>
                    <Td className="text-ink-muted">§ 275 Abs. 2 Nr. 1 HGB</Td>
                  </Tr>
                  <Tr>
                    <Td>Arbeitnehmer im Jahresdurchschnitt</Td>
                    <Td className="num">{sizeClass.criteria.employees}</Td>
                    <Td className="text-ink-muted">§ 267 Abs. 5 HGB</Td>
                  </Tr>
                  <Tr>
                    <Td>Gliederungstiefe</Td>
                    <Td className="whitespace-normal">
                      {SIZE_DEPTH_LABELS[sizeClass.obligations.depth]}
                    </Td>
                    <Td className="text-ink-muted whitespace-normal">
                      {sizeClass.obligations.depthReference}
                    </Td>
                  </Tr>
                  <Tr>
                    <Td>Anhang</Td>
                    <Td>{sizeClass.obligations.notesRequired ? 'Ja' : 'Nein'}</Td>
                    <Td className="text-ink-muted whitespace-normal">
                      {sizeClass.obligations.notesReference}
                    </Td>
                  </Tr>
                  <Tr>
                    <Td>Lagebericht</Td>
                    <Td>{sizeClass.obligations.managementReport ? 'Ja' : 'Nein'}</Td>
                    <Td className="text-ink-muted whitespace-normal">
                      {sizeClass.obligations.managementReportReference || '§ 264 Abs. 1 Satz 4 HGB'}
                    </Td>
                  </Tr>
                  <Tr>
                    <Td>Prüfung</Td>
                    <Td>{sizeClass.obligations.auditRequired ? 'Ja' : 'Nein'}</Td>
                    <Td className="text-ink-muted whitespace-normal">
                      {sizeClass.obligations.auditReference || '§ 316 Abs. 1 Satz 1 HGB'}
                    </Td>
                  </Tr>
                  <Tr>
                    <Td>Aufstellungsfrist</Td>
                    <Td>{`${sizeClass.obligations.preparationMonths} Monate`}</Td>
                    <Td className="text-ink-muted whitespace-normal">
                      {sizeClass.obligations.preparationReference}
                    </Td>
                  </Tr>
                  <Tr>
                    <Td>Offenlegung</Td>
                    <Td className="whitespace-normal">{sizeClass.obligations.disclosureScope}</Td>
                    <Td className="text-ink-muted whitespace-normal">
                      {sizeClass.obligations.disclosureScopeReference}
                    </Td>
                  </Tr>
                </Tbody>
              </Table>
            </div>
          )
        )}
      </Section>

      <Section
        title="Saldenvortrag ins Folgejahr"
        context={preview ? `${preview.fromYear} → ${preview.toYear}` : `${year} → ${state.nextYear}`}
        action={
          <div className="flex items-center gap-3">
            <HelpPopover label="Erklärung zum Saldenvortrag">
              § 252 Abs. 1 Nr. 1 HGB verlangt, dass die Eröffnungsbilanz mit der Schlussbilanz des
              Vorjahres übereinstimmt. Vorgetragen werden die Bilanzkonten gegen 9000, die offenen
              Posten der Debitoren gegen 9008 und der Kreditoren gegen 9009. Das Jahresergebnis geht
              auf den Gewinn- oder Verlustvortrag; über seine Verwendung wird gesondert beschlossen.
            </HelpPopover>
            <Button
              variant="secondary"
              icon={<Layers className="w-4 h-4" strokeWidth={1.5} />}
              disabled={carryBlocked || busy}
              title={carryTitle}
              onClick={() => setConfirmCarry(true)}
            >
              {carryLabel}
            </Button>
          </div>
        }
      >
        {previewError && (
          <div className="mb-5 flex items-start gap-2.5 rounded-control border border-negative-line bg-negative-soft px-4 py-3">
            <AlertCircle className="w-4 h-4 mt-0.5 shrink-0 text-negative" strokeWidth={1.5} />
            <p className="text-body text-negative-text">{previewError}</p>
          </div>
        )}

        {hints.map((hint) => (
          <div
            key={hint.key}
            className="mb-5 flex items-start gap-2.5 rounded-control border border-negative-line bg-negative-soft px-4 py-3"
          >
            <AlertCircle className="w-4 h-4 mt-0.5 shrink-0 text-negative" strokeWidth={1.5} />
            <div className="text-body text-negative-text">
              <p>{hint.cause}</p>
              <p className="mt-1">{hint.next}</p>
            </div>
          </div>
        ))}

        {preview && (
          <>
            <div className="mb-6">
              <StatRow>
                {/* Der Abschnitt zeigt Ergebnis und Zielkonto nebeneinander: das
                    Jahresergebnis ist der einzige Wert der Vortragsbuchung, der
                    nicht aus einem Schlusssaldo stammt, sondern aus der GuV. */}
                <Stat
                  label="Jahresergebnis"
                  value={formatCents(preview.netIncome)}
                  context={`Aus dem Geschäftsjahr ${preview.fromYear}`}
                  tone={resultTone(preview.netIncome)}
                />
                <Stat
                  label="Ergebniskonto"
                  value={preview.resultAccount}
                  context={preview.resultAccountName}
                />
                <Stat
                  label="Buchungsdatum"
                  value={formatDate(preview.bookingDate)}
                  context={
                    preview.deferred
                      ? 'Erster offener Tag, der Jahresbeginn ist festgeschrieben'
                      : 'Erster Tag des neuen Geschäftsjahres'
                  }
                />
                <Stat
                  label="Bilanzidentität"
                  value={preview.isBalanced ? 'Ausgeglichen' : formatCents(preview.balanceDifference)}
                  context={
                    preview.isBalanced
                      ? 'Summe aller Vortragswerte ist null'
                      : 'Differenz aus Aktiva, Passiva und Ergebnis'
                  }
                  tone={preview.isBalanced ? 'positive' : 'negative'}
                />
              </StatRow>
            </div>

            {preview.rows.length === 0 ? (
              <EmptyState
                title="Keine Salden zum Vortrag"
                description={`Im Geschäftsjahr ${preview.fromYear} stehen weder Bilanzsalden noch offene Posten.`}
              />
            ) : (
              <Table>
                <Thead>
                  <Tr>
                    <Th className="w-28">Konto</Th>
                    <Th>Bezeichnung</Th>
                    <Th numeric>
                      <span className="inline-flex items-center gap-1.5">
                        Schlusssaldo
                        <HelpTooltip
                          label="Erklärung zum Schlusssaldo"
                          content="Positive Beträge stehen im Soll, negative im Haben."
                        />
                      </span>
                    </Th>
                    <Th numeric>Vorgetragen</Th>
                    <Th numeric>Differenz</Th>
                  </Tr>
                </Thead>
                <Tbody>
                  {preview.rows.map((row) => (
                    <Tr key={row.account}>
                      <Td code>{row.account}</Td>
                      <Td>
                        {row.name}
                        {/* Auf dem Ergebniskonto steht unter „Schlusssaldo" nicht der
                            Schlusssaldo des Vorjahres allein, sondern er zuzüglich des
                            Jahresergebnisses. Ohne diesen Zusatz fände jeder, der die
                            Zeile gegen die Summen- und Saldenliste des Vorjahres hält,
                            eine unerklärte Abweichung — und zwar genau auf dem Konto,
                            das die Bilanzidentität trägt. */}
                        {row.includesNetIncome && (
                          <span className="text-ink-muted">
                            {` · inkl. Jahresergebnis ${formatCents(preview.netIncome)}`}
                          </span>
                        )}
                        {/* Ein Personenkonto wird je offenem Posten vorgetragen und
                            läuft gegen ein anderes Vortragskonto als ein Sachkonto;
                            unterschieden wird das im Wort, nicht in der Farbe (§3.4). */}
                        {row.kind !== 'sachkonto' && (
                          <span className="text-ink-muted">
                            {` · ${row.kind === 'debitor' ? 'Debitor' : 'Kreditor'}`}
                            {row.openItems
                              ? `, ${row.openItems} ${row.openItems === 1 ? 'offener Posten' : 'offene Posten'}`
                              : ''}
                          </span>
                        )}
                      </Td>
                      <Td numeric>{formatCents(row.closingBalance)}</Td>
                      <Td numeric className="text-ink-muted">
                        {formatCents(row.carried)}
                      </Td>
                      <Td numeric className={row.difference === 0 ? 'text-ink-faint' : undefined}>
                        {formatCents(row.difference)}
                      </Td>
                    </Tr>
                  ))}
                  <Tr variant="sum">
                    <Td>Summe</Td>
                    <Td />
                    <Td numeric>{formatCents(preview.balanceDifference)}</Td>
                    <Td numeric>
                      {formatCents(preview.rows.reduce((sum, row) => sum + row.carried, 0))}
                    </Td>
                    <Td numeric>
                      {formatCents(preview.rows.reduce((sum, row) => sum + row.difference, 0))}
                    </Td>
                  </Tr>
                </Tbody>
              </Table>
            )}
          </>
        )}
      </Section>

      <Dialog
        open={stepStatus !== null}
        onOpenChange={(next) => !next && setStepStatus(null)}
        title={(stepStatus && STEP_ACTION[stepStatus]) || ''}
        width="max-w-lg"
        footer={
          <>
            <Button variant="secondary" onClick={() => setStepStatus(null)}>
              Abbrechen
            </Button>
            <Button variant="primary" loading={busy} onClick={() => void submitStep()}>
              {(stepStatus && STEP_ACTION[stepStatus]) || ''}
            </Button>
          </>
        }
      >
        <Field label="Datum" error={stepFieldError || undefined}>
          <Input
            type="date"
            value={stepDate}
            onChange={(e) => {
              setStepDate(e.target.value);
              setStepFieldError('');
            }}
          />
        </Field>

        {stepStatus === 'adopted' && (
          <Field
            label="Beschlussbezug"
            optional
            help="Welcher Gesellschafterbeschluss den Abschluss festgestellt hat."
            className="mt-4"
          >
            <Input
              value={stepNote}
              onChange={(e) => setStepNote(e.target.value)}
              placeholder="Gesellschafterbeschluss vom …"
            />
          </Field>
        )}

        {stepBackendError && (
          <div className="mt-5 flex items-start gap-2.5 rounded-control border border-negative-line bg-negative-soft px-4 py-3">
            <AlertCircle className="w-4 h-4 mt-0.5 shrink-0 text-negative" strokeWidth={1.5} />
            <p className="text-body text-negative-text">{stepBackendError}</p>
          </div>
        )}
      </Dialog>

      <Dialog
        open={reopenOpen}
        onOpenChange={(next) => !next && setReopenOpen(false)}
        title="Feststellung zurücksetzen"
        width="max-w-lg"
        footer={
          <>
            <Button variant="secondary" onClick={() => setReopenOpen(false)}>
              Abbrechen
            </Button>
            <Button variant="danger" loading={busy} onClick={() => void submitReopen()}>
              Feststellung zurücksetzen
            </Button>
          </>
        }
      >
        <Field
          label="Grund"
          error={reopenFieldError || undefined}
          help="Geht ins Änderungsprotokoll und bleibt dort."
        >
          <Textarea
            rows={3}
            value={reopenReason}
            onChange={(e) => {
              setReopenReason(e.target.value);
              setReopenFieldError('');
            }}
          />
        </Field>

        {reopenBackendError && (
          <div className="mt-5 flex items-start gap-2.5 rounded-control border border-negative-line bg-negative-soft px-4 py-3">
            <AlertCircle className="w-4 h-4 mt-0.5 shrink-0 text-negative" strokeWidth={1.5} />
            <p className="text-body text-negative-text">{reopenBackendError}</p>
          </div>
        )}
      </Dialog>

      <ConfirmDialog
        open={confirmCarry}
        onOpenChange={setConfirmCarry}
        title={carryLabel}
        description={
          preview
            ? `${preview.entries} Buchungen zum ${formatDate(preview.bookingDate)} im Geschäftsjahr ${preview.toYear}.` +
              (preview.alreadyCarried
                ? ' Der bestehende Vortrag wird zuvor per Generalumkehr zurückgenommen.'
                : '')
            : ''
        }
        confirmLabel={carryLabel}
        onConfirm={() => void runCarryForward()}
      />
    </div>
  );
};
