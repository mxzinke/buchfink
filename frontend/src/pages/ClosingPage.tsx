import React, { useCallback, useEffect, useState } from 'react';
import { AlertCircle, Layers, Undo2 } from 'lucide-react';
import {
  AccrualKind,
  CarryForwardPreview,
  ClosingState,
  ClosingStepKey,
  ClosingSteps,
  ClosingStepView,
  FiscalYearStatus,
  SizeClass,
} from '../types';
import { Api } from '../services/api';
import type { NavigateFn, NavigationParams, TabType } from '../components/Sidebar';
import { useWriteLock } from '../components/WriteLock';
import { OpeningBalanceDialog } from '../components/OpeningBalanceDialog';
// Die Zuordnung Baustein → Reiter steht dort, wo die Reiter stehen. Eine zweite
// Kopie hier liefe auseinander, sobald ein Baustein dazukommt.
import { SkippedMark, STEP_OBLIGATIONS, STEP_TABS } from './ClosingModulesPage';
import { formatCents, formatDate, formatDateTime, parseCents } from '../utils/formatters';
import {
  Button,
  ConfirmDialog,
  Dialog,
  EmptyState,
  Field,
  Help,
  Input,
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

/**
 * Der Anker des Abschnitts „Schritte" auf dieser Seite.
 *
 * Die Feststellung ist der einzige Baustein, dessen Arbeit hier selbst liegt.
 * Sein „Öffnen" springt deshalb in den Abschnitt darunter — ohne Ziel bliebe
 * die Zeile als einzige ohne Knopf.
 */
const STEPS_ANCHOR = 'jahresabschluss-schritte';

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

/**
 * Die Arten der Rechnungsabgrenzung im Klartext. Sie stehen hier wie auf der
 * Seite der Abschlussbausteine: der Paragraph gehört in die Erklärung, nicht in
 * die Tabellenzelle.
 */
const ACCRUAL_KIND_LABELS: Record<AccrualKind, string> = {
  active: 'Ausgabe für spätere Jahre',
  passive: 'Einnahme für spätere Jahre',
  disagio: 'Damnum oder Disagio aus einem Darlehen',
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
 * (§6.2 Nr. 4) — mit derselben Diagnose, die auch die Ablehnung begründen würde.
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
 * Ein Hinweisstreifen hat einen Satz Ursache und einen Satz nächsten Schritt
 * (§15.1); die Kette der Vorträge und die Liste der betroffenen Konten gehören
 * deshalb in den `title` des gesperrten Knopfes, wo sie abrufbar bleiben, ohne
 * dauerhaft in der Arbeitsansicht zu stehen (§15.2).
 */
function carryForwardDetail(preview: CarryForwardPreview | null): string {
  if (!preview) return '';

  if (!preview.isBalanced) {
    return (
      `Aktiva, Passiva und das Jahresergebnis von ${formatCents(preview.netIncome)} stimmen nicht ` +
      `zusammen; ein Vortrag würde die Differenz ins Geschäftsjahr ${preview.toYear} mitnehmen.` +
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
  /** Der Weg von einem offenen Baustein zu der Ansicht, die ihn bucht. */
  onNavigate?: NavigateFn;
}

export const ClosingPage: React.FC<ClosingPageProps> = ({
  year,
  onFiscalYearChanged,
  onNavigate,
}) => {
  // Feststellung, Vortrag und Arbeitnehmerzahl ändern die Bücher: im
  // Prüfermodus gesperrt, der Stand bleibt lesbar (§10.4).
  const writeLock = useWriteLock();
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

  // Der Weg zurück im geführten Weg (Architektur 6.3): ein übersprungener
  // Baustein wird wieder aufgenommen, solange das Jahr nicht festgeschrieben
  // ist. Er steht hier und nicht nur unter „Abschlussbausteine", weil die
  // Entscheidung hier zu sehen ist — und was man sieht, muss man auch
  // zurücknehmen können.
  const [stepReopen, setStepReopen] = useState<ClosingStepView | null>(null);
  const [stepReopenReason, setStepReopenReason] = useState('');
  const [stepReopenFieldError, setStepReopenFieldError] = useState('');
  const [stepReopenBackendError, setStepReopenBackendError] = useState('');

  const [confirmCarry, setConfirmCarry] = useState(false);
  const [busy, setBusy] = useState(false);

  // Die Größenklasse richtet sich nach der Bilanzsumme und damit nach einem
  // Abschluss, der aufgeht. Sie bekommt deshalb einen eigenen Fehlerpfad:
  // schlägt sie fehl, bleiben Schritte und Saldenvortrag benutzbar.
  const [sizeClass, setSizeClass] = useState<SizeClass | null>(null);
  const [sizeClassError, setSizeClassError] = useState('');
  const [employees, setEmployees] = useState('0');
  const [savingEmployees, setSavingEmployees] = useState(false);
  // Der Vorjahresumsatz entscheidet über die Übergangsfrist des § 27 Abs. 38
  // Nr. 2 UStG. Er steht am Geschäftsjahr, weil er für dieses Jahr gilt.
  const [priorRevenue, setPriorRevenue] = useState('');
  // Ein unleserlicher Betrag ist ein Fehler der Eingabe: er steht am Feld und
  // nicht als Toast, der nach vier Sekunden weg ist (§10.4).
  const [priorRevenueError, setPriorRevenueError] = useState('');
  const [savingRevenue, setSavingRevenue] = useState(false);

  // Die Bausteine des Abschlusses — Abgrenzung, Rückstellungen, Inventurwert,
  // Verrechnungen. Gearbeitet wird an ihnen unter „Abschlussbausteine"; hier
  // stehen sie, weil der Weg zur Feststellung über sie führt und ein offener
  // Baustein sonst erst am Prüflauf auffiele. Sie bekommen einen eigenen
  // Fehlerpfad: fehlen sie, bleibt der Abschlussstand benutzbar.
  const [closingSteps, setClosingSteps] = useState<ClosingSteps | null>(null);
  const [closingStepsError, setClosingStepsError] = useState('');

  // Die Eröffnungsbilanz des Umsteigers (ARC-05). Der Baustein steht nur im
  // ersten Geschäftsjahr und nur, solange keine Eröffnungsbuchungen da sind:
  // eine zweite verdoppelte die Bestände. Die Regel selbst hält das Backend —
  // eine Regel, die an der Sichtbarkeit eines Knopfes hängt, ist keine.
  const [isFirstYear, setIsFirstYear] = useState(false);
  const [openingBooked, setOpeningBooked] = useState(false);
  const [openingOpen, setOpeningOpen] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    setPreviewError('');
    try {
      const closing = await Api.getClosingState(year);
      setState(closing);
      setEmployees(String(closing.fiscalYear.averageEmployees ?? 0));
      setPriorRevenue(
        closing.fiscalYear.priorYearRevenue
          ? formatCents(closing.fiscalYear.priorYearRevenue, '').trim()
          : '',
      );
      try {
        setSizeClass(await Api.getSizeClass(year));
        setSizeClassError('');
      } catch (e) {
        setSizeClass(null);
        setSizeClassError(message(e));
      }
      try {
        setClosingSteps(await Api.getClosingSteps(year));
        setClosingStepsError('');
      } catch (e) {
        setClosingSteps(null);
        setClosingStepsError(message(e));
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
      try {
        // Ob dieses Jahr das erste ist, sagt die Liste der Geschäftsjahre. Die
        // Buchungen werden nur in diesem Fall geholt — sonst wäre es eine
        // Abfrage über alle Jahre für eine Frage, die sich nicht mehr stellt.
        const years = await Api.getFiscalYears();
        const earliest = years.reduce(
          (min, entry) => (min === 0 || entry.year < min ? entry.year : min),
          0,
        );
        const first = earliest === year;
        const foundation = await Api.getFoundationState();
        setIsFirstYear(first && !foundation?.foundation);
        setOpeningBooked(
          first
            ? (await Api.getAllJournalEntries()).some(
                (entry) => entry.fiscalYear === year && entry.source === 'opening',
              )
            : false,
        );
      } catch {
        // Ohne Antwort bleibt der Baustein aus: ihn anzubieten, ohne zu wissen,
        // ob schon Eröffnungsbuchungen stehen, führte zu doppelten Beständen.
        setIsFirstYear(false);
        setOpeningBooked(false);
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

  /**
   * Der Gesamtumsatz des Vorjahres. An ihm hängt § 27 Abs. 38 Nr. 2 UStG: bis
   * 800.000 € darf 2027 noch eine sonstige Rechnung ohne strukturierten
   * Datensatz ausgestellt werden. Vorbelegt aus der Gewinn- und
   * Verlustrechnung, überschreibbar — der Gesamtumsatz des § 19 Abs. 3 UStG ist
   * nicht dasselbe wie die Umsatzerlöse des § 275 HGB.
   */
  async function savePriorRevenue() {
    const amount = parseCents(priorRevenue);
    if (amount === null || amount < 0) {
      setPriorRevenueError('Erwartet wird ein Betrag ab null, etwa 800.000,00.');
      return;
    }
    setPriorRevenueError('');
    setSavingRevenue(true);
    try {
      await Api.setPriorYearRevenue(year, amount);
      await load();
    } catch (e) {
      setPriorRevenueError(message(e));
    } finally {
      setSavingRevenue(false);
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

  async function submitStepReopen() {
    if (!stepReopen) return;
    if (!stepReopenReason.trim()) {
      setStepReopenFieldError(
        'Der Grund fehlt. Er tritt an die Stelle des Grundes, mit dem der Schritt übergangen wurde.',
      );
      return;
    }
    setBusy(true);
    try {
      setClosingSteps(await Api.reopenClosingStep(year, stepReopen.key, stepReopenReason));
      setClosingStepsError('');
      setStepReopen(null);
      setStepReopenReason('');
    } catch (e) {
      setStepReopenBackendError(message(e));
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

  // Der geführte Weg durch die Bausteine (Architektur 6.3): wie weit er ist und
  // welcher Schritt als Nächster dran ist. Übersprungene zählen als erledigt —
  // sie sind eine Entscheidung mit Grund und keine offene Arbeit.
  const moduleSteps = closingSteps?.steps ?? [];
  // Die Zähler kommen aus dem Backend (ClosingSteps.doneCount/skippedCount/
  // total). Hier nachzurechnen hieße, dieselbe Frage zweimal zu beantworten —
  // und die zweite Antwort wiche ab, sobald ein Zustand dazukommt.
  const totalSteps = closingSteps?.total ?? moduleSteps.length;
  const doneSteps = (closingSteps?.doneCount ?? 0) + (closingSteps?.skippedCount ?? 0);
  const nextStep = moduleSteps.find((step) => step.state === 'open') ?? null;

  /**
   * Öffnet die Arbeit eines Bausteins: die Seite, auf der sie liegt, oder — bei
   * der Feststellung — den Abschnitt dieser Seite. Ein Schritt ohne Knopf
   * benennt eine Arbeit und führt nicht zu ihr (Architektur 6.3).
   */
  const openStep = (key: ClosingStepKey) => {
    const target = stepTarget(key);
    if (!target) return;
    if (target.kind === 'anchor') {
      document.getElementById(target.anchor)?.scrollIntoView({ behavior: 'smooth', block: 'start' });
      return;
    }
    onNavigate?.(target.tab, target.params);
  };

  /** Ob der Knopf überhaupt etwas tut: ohne Navigation führt kein Seitenziel. */
  const canOpenStep = (key: ClosingStepKey) => {
    const target = stepTarget(key);
    if (!target) return false;
    return target.kind === 'anchor' || Boolean(onNavigate);
  };

  const steps: Step[] = [
    {
      key: 'commitment',
      title: 'Jahres-Festschreibung',
      status: state.hasYearCommitment ? 'festgeschrieben' : 'offen',
      date: state.hasYearCommitment ? state.committedUntil ?? '' : '',
      note: '',
      action: !state.hasYearCommitment && onNavigate ? <Button variant="secondary" size="sm" onClick={() => onNavigate('deadlines')}>Zur Festschreibung</Button> : undefined,
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
      title: 'Offenlegung vermerkt',
      status: fy.disclosedOn ? 'offengelegt' : 'offen',
      date: fy.disclosedOn ?? '',
      note: fy.disclosureNote ?? '',
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
                disabled={!state.canAdopt || writeLock.locked}
                title={writeLock.hint ?? (state.canAdopt ? undefined : state.blocker)}
                onClick={() => openStepDialog(state.nextStatus as FiscalYearStatus)}
              >
                {STEP_ACTION[state.nextStatus]}
              </Button>
            )}
          </div>
        }
      />

      {state.legalReserve?.applies && <Section title="Gesetzliche UG-Rücklage" className="mt-6"
        context={`Für den Abschluss ${year}; die Rücklage wird vor der Festschreibung gebildet.`}>
        <p className="text-body text-ink-muted">Jahresüberschuss {formatCents(state.legalReserve.netIncome)}, Verlustvortrag {formatCents(state.legalReserve.lossCarryForward)}. Davon ein Viertel: {formatCents(state.legalReserve.required)}. Bereits zugeführt: {formatCents(state.legalReserve.booked)}.</p>
        {state.legalReserve.difference !== 0 ? <Button className="mt-3" disabled={writeLock.locked || state.hasYearCommitment} title={writeLock.hint || (state.hasYearCommitment ? 'Der Abschluss ist festgeschrieben. Für eine Berichtigung muss zuerst der Abschluss geöffnet werden.' : undefined)}
          onClick={() => { void (async () => { try { await Api.bookLegalReserve(year); toast.success('Gesetzliche Rücklage im Abschlussjahr gebucht.'); await load(); } catch (e) { toast.error(message(e)); } })(); }}>
          Rücklage um {formatCents(state.legalReserve.difference)} anpassen
        </Button> : <p className="text-body text-positive-text mt-2">Die erforderliche Rücklage ist berücksichtigt.</p>}
      </Section>}
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

      {/* Der Umsteiger: im ersten Geschäftsjahr fehlen die Anfangsbestände,
          weil es kein Vorjahr gibt, aus dem vorgetragen werden könnte. Sie
          kommen aus der Schlussbilanz des Altsystems (ARC-05). Nach der
          Buchung verschwindet der Baustein — eine zweite Eröffnungsbilanz
          verdoppelte die Bestände. */}
      {isFirstYear && (
        <Section helpSummary="Übernehmen Sie Ihre Anfangsbestände aus dem bisherigen Buchhaltungsprogramm."
          title="Eröffnungsbilanz des Umsteigers"
          context={
            openingBooked
              ? 'Die Anfangsbestände sind übernommen'
              : 'Anfangsbestände aus dem Altsystem übernehmen'
          }
          explain={
            <>
              Wer die Buchführung aus einem anderen Programm übernimmt, braucht die
              Anfangsbestände: Sachkonten gegen das Saldenvortragskonto 9000, jede offene
              Forderung und Verbindlichkeit einzeln gegen 9008 und 9009. Als Beleg dient die
              Schlussbilanz des Altsystems; sie ist zuvor als Beleg abzulegen (§ 146 Abs. 1 AO).
              Die Herkunftskennung jeder Position bleibt an der Buchung stehen.
            </>
          }
          action={
            <Button
              variant="secondary"
              disabled={writeLock.locked || openingBooked}
              title={
                writeLock.hint ??
                (openingBooked
                  ? 'Die Eröffnungsbilanz ist gebucht. Eine Korrektur geschieht durch den Storno der vorhandenen Buchungen.'
                  : undefined)
              }
              onClick={() => setOpeningOpen(true)}
            >
              Eröffnungsbilanz erfassen
            </Button>
          }
        >
          {openingBooked ? (
            <p className="text-body text-ink-muted">
              Die Anfangsbestände stehen im Journal als Buchungen der Herkunft „Eröffnungsbilanz".
            </p>
          ) : (
            <EmptyState
              title="Noch keine Anfangsbestände übernommen"
              description="Ohne sie beginnt das erste Geschäftsjahr bei null — die Bilanz zeigte dann weder Bank noch Forderungen aus der Zeit davor."
            />
          )}
        </Section>
      )}

      <Section helpSummary="Hier halten Sie fest, wie weit die Erstellung und Bestätigung Ihres Abschlusses ist."
        id={STEPS_ANCHOR}
        title="Schritte"
        context="Von der Festschreibung bis zur Offenlegung"
        explain={
          <>
            § 242 HGB verlangt zum Ende jedes Geschäftsjahres einen Abschluss; unterzeichnet wird er
            unter Angabe des Datums (§ 245 HGB). Festgestellt wird er von den Gesellschaftern
            (§ 42a Abs. 2 GmbHG). Ab der Feststellung nimmt das Geschäftsjahr keine Buchung mehr an;
            zurück geht es nur über die Rücksetzung, und die verlangt einen Grund.
          </>
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

      <Section helpSummary="Erledigen Sie die nötigen Abschlussbuchungen, bevor Sie die Bilanz aufstellen."
        title="Abschlussbausteine"
        context={
          closingSteps
            ? `${doneSteps} von ${totalSteps} erledigt · Stichtag ${formatDate(closingSteps.cutoff)}`
            : 'Die Arbeit, die zur Feststellung führt'
        }
        explain={
          <>
            Bevor ein Abschluss aufgestellt wird, sind die Abschlussbuchungen zu machen:
            Abschreibungen, Rechnungsabgrenzung, Rückstellungen, der Inventurwert der Vorräte, die
            Umsatzsteuer-Verrechnung und die Steuerrückstellung. Der Stand folgt, wo möglich, aus
            den Daten. Ein bewusst ausgelassener Baustein wird übersprungen — mit Grund, damit
            später erkennbar bleibt, dass er nicht vergessen wurde.
            Wenn Sie beispielsweise keine Vorräte und keine Fremdwährungsposten haben,
            überspringen Sie diese Prüfungen mit genau dieser Begründung.
          </>
        }
        action={
          onNavigate && (
            <Button variant="secondary" onClick={() => onNavigate('closingmodules')}>
              Bausteine bearbeiten
            </Button>
          )
        }
      >
        {closingStepsError ? (
          <div className="flex items-start gap-2.5 rounded-control border border-negative-line bg-negative-soft px-4 py-3">
            <AlertCircle className="w-4 h-4 mt-0.5 shrink-0 text-negative" strokeWidth={1.5} />
            <p className="text-body text-negative-text">{closingStepsError}</p>
          </div>
        ) : (closingSteps?.steps ?? []).length === 0 ? (
          <p className="text-body text-ink-muted">
            Für dieses Geschäftsjahr sind keine Bausteine hinterlegt.
          </p>
        ) : (
          <>
            {/* Der Fortschritt steht über der Liste, weil er die Frage
                beantwortet, mit der jemand hierher kommt: wie weit ist der
                Abschluss? Die Zahl der Schritte kommt aus dem Backend und wird
                hier nicht gesetzt — sie stünde falsch da, sobald ein Baustein
                dazukommt. */}
            <Progress
              className="mb-5"
              label="Weg zum Abschluss"
              value={progressPercent(closingSteps)}
              detail={`${doneSteps} von ${totalSteps} Schritten`}
            />

            {nextStep && (
              <div className="flex items-center justify-between gap-4 mb-5">
                <p className="text-body text-ink-muted">
                  Als Nächstes: <span className="text-ink">{nextStep.label}</span>
                </p>
                {canOpenStep(nextStep.key) && (
                  <Button variant="secondary" size="sm" onClick={() => openStep(nextStep.key)}>
                    Schritt öffnen
                  </Button>
                )}
              </div>
            )}

          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th className="w-12" numeric>
                  Nr.
                </Th>
                <Th>Baustein</Th>
                <Th className="w-40">Stand</Th>
                <Th>Woran der Stand liegt</Th>
                <Th className="w-56" aria-label="Ziel" />
              </Tr>
            </Thead>
            <Tbody>
              {(closingSteps?.steps ?? []).map((step) => (
                <Tr key={step.key}>
                  <Td numeric className="text-ink-subtle">
                    {step.order}
                  </Td>
                  <Td>
                    <span className="inline-flex items-center gap-1.5">
                      {step.label}
                      <Help summary="Hier erfahren Sie, was bei diesem Abschlussschritt zu erledigen ist." label={`Erklärung zu ${step.label}`}>{step.hint}</Help>
                    </span>
                  </Td>
                  <Td>
                    {/* „Übersprungen" ist kein Zustand des Statusvokabulars: es
                        beschreibt eine Entscheidung, nicht den Stand einer
                        Buchung. Es zeigt deshalb kein erfundenes Abzeichen,
                        sondern dieselbe neutrale Form wie auf der
                        Bausteinseite — sonst sähe derselbe Schritt an zwei
                        Stellen verschieden aus. */}
                    {step.state === 'skipped' ? (
                      <SkippedMark />
                    ) : (
                      <StatusBadge status={step.state === 'done' ? 'erledigt' : 'offen'} />
                    )}
                  </Td>
                  <Td className="text-ink-muted whitespace-normal">
                    {step.state === 'skipped' ? step.reason || '—' : step.detail || '—'}
                  </Td>
                  <Td className="pl-0 text-right">
                    <div className="flex justify-end gap-1">
                      {/* Zurück, solange nicht festgeschrieben (Architektur
                          6.3). Danach bleibt der Knopf stehen und nennt am
                          Zeiger den Grund — ein Knopf, der verschwindet,
                          beantwortet die Frage nicht, warum er weg ist. */}
                      {step.state === 'skipped' && (
                        <Button
                          variant="quiet"
                          size="sm"
                          disabled={busy || writeLock.locked || !closingSteps?.reopenable}
                          title={writeLock.hint || closingSteps?.reopenBlocker}
                          onClick={() => {
                            setStepReopen(step);
                            setStepReopenReason('');
                            setStepReopenFieldError('');
                            setStepReopenBackendError('');
                          }}
                        >
                          Zurücknehmen
                        </Button>
                      )}
                      {canOpenStep(step.key) && (
                        <Button variant="quiet" size="sm" onClick={() => openStep(step.key)}>
                          Öffnen
                        </Button>
                      )}
                    </div>
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
          </>
        )}
      </Section>

      <Section helpSummary="Die Unternehmensgröße bestimmt den Umfang der Abschlussunterlagen."
        title="Größenklasse"
        context={
          sizeClass
            ? `${SIZE_LABELS[sizeClass.class] ?? sizeClass.class}`
            : 'Bilanzsumme, Umsatzerlöse und Arbeitnehmerzahl entscheiden'
        }
        explain={
          <>
            Die §§ 267, 267a HGB ordnen eine Kapitalgesellschaft nach Bilanzsumme, Umsatzerlösen und
            Arbeitnehmerzahl ein; zwei der drei Merkmale entscheiden. Die Rechtsfolge tritt erst
            ein, wenn zwei aufeinander folgende Stichtage dieselbe Klasse ergeben (§ 267 Abs. 4
            HGB).
          </>
        }
      >
        {/* Ab der Feststellung bleibt das Feld an seinem Platz und wird
            deaktiviert (§11.5): an der Zahl hängt über die Größenklasse die
            Gliederungstiefe, und die eines festgestellten Abschlusses steht.
            Der Service weist die Änderung ebenfalls ab — die Sperre gehört
            nicht allein in die Oberfläche. */}
        <Field helpSummary="Geben Sie die durchschnittliche Zahl der Beschäftigten ohne Auszubildende an."
          label="Arbeitnehmer im Jahresdurchschnitt"
          explain="Durchschnitt der an den vier Quartalsstichtagen Beschäftigten (§ 267 Abs. 5 HGB); Auszubildende bleiben außer Betracht."
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
              disabled={adopted || writeLock.locked}
              title={writeLock.hint}
            >
              Übernehmen
            </Button>
          </div>
        </Field>

        <Field helpSummary="Der Umsatz des Vorjahres kann bestimmen, ab wann Sie E-Rechnungen ausstellen müssen."
          label="Gesamtumsatz des Vorjahres"
          error={priorRevenueError || undefined}
          explain="Der Vorjahresumsatz entscheidet über die Übergangsfrist der E-Rechnung (§ 27 Abs. 38 Nr. 2 UStG): Bis 800.000 € darf im Jahr 2027 noch eine sonstige Rechnung ohne strukturierten Datensatz ausgestellt werden, ab 2028 nicht mehr. Vorbelegt ist der Wert aus der Gewinn- und Verlustrechnung des Vorjahres — der Gesamtumsatz des § 19 Abs. 3 UStG ist damit nicht identisch, deshalb ist er überschreibbar."
          className="mt-4 max-w-sm"
        >
          <div className="flex items-center gap-2">
            <Input
              value={priorRevenue}
              inputMode="decimal"
              align="right"
              placeholder="0,00"
              onChange={(e) => {
                setPriorRevenue(e.target.value);
                setPriorRevenueError('');
              }}
            />
            <Button
              variant="secondary"
              onClick={savePriorRevenue}
              loading={savingRevenue}
              disabled={writeLock.locked}
              title={writeLock.hint}
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
              {/* Die Norm steht hinter dem Erklärzeichen und nicht in einer
                  eigenen Spalte: in der Arbeitsansicht zählt, was gilt, und
                  nicht, woraus es folgt (Architektur 6.4). */}
              <Table density="kompakt">
                <Thead>
                  <Tr>
                    <Th>Merkmal oder Folge</Th>
                    <Th className="w-96">Wert</Th>
                  </Tr>
                </Thead>
                <Tbody>
                  {sizeClassRows(sizeClass).map((row) => (
                    <Tr key={row.label}>
                      <Td>
                        <span className="inline-flex items-center gap-1.5">
                          {row.label}
                          <Help summary="Hier erfahren Sie, wie dieser Betrag für den Abschluss berechnet wird." label={`Erklärung zu ${row.label}`}>
                            {row.explanation}
                          </Help>
                        </span>
                      </Td>
                      <Td className={row.numeric ? 'num' : 'whitespace-normal'}>{row.value}</Td>
                    </Tr>
                  ))}
                </Tbody>
              </Table>
            </div>
          )
        )}
      </Section>

      <Section helpSummary="Übernehmen Sie die Endbestände als Anfangsbestände ins nächste Geschäftsjahr."
        title="Saldenvortrag ins Folgejahr"
        context={preview ? `${preview.fromYear} → ${preview.toYear}` : `${year} → ${state.nextYear}`}
        explain={
          <>
            § 252 Abs. 1 Nr. 1 HGB verlangt, dass die Eröffnungsbilanz mit der Schlussbilanz des
            Vorjahres übereinstimmt. Vorgetragen werden die Bilanzkonten gegen 9000, die offenen
            Posten der Debitoren gegen 9008 und der Kreditoren gegen 9009. Das Jahresergebnis geht
            auf den Gewinn- oder Verlustvortrag; über seine Verwendung wird gesondert beschlossen.
          </>
        }
        action={
          <Button
            variant="secondary"
            icon={<Layers className="w-4 h-4" strokeWidth={1.5} />}
            disabled={carryBlocked || busy || writeLock.locked}
            title={writeLock.hint ?? carryTitle}
            onClick={() => setConfirmCarry(true)}
          >
            {carryLabel}
          </Button>
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
                    aus der GuV stammt statt aus einem Schlusssaldo. */}
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
                        <Help summary="Der Betrag zeigt den Stand des Kontos zum Jahresende." label="Erklärung zum Schlusssaldo">
                          Positive Beträge stehen im Soll, negative im Haben.
                        </Help>
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
                        {/* Auf dem Ergebniskonto steht unter „Schlusssaldo" der
                            Schlusssaldo des Vorjahres zuzüglich des Jahresergebnisses.
                            Ohne diesen Zusatz fände jeder, der die Zeile gegen die
                            Summen- und Saldenliste des Vorjahres hält, eine unerklärte
                            Abweichung — und zwar genau auf dem Konto, das über die
                            Bilanzidentität entscheidet. */}
                        {row.includesNetIncome && (
                          <span className="text-ink-muted">
                            {` · inkl. verbleibendem Jahresergebnis ${formatCents(preview.resultToCarry)}`}
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

            {/* Der Vortrag bucht die fälligen Auflösungen der
                Rechnungsabgrenzung im neuen Jahr gleich mit
                (internal/service/closing_service.go). Sie stehen deshalb hier:
                eine Freigabe, die Buchungen auslöst, die die Vorschau nicht
                nennt, wäre keine Freigabe (§8.2). */}
            {preview.accrualReleases.length > 0 && (
              <div className="mt-8">
                <h3 className="text-label text-ink-muted mb-2">
                  Auflösung der Rechnungsabgrenzung
                </h3>
                <p className="flex items-center gap-1.5 text-body text-ink-muted mb-3">
                  {`Der Vortrag bucht diese ${preview.accrualReleases.length === 1 ? 'Auflösung' : `${preview.accrualReleases.length} Auflösungen`} im Geschäftsjahr ${preview.toYear} mit.`}
                  <Help summary="Bereits zeitlich abgegrenzte Beträge werden im passenden Folgezeitraum berücksichtigt." label="Erklärung zur Auflösung der Rechnungsabgrenzung">
                    Der abgegrenzte Betrag geht an seinem eigenen Datum auf das Aufwands- oder
                    Ertragskonto zurück, zu dem er gehört. Die Abgrenzung selbst folgt aus § 250
                    HGB: Ausgaben vor dem Stichtag, die Aufwand einer bestimmten Zeit danach sind,
                    stehen bis dahin in der Bilanz.
                  </Help>
                </p>
                <Table density="kompakt">
                  <Thead>
                    <Tr>
                      <Th>Posten</Th>
                      <Th className="w-56">Art</Th>
                      <Th className="w-24">Konto</Th>
                      <Th className="w-32">Datum</Th>
                      <Th numeric className="w-40">
                        Betrag
                      </Th>
                    </Tr>
                  </Thead>
                  <Tbody>
                    {preview.accrualReleases.map((release) => (
                      <Tr key={`${release.accrualId}-${release.releaseId}`}>
                        <Td className="whitespace-normal">{release.text}</Td>
                        <Td className="text-ink-muted">
                          {ACCRUAL_KIND_LABELS[release.kind] ?? release.kind}
                        </Td>
                        <Td code>{release.account}</Td>
                        <Td>{formatDate(release.date)}</Td>
                        <Td numeric>{formatCents(release.amount)}</Td>
                      </Tr>
                    ))}
                    <Tr variant="sum">
                      <Td>Summe</Td>
                      <Td />
                      <Td />
                      <Td />
                      <Td numeric>
                        {formatCents(
                          preview.accrualReleases.reduce((sum, item) => sum + item.amount, 0),
                        )}
                      </Td>
                    </Tr>
                  </Tbody>
                </Table>
              </div>
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
            <Button
              variant="primary"
              loading={busy}
              disabled={writeLock.locked}
              title={writeLock.hint}
              onClick={() => void submitStep()}
            >
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

        {(stepStatus === 'adopted' || stepStatus === 'disclosed') && (
          <Field helpSummary={stepStatus === 'disclosed' ? 'Tragen Sie die Referenz des externen Übermittlungsnachweises ein.' : 'Geben Sie an, mit welchem Beschluss die Gesellschafter den Abschluss bestätigt haben.'}
            label={stepStatus === 'disclosed' ? 'Übermittlungsnachweis oder Auftragsnummer' : 'Beschlussbezug'}
            optional={stepStatus !== 'disclosed'}
            explain={stepStatus === 'disclosed' ? 'Tragen Sie die Referenz Ihres externen Übermittlungsnachweises ein. Dieser lokale Vermerk löst keine Offenlegung aus.' : 'Welcher Gesellschafterbeschluss den Abschluss festgestellt hat.'}
            className="mt-4"
          >
            <Input
              value={stepNote}
              onChange={(e) => setStepNote(e.target.value)}
              placeholder={stepStatus === 'disclosed' ? 'Auftragsnummer beim Unternehmensregister …' : 'Gesellschafterbeschluss vom …'}
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
        open={stepReopen !== null}
        onOpenChange={(next) => !next && setStepReopen(null)}
        title={stepReopen ? `${stepReopen.label} wieder aufnehmen` : ''}
        width="max-w-lg"
        footer={
          <>
            <Button variant="secondary" onClick={() => setStepReopen(null)}>
              Abbrechen
            </Button>
            <Button
              variant="primary"
              loading={busy}
              disabled={writeLock.locked}
              title={writeLock.hint}
              onClick={() => void submitStepReopen()}
            >
              Schritt wieder aufnehmen
            </Button>
          </>
        }
      >
        {stepReopen?.reason && (
          <p className="text-body text-ink-muted mb-5">
            {`Übersprungen wurde der Schritt mit: ${stepReopen.reason}`}
          </p>
        )}
        <Field helpSummary="Begründen Sie, warum Sie den übersprungenen Schritt wieder aufnehmen."
          label="Grund"
          error={stepReopenFieldError || undefined}
          explain="Geht ins Änderungsprotokoll und steht dort neben dem Grund des Überspringens."
        >
          <Textarea
            rows={3}
            value={stepReopenReason}
            onChange={(e) => {
              setStepReopenReason(e.target.value);
              setStepReopenFieldError('');
            }}
          />
        </Field>

        {stepReopenBackendError && (
          <div className="mt-5 flex items-start gap-2.5 rounded-control border border-negative-line bg-negative-soft px-4 py-3">
            <AlertCircle className="w-4 h-4 mt-0.5 shrink-0 text-negative" strokeWidth={1.5} />
            <p className="text-body text-negative-text">{stepReopenBackendError}</p>
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
            <Button
              variant="danger"
              loading={busy}
              disabled={writeLock.locked}
              title={writeLock.hint}
              onClick={() => void submitReopen()}
            >
              Feststellung zurücksetzen
            </Button>
          </>
        }
      >
        <Field helpSummary="Halten Sie den Grund für diese Änderung fest."
          label="Grund"
          error={reopenFieldError || undefined}
          explain="Geht ins Änderungsprotokoll und bleibt dort."
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
            ? `${preview.entries} ${preview.entries === 1 ? 'Buchung' : 'Buchungen'} zum ${formatDate(preview.bookingDate)} im Geschäftsjahr ${preview.toYear}.` +
              // Die Auflösungen der Abgrenzung sind eigene Buchungen mit
              // eigenem Datum; sie in `entries` unterzuschlagen hieße, die
              // Freigabe über ihren Umfang zu täuschen.
              (preview.accrualReleases.length > 0
                ? ` Dazu ${preview.accrualReleases.length === 1 ? 'kommt eine Auflösung' : `kommen ${preview.accrualReleases.length} Auflösungen`} der Rechnungsabgrenzung über ${formatCents(preview.accrualReleases.reduce((sum, item) => sum + item.amount, 0))}.`
                : '') +
              (preview.alreadyCarried
                ? ' Der bestehende Vortrag wird zuvor per Generalumkehr zurückgenommen.'
                : '')
            : ''
        }
        confirmLabel={carryLabel}
        onConfirm={() => void runCarryForward()}
      />

      <OpeningBalanceDialog
        open={openingOpen}
        fiscalYear={year}
        startDate={fy.startDate}
        onOpenChange={setOpeningOpen}
        onNavigate={onNavigate}
        onBooked={async (result) => {
          setOpeningOpen(false);
          await load();
          toast.success(
            `Eröffnungsbilanz gebucht: ${result.entries.length} Buchungen über ${formatCents(
              result.debitTotal,
            )}.`,
          );
        }}
      />
    </div>
  );
};


/**
 * Der Fortschritt des Abschlusses in Prozent; null heißt: noch keine Schritte.
 *
 * Aus den Zählern des Backends und nicht aus einer festen Elf: eine Zahl in der
 * Ansicht stünde falsch da, sobald ein Baustein dazukommt. Gezählt wird auch
 * nicht selbst nach — erledigt und übersprungen zusammen sind der Fortschritt,
 * und diese Entscheidung trifft der Abschlussassistent.
 */
function progressPercent(steps: ClosingSteps | null): number | null {
  const total = steps?.total ?? 0;
  if (!steps || total === 0) return null;
  return Math.round(((steps.doneCount + steps.skippedCount) / total) * 100);
}

/**
 * Wohin ein Baustein führt.
 *
 * Der geführte Weg (Architektur 6.3) verlangt, dass jeder Schritt zu seiner
 * Arbeit führt. Sechs Bausteine wohnen auf den Abschlussbausteinen, drei auf
 * den Nebenpflichten — und die übrigen auf eigenen Seiten: die Abschreibungen
 * im Anlagenverzeichnis, der Prüfbericht bei den Steuerfristen (dort läuft er
 * vor der Festschreibung; die Seite „Sicherheit & Protokoll" zeigt nur die
 * Läufe von gestern), Bilanz und GuV in den Auswertungen, die Offenlegung in
 * der E-Bilanz. Die Feststellung bleibt auf dieser Seite und springt in den
 * Abschnitt darunter.
 */
const STEP_PAGES: Partial<Record<ClosingStepKey, TabType>> = {
  depreciation: 'assets',
  check_run: 'deadlines',
  statement: 'reports',
  disclosure: 'ebilanz',
};

/**
 * Ziel eines Bausteins: eine andere Seite oder ein Abschnitt dieser Seite.
 *
 * „Öffnen" springt an die Stelle, an der der Baustein wohnt, und öffnet nicht
 * selbst dessen Dialog. Das ist bewusst so: der Dialog mit Vorschau in
 * Vorgangssprache und aufklappbarem Buchungssatz (Regel 11.1) gehört dem
 * Baustein und lebt dort mit seinen Eingaben, seinem Ladezustand und seiner
 * Sperre. Ihn von hier aus fernzusteuern hieße, jede dieser Ansichten ein
 * zweites Mal von außen zu öffnen — zwei Wege in denselben Dialog, von denen
 * einer irgendwann anders funktioniert. Der geführte Weg übernimmt hier den
 * Fortschritt („n von total"), die Reihenfolge und das Zurücknehmen; die
 * Buchung selbst bleibt beim Baustein.
 */
type StepTarget =
  | { kind: 'page'; tab: TabType; params: NavigationParams }
  | { kind: 'anchor'; anchor: string };

function stepTarget(key: ClosingStepKey): StepTarget | null {
  const moduleTab = STEP_TABS[key];
  if (moduleTab) return { kind: 'page', tab: 'closingmodules', params: { closingTab: moduleTab } };
  const obligation = STEP_OBLIGATIONS[key];
  if (obligation) return { kind: 'page', tab: 'obligations', params: { obligationsTab: obligation } };
  const page = STEP_PAGES[key];
  if (page) return { kind: 'page', tab: page, params: {} };
  if (key === 'adoption') return { kind: 'anchor', anchor: STEPS_ANCHOR };
  return null;
}


/**
 * Die Merkmale der Größenklasse mit ihrem Wert und ihrer Rechtsgrundlage.
 *
 * Die Grundlage steht hier neben dem Merkmal und wird in der Ansicht hinter dem
 * Erklärzeichen gezeigt: Wer den Abschluss aufstellt, will wissen, was gilt;
 * woraus es folgt, ist die zweite Frage.
 */
function sizeClassRows(
  sizeClass: SizeClass,
): { label: string; value: string; explanation: string; numeric?: boolean }[] {
  const obligations = sizeClass.obligations;
  return [
    {
      label: 'Bilanzsumme',
      value: formatCents(sizeClass.criteria.balanceSheetTotal),
      explanation: 'Rechtsgrundlage: § 267 Abs. 4a HGB',
      numeric: true,
    },
    {
      label: 'Umsatzerlöse',
      value: formatCents(sizeClass.criteria.revenue),
      explanation: 'Rechtsgrundlage: § 275 Abs. 2 Nr. 1 HGB',
      numeric: true,
    },
    {
      label: 'Arbeitnehmer im Jahresdurchschnitt',
      value: String(sizeClass.criteria.employees),
      explanation: 'Rechtsgrundlage: § 267 Abs. 5 HGB',
      numeric: true,
    },
    {
      label: 'Gliederungstiefe',
      value: SIZE_DEPTH_LABELS[obligations.depth] ?? obligations.depth,
      explanation: `Rechtsgrundlage: ${obligations.depthReference}`,
    },
    {
      label: 'Anhang',
      value: obligations.notesRequired ? 'Ja' : 'Entfällt bei vollständigen Ersatzangaben',
      explanation: `Rechtsgrundlage: ${obligations.notesReference}`,
    },
    {
      label: 'Lagebericht',
      value: obligations.managementReport ? 'Ja' : 'Nein',
      explanation: `Rechtsgrundlage: ${obligations.managementReportReference || '§ 264 Abs. 1 Satz 4 HGB'}`,
    },
    {
      label: 'Prüfung',
      value: obligations.auditRequired ? 'Ja' : 'Nein',
      explanation: `Rechtsgrundlage: ${obligations.auditReference || '§ 316 Abs. 1 Satz 1 HGB'}`,
    },
    {
      label: 'Aufstellungsfrist',
      value: `${obligations.preparationMonths} Monate`,
      explanation: `Rechtsgrundlage: ${obligations.preparationReference}`,
    },
    {
      label: 'Offenlegung',
      value: obligations.disclosureScope,
      explanation: `Rechtsgrundlage: ${obligations.disclosureScopeReference}`,
    },
  ];
}
