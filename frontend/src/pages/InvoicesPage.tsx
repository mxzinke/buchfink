import React, { useEffect, useMemo, useState } from 'react';
import { AlertCircle, FileText, MoreHorizontal, Plus, Trash2 } from 'lucide-react';
import type {
  Account,
  Contact,
  EInvoiceProfileInfo,
  Invoice,
  InvoiceSentVia,
  InvoiceSentViaOption,
  NumberGapReason,
  NumberGapReasonOption,
  NumberGapReport,
  ReceiptPreview,
  InvoiceItem,
  PostingPreview,
  TaxRate,
  TaxTreatment,
  TaxTreatmentInfo,
  UnitCode,
} from '../types';
import { TAX_RATE_NONE, TAX_RATE_REDUCED, TAX_RATE_STANDARD } from '../types';
import { Api } from '../services/api';
import { useWriteLock } from '../components/WriteLock';
import { formatCents, formatDate, formatTaxRate, parseCents } from '../utils/formatters';
import {
  Button,
  Checkbox,
  Dialog,
  EmptyState,
  Field,
  FieldValue,
  HelpPopover,
  Input,
  Menu,
  MenuItem,
  MenuSeparator,
  Notice,
  PageHeader,
  Section,
  Select,
  SkeletonRows,
  StatusBadge,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  cn,
  toast,
  type Status,
} from '../components/ui';

/**
 * Ausgangsrechnungen.
 *
 * Rechnungsnummer und Buchung entstehen beim Ausstellen im Backend: die Nummer
 * kommt aus dem lückenlosen Nummernkreis nach § 14 Abs. 4 Nr. 4 UStG, die
 * Forderung wird im selben Schritt gebucht. Es gibt keinen Zustand, in dem eine
 * Rechnung existiert, aber nicht im Journal steht.
 */

const STATUS: Record<Invoice['status'], Status> = {
  draft: 'entwurf',
  issued: 'offen',
  // Nummer und Buchung stehen, das Dokument fehlt: die Forderung ist offen wie
  // bei jeder ausgestellten Rechnung. Dass das Dokument fehlt, sagt der
  // Hinweisstreifen über der Tabelle — das Statuswort ist abschließend (§11.3).
  issued_pending_document: 'offen',
  paid: 'ausgeglichen',
  cancelled: 'storniert',
};

/** Die Dokumentart in einem Wort — sie entscheidet über den Typcode (BT-3). */
const KIND_LABEL: Record<Invoice['kind'], string> = {
  invoice: 'Rechnung',
  advance: 'Abschlag',
  final: 'Schlussrechnung',
  correction: 'Korrektur',
  cancellation: 'Storno',
};

interface DraftItem {
  description: string;
  quantity: string;
  unit: string;
  unitPrice: string;
  taxRate: TaxRate;
}

const newItem = (rate: TaxRate): DraftItem => ({
  description: '',
  quantity: '1',
  // C62 ist der Schlüssel für „Stück" nach UN/ECE Rec. 20. EN 16931 verlangt an
  // jeder Position einen Schlüssel aus dieser Liste (BT-130); das Wort „Stück"
  // ist dort keiner.
  unit: 'C62',
  unitPrice: '',
  taxRate: rate,
});

/**
 * Das Statuswort einer Zeile.
 *
 * Der Status allein trüge es nicht: das Stornodokument ist nach dem Ausstellen
 * `issued` wie jede Rechnung, aber es ist kein offener Posten, sondern die
 * Buchung, die einen zurücknimmt. §11.3 hat dafür das Wort „Gebucht" — „Offen"
 * verspräche einen Zahlungseingang, den niemand erwartet.
 */
const statusOf = (invoice: Invoice): Status =>
  invoice.kind === 'cancellation' && invoice.status !== 'cancelled'
    ? 'gebucht'
    : // Ein Status, den diese Ansicht nicht kennt, ist ein Zustand aus einer
      // neueren Fassung des Backends — er darf die Seite nicht mitnehmen.
      // „Offen" ist dafür die vorsichtige Auskunft: Der Vorgang ist da und
      // nicht abgeschlossen.
      (STATUS[invoice.status] ?? 'offen');

/**
 * Der Bezug, den §11.2 auf der Gegenbuchung verlangt: „Storno zu RE-…".
 *
 * Er tritt an die Stelle des Statusworts, wie §11.2 es zeigt — nicht als
 * „Storniert": storniert ist die Ursprungsrechnung, und sie trägt das Wort und
 * die Rosé-Zeile bereits. Das Stornodokument ist die Buchung, die sie
 * zurücknimmt; sein Zustand steht in der Farbe des Abzeichens.
 */
const cancellationReference = (invoice: Invoice): string | undefined =>
  invoice.kind === 'cancellation' && invoice.correctsInvoiceNumber
    ? `Storno zu ${invoice.correctsInvoiceNumber}`
    : undefined;

const todayISO = () => new Date().toISOString().split('T')[0];

export const InvoicesPage: React.FC = () => {
  // Ausstellen und Stornieren sind Buchungen; Ansehen und Ausgeben bleiben im
  // Prüfermodus möglich (§10.4).
  const writeLock = useWriteLock();
  const [invoices, setInvoices] = useState<Invoice[]>([]);
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [treatments, setTreatments] = useState<TaxTreatmentInfo[]>([]);
  const [units, setUnits] = useState<UnitCode[]>([]);
  const [profiles, setProfiles] = useState<EInvoiceProfileInfo[]>([]);
  // Versandwege und Lückengründe sind Wertelisten des Fachmodells und kommen
  // wie Einheiten und Profile aus dem Backend: dieselben Wörter zweimal zu
  // pflegen, heißt sie einmal zu ändern und einmal zu vergessen.
  const [sentViaOptions, setSentViaOptions] = useState<InvoiceSentViaOption[]>([]);
  const [gapReasons, setGapReasons] = useState<NumberGapReasonOption[]>([]);
  const [paymentAccounts, setPaymentAccounts] = useState<Account[]>([]);
  const [gaps, setGaps] = useState<NumberGapReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [preview, setPreview] = useState<{ invoice: Invoice; xml: string } | null>(null);
  const [documentPreview, setDocumentPreview] = useState<
    ({ invoice: Invoice } & ReceiptPreview) | null
  >(null);
  const [cancelling, setCancelling] = useState<Invoice | null>(null);
  const [correcting, setCorrecting] = useState<Invoice | null>(null);
  const [sending, setSending] = useState<Invoice | null>(null);
  const [gapReason, setGapReason] = useState<{ sequence: number; number: string } | null>(null);

  useEffect(() => {
    void load();
  }, []);

  async function load() {
    setLoading(true);
    try {
      const [list, contactList, treatmentList, unitList, profileList, accounts, vias, reasons] =
        await Promise.all([
          Api.getInvoices(),
          Api.getContacts(),
          Api.getTaxTreatments('outgoing'),
          Api.getUnitCodes(),
          Api.getEInvoiceProfiles(),
          Api.getPaymentAccounts(),
          Api.getInvoiceSentViaOptions(),
          Api.getNumberGapReasons(),
        ]);
      setInvoices(list);
      setContacts(contactList.filter((c) => c.type === 'customer'));
      setTreatments(treatmentList);
      setUnits(unitList);
      setProfiles(profileList);
      setPaymentAccounts(accounts);
      setSentViaOptions(vias);
      setGapReasons(reasons);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
    // Der Lückenbericht hat seinen eigenen Fehlerpfad: er ist eine Auskunft
    // über den Nummernkreis und darf die Rechnungsliste nicht mitnehmen, wenn
    // er scheitert.
    try {
      setGaps(await Api.getInvoiceNumberGaps());
    } catch {
      setGaps(null);
    }
  }

  /** Holt ein fehlendes Dokument nach; die Nummer bleibt dieselbe. */
  async function regenerate(invoice: Invoice) {
    try {
      await Api.regenerateInvoiceDocument(invoice.id);
      toast.success(`Dokument zu ${invoice.invoiceNumber} erzeugt.`);
      await load();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    }
  }

  /**
   * Holt alle fehlenden Dokumente nach.
   *
   * Der Hinweisstreifen zählt die betroffenen Rechnungen; ein Knopf, der nur
   * die erste nachholt, ließe ihn stehen und den Anwender raten, wie oft er
   * noch drücken muss. Erzeugt wird nacheinander — jedes Dokument ist ein
   * eigener Vorgang im Backend —, und die Meldung nennt am Ende, was entstanden
   * ist. Bricht eines ab, bleibt der Streifen für den Rest stehen.
   */
  async function regenerateAll(list: Invoice[]) {
    const done: string[] = [];
    try {
      for (const invoice of list) {
        await Api.regenerateInvoiceDocument(invoice.id);
        done.push(invoice.invoiceNumber);
      }
      toast.success(
        done.length === 1
          ? `Dokument zu ${done[0]} erzeugt.`
          : `${done.length} Dokumente erzeugt: ${done.join(', ')}.`,
      );
    } catch (e) {
      const message = e instanceof Error ? e.message : String(e);
      toast.error(
        done.length === 0
          ? message
          : `${done.length} von ${list.length} Dokumenten erzeugt, dann: ${message}`,
      );
    } finally {
      await load();
    }
  }

  async function showZugferd(invoice: Invoice) {
    try {
      const [xml] = await Api.generateInvoiceZUGFeRD(invoice.id);
      setPreview({ invoice, xml });
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    }
  }

  /**
   * Öffnet das archivierte Rechnungsdokument.
   *
   * Gezeigt wird das hybride PDF, das beim Ausstellen abgelegt wurde — nicht eine
   * frische Darstellung. Was der Kunde bekommen hat, ist das, was hier erscheint.
   */
  async function showDocument(invoice: Invoice) {
    try {
      const doc = await Api.getInvoiceDocument(invoice.id);
      setDocumentPreview({ invoice, ...doc });
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    }
  }

  // Offen ist nur, was eine Forderung trägt — und das entscheidet die Art, nicht
  // der Status.
  //
  // Das Stornodokument steht nach dem Ausstellen auf „ausgestellt" wie jede
  // Rechnung, ist aber die gebuchte Rücknahme einer Forderung: gezählt zöge es
  // sein negatives Brutto in die Summe der offenen Posten, und der Kopf zeigte
  // nach dem ersten Storno eine Zahl, die weder Forderung noch Erlös ist.
  //
  // Die Abschlagsrechnung führt ihren offenen Posten im Rechnungsverbund
  // (AdvanceItem.SettledAt) und nicht am Rechnungsstatus; ob sie vereinnahmt
  // ist, weiß diese Seite nicht. Sie stünde hier für immer als offen und gehört
  // auf die Anzahlungsseite, die den Stand kennt.
  const carriesReceivable = (invoice: Invoice) =>
    invoice.kind !== 'cancellation' && invoice.kind !== 'advance';
  const open = invoices.filter(
    (i) =>
      carriesReceivable(i) && (i.status === 'issued' || i.status === 'issued_pending_document'),
  );
  // Die Forderung einer Schlussrechnung ist nicht ihr Brutto: Sie weist die
  // ganze Leistung aus und setzt die vereinnahmten Anzahlungen davon ab
  // (BT-113). Offen ist der Zahlbetrag — dieselbe Rechnung, die das Backend in
  // `Invoice.OpenAmount()` führt und auf das Dokument schreibt. Mit dem Brutto
  // gezählt stünde nach jeder Schlussrechnung eine zu hohe Summe im Kopf.
  const openTotal = open.reduce((sum, i) => sum + i.grossAmount - (i.prepaidAmount ?? 0), 0);
  const advances = invoices.filter((i) => i.kind === 'advance');
  const pendingDocument = invoices.filter((i) => i.status === 'issued_pending_document');

  return (
    <div className="max-w-[1200px] mx-auto px-8 py-8">
      <PageHeader
        title="Ausgangsrechnungen"
        context={
          loading
            ? undefined
            : [
                `${invoices.length} im Geschäftsjahr`,
                `${open.length} offen über ${formatCents(openTotal)}`,
                advances.length > 0
                  ? `${advances.length} ${
                      advances.length === 1 ? 'Abschlag' : 'Abschläge'
                    } auf der Anzahlungsseite`
                  : null,
              ]
                .filter(Boolean)
                .join(' · ')
        }
        action={
          <div className="flex items-center gap-2">
            <HelpPopover label="Erklärung zum Ausstellen">
              Ausstellen und Buchen sind ein Schritt. Die Rechnungsnummer wird lückenlos und
              fortlaufend vergeben, die Forderung sofort auf das Personenkonto des Kunden gebucht.
              Eine Rechnung, die nicht im Journal steht, kann es deshalb nicht geben.
            </HelpPopover>
            <Button
              variant="primary"
              icon={<Plus className="w-4 h-4" strokeWidth={1.5} />}
              disabled={contacts.length === 0 || writeLock.locked}
              title={
                writeLock.hint ??
                (contacts.length === 0 ? 'Zuerst einen Kunden in den Stammdaten anlegen' : undefined)
              }
              onClick={() => setShowForm(true)}
            >
              Neue Rechnung
            </Button>
          </div>
        }
      />

      {pendingDocument.length > 0 && (
        // Hinweisstreifen nach §6.2, Fall 4: die Rechnung ist gebucht, der
        // Kunde hat aber nichts bekommen.
        <Notice
          className="mt-6"
          text={`Zu ${pendingDocument.length} ausgestellten ${
            pendingDocument.length === 1 ? 'Rechnung' : 'Rechnungen'
          } fehlt das Dokument; die Nummer bleibt vergeben.`}
          action={
            <Button
              variant="secondary"
              size="sm"
              disabled={writeLock.locked}
              title={writeLock.hint}
              onClick={() => void regenerateAll(pendingDocument)}
            >
              {pendingDocument.length === 1
                ? `Dokument zu ${pendingDocument[0].invoiceNumber} erzeugen`
                : `Alle ${pendingDocument.length} Dokumente erzeugen`}
            </Button>
          }
        />
      )}

      <Section divider={false} className="mt-8">
        {loading ? (
          <SkeletonRows rows={6} />
        ) : invoices.length === 0 ? (
          <EmptyState
            icon={<FileText className="w-6 h-6" strokeWidth={1.5} />}
            title="Noch keine Rechnungen im aktiven Geschäftsjahr"
            description={
              contacts.length === 0
                ? 'Lege zuerst einen Kunden in den Stammdaten an.'
                : 'Die erste Rechnung wird beim Ausstellen sofort gebucht.'
            }
          />
        ) : (
          <Table>
            <Thead>
              <Tr>
                <Th className="w-32">Nummer</Th>
                <Th className="w-28">Datum</Th>
                <Th className="w-32">Art</Th>
                <Th>Kunde</Th>
                <Th numeric className="w-36">
                  Netto
                </Th>
                <Th numeric className="w-36">
                  Brutto
                </Th>
                <Th className="w-28">Versand</Th>
                <Th className="w-48">Status</Th>
                <Th className="w-24" aria-label="Aktionen" />
              </Tr>
            </Thead>
            <Tbody>
              {invoices.map((invoice) => (
                <Tr
                  key={invoice.id}
                  variant={invoice.status === 'cancelled' ? 'storno' : 'default'}
                  className="group"
                >
                  <Td code>
                    {invoice.invoiceNumber}
                    {/* Beim Storno steht der Bezug im Abzeichen („Storno zu
                        RE-…", §11.2) und wäre hier eine zweite Nennung
                        derselben Sache. Die Berichtigung nennt ihn dagegen
                        hier: sie hat einen eigenen Zustand und trägt im
                        Abzeichen ihr eigenes Statuswort. */}
                    {invoice.correctsInvoiceNumber && invoice.kind !== 'cancellation' && (
                      <span className="block text-caption text-ink-subtle">
                        zu {invoice.correctsInvoiceNumber}
                      </span>
                    )}
                    {/* Die Kette zeigt in beide Richtungen (§11.2): das Storno
                        nennt die Ursprungsrechnung, die Ursprungsrechnung das
                        Dokument, das sie zurückgenommen hat. */}
                    {invoice.cancelledByInvoiceId !== undefined && (
                      <span className="block text-caption text-ink-subtle">
                        storniert durch{' '}
                        {invoices.find((i) => i.id === invoice.cancelledByInvoiceId)
                          ?.invoiceNumber ?? '—'}
                      </span>
                    )}
                  </Td>
                  <Td className="text-ink-subtle num">{formatDate(invoice.date)}</Td>
                  <Td className="text-ink-muted">{KIND_LABEL[invoice.kind] ?? 'Rechnung'}</Td>
                  <Td className="max-w-[20rem] truncate">{invoice.contactName || 'Barverkauf'}</Td>
                  <Td numeric className="text-ink-muted">
                    {formatCents(invoice.netAmount, invoice.currency)}
                  </Td>
                  <Td numeric>{formatCents(invoice.grossAmount, invoice.currency)}</Td>
                  <Td className="text-ink-subtle num">
                    {invoice.sentAt ? formatDate(invoice.sentAt) : '—'}
                  </Td>
                  <Td>
                    <StatusBadge
                      status={statusOf(invoice)}
                      reference={cancellationReference(invoice)}
                    />
                  </Td>
                  <Td className="pl-0">
                    <span className="flex items-center justify-end">
                      <Menu
                        trigger={
                          <Button
                            variant="quiet"
                            size="sm"
                            iconOnly
                            title="Aktionen zu dieser Rechnung"
                            aria-label={`Aktionen zu ${invoice.invoiceNumber}`}
                          >
                            <MoreHorizontal className="w-4 h-4" strokeWidth={1.5} />
                          </Button>
                        }
                      >
                        {invoice.receiptId && (
                          <MenuItem onClick={() => showDocument(invoice)}>
                            Dokument ansehen
                          </MenuItem>
                        )}
                        <MenuItem onClick={() => showZugferd(invoice)}>
                          Strukturierten Datensatz ansehen
                        </MenuItem>
                        {invoice.status === 'issued_pending_document' && (
                          <MenuItem
                            disabled={writeLock.locked}
                            title={writeLock.hint}
                            onClick={() => void regenerate(invoice)}
                          >
                            Dokument erneut erzeugen
                          </MenuItem>
                        )}
                        {invoice.status !== 'cancelled' && invoice.status !== 'draft' && (
                          // Versendet werden kann nur, was es gibt: solange das
                          // Dokument fehlt, hat der Empfänger nichts bekommen.
                          <MenuItem
                            disabled={
                              writeLock.locked || invoice.status === 'issued_pending_document'
                            }
                            title={
                              invoice.status === 'issued_pending_document'
                                ? 'Zuerst das Dokument erzeugen'
                                : writeLock.hint
                            }
                            onClick={() => setSending(invoice)}
                          >
                            Als versendet vermerken
                          </MenuItem>
                        )}
                        {invoice.kind !== 'cancellation' &&
                          invoice.status !== 'cancelled' &&
                          invoice.status !== 'draft' && (
                          // Ein Storno des Stornos gibt es nicht: es negierte
                          // die schon negierten Beträge und wäre die
                          // Generalumkehr der Generalumkehr — die
                          // Ursprungsrechnung stünde danach wieder im Journal,
                          // ohne dass irgendein Dokument das sagt. Berichtigt
                          // wird die Ursprungsrechnung (Entscheidung 3); das
                          // Backend weist den Weg ebenfalls zurück.
                          <>
                            <MenuSeparator />
                            <MenuItem
                              disabled={writeLock.locked}
                              title={writeLock.hint}
                              onClick={() => setCorrecting(invoice)}
                            >
                              Rechnung berichtigen
                            </MenuItem>
                            <MenuItem
                              disabled={writeLock.locked}
                              title={writeLock.hint}
                              onClick={() => setCancelling(invoice)}
                            >
                              Stornorechnung ausstellen
                            </MenuItem>
                          </>
                        )}
                      </Menu>
                    </span>
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>

      <Section
        title="Nummernkreis"
        context={
          gaps
            ? `${gaps.issued} Nummern vergeben · ${gaps.used} mit Dokument · ${gaps.gaps.length} ohne`
            : 'Der Lückenbericht ließ sich nicht laden'
        }
        action={
          <HelpPopover label="Erklärung zum Lückenbericht">
            § 14 Abs. 4 Nr. 4 UStG verlangt eine einmalige, fortlaufende Nummer. Nummer, Rechnung
            und Buchung entstehen in einer Transaktion; eine Lücke bleibt daher nur nach einem
            Abbruch oder aus übernommenen Beständen. Die Betriebsprüfung fragt nach jeder einzelnen
            — deshalb wird der Grund festgehalten und nicht erinnert.
          </HelpPopover>
        }
      >
        {!gaps || gaps.gaps.length === 0 ? (
          <EmptyState
            title="Keine Lücke im Rechnungsnummernkreis"
            description={
              gaps
                ? 'Jede vergebene Nummer trägt ein Dokument.'
                : 'Der Bericht steht wieder zur Verfügung, sobald das Geschäftsjahr geladen ist.'
            }
          />
        ) : (
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th className="w-40">Nummer</Th>
                <Th className="w-56">Grund</Th>
                <Th>Vermerk</Th>
                <Th className="w-40">Festgehalten</Th>
                <Th className="w-32" aria-label="Aktionen" />
              </Tr>
            </Thead>
            <Tbody>
              {gaps.gaps.map((gap) => (
                <Tr key={gap.sequence} className="group">
                  <Td code>{gap.number}</Td>
                  <Td className={gap.reason === 'unknown' ? 'text-attention-text' : 'text-ink-muted'}>
                    {gap.label}
                  </Td>
                  <Td className="max-w-[24rem] truncate">{gap.detail || '—'}</Td>
                  <Td className="text-ink-subtle num">
                    {gap.recordedAt ? formatDate(gap.recordedAt.split('T')[0]) : '—'}
                  </Td>
                  <Td className="pl-0">
                    <Button
                      variant="quiet"
                      size="sm"
                      disabled={writeLock.locked}
                      title={writeLock.hint}
                      className="opacity-0 transition-opacity duration-120 ease-quiet
                                 group-hover:opacity-100 focus-visible:opacity-100"
                      onClick={() => setGapReason({ sequence: gap.sequence, number: gap.number })}
                    >
                      Begründen
                    </Button>
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>

      {showForm && (
        <InvoiceForm
          contacts={contacts}
          treatments={treatments}
          units={units}
          profiles={profiles}
          paymentAccounts={paymentAccounts}
          onClose={() => setShowForm(false)}
          onIssued={async (number) => {
            setShowForm(false);
            toast.success(`Rechnung ${number} ausgestellt und gebucht.`);
            await load();
          }}
        />
      )}

      <Dialog
        open={documentPreview !== null}
        onOpenChange={(next) => !next && setDocumentPreview(null)}
        title={documentPreview?.invoice.invoiceNumber ?? ''}
        width="max-w-4xl"
      >
        {documentPreview && (
          <>
            <p className="text-caption text-ink-subtle -mt-1 mb-3">
              Hybrides PDF/A-3 mit eingebettetem ZUGFeRD-XML — das archivierte Dokument, nicht eine
              neue Darstellung.
            </p>
            {!documentPreview.intact && (
              <div className="mb-3 flex items-start gap-2.5 rounded-control border border-negative-line bg-negative-soft px-4 py-3">
                <AlertCircle className="w-4 h-4 mt-0.5 shrink-0 text-negative" strokeWidth={1.5} />
                <p className="text-body text-negative-text">
                  Die Datei im Datenspeicher passt nicht mehr zu ihrer Prüfsumme.
                </p>
              </div>
            )}
            {/* Belegvorschau: fremdes Dokument, deshalb eigene Fläche (§6.2, Fall 3). */}
            <iframe
              title={documentPreview.fileName}
              src={documentPreview.dataUrl}
              className="w-full h-[62vh] rounded-card border border-line bg-sunken"
            />
          </>
        )}
      </Dialog>

      <Dialog
        open={preview !== null}
        onOpenChange={(next) => !next && setPreview(null)}
        title={`ZUGFeRD-XML · ${preview?.invoice.invoiceNumber ?? ''}`}
        width="max-w-3xl"
      >
        {/* Technische Ausgabe, deshalb Monospace auf dunkler Fläche (§4.1). */}
        <pre className="rounded-card border border-shell-line bg-shell text-shell-text
                        font-mono text-caption leading-relaxed p-5 max-h-[60vh] overflow-auto">
          {preview?.xml}
        </pre>
      </Dialog>

      <CancelDialog
        invoice={cancelling}
        onClose={() => setCancelling(null)}
        onDone={async () => {
          setCancelling(null);
          await load();
        }}
      />

      <CorrectDialog
        invoice={correcting}
        units={units}
        onClose={() => setCorrecting(null)}
        onDone={async (number) => {
          setCorrecting(null);
          toast.success(`Berichtigte Rechnung ${number} ausgestellt.`);
          await load();
        }}
      />

      <SentDialog
        invoice={sending}
        options={sentViaOptions}
        onClose={() => setSending(null)}
        onDone={async () => {
          setSending(null);
          await load();
        }}
      />

      <GapReasonDialog
        gap={gapReason}
        reasons={gapReasons}
        year={gaps?.fiscalYear ?? 0}
        onClose={() => setGapReason(null)}
        onDone={async () => {
          setGapReason(null);
          await load();
        }}
      />
    </div>
  );
};

// -------------------------------------------------------------------------

/** Kopf- und Zeilenraster der Positionen. Eine Definition für beide. */
const ITEM_GRID = 'grid grid-cols-[minmax(0,1fr)_5rem_6rem_7rem_6rem_7rem_2rem] gap-2 items-center';

const InvoiceForm: React.FC<{
  contacts: Contact[];
  treatments: TaxTreatmentInfo[];
  units: UnitCode[];
  profiles: EInvoiceProfileInfo[];
  paymentAccounts: Account[];
  onClose: () => void;
  onIssued: (invoiceNumber: string) => void;
}> = ({ contacts, treatments, units, profiles, paymentAccounts, onClose, onIssued }) => {
  const writeLock = useWriteLock();
  const today = todayISO();
  const [contactId, setContactId] = useState(contacts[0]?.id ?? 0);
  const [date, setDate] = useState(today);
  const [serviceFrom, setServiceFrom] = useState(today);
  const [serviceTo, setServiceTo] = useState(today);
  const [treatment, setTreatment] = useState<TaxTreatment>('domestic');
  const [items, setItems] = useState<DraftItem[]>([newItem(TAX_RATE_STANDARD)]);
  const [dueDays, setDueDays] = useState('');
  const [discountPermille, setDiscountPermille] = useState('');
  const [discountDays, setDiscountDays] = useState('');
  const [smallAmount, setSmallAmount] = useState(false);
  const [paymentAccount, setPaymentAccount] = useState('');
  const [busy, setBusy] = useState(false);
  const [preview, setPreview] = useState<PostingPreview | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);

  const contact = contacts.find((c) => c.id === contactId);
  const treatmentInfo = treatments.find((t) => t.treatment === treatment);
  const taxable = treatment === 'domestic';
  // § 33 Satz 2 UStDV nimmt die innergemeinschaftliche Lieferung, den
  // Fernverkauf und die Steuerschuldnerschaft des Leistungsempfängers von der
  // Kleinbetragsrechnung aus. Angeboten wird sie deshalb nur beim
  // steuerpflichtigen Inlandsumsatz.
  //
  // Die Betragsgrenze kommt datiert aus dem Backend und reist mit der Vorschau
  // (PostingPreview.smallAmountLimit) — dort steht auch der Bruttobetrag, gegen
  // den sie zu vergleichen ist. Nachgerechnet wird hier nichts; die Zahl liegt
  // an einer Stelle, und das ist die, die sie datiert kennt.
  const smallAmountLimit = preview?.smallAmountLimit ?? 0;
  const overSmallAmountLimit = smallAmountLimit > 0 && (preview?.gross ?? 0) > smallAmountLimit;
  const smallAmountPossible = taxable && !overSmallAmountLimit;
  // Der Barverkauf (Kleinbetrag ohne erfassten Empfänger) geht als reines PDF
  // hinaus: EN 16931 verlangt den Namen des Erwerbers (BR-07), § 33 UStDV
  // erlässt ihn, und Kleinbetragsrechnungen sind von der E-Rechnungspflicht
  // ausgenommen. Das Backend setzt in diesem Fall pdf_only (invoice_service.go);
  // stünde hier weiter das Profil des Standardfalls, zeigte der Dialog ein
  // Format an, das nicht erzeugt wird.
  const cashSale = smallAmount && contactId === 0;
  const effectiveProfile = cashSale
    ? 'pdf_only'
    : contact?.eInvoiceProfile || 'zugferd_en16931';
  const profile = profiles.find((p) => p.profile === effectiveProfile);
  const missingLeitwegID =
    !cashSale && contact?.eInvoiceProfile === 'xrechnung_cii' && !contact.leitwegId;

  useEffect(() => {
    if (!smallAmountPossible) setSmallAmount(false);
  }, [smallAmountPossible]);

  // „Ohne Empfänger" gibt es nur bei der Kleinbetragsrechnung (§ 33 UStDV).
  // Fällt die Option weg, zeigte die Auswahl auf einen Eintrag, den die Liste
  // nicht mehr führt: das Feld stünde auf dem Platzhalter, die Vorschau bliebe
  // leer, und die Rechnung liefe bis zur Fehlermeldung des Backends.
  useEffect(() => {
    if (!smallAmount && contactId === 0) setContactId(contacts[0]?.id ?? 0);
  }, [smallAmount, contactId, contacts]);

  // Die Positionen in der Form, die das Backend erwartet. Hier wird nur
  // umgerechnet, nicht gerechnet: Mengen von drei Nachkommastellen auf Milli,
  // Preise auf Cent.
  const draft = useMemo(
    () => ({
      // Ohne Empfänger (0) ist die Rechnung ein Barverkauf; das geht nur als
      // Kleinbetragsrechnung (§ 33 UStDV), und dann gibt es kein Personenkonto,
      // gegen das eine Forderung liefe. Dass die 0 nur dort entstehen kann,
      // hält der Effekt oben fest — hier wird sie nur weitergereicht.
      contactId,
      date,
      serviceDateFrom: serviceFrom,
      serviceDateTo: serviceTo,
      taxTreatment: treatment,
      currency: 'EUR',
      smallAmount,
      paymentAccount: smallAmount && contactId === 0 ? paymentAccount : undefined,
      terms: {
        dueDays: Number.parseInt(dueDays, 10) || 0,
        discountPermille: Number.parseInt(discountPermille, 10) || 0,
        discountDays: Number.parseInt(discountDays, 10) || 0,
      },
      items: items.map((item, index) => ({
        position: index + 1,
        description: item.description,
        quantityMilli: Math.round((Number(item.quantity.replace(',', '.')) || 0) * 1000),
        unit: item.unit,
        unitPrice: parseCents(item.unitPrice) ?? 0,
        taxRate: item.taxRate,
      })) as InvoiceItem[],
    }),
    [
      contactId,
      date,
      serviceFrom,
      serviceTo,
      treatment,
      items,
      smallAmount,
      paymentAccount,
      dueDays,
      discountPermille,
      discountDays,
    ],
  );

  // Netto, Steuer und Brutto kommen aus dem Backend. Diese Maske hat die
  // Steuerrechnung früher selbst nachgebaut, samt Rundung je Steuersatzgruppe —
  // eine zweite Wahrheit, die auseinanderläuft, sobald ein Steuerfall dazukommt.
  useEffect(() => {
    const complete =
      (contactId > 0 || smallAmount) && draft.items.every((i) => i.description && i.unitPrice > 0);
    if (!complete) {
      setPreview(null);
      setPreviewError(null);
      return;
    }
    let cancelled = false;
    const timer = window.setTimeout(() => {
      Api.previewOutgoingInvoice(draft)
        .then((p) => {
          if (cancelled) return;
          setPreview(p);
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
  }, [draft, contactId, smallAmount]);

  function update(index: number, patch: Partial<DraftItem>) {
    setItems((prev) => prev.map((item, i) => (i === index ? { ...item, ...patch } : item)));
  }

  async function submit() {
    setBusy(true);
    try {
      const invoice = await Api.issueInvoice(draft);
      onIssued(invoice.invoiceNumber);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open
      onOpenChange={(next) => !next && onClose()}
      title="Neue Rechnung"
      width="max-w-4xl"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            loading={busy}
            disabled={!preview || preview.gross <= 0 || writeLock.locked}
            title={writeLock.hint}
            onClick={submit}
          >
            Ausstellen und buchen
          </Button>
        </>
      }
    >
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <Field
          label="Art"
          hint="Abschlag und Schlussrechnung: Seite „Anzahlungen&quot;"
          explain="Abschlags- und Schlussrechnung gehören in einen Rechnungsverbund: er hält den vereinbarten Gesamtbetrag, die gestellten Abschläge und die Verrechnung zusammen. Beim Abschlag entsteht die Steuer außerdem erst mit der Vereinnahmung (§ 13 Abs. 1 Nr. 1 Buchst. a Satz 4 UStG), es gibt also zwei Buchungszeitpunkte. Beides steht auf der Seite „Anzahlungen&quot;; hier entsteht die gewöhnliche Rechnung."
        >
          {/* Die Art ist hier keine Auswahl, sondern die Auskunft, welche
              gerade entsteht: eine Abschlagsrechnung ohne Verbund wäre eine
              Anzahlung ohne den Auftrag, auf den sie sich anrechnet — und
              genau die Absetzung in der Schlussrechnung ist der teuerste
              Fehler des Themas (§ 14c Abs. 1 UStG). Der Weg dorthin steht im
              Hinweis, statt als dritter Eintrag in einer Liste, die zu nichts
              führt. */}
          <FieldValue>Rechnung</FieldValue>
        </Field>

        <Field
          label="Kunde"
          hint={
            contact
              ? `${contact.countryCode || 'DE'}${
                  contact.vatId ? ` · USt-IdNr. ${contact.vatId}` : ' · keine USt-IdNr. hinterlegt'
                }`
              : undefined
          }
        >
          <Select
            items={[
              ...(smallAmount ? [{ value: 0, label: 'Ohne Empfänger (Barverkauf)' }] : []),
              ...contacts.map((c) => ({
                value: c.id,
                label: `${c.name} · Debitor ${c.ledgerAccount}`,
              })),
            ]}
            value={contactId}
            onValueChange={setContactId}
          />
        </Field>

        <Field
          label="Steuerfall"
          hint={treatmentInfo?.hint}
          help="Der Steuerfall entscheidet über Erlöskonto und Steuerzeile. Für steuerfreie Lieferungen ins EU-Ausland ist die USt-IdNr. des Empfängers Voraussetzung."
        >
          <Select
            items={treatments.map((t) => ({ value: t.treatment, label: t.label }))}
            value={treatment}
            onValueChange={setTreatment}
          />
        </Field>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mt-4">
        <Field label="Rechnungsdatum">
          <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
        </Field>
        <Field label="Leistung von">
          <Input type="date" value={serviceFrom} onChange={(e) => setServiceFrom(e.target.value)} />
        </Field>
        <Field label="Leistung bis">
          <Input type="date" value={serviceTo} onChange={(e) => setServiceTo(e.target.value)} />
        </Field>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mt-4">
        <Field
          label="Format"
          hint={profile?.label}
          explain={
            <>
              {profile?.hint}
              {cashSale &&
                ' Ohne erfassten Empfänger geht die Kleinbetragsrechnung als reines PDF hinaus; Kleinbetragsrechnungen sind von der E-Rechnungspflicht ausgenommen (§ 33 UStDV).'}
              {missingLeitwegID &&
                ' Für die XRechnung fehlt die Leitweg-ID dieses Empfängers; ohne sie weist Buchfink die Ausstellung zurück (BR-DE-15).'}
            </>
          }
        >
          {/* Das Format ist eine Eigenschaft des Empfängers und wird an ihm
              gepflegt, nicht je Rechnung gewählt: eine Behörde nimmt XRechnung
              und sonst nichts, und wer bei jeder Rechnung neu wählt, wählt
              irgendwann falsch. */}
          <FieldValue>{profile?.label ?? '—'}</FieldValue>
        </Field>

        <Field
          label="Kleinbetragsrechnung"
          hint={
            !taxable
              ? 'nur beim Inlandsumsatz'
              : overSmallAmountLimit
                ? `nur bis ${formatCents(smallAmountLimit)} brutto`
                : undefined
          }
          explain={
            <>
              § 33 UStDV lässt bei kleinen Beträgen die verkürzten Angaben zu: kein Empfänger,
              Bruttobetrag mit Steuersatz. Bei innergemeinschaftlicher Lieferung und
              Steuerschuldnerschaft des Leistungsempfängers ist sie ausgeschlossen.
              {smallAmountLimit > 0 &&
                ` Die Grenze am Rechnungsdatum liegt bei ${formatCents(smallAmountLimit)} brutto.`}
              {overSmallAmountLimit &&
                ' Dieser Betrag liegt darüber; die Rechnung braucht die vollständigen Angaben nach § 14 Abs. 4 UStG.'}
            </>
          }
        >
          <Checkbox
            checked={smallAmount}
            disabled={!smallAmountPossible}
            onCheckedChange={(checked) => setSmallAmount(Boolean(checked))}
            label="Als Kleinbetragsrechnung ausstellen"
          />
        </Field>
      </div>

      {smallAmount && contactId === 0 && (
        <Field
          label="Zahlungsmittel"
          hint="Leer heißt Kasse"
          className="mt-4 max-w-sm"
          explain="Ohne erfassten Kunden gibt es kein Personenkonto und keine Forderung: der Barverkauf ist im selben Augenblick bezahlt und wird gegen Kasse oder Bank gebucht."
        >
          <Select
            items={paymentAccounts.map((a) => ({
              value: a.number,
              label: `${a.number} · ${a.name}`,
            }))}
            value={paymentAccount}
            onValueChange={setPaymentAccount}
            placeholder="Kasse"
          />
        </Field>
      )}

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 mt-4">
        <Field
          label="Zahlungsziel"
          hint="Tage"
          optional
          explain="Die im Voraus vereinbarte Minderung des Entgelts ist Pflichtangabe (§ 14 Abs. 4 Nr. 7 UStG). Sie steht als Satz auf dem Dokument und als BT-20 im Datensatz; der Skonto selbst mindert Entgelt und Steuer erst, wenn er in Anspruch genommen wird (§ 17 Abs. 1 UStG)."
        >
          <Input
            inputMode="numeric"
            align="right"
            placeholder="14"
            value={dueDays}
            onChange={(e) => setDueDays(e.target.value)}
          />
        </Field>
        <Field label="Skonto" hint="Promille: 20 sind 2 %" optional>
          <Input
            inputMode="numeric"
            align="right"
            placeholder="0"
            value={discountPermille}
            onChange={(e) => setDiscountPermille(e.target.value)}
          />
        </Field>
        <Field label="Skontofrist" hint="Tage" optional>
          <Input
            inputMode="numeric"
            align="right"
            placeholder="0"
            value={discountDays}
            onChange={(e) => setDiscountDays(e.target.value)}
          />
        </Field>
      </div>

      <div className="mt-6 pt-6 border-t border-line">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-label text-ink-muted">Positionen</h3>
          <Button
            variant="quiet"
            size="sm"
            icon={<Plus className="w-3.5 h-3.5" strokeWidth={1.5} />}
            onClick={() => setItems((prev) => [...prev, newItem(TAX_RATE_STANDARD)])}
          >
            Position hinzufügen
          </Button>
        </div>

        <div className={cn(ITEM_GRID, 'text-caption text-ink-subtle mb-1')}>
          <span>Bezeichnung</span>
          <span className="text-right">Menge</span>
          <span>Einheit</span>
          <span className="text-right">Einzelpreis</span>
          <span>USt</span>
          <span className="text-right">Betrag</span>
          <span />
        </div>

        <div className="flex flex-col gap-2">
          {items.map((item, index) => (
            <div key={index} className={ITEM_GRID}>
              <Input
                value={item.description}
                onChange={(e) => update(index, { description: e.target.value })}
                aria-label={`Bezeichnung der Position ${index + 1}`}
              />
              <Input
                align="right"
                inputMode="decimal"
                value={item.quantity}
                onChange={(e) => update(index, { quantity: e.target.value })}
                aria-label={`Menge der Position ${index + 1}`}
              />
              {/* Die Einheit ist ein Schlüssel aus UN/ECE Rec. 20 (BT-130) und
                  kein Freitext: „Stunde" ist dort kein zulässiger Wert, „HUR"
                  ist es, und beim Empfänger stand bisher jede Stunde als Stück. */}
              <Select
                items={units.map((u) => ({ value: u.code, label: u.label }))}
                value={item.unit}
                onValueChange={(unit) => update(index, { unit })}
                aria-label={`Einheit der Position ${index + 1}`}
              />
              <Input
                align="right"
                inputMode="decimal"
                placeholder="0,00"
                value={item.unitPrice}
                onChange={(e) => update(index, { unitPrice: e.target.value })}
                aria-label={`Einzelpreis der Position ${index + 1}`}
              />
              <Select
                items={[TAX_RATE_STANDARD, TAX_RATE_REDUCED, TAX_RATE_NONE].map((rate) => ({
                  value: rate,
                  label: formatTaxRate(rate),
                }))}
                value={item.taxRate}
                onValueChange={(taxRate) => update(index, { taxRate })}
                disabled={!taxable}
                aria-label={`Umsatzsteuersatz der Position ${index + 1}`}
              />
              {/* Menge mal Einzelpreis — dieselbe Rechnung wie InvoiceItem.TotalNet
                  im Backend. Reine Darstellung: Steuer und Rundung je Satzgruppe
                  stehen weiter allein in der Buchungsvorschau. */}
              <span className="text-right num text-body text-ink">
                {formatCents(
                  Math.round(
                    ((draft.items[index]?.unitPrice ?? 0) * (draft.items[index]?.quantityMilli ?? 0)) /
                      1000,
                  ),
                )}
              </span>
              {items.length > 1 ? (
                <Button
                  variant="quiet"
                  size="sm"
                  iconOnly
                  title="Position entfernen"
                  aria-label={`Position ${index + 1} entfernen`}
                  onClick={() => setItems((prev) => prev.filter((_, i) => i !== index))}
                >
                  <Trash2 className="w-4 h-4" strokeWidth={1.5} />
                </Button>
              ) : (
                <span />
              )}
            </div>
          ))}
        </div>

        <div className="mt-6 ml-auto w-72 text-body">
          {previewError ? (
            <p className="text-caption text-negative-text whitespace-pre-line">{previewError}</p>
          ) : !preview ? (
            <p className="text-caption text-ink-subtle text-right">
              Die Summen erscheinen, sobald Kunde und Positionen vollständig sind.
            </p>
          ) : (
            <>
              <div className="flex justify-between py-1">
                <span className="text-ink-muted">Nettobetrag</span>
                <span className="num">{formatCents(preview.net)}</span>
              </div>
              <div className="flex justify-between py-1">
                <span className="text-ink-muted">Umsatzsteuer</span>
                <span className="num">{formatCents(preview.tax)}</span>
              </div>
              <div className="flex justify-between pt-2 mt-1 rule-total font-semibold">
                <span>Gesamtbetrag</span>
                <span className="num">{formatCents(preview.gross)}</span>
              </div>
              {!taxable && (
                <p className="text-caption text-ink-subtle mt-2">
                  Steuerfreier Umsatz. Die Rechnung nennt den Befreiungsgrund.
                </p>
              )}
            </>
          )}
        </div>
      </div>
    </Dialog>
  );
};

// -------------------------------------------------------------------------

const CancelDialog: React.FC<{
  invoice: Invoice | null;
  onClose: () => void;
  onDone: () => void;
}> = ({ invoice, onClose, onDone }) => {
  const writeLock = useWriteLock();
  const [reason, setReason] = useState('');
  // Zwei Fehlerarten, zwei Orte (§10.4): die fehlende Pflichtangabe steht am
  // Feld, das sie meint; die Ablehnung des Backends ist eine fachliche Aussage
  // über den ganzen Vorgang und gehört auf die Hinweisfläche über die Aktionen.
  const [error, setError] = useState<string | null>(null);
  const [failure, setFailure] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (invoice) {
      setReason('');
      setError(null);
      setFailure(null);
    }
  }, [invoice]);

  async function submit() {
    if (!reason.trim()) {
      setError('Ohne Grund lässt sich die Stornierung später nicht nachvollziehen.');
      return;
    }
    setError(null);
    setFailure(null);
    setBusy(true);
    try {
      // Storniert wird mit Dokument: eine stornierte Rechnung ist beim
      // Empfänger in der Welt, und die Rücknahme muss bei ihm ankommen. Das
      // Stornodokument trägt eine eigene Nummer aus demselben Kreis.
      const storno = await Api.cancelInvoiceWithDocument(invoice!.id, reason);
      toast.success(
        `Stornorechnung ${storno.invoiceNumber} zu ${invoice!.invoiceNumber} ausgestellt.`,
      );
      onDone();
    } catch (e) {
      setFailure(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open={invoice !== null}
      onOpenChange={(next) => !next && onClose()}
      title="Stornorechnung ausstellen"
      width="max-w-lg"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button
            variant="danger"
            loading={busy}
            disabled={writeLock.locked}
            title={writeLock.hint}
            onClick={submit}
          >
            Stornorechnung ausstellen
          </Button>
        </>
      }
    >
      {invoice && (
        <>
          <p className="text-body text-ink-muted">
            <span className="code-num text-ink">{invoice.invoiceNumber}</span> über{' '}
            <span className="num text-ink">{formatCents(invoice.grossAmount, invoice.currency)}</span>{' '}
            an {invoice.contactName || 'Barverkauf'} bekommt ein Stornodokument mit eigener Nummer.
            <HelpPopover label="Erklärung zur Stornierung">
              Forderung, Erlös und Umsatzsteuer gehen per Generalumkehr auf null zurück. Das
              Stornodokument trägt die negierten Beträge und den Bezug auf die Ursprungsrechnung;
              diese bleibt unverändert im Archiv. Das Wort „Gutschrift" steht bewusst nirgends: eine
              Gutschrift nach § 14 Abs. 2 Satz 2 UStG ist die Abrechnung des Leistungsempfängers,
              und die stellt Buchfink nicht aus.
            </HelpPopover>
          </p>

          <Field label="Grund der Stornierung" className="mt-4" error={error ?? undefined}>
            <Input
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Leistung nicht erbracht"
            />
          </Field>

          {failure && <Notice tone="negative" text={failure} className="mt-6" />}
        </>
      )}
    </Dialog>
  );
};

// -------------------------------------------------------------------------

/**
 * Rechnung berichtigen: Storno plus neue Rechnung mit vollständigem Inhalt.
 *
 * Zwei Dokumente und nicht eines. Eine „Korrekturrechnung über die Differenz"
 * wäre zulässig, lässt den Empfänger aber zwei Dokumente zusammenrechnen — und
 * in der Praxis rechnet er falsch.
 */
const CorrectDialog: React.FC<{
  invoice: Invoice | null;
  units: UnitCode[];
  onClose: () => void;
  onDone: (invoiceNumber: string) => void;
}> = ({ invoice, units, onClose, onDone }) => {
  const writeLock = useWriteLock();
  const [reason, setReason] = useState('');
  const [date, setDate] = useState(todayISO());
  const [items, setItems] = useState<DraftItem[]>([]);
  // Pflichtangabe am Feld, Ablehnung des Backends auf der Hinweisfläche (§10.4).
  const [error, setError] = useState<string | null>(null);
  const [failure, setFailure] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  // Die berichtigte Rechnung startet mit dem Inhalt der berichtigten: geändert
  // wird meist eine Zeile, und wer alles neu tippt, vertippt sich woanders.
  useEffect(() => {
    if (!invoice) return;
    setReason('');
    setError(null);
    setFailure(null);
    setDate(todayISO());
    setItems(
      invoice.items.map((item) => ({
        description: item.description,
        quantity: String(item.quantityMilli / 1000).replace('.', ','),
        unit: item.unit || 'C62',
        unitPrice: formatCents(item.unitPrice, ''),
        taxRate: item.taxRate,
      })),
    );
  }, [invoice]);

  function update(index: number, patch: Partial<DraftItem>) {
    setItems((prev) => prev.map((item, i) => (i === index ? { ...item, ...patch } : item)));
  }

  async function submit() {
    if (!reason.trim()) {
      setError('Ohne Grund lässt sich die Berichtigung später nicht nachvollziehen.');
      return;
    }
    setBusy(true);
    try {
      const replacement = await Api.correctInvoice(invoice!.id, reason, {
        contactId: invoice!.contactId,
        date,
        serviceDateFrom: invoice!.serviceDateFrom,
        serviceDateTo: invoice!.serviceDateTo,
        taxTreatment: invoice!.taxTreatment,
        currency: invoice!.currency,
        smallAmount: invoice!.smallAmount,
        // Der Barverkauf behält sein Zahlungskonto, sonst fiele die berichtigte
        // Rechnung auf die Kasse zurück, auch wenn gegen Bank gebucht war.
        paymentAccount: invoice!.paymentAccount,
        terms: invoice!.terms,
        items: items.map((item, index) => ({
          position: index + 1,
          description: item.description,
          quantityMilli: Math.round((Number(item.quantity.replace(',', '.')) || 0) * 1000),
          unit: item.unit,
          unitPrice: parseCents(item.unitPrice) ?? 0,
          taxRate: item.taxRate,
        })),
      });
      onDone(replacement.invoiceNumber);
    } catch (e) {
      setFailure(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open={invoice !== null}
      onOpenChange={(next) => !next && onClose()}
      title="Rechnung berichtigen"
      width="max-w-4xl"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            loading={busy}
            disabled={items.length === 0 || writeLock.locked}
            title={writeLock.hint}
            onClick={submit}
          >
            Stornieren und berichtigt ausstellen
          </Button>
        </>
      }
    >
      {invoice && (
        <>
          <p className="text-body text-ink-muted">
            <span className="code-num text-ink">{invoice.invoiceNumber}</span> wird storniert; die
            berichtigte Rechnung verweist auf sie.
            <HelpPopover label="Erklärung zur Berichtigung">
              Eine ausgestellte Rechnung wird nicht geändert: GoBD Rz. 58 lässt einen erfassten
              Geschäftsvorfall nicht mehr veränderbar sein, und § 14 Abs. 4 Nr. 4 UStG lässt keine
              zweite Rechnung unter derselben Nummer zu. Es entstehen deshalb zwei Dokumente mit
              eigenen Nummern. Die Steuer folgt dem Tag des Stornodokuments (§ 17 Abs. 1 UStG).
            </HelpPopover>
          </p>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mt-4">
            <Field label="Grund der Berichtigung" error={error ?? undefined}>
              <Input
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                placeholder="Falscher Steuersatz"
              />
            </Field>
            <Field label="Datum der berichtigten Rechnung">
              <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
            </Field>
          </div>

          <div className="mt-6 pt-6 border-t border-line">
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-label text-ink-muted">Positionen der berichtigten Rechnung</h3>
              <Button
                variant="quiet"
                size="sm"
                icon={<Plus className="w-3.5 h-3.5" strokeWidth={1.5} />}
                onClick={() => setItems((prev) => [...prev, newItem(TAX_RATE_STANDARD)])}
              >
                Position hinzufügen
              </Button>
            </div>

            <div className={cn(ITEM_GRID, 'text-caption text-ink-subtle mb-1')}>
              <span>Bezeichnung</span>
              <span className="text-right">Menge</span>
              <span>Einheit</span>
              <span className="text-right">Einzelpreis</span>
              <span>USt</span>
              <span className="text-right">Betrag</span>
              <span />
            </div>

            <div className="flex flex-col gap-2">
              {items.map((item, index) => (
                <div key={index} className={ITEM_GRID}>
                  <Input
                    value={item.description}
                    onChange={(e) => update(index, { description: e.target.value })}
                    aria-label={`Bezeichnung der Position ${index + 1}`}
                  />
                  <Input
                    align="right"
                    inputMode="decimal"
                    value={item.quantity}
                    onChange={(e) => update(index, { quantity: e.target.value })}
                    aria-label={`Menge der Position ${index + 1}`}
                  />
                  <Select
                    items={units.map((u) => ({ value: u.code, label: u.label }))}
                    value={item.unit}
                    onValueChange={(unit) => update(index, { unit })}
                    aria-label={`Einheit der Position ${index + 1}`}
                  />
                  <Input
                    align="right"
                    inputMode="decimal"
                    placeholder="0,00"
                    value={item.unitPrice}
                    onChange={(e) => update(index, { unitPrice: e.target.value })}
                    aria-label={`Einzelpreis der Position ${index + 1}`}
                  />
                  <Select
                    items={[TAX_RATE_STANDARD, TAX_RATE_REDUCED, TAX_RATE_NONE].map((rate) => ({
                      value: rate,
                      label: formatTaxRate(rate),
                    }))}
                    value={item.taxRate}
                    onValueChange={(taxRate) => update(index, { taxRate })}
                    aria-label={`Umsatzsteuersatz der Position ${index + 1}`}
                  />
                  <span className="text-right num text-body text-ink">
                    {formatCents(
                      Math.round(
                        (parseCents(item.unitPrice) ?? 0) *
                          (Number(item.quantity.replace(',', '.')) || 0),
                      ),
                    )}
                  </span>
                  {items.length > 1 ? (
                    <Button
                      variant="quiet"
                      size="sm"
                      iconOnly
                      title="Position entfernen"
                      aria-label={`Position ${index + 1} entfernen`}
                      onClick={() => setItems((prev) => prev.filter((_, i) => i !== index))}
                    >
                      <Trash2 className="w-4 h-4" strokeWidth={1.5} />
                    </Button>
                  ) : (
                    <span />
                  )}
                </div>
              ))}
            </div>
          </div>

          {failure && <Notice tone="negative" text={failure} className="mt-6" />}
        </>
      )}
    </Dialog>
  );
};

// -------------------------------------------------------------------------

/**
 * Der Versandvermerk. Buchfink versendet nicht selbst; wer im Streitfall den
 * Zugang belegen muss, braucht festgehalten, wann und wie die Rechnung
 * hinausgegangen ist.
 */
const SentDialog: React.FC<{
  invoice: Invoice | null;
  options: InvoiceSentViaOption[];
  onClose: () => void;
  onDone: () => void;
}> = ({ invoice, options, onClose, onDone }) => {
  const writeLock = useWriteLock();
  const [date, setDate] = useState(todayISO());
  const [via, setVia] = useState<InvoiceSentVia>('email');
  const [note, setNote] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!invoice) return;
    setDate(invoice.sentAt || todayISO());
    setVia(invoice.sentVia ?? 'email');
    setNote(invoice.sentNote ?? '');
    setError(null);
  }, [invoice]);

  async function submit() {
    setBusy(true);
    try {
      await Api.markInvoiceSent(invoice!.id, date, via, note);
      toast.success(`${invoice!.invoiceNumber} als versendet vermerkt.`);
      onDone();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open={invoice !== null}
      onOpenChange={(next) => !next && onClose()}
      title="Als versendet vermerken"
      width="max-w-lg"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            loading={busy}
            disabled={writeLock.locked}
            title={writeLock.hint}
            onClick={submit}
          >
            Vermerken
          </Button>
        </>
      }
    >
      {invoice && (
        <>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <Field label="Versendet am">
              <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
            </Field>
            <Field
              label="Weg"
              explain="Buchfink verschickt nichts. Der Vermerk ist der Nachweis, dass die Rechnung den Empfänger erreicht hat — § 14 Abs. 1 UStG kennt sie als Abrechnung gegenüber dem Empfänger."
            >
              <Select
                items={options.map((o) => ({ value: o.via, label: o.label }))}
                value={via}
                onValueChange={setVia}
              />
            </Field>
          </div>
          <Field label="Vermerk" optional className="mt-4" error={error ?? undefined}>
            <Input
              value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder="An buchhaltung@kunde.de"
            />
          </Field>
        </>
      )}
    </Dialog>
  );
};

// -------------------------------------------------------------------------

/** Der Grund einer Lücke im Nummernkreis — die Frage der Betriebsprüfung. */
const GapReasonDialog: React.FC<{
  gap: { sequence: number; number: string } | null;
  reasons: NumberGapReasonOption[];
  year: number;
  onClose: () => void;
  onDone: () => void;
}> = ({ gap, reasons, year, onClose, onDone }) => {
  const writeLock = useWriteLock();
  const [reason, setReason] = useState<NumberGapReason>('aborted');
  const [detail, setDetail] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!gap) return;
    setReason('aborted');
    setDetail('');
    setError(null);
  }, [gap]);

  async function submit() {
    setBusy(true);
    try {
      await Api.recordInvoiceNumberGapReason(year, gap!.sequence, reason, detail);
      toast.success(`Lücke ${gap!.number} begründet.`);
      onDone();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open={gap !== null}
      onOpenChange={(next) => !next && onClose()}
      title={`Lücke ${gap?.number ?? ''} begründen`}
      width="max-w-lg"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            loading={busy}
            disabled={writeLock.locked}
            title={writeLock.hint}
            onClick={submit}
          >
            Grund festhalten
          </Button>
        </>
      }
    >
      <Field label="Grund">
        <Select
          items={reasons.map((o) => ({ value: o.reason, label: o.label }))}
          value={reason}
          onValueChange={setReason}
        />
      </Field>
      <Field label="Vermerk" optional className="mt-4" error={error ?? undefined}>
        <Input
          value={detail}
          onChange={(e) => setDetail(e.target.value)}
          placeholder="Abbruch beim Erzeugen des Dokuments"
        />
      </Field>
    </Dialog>
  );
};
