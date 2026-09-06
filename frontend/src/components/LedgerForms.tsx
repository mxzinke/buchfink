import React, { useEffect, useMemo, useState } from 'react';
import { Plus, Trash2 } from 'lucide-react';
import { Api } from '../services/api';
import { usePostingLock, useWriteLock } from './WriteLock';
import { formatCents, formatDate, parseCents } from '../utils/formatters';
import {
  RETENTION_CLASS_LABELS,
  type Account,
  type CustomAccountRequest,
  type Direction,
  type JournalEntry,
  type Receipt,
  type RetentionClass,
  type SelfIssuedReceiptRequest,
  type StatementPositionOption,
  type TaxTreatment,
  type TaxTreatmentInfo,
} from '../types';
import {
  Button,
  Combobox,
  Dialog,
  Field,
  FieldRow,
  HelpPopover,
  Input,
  Notice,
  Select,
  Table,
  Tbody,
  Td,
  Textarea,
  Th,
  Thead,
  Tr,
  toast,
} from './ui';

/**
 * Die Erfassungsmasken der laufenden Buchhaltung (Welle 8).
 *
 * Sie stehen als eigener Baustein und nicht in einer Seite: der Eigenbeleg wird
 * an der Belegablage und in der Handbuchung gebraucht, das eigene Konto in der
 * Kontenübersicht, die Fristverlängerung am Beleg. Zwei Fassungen derselben
 * Maske liefen bei der nächsten Pflichtangabe auseinander — und eine davon
 * bliebe stehen.
 *
 * Die Masken rechnen nichts nach und prüfen keine fachliche Regel: welche
 * Angabe fehlt und ob der Zeitraum noch offen ist, beantwortet das Backend.
 */

/** Der heutige Tag als „JJJJ-MM-TT" — das Format, das das Backend liest. */
export function today(): string {
  return new Date().toISOString().slice(0, 10);
}

/** Beschriftet einen Beleg für die Auswahl: Nummer, Aussteller, Betrag. */
export function receiptLabel(receipt: Receipt): string {
  const parts = [receipt.receiptNumber];
  if (receipt.issuerName) parts.push(receipt.issuerName);
  if (receipt.grossAmount) parts.push(formatCents(receipt.grossAmount, receipt.currency || 'EUR'));
  return parts.join(' · ');
}

/** Die Belege als Einträge einer Auswahlliste. */
export function receiptOptions(receipts: Receipt[]) {
  return receipts.map((receipt) => ({
    value: receipt.id,
    label: receiptLabel(receipt),
    meta: formatDate(receipt.documentDate ?? ''),
  }));
}

/** Die Aufbewahrungsklassen zur Auswahl. Die leere Klasse ist keine Wahl. */
const RETENTION_CLASSES: RetentionClass[] = ['letters', 'vouchers', 'books'];

// -------------------------------------------------------------------------
// Eigenbeleg
// -------------------------------------------------------------------------

/**
 * Der Eigenbeleg, solange er in der Maske steht.
 *
 * Betrag und Steuer sind Zeichenketten und keine Centbeträge: ein Feld, das
 * jede Eingabe sofort in Cent wandelt und formatiert zurückschreibt, lässt sich
 * nicht tippen — nach „1" stünde „1,00" im Feld, und die zweite Ziffer ergäbe
 * „1,002", was keine Zahl mehr ist. Gewandelt wird deshalb erst beim Absenden.
 */
export interface SelfIssuedDraft {
  documentDate: string;
  grossAmount: string;
  taxAmount: string;
  reason: string;
  direction: SelfIssuedReceiptRequest['direction'];
}

export const emptySelfIssued = (): SelfIssuedDraft => ({
  documentDate: today(),
  grossAmount: '',
  taxAmount: '',
  reason: '',
  direction: 'incoming',
});

/** Der Entwurf als Anfrage an das Backend; die Beträge erst hier gelesen. */
export function selfIssuedRequest(draft: SelfIssuedDraft): SelfIssuedReceiptRequest {
  return {
    documentDate: draft.documentDate,
    grossAmount: parseCents(draft.grossAmount) ?? 0,
    taxAmount: parseCents(draft.taxAmount) ?? 0,
    reason: draft.reason,
    direction: draft.direction,
  };
}

/** Trägt der Entwurf eine begonnene Eingabe? */
function selfIssuedTouched(draft: SelfIssuedDraft): boolean {
  return (
    draft.reason.trim() !== '' || draft.grossAmount.trim() !== '' || draft.taxAmount.trim() !== ''
  );
}

interface SelfIssuedFieldsProps {
  value: SelfIssuedDraft;
  onChange: (next: SelfIssuedDraft) => void;
}

/**
 * Die Angaben eines Eigenbelegs.
 *
 * Sie stehen als eigener Baustein, weil sie an zwei Stellen gebraucht werden:
 * im Dialog, der nur den Beleg erzeugt, und in der Handbuchung, die ihn im
 * selben Vorgang mit anlegt.
 */
export const SelfIssuedFields: React.FC<SelfIssuedFieldsProps> = ({ value, onChange }) => (
  <>
    {/* Die Richtung gehört zu den Angaben und nicht nur in den eigenen Dialog:
        sie entscheidet über die Steuerfälle, die zur Buchung wählbar sind
        (eingehend Erwerb/Reverse Charge, ausgehend Lieferung/Ausfuhr). */}
    <FieldRow className="mb-4">
      <Field label="Richtung">
        <Select
          items={[
            { value: 'incoming', label: 'Ausgabe' },
            { value: 'outgoing', label: 'Einnahme' },
          ]}
          value={value.direction ?? 'incoming'}
          onValueChange={(next) =>
            onChange({ ...value, direction: next as SelfIssuedDraft['direction'] })
          }
          aria-label="Richtung des Vorgangs"
        />
      </Field>
    </FieldRow>
    <FieldRow>
      <Field label="Datum des Vorgangs" hint="Nicht der Tag der Erfassung">
        <Input
          type="date"
          value={value.documentDate}
          onChange={(e) => onChange({ ...value, documentDate: e.target.value })}
        />
      </Field>
      <Field label="Betrag brutto">
        <Input
          align="right"
          inputMode="decimal"
          value={value.grossAmount}
          placeholder="0,00"
          onChange={(e) => onChange({ ...value, grossAmount: e.target.value })}
        />
      </Field>
      <Field label="Enthaltene Steuer" optional>
        <Input
          align="right"
          inputMode="decimal"
          value={value.taxAmount}
          placeholder="0,00"
          onChange={(e) => onChange({ ...value, taxAmount: e.target.value })}
        />
      </Field>
    </FieldRow>
    <Field
      label="Grund"
      className="mt-4"
      hint="Was belegt dieser Beleg?"
      explain={
        <>
          Der Grund ist der ganze Inhalt des Dokuments: was aufgewendet oder vereinnahmt wurde und
          warum es dazu keinen fremden Beleg gibt — das Trinkgeld, der verlorene Parkschein, die
          Entnahme aus der Kasse. Buchfink setzt daraus ein PDF und legt es als Original ab, mit
          dem Satz, dass kein Fremdbeleg vorliegt (§ 146 Abs. 1 AO).
        </>
      }
    >
      <Textarea
        value={value.reason}
        onChange={(e) => onChange({ ...value, reason: e.target.value })}
      />
    </Field>
  </>
);

interface SelfIssuedDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: (receipt: Receipt) => void;
}

export const SelfIssuedDialog: React.FC<SelfIssuedDialogProps> = ({
  open,
  onOpenChange,
  onCreated,
}) => {
  const lock = usePostingLock();
  const [draft, setDraft] = useState<SelfIssuedDraft>(emptySelfIssued);
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);

  const submit = async () => {
    setSaving(true);
    setError('');
    try {
      const receipt = await Api.createSelfIssuedReceipt(selfIssuedRequest(draft));
      onCreated(receipt);
      onOpenChange(false);
      setDraft(emptySelfIssued());
      toast.success(`Eigenbeleg ${receipt.receiptNumber} abgelegt.`);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title="Eigenbeleg erstellen"
      dirty={selfIssuedTouched(draft)}
      footer={
        <>
          <Button variant="secondary" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            loading={saving}
            disabled={lock.locked}
            title={lock.hint}
            onClick={submit}
          >
            Eigenbeleg ablegen
          </Button>
        </>
      }
    >
      {error && <Notice tone="negative" text={error} className="mb-5" />}
      <SelfIssuedFields value={draft} onChange={setDraft} />
    </Dialog>
  );
};

// -------------------------------------------------------------------------
// Handbuchung mit Beleg
// -------------------------------------------------------------------------

/** Eine Zeile der Erfassungsmaske, solange sie noch Text ist. */
interface DraftLine {
  side: 'S' | 'H';
  account: string;
  amount: string;
  taxKey: string;
  taxBase: string;
}

const EMPTY_LINE: DraftLine = { side: 'S', account: '', amount: '', taxKey: '', taxBase: '' };

const emptyLines = (): DraftLine[] => [
  { ...EMPTY_LINE, side: 'S' },
  { ...EMPTY_LINE, side: 'H' },
];

interface ManualEntryDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  accounts: Account[];
  receipts: Receipt[];
  onPosted: (entry: JournalEntry) => void;
}

/**
 * Die Handbuchung, die ihren Beleg verlangt (BEL-01).
 *
 * Zwei Wege, einer davon Pflicht: der abgelegte Beleg oder der Eigenbeleg, der
 * in derselben Transaktion entsteht. Welcher Beleg fehlt, welche Steuerzeile
 * unvollständig ist und ob der Zeitraum noch offen ist, sagt das Backend beim
 * Buchen — die Maske hält die Regel nicht ein zweites Mal vor.
 */
export const ManualEntryDialog: React.FC<ManualEntryDialogProps> = ({
  open,
  onOpenChange,
  accounts,
  receipts,
  onPosted,
}) => {
  const lock = usePostingLock();
  const [bookingDate, setBookingDate] = useState(today());
  const [documentDate, setDocumentDate] = useState(today());
  const [serviceDate, setServiceDate] = useState(today());
  const [description, setDescription] = useState('');
  const [treatment, setTreatment] = useState<TaxTreatment>('domestic');
  const [treatments, setTreatments] = useState<TaxTreatmentInfo[]>([]);
  const [lines, setLines] = useState<DraftLine[]>(emptyLines);
  const [route, setRoute] = useState<'existing' | 'self'>('existing');
  const [receiptId, setReceiptId] = useState<number | null>(null);
  const [selfIssued, setSelfIssued] = useState<SelfIssuedDraft>(emptySelfIssued);
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);

  /**
   * Die Richtung des Vorgangs, aus dem Beleg abgeleitet.
   *
   * Sie entscheidet über die wählbaren Steuerfälle: zum Ausgangsvorgang gehören
   * innergemeinschaftliche Lieferung und Ausfuhr, zum Eingangsvorgang Erwerb
   * und Reverse Charge. Ohne gewählten Beleg bleibt es beim Eingang, dem
   * häufigen Fall.
   */
  const direction: Direction =
    route === 'self'
      ? (selfIssued.direction ?? 'incoming')
      : (receipts.find((r) => r.id === receiptId)?.direction ?? 'incoming');

  useEffect(() => {
    // Die Steuerfälle kommen aus dem Fachbereich. Fehlen sie, bleibt die
    // Auswahl leer und das Backend nennt beim Buchen den Grund.
    let current = true;
    Api.getTaxTreatments(direction)
      .then((list) => {
        if (!current) return;
        const available = list ?? [];
        setTreatments(available);
        // Ein Steuerfall der anderen Richtung bliebe sonst stehen und ginge
        // beim Buchen als Fehler zurück; „Inland" gilt in beide Richtungen.
        setTreatment((prev) =>
          available.some((info) => info.treatment === prev) ? prev : 'domestic',
        );
      })
      .catch(() => {
        if (current) setTreatments([]);
      });
    return () => {
      current = false;
    };
  }, [direction]);

  const accountOptions = useMemo(
    () =>
      accounts
        .filter((a) => !a.isRange && !a.isReserved && a.isActive)
        .map((a) => ({ value: a.number, label: `${a.number} ${a.name}`, meta: a.kontenklasseName })),
    [accounts],
  );

  const options = useMemo(() => receiptOptions(receipts), [receipts]);

  const parsed = lines.map((line) => ({ ...line, cents: parseCents(line.amount) ?? 0 }));
  const debitTotal = parsed
    .filter((line) => line.side === 'S')
    .reduce((sum, line) => sum + line.cents, 0);
  const creditTotal = parsed
    .filter((line) => line.side === 'H')
    .reduce((sum, line) => sum + line.cents, 0);

  const updateLine = (index: number, patch: Partial<DraftLine>) =>
    setLines((prev) => prev.map((line, i) => (i === index ? { ...line, ...patch } : line)));

  /**
   * Der Entwurf nach dem Buchen zurück auf Anfang.
   *
   * Sonst stünde der eben gebuchte Satz beim nächsten Öffnen wieder da, und die
   * zweite Buchung wäre die Wiederholung der ersten.
   */
  const reset = () => {
    setBookingDate(today());
    setDocumentDate(today());
    setServiceDate(today());
    setDescription('');
    setLines(emptyLines());
    setRoute('existing');
    setReceiptId(null);
    setSelfIssued(emptySelfIssued());
    setError('');
  };

  /** Steht im Dialog eine begonnene Eingabe? Alle Felder, nicht nur der Text. */
  const dirty =
    description.trim() !== '' ||
    receiptId !== null ||
    selfIssuedTouched(selfIssued) ||
    lines.some(
      (line) =>
        line.account.trim() !== '' ||
        line.amount.trim() !== '' ||
        line.taxKey.trim() !== '' ||
        line.taxBase.trim() !== '',
    );

  const submit = async () => {
    setError('');
    setSaving(true);
    try {
      const entry = await Api.postManualEntry({
        entry: {
          source: 'manual',
          bookingDate,
          documentDate,
          serviceDateFrom: serviceDate,
          serviceDateTo: serviceDate,
          description,
          taxTreatment: treatment,
          lines: parsed.map((line, index) => ({
            position: index + 1,
            side: line.side,
            account: line.account.trim(),
            amount: line.cents,
            taxKey: line.taxKey.trim() || undefined,
            taxBase: parseCents(line.taxBase) ?? undefined,
          })) as JournalEntry['lines'],
        },
        receiptId: route === 'existing' ? (receiptId ?? 0) : undefined,
        selfIssued: route === 'self' ? selfIssuedRequest(selfIssued) : undefined,
      });
      onPosted(entry);
      onOpenChange(false);
      reset();
      toast.success(`Buchung ${entry.entryNumber} erfasst.`);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title="Buchung mit Beleg erfassen"
      width="max-w-3xl"
      dirty={dirty}
      footer={
        <>
          <Button variant="secondary" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            loading={saving}
            disabled={lock.locked}
            title={lock.hint}
            onClick={submit}
          >
            Buchen
          </Button>
        </>
      }
    >
      {error && <Notice tone="negative" text={error} className="mb-5" />}

      <FieldRow>
        <Field label="Buchungsdatum">
          <Input type="date" value={bookingDate} onChange={(e) => setBookingDate(e.target.value)} />
        </Field>
        <Field label="Belegdatum">
          <Input
            type="date"
            value={documentDate}
            onChange={(e) => setDocumentDate(e.target.value)}
          />
        </Field>
        <Field
          label="Leistungsdatum"
          explain={
            <>
              Der Leistungszeitpunkt ist Pflichtangabe nach § 14 Abs. 4 Nr. 6 UStG, und der
              Steuersatz folgt ihm: eine heute nacherfasste Leistung aus dem zweiten Halbjahr 2020
              trägt 16 Prozent.
            </>
          }
        >
          <Input type="date" value={serviceDate} onChange={(e) => setServiceDate(e.target.value)} />
        </Field>
      </FieldRow>

      <FieldRow className="mt-4">
        <Field label="Buchungstext" className="flex-1">
          <Input value={description} onChange={(e) => setDescription(e.target.value)} />
        </Field>
        <Field
          label="Steuerfall"
          explain={
            <>
              Der Steuerfall sagt, warum Steuer entsteht oder nicht — Inlandsumsatz,
              innergemeinschaftlicher Erwerb, Umkehr der Steuerschuld nach § 13b UStG oder
              steuerfreier Vorgang. Ohne ihn liefe die Buchung an der Voranmeldung vorbei.
            </>
          }
        >
          <Select
            items={treatments.map((t) => ({ value: t.treatment, label: t.label }))}
            value={treatment}
            onValueChange={(next) => setTreatment(next as TaxTreatment)}
            aria-label="Steuerfall der Buchung"
          />
        </Field>
      </FieldRow>

      <div className="mt-6">
        <div className="flex items-center justify-between mb-2">
          <h3 className="text-label text-ink-muted">Buchungssatz</h3>
          <span className="text-caption text-ink-subtle num">
            {`Soll ${formatCents(debitTotal)} · Haben ${formatCents(creditTotal)} · Differenz ${formatCents(debitTotal - creditTotal)}`}
          </span>
        </div>
        <Table>
          <Thead>
            <Tr>
              <Th className="w-24">Seite</Th>
              <Th>Konto</Th>
              <Th numeric className="w-36">
                Betrag
              </Th>
              <Th className="w-28">Steuerschlüssel</Th>
              <Th numeric className="w-36">
                Bemessungsgrundlage
              </Th>
              <Th className="w-10" aria-label="Zeile entfernen" />
            </Tr>
          </Thead>
          <Tbody>
            {lines.map((line, index) => (
              <Tr key={index}>
                <Td>
                  <Select
                    items={[
                      { value: 'S', label: 'Soll' },
                      { value: 'H', label: 'Haben' },
                    ]}
                    value={line.side}
                    onValueChange={(next) => updateLine(index, { side: next as 'S' | 'H' })}
                    aria-label={`Seite der Zeile ${index + 1}`}
                  />
                </Td>
                <Td>
                  <Combobox
                    items={accountOptions}
                    value={line.account || null}
                    onValueChange={(next) => updateLine(index, { account: next ?? '' })}
                    placeholder="Konto suchen …"
                  />
                </Td>
                <Td>
                  <Input
                    align="right"
                    inputMode="decimal"
                    value={line.amount}
                    placeholder="0,00"
                    onChange={(e) => updateLine(index, { amount: e.target.value })}
                    aria-label={`Betrag der Zeile ${index + 1}`}
                  />
                </Td>
                <Td>
                  <Input
                    value={line.taxKey}
                    placeholder="VST19"
                    onChange={(e) => updateLine(index, { taxKey: e.target.value })}
                    aria-label={`Steuerschlüssel der Zeile ${index + 1}`}
                  />
                </Td>
                <Td>
                  <Input
                    align="right"
                    inputMode="decimal"
                    value={line.taxBase}
                    placeholder="0,00"
                    onChange={(e) => updateLine(index, { taxBase: e.target.value })}
                    aria-label={`Bemessungsgrundlage der Zeile ${index + 1}`}
                  />
                </Td>
                <Td className="text-right">
                  <Button
                    variant="quiet"
                    size="sm"
                    iconOnly
                    title="Zeile entfernen"
                    aria-label={`Zeile ${index + 1} entfernen`}
                    disabled={lines.length <= 2}
                    onClick={() => setLines((prev) => prev.filter((_, i) => i !== index))}
                  >
                    <Trash2 className="w-3.5 h-3.5" strokeWidth={1.5} />
                  </Button>
                </Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
        <Button
          variant="quiet"
          size="sm"
          className="mt-2"
          icon={<Plus className="w-3.5 h-3.5" strokeWidth={1.5} />}
          onClick={() => setLines((prev) => [...prev, { ...EMPTY_LINE, side: 'H' }])}
        >
          Zeile hinzufügen
        </Button>
      </div>

      <div className="mt-6">
        <div className="flex items-center mb-2">
          <h3 className="text-label text-ink-muted">Beleg</h3>
          <HelpPopover label="Erklärung zum Belegzwang">
            Zu jeder Buchung gehört ein Beleg (§ 146 Abs. 1 AO, GoBD Rz. 61). Wähle den abgelegten
            Beleg zu diesem Vorgang — oder erstelle einen Eigenbeleg, wenn es keinen fremden gibt.
            Beleg und Buchung entstehen dann in einem Vorgang.
          </HelpPopover>
        </div>
        {/* Der Weg ist eine Auswahl und kein Knopfpaar: zwei Primärknöpfe neben
            dem Buchen ließen offen, welcher die Aktion des Dialogs ist (§10.4). */}
        <Field label="Belegweg" className="max-w-xs">
          <Select
            items={[
              { value: 'existing', label: 'Abgelegter Beleg' },
              { value: 'self', label: 'Eigenbeleg' },
            ]}
            value={route}
            onValueChange={(next) => setRoute(next as 'existing' | 'self')}
            aria-label="Belegweg der Buchung"
          />
        </Field>
        <div className="mt-4">
          {route === 'existing' ? (
            <Field label="Beleg">
              <Combobox
                items={options}
                value={receiptId}
                onValueChange={setReceiptId}
                placeholder="Beleg suchen …"
                emptyText="Kein abgelegter Beleg gefunden."
              />
            </Field>
          ) : (
            <SelfIssuedFields value={selfIssued} onChange={setSelfIssued} />
          )}
        </div>
      </div>
    </Dialog>
  );
};

// -------------------------------------------------------------------------
// Aufbewahrung überschreiben
// -------------------------------------------------------------------------

interface RetentionDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  receipt: Receipt | null;
  onChanged: (receipt: Receipt) => void;
}

export const RetentionDialog: React.FC<RetentionDialogProps> = ({
  open,
  onOpenChange,
  receipt,
  onChanged,
}) => {
  const lock = useWriteLock();
  const [target, setTarget] = useState<RetentionClass>('books');
  const [reason, setReason] = useState('');
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);

  const submit = async () => {
    if (!receipt) return;
    setError('');
    setSaving(true);
    try {
      const next = await Api.overrideReceiptRetention(receipt.id, target, reason);
      onChanged(next);
      onOpenChange(false);
      setReason('');
      toast.success(`Frist für ${next.receiptNumber} verlängert.`);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title="Aufbewahrungsfrist verlängern"
      dirty={reason.trim() !== ''}
      footer={
        <>
          <Button variant="secondary" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            loading={saving}
            disabled={lock.locked}
            title={lock.hint}
            onClick={submit}
          >
            Frist verlängern
          </Button>
        </>
      }
    >
      {error && <Notice tone="negative" text={error} className="mb-5" />}
      <Field
        label="Aufbewahrungsklasse"
        explain={
          <>
            Verlängert und nie verkürzt: die gesetzliche Frist ist die Untergrenze (§ 257 Abs. 4
            HGB, § 147 Abs. 3 AO). Länger aufzubewahren steht frei und ist oft geboten — etwa bei
            einem laufenden Rechtsstreit. Der Grund steht danach am Beleg und im
            Änderungsprotokoll.
          </>
        }
      >
        <Select
          items={RETENTION_CLASSES.map((value) => ({
            value,
            label: RETENTION_CLASS_LABELS[value],
          }))}
          value={target}
          onValueChange={(next) => setTarget(next as RetentionClass)}
          aria-label="Neue Aufbewahrungsklasse"
        />
      </Field>
      <Field label="Grund" className="mt-4">
        <Textarea value={reason} onChange={(e) => setReason(e.target.value)} />
      </Field>
    </Dialog>
  );
};

// -------------------------------------------------------------------------
// Eigenes Konto
// -------------------------------------------------------------------------

interface CustomAccountDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  positions: StatementPositionOption[];
  onCreated: (account: Account) => void;
}

export const CustomAccountDialog: React.FC<CustomAccountDialogProps> = ({
  open,
  onOpenChange,
  positions,
  onCreated,
}) => {
  const lock = useWriteLock();
  const [draft, setDraft] = useState<CustomAccountRequest>({
    number: '',
    name: '',
    hgbPosition: '',
    taxKeyDefault: '',
    description: '',
  });
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);

  const options = useMemo(
    () =>
      positions.map((p) => ({
        value: p.id,
        label: p.name,
        meta: p.statementType === 'guv' ? 'Gewinn- und Verlustrechnung' : 'Bilanz',
      })),
    [positions],
  );

  const submit = async () => {
    setError('');
    setSaving(true);
    try {
      const account = await Api.createCustomAccount(draft);
      onCreated(account);
      onOpenChange(false);
      setDraft({ number: '', name: '', hgbPosition: '', taxKeyDefault: '', description: '' });
      toast.success(`Konto ${account.number} angelegt.`);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title="Eigenes Konto anlegen"
      dirty={draft.number.trim() !== '' || draft.name.trim() !== ''}
      footer={
        <>
          <Button variant="secondary" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            loading={saving}
            disabled={lock.locked}
            title={lock.hint}
            onClick={submit}
          >
            Konto anlegen
          </Button>
        </>
      }
    >
      {error && <Notice tone="negative" text={error} className="mb-5" />}
      <FieldRow>
        <Field label="Kontonummer" hint="Vier Stellen, freie Nummer">
          <Input
            className="code-num"
            value={draft.number}
            onChange={(e) => setDraft({ ...draft, number: e.target.value })}
          />
        </Field>
        <Field label="Bezeichnung" className="flex-1">
          <Input value={draft.name} onChange={(e) => setDraft({ ...draft, name: e.target.value })} />
        </Field>
      </FieldRow>
      <Field
        label="Gliederungsposition"
        className="mt-4"
        explain={
          <>
            Die Position nach §§ 266, 275 HGB entscheidet, wo das Konto in Bilanz, GuV und
            E-Bilanz erscheint. Ohne sie wäre es ein Konto, auf dem Geld liegt, das im Abschluss
            fehlt — deshalb ist sie Pflicht.
          </>
        }
      >
        <Combobox
          items={options}
          value={draft.hgbPosition || null}
          onValueChange={(next) => setDraft({ ...draft, hgbPosition: next ?? '' })}
          placeholder="Position suchen …"
        />
      </Field>
      <FieldRow className="mt-4">
        <Field label="Steuerschlüssel" optional hint="Vorschlag, keine Automatik">
          <Input
            value={draft.taxKeyDefault}
            onChange={(e) => setDraft({ ...draft, taxKeyDefault: e.target.value })}
          />
        </Field>
        <Field label="Beschreibung" optional className="flex-1">
          <Input
            value={draft.description}
            onChange={(e) => setDraft({ ...draft, description: e.target.value })}
          />
        </Field>
      </FieldRow>
    </Dialog>
  );
};

interface BlockAccountDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  account: Account | null;
  onChanged: (account: Account) => void;
}

/**
 * Das Sperren eines eigenen Kontos. Kein Löschen: die Buchungen zeigen auf die
 * Nummer, und ein gelöschtes Konto ließe Bilanz und Kontoblatt mit einer
 * Nummer zurück, zu der es keine Bezeichnung mehr gibt.
 */
export const BlockAccountDialog: React.FC<BlockAccountDialogProps> = ({
  open,
  onOpenChange,
  account,
  onChanged,
}) => {
  const lock = useWriteLock();
  const [reason, setReason] = useState('');
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);

  const submit = async () => {
    if (!account) return;
    setError('');
    setSaving(true);
    try {
      const next = await Api.setAccountBlocked(account.number, true, reason);
      onChanged(next);
      onOpenChange(false);
      setReason('');
      toast.success(`Konto ${next.number} gesperrt.`);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={account ? `Konto ${account.number} sperren` : 'Konto sperren'}
      dirty={reason.trim() !== ''}
      footer={
        <>
          <Button variant="secondary" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            loading={saving}
            disabled={lock.locked}
            title={lock.hint}
            onClick={submit}
          >
            Konto sperren
          </Button>
        </>
      }
    >
      {error && <Notice tone="negative" text={error} className="mb-5" />}
      <Field label="Grund" hint="Steht im Änderungsprotokoll">
        <Textarea value={reason} onChange={(e) => setReason(e.target.value)} />
      </Field>
    </Dialog>
  );
};
