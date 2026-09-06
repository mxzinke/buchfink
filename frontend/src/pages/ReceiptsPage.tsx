import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  AlertTriangle,
  Download,
  FileCode,
  FilePlus,
  FileText,
  Paperclip,
  Plus,
  RefreshCw,
  ShieldCheck,
  Trash2,
} from 'lucide-react';
import type {
  Account,
  AdvanceTarget,
  AuditTrail,
  AdvanceTargetOption,
  Cents,
  Contact,
  Conversion,
  EInvoiceProposal,
  EntertainmentDetail,
  InputTaxFinding,
  JournalEntry,
  PostingGroup,
  PostingPreview,
  PostingWarning,
  Provision,
  Receipt,
  ReceiptFileInput,
  ReceiptFindings,
  ReceiptFileRole,
  ReceiptHeader,
  ReceiptKind,
  ReceiptRequest,
  Settlement,
  TaxRate,
  TaxTreatment,
  TaxTreatmentInfo,
  ValidationFinding,
  VendorAdvance,
} from '../types';
import {
  LIMIT_GIFT_PER_RECIPIENT,
  RETENTION_CLASS_LABELS,
  VALIDATION_FINDING_CLASS_LABELS,
  TAX_RATE_NONE,
  TAX_RATE_REDUCED,
  TAX_RATE_STANDARD,
} from '../types';
import type { NavigateFn } from '../components/Sidebar';
import { Api } from '../services/api';
import { usePostingLock, useWriteLock } from '../components/WriteLock';
import { RetentionDialog, SelfIssuedDialog } from '../components/LedgerForms';
import {
  formatCents,
  formatDate,
  formatDateOfTimestamp,
  formatDateTime,
  formatCentsPlain,
  formatSide,
  formatExchangeRate,
  formatPermille,
  parseCents,
  parseExchangeRate,
} from '../utils/formatters';
import {
  Button,
  Checkbox,
  Combobox,
  Dialog,
  EmptyState,
  Field,
  HelpPopover,
  Input,
  Notice,
  PageHeader,
  Section,
  Select,
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
  cn,
  toast,
  type Status,
} from '../components/ui';

/**
 * Belege.
 *
 * Ablegen und Buchen sind zwei Schritte. Eine XRechnung muss sofort ablegbar
 * sein — die GoBD verlangen die Aufbewahrung in der empfangenen Form —, aber sie
 * darf nicht buchbar sein, bevor eine Darstellung existiert: niemand soll eine
 * Buchung zu einem Beleg freigeben, den er nicht ansehen kann.
 *
 * Diese Seite rechnet nichts. Der Buchungssatz samt Netto, Steuer und Brutto
 * kommt aus der Vorschau des Backends.
 */

const ROLE_LABELS: Record<ReceiptFileRole, string> = {
  original: 'Original',
  structured: 'Strukturierter Teil',
  rendering: 'Darstellung',
  attachment: 'Anhang',
};

/**
 * Die Belegart (Entscheidung 9). Sie steht an der Liste, weil sie über den
 * nächsten Schritt entscheidet: eine Rechnung wird gebucht, ein Kontoauszug
 * belegt die Umsätze, die im Bankimport gebucht werden.
 */
const KIND_LABELS: Record<ReceiptKind, string> = {
  invoice: 'Rechnung',
  statement: 'Kontoauszug',
  self_issued: 'Eigenbeleg',
  letter: 'Handelsbrief',
  other: 'Sonstiger Beleg',
};

/** Belegart mit Standardwert: ältere Belege haben das Feld noch nicht. */
function kindOf(receipt: Receipt): ReceiptKind {
  return receipt.kind ?? 'invoice';
}

/** Die Belegarten zur Auswahl, in der Reihenfolge von KIND_LABELS. */
const KIND_ITEMS = (Object.keys(KIND_LABELS) as ReceiptKind[]).map((value) => ({
  value,
  label: KIND_LABELS[value],
}));

/**
 * Ob der Beleg seine Kopfdaten hat (BEL-02).
 *
 * Das Belegdatum ist die Weiche: das Backend hasht einen Beleg ohne es nach der
 * alten Form, und ohne es lässt sich die zeitgerechte Erfassung nicht
 * beurteilen (§ 146 Abs. 1 AO). Welche Felder darüber hinaus zum Buchen nötig
 * sind, entscheidet das Backend je Belegart — nachgebaut wird die Regel hier
 * nicht.
 */
function hasHeader(receipt: Receipt): boolean {
  return Boolean(receipt.documentDate);
}

/**
 * Der erste Tag, an dem der Beleg gelöscht werden darf.
 *
 * Das Backend rechnet die Frist und speichert ihren letzten Tag am Beleg; hier
 * wird nur der Folgetag gebildet, damit die Ansicht „aufzubewahren bis" und
 * „löschbar ab" nicht verwechselt. Ein aktiver Hold auf dem Geschäftsjahr hebt
 * das Datum auf — er steht auf der Seite „Nachweise", weil er das ganze Jahr
 * betrifft und nicht diesen einen Beleg.
 */
function earliestDeletion(retentionUntil: string | undefined): string {
  if (!retentionUntil) return '';
  const parsed = new Date(`${retentionUntil}T00:00:00Z`);
  if (Number.isNaN(parsed.getTime())) return '';
  parsed.setUTCDate(parsed.getUTCDate() + 1);
  return parsed.toISOString().slice(0, 10);
}

/**
 * Ob eine Belegart überhaupt gebucht wird — die Spiegelung von
 * `domain.ReceiptKind.RequiresBooking`.
 *
 * Zwei Arten werden nicht gebucht. Der Kontoauszug belegt die Umsätze, die im
 * Bankimport aus derselben Datei entstehen (Entscheidung 9). Der Handelsbrief
 * belegt eine Abrede und keinen Geschäftsvorfall — aus einem Angebot, einer
 * Bestellung, einer Kündigung folgt kein Buchungssatz. Beide als „noch zu
 * buchen" zu zählen ließe eine Aufgabe stehen, die sich nur durch eine fachlich
 * falsche Buchung erledigen ließe; der Prüflauf meldet sie folgerichtig nicht.
 *
 * Die Regel steht als eigene Funktion, weil Liste und Detailansicht sie beide
 * brauchen und zwei Kopien auseinanderliefen — das war der Fall, als die
 * Liste den Handelsbrief als offen führte und der Prüflauf nicht.
 */
function requiresBooking(kind: ReceiptKind): boolean {
  return kind !== 'statement' && kind !== 'letter';
}

/**
 * Ob dieser Beleg noch zu buchen ist: abgelegt und von einer Art, die gebucht
 * wird.
 */
function needsBooking(receipt: Receipt): boolean {
  return receipt.status === 'filed' && requiresBooking(kindOf(receipt));
}

/** Welche Belege die Liste zeigt. */
type ReceiptFilter = 'all' | 'open' | 'unclear';

/**
 * Ob ein Beleg zu klären ist.
 *
 * Zwei Fälle: Der strukturierte Datensatz verstößt gegen das Regelwerk, oder er
 * ist da und Buchfink konnte ihn nicht lesen. Beides steht zwischen dem Beleg
 * und seiner Buchung — und beides fällt sonst erst auf, wenn jemand den Beleg
 * zufällig öffnet.
 *
 * Die Buchung beendet den Befund nicht: Eine Rechnung mit fehlerhaftem
 * Datensatz bleibt zu klären, auch wenn sie schon im Journal steht — der
 * Vorsteuerabzug richtet sich nach der Rechnung und nicht nach der Buchung
 * (§ 15 Abs. 1 Satz 1 Nr. 1 UStG). Ein verworfener Beleg fällt heraus: An ihm
 * ist nichts mehr zu klären.
 */
function needsClarification(receipt: Receipt): boolean {
  if (receipt.status === 'discarded') return false;
  if (receipt.validationErrors > 0) return true;
  return receipt.files.some((f) => f.role === 'structured') && !receipt.detectedFormat;
}

/**
 * Das Status-Vokabular ist abgeschlossen (§11.3). Ein abgelegter Beleg ist ein
 * offener Vorgang, ein verworfener ist zurückgenommen — dafür gibt es keine
 * eigenen Wörter, und es sollen auch keine entstehen.
 */
const STATUS: Record<Receipt['status'], Status> = {
  filed: 'offen',
  sealed: 'gebucht',
  discarded: 'storniert',
};

/**
 * Die Kennung, mit der die Vorschau den fehlenden Leistungsnachweis meldet
 * (`service.serviceProofWarningCode`). Sie ist der Anlass, das Feld im
 * Buchungsdialog zum Pflichtfeld zu machen — die Grenze selbst steht im
 * Backend und wird hier nicht nachgebaut.
 */
const SERVICE_PROOF_WARNING = 'service_proof_required';

/** Hinweisfläche nach §6.2, Fall 4. Trägt Rand und Fläche immer zusammen. */
const NOTE = 'rounded-control border px-4 py-3';
const NOTE_TONE = {
  neutral: 'border-line-strong bg-sunken',
  attention: 'border-attention-line bg-attention-soft',
  positive: 'border-positive-line bg-positive-soft',
  negative: 'border-negative-line bg-negative-soft',
};

interface DraftPosition {
  postingGroup: string;
  net: string;
  taxRate: TaxRate;
  text: string;
  /**
   * Der abziehbare Vorsteueranteil in Promille als Text — leer heißt: voll
   * abziehbar. Als Text und nicht als Zahl, damit ein halb getippter Wert nicht
   * schon als 6 ‰ an die Vorschau geht.
   */
  inputTaxShare: string;
  inputTaxShareReason: string;
  /** Der Empfänger eines Geschenks: aus der Kartei oder als Freitext. */
  giftContactId: number;
  giftName: string;
  giftOccasion: string;
}

const emptyPosition = (group?: PostingGroup): DraftPosition => ({
  postingGroup: group?.key ?? '',
  net: '',
  taxRate: group?.defaultRate ?? TAX_RATE_STANDARD,
  text: '',
  inputTaxShare: '',
  inputTaxShareReason: '',
  giftContactId: 0,
  giftName: '',
  giftOccasion: '',
});

/**
 * Ob zu dieser Gruppe der Empfänger gehört.
 *
 * Die Frage beantwortet der Katalog und nicht diese Datei: an der Gruppe hängt
 * die Aufzeichnungspflicht des § 4 Abs. 7 EStG, und ohne den Empfänger weist
 * das Backend die Buchung zurück. Eine Liste von Gruppenschlüsseln hier wäre
 * eine zweite Fassung derselben Regel.
 */
function needsRecipient(group?: PostingGroup): boolean {
  return Boolean(group?.recipientRequired) || group?.limit === LIMIT_GIFT_PER_RECIPIENT;
}

export interface ReceiptsPageProps {
  /**
   * Der Filter, mit dem die Liste öffnet: „filed" zeigt die abgelegten, noch
   * nicht gebuchten Belege. Aus der Aufgabenliste führt sonst kein Weg zu den
   * Belegen, die die Aufgabe zählt — man käme auf der Gesamtliste an.
   */
  initialStatus?: string;
  /**
   * Der Beleg, der gleich aufzuschlagen ist — aus der Buchung im Journal
   * (GOB-02). Ohne ihn endete der Weg von der Bilanzposition zum Beleg auf der
   * Belegliste, und der Beleg wäre dort erneut zu suchen.
   */
  initialReceiptId?: number;
  /**
   * Der Weg vom Beleg zur Buchung und zum Kontoblatt (GOB-02 K2, BEL-01 K3).
   * Ohne ihn endete die Kette am Beleg, und die Buchung wäre im Journal erneut
   * zu suchen.
   */
  onNavigate?: NavigateFn;
}

export const ReceiptsPage: React.FC<ReceiptsPageProps> = ({
  initialStatus,
  initialReceiptId,
  onNavigate,
}) => {
  const [receipts, setReceipts] = useState<Receipt[]>([]);
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [groups, setGroups] = useState<PostingGroup[]>([]);
  const [treatments, setTreatments] = useState<TaxTreatmentInfo[]>([]);
  const [paymentAccounts, setPaymentAccounts] = useState<Account[]>([]);
  const [selected, setSelected] = useState<Receipt | null>(null);
  const [proposal, setProposal] = useState<EInvoiceProposal | null>(null);
  const [proposalError, setProposalError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [filing, setFiling] = useState(false);
  // „filed" ist der Zustand des Backends, „open" der Filter dieser Liste: beide
  // meinen den abgelegten, noch nicht gebuchten Beleg.
  const [filter, setFilter] = useState<ReceiptFilter>(initialStatus === 'filed' ? 'open' : 'all');
  // Der Beleg, dessen Kopfdaten gerade erfasst werden. Der Dialog folgt dem
  // Ablegen und lässt sich am offenen Beleg erneut öffnen (BEL-02).
  const [headerFor, setHeaderFor] = useState<Receipt | null>(null);
  // Der Eigenbeleg für den Vorgang ohne Fremdbeleg (BEL-01 K2, GOB-05 K1). Er
  // gehört an die Belegablage: er ist ein Beleg und entsteht nicht im Journal.
  const [selfIssuedOpen, setSelfIssuedOpen] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [list, contactList, groupList, treatmentList, accounts] = await Promise.all([
        Api.getReceipts(''),
        // Die auswählbaren und nicht alle Kontakte: ein nach einem
        // Löschverlangen gesperrter Geschäftspartner darf in keiner Auswahl
        // mehr auftauchen, bleibt aber in bestehenden Buchungen stehen.
        Api.getSelectableContacts(),
        Api.getPostingGroups('incoming'),
        Api.getTaxTreatments('incoming'),
        Api.getPaymentAccounts(),
      ]);
      setReceipts(list ?? []);
      setContacts(contactList ?? []);
      setGroups(groupList ?? []);
      setTreatments(treatmentList ?? []);
      setPaymentAccounts(accounts ?? []);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  // Der Beleg aus dem Navigationsziel wird ausgewählt, sobald die Liste da ist.
  // Er entscheidet nichts über den Filter: kommt der Weg aus einer gebuchten
  // Buchung, stünde der Beleg unter „Offen" nicht in der Liste.
  useEffect(() => {
    if (!initialReceiptId) return;
    const hit = receipts.find((receipt) => receipt.id === initialReceiptId);
    if (hit) {
      setFilter('all');
      setSelected(hit);
    }
  }, [initialReceiptId, receipts]);

  // Liegt ein strukturierter Teil vor, ist er die Buchungsquelle — der
  // Vorsteuerabzug ist nur aus ihm möglich (UStAE 14c.1 Abs. 4a Satz 4).
  useEffect(() => {
    setProposalError(null);
    if (!selected || selected.status !== 'filed') {
      setProposal(null);
      return;
    }
    if (!selected.files.some((f) => f.role === 'structured')) {
      setProposal(null);
      return;
    }
    let cancelled = false;
    Api.proposeFromEInvoice(selected.id)
      .then((p) => {
        if (!cancelled) setProposal(p);
      })
      .catch((e) => {
        // Kein Toast: warum aus diesem Datensatz kein Vorschlag wird — eine
        // Gutschrift, ein unbekannter Steuerkategoriecode —, muss stehen
        // bleiben, solange der Beleg offen ist. Eine Meldung, die nach fünf
        // Sekunden verschwindet, beantwortet die Frage genau einmal.
        if (!cancelled) {
          setProposal(null);
          setProposalError(e instanceof Error ? e.message : String(e));
        }
      });
    return () => {
      cancelled = true;
    };
  }, [selected]);

  // TODO: Drag & Drop über den Wails-Drop-Handler. Er liefert wie der Dialog
  // Pfade, sodass mehrere Megabyte große Scans nicht über die IPC-Grenze
  // müssen. Heute führt der einzige Weg über den Knopf.
  async function fileReceipt() {
    setFiling(true);
    try {
      const paths = await Api.selectReceiptFiles();
      if (!paths || paths.length === 0) return;

      // Die erste Datei ist die empfangene Form, weitere sind Anhänge. Rollen
      // lassen sich am abgelegten Beleg noch ändern.
      const files: ReceiptFileInput[] = paths.map((path, index) => ({
        path,
        role: index === 0 ? 'original' : 'attachment',
      }));
      const receipt = await Api.fileIncomingReceipt(files);
      toast.success(`Beleg ${receipt.receiptNumber} abgelegt.`);
      await load();
      setSelected(receipt);
      // Die Kopfdaten unmittelbar danach: bei einer E-Rechnung stehen sie
      // schon im Dialog und sind nur zu bestätigen, sonst liegt der Beleg
      // gerade vor. Wer sie später erfassen will, schließt den Dialog.
      setHeaderFor(receipt);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setFiling(false);
    }
  }

  const vendors = useMemo(() => contacts.filter((c) => c.type === 'vendor'), [contacts]);
  const openCount = receipts.filter(needsBooking).length;
  const unclearCount = receipts.filter(needsClarification).length;
  // Die Klärungsliste (RECH-07): ein Beleg mit Befund bleibt sonst zwischen
  // hundert gebuchten stehen und fällt erst bei der Voranmeldung auf.
  const visible = useMemo(() => {
    if (filter === 'open') return receipts.filter(needsBooking);
    if (filter === 'unclear') return receipts.filter(needsClarification);
    return receipts;
  }, [receipts, filter]);
  // Der Prüfermodus sperrt jede Erfassung; der Knopf sagt das, statt es dem
  // Anwender nach dem Dateidialog als Fehlermeldung zu zeigen (§10.4).
  const writeLock = usePostingLock();

  return (
    <div className="max-w-[1440px] mx-auto px-8 py-8">
      <PageHeader
        title="Belege"
        context={
          loading
            ? undefined
            : `${receipts.length} abgelegt · ${openCount} zu buchen · ${unclearCount} zu klären`
        }
        explain={
          <>
            Belege werden zuerst abgelegt und dann gebucht. Ein Beleg kann aus mehreren Dateien
            bestehen: Eine ZUGFeRD-Rechnung ist ein PDF mit eingebettetem XML, eine XRechnung ist
            reines XML und braucht vor dem Buchen eine erzeugte Darstellung.
          </>
        }
        action={
          <div className="flex items-center gap-2">
            <Button
              variant="secondary"
              disabled={loading}
              onClick={() => void load()}
              icon={<RefreshCw className={cn('w-4 h-4', loading && 'animate-spin')} strokeWidth={1.5} />}
            >
              Aktualisieren
            </Button>
            <Button
              variant="secondary"
              disabled={writeLock.locked}
              title={writeLock.hint}
              onClick={() => setSelfIssuedOpen(true)}
              icon={<FilePlus className="w-4 h-4" strokeWidth={1.5} />}
            >
              Eigenbeleg erstellen
            </Button>
            <Button
              variant="primary"
              loading={filing}
              disabled={writeLock.locked}
              title={writeLock.hint}
              onClick={() => void fileReceipt()}
              icon={<Plus className="w-4 h-4" strokeWidth={1.5} />}
            >
              Beleg ablegen
            </Button>
          </div>
        }
      />

      <div className="mt-6 grid grid-cols-1 xl:grid-cols-[19rem_minmax(0,1fr)] gap-8">
        <aside className="xl:border-r xl:border-line xl:pr-6">
          <Select
            className="mb-4"
            items={[
              { value: 'all', label: `Alle Belege (${receipts.length})` },
              { value: 'open', label: `Zu buchen (${openCount})` },
              { value: 'unclear', label: `Zu klären (${unclearCount})` },
            ]}
            value={filter}
            onValueChange={(next) => setFilter(next as ReceiptFilter)}
          />
          <ReceiptList
            receipts={visible}
            loading={loading}
            filtered={filter !== 'all'}
            onResetFilter={() => setFilter('all')}
            selectedId={selected?.id}
            onSelect={setSelected}
          />
        </aside>

        {selected ? (
          <ReceiptDetail
            key={selected.id}
            receipt={selected}
            vendors={vendors}
            contacts={contacts}
            groups={groups}
            treatments={treatments}
            paymentAccounts={paymentAccounts}
            proposal={proposal}
            proposalError={proposalError}
            onChanged={async (updated) => {
              setSelected(updated);
              await load();
            }}
            onEditHeader={() => setHeaderFor(selected)}
            onNavigate={onNavigate}
            onBooked={async (entryNumber) => {
              toast.success(`Beleg gebucht als ${entryNumber}.`);
              setSelected(null);
              await load();
            }}
          />
        ) : (
          <EmptyState
            icon={<FileText className="w-6 h-6" strokeWidth={1.5} />}
            title="Kein Beleg ausgewählt"
            description="Links einen Beleg wählen, um ihn anzusehen und zu buchen."
          />
        )}
      </div>

      <SelfIssuedDialog
        open={selfIssuedOpen}
        onOpenChange={setSelfIssuedOpen}
        onCreated={async (receipt) => {
          await load();
          setSelected(receipt);
        }}
      />

      <ReceiptHeaderDialog
        receipt={headerFor}
        onClose={() => setHeaderFor(null)}
        onSaved={async (updated) => {
          setHeaderFor(null);
          setSelected(updated);
          await load();
        }}
      />
    </div>
  );
};

// -------------------------------------------------------------------------

/**
 * Die Kopfdaten eines Belegs (BEL-02).
 *
 * Sie werden beim Ablegen abgefragt und nicht erst beim Buchen: Belegdatum,
 * Aussteller und Betrag stehen auf dem Papier, das gerade in der Hand liegt,
 * und wer sie erst Wochen später sucht, sucht sie zweimal. Freiwillig bleiben
 * sie trotzdem — ein Beleg muss sich sofort ablegen lassen (GoBD Rz. 47);
 * Pflicht werden sie beim Buchen, und das prüft das Backend.
 *
 * Bei einer E-Rechnung sind die Felder schon belegt: sie kommen aus dem
 * strukturierten Teil und werden hier nur bestätigt.
 */
const ReceiptHeaderDialog: React.FC<{
  receipt: Receipt | null;
  onClose: () => void;
  onSaved: (updated: Receipt) => Promise<void>;
}> = ({ receipt, onClose, onSaved }) => {
  const [kind, setKind] = useState<ReceiptKind>('invoice');
  const [documentDate, setDocumentDate] = useState('');
  const [issuerName, setIssuerName] = useState('');
  const [subject, setSubject] = useState('');
  const [gross, setGross] = useState('');
  const [tax, setTax] = useState('');
  const [currency, setCurrency] = useState('EUR');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!receipt) return;
    setKind(kindOf(receipt));
    setDocumentDate(receipt.documentDate ?? '');
    setIssuerName(receipt.issuerName ?? '');
    setSubject(receipt.subject ?? '');
    setGross(receipt.grossAmount ? formatCentsPlain(receipt.grossAmount) : '');
    setTax(receipt.taxAmount ? formatCentsPlain(receipt.taxAmount) : '');
    setCurrency(receipt.currency || 'EUR');
    setError(null);
  }, [receipt]);

  // Ein Betrag, den parseCents nicht lesen kann, darf nicht als 0 durchgehen:
  // 0 € ist an einem Eigenbeleg eine Aussage und keine leere Eingabe.
  const unreadable = (value: string) => value.trim() !== '' && parseCents(value) === null;
  const amountsUnreadable = unreadable(gross) || unreadable(tax);

  // Was die jeweilige Belegart zum Buchen braucht. Der Satz steht am Feld und
  // nicht erst in der Fehlermeldung des Buchungsformulars.
  const needsIssuer = kind === 'invoice';
  const needsSubject = kind === 'self_issued' || kind === 'letter' || kind === 'other';
  const needsAmount = kind === 'invoice' || kind === 'self_issued';
  // Kontoauszug und Handelsbrief werden nicht gebucht (requiresBooking). „Zum
  // Buchen nötig" verspräche ihnen einen Schritt, den es für sie nicht gibt;
  // Pflicht bleiben die Angaben trotzdem: der Betreff ist bei einem
  // Handelsbrief das Einzige, woran sich später sagen lässt, was das Schreiben
  // belegt, und das Belegdatum bestimmt bei jeder Belegart das Jahr, mit dem
  // die Aufbewahrungsfrist beginnt.
  const bookable = requiresBooking(kind);
  const subjectHint = bookable ? 'zum Buchen nötig' : 'Pflichtangabe';
  const dateHint = bookable ? 'zum Buchen nötig' : 'Pflichtangabe';

  async function save() {
    if (!receipt) return;
    if (amountsUnreadable) {
      setError('Der Betrag ist nicht lesbar. Erwartet wird eine Zahl wie 1234,56.');
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const header: ReceiptHeader = {
        kind,
        documentDate,
        issuerName: issuerName.trim(),
        grossAmount: parseCents(gross) ?? 0,
        taxAmount: parseCents(tax) ?? 0,
        currency: currency.trim() || 'EUR',
        subject: subject.trim(),
        // Klasse und Fristende rechnet das Backend aus Belegart und
        // Belegdatum. Sie hier zu füllen hieße, die Fristentabelle des
        // § 257 HGB ein zweites Mal zu führen.
        retentionClass: '',
        retentionUntil: '',
      };
      await onSaved(await Api.saveReceiptHeader(receipt.id, header));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  // Eine begonnene Eingabe geht beim Schließen nicht ohne Rückfrage verloren
  // (§8.7). Verglichen wird mit dem Stand, den der Beleg beim Öffnen trug.
  const dirty =
    receipt !== null &&
    (kind !== kindOf(receipt) ||
      documentDate !== (receipt.documentDate ?? '') ||
      issuerName !== (receipt.issuerName ?? '') ||
      subject !== (receipt.subject ?? '') ||
      gross !== (receipt.grossAmount ? formatCentsPlain(receipt.grossAmount) : '') ||
      tax !== (receipt.taxAmount ? formatCentsPlain(receipt.taxAmount) : '') ||
      currency !== (receipt.currency || 'EUR'));

  return (
    <Dialog
      open={receipt !== null}
      onOpenChange={(open) => !open && onClose()}
      dirty={dirty}
      title={`Kopfdaten zu Beleg ${receipt?.receiptNumber ?? ''}`}
      width="max-w-2xl"
      footer={
        <>
          {/* „Später" und nicht „Abbrechen": der Beleg ist bereits abgelegt,
              hier wird nichts zurückgenommen. */}
          <Button variant="secondary" onClick={onClose}>
            Später erfassen
          </Button>
          <Button variant="primary" loading={busy} onClick={() => void save()}>
            Kopfdaten speichern
          </Button>
        </>
      }
    >
      {/* Der Fehler des Backends steht über den Feldern und nicht als Toast:
          er nennt eine Eingabe, die zu ändern ist (§11.4). */}
      {error && <Notice tone="negative" text={error} className="mb-4" />}

      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Field
          label="Belegart"
          explain="Aus ihr folgt die Aufbewahrungsfrist: Rechnungen und Buchungsbelege acht Jahre, Handelsbriefe sechs, Bücher und Abschlüsse zehn (§ 257 Abs. 4 HGB, § 147 Abs. 3 AO)."
        >
          <Select<ReceiptKind>
            items={KIND_ITEMS}
            value={kind}
            onValueChange={(next) => setKind(next)}
            aria-label="Belegart"
          />
        </Field>
        <Field
          label="Belegdatum"
          hint={dateHint}
          explain="Das Datum auf dem Beleg, nicht der Tag des Eingangs. An ihm hängen die zeitgerechte Erfassung (§ 146 Abs. 1 AO) und der Beginn der Aufbewahrungsfrist."
        >
          <Input
            type="date"
            value={documentDate}
            onChange={(e) => setDocumentDate(e.target.value)}
          />
        </Field>
        <Field
          label="Aussteller"
          optional={!needsIssuer}
          hint={needsIssuer ? 'zum Buchen nötig' : undefined}
          explain="Der vollständige Name des leistenden Unternehmers ist Pflichtangabe einer Rechnung (§ 14 Abs. 4 Nr. 1 UStG)."
        >
          <Input value={issuerName} onChange={(e) => setIssuerName(e.target.value)} />
        </Field>
        <Field
          label="Betreff"
          optional={!needsSubject}
          hint={needsSubject ? subjectHint : undefined}
          explain="Bei Eigenbelegen, Handelsbriefen und sonstigen Dokumenten die einzige Bezeichnung dessen, was der Beleg belegt."
        >
          <Input value={subject} onChange={(e) => setSubject(e.target.value)} />
        </Field>
        <Field
          label="Bruttobetrag"
          optional={!needsAmount}
          hint={needsAmount ? 'zum Buchen nötig' : undefined}
          error={unreadable(gross) ? 'Nicht lesbar — erwartet wird 1234,56' : undefined}
        >
          <Input
            className="num text-right"
            inputMode="decimal"
            value={gross}
            onChange={(e) => setGross(e.target.value)}
            placeholder="0,00"
          />
        </Field>
        <Field
          label="Steuerbetrag"
          optional
          error={unreadable(tax) ? 'Nicht lesbar — erwartet wird 1234,56' : undefined}
        >
          <Input
            className="num text-right"
            inputMode="decimal"
            value={tax}
            onChange={(e) => setTax(e.target.value)}
            placeholder="0,00"
          />
        </Field>
        <Field label="Währung" optional>
          <Input
            className="code-num"
            value={currency}
            onChange={(e) => setCurrency(e.target.value.toUpperCase())}
            placeholder="EUR"
          />
        </Field>
      </div>
    </Dialog>
  );
};

// -------------------------------------------------------------------------

const ReceiptList: React.FC<{
  receipts: Receipt[];
  loading: boolean;
  /** Ob ein Filter greift — der leere Zustand liest sich dann anders (§13). */
  filtered: boolean;
  onResetFilter: () => void;
  selectedId?: number;
  onSelect: (r: Receipt) => void;
}> = ({ receipts, loading, filtered, onResetFilter, selectedId, onSelect }) => {
  if (loading) return <SkeletonRows rows={6} />;
  if (receipts.length === 0) {
    return filtered ? (
      <EmptyState
        variant="gefiltert"
        title="Kein Beleg passt zu diesem Filter"
        action={
          <Button variant="secondary" onClick={onResetFilter}>
            Filter zurücksetzen
          </Button>
        }
      />
    ) : (
      <EmptyState title="Noch keine Belege abgelegt" />
    );
  }

  return (
    <nav className="flex flex-col gap-0.5" aria-label="Belege">
      {receipts.map((receipt) => {
        const active = selectedId === receipt.id;
        const original = receipt.files.find((f) => f.role === 'original');
        return (
          <button
            key={receipt.id}
            type="button"
            aria-current={active || undefined}
            onClick={() => onSelect(receipt)}
            className={cn(
              'flex items-start gap-2.5 rounded-card px-3 py-2.5 text-left',
              'transition-colors duration-120 ease-quiet',
              active ? 'bg-accent-soft' : 'hover:bg-sunken',
            )}
          >
            {/* Einseitige Markierung als Pille, nicht als gekrümmte Border (§12). */}
            <span
              className={cn(
                'mt-0.5 h-8 w-0.5 shrink-0 rounded-full',
                active ? 'bg-accent' : 'bg-transparent',
              )}
            />
            <span className="min-w-0 flex-1">
              <span className="flex items-center justify-between gap-2">
                <span className="code-num text-caption text-ink">{receipt.receiptNumber}</span>
                <StatusBadge status={STATUS[receipt.status]} />
              </span>
              {/* Aussteller und Betrag stehen vor dem Dateinamen: wer einen
                  Beleg sucht, sucht die Rechnung von jemandem über etwas und
                  nicht „scan_0042.pdf". Ohne Kopfdaten bleibt der Dateiname. */}
              <span className="block text-caption text-ink-muted truncate mt-1">
                {receipt.issuerName || receipt.subject || original?.fileName || '—'}
              </span>
              {/* Die Belegart nur, wenn sie nicht der Regelfall ist: an jeder
                  Rechnung „Rechnung" zu schreiben, sagt nichts (§3.4). */}
              {kindOf(receipt) !== 'invoice' && (
                <span className="block text-caption text-ink-subtle mt-0.5">
                  {KIND_LABELS[kindOf(receipt)]}
                </span>
              )}
              <span className="flex items-center gap-2 text-caption text-ink-subtle mt-0.5">
                {/* Das Belegdatum, solange es da ist: es ist die Ordnung, in
                    der ein Beleg gesucht wird. Sonst der Eingangstag. */}
                <span className="num">
                  {receipt.documentDate
                    ? formatDate(receipt.documentDate)
                    : receipt.receivedAt
                      ? formatDate(receipt.receivedAt)
                      : '—'}
                </span>
                {Boolean(receipt.grossAmount) && (
                  <span className="num">{formatCents(receipt.grossAmount ?? 0)}</span>
                )}
                {receipt.files.length > 1 && (
                  <span className="inline-flex items-center gap-1">
                    <Paperclip className="w-3 h-3" strokeWidth={1.5} />
                    {receipt.files.length}
                  </span>
                )}
              </span>
            </span>
          </button>
        );
      })}
    </nav>
  );
};

// -------------------------------------------------------------------------

const ReceiptDetail: React.FC<{
  receipt: Receipt;
  vendors: Contact[];
  /** Alle Kontakte — der Empfänger eines Geschenks ist selten ein Lieferant. */
  contacts: Contact[];
  groups: PostingGroup[];
  treatments: TaxTreatmentInfo[];
  paymentAccounts: Account[];
  onChanged: (updated: Receipt) => Promise<void>;
  onBooked: (entryNumber: string) => Promise<void>;
  /** Öffnet den Dialog der Kopfdaten für diesen Beleg. */
  onEditHeader: () => void;
  proposal: EInvoiceProposal | null;
  proposalError: string | null;
  onNavigate?: NavigateFn;
}> = ({
  receipt,
  vendors,
  contacts,
  groups,
  treatments,
  paymentAccounts,
  proposal,
  proposalError,
  onChanged,
  onBooked,
  onEditHeader,
  onNavigate,
}) => (
  <>
  <div className="grid grid-cols-1 xl:grid-cols-2 gap-8 items-start">
    <ReceiptViewer receipt={receipt} onChanged={onChanged} onEditHeader={onEditHeader} />

    {receipt.status === 'filed' && !requiresBooking(kindOf(receipt)) ? (
      // Kontoauszug und Handelsbrief werden aufbewahrt, nicht gebucht: die
      // Umsätze des Auszugs entstehen im Bankimport aus derselben Datei
      // (Entscheidung 9), und der Handelsbrief belegt eine Abrede und keinen
      // Geschäftsvorfall. Ein Buchungsformular hier führte im einen Fall zu
      // einer zweiten, doppelten Buchung und im anderen zu einer erfundenen.
      <div>
        <h2 className="flex items-center gap-1.5 text-heading text-ink">
          {KIND_LABELS[kindOf(receipt)]}
          <HelpPopover label={`Erklärung zu ${KIND_LABELS[kindOf(receipt)]}`}>
            {kindOf(receipt) === 'statement'
              ? 'Der Kontoauszug ist ein Beleg ohne eigene Buchung: seine Umsätze sind über den Bankimport aus derselben Datei eingelesen und werden dort zugeordnet. Eine Buchung an dieser Stelle wäre die zweite zum selben Vorgang.'
              : 'Ein Handelsbrief belegt eine Abrede und keinen Geschäftsvorfall; gebucht wird er deshalb nicht. Aufzubewahren ist er trotzdem sechs Jahre (§ 257 Abs. 4 HGB, § 147 Abs. 3 AO).'}
          </HelpPopover>
        </h2>
        <p className="text-body text-ink-muted mt-2">
          {kindOf(receipt) === 'statement'
            ? 'Dieser Beleg wird nicht gebucht; seine Umsätze werden im Bankimport zugeordnet.'
            : 'Dieser Beleg wird nicht gebucht, aber aufbewahrt.'}
        </p>
      </div>
    ) : receipt.status === 'filed' ? (
      <div>
        {/* Ohne Kopfdaten weist das Backend die Buchung zurück (BEL-02). Der
            Hinweis steht über dem Formular und nicht in der Fehlermeldung nach
            dem Ausfüllen: sonst ist die Arbeit schon getan. */}
        {!hasHeader(receipt) && (
          <Notice
            className="mb-5"
            text="Zu diesem Beleg fehlen die Kopfdaten. Ohne Belegdatum und die Angaben seiner Belegart lässt er sich nicht buchen."
            action={
              <Button variant="secondary" size="sm" onClick={onEditHeader}>
                Kopfdaten erfassen
              </Button>
            }
          />
        )}
        <ProposalRefusal message={proposalError} />
        <BookingForm
          key={proposal ? `proposal-${receipt.id}` : `blank-${receipt.id}`}
          receipt={receipt}
          vendors={vendors}
          contacts={contacts}
          groups={groups}
          treatments={treatments}
          paymentAccounts={paymentAccounts}
          proposal={proposal}
          onBooked={onBooked}
        />
      </div>
    ) : (
      <div>
        <h2 className="flex items-center gap-2 text-heading text-ink">
          <ShieldCheck className="w-4 h-4 shrink-0 text-positive" strokeWidth={1.5} />
          {receipt.status === 'sealed' ? 'Gebucht und versiegelt' : 'Verworfen'}
        </h2>
        {receipt.status === 'sealed' && (
          <p className="text-body text-ink-muted mt-2">
            Was später dazukommt — eine Mahnung, ein Zahlungsnachweis, eine korrigierte Rechnung —
            ist ein eigener Beleg auf denselben Geschäftsvorfall. Eine inhaltliche Korrektur läuft
            über den Storno der Buchung.
          </p>
        )}
        {receipt.inputTaxOverride && (
          // Der Grund gehört nach der Buchung an den Beleg: wer den Vorsteuerabzug
          // trotz eines Befundes genommen hat, muss das später belegen können.
          <div className="mt-4">
            <h3 className="text-label text-ink">Übersteuerter Vorsteuerabzug</h3>
            <p className="text-body text-ink-muted mt-1">{receipt.inputTaxOverride}</p>
            {receipt.inputTaxOverrideAt && (
              <p className="text-caption text-ink-muted mt-1">
                festgehalten am {formatDateTime(receipt.inputTaxOverrideAt)}
              </p>
            )}
          </div>
        )}
        {receipt.discardReason && (
          <p className="text-body text-ink-muted mt-2">Grund: {receipt.discardReason}</p>
        )}
      </div>
    )}
  </div>

  {/* Der Prüfpfad steht über die ganze Breite und nicht in einer der beiden
      Spalten: er ist die Kette über den Beleg hinaus — Buchung, Zahlung,
      Bankumsatz — und keine Angabe des Dokuments. Ein verworfener Beleg hat
      keine. */}
  {receipt.direction === 'incoming' && receipt.status !== 'discarded' && (
    <AuditTrailPanel receipt={receipt} onChanged={onChanged} />
  )}

  {/* Beanstandungen, Buchungen und Aufbewahrung stehen am Beleg und nicht in
      einer eigenen Ansicht: wer den Beleg vor sich hat, stellt diese
      drei Fragen — was fehlt daran, was wurde daraus gebucht, wie lange ist er
      zu halten (RECH-07 K2, GOB-02 K2, ARC-01 K2). */}
  {/* Beanstandungen nur am Eingangsbeleg: eine eigene Rechnung und ein
      Eigenbeleg haben keinen strukturierten Teil, den eine Prüfung gegen die
      Rechnungspflichten beanstanden könnte — „nicht geprüft" führte dort in
      die Irre (RECH-07 K2). */}
  {(receipt.direction === 'incoming' || receipt.validatedAt) && receipt.status !== 'discarded' && (
    <ReceiptFindingsPanel receipt={receipt} />
  )}
  {receipt.status !== 'discarded' && (
    <ReceiptEntriesPanel receipt={receipt} onNavigate={onNavigate} />
  )}
  {receipt.status !== 'discarded' && (
    <RetentionSection receipt={receipt} onChanged={onChanged} />
  )}
  </>
);

// -------------------------------------------------------------------------

/**
 * Die Beanstandungen eines Belegs nach Fehlerklassen (RECH-02 K5, RECH-07 K2).
 *
 * Getrennt nach Formatfehler, Geschäftsregelfehler und Inhaltsfehler, weil die
 * Klasse sagt, wer den Fehler beheben kann und was er kostet. Klassifiziert
 * wird im Backend: eine zweite Einteilung in der Maske wäre die, die bei der
 * nächsten Regeländerung stehen bleibt.
 */
const ReceiptFindingsPanel: React.FC<{ receipt: Receipt }> = ({ receipt }) => {
  const [findings, setFindings] = useState<ReceiptFindings | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    Api.getReceiptFindings(receipt.id)
      .then((result) => {
        if (cancelled) return;
        setFindings(result);
        setError(null);
      })
      .catch((e) => {
        if (cancelled) return;
        setFindings(null);
        setError(e instanceof Error ? e.message : String(e));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [receipt.id, receipt.validatedAt]);

  const groups = (findings?.groups ?? []).filter((group) => (group.findings ?? []).length > 0);

  return (
    <Section
      title="Beanstandungen"
      context={
        findings?.checked
          ? `${findings.total} Befunde · ${findings.blocking} blockierend`
          : 'Der strukturierte Teil wurde nicht geprüft'
      }
      explain={
        <>
          Die Klasse sagt, wer den Fehler beheben kann und was er kostet: ein Formatfehler liegt am
          System des Lieferanten, ein Geschäftsregelfehler an der Rechnung, ein Inhaltsfehler an
          einer Pflichtangabe des § 14 Abs. 4 UStG — und der kostet den Vorsteuerabzug (§ 15
          Abs. 1 Satz 1 Nr. 1 UStG).
        </>
      }
    >
      {error && <Notice tone="negative" text={error} className="mb-5" />}
      {loading ? (
        <SkeletonRows rows={3} />
      ) : groups.length === 0 ? (
        <EmptyState
          title="Keine Beanstandung"
          description="Die Prüfung hat an diesem Beleg nichts gefunden."
        />
      ) : (
        groups.map((group) => (
          <div key={group.class} className="mt-6 first:mt-0">
            <h3 className="text-label text-ink-muted mb-2">
              {`${group.label || VALIDATION_FINDING_CLASS_LABELS[group.class]} · ${group.findings.length}`}
            </h3>
            <Table>
              <Thead>
                <Tr>
                  <Th className="w-40">Regel</Th>
                  <Th>Befund</Th>
                  <Th className="w-48">Norm</Th>
                  <Th className="w-64">Folge für den Vorsteuerabzug</Th>
                </Tr>
              </Thead>
              <Tbody>
                {group.findings.map((finding, index) => (
                  <Tr key={`${finding.rule}-${index}`}>
                    <Td code>
                      <span className="flex items-center gap-1.5">
                        <span
                          className={cn(
                            'mark-diamond shrink-0',
                            finding.blocking ? 'bg-negative' : 'bg-attention',
                          )}
                          aria-hidden="true"
                        />
                        {finding.rule}
                      </span>
                    </Td>
                    <Td>
                      {finding.message}
                      {finding.where ? ` (${finding.where})` : ''}
                    </Td>
                    <Td className="text-ink-subtle">{finding.norm || '—'}</Td>
                    <Td className="text-ink-muted">{finding.inputTaxEffect || '—'}</Td>
                  </Tr>
                ))}
              </Tbody>
            </Table>
          </div>
        ))
      )}
    </Section>
  );
};

/**
 * Die Buchungen zu diesem Beleg (GOB-02 K2, BEL-01 K3).
 *
 * Der Weg zurück: das Journal zeigt seit jeher den Beleg, der Beleg zeigte die
 * Buchung nicht. Ein Prüfer, der von der Ablage ausgeht — und das ist der
 * übliche Weg —, kam damit nicht ins Journal.
 */
const ReceiptEntriesPanel: React.FC<{
  receipt: Receipt;
  onNavigate?: NavigateFn;
}> = ({ receipt, onNavigate }) => {
  const [entries, setEntries] = useState<JournalEntry[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    Api.getEntriesForReceipt(receipt.id)
      .then((list) => {
        if (cancelled) return;
        setEntries(list ?? []);
        setError(null);
      })
      .catch((e) => {
        if (cancelled) return;
        setEntries([]);
        setError(e instanceof Error ? e.message : String(e));
      });
    return () => {
      cancelled = true;
    };
  }, [receipt.id, receipt.journalEntryId]);

  return (
    <Section
      title="Buchungen zu diesem Beleg"
      context="Der Weg zurück ins Journal und ins Kontoblatt"
    >
      {error && <Notice tone="negative" text={error} className="mb-5" />}
      {entries.length === 0 ? (
        <EmptyState
          title="Noch nicht gebucht"
          description="Zu diesem Beleg steht keine Buchung im Journal."
        />
      ) : (
        <Table>
          <Thead>
            <Tr>
              <Th>Buchung</Th>
              <Th>Datum</Th>
              <Th>Buchungstext</Th>
              <Th>Konten</Th>
              <Th numeric>Betrag</Th>
              <Th className="w-28" aria-label="Ziel" />
            </Tr>
          </Thead>
          <Tbody>
            {entries.map((entry) => {
              const lines = entry.lines ?? [];
              const gross = lines
                .filter((line) => line.side === 'S')
                .reduce((sum, line) => sum + line.amount, 0);
              return (
                <Tr key={entry.id} variant={entry.kind === 'reversal' ? 'storno' : 'default'}>
                  <Td code>{entry.entryNumber}</Td>
                  <Td className="text-ink-subtle num">{formatDate(entry.bookingDate)}</Td>
                  <Td className="max-w-[20rem] truncate" title={entry.description}>
                    {entry.description}
                  </Td>
                  <Td>
                    {/* Nummer und Bezeichnung zusammen (§11.1): die Nummer allein
                        sagt nicht, welches Konto angesprochen wurde. */}
                    <span className="flex flex-wrap gap-x-3 gap-y-1">
                      {lines.map((line) => (
                        <span
                          key={`${entry.id}-${line.position}`}
                          className="flex items-baseline gap-1"
                        >
                          {onNavigate ? (
                            <Button
                              variant="quiet"
                              size="sm"
                              className="code-num"
                              title={`${formatSide(line.side)} ${line.account}`}
                              onClick={() => onNavigate('accounts', { account: line.account })}
                            >
                              {line.account}
                            </Button>
                          ) : (
                            <span className="code-num">{line.account}</span>
                          )}
                          {line.accountName && (
                            <span className="text-ink-muted truncate max-w-[10rem]">
                              {line.accountName}
                            </span>
                          )}
                        </span>
                      ))}
                    </span>
                  </Td>
                  <Td numeric>{formatCents(gross, entry.currency)}</Td>
                  <Td className="text-right pl-0">
                    {onNavigate && (
                      <Button
                        variant="quiet"
                        size="sm"
                        onClick={() => onNavigate('journal', { entryNumber: entry.entryNumber })}
                      >
                        Im Journal
                      </Button>
                    )}
                  </Td>
                </Tr>
              );
            })}
          </Tbody>
        </Table>
      )}
    </Section>
  );
};

/**
 * Die Aufbewahrung des Belegs und ihre Verlängerung (ARC-01 K2).
 *
 * Nur nach oben: die gesetzliche Frist ist die Untergrenze. Geprüft wird das im
 * Backend; hier steht, was gilt und wer es wann verlängert hat.
 */
const RetentionSection: React.FC<{
  receipt: Receipt;
  onChanged: (updated: Receipt) => Promise<void>;
}> = ({ receipt, onChanged }) => {
  const lock = useWriteLock();
  const [open, setOpen] = useState(false);

  return (
    <Section
      title="Aufbewahrung"
      context={`Frist bis ${formatDate(receipt.retentionUntil ?? '')}`}
      action={
        <Button
          variant="secondary"
          disabled={lock.locked}
          title={lock.hint}
          onClick={() => setOpen(true)}
        >
          Frist verlängern
        </Button>
      }
    >
      <StatRow>
        <Stat
          label="Klasse"
          value={RETENTION_CLASS_LABELS[receipt.retentionClass ?? '']}
          /* Der Termin kommt aus dem Backend, sobald er dort steht: es kennt
             Hold und verlängerte Frist, die lokale Hilfe rechnet nur nach. */
          context={`Löschung frühestens ${formatDate(
            receipt.earliestDeletion ?? earliestDeletion(receipt.retentionUntil),
          )}`}
        />
        <Stat
          label="Verlängert"
          value={
            receipt.retentionOverrideAt ? formatDateOfTimestamp(receipt.retentionOverrideAt) : '—'
          }
          context={receipt.retentionOverrideReason || 'Gesetzliche Frist'}
        />
      </StatRow>
      <RetentionDialog
        open={open}
        onOpenChange={setOpen}
        receipt={receipt}
        onChanged={(next) => {
          void onChanged(next);
        }}
      />
    </Section>
  );
};

// -------------------------------------------------------------------------

/**
 * Der Prüfpfad eines Eingangsbelegs und sein Leistungsnachweis (RECH-08).
 *
 * Die Kette Beleg → Buchung → Zahlung → Bankumsatz beantwortet die Frage einer
 * Prüfung in beide Richtungen (GoBD Rz. 36); der Vermerk daneben beantwortet
 * die andere: Wer hat bestätigt, dass die Leistung so bezogen wurde? Die Kette
 * kommt aus dem Backend und wird hier nicht zusammengesucht — dieselbe, die
 * auch in die Datei geht.
 */
const AuditTrailPanel: React.FC<{
  receipt: Receipt;
  onChanged: (updated: Receipt) => Promise<void>;
}> = ({ receipt, onChanged }) => {
  // Der Vermerk wird an den Beleg geschrieben und steht im Protokoll: im
  // Prüfermodus gesperrt, der Prüfpfad selbst bleibt lesbar und ausgebbar
  // (§10.4) — er ist die Antwort auf die Frage, die eine Prüfung stellt.
  const writeLock = usePostingLock();
  const [trail, setTrail] = useState<AuditTrail | null>(null);
  const [threshold, setThreshold] = useState<Cents>(0);
  const [proof, setProof] = useState(receipt.serviceProof ?? '');
  const [proofDate, setProofDate] = useState(
    receipt.serviceProofAt || new Date().toISOString().slice(0, 10),
  );
  const [saving, setSaving] = useState(false);
  const [exporting, setExporting] = useState<'pdf' | 'csv' | null>(null);
  const [failure, setFailure] = useState<string | null>(null);

  useEffect(() => {
    setProof(receipt.serviceProof ?? '');
    setProofDate(receipt.serviceProofAt || new Date().toISOString().slice(0, 10));
  }, [receipt.id, receipt.serviceProof, receipt.serviceProofAt]);

  useEffect(() => {
    let cancelled = false;
    Api.getAuditTrail(receipt.id)
      .then((result) => {
        if (!cancelled) setTrail(result);
      })
      .catch(() => {
        if (!cancelled) setTrail(null);
      });
    return () => {
      cancelled = true;
    };
  }, [receipt.id, receipt.journalEntryId, receipt.serviceProof]);

  useEffect(() => {
    // Die Grenze steht in den Einstellungen und wird hier nur gelesen: eine
    // Zahl in der Ansicht wäre eine zweite Wahrheit neben der Prüfregel.
    Api.getCompanySettings()
      .then((settings) => setThreshold(settings.invoiceCheckThreshold ?? 0))
      .catch(() => setThreshold(0));
  }, []);

  async function save() {
    setSaving(true);
    setFailure(null);
    try {
      const updated = await Api.saveServiceProof(receipt.id, proof, proofDate);
      await onChanged(updated);
      toast.success('Leistungsnachweis festgehalten.');
    } catch (e) {
      setFailure(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  }

  async function exportTrail(format: 'pdf' | 'csv') {
    setExporting(format);
    setFailure(null);
    try {
      const path = await Api.exportAuditTrail(receipt.id, format);
      // Ein abgebrochener Speichern-Dialog ist keine Fehlermeldung wert.
      if (path) toast.success(`Prüfpfad gespeichert: ${path}`);
    } catch (e) {
      setFailure(e instanceof Error ? e.message : String(e));
    } finally {
      setExporting(null);
    }
  }

  const steps = trail?.steps ?? [];
  const required = threshold > 0 && (receipt.grossAmount ?? 0) >= threshold;
  // Zuerst der Bestellbezug aus dem Prüfpfad: er ist derselbe, der in die Datei
  // und ins Prüferpaket geht. Der Beleg hat ihn ebenfalls und springt ein,
  // solange die Kette noch lädt.
  const orderReference = trail?.orderReference ?? receipt.orderReference ?? '';

  return (
    <Section
      title="Prüfpfad"
      context={trail?.note || 'Beleg, Buchung, Zahlung und Bankumsatz in einer Kette'}
      explain={
        <>
          Eine Buchführung muss sich in beide Richtungen verfolgen lassen: vom Beleg zur Buchung
          und zurück (GoBD Rz. 36). Die Kette steht hier zusammen und geht so auch in die Datei
          und ins Prüferpaket. Der Leistungsnachweis daneben hält fest, wer die sachliche
          Richtigkeit bestätigt hat — ohne ihn steht später nur die Rechnung da.
        </>
      }
      action={
        <div className="flex items-center gap-2">
          <Button
            variant="quiet"
            size="sm"
            loading={exporting === 'csv'}
            onClick={() => void exportTrail('csv')}
          >
            Als CSV
          </Button>
          <Button
            variant="secondary"
            size="sm"
            icon={<Download className="w-4 h-4" strokeWidth={1.5} />}
            loading={exporting === 'pdf'}
            onClick={() => void exportTrail('pdf')}
          >
            Prüfpfad ausgeben
          </Button>
        </div>
      }
    >
      {failure && <Notice tone="negative" text={failure} className="mb-5" />}

      {steps.length === 0 ? (
        <p className="text-body text-ink-muted">
          Zu diesem Beleg gibt es noch keine Buchung; die Kette beginnt mit ihr.
        </p>
      ) : (
        <ul className="flex flex-col">
          {steps.map((step, index) => (
            <li
              key={`${step.stage}-${step.reference}-${index}`}
              className="flex items-start gap-4 py-2 border-t border-line first:border-t-0"
            >
              <span className="w-28 shrink-0 text-caption text-ink-subtle num">
                {step.date ? formatDate(step.date) : '—'}
              </span>
              <span className="flex-1 min-w-0">
                <span className="block text-body text-ink">{step.title}</span>
                <span className="block text-caption text-ink-muted">{step.detail}</span>
              </span>
              <span className="w-40 shrink-0 text-caption code-num truncate">
                {step.reference}
              </span>
              <span className="w-32 shrink-0 text-right num">{formatCents(step.amount)}</span>
            </li>
          ))}
        </ul>
      )}

      {/* Der Bestellbezug gehört in die Kette und nicht an das Feld daneben:
          er ist eine Angabe der E-Rechnung und steht auch dann da, wenn kein
          Nachweis verlangt ist. Als Hinweis am Nachweisfeld verschwand er
          genau bei den großen Belegen, bei denen der Pflichthinweis vorgeht.
          Schreibgeschützt: geändert wird er mit dem Beleg, nicht hier. */}
      <div className="mt-6 max-w-md">
        <Field
          label="Bestellbezug"
          explain={
            <>
              Die Bestellnummer aus dem strukturierten Datensatz der Rechnung. Sie ist der Anfang
              der Kette: gegen sie wird die Rechnung sachlich geprüft, und der Vermerk darunter
              hält fest, wer das getan hat.
            </>
          }
        >
          <Input value={orderReference || 'kein Bestellbezug in der Rechnung'} readOnly disabled />
        </Field>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-[minmax(0,1fr)_12rem] gap-4 mt-6 items-start">
        <Field
          label="Leistungsnachweis"
          hint={required ? `Pflicht ab ${formatCents(threshold)}` : undefined}
          explain={
            <>
              Der Vermerk hält fest, wogegen die Rechnung sachlich geprüft wurde — etwa „geprüft
              gegen Bestellung 4711 vom 3. März". Der Vorsteuerabzug setzt eine tatsächlich
              bezogene Leistung voraus (§ 15 UStG); wer ohne Prüfung bucht, hat dafür keinen
              Nachweis. Nachtragen können Sie ihn auch noch, wenn der Beleg schon gebucht ist.
            </>
          }
        >
          <Textarea
            value={proof}
            onChange={(e) => setProof(e.target.value)}
            placeholder="geprüft gegen Bestellung 4711 vom 3. März"
          />
        </Field>
        <div className="flex flex-col gap-4">
          <Field label="Datum der Prüfung">
            <Input type="date" value={proofDate} onChange={(e) => setProofDate(e.target.value)} />
          </Field>
          <Button
            variant="secondary"
            loading={saving}
            disabled={writeLock.locked}
            title={writeLock.hint}
            onClick={() => void save()}
          >
            Nachweis festhalten
          </Button>
        </div>
      </div>
    </Section>
  );
};

// -------------------------------------------------------------------------

/**
 * Die Kopfdaten und die Aufbewahrungsfrist am Beleg (BEL-02, ARC-01).
 *
 * Eine Auskunftszeile und kein Formular: geändert werden die Werte im Dialog.
 * Die Frist steht dabei, weil sie an diesem Beleg hängt und nirgends sonst zu
 * finden wäre — ein Prüfer fragt sie am einzelnen Beleg, nicht in einer
 * Jahresübersicht.
 */
const ReceiptHeaderFacts: React.FC<{ receipt: Receipt }> = ({ receipt }) => {
  // Das Backend liefert das Löschdatum; die lokale Rechnung bleibt nur als Rückfall für Läufe älterer Fassungen.
  const deletable = receipt.earliestDeletion ?? earliestDeletion(receipt.retentionUntil);
  const facts: { label: string; value: string; code?: boolean }[] = [
    { label: 'Belegart', value: KIND_LABELS[kindOf(receipt)] },
    {
      label: 'Belegdatum',
      value: receipt.documentDate ? formatDate(receipt.documentDate) : 'nicht erfasst',
      code: true,
    },
    { label: 'Aussteller', value: receipt.issuerName || '—' },
    ...(receipt.subject ? [{ label: 'Betreff', value: receipt.subject }] : []),
    ...(receipt.grossAmount
      ? [
          {
            label: 'Betrag',
            value: `${formatCents(receipt.grossAmount)}${
              receipt.taxAmount ? ` · davon ${formatCents(receipt.taxAmount)} Steuer` : ''
            }`,
            code: true,
          },
        ]
      : []),
    {
      label: 'Aufbewahrung',
      value: receipt.retentionUntil
        ? `${RETENTION_CLASS_LABELS[receipt.retentionClass ?? '']} · bis ${formatDate(
            receipt.retentionUntil,
          )}`
        : 'ohne Belegdatum nicht bestimmt',
    },
    ...(deletable ? [{ label: 'Löschbar ab', value: formatDate(deletable), code: true }] : []),
  ];

  return (
    <dl className="mt-3 grid grid-cols-[8rem_minmax(0,1fr)] gap-x-4 gap-y-1">
      {facts.map((fact) => (
        <React.Fragment key={fact.label}>
          <dt className="text-caption text-ink-subtle">{fact.label}</dt>
          <dd className={cn('text-caption text-ink-muted min-w-0 break-words', fact.code && 'num')}>
            {fact.value}
          </dd>
        </React.Fragment>
      ))}
    </dl>
  );
};

// -------------------------------------------------------------------------

const ReceiptViewer: React.FC<{
  receipt: Receipt;
  onChanged: (updated: Receipt) => Promise<void>;
  onEditHeader: () => void;
}> = ({ receipt, onChanged, onEditHeader }) => {
  const writeLock = usePostingLock();
  const [preview, setPreview] = useState<{ dataUrl: string; mimeType: string; intact: boolean } | null>(
    null,
  );
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [discarding, setDiscarding] = useState(false);
  const [discardReason, setDiscardReason] = useState('');

  useEffect(() => {
    let cancelled = false;
    setPreview(null);
    setPreviewError(null);
    Api.getReceiptPreview(receipt.id)
      .then((p) => {
        if (!cancelled) setPreview(p);
      })
      .catch((e) => {
        if (!cancelled) setPreviewError(e instanceof Error ? e.message : String(e));
      });
    return () => {
      cancelled = true;
    };
  }, [receipt.id, receipt.receiptHash]);

  async function addFile(role: ReceiptFileRole) {
    setBusy(true);
    try {
      const paths = await Api.selectReceiptFiles(
        role === 'rendering' ? 'Darstellung auswählen' : 'Anhang auswählen',
      );
      if (!paths || paths.length === 0) return;
      let updated = receipt;
      for (const path of paths) {
        updated = await Api.addReceiptFile(receipt.id, { path, role });
      }
      await onChanged(updated);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  /**
   * Gibt eine Belegdatei unter ihrem Originalnamen nach draußen.
   *
   * Eine Kopie, kein Umzug: der Beleg bleibt im Archiv. Das Backend prüft dabei
   * die Prüfsumme und verweigert eine beschädigte Datei, statt sie
   * weiterzugeben.
   */
  async function saveFile(fileId: number) {
    setBusy(true);
    try {
      const path = await Api.saveReceiptFileAs(receipt.id, fileId);
      // Leerer Pfad heißt: der Speichern-Dialog wurde abgebrochen.
      if (path) toast.success(`Gespeichert: ${path}`);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function removeFile(fileId: number) {
    setBusy(true);
    try {
      await onChanged(await Api.removeReceiptFile(receipt.id, fileId));
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  /**
   * Zieht den strukturierten Rechnungsdatensatz aus dem Beleg.
   *
   * Ein eigener Schritt: der Beleg wird in der empfangenen Form abgelegt und
   * erst danach untersucht. Findet sich nichts, ist es eine sonstige Rechnung —
   * die Meldung sagt das.
   */
  async function extractStructured() {
    setBusy(true);
    try {
      await onChanged(await Api.extractStructuredPart(receipt.id));
      toast.success('Strukturierter Rechnungsdatensatz übernommen.');
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function discard() {
    const reason = discardReason.trim();
    if (!reason) return;
    setDiscarding(false);
    setBusy(true);
    try {
      await Api.discardReceipt(receipt.id, reason);
      await onChanged({ ...receipt, status: 'discarded', discardReason: reason });
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  const open = receipt.status === 'filed';

  return (
    <div>
      <div className="flex items-start justify-between gap-4">
        <div className="min-w-0">
          <h2 className="code-num text-heading text-ink">{receipt.receiptNumber}</h2>
          <p className="code-num text-caption text-ink-subtle mt-0.5">
            Beleg-Hash {receipt.receiptHash.slice(0, 12)}…
          </p>
        </div>
        {open && (
          <div className="flex shrink-0 items-center gap-1">
            <Button
              variant="quiet"
              size="sm"
              disabled={busy || writeLock.locked}
              title={writeLock.hint}
              onClick={onEditHeader}
            >
              Kopfdaten
            </Button>
            <Button
              variant="quiet"
              size="sm"
              disabled={busy || writeLock.locked}
              title={writeLock.hint}
              onClick={() => setDiscarding(true)}
            >
              Verwerfen
            </Button>
          </div>
        )}
      </div>

      <ReceiptHeaderFacts receipt={receipt} />

      {/* Der Beleg ist ein Fremdkörper in der Oberfläche und bekommt deshalb
          eine eigene Fläche (§6.2, Fall 3). */}
      <div className="mt-4 rounded-card border border-line bg-sunken min-h-[20rem] flex items-center justify-center p-4">
        {previewError ? (
          <div className="text-center max-w-sm">
            <AlertTriangle className="w-6 h-6 mx-auto text-attention" strokeWidth={1.5} />
            <p className="text-body text-ink-muted mt-3">{previewError}</p>
            {open && (
              <Button
                variant="secondary"
                className="mt-4"
                disabled={busy || writeLock.locked}
                title={writeLock.hint}
                onClick={() => void addFile('rendering')}
              >
                Darstellung hinzufügen
              </Button>
            )}
          </div>
        ) : !preview ? (
          <span className="text-body text-ink-subtle">Vorschau wird geladen …</span>
        ) : preview.mimeType === 'application/pdf' ? (
          <iframe
            title="Belegvorschau"
            src={preview.dataUrl}
            className="w-full h-[30rem] rounded-control bg-surface"
          />
        ) : (
          <img src={preview.dataUrl} alt="Beleg" className="max-h-[30rem] rounded-control" />
        )}
      </div>

      {preview && !preview.intact && (
        <p className={cn(NOTE, NOTE_TONE.negative, 'text-body text-negative-text mt-3')}>
          Die Datei im Datenspeicher passt nicht mehr zu ihrer Prüfsumme. Sie wurde nach dem Ablegen
          verändert.
        </p>
      )}

      <ValidationPanel receipt={receipt} />

      <ul className="mt-4 divide-y divide-line border-t border-line">
        {receipt.files.map((file) => (
          <li key={file.id} className="flex items-center gap-2 py-2 text-body">
            <FileText className="w-4 h-4 shrink-0 text-ink-faint" strokeWidth={1.5} />
            <span className="truncate flex-1 text-ink">{file.fileName}</span>
            <span className="shrink-0 text-caption text-ink-subtle">
              {ROLE_LABELS[file.role]}
              {file.derived && ' · erzeugt'}
            </span>
            {/* Herausgeben lässt sich jede Datei, auch die eines gebuchten oder
                verworfenen Belegs: gerade die gebuchten sind die, die ein Prüfer
                verlangt (Entscheidung 10). Entfernen dagegen bleibt am offenen
                Beleg — der Hash eines gebuchten Belegs erstreckt sich über die
                Dateien. */}
            <Button
              variant="quiet"
              size="sm"
              iconOnly
              disabled={busy}
              title="Datei speichern"
              aria-label={`${file.fileName} speichern`}
              onClick={() => void saveFile(file.id)}
            >
              <Download className="w-4 h-4" strokeWidth={1.5} />
            </Button>
            {open && receipt.files.length > 1 && file.role !== 'original' && (
              <Button
                variant="quiet"
                size="sm"
                iconOnly
                disabled={busy || writeLock.locked}
                title={writeLock.hint ?? 'Datei entfernen'}
                aria-label={`${file.fileName} entfernen`}
                onClick={() => void removeFile(file.id)}
              >
                <Trash2 className="w-4 h-4" strokeWidth={1.5} />
              </Button>
            )}
          </li>
        ))}
      </ul>

      {open && (
        <div className="mt-4 flex flex-wrap gap-2">
          {!receipt.files.some((f) => f.role === 'structured') && (
            <Button
              variant="secondary"
              size="sm"
              disabled={busy || writeLock.locked}
              title={writeLock.hint}
              onClick={() => void extractStructured()}
              icon={<FileCode className="w-3.5 h-3.5" strokeWidth={1.5} />}
            >
              E-Rechnung auslesen
            </Button>
          )}
          <Button
            variant="secondary"
            size="sm"
            disabled={busy || writeLock.locked}
            title={writeLock.hint}
            onClick={() => void addFile('attachment')}
          >
            Anhang hinzufügen
          </Button>
          {!receipt.files.some((f) => f.role === 'rendering') && (
            <Button
              variant="secondary"
              size="sm"
              disabled={busy || writeLock.locked}
              title={writeLock.hint}
              onClick={() => void addFile('rendering')}
            >
              Darstellung hinzufügen
            </Button>
          )}
        </div>
      )}

      <Dialog
        open={discarding}
        onOpenChange={(next) => {
          setDiscarding(next);
          if (!next) setDiscardReason('');
        }}
        title="Beleg verwerfen"
        width="max-w-lg"
        footer={
          <>
            <Button variant="secondary" onClick={() => setDiscarding(false)}>
              Abbrechen
            </Button>
            <Button variant="danger" disabled={!discardReason.trim()} onClick={() => void discard()}>
              Verwerfen
            </Button>
          </>
        }
      >
        <p className="text-body text-ink-muted">
          <span className="code-num text-ink">{receipt.receiptNumber}</span> bleibt aufbewahrt und
          nachvollziehbar, wird aber nicht gebucht.
        </p>
        <Field label="Grund" className="mt-4">
          <Input
            value={discardReason}
            onChange={(e) => setDiscardReason(e.target.value)}
            placeholder="Doppelt erhalten"
          />
        </Field>
      </Dialog>
    </div>
  );
};

// -------------------------------------------------------------------------

const POSITION_GRID = 'grid grid-cols-[minmax(0,1fr)_7rem_6rem_2rem] gap-2 items-start';

const BookingForm: React.FC<{
  receipt: Receipt;
  vendors: Contact[];
  contacts: Contact[];
  groups: PostingGroup[];
  treatments: TaxTreatmentInfo[];
  paymentAccounts: Account[];
  proposal: EInvoiceProposal | null;
  onBooked: (entryNumber: string) => Promise<void>;
}> = ({ receipt, vendors, contacts, groups, treatments, paymentAccounts, proposal, onBooked }) => {
  const writeLock = usePostingLock();
  const today = receipt.receivedAt || new Date().toISOString().split('T')[0];
  const p = proposal?.request;
  const [contactId, setContactId] = useState(p?.contactId || vendors[0]?.id || 0);
  const [documentDate, setDocumentDate] = useState(p?.documentDate || today);
  const [bookingDate, setBookingDate] = useState(p?.bookingDate || today);
  const [serviceFrom, setServiceFrom] = useState(p?.serviceDateFrom || today);
  const [serviceTo, setServiceTo] = useState(p?.serviceDateTo || today);
  // Ohne Vorschlag steht „Inland" — die gewöhnliche Eingangsrechnung. Kam ein
  // Vorschlag, ließ sich der Steuerfall daraus aber nicht ableiten, bleibt das
  // Feld leer: das Backend hat sich bewusst enthalten (gemischte Kategorien, ein
  // Code ohne deutsche Entsprechung), und „Inland" hier einzusetzen überschriebe
  // diese Enthaltung. Das Backend nimmt eine Buchung ohne Steuerfall nicht an.
  const [treatment, setTreatment] = useState<TaxTreatment | ''>(
    proposal ? p?.taxTreatment || '' : 'domestic',
  );
  const [positions, setPositions] = useState<DraftPosition[]>(
    // Beträge und Steuersätze kommen aus dem strukturierten Teil, die
    // Buchungsgruppe bleibt offen — sie steht in keiner Rechnung.
    p && p.positions.length > 0
      ? p.positions.map((pos) => ({
          ...emptyPosition(),
          net: (pos.net / 100).toFixed(2).replace('.', ','),
          taxRate: pos.taxRate,
          text: pos.text ?? '',
        }))
      : [emptyPosition(groups[0])],
  );
  const [settlement, setSettlement] = useState<Settlement>('open');
  const [paymentAccount, setPaymentAccount] = useState(paymentAccounts[0]?.number ?? '');
  // Eine Rechnung, die eine zurückgestellte Verpflichtung erfüllt, läuft gegen
  // die Rückstellung und nicht gegen den Aufwand: der Aufwand ist im Jahr der
  // Bildung entstanden. Ein zweites Mal Aufwand zu buchen ist die häufigste Art,
  // eine Rückstellung falsch abzuwickeln.
  const [provisions, setProvisions] = useState<Provision[]>([]);
  const [provisionId, setProvisionId] = useState('');
  // Die Anzahlung ist die zweite Ausnahme vom Aufwand: bezahlt ist etwas,
  // geliefert nichts. Wofür angezahlt wurde, entscheidet über den Bilanzposten
  // und lässt sich aus dem Betrag nicht ableiten.
  const [advanceTargets, setAdvanceTargets] = useState<AdvanceTargetOption[]>([]);
  const [advanceTarget, setAdvanceTarget] = useState<AdvanceTarget | ''>('');
  const [openAdvances, setOpenAdvances] = useState<VendorAdvance[]>([]);
  const [settledAdvanceIds, setSettledAdvanceIds] = useState<number[]>([]);
  const [description, setDescription] = useState(p?.description ?? '');
  const [entertainment, setEntertainment] = useState<EntertainmentDetail>({
    place: '',
    day: today,
    participants: '',
    occasion: '',
  });
  // Die Währung des Belegs. Leer heißt Euro; die Positionsbeträge sind dann die
  // Eurobeträge. Steht eine Fremdwährung, sind sie die der Fremdwährung — der
  // Anwender tippt ab, was auf der Rechnung steht, und rechnet nicht selbst um.
  const [currency, setCurrency] = useState(p?.currency && p.currency !== 'EUR' ? p.currency : '');
  const [foreignTotal, setForeignTotal] = useState('');
  // Der Grund, mit dem ein blockierender Befund der Rechnungsprüfung
  // übersteuert wird. Er steht am Beleg und im Protokoll; ohne ihn gibt es
  // keine Buchung mit Vorsteuer.
  const [overrideReason, setOverrideReason] = useState('');
  // Der Leistungsnachweis geht mit der Buchung mit (RECH-08) und nicht in einem
  // zweiten Vorgang darunter: ein abgebrochener Buchungsversuch hinterließe
  // sonst einen Vermerk ohne Buchung. Ob er Pflicht ist, sagt die Vorschau —
  // die Grenze steht in den Einstellungen und wird hier nicht nachgebaut.
  const [serviceProof, setServiceProof] = useState(receipt.serviceProof ?? '');
  const [serviceProofAt, setServiceProofAt] = useState(
    receipt.serviceProofAt || new Date().toISOString().slice(0, 10),
  );

  // Ein Betrag, den parseCents nicht lesen kann ("1.2.3", "250,--", auch das
  // deutsche "1.234"), fällt beim Aufbau der Anfrage unten aus den Positionen
  // heraus. Ohne diese Prüfung würde der Beleg ohne diese Position gebucht, und
  // die Vorschau sähe dabei vollständig aus. Das ist keine Fachlogik: gerechnet
  // wird nichts, es wird nur gesagt, dass hier etwas nicht lesbar ist.
  const unreadableAmount = (value: string) => value.trim() !== '' && parseCents(value) === null;
  const hasUnreadableAmount = positions.some((pos) => unreadableAmount(pos.net));

  // Ob die Aufzeichnung nötig ist, sagt der Katalog: die Gruppe hat das Konto
  // für den nicht abzugsfähigen Anteil. Das Backend besteht darauf.
  const needsEntertainment = positions.some(
    (pos) => groups.find((g) => g.key === pos.postingGroup)?.deductibleQuota === 'entertainment',
  );

  const groupOf = useCallback(
    (key: string) => groups.find((g) => g.key === key),
    [groups],
  );

  /**
   * Der Vorsteueranteil einer Position in Promille — oder null, wo das Feld
   * leer oder unlesbar ist.
   *
   * Leer heißt voll abziehbar und geht gar nicht erst mit; das Backend liest
   * eine fehlende Angabe als 1000. Ein unlesbarer Wert darf nicht als 0
   * durchgehen: 0 ‰ ist der vollständige Ausschluss des Abzugs und damit eine
   * Aussage, keine leere Eingabe.
   */
  const shareOf = (value: string): number | null => {
    const trimmed = value.trim();
    if (trimmed === '') return null;
    const parsed = Number.parseInt(trimmed, 10);
    return Number.isNaN(parsed) ? null : parsed;
  };
  const unreadableShare = (value: string) => value.trim() !== '' && shareOf(value) === null;
  /**
   * Der Anteil liegt zwischen 1 und 1000 ‰ — die Null gehört nicht dazu.
   *
   * Ein vollständig ausgeschlossener Vorsteuerabzug ist der Ausschluss des
   * § 15 Abs. 1a UStG. Eine Aufteilung nach § 15 Abs. 4 UStG ist das nicht: der
   * Ausschluss richtet sich nach der Buchungsgruppe (Gästehaus, Jagd, Yacht)
   * und nicht nach einem Schlüssel; eine Null hier ginge im Backend als „nicht
   * angegeben" durch und führte zum vollen Abzug — dem Gegenteil dessen, was
   * sie sagen sollte.
   */
  const shareOutOfRange = (value: string) => {
    const share = shareOf(value);
    return share !== null && (share < 1 || share > 1000);
  };
  /** Ein geteilter Abzug ohne seinen Maßstab: § 15 Abs. 4 Satz 2 UStG verlangt ihn. */
  const shareWithoutReason = (pos: DraftPosition) => {
    const share = shareOf(pos.inputTaxShare);
    return share !== null && share < 1000 && pos.inputTaxShareReason.trim() === '';
  };
  /** Ein Geschenk ohne Empfänger: § 4 Abs. 7 EStG lässt den Abzug dann nicht zu. */
  const giftWithoutRecipient = (pos: DraftPosition) =>
    needsRecipient(groupOf(pos.postingGroup)) &&
    pos.giftContactId === 0 &&
    pos.giftName.trim() === '';

  const hasShareProblem = positions.some(
    (pos) =>
      unreadableShare(pos.inputTaxShare) ||
      shareOutOfRange(pos.inputTaxShare) ||
      shareWithoutReason(pos),
  );
  const hasGiftProblem = positions.some(giftWithoutRecipient);
  const foreignTotalCents = parseCents(foreignTotal);
  const unreadableForeignTotal = foreignTotal.trim() !== '' && foreignTotalCents === null;

  const [preview, setPreview] = useState<PostingPreview | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Solange keine Umrechnung in der Vorschau steht, fehlt der Kurs: entweder
  // scheitert die Vorschau, oder sie kommt ohne Umrechnung zurück. Die frühere
  // Prüfung suchte das Wort „Kurs" im Fehlertext und hing damit am Wortlaut des
  // Backends — eine Umformulierung dort hätte die Vorschau zur Sackgasse
  // gemacht, und die soll das Formular vermeiden.
  const rateMissing =
    currency.trim() !== '' &&
    (previewError !== null || (preview !== null && !preview.conversion));

  // Ein von Hand erfasster Kurs ändert die Antwort des Backends, nicht die
  // Eingabe. Ohne diesen Zähler bliebe die Vorschau auf ihrem Fehler stehen,
  // bis der Anwender irgendein Feld anfasst.
  const [rateNonce, setRateNonce] = useState(0);

  // Ein offener Befund hält die Buchung an, solange kein Grund dasteht — genau
  // wie im Backend. Die Sperre steht hier, damit der Anwender nicht erst nach
  // dem Klick erfährt, dass ihm ein Feld fehlt.
  const findings = preview?.inputTaxFindings ?? [];
  const findingsOpen = findings.length > 0 && overrideReason.trim() === '';

  // Die Pflicht zum Leistungsnachweis kommt aus der Vorschau: dieselbe Regel,
  // die das Buchen anhält, macht hier das Feld zum Pflichtfeld.
  const proofRequired = (preview?.warnings ?? []).some(
    (warning) => warning.code === SERVICE_PROOF_WARNING,
  );
  const proofOpen = proofRequired && serviceProof.trim() === '';

  // Ein deaktivierter Knopf sagt, woran es liegt (Konzept 10.4). Die Reihenfolge
  // entspricht der Sperrbedingung am Knopf, damit Grund und Sperre nicht
  // auseinanderlaufen.
  const postBlockedHint = hasUnreadableAmount
    ? 'Ein Nettobetrag einer Position ist nicht lesbar.'
    : unreadableForeignTotal
      ? 'Die Endsumme in Fremdwährung ist nicht lesbar.'
      : hasShareProblem
        ? 'Der Vorsteueranteil einer Position ist unvollständig.'
        : hasGiftProblem
          ? 'Einem Geschenk fehlt der Empfänger.'
          : findingsOpen
            ? 'Ohne festgehaltenen Grund wird eine Rechnung mit fehlender Pflichtangabe nicht mit Vorsteuer gebucht.'
            : proofOpen
              ? 'Ab der eingestellten Grenze wird ohne Leistungsnachweis nicht gebucht.'
              : !preview?.balanced
              ? 'Solange die Vorschau nicht ausgeglichen ist, wird nicht gebucht.'
              : writeLock.hint;

  const request: ReceiptRequest = useMemo(
    () => ({
      contactId,
      receiptId: receipt.id,
      bookingDate,
      documentDate,
      serviceDateFrom: serviceFrom,
      serviceDateTo: serviceTo,
      description,
      taxTreatment: treatment as TaxTreatment,
      positions: positions
        .filter((pos) => pos.postingGroup && parseCents(pos.net) !== null)
        .map((pos) => {
          const share = shareOf(pos.inputTaxShare);
          const recipient = needsRecipient(groups.find((g) => g.key === pos.postingGroup));
          return {
            postingGroup: pos.postingGroup,
            net: parseCents(pos.net) ?? 0,
            taxRate: pos.taxRate,
            text: pos.text || undefined,
            // 1000 ist der Regelfall und geht nicht mit: das Backend liest eine
            // fehlende Angabe genauso.
            inputTaxShare: share !== null && share !== 1000 ? share : undefined,
            inputTaxShareReason:
              share !== null && share < 1000 ? pos.inputTaxShareReason.trim() || undefined : undefined,
            gift: recipient
              ? {
                  contactId: pos.giftContactId || undefined,
                  name: pos.giftContactId ? undefined : pos.giftName.trim() || undefined,
                  occasion: pos.giftOccasion.trim() || undefined,
                }
              : undefined,
          };
        }),
      settlement,
      paymentAccount: settlement === 'paid' ? paymentAccount : undefined,
      currency: currency.trim().toUpperCase() || 'EUR',
      foreignAmount: currency.trim() ? foreignTotalCents ?? undefined : undefined,
      entertainment: needsEntertainment ? entertainment : undefined,
      provisionId: provisionId ? Number.parseInt(provisionId, 10) : undefined,
      advanceTarget: advanceTarget || undefined,
      settledAdvanceIds: settledAdvanceIds.length > 0 ? settledAdvanceIds : undefined,
    }),
    [
      contactId,
      receipt.id,
      bookingDate,
      documentDate,
      serviceFrom,
      serviceTo,
      description,
      treatment,
      positions,
      groups,
      settlement,
      paymentAccount,
      currency,
      foreignTotalCents,
      needsEntertainment,
      entertainment,
      provisionId,
      advanceTarget,
      settledAdvanceIds,
    ],
  );

  // Die offenen Rückstellungen des Geschäftsjahres. Erledigte stehen nicht zur
  // Auswahl: gegen sie zu buchen brächte den Bestand unter null.
  useEffect(() => {
    Api.getProvisions(0)
      .then((rows) => setProvisions(rows.filter((provision) => !provision.settledOn)))
      .catch(() => setProvisions([]));
    Api.getAdvanceTargets()
      .then(setAdvanceTargets)
      .catch(() => setAdvanceTargets([]));
  }, []);

  // Die noch nicht verrechneten Anzahlungen dieses Lieferanten: seine
  // Schlussrechnung setzt sie ab. Ohne die Absetzung stünde die Anzahlung
  // weiter im Vermögen und die Vorsteuer würde ein zweites Mal gezogen.
  useEffect(() => {
    setSettledAdvanceIds([]);
    if (!contactId) {
      setOpenAdvances([]);
      return;
    }
    let cancelled = false;
    Api.getOpenVendorAdvances(contactId)
      .then((rows) => {
        if (!cancelled) setOpenAdvances(rows);
      })
      .catch(() => {
        if (!cancelled) setOpenAdvances([]);
      });
    return () => {
      cancelled = true;
    };
  }, [contactId]);

  // Der Buchungssatz kommt aus dem Backend. Das Frontend rechnet ihn nicht nach:
  // eine zweite Steuerrechnung hier wäre eine zweite Wahrheit.
  useEffect(() => {
    if (!contactId || !treatment || request.positions.length === 0) {
      setPreview(null);
      setPreviewError(null);
      return;
    }
    let cancelled = false;
    const timer = window.setTimeout(() => {
      Api.previewIncomingReceipt(request)
        .then((result) => {
          if (cancelled) return;
          setPreview(result);
          setPreviewError(null);
        })
        .catch((e) => {
          if (cancelled) return;
          setPreview(null);
          setPreviewError(e instanceof Error ? e.message : String(e));
        });
    }, 250);
    return () => {
      cancelled = true;
      window.clearTimeout(timer);
    };
  }, [request, contactId, treatment, rateNonce]);

  function updatePosition(index: number, patch: Partial<DraftPosition>) {
    setPositions((prev) => prev.map((pos, i) => (i === index ? { ...pos, ...patch } : pos)));
  }

  function selectGroup(index: number, key: string) {
    const group = groups.find((g) => g.key === key);
    updatePosition(index, { postingGroup: key, taxRate: group?.defaultRate ?? TAX_RATE_STANDARD });
    // Die Gruppe schlägt den Steuerfall vor — Miete ist steuerfrei, Gehälter sind
    // nicht steuerbar. Ein Satz von null allein sagt nicht, warum keine Steuer
    // anfällt, und das Backend besteht auf der Begründung.
    if (group?.defaultTreatment) setTreatment(group.defaultTreatment);
  }

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      // Der Grund geht erst mit der Buchung mit und nicht schon mit jeder
      // Vorschau: sonst holte jeder Tastendruck im Grundfeld einen neuen
      // Buchungssatz, der davon nicht anders aussieht.
      const entry = await Api.postIncomingReceipt({
        ...request,
        overrideReason: overrideReason.trim() || undefined,
        // Der Vermerk geht mit der Buchung mit: das Backend schreibt ihn über
        // den Belegdienst, bevor es prüft, ob er fehlt. Ein leeres Feld ändert
        // dort nichts und löscht keinen schon erfassten Vermerk.
        serviceProof: serviceProof.trim() || undefined,
        serviceProofAt: serviceProof.trim() ? serviceProofAt : undefined,
      });
      await onBooked(entry.entryNumber);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  const groupItems = useMemo(
    () => groups.map((g) => ({ value: g.key, label: g.label, meta: g.category })),
    [groups],
  );

  if (vendors.length === 0) {
    return (
      <EmptyState
        title="Noch kein Lieferant angelegt"
        description="Für einen Eingangsbeleg braucht es einen Lieferanten. Unter Kunden & Lieferanten bekommt er sein Personenkonto aus dem Kreditoren-Nummernkreis."
      />
    );
  }

  return (
    <form onSubmit={submit}>
      <h2 className="text-heading text-ink">Beleg buchen</h2>

      <div className="mt-5 flex flex-col gap-4">
        <Field label="Lieferant">
          <Select
            items={vendors.map((v) => ({ value: v.id, label: `${v.name} · ${v.ledgerAccount}` }))}
            value={contactId}
            onValueChange={setContactId}
          />
        </Field>

        <div className="grid grid-cols-2 gap-4">
          <Field label="Belegdatum" hint="Rechnungsdatum">
            <Input type="date" value={documentDate} onChange={(e) => setDocumentDate(e.target.value)} />
          </Field>
          <Field label="Buchungsdatum" hint="bestimmt die Periode">
            <Input type="date" value={bookingDate} onChange={(e) => setBookingDate(e.target.value)} />
          </Field>
          <Field
            label="Leistung von"
            hint="Zeitpunkt der Leistung"
            explain="Der Zeitpunkt der Lieferung oder sonstigen Leistung ist Pflichtangabe der Rechnung (§ 14 Abs. 4 Nr. 6 UStG); er entscheidet über den Zeitraum des Vorsteuerabzugs."
          >
            <Input type="date" value={serviceFrom} onChange={(e) => setServiceFrom(e.target.value)} />
          </Field>
          <Field label="Leistung bis">
            <Input type="date" value={serviceTo} onChange={(e) => setServiceTo(e.target.value)} />
          </Field>
        </div>

        <Field
          label="Steuerfall"
          hint={treatments.find((t) => t.treatment === treatment)?.hint}
          explain="Der Steuerfall entscheidet über Aufwandskonto und Steuerzeile. Ohne ihn nimmt Buchfink die Buchung nicht an."
        >
          <Select
            items={treatments.map((t) => ({ value: t.treatment, label: t.label }))}
            value={treatment || undefined}
            onValueChange={(next) => setTreatment(next as TaxTreatment)}
            placeholder="Steuerfall wählen"
          />
        </Field>

        {/* Fremdwährung. Leer heißt Euro — der Regelfall bekommt kein Feld, das
            er ausfüllen müsste. Steht eine Währung, sind die Positionsbeträge
            die der Rechnung, und Buchfink rechnet um; geraten wird kein Kurs. */}
        <div className="grid grid-cols-2 gap-4">
          <Field
            label="Währung"
            optional
            hint="ISO 4217, leer für Euro"
            explain="Bei einem Beleg in Fremdwährung werden die Positionsbeträge in der Fremdwährung erfasst. Buchfink holt den Referenzkurs der EZB zum Belegdatum und rechnet damit den Aufwand; die Bemessungsgrundlage der Umsatzsteuer folgt dem Durchschnittskurs des Monats (§ 16 Abs. 6 UStG), sofern einer vorliegt. Liegt kein Kurs vor, wird keiner geraten — er ist dann von Hand zu erfassen."
          >
            <Input
              className="code-num"
              maxLength={3}
              placeholder="EUR"
              value={currency}
              onChange={(e) => setCurrency(e.target.value.toUpperCase().replace(/[^A-Z]/g, ''))}
            />
          </Field>
          {currency.trim() !== '' && (
            <Field
              label={`Endsumme in ${currency}`}
              optional
              hint="Kontrollsumme zu den Positionen"
              error={
                unreadableForeignTotal
                  ? 'Der Betrag ist nicht lesbar. Erwartet wird etwa 1234,56.'
                  : undefined
              }
              explain="Die Endsumme der Rechnung. Stimmt sie nicht mit den Positionen überein, weist das Backend die Buchung zurück — ein Tippfehler fällt hier auf und nicht in der Bilanz."
            >
              <Input
                align="right"
                inputMode="decimal"
                placeholder="0,00"
                value={foreignTotal}
                onChange={(e) => setForeignTotal(e.target.value)}
              />
            </Field>
          )}
        </div>
      </div>

      <div className="mt-6 pt-6 border-t border-line">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-label text-ink-muted">Positionen</h3>
          <Button
            variant="quiet"
            size="sm"
            icon={<Plus className="w-3.5 h-3.5" strokeWidth={1.5} />}
            onClick={() => setPositions((prev) => [...prev, emptyPosition(groups[0])])}
          >
            Position hinzufügen
          </Button>
        </div>

        <div className={cn(POSITION_GRID, 'text-caption text-ink-subtle mb-1')}>
          <span>Buchungsgruppe</span>
          <span className="text-right">Netto{currency.trim() ? ` in ${currency}` : ''}</span>
          <span>USt</span>
          <span />
        </div>

        <div className="flex flex-col gap-2">
          {positions.map((position, index) => (
            <div key={index} className="flex flex-col gap-2">
              <div className={POSITION_GRID}>
                <Combobox
                  items={groupItems}
                  value={position.postingGroup || null}
                  onValueChange={(key) => selectGroup(index, key ?? '')}
                  placeholder="Gruppe suchen"
                  emptyText="Keine passende Buchungsgruppe."
                />
                <Input
                  align="right"
                  inputMode="decimal"
                  placeholder="0,00"
                  value={position.net}
                  onChange={(e) => updatePosition(index, { net: e.target.value })}
                  aria-label={`Nettobetrag der Position ${index + 1}`}
                  title={
                    unreadableAmount(position.net)
                      ? 'Der Betrag ist nicht lesbar. Erwartet wird etwa 1234,56 — ohne Tausenderpunkt.'
                      : undefined
                  }
                  className={cn(
                    unreadableAmount(position.net) && 'border-negative ring-2 ring-negative/20',
                  )}
                />
                <Select
                  items={[
                    { value: TAX_RATE_STANDARD, label: '19 %' },
                    { value: TAX_RATE_REDUCED, label: '7 %' },
                    { value: TAX_RATE_NONE, label: 'ohne' },
                  ]}
                  value={position.taxRate}
                  onValueChange={(taxRate) => updatePosition(index, { taxRate })}
                />
                <Button
                  variant="quiet"
                  size="sm"
                  iconOnly
                  disabled={positions.length === 1}
                  title="Position entfernen"
                  aria-label={`Position ${index + 1} entfernen`}
                  onClick={() => setPositions((prev) => prev.filter((_, i) => i !== index))}
                >
                  <Trash2 className="w-4 h-4" strokeWidth={1.5} />
                </Button>
              </div>

              {/* Der Empfänger gehört zur Position und nicht zum Beleg: eine
                  Rechnung kann Geschenke an zwei Empfänger enthalten, und die
                  Freigrenze läuft je Empfänger. */}
              {needsRecipient(groupOf(position.postingGroup)) && (
                <div className={cn(NOTE, NOTE_TONE.attention)}>
                  <h4 className="text-label text-attention-text">
                    Empfänger des Geschenks
                    <HelpPopover label="Erklärung zur Aufzeichnung des Empfängers">
                      § 4 Abs. 7 EStG lässt den Abzug nur zu, wenn die Aufwendung einzeln und
                      getrennt aufgezeichnet ist. Ohne den Empfänger ließe sich außerdem die
                      Freigrenze des § 4 Abs. 5 Satz 1 Nr. 1 EStG je Empfänger und Wirtschaftsjahr
                      nicht führen — und mit ihrer Überschreitung entfällt der Abzug für sämtliche
                      Geschenke an diesen Empfänger, nach § 15 Abs. 1a UStG auch der Vorsteuerabzug.
                    </HelpPopover>
                  </h4>
                  <div className="mt-3 grid grid-cols-2 gap-3">
                    <Field label="Aus der Kartei" optional>
                      <Select
                        items={[
                          { value: 0, label: 'Nicht erfasst — als Freitext' },
                          ...contacts.map((c) => ({
                            value: c.id,
                            label: `${c.name} · ${c.ledgerAccount}`,
                          })),
                        ]}
                        value={position.giftContactId}
                        onValueChange={(next) =>
                          updatePosition(index, { giftContactId: Number(next) })
                        }
                      />
                    </Field>
                    <Field
                      label="Name des Empfängers"
                      hint={position.giftContactId ? 'kommt aus der Kartei' : 'Pflicht'}
                      disabled={position.giftContactId !== 0}
                    >
                      <Input
                        disabled={position.giftContactId !== 0}
                        value={position.giftName}
                        onChange={(e) => updatePosition(index, { giftName: e.target.value })}
                      />
                    </Field>
                  </div>
                  <Field label="Anlass" className="mt-3" optional hint="Jubiläum, Weihnachten, Messe">
                    <Input
                      value={position.giftOccasion}
                      onChange={(e) => updatePosition(index, { giftOccasion: e.target.value })}
                    />
                  </Field>
                </div>
              )}

              {/* Der Vorsteuerschlüssel steht bei der Position, weil er zu ihr
                  gehört: ein Beleg kann eine voll und eine anteilig abziehbare
                  Leistung enthalten. Er erscheint nur, wo es überhaupt Vorsteuer zu
                  teilen gibt — beim steuerpflichtigen Inlandsumsatz mit
                  Steuersatz und außerhalb der Gruppen, denen § 15 Abs. 1a UStG
                  den Abzug ohnehin nimmt. */}
              {treatment === 'domestic' &&
                position.taxRate !== TAX_RATE_NONE &&
                !groupOf(position.postingGroup)?.inputTaxExcluded && (
                <div className="grid grid-cols-[9rem_minmax(0,1fr)] gap-3">
                  <Field
                    label="Vorsteuer ‰"
                    optional
                    hint={
                      shareOf(position.inputTaxShare) !== null
                        ? formatPermille(shareOf(position.inputTaxShare) as number)
                        : 'leer = voll'
                    }
                    error={
                      unreadableShare(position.inputTaxShare)
                        ? 'Erwartet wird eine ganze Zahl von 1 bis 1000.'
                        : shareOutOfRange(position.inputTaxShare)
                          ? 'Der Anteil liegt zwischen 1 und 1000 ‰. Wo gar kein Abzug zusteht, ist die Buchungsgruppe die richtige Stelle (§ 15 Abs. 1a UStG).'
                          : undefined
                    }
                    explain="Der abziehbare Anteil der Vorsteuer in Promille bei gemischter Nutzung (§ 15 Abs. 4 UStG). 600 heißt: 60 % der Vorsteuer sind abziehbar, der Rest wird dem Aufwand zugeschlagen (§ 9b Abs. 1 EStG). Leer heißt voll abziehbar — der Regelfall."
                  >
                    <Input
                      align="right"
                      inputMode="numeric"
                      placeholder="1000"
                      value={position.inputTaxShare}
                      onChange={(e) => updatePosition(index, { inputTaxShare: e.target.value })}
                    />
                  </Field>
                  {shareOf(position.inputTaxShare) !== null &&
                    (shareOf(position.inputTaxShare) as number) < 1000 && (
                      <Field
                        label="Maßstab der Aufteilung"
                        hint="Pflicht unter 1000 ‰"
                        error={
                          shareWithoutReason(position)
                            ? 'Ohne den Maßstab nimmt das Backend die Buchung nicht an.'
                            : undefined
                        }
                        explain="§ 15 Abs. 4 Satz 2 UStG lässt die Aufteilung nach einer sachgerechten Schätzung zu. Festzuhalten ist, worauf sie beruht — etwa „Kfz zu 60 % betrieblich genutzt, Fahrtenbuch 2026&quot;."
                      >
                        <Input
                          value={position.inputTaxShareReason}
                          onChange={(e) =>
                            updatePosition(index, { inputTaxShareReason: e.target.value })
                          }
                        />
                      </Field>
                    )}
                </div>
              )}
            </div>
          ))}
        </div>
      </div>

      <div className="mt-6 pt-6 border-t border-line flex flex-col gap-4">
        <div className="grid grid-cols-2 gap-4">
          <Field label="Zahlung">
            <Select
              items={[
                { value: 'open', label: 'Auf Ziel — offener Posten' },
                { value: 'paid', label: 'Sofort bezahlt' },
              ]}
              value={settlement}
              onValueChange={(next) => setSettlement(next as Settlement)}
            />
          </Field>
          {settlement === 'paid' && (
            <Field label="Zahlungsmittel" hint="Bar heißt Kasse, nicht Bank">
              <Select
                items={paymentAccounts.map((a) => ({ value: a.number, label: `${a.number} ${a.name}` }))}
                value={paymentAccount}
                onValueChange={setPaymentAccount}
              />
            </Field>
          )}
          <Field
            label="Anzahlung"
            optional
            explain="Eine geleistete Anzahlung ist kein Aufwand: Sie steht als eigener Posten im Vermögen (§ 266 Abs. 2 HGB) und wird erst mit der Schlussrechnung des Lieferanten umgebucht. Der Vorsteuerabzug setzt neben der Rechnung die Zahlung voraus (§ 15 Abs. 1 Satz 1 Nr. 1 Satz 3 UStG) — der Beleg wird deshalb als bezahlt erfasst."
          >
            <Select
              items={[
                { value: '', label: 'Keine Anzahlung' },
                ...advanceTargets.map((target) => ({
                  value: target.key,
                  label: `${target.label} · ${target.account}`,
                })),
              ]}
              value={advanceTarget}
              onValueChange={(next) => {
                setAdvanceTarget(next as AdvanceTarget | '');
                // Der Beleg ist entweder eine geleistete Anzahlung oder die
                // Schlussrechnung, die eine absetzt — beides zusammen weist
                // das Backend zurück. Die angehakten Anzahlungen verschwinden
                // hier ohnehin aus der Ansicht; blieben sie im Zustand, käme
                // ein Fehler zu Kästchen, die der Anwender nicht mehr sieht.
                if (next) setSettledAdvanceIds([]);
              }}
              placeholder="Keine Anzahlung"
            />
          </Field>
          {provisions.length > 0 && (
            <Field
              label="Gehört zu Rückstellung"
              optional
              explain="Gebucht wird gegen die Rückstellung; nur der Mehrbetrag bleibt Aufwand."
            >
              <Select
                items={[
                  { value: '', label: 'Keine Rückstellung' },
                  ...provisions.map((provision) => ({
                    value: String(provision.id),
                    label: `${provision.text} · ${formatCents(provision.settlementAmount)}`,
                  })),
                ]}
                value={provisionId}
                onValueChange={(next) => setProvisionId(next as string)}
                placeholder="Keine Rückstellung"
              />
            </Field>
          )}
        </div>

        {openAdvances.length > 0 && !advanceTarget && (
          <div>
            <h4 className="text-label text-ink-muted">
              Gehört zu Anzahlung
              <HelpPopover label="Erklärung zur Absetzung der Anzahlung">
                Die Schlussrechnung des Lieferanten weist den Gesamtbetrag aus, die Vorsteuer auf
                den angezahlten Teil ist aber schon gezogen. Die abgesetzte Anzahlung wird deshalb
                vom Konto der geleisteten Anzahlungen aufgelöst; ohne die Angabe stünde sie weiter
                im Vermögen.
              </HelpPopover>
            </h4>
            <div className="mt-3 flex flex-col gap-2">
              {openAdvances.map((advance) => (
                <Checkbox
                  key={advance.id}
                  checked={settledAdvanceIds.includes(advance.id)}
                  onCheckedChange={(checked) =>
                    setSettledAdvanceIds((prev) =>
                      checked ? [...prev, advance.id] : prev.filter((id) => id !== advance.id),
                    )
                  }
                  label={`${advance.documentNumber} · ${formatDate(advance.paidAt)} · ${formatCents(
                    advance.grossAmount,
                  )}`}
                />
              ))}
            </div>
          </div>
        )}

        {needsEntertainment && (
          <div className={cn(NOTE, NOTE_TONE.attention)}>
            <h4 className="text-label text-attention-text">
              Aufzeichnung zur Bewirtung
              <HelpPopover label="Erklärung zur Bewirtungsaufzeichnung">
                § 4 Abs. 5 Satz 1 Nr. 2 EStG verlangt Ort, Tag, Teilnehmer und Anlass. Ohne diese
                Angaben ist der Abzug auch für die abziehbaren 70 % verloren.
              </HelpPopover>
            </h4>
            <div className="mt-3 flex flex-col gap-3">
              <div className="grid grid-cols-2 gap-3">
                <Field label="Ort">
                  <Input
                    value={entertainment.place}
                    onChange={(e) => setEntertainment({ ...entertainment, place: e.target.value })}
                  />
                </Field>
                <Field label="Tag">
                  <Input
                    type="date"
                    value={entertainment.day}
                    onChange={(e) => setEntertainment({ ...entertainment, day: e.target.value })}
                  />
                </Field>
              </div>
              <Field label="Teilnehmer" hint="alle bewirteten Personen, mit Firma">
                <Input
                  value={entertainment.participants}
                  onChange={(e) => setEntertainment({ ...entertainment, participants: e.target.value })}
                />
              </Field>
              <Field label="Anlass" hint="der konkrete geschäftliche Anlass">
                <Input
                  value={entertainment.occasion}
                  onChange={(e) => setEntertainment({ ...entertainment, occasion: e.target.value })}
                />
              </Field>
            </div>
          </div>
        )}

        <Field label="Buchungstext" hint="leer lassen für den Standardtext" optional>
          <Input value={description} onChange={(e) => setDescription(e.target.value)} />
        </Field>
      </div>

      <div className="mt-6 flex flex-col gap-3">
        {proposal && <ProposalNotes proposal={proposal} />}
        {hasUnreadableAmount && (
          <p className={cn(NOTE, NOTE_TONE.negative, 'text-body text-negative-text')}>
            Ein Nettobetrag ist nicht lesbar. Erwartet wird etwa <span className="code-num">1234,56</span>{' '}
            — ohne Tausenderpunkt. Solange er so dasteht, fiele die Position aus der Buchung.
          </p>
        )}
        {hasGiftProblem && (
          <p className={cn(NOTE, NOTE_TONE.negative, 'text-body text-negative-text')}>
            <span className="inline-flex items-center gap-1.5">
              Zu einem Geschenk gehört der Empfänger — aus der Kartei oder als Name.
              <HelpPopover label="Erklärung zum Empfänger eines Geschenks">
                Geschenke an Geschäftsfreunde sind nur abziehbar, wenn sie einzeln und getrennt
                aufgezeichnet sind; dazu gehört der Name des Empfängers (§ 4 Abs. 7 EStG). Ohne ihn
                ist der Aufwand nicht abziehbar, auch wenn die Grenze eingehalten ist.
              </HelpPopover>
            </span>
          </p>
        )}
        <ConversionPanel conversion={preview?.conversion} date={documentDate} />
        {rateMissing && (
          <ManualRateForm
            currency={currency.trim().toUpperCase()}
            date={documentDate}
            onSaved={() => setRateNonce((n) => n + 1)}
          />
        )}
        <PostingWarnings warnings={preview?.warnings} />
        {/* Der Leistungsnachweis steht als Pflichtfeld in der Maske und nicht
            als Fehlermeldung nach dem Klick: geprüft wird gegen die Bestellung,
            und das geschieht vor dem Buchen oder gar nicht (RECH-08). */}
        {proofRequired && (
          <div className="grid grid-cols-1 sm:grid-cols-[minmax(0,1fr)_12rem] gap-4">
            <Field
              label="Leistungsnachweis"
              hint="wogegen geprüft wurde"
              error={proofOpen ? 'Ohne diesen Vermerk wird der Beleg nicht gebucht.' : undefined}
              explain="Ab der in den Einstellungen hinterlegten Grenze hält Buchfink die Buchung an, bis der Vermerk dasteht. Wer ihn erst nach dem Buchen nachträgt, hat die Rechnung bezahlt, bevor jemand geprüft hat, ob die Leistung erbracht wurde."
            >
              <Input
                value={serviceProof}
                onChange={(e) => setServiceProof(e.target.value)}
                placeholder="geprüft gegen Bestellung 4711 vom 12.03.2026"
              />
            </Field>
            <Field label="Geprüft am">
              <Input
                type="date"
                value={serviceProofAt}
                onChange={(e) => setServiceProofAt(e.target.value)}
              />
            </Field>
          </div>
        )}
        <InputTaxFindings
          findings={preview?.inputTaxFindings}
          reason={overrideReason}
          onReasonChange={setOverrideReason}
        />
        <PostingPreviewPanel preview={preview} error={previewError} />
        {error && (
          <p className={cn(NOTE, NOTE_TONE.negative, 'text-body text-negative-text')}>{error}</p>
        )}
      </div>

      <div className="mt-6 flex justify-end">
        <Button
          type="submit"
          variant="primary"
          loading={busy}
          disabled={
            hasUnreadableAmount ||
            unreadableForeignTotal ||
            hasShareProblem ||
            hasGiftProblem ||
            findingsOpen ||
            proofOpen ||
            !preview?.balanced ||
            writeLock.locked
          }
          title={postBlockedHint}
        >
          Buchen
        </Button>
      </div>
    </form>
  );
};

// -------------------------------------------------------------------------

/**
 * Das Prüfergebnis am Beleg.
 *
 * Der Prüfumfang wird benannt, nicht behauptet: die Referenzumsetzung von
 * EN 16931 ist ein Schematron-Regelwerk, das kein Go-Prozessor ausführt. Was
 * Buchfink prüft, ist eine belegte Teilmenge — das steht hier, statt
 * Vollständigkeit vorzutäuschen.
 */
const ValidationPanel: React.FC<{ receipt: Receipt }> = ({ receipt }) => {
  if (!receipt.validatedAt) return null;

  let findings: ValidationFinding[] = [];
  let unreadable = false;
  try {
    const parsed = receipt.validationFindings ? JSON.parse(receipt.validationFindings) : [];
    findings = Array.isArray(parsed) ? parsed : [];
    unreadable = !Array.isArray(parsed);
  } catch {
    unreadable = true;
  }
  const errors = findings.filter((f) => f.severity === 'fatal');
  const rest = findings.filter((f) => f.severity !== 'fatal');
  // Grün nur, wenn beide Quellen es sagen. Die Zahl kommt aus dem Backend und
  // überlebt jede Formatänderung an der Befundliste; ohne sie hieße eine Liste,
  // die sich nicht lesen lässt, „keine Verstöße" — die Falschaussage, die von
  // allen die teuerste ist.
  const clean = errors.length === 0 && !receipt.validationErrors && !unreadable;
  const complete = receipt.validationCoverage === 'full';

  return (
    <div className={cn(NOTE, clean ? NOTE_TONE.positive : NOTE_TONE.negative, 'mt-3')}>
      <h3
        className={cn(
          'flex items-start gap-2 text-label',
          clean ? 'text-positive-text' : 'text-negative-text',
        )}
      >
        {clean ? (
          <ShieldCheck className="w-4 h-4 mt-px shrink-0" strokeWidth={1.5} />
        ) : (
          <AlertTriangle className="w-4 h-4 mt-px shrink-0" strokeWidth={1.5} />
        )}
        {clean
          ? complete
            ? 'Alle Regeln der Norm erfüllt'
            : 'Geprüfte Regeln erfüllt'
          : errors.length > 0
            ? `${errors.length} ${errors.length === 1 ? 'Verstoß' : 'Verstöße'} gegen die geprüften Regeln`
            : `${receipt.validationErrors} ${
                receipt.validationErrors === 1 ? 'Verstoß' : 'Verstöße'
              } gegen die geprüften Regeln — die Einzelheiten sind nicht lesbar`}
      </h3>
      <p className="text-caption text-ink-muted mt-1">
        {receipt.detectedProfile} · Regelwerk {receipt.validationRuleset}
        {complete
          ? ' · alle 223 Geschäftsregeln der Norm'
          : ' · Teilprüfung, die Extension-Regeln bleiben offen'}
      </p>
      {/* Die einzelnen Befunde stehen unten unter „Beanstandungen", nach
          Fehlerklassen getrennt und mit Norm und Folge für den Vorsteuerabzug.
          Zweimal dieselbe Liste hieße zweimal derselbe Befund — und die obere
          wüsste nichts von den Pflichtangaben, die erst dort dazukommen. */}
      {rest.length + errors.length > 0 && (
        <p className="text-caption text-ink-muted mt-1">
          Die einzelnen Befunde stehen unter „Beanstandungen".
        </p>
      )}
    </div>
  );
};

/**
 * Warum aus dem strukturierten Teil kein Buchungsvorschlag wurde.
 *
 * Der Beleg bleibt buchbar — von Hand. Was das Backend ablehnt, ist der
 * Vorschlag, nicht die Buchung: eine Gutschrift ist ein gültiges Dokument mit
 * einem anderen Geschäftsvorfall dahinter, und diese Entscheidung trifft der
 * Nutzer, nicht Buchfink.
 */
const ProposalRefusal: React.FC<{ message: string | null }> = ({ message }) => {
  if (!message) return null;
  return (
    <div className={cn(NOTE, NOTE_TONE.attention, 'mb-5')}>
      <h3 className="flex items-start gap-2 text-label text-attention-text">
        <AlertTriangle className="w-4 h-4 mt-px shrink-0" strokeWidth={1.5} />
        Kein Buchungsvorschlag aus dem Rechnungsdatensatz
      </h3>
      <p className="text-body text-ink-muted mt-1.5">{message}</p>
      <p className="text-caption text-ink-subtle mt-1">
        Der Beleg lässt sich weiterhin von Hand kontieren.
      </p>
    </div>
  );
};

/** Was aus dem strukturierten Teil kam und was offen blieb. */
const ProposalNotes: React.FC<{ proposal: EInvoiceProposal }> = ({ proposal }) => (
  <div className={cn(NOTE, NOTE_TONE.neutral)}>
    <h3 className="flex items-start gap-2 text-label text-ink">
      <FileCode className="w-4 h-4 mt-px shrink-0 text-ink-subtle" strokeWidth={1.5} />
      Aus der E-Rechnung übernommen
    </h3>
    <p className="text-caption text-ink-muted mt-1.5">
      {proposal.supplierName}
      {proposal.invoiceNumber && ` · Rechnung ${proposal.invoiceNumber}`}
      {proposal.kindLabel && ` · ${proposal.kindLabel}`}
      {proposal.profile && ` · Profil ${proposal.profile}`}
    </p>
    {proposal.precedingInvoices && proposal.precedingInvoices.length > 0 && (
      <p className="text-caption text-ink-muted mt-1">
        Bezug auf {proposal.precedingInvoices.join(', ')} — die Verrechnung mit der genannten
        Rechnung führt Buchfink noch nicht; sie ist von Hand zu prüfen.
      </p>
    )}
    {proposal.notes && proposal.notes.length > 0 && (
      <ul className="text-caption text-ink-muted list-disc pl-4 mt-1 space-y-0.5">
        {proposal.notes.map((note, i) => (
          <li key={i}>{note}</li>
        ))}
      </ul>
    )}
  </div>
);

/**
 * Hinweise zur Buchung.
 *
 * Sie blockieren nie. Was aus einer fehlenden E-Rechnung folgt, ist eine
 * Rechtsfrage — Buchfink zeigt sie an und bewertet sie nicht.
 */
const PostingWarnings: React.FC<{ warnings?: PostingWarning[] }> = ({ warnings }) => {
  const [copied, setCopied] = useState<string | null>(null);
  if (!warnings || warnings.length === 0) return null;

  return (
    <div className="flex flex-col gap-3">
      {warnings.map((warning) => {
        const loud = warning.severity === 'warning';
        return (
          <div key={warning.code} className={cn(NOTE, loud ? NOTE_TONE.attention : NOTE_TONE.neutral)}>
            <h3
              className={cn(
                'flex items-start gap-2 text-label',
                loud ? 'text-attention-text' : 'text-ink',
              )}
            >
              <AlertTriangle
                className={cn('w-4 h-4 mt-px shrink-0', !loud && 'text-ink-subtle')}
                strokeWidth={1.5}
              />
              {warning.title}
            </h3>
            <p className="text-body text-ink-muted mt-1.5">{warning.detail}</p>
            {warning.supplierNote && (
              <Button
                variant="quiet"
                size="sm"
                className="mt-2 -ml-2.5"
                onClick={() => {
                  void navigator.clipboard?.writeText(warning.supplierNote ?? '');
                  setCopied(warning.code);
                  window.setTimeout(() => setCopied(null), 2000);
                }}
              >
                {copied === warning.code ? 'Kopiert' : 'Hinweistext für den Lieferanten kopieren'}
              </Button>
            )}
          </div>
        );
      })}
    </div>
  );
};

/**
 * Die blockierenden Befunde der Rechnungsprüfung samt dem Feld für die
 * Übersteuerung.
 *
 * Sie stehen neben den Warnungen und nicht in ihnen: eine Warnung zeigt, ein
 * Befund hält an. § 15 Abs. 1 Satz 1 Nr. 1 UStG lässt den Abzug nur aus einer
 * Rechnung nach §§ 14, 14a UStG zu — fehlt eine Pflichtangabe, gibt es zwei
 * Wege, und beide stehen hier: die Angabe ergänzen oder mit einem Grund buchen,
 * der am Beleg und im Protokoll bleibt. Ohne dieses Feld wäre die Maske eine
 * Sackgasse, weil das Backend die Buchung anhält und nichts anbietet.
 */
const InputTaxFindings: React.FC<{
  findings?: InputTaxFinding[];
  reason: string;
  onReasonChange: (value: string) => void;
}> = ({ findings, reason, onReasonChange }) => {
  const rows = findings ?? [];
  if (rows.length === 0) return null;

  return (
    <div className={cn(NOTE, NOTE_TONE.negative)}>
      <h3 className="flex items-start gap-2 text-label text-negative-text">
        <AlertTriangle className="w-4 h-4 mt-px shrink-0" strokeWidth={1.5} />
        {rows.length === 1
          ? 'Eine Pflichtangabe der Rechnung fehlt'
          : `${rows.length} Pflichtangaben der Rechnung fehlen`}
      </h3>
      <ul className="mt-2 flex flex-col gap-2">
        {rows.map((finding) => (
          <li key={finding.code}>
            <p className="text-body text-ink">{finding.title}</p>
            <p className="text-caption text-ink-muted mt-0.5">
              {finding.detail}
              {finding.fixable &&
                ' Diese Angabe lässt sich in den Stammdaten des Ausstellers nachtragen.'}
            </p>
          </li>
        ))}
      </ul>
      <Field
        label="Grund der Übersteuerung"
        className="mt-3"
        hint={reason.trim() === '' ? 'Pflicht, solange ein Befund offen ist' : undefined}
        explain="Der Grund wird am Beleg und im Änderungsprotokoll festgehalten. Er ersetzt die fehlende Angabe nicht — er hält fest, worauf der Abzug trotzdem gestützt wird, etwa weil die berichtigte Rechnung vorliegt oder die Angabe aus anderen Unterlagen hervorgeht."
      >
        <Textarea
          rows={2}
          value={reason}
          onChange={(e) => onReasonChange(e.target.value)}
          placeholder="Worauf der Vorsteuerabzug trotz des Befundes gestützt wird"
        />
      </Field>
    </div>
  );
};

/**
 * Der Kurs von Hand — der Ausweg, wenn der Kursdienst nicht antwortet.
 *
 * Buchfink rät keinen Kurs: ohne Kurs gibt es keine Umrechnung, und ein
 * Rückfall auf 1,0 wäre eine erfundene Zahl im Buchungssatz. Damit die
 * Ablehnung keine Sackgasse ist, steht der zweite Weg an derselben Stelle wie
 * die Meldung. Die Quelle ist Pflicht: ein Kurs ohne Herkunft ist eine
 * Behauptung, und er entscheidet über den Aufwand.
 */
const ManualRateForm: React.FC<{
  currency: string;
  date: string;
  onSaved: () => void;
}> = ({ currency, date, onSaved }) => {
  const writeLock = usePostingLock();
  const [rate, setRate] = useState('');
  const [source, setSource] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const micros = parseExchangeRate(rate);
  const unreadable = rate.trim() !== '' && micros === null;

  async function save() {
    if (micros === null || source.trim() === '') {
      setError('Ein Kurs von Hand braucht seinen Wert und seine Quelle.');
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await Api.saveExchangeRate({
        currency,
        date,
        rateMicros: micros,
        source: source.trim(),
        manual: true,
      });
      setRate('');
      setSource('');
      onSaved();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className={cn(NOTE, NOTE_TONE.attention)}>
      <h3 className="text-label text-attention-text">
        Kurs für {currency} am {formatDate(date)} von Hand erfassen
        <HelpPopover label="Erklärung zum Kurs von Hand">
          Buchfink holt den Referenzkurs der Europäischen Zentralbank zum Belegdatum. Ist der
          Kursdienst nicht erreichbar oder gibt es für den Tag keinen Kurs, wird keiner geraten —
          der Beleg bleibt dann liegen, bis ein Kurs mit seiner Quelle erfasst ist. Der erfasste
          Kurs bleibt als „von Hand" erkennbar.
        </HelpPopover>
      </h3>
      <div className="mt-3 grid grid-cols-[10rem_minmax(0,1fr)_auto] gap-3 items-end">
        <Field
          label={`${currency} je Euro`}
          error={unreadable ? 'Erwartet wird ein Kurs wie 1,0874.' : undefined}
        >
          <Input
            align="right"
            inputMode="decimal"
            placeholder="1,0874"
            value={rate}
            onChange={(e) => setRate(e.target.value)}
          />
        </Field>
        <Field label="Quelle" hint="woher der Kurs stammt">
          <Input
            value={source}
            onChange={(e) => setSource(e.target.value)}
            placeholder="EZB-Referenzkurs, abgelesen am …"
          />
        </Field>
        <Button
          variant="secondary"
          loading={busy}
          disabled={writeLock.locked || micros === null || source.trim() === ''}
          title={writeLock.hint}
          onClick={save}
        >
          Kurs übernehmen
        </Button>
      </div>
      {error && <p className="text-body text-negative-text mt-2">{error}</p>}
    </div>
  );
};

/**
 * Die Umrechnung eines Fremdwährungsbelegs.
 *
 * Zwei Kurse und ihre Differenz: der Tageskurs bewertet den Aufwand, der
 * Durchschnittskurs des Monats die Bemessungsgrundlage der Umsatzsteuer
 * (§ 16 Abs. 6 UStG). Dass beide auseinandergehen, ist kein Fehler — es ist
 * Kursaufwand oder Kursertrag, und wer den Buchungssatz später liest, sucht
 * diese Zeile.
 */
const ConversionPanel: React.FC<{ conversion?: Conversion; date: string }> = ({
  conversion,
  date,
}) => {
  if (!conversion) return null;
  const rate = conversion.rate;
  return (
    <div className={cn(NOTE, NOTE_TONE.neutral)}>
      <h3 className="text-label text-ink">Umrechnung aus {conversion.currency}</h3>
      <dl className="mt-2 grid grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-1 text-caption">
        <dt className="text-ink-subtle">Tageskurs</dt>
        <dd className="text-ink-muted">
          <span className="num">
            1 € = {formatExchangeRate(rate?.rateMicros ?? 0)} {conversion.currency}
          </span>
          {rate?.source ? ` · ${rate.source}` : ''}
          {rate?.date ? ` · ${formatDate(rate.date)}` : ''}
          {rate?.manual ? ' · von Hand erfasst' : ''}
        </dd>
        <dt className="text-ink-subtle">Aufwand</dt>
        <dd className="text-ink-muted num">{formatCents(conversion.amount)}</dd>
        <dt className="text-ink-subtle">Bemessungsgrundlage</dt>
        <dd className="text-ink-muted">
          <span className="num">{formatCents(conversion.taxBaseAmount)}</span>
          {conversion.vatRate
            ? ` · Durchschnittskurs ${conversion.vatRate.month}`
            : ' · kein Durchschnittskurs hinterlegt, es bleibt beim Tageskurs'}
          <HelpPopover label="Erklärung zur Bemessungsgrundlage">
            Für die Umsatzsteuer wird der monatliche Durchschnittskurs des Bundesministeriums der Finanzen genommen (§ 16 Abs. 6 UStG); fehlt er, bleibt es beim Tageskurs.
          </HelpPopover>
        </dd>
        {conversion.difference !== 0 && (
          <>
            <dt className="text-ink-subtle">Kursdifferenz</dt>
            <dd className="text-ink-muted num">{formatCents(conversion.difference)}</dd>
          </>
        )}
      </dl>
      {conversion.note && <p className="text-caption text-ink-muted mt-2">{conversion.note}</p>}
      <p className="text-caption text-ink-subtle mt-2">Belegdatum {formatDate(date)}</p>
    </div>
  );
};

/** Zeigt den Buchungssatz, den das Backend berechnet hat. */
const PostingPreviewPanel: React.FC<{ preview: PostingPreview | null; error: string | null }> = ({
  preview,
  error,
}) => {
  if (error) {
    return <p className={cn(NOTE, NOTE_TONE.negative, 'text-body text-negative-text')}>{error}</p>;
  }
  if (!preview) {
    return (
      <p className="text-caption text-ink-subtle">
        Der Buchungssatz erscheint, sobald Lieferant und Positionen vollständig sind.
      </p>
    );
  }

  return (
    <Section title="Buchungssatz" divider={false} className="mt-2">
      <Table density="kompakt">
        <Tbody>
          {preview.lines.map((line, index) => (
            <Tr key={index}>
              <Td className="w-14 text-ink-subtle">{line.side === 'S' ? 'Soll' : 'Haben'}</Td>
              <Td code className="w-20">
                {line.account}
              </Td>
              <Td className="max-w-[16rem] truncate text-ink-muted">{line.accountName}</Td>
              <Td numeric>{formatCents(line.amount)}</Td>
            </Tr>
          ))}
          <Tr>
            <Td colSpan={3} className="text-ink-subtle">
              Netto
            </Td>
            <Td numeric className="text-ink-muted">
              {formatCents(preview.net)}
            </Td>
          </Tr>
          <Tr>
            <Td colSpan={3} className="text-ink-subtle">
              Steuer
            </Td>
            <Td numeric className="text-ink-muted">
              {formatCents(preview.tax)}
            </Td>
          </Tr>
          <Tr variant="sum">
            <Td colSpan={3}>Zahlbetrag</Td>
            <Td numeric>{formatCents(preview.gross)}</Td>
          </Tr>
        </Tbody>
      </Table>
    </Section>
  );
};
