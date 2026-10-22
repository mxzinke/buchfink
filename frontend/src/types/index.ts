/**
 * Spiegelt die Go-Domäne (internal/domain). Änderungen dort müssen hier
 * nachgezogen werden.
 */

/**
 * Cents ist ein Geldbetrag in ganzen Cent — 119000 sind 1.190,00 €.
 *
 * Das Backend rechnet ausschließlich in Cent, damit die Grundinvariante
 * Soll = Haben exakt prüfbar bleibt. Im Frontend gilt dasselbe: niemals in
 * Euro-Fließkommazahlen umrechnen, zur Anzeige `formatCents` benutzen.
 */
export type Cents = number;

export type Side = 'S' | 'H';
export type EntryKind = 'normal' | 'reversal';
export type EntrySource =
  | 'manual'
  | 'receipt'
  | 'invoice'
  | 'payment'
  | 'opening'
  | 'depreciation'
  | 'closing';

export type Direction = 'incoming' | 'outgoing';

export type TaxTreatment =
  | 'domestic'
  /** § 13b Abs. 1 UStG: sonstige Leistung eines EU-Unternehmers (Kz 46/47). */
  | 'reverse_charge_eu'
  /** § 13b Abs. 2 UStG: übrige Leistungen aus dem Ausland (Kz 84/85). */
  | 'reverse_charge'
  | 'intra_community_acquisition'
  | 'reverse_charge_supply'
  /** Nullsteuersatz § 12 Abs. 3 UStG — steuerpflichtig zum Satz null, nicht steuerfrei. */
  | 'zero_rated'
  | 'exempt'
  | 'not_taxable';

/** Steuersatz in Basispunkten: 1900 = 19 %. */
export type TaxRate = number;

export const TAX_RATE_NONE: TaxRate = 0;
export const TAX_RATE_REDUCED: TaxRate = 700;
export const TAX_RATE_STANDARD: TaxRate = 1900;

export type DifferenceKind =
  | 'none'
  | 'skonto'
  | 'bank_fee'
  | 'rounding'
  | 'currency'
  /** Ausbuchung eines uneinbringlichen Postens — ohne Zahlung, § 17 Abs. 2 Nr. 1 UStG. */
  | 'writeoff';

export type ContactType = 'customer' | 'vendor';
/** Zahlungsweg eines Belegs; `credit_card` bucht gegen 3610, `shareholder_expense` ist die Auslage. */
export type Settlement = 'open' | 'paid' | 'credit_card' | 'shareholder_expense';
export type AccountType = 'asset' | 'liability' | 'equity' | 'revenue' | 'expense' | 'statistical';

// -------------------------------------------------------------------------

export interface TenantConfig {
  id: string;
  name: string;
  dataDir: string;
  createdAt: string;
  /** Zielordner der Sicherung. Leer heißt: keine Sicherung eingerichtet. */
  backupDir?: string;
  lastBackupAt?: string;
}

export interface AppConfig {
  tenants: TenantConfig[];
  activeTenantId: string;
  dataDir: string;
  isConfigured: boolean;
  lastFiscalYear: number;

  // Der Zustand des aktiven Mandanten, vom Backend nach oben gespiegelt. Die
  // Oberfläche liest ihn hier und sucht ihn nicht in der Mandantenliste.
  backupDir: string;
  lastBackupAt: string;
  /** Gilt der Prüfermodus heute noch? Dann weist die Bridge jede Änderung ab. */
  readOnly: boolean;
  readOnlyUntil: string;
  readOnlyReason: string;
  programVersion: string;
  /** Port des MCP-Servers für verbundene Apps; fehlt er, gilt 7465. */
  mcpPort?: number;
}

export interface Account {
  id: number;
  number: string;
  name: string;
  type: AccountType;
  category: string;
  subcategory: string;
  kontenklasse: number;
  kontenklasseName: string;
  positionId: string;
  posten: string;
  balanceSide: string;
  hgbCode: string;
  statementType: string;
  taxRate: number;
  hauptfunktion: string;
  hauptfunktionDesc: string;
  zusatzfunktion: string;
  zusatzfunktionDesc: string;
  abschlusszweck: string;
  isRange: boolean;
  rangeStart: string;
  rangeEnd: string;
  isReserved: boolean;
  /**
   * Selbst angelegt statt aus dem SKR04 (BEL-06). Nur ein solches Konto lässt
   * sich sperren; ein Konto des Kontenrahmens fehlte sonst dort, wo eine
   * Auswertung es erwartet.
   */
  isCustom?: boolean;
  description: string;
  isActive: boolean;
  debitSum: Cents;
  creditSum: Cents;
  balance: Cents;
  bookingsCount: number;
  /** Zahl der Personenkonten, aus denen diese Bilanzposition verdichtet ist. */
  aggregatedAccounts?: number;
}

// -------------------------------------------------------------------------
// Journal

export interface JournalLine {
  id: number;
  entryId: number;
  position: number;
  side: Side;
  amount: Cents;
  account: string;
  accountName?: string;
  contactId?: number;
  taxKey?: string;
  taxBase?: Cents;
  /**
   * Der abziehbare Anteil der Vorsteuer in Promille (§ 15 Abs. 4 UStG).
   *
   * Null heißt „nicht einschlägig", 1000 heißt voll abziehbar. `-1` steht für
   * den ganz ausgeschlossenen Abzug (§ 15 Abs. 1a UStG) — eine Vorsteuer, die
   * es gibt und die niemand ziehen darf, ist etwas anderes als keine.
   */
  inputTaxShare?: number;
  /** Der Betrag der Zeile in der Fremdwährung; null bei einer Buchung in Euro. */
  foreignAmount?: Cents;
  text?: string;
}

export interface JournalEntry {
  id: number;
  fiscalYear: number;
  entryNumber: string;
  bookingDate: string;
  documentDate: string;
  serviceDateFrom: string;
  serviceDateTo: string;
  valueDate?: string;
  description: string;
  source: EntrySource;
  documentNumber?: string;
  receiptId?: number;
  receiptHash?: string;
  taxTreatment?: TaxTreatment;
  contactId?: number;
  bankTxId?: number;
  kind: EntryKind;
  /** Abschlussperiode („13. Monat"): Abschlussbuchung zum Bilanzstichtag. */
  closingPeriod?: boolean;
  reversalOfId?: number;
  reversalReason?: string;
  currency: string;
  exchangeRateMicros: number;
  exchangeRateSource?: string;
  exchangeRateDate?: string;
  postingRuleVersion?: string;
  /** Programmfassung, Bearbeiterkennung und Erfassungsweg der Buchung (UNV-04, UNV-06). */
  appVersion?: string;
  actor?: string;
  /** `ui`, `system` oder `agent:<kurzname>`. */
  channel?: string;
  /**
   * Wann und mit welcher Festschreibung diese Buchung festgeschrieben wurde.
   * Leer heißt: noch nicht festgeschrieben.
   */
  committedAt?: string;
  festschreibungId?: number;
  /**
   * Die Buchung, die diese hier ersetzt — der Weg „stornieren und neu buchen".
   * Die Gegenrichtung ist `reversalOfId` an der Generalumkehr.
   */
  correctsEntryId?: number;
  /** Die Herkunftskennung aus einem Altsystem, bei Eröffnungswerten belegt. */
  legacyRef?: string;
  /** Die vereinbarte Fälligkeit des offenen Postens dieser Buchung. */
  dueDate?: string;
  lines: JournalLine[];
  /**
   * Die Aufzeichnung nach § 4 Abs. 5 Satz 1 Nr. 2 EStG zur Bewirtung. Sie
   * gehört zur Buchung und nicht zum Beleg, weil sich der Abzug nach ihr
   * richtet — und eine Buchung, die eine Bewirtungsbuchung ersetzt, muss sie
   * mitnehmen.
   */
  entertainment?: EntertainmentDetail;
  /** Die Aufzeichnung nach § 4 Abs. 7 EStG zu einem Geschenk. */
  gifts?: GiftRecord[];
  previousHash: string;
  entryHash: string;
  createdAt: string;
}

export interface CounterAccount {
  account: string;
  name: string;
  amount: Cents;
}

export interface AccountLedgerRow {
  entryId: number;
  entryNumber: string;
  bookingDate: string;
  documentDate: string;
  documentNumber?: string;
  description: string;
  kind: EntryKind;
  side: Side;
  debitAmount: Cents;
  creditAmount: Cents;
  runningBalance: Cents;
  counterAccounts: CounterAccount[] | null;
  taxKey?: string;
}

export interface AccountLedger {
  account: Account;
  fiscalYear: number;
  /** Grenzen des ausgewerteten Zeitraums. Leer heißt: das ganze Jahr. */
  from?: string;
  to?: string;
  openingBalance: Cents;
  totalDebit: Cents;
  totalCredit: Cents;
  closingBalance: Cents;
  rowCount: number;
  rows: AccountLedgerRow[] | null;
}

export interface SuSaClassSummary {
  kontenklasse: number;
  kontenklasseName: string;
  totalDebit: Cents;
  totalCredit: Cents;
  totalSaldoDebit: Cents;
  totalSaldoCredit: Cents;
  accountsCount: number;
  accounts: Account[];
}

export interface SuSaOverview {
  fiscalYear: number;
  /** Stichtag, bis zu dem summiert wurde. Leer heißt: das ganze Jahr. */
  cutoff?: string;
  totalDebit: Cents;
  totalCredit: Cents;
  totalSaldoDebit: Cents;
  totalSaldoCredit: Cents;
  isBalanced: boolean;
  difference: Cents;
  classes: SuSaClassSummary[];
}

// -------------------------------------------------------------------------
// Kontierung

export interface PostingGroup {
  key: string;
  label: string;
  category: string;
  hint?: string;
  direction: Direction;
  account: string;
  defaultRate: TaxRate;
  /** Der Steuerfall, den die Gruppe vorschlägt. Leer = Inland, steuerpflichtig. */
  defaultTreatment?: TaxTreatment;
  /** Konto für den nicht abzugsfähigen Anteil, z. B. 6644 bei Bewirtung. */
  nonDeductibleAccount?: string;
  /** Gesetzliche Abzugsquote, die für diese Gruppe gilt. */
  deductibleQuota?: string;
  /**
   * Eine Freigrenze, die über das Konto entscheidet statt den Betrag zu teilen.
   * `gift_per_recipient` ist die des § 4 Abs. 5 Satz 1 Nr. 1 EStG.
   */
  limit?: string;
  /**
   * Der Empfänger ist aufzuzeichnen (§ 4 Abs. 7 EStG). Ohne ihn nimmt das
   * Backend die Buchung nicht an — die Maske muss ihn deshalb erfragen.
   */
  recipientRequired?: boolean;
  /** Zu dieser Gruppe gehört kein Vorsteuerabzug (§ 15 Abs. 1a UStG). */
  inputTaxExcluded?: boolean;
}

/** Die Freigrenze je Empfänger und Wirtschaftsjahr (§ 4 Abs. 5 Satz 1 Nr. 1 EStG). */
export const LIMIT_GIFT_PER_RECIPIENT = 'gift_per_recipient';

export interface TaxTreatmentInfo {
  treatment: TaxTreatment;
  label: string;
  hint: string;
  direction: Direction;
  requiresRate: boolean;
  requiresVatId: boolean;
}

export interface ReceiptPosition {
  postingGroup: string;
  account?: string;
  net: Cents;
  taxRate: TaxRate;
  text?: string;
  /**
   * Der Vorsteuerschlüssel der gemischten Nutzung in Promille. Null heißt voll
   * abziehbar; der nicht abziehbare Teil wird dem Aufwand zugeschlagen
   * (§ 9b Abs. 1 EStG).
   */
  inputTaxShare?: number;
  /** Pflicht unter 1000: die Aufteilung ist eine Schätzung und braucht ihren Maßstab. */
  inputTaxShareReason?: string;
  /** Pflicht auf einem Geschenkekonto: die Aufzeichnung nach § 4 Abs. 7 EStG. */
  gift?: GiftInput;
}

/** Der Empfänger eines Geschenks, wie ihn die Maske übergibt. */
export interface GiftInput {
  /** Der Empfänger als erfasster Geschäftspartner. */
  contactId?: number;
  /** Der Empfänger als Freitext — für den, der nicht in der Kontaktliste steht. */
  name?: string;
  occasion?: string;
}

export interface ReceiptRequest {
  contactId: number;
  /** Der abgelegte Beleg. Pflicht: keine Buchung ohne Beleg. */
  receiptId: number;
  bookingDate: string;
  documentDate: string;
  serviceDateFrom: string;
  serviceDateTo: string;
  description: string;
  taxTreatment: TaxTreatment;
  positions: ReceiptPosition[];
  settlement: Settlement;
  paymentAccount?: string;
  /** Bei einer Auslage der Gesellschafter oder Geschäftsführer, der bezahlt hat. */
  creditorContactId?: number;
  currency?: string;
  /** Pflicht, sobald auf ein Bewirtungskonto gebucht wird. */
  entertainment?: EntertainmentDetail;
  /**
   * Ordnet den Beleg einer Rückstellung zu: gebucht wird dann gegen das
   * Rückstellungskonto und nicht gegen den Aufwand. Was die Rückstellung nicht
   * deckt, bleibt Aufwand.
   */
  provisionId?: number;
  /**
   * Kennzeichnet den Beleg als geleistete Anzahlung und sagt, wofür angezahlt
   * wurde. Gebucht wird dann auf das Konto der geleisteten Anzahlungen statt
   * auf den Aufwand; die Vorsteuer richtet sich nach der Zahlung (§ 15 Abs. 1
   * Satz 1 Nr. 1 Satz 3 UStG).
   */
  advanceTarget?: AdvanceTarget;
  /**
   * Die Endsumme des Belegs in der Fremdwährung — die Kontrollsumme zu den
   * Positionen. Freiwillig; wo sie steht, wird sie gegen die Summe gehalten.
   */
  foreignAmount?: Cents;
  /**
   * Übersteuert einen blockierenden Befund der Rechnungsprüfung. Ohne Grund
   * wird eine Rechnung mit fehlender Pflichtangabe nicht mit Vorsteuer gebucht.
   */
  overrideReason?: string;
  /** Die geleisteten Anzahlungen, die dieser Beleg als Schlussrechnung absetzt. */
  settledAdvanceIds?: number[];
  /**
   * Der Leistungsnachweis, den der Buchungsdialog mit der Buchung mitschickt,
   * und sein Datum (leer heißt heute) — RECH-08. Ohne dieses Feld müsste die
   * Maske zweimal rufen, und ein abgebrochener Vorgang hinterließe einen
   * Vermerk ohne Buchung.
   */
  serviceProof?: string;
  serviceProofAt?: string;
}

/**
 * Der Buchungssatz, wie das Backend ihn berechnet — ohne ihn zu schreiben.
 *
 * Die Oberfläche zeigt diese Zahlen an und rechnet sie nicht selbst nach. Eine
 * zweite Steuerrechnung im Frontend wäre eine zweite Wahrheit, die auseinander
 * läuft, sobald ein Steuerfall dazukommt.
 */
export interface PostingPreview {
  issuingFindings?: string[];
  lines: JournalLine[];
  /** Summe der Aufwands- bzw. Ertragszeilen. */
  net: Cents;
  /** Differenz zwischen Brutto und Netto. Bei § 13b null: gezahlt wird netto. */
  tax: Cents;
  /** Was tatsächlich gezahlt oder vereinnahmt wird — die Gegenzeile. */
  gross: Cents;
  balanced: boolean;
  warnings?: PostingWarning[];
  /**
   * Die Bruttogrenze der Kleinbetragsrechnung am Rechnungsdatum (§ 33 UStDV),
   * datiert im Backend. Fehlt außerhalb der Ausgangsrechnung.
   */
  smallAmountLimit?: Cents;
  /**
   * Die blockierenden Befunde der Rechnungsprüfung. Sie stehen neben den
   * Warnungen und nicht in ihnen: eine Warnung zeigt, ein Befund hält an.
   */
  inputTaxFindings: InputTaxFinding[];
  /** Die Umrechnung eines Fremdwährungsbelegs; fehlt bei einem Beleg in Euro. */
  conversion?: Conversion;
}

/**
 * Ein blockierender Befund der Rechnungsprüfung: eine Pflichtangabe der
 * §§ 14, 14a UStG fehlt, und ohne sie gibt es keinen Vorsteuerabzug.
 */
export interface InputTaxFinding {
  code: string;
  title: string;
  /** Was fehlt, samt der Vorschrift dazu. */
  detail: string;
  /** Lässt sich der Befund durch eine Ergänzung der Stammdaten beheben? */
  fixable: boolean;
}

/** Die Umrechnung eines Fremdwährungsbelegs. */
export interface Conversion {
  currency: string;
  date: string;
  /** Der Tag, dessen Monat den Umsatzsteuerkurs bestimmt: Leistungsende oder frühere Zahlung. */
  vatDate: string;
  /** Der Tageskurs, mit dem der Aufwand bewertet wird. */
  rate: ExchangeRate;
  /**
   * Der Durchschnittskurs, mit dem die Bemessungsgrundlage der Umsatzsteuer
   * gerechnet wird (§ 16 Abs. 6 UStG). Fehlt er, ist mit dem Tageskurs
   * gerechnet, und `note` sagt das.
   */
  vatRate?: VatExchangeRate;
  foreignAmount: Cents;
  /** Gegenwert zum Tageskurs. */
  amount: Cents;
  /** Gegenwert zum Umsatzsteuerkurs. */
  taxBaseAmount: Cents;
  /** Die Differenz zwischen beiden: Kursaufwand oder Kursertrag. */
  difference: Cents;
  note: string;
}

// -------------------------------------------------------------------------
// Belege

export type ReceiptFileRole = 'original' | 'structured' | 'rendering' | 'attachment';
export type ReceiptStatus = 'filed' | 'sealed' | 'discarded';
/**
 * Die Belegart entscheidet über die Buchungspflicht: ein Kontoauszug wird
 * abgelegt, aber nicht gebucht — gebucht werden die Umsätze daraus.
 */
/**
 * Die Belegart. `letter` ist ein Brief ohne Buchung (Handelsbrief oder
 * Schreiben einer Behörde) mit kürzerer Aufbewahrungsfrist als ein
 * Buchungsbeleg (§ 257 Abs. 4 HGB, § 147 Abs. 3 AO).
 */
export type ReceiptKind = 'invoice' | 'ticket' | 'statement' | 'self_issued' | 'letter' | 'other';

export interface ReceiptFile {
  id: number;
  receiptId: number;
  position: number;
  role: ReceiptFileRole;
  fileName: string;
  mimeType: string;
  size: number;
  sha256: string;
  /** Aus einer anderen Datei erzeugt statt empfangen. */
  derived: boolean;
  storedPath: string;
  createdAt: string;
}

export interface Receipt {
  id: number;
  fiscalYear: number;
  receiptNumber: string;
  direction: Direction;
  status: ReceiptStatus;
  kind: ReceiptKind;
  files: ReceiptFile[];
  /** Über die geordnete Dateiliste; steht so in der Buchung. */
  receiptHash: string;
  receivedAt?: string;
  receivedVia?: string;
  journalEntryId?: number;
  discardReason?: string;

  /**
   * Die Kopfdaten (BEL-02). Beim Ablegen freiwillig, beim Buchen Pflicht — mit
   * anderen Pflichtfeldern je Belegart, siehe `Receipt.ValidateHeader` im
   * Backend. Ein Altbeleg hat sie nicht; jede Anzeige braucht deshalb einen
   * Standardwert.
   */
  documentDate?: string;
  issuerName?: string;
  grossAmount?: Cents;
  taxAmount?: Cents;
  currency?: string;
  subject?: string;
  /**
   * Die Aufbewahrungsfrist, die beim Ablegen für diesen Beleg galt.
   * `retentionUntil` ist der letzte Tag der Aufbewahrung.
   */
  retentionClass?: RetentionClass;
  retentionUntil?: string;
  /** Erster Tag, an dem der Beleg gelöscht werden darf — aus dem Backend, nicht nachgerechnet. */
  earliestDeletion?: string;
  /**
   * Der Grund, mit dem jemand die Aufbewahrungsfrist dieses Belegs verlängert
   * hat, samt Zeitpunkt (ARC-01). Verlängert und nie verkürzt: die gesetzliche
   * Frist ist die Untergrenze. Die geltende Frist steht weiterhin in
   * `retentionClass` und `retentionUntil`.
   */
  retentionOverrideReason?: string;
  retentionOverrideAt?: string;

  // E-Rechnung: leer bei einem Scan oder einer gewöhnlichen PDF-Rechnung.
  detectedFormat?: string;
  detectedProfile?: string;
  validatedAt?: string;
  validationRuleset?: string;
  validationVersion?: string;
  validationCoverage?: string;
  validationErrors: number;
  /** Die Befunde als JSON, siehe ValidationFinding. */
  validationFindings?: string;

  /**
   * Der Grund, mit dem ein blockierender Befund der Rechnungsprüfung
   * übersteuert wurde, samt dem Zeitpunkt. Er steht am Beleg und im Protokoll.
   */
  inputTaxOverride?: string;
  inputTaxOverrideAt?: string;

  /**
   * Der Prüfpfad des Eingangsbelegs (RECH-08).
   *
   * `orderReference` bringt die E-Rechnung mit, `serviceProof` ist der Vermerk,
   * mit dem jemand die sachliche Richtigkeit bestätigt hat. Beide stehen
   * außerhalb des Beleg-Hashes: der Vermerk entsteht regelmäßig erst beim
   * Buchen oder danach.
   */
  orderReference?: string;
  serviceProof?: string;
  serviceProofAt?: string;

  createdAt: string;
  updatedAt: string;
}

export interface ReceiptFileInput {
  path: string;
  role: ReceiptFileRole;
}

/**
 * Was Buchfink aus einer empfangenen E-Rechnung liest.
 *
 * Ein Vorschlag, keine Buchung. Die Buchungsgruppe bleibt leer — welches
 * Aufwandskonto zutrifft, sagt keine Rechnung.
 */
export interface EInvoiceProposal {
  request: ReceiptRequest;
  format: string;
  profile: string;
  /** Was der Datensatz zu sein angibt — entscheidet über das Vorzeichen. */
  kind: string;
  kindLabel: string;
  supplierName: string;
  supplierVatId?: string;
  supplierTaxId?: string;
  invoiceNumber: string;
  grossAmount: Cents;
  matchedContact: boolean;
  /** Rechnungen, auf die sich dieser Beleg bezieht (BG-3). */
  precedingInvoices?: string[];
  /** Was nicht gefüllt werden konnte und warum. */
  notes?: string[];
}

/**
 * Das Ergebnis der EN-16931-Prüfung.
 *
 * `coverage` kennt bewusst keinen Wert für „vollständig geprüft": die
 * Referenzumsetzung ist ein Schematron-Regelwerk, das kein Go-Prozessor
 * ausführt. Die geprüften Regeln sind über `Api.getEInvoiceRules()` abrufbar.
 */
export interface ValidationFinding {
  rule: string;
  /** Der Schweregrad stammt aus der Spezifikation, nicht aus Buchfink. */
  severity: 'fatal' | 'warning' | 'information';
  /** Wo im Beleg, z. B. "Position 3". Leer bei Dokumentebene. */
  where?: string;
  /** Die betroffenen Geschäftsbegriffe (BT/BG). */
  terms?: string[];
  message: string;
}

export interface ReceiptPreview {
  dataUrl: string;
  fileName: string;
  mimeType: string;
  /** false, wenn die Datei auf der Platte nicht mehr zu ihrer Prüfsumme passt. */
  intact: boolean;
}

// -------------------------------------------------------------------------
// Offene Posten & Zahlungen

/**
 * Woher ein offener Posten stammt.
 *
 * Die Liste hat zwei Quellen. Der gewöhnliche Posten kommt aus einer Buchung
 * auf einem Personenkonto; die Abschlagsrechnung hat vor der Zahlung keine,
 * weil die Steuer erst mit der Vereinnahmung entsteht. Beide stehen in
 * derselben Liste und gehen beim Ausgleich verschiedene Wege.
 */
export type OpenItemSource = 'journal' | 'advance';

export interface OpenItem {
  source: OpenItemSource;
  /** Die Abschlagsrechnung hinter einem Posten der Quelle „Abschlag". */
  advanceInvoiceId?: number;
  entryId: number;
  entryNumber: string;
  contactId: number;
  contactName: string;
  contactType: ContactType;
  ledgerAccount: string;
  documentNumber: string;
  documentDate: string;
  dueDate: string;
  grossAmount: Cents;
  settledAmount: Cents;
  openAmount: Cents;
  taxRate: TaxRate;
  /**
   * Der Steuerfall der ursprünglichen Buchung. Er entscheidet, wie ein Skonto
   * berichtigt wird: nur beim steuerpflichtigen Inlandsumsatz steckt die Steuer
   * im offenen Betrag, und § 13b und der innergemeinschaftliche Erwerb haben
   * zwei Steuerzeilen statt einer (§ 17 Abs. 1 Satz 5 UStG).
   */
  taxTreatment?: TaxTreatment;
}

/** Eine Zahlung, die einen offenen Posten ausgeglichen hat. */
export interface PaymentAllocation {
  id: number;
  openItemEntryId: number;
  paymentEntryId: number;
  bankTxId?: number;
  contactId: number;
  /** Betrag, um den der offene Posten sinkt — samt Skonto. */
  settledAmount: Cents;
  /** Was auf dem Geldkonto tatsächlich bewegt wurde. */
  cashAmount: Cents;
  differenceKind: DifferenceKind;
  differenceAmount: Cents;
}

/**
 * Die Einzelposten einer Zahlungsbuchung: gegen welchen Beleg welchen Partners
 * die Zahlung lief (GoBD Rz. 36).
 */
export interface PaymentAllocationDetail extends PaymentAllocation {
  openItemEntryNumber: string;
  documentNumber?: string;
  documentDate?: string;
  contactName: string;
  contactType: ContactType;
  ledgerAccount: string;
  description?: string;
}

export interface AllocationRequest {
  openItemEntryId: number;
  settledAmount: Cents;
  differenceKind: DifferenceKind;
  differenceAmount: Cents;
}

export interface PaymentRequest {
  bankTxId?: number;
  paymentAccount: string;
  paymentDate: string;
  valueDate?: string;
  description?: string;
  allocations: AllocationRequest[];
}

// -------------------------------------------------------------------------

export interface BankTransaction {
  id: number;
  fiscalYear: number;
  accountIban: string;
  bookingDate: string;
  valueDate: string;
  /** Positiv = Eingang, negativ = Ausgang. */
  amount: Cents;
  currency: string;
  counterpartyName: string;
  counterpartyIban: string;
  remittanceInfo: string;
  endToEndId: string;
  matchStatus: 'unmatched' | 'matched' | 'ignored';
  ledgerAccount: string;
  /** Der abgelegte Kontoauszug, aus dem dieser Umsatz stammt. */
  statementReceiptId?: number;
  matchedAmount: Cents;
}

/**
 * Das Format, in dem eine Ausgangsrechnung hinausgeht.
 *
 * XRechnung in der UBL-Syntax fehlt: Buchfink hat keinen UBL-Schreiber, und ein
 * Profil anzubieten, das nichts erzeugt, wäre ein Versprechen, das erst beim
 * Ausstellen bricht.
 */
export type EInvoiceProfile = 'zugferd_en16931' | 'xrechnung_cii' | 'pdf_only';

export interface EInvoiceProfileInfo {
  profile: EInvoiceProfile;
  label: string;
  hint: string;
}

export interface Contact {
  id: number;
  type: ContactType;
  ledgerAccount: string;
  name: string;
  company: string;
  email: string;
  /**
   * Straße, Postleitzahl und Ort einzeln. § 14 Abs. 4 Nr. 1 UStG verlangt die
   * vollständige Anschrift des Empfängers, EN 16931 verlangt sie in Feldern
   * (BT-50, BT-52, BT-53).
   */
  street: string;
  postalCode: string;
  city: string;
  taxId: string;
  vatId: string;
  countryCode: string;
  iban: string;
  bic: string;
  paymentTermsDays: number;
  /** Das Zielformat, in dem dieser Empfänger seine Rechnungen bekommt. */
  eInvoiceProfile: EInvoiceProfile;
  /** Route-ID des öffentlichen Auftraggebers (BT-10); bei XRechnung Pflicht. */
  leitwegId: string;
  /** Keine Unternehmerin/kein Unternehmer — dann greift keine E-Rechnungspflicht. */
  isPrivate: boolean;
  /** Kleinunternehmer nach § 19 UStG: darf immer eine sonstige Rechnung stellen. */
  isSmallBusiness: boolean;
  /**
   * Die Freistellungsbescheinigung nach § 48b EStG mit ihrem letzten
   * Gültigkeitstag. Buchfink rechnet den Steuerabzug bei Bauleistungen nicht,
   * führt die Bescheinigung aber und weist 30 Tage vorher auf ihren Ablauf hin.
   */
  exemptionCertificateNumber?: string;
  exemptionCertificateValidUntil?: string;
  openAmount: Cents;
  /**
   * Der Hinweis zum Stand der Bestätigungsabfrage, den das Speichern eines
   * Kontakts mit einer USt-IdNr. aus einem anderen Mitgliedstaat zurückgibt.
   * Nicht gespeichert und nicht blockierend.
   */
  vatIdNotice?: string;
  /**
   * Gesperrt nach einem Löschverlangen: der Geschäftspartner steht in keiner
   * Auswahl mehr, bleibt aber in Buchungen und Exporten sichtbar — die
   * Aufbewahrungspflicht geht dem Löschanspruch vor (Art. 17 Abs. 3 Buchst. b
   * DSGVO, § 257 HGB, § 147 AO).
   */
  blocked: boolean;
  blockedAt?: string;
  blockedReason?: string;
  /**
   * Die Standard-Buchungsgruppe eines Lieferanten. Gesetzt ist sie nur, wenn
   * der Mensch sie in der App bestätigt hat; erst dann bucht ein Agent
   * E-Rechnungen des Lieferanten ohne Rückfrage. Ein Agent schlägt nur vor.
   */
  defaultPostingGroup?: string;
  defaultPostingGroupConfirmedAt?: string;
  defaultPostingGroupConfirmedBy?: string;
  proposedPostingGroup?: string;
  proposedPostingGroupNote?: string;
  /** Der Erfassungsweg des Vorschlags, etwa `agent:claude-code`. */
  proposedPostingGroupBy?: string;
  createdAt: string;
}

/** Aufzeichnung zu einer Bewirtung, § 4 Abs. 5 Satz 1 Nr. 2 EStG. */
export interface EntertainmentDetail {
  place: string;
  day: string;
  participants: string;
  occasion: string;
}

/** Ein Hinweis zur Buchung. Blockiert nie — was folgt, ist eine Rechtsfrage. */
export interface PostingWarning {
  code: string;
  severity: 'info' | 'warning';
  title: string;
  detail: string;
  /** Text zum Weitergeben an den Lieferanten. */
  supplierNote?: string;
}

export interface InvoiceItem {
  id?: number;
  invoiceId?: number;
  position: number;
  description: string;
  /** Menge mit drei Nachkommastellen: 1500 = 1,5. */
  quantityMilli: number;
  unit: string;
  unitPrice: Cents;
  taxRate: TaxRate;
  postingGroup?: string;
}

/**
 * Der Lebenslauf einer Ausgangsrechnung.
 *
 * `issued_pending_document` ist ausgestellt und gebucht, aber ohne Dokument:
 * Nummer und Buchung stehen, das Erzeugen des PDF ist gescheitert. Der Zustand
 * ist sichtbar, weil der Kunde noch nichts bekommen hat — und nachholbar, damit
 * die vergebene Nummer nicht verfällt.
 */
export type InvoiceStatus =
  | 'draft'
  | 'issued'
  | 'issued_pending_document'
  | 'paid'
  | 'cancelled';

/**
 * Die Dokumentart entscheidet über den Typcode (BT-3). Ein Empfängersystem
 * bucht danach: eine Rechnungskorrektur als zweite Rechnung gelesen eröffnet
 * eine zweite Verbindlichkeit.
 */
export type InvoiceKind = 'invoice' | 'advance' | 'final' | 'correction' | 'cancellation';

export type InvoiceSentVia = 'email' | 'portal' | 'post' | 'other';

/**
 * Die im Voraus vereinbarten Zahlungsbedingungen (§ 14 Abs. 4 Nr. 7 UStG,
 * BT-20). Der Skontosatz steht in Promille: 20 sind 2 %.
 */
export interface PaymentTerms {
  dueDays: number;
  discountPermille: number;
  discountDays: number;
}

/** Eine vorausgegangene Rechnung, auf die ein Dokument verweist (BG-3). */
export interface InvoiceReference {
  id: number;
  invoiceId: number;
  /** BT-25 */
  number: string;
  /** BT-26 */
  date: string;
}

export interface Invoice {
  id: number;
  fiscalYear: number;
  invoiceNumber: string;
  date: string;
  serviceDateFrom: string;
  serviceDateTo: string;
  dueDate: string;
  contactId: number;
  contactName: string;
  items: InvoiceItem[];
  taxTreatment: TaxTreatment;
  netAmount: Cents;
  taxAmount: Cents;
  grossAmount: Cents;
  currency: string;
  status: InvoiceStatus;
  journalEntryId?: number;
  /** Der Beleg mit dem hybriden PDF und dem ZUGFeRD-XML. */
  receiptId?: number;
  paidAmount: Cents;
  createdAt: string;
  /** Leer heißt „Rechnung". */
  kind: InvoiceKind;
  terms: PaymentTerms;
  /** Kleinbetragsrechnung nach § 33 UStDV: verkürzte Angaben. */
  smallAmount: boolean;
  /** Das Format, in dem dieses Dokument erzeugt wurde. */
  eInvoiceProfile?: EInvoiceProfile;
  /**
   * Der Grund, mit dem eine Leistung ins übrige Gemeinschaftsgebiet ohne
   * Bestätigung der USt-IdNr. ausgestellt wurde. Er steht an der Rechnung, weil
   * die Frage nach dem Leistungsort später an ihr gestellt wird.
   */
  vatIdOverrideReason?: string;
  /** Bezug auf die berichtigte oder stornierte Rechnung (BG-3). */
  correctsInvoiceId?: number;
  correctsInvoiceNumber?: string;
  correctsInvoiceDate?: string;
  /** Gegenrichtung: das Dokument, das diese Rechnung storniert hat. */
  cancelledByInvoiceId?: number;
  /** Die vorausgegangenen Rechnungen — bei der Schlussrechnung die Abschläge. */
  precedingRefs: InvoiceReference[];
  groupId?: number;
  /** Die abgesetzten Anzahlungen der Schlussrechnung (BT-113). */
  prepaidAmount: Cents;
  /** Zeitpunkt der Vereinnahmung auf einer Abschlagsrechnung. */
  paymentReceivedAt?: string;
  sentAt?: string;
  sentVia?: InvoiceSentVia;
  sentNote?: string;
}

/** Eine Mengeneinheit nach UN/ECE Rec. 20 (BT-130). */
export interface UnitCode {
  code: string;
  label: string;
}

// -------------------------------------------------------------------------
// Nummernkreis: der Lückenbericht

export type NumberGapReason = 'aborted' | 'test' | 'cancelled' | 'unknown';

/** Ein Lückengrund zur Auswahl; die Beschriftung ist die des Berichts. */
export interface NumberGapReasonOption {
  reason: NumberGapReason;
  label: string;
}

/** Eine fehlende Nummer mit dem, was über sie bekannt ist. */
export interface NumberGapEntry {
  sequence: number;
  number: string;
  reason: NumberGapReason;
  label: string;
  detail?: string;
  recordedAt?: string;
}

/** Die Antwort auf „welche Rechnungsnummern fehlen" (§ 14 Abs. 4 Nr. 4 UStG). */
export interface NumberGapReport {
  fiscalYear: number;
  /** So viele Nummern hat der Zähler ausgegeben. */
  issued: number;
  /** So viele davon haben ein Dokument. */
  used: number;
  gaps: NumberGapEntry[];
}

// -------------------------------------------------------------------------
// Anzahlungen: Rechnungsverbund, Abschlag, Schlussrechnung

/** Der offene Posten einer Abschlagsrechnung. */
export interface AdvanceItem {
  id: number;
  groupId: number;
  invoiceId: number;
  contactId: number;
  invoiceNumber: string;
  invoiceDate: string;
  netAmount: Cents;
  taxAmount: Cents;
  grossAmount: Cents;
  taxRate: TaxRate;
  /** Der Tag der Vereinnahmung; erst mit ihm entsteht die Steuer. */
  settledAt?: string;
  settlementEntryId?: number;
  cancelled: boolean;
  settledInFinal: boolean;
}

export interface SettleAdvanceRequest {
  /** Die Abschlagsrechnung, auf die das Geld eingegangen ist. */
  advanceId: number;
  /** Der Bankumsatz, aus dem die Vereinnahmung stammt. */
  bankTxId?: number;
  paymentDate: string;
  paymentAccount: string;
}

/** Wofür angezahlt wurde — die Angabe entscheidet über den Bilanzposten. */
export type AdvanceTarget = 'inventory' | 'tangible' | 'intangible';

/**
 * Woran die Kette zerbrochen ist: `linkage` heißt, eine Buchung wurde
 * eingefügt oder entfernt; `content` heißt, eine Buchung wurde verändert;
 * `receipt` heißt, ihr Beleg passt nicht mehr zum Beleg-Hash der Buchung.
 */
export type IntegrityBreakReason = 'linkage' | 'content' | 'receipt';

export interface IntegrityBreak {
  fiscalYear: number;
  entryId: number;
  entryNumber: string;
  reason: IntegrityBreakReason;
  /** Erwarteter und tatsächlicher Hash, damit der Bruch nachrechenbar ist. */
  expectedHash: string;
  actualHash: string;
  message: string;
}

export interface IntegrityCheckResult {
  isValid: boolean;
  totalEntries: number;
  checkedEntries: number;
  firstBrokenId?: number;
  message: string;
  lastVerifiedHash: string;
  checkedAt: string;
  /** Die geprüften Geschäftsjahre, aufsteigend. Jedes hat eine eigene Kette. */
  fiscalYears: number[];
  /** Alle Brüche, nicht nur der erste. Leer heißt: unversehrt. */
  breaks: IntegrityBreak[];
  /**
   * Das Ergebnis der Prüfung des Änderungsprotokolls. Fehlt, wenn der Aufrufer
   * keinen Protokollzugang hat — dann behauptet das Feld nichts.
   */
  auditChain?: AuditChainResult;
}

export interface CompanySettings {
  managingDirectors?: string;
  supervisoryBoardChair?: string;
  sellerIdentifier?: string;
  /** Der erfasste Name. Das Eingabefeld führt ihn roh. */
  companyName: string;
  /**
   * Beurkundet, aber noch nicht eingetragen. Abgeleitet aus der Gründung und
   * nirgends gespeichert.
   */
  inGruendung?: boolean;
  legalForm: string;
  fiscalYear: number;
  fiscalYearStartMonth: number;
  taxNumber: string;
  vatId: string;
  taxOffice: string;
  /**
   * Zahlungsverbindung auf Rechnungen. Abgeleitet aus dem Bankkonto, das im
   * Bankbereich als Konto für Rechnungen festgelegt ist; nur lesbar.
   */
  iban: string;
  bic: string;
  bankName: string;
  /** Straße oder Postfach. */
  street: string;
  /** Hausnummer; leer bei einem Postfach. */
  houseNumber?: string;
  /** Bundesland am Sitz als Kürzel („BY"); bestimmt die Feiertage im Fristende. */
  federalState?: string;
  /** Steuernummer im 13-stelligen ELSTER-Format, abgeleitet; nur lesbar. */
  taxNumberElster?: string;
  /** Beauftragter Steuerberater; verlängert die Erklärungsfristen (§ 149 Abs. 3 AO). */
  taxAdvisor?: TaxAdvisor;
  /** Merkmale, für die Buchfink keinen Abschluss und keine Steuerrückstellung rechnet. */
  nonProfit?: boolean;
  inLiquidation?: boolean;
  fiscalUnity?: boolean;
  /** Zusatz wie „c/o" oder „Gebäude B". */
  addressAddition: string;
  postalCode: string;
  city: string;
  /** ISO 3166-1 Alpha-2, leer heißt DE. */
  countryCode: string;
  /** Tag der Gründung; bei Kapitalgesellschaften die Beurkundung. */
  foundedOn: string;
  notary: string;
  /** Urkundenrollennummer des Gesellschaftsvertrags. */
  deedNumber: string;
  /** Aktuelles Stammkapital. */
  shareCapital: Cents;
  shareholders: CompanyShareholder[];
  /**
   * Ansprechpartner, Telefon und E-Mail des Ausstellers. Bei einer XRechnung
   * Pflicht (BR-DE-2 bis BR-DE-7): eine Behörde, die zu einer Rechnung nicht
   * zurückfragen kann, weist sie zurück.
   */
  contactName: string;
  contactPhone: string;
  contactEmail: string;
  /**
   * Die Systematik des Rechnungsnummernkreises mit den Platzhaltern {JAHR} und
   * {NR:n}. Leer heißt: die Voreinstellung RE-{JAHR}-{NR:4}.
   */
  invoiceNumberFormat: string;
  /**
   * Dieselbe Systematik für den Belegnummernkreis (BEL-02). Leer heißt: die
   * Voreinstellung ER-{JAHR}-{NR:4}. Bestehende Nummern bleiben gültig.
   */
  receiptNumberFormat: string;
  /**
   * Sitz, Registergericht und Registernummer sind die Pflichtangaben des
   * § 264 Abs. 1a HGB auf jedem Jahresabschluss.
   */
  seat: string;
  registerCourt: string;
  registerNumber: string;
  registeredOn?: string;
  currency: string;
  skr: string;
  vatPeriod: string;
  taxationType: string;
  /**
   * Dauerfristverlängerung nach §§ 46 bis 48 UStDV: jede Voranmeldung wird
   * einen Monat später fällig.
   */
  permanentExtension: boolean;
  /**
   * Die angemeldete Sondervorauszahlung (§ 47 Abs. 1 UStDV). Erfasst wird, was
   * angemeldet wurde — nicht, was Buchfink daraus errechnet.
   */
  specialPrepayment: Cents;
  /** Nach so vielen Tagen fällt ein abgelegter, ungebuchter Beleg auf. */
  receiptCaptureDays: number;
  /**
   * Der Betrag, ab dem ein Eingangsbeleg einen Leistungsnachweis haben muss.
   * Null heißt die Voreinstellung von 1.000 Euro, nicht „kein Nachweis".
   */
  invoiceCheckThreshold: Cents;
  /** Nachfrist für die Festschreibung des Vormonats; 0 heißt Monatsende. */
  commitGraceDays: number;
  /**
   * Legt die Anlegerstellung für § 20 InvStG ausdrücklich fest — normalerweise
   * leer, weil sie aus der Rechtsform folgt. Gebraucht wird sie bei den
   * Ausnahmen des § 20 Abs. 1 Sätze 4 und 5 — Lebens- und Krankenversicherer,
   * Kreditinstitute mit Handelsbestand, Pensionsfonds.
   */
  investorOverride: InvestorType;
}

/** Die Beauftragung eines Steuerberaters mit Jahreserklärungen. */
export interface TaxAdvisor {
  engaged: boolean;
  /** Erster Besteuerungszeitraum, den der Berater erstellt; 0 heißt alle. */
  sinceYear: number;
  /** Übernommene Erklärungen (kst, gewst, ust_jahr, ebilanz); leer heißt alle. */
  declarations: string[];
}

export interface AuditLogEntry {
  id: number;
  /** In UTC geschrieben; die Anzeige rechnet über `formatDateTime` um. */
  timestamp: string;
  action: string;
  entityType: string;
  entityId: string;
  details: string;
  /**
   * Die geänderten Felder als JSON-Objekt, und zwar nur die geänderten
   * (GoBD Rz. 34). Ein leeres `before` heißt: neu angelegt.
   */
  before?: string;
  after?: string;
  /** Bearbeiterkennung, Erfassungsweg und Programmfassung des Vorgangs (UNV-04, UNV-06). */
  actor?: string;
  channel?: string;
  appVersion?: string;
  previousHash?: string;
  entryHash?: string;
}

/** Schränkt die Protokollabfrage ein. Leere Felder heißen: alles. */
export interface AuditFilter {
  action?: string;
  entityType?: string;
  entityId?: string;
  actor?: string;
  channel?: string;
  /** Tagesgrenzen im Format JJJJ-MM-TT, beide einschließlich. */
  from?: string;
  to?: string;
  /** Nur die Lesezugriffe auf personenbezogene Daten (Ausgabe, Prüferpaket, Prüfermodus). */
  access?: boolean;
}

/** Ein einzelner Bruch der Protokollkette. */
export interface AuditChainBreak {
  entryId: number;
  reason: IntegrityBreakReason;
  expectedHash: string;
  actualHash: string;
  message: string;
}

/**
 * Das Ergebnis der Prüfung des Änderungsprotokolls. Anders als das Journal ist
 * das Protokoll eine einzige Kette über alle Geschäftsjahre.
 */
export interface AuditChainResult {
  isValid: boolean;
  totalEntries: number;
  checkedEntries: number;
  firstBrokenId?: number;
  breaks: AuditChainBreak[];
  lastVerifiedHash: string;
  checkedAt: string;
  message: string;
}

export interface Festschreibung {
  id: number;
  fiscalYear: number;
  periodType: string;
  periodLabel: string;
  cutoffDate: string;
  chainHead: string;
  entryCount: number;
  tsaName: string;
  tsaGenTime?: string;
  timestampStatus: string;
  /**
   * Hält fest, dass die Systemzeit beim Festschreiben von der beglaubigten Zeit
   * abwich. Leer heißt: keine nennenswerte Abweichung oder kein Zeitstempel.
   */
  timeDriftNote?: string;
  appVersion?: string;
  actor?: string;
  /** Buchungen, die mit dieser Festschreibung ihren Zeitpunkt bekommen haben. */
  entriesStamped: number;
  createdAt: string;
}

export interface FestschreibungVerification {
  id: number;
  hasTimestamp: boolean;
  isValid: boolean;
  coversCurrent: boolean;
  genTime?: string;
  tsaName: string;
  message: string;
}

// -------------------------------------------------------------------------
// Anlagevermögen (internal/domain/asset.go)

/** Die drei Blöcke des Anlagevermögens nach § 266 Abs. 2 A HGB. */
export type AssetClass = 'intangible' | 'tangible' | 'financial';

export type DepreciationMethod =
  | 'linear'
  | 'degressive'
  /** Die Staffel des § 7 Abs. 2a EStG: 75, 10, 5, 5, 3, 2 % der Anschaffungskosten. */
  | 'electric_vehicle'
  /** Die festen Sätze des § 7 Abs. 4 EStG für Gebäude. */
  | 'building_linear'
  | 'pool'
  | 'immediate'
  | 'none';

export type AssetStatus =
  | 'active'
  | 'fully_written'
  | 'disposed'
  | 'unbooked'
  | 'depreciate_due';

export type DisposalKind = 'sale' | 'scrapped' | 'repayment';

/** Die Fondsarten, an denen § 20 InvStG die Teilfreistellung festmacht. */
export type FundClass = '' | 'equity' | 'mixed' | 'real_estate' | 'foreign_real_estate' | 'other';

/** Die Anlegerstellung, an der § 20 Abs. 1 InvStG die Höhe des Satzes festmacht. */
export type InvestorType = '' | 'basic' | 'corporate';

export type AssetMovementKind =
  | 'transfer'
  | 'acquisition'
  | 'subsequent_cost'
  | 'cost_reduction'
  | 'depreciation'
  | 'special_depreciation'
  | 'impairment'
  | 'write_up'
  | 'income'
  | 'vorabpauschale'
  | 'disposal';

/**
 * Stückzahl in Zehntausendstel: 100 Anteile sind 1_000_000.
 *
 * Wie bei den Beträgen eine ganze Zahl, damit die Summe der Zu- und Abgänge
 * nicht vom Bestand abdriftet — Fondsanteile gibt es in Bruchteilen.
 */
export type Units = number;

/** Devisenkurse werden als Fremdwährungseinheiten je Euro mal einer Million geführt. */
export const RATE_SCALE = 1_000_000;

export interface AssetMovement {
  id: number;
  assetId: number;
  kind: AssetMovementKind;
  /** Konto, das diese Bewegung berührt — nach einer Umbuchung nicht das aktuelle. */
  account?: string;
  date: string;
  fiscalYear: number;
  /** Verändert die Anschaffungs- und Herstellungskosten. */
  costAmount: Cents;
  /** Verändert die kumulierten Abschreibungen. */
  depreciationAmount: Cents;
  journalEntryId?: number;
  entryNumber?: string;
  /** Stückzahl, die diese Bewegung bewegt: positiv beim Zugang, negativ beim Abgang. */
  quantity?: Units;
  /** Betrag, der nur steuerlich zählt — die Vorabpauschale wird nicht gebucht. */
  taxAmount?: Cents;
  /** Monate, um die diese Bewegung die Restnutzungsdauer verlängert. */
  lifeExtensionMonths?: number;
  note?: string;
  createdAt: string;
}

export interface FixedAsset {
  id: number;
  inventoryNumber: string;
  name: string;
  description?: string;
  class: AssetClass;
  account: string;
  depreciationAccount?: string;
  acquisitionDate: string;
  /** Tag der Betriebsbereitschaft; ab hier läuft die AfA. Leer = mit der Anschaffung. */
  inServiceDate?: string;
  acquisitionCost: Cents;
  method: DepreciationMethod;
  usefulLifeMonths: number;
  /**
   * Die Begründung einer Nutzungsdauer, die vom Vorschlag des Kontos abweicht
   * — etwa bei EDV-Hardware, für die das BMF-Schreiben vom 22.02.2022 zwölf
   * Monate zulässt.
   */
  usefulLifeReason?: string;
  /**
   * Vorsteuer der Anschaffung und der Anteil, mit dem sie gezogen wurde. Beide
   * gehen ins Verzeichnis nach § 15a UStG ein.
   */
  inputTaxAmount?: Cents;
  inputTaxPermille?: number;
  /** Rein elektrisch betriebenes Fahrzeug — Voraussetzung des § 7 Abs. 2a EStG. */
  isElectric?: boolean;
  /**
   * Der Stichtag, an dem § 7 Abs. 4 EStG den Gebäudesatz festmacht: Bauantrag
   * bzw. Fertigstellung.
   */
  buildingReferenceDate?: string;
  /**
   * Das Geschäftsjahr, für das bestätigt wurde, dass der Grund einer
   * außerplanmäßigen Abschreibung fortbesteht, samt der Begründung.
   */
  impairmentPersistsYear?: number;
  impairmentPersistsNote?: string;
  poolYear?: number;
  /** Sonderabschreibung nach § 7g Abs. 5 EStG: Satz in Promille, höchstens 400. */
  specialPermille?: number;
  /** Jahre, auf die der Betrag gleichmäßig verteilt wird — eins bis fünf. */
  specialYears?: number;
  /** Aufwandskonto der Sonderabschreibung: 6242 für Fahrzeuge, sonst 6241. */
  specialAccount?: string;
  /** Pflichtangabe zu den Voraussetzungen des § 7g Abs. 6 EStG. */
  specialReason?: string;
  identifier?: string;
  /** Stückzahl des Zugangs. Null heißt: dieses Anlagegut wird nicht in Stück geführt. */
  quantity?: Units;
  /** Notierungswährung (ISO 4217). Leer heißt Euro. */
  currency?: string;
  /** Anschaffungskosten in der Notierungswährung. */
  foreignCost?: Cents;
  /** Fälligkeit einer Ausleihung. Entscheidet über § 256a Satz 2 HGB. */
  maturityDate?: string;
  /** Fondsart eines Investmentanteils. Leer heißt: kein Investmentanteil. */
  fundClass?: FundClass;
  /** Beteiligungsquote in Promille: 200 sind 20 %. */
  holdingPermille?: number;
  taxPrivileged?: boolean;
  contactId?: number;
  acquisitionEntryId?: number;
  disposalDate?: string;
  disposalKind?: DisposalKind;
  disposalProceeds?: Cents;
  disposalEntryId?: number;
  notes?: string;
  createdAt: string;
  updatedAt: string;
  movements?: AssetMovement[];

  // Abgeleitet vom Backend, nicht gespeichert.
  accountName?: string;
  cost: Cents;
  accumulated: Cents;
  bookValue: Cents;
  yearAmount: Cents;
  dueAmount: Cents;
  /** Noch fällige Sonderabschreibung des Geschäftsjahres. */
  specialDue: Cents;
  /** Gehaltene Stückzahl nach allen Bewegungen. */
  unitsHeld?: Units;
  /** Summe der über die Besitzzeit angesetzten Vorabpauschalen. */
  vorabpauschalen?: Cents;
  status: AssetStatus;
  statusNote?: string;
}

export interface AssetScheduleYear {
  fiscalYear: number;
  months: number;
  method: DepreciationMethod;
  rateLabel: string;
  openingBookValue: Cents;
  amount: Cents;
  /** Sonderabschreibung des Jahres, getrennt geführt: eigenes Aufwandskonto. */
  specialAmount?: Cents;
  closingBookValue: Cents;
  note?: string;
  booked: Cents;
  due: Cents;
  specialBooked: Cents;
  specialDue: Cents;
  status: 'gebucht' | 'offen' | 'teilweise' | 'geplant';
}

export interface AssetDetail {
  asset: FixedAsset;
  schedule: AssetScheduleYear[];
  movements: AssetMovement[];
  /** Höchstbetrag einer Zuschreibung (§ 253 Abs. 5 Satz 1 HGB), vom Backend gerechnet. */
  writeUpCeiling: Cents;
  /** Die Sätze, die zu diesem Anlagegut gehören — vom Backend gerechnet. */
  notes: string[];
}

/** Die Vorabpauschale eines Kalenderjahres (§ 18 InvStG), mit jedem Schritt. */
export interface Vorabpauschale {
  year: number;
  basisReturn: Cents;
  growth: Cents;
  capped: boolean;
  distributions: Cents;
  monthsCounted: number;
  amount: Cents;
  accruedOn: string;
  explanation: string;
}

export interface AcquisitionCandidate {
  entryId: number;
  entryNumber: string;
  bookingDate: string;
  description: string;
  account: string;
  accountName: string;
  amount: Cents;
  contactId?: number;
  /** Vorsteuer der Zugangsbuchung; 0 heißt: die Buchung gibt sie nicht eindeutig her. */
  inputTaxAmount: Cents;
  /** Anteil, mit dem die Vorsteuer gezogen wurde (Promille). */
  inputTaxPermille: number;
}

export interface AnlagenspiegelRow {
  class: AssetClass;
  account: string;
  accountName: string;
  assetCount: number;
  costOpening: Cents;
  additions: Cents;
  disposals: Cents;
  transfers: Cents;
  costClosing: Cents;
  depreciationOpening: Cents;
  depreciationYear: Cents;
  writeUpsYear: Cents;
  depreciationDisposal: Cents;
  depreciationTransfer: Cents;
  depreciationClosing: Cents;
  bookValueOpening: Cents;
  bookValueClosing: Cents;
}

export interface Anlagenspiegel {
  fiscalYear: number;
  rows: AnlagenspiegelRow[];
  totals: AnlagenspiegelRow;
  classTotals: AnlagenspiegelRow[];
}

// -------------------------------------------------------------
// Gründung: von der Beurkundung bis zur Eintragung
// -------------------------------------------------------------

/** Die Phase der Gründung. Sie folgt aus dem Eintragungsdatum, nicht umgekehrt. */
export type FoundationStage = 'vorgesellschaft' | 'eingetragen';

/** Bar- oder Sacheinlage. Der Unterschied entscheidet über die Mindesteinzahlung. */
export type ContributionKind = 'cash' | 'kind';

/** Ein Gesellschafter und der Geschäftsanteil, den er übernommen hat. */
export interface Shareholder {
  id: number;
  foundationId: number;
  name: string;
  /** Nennbetrag des übernommenen Geschäftsanteils. */
  shareCapital: Cents;
  /** Was darauf tatsächlich geleistet wurde. */
  paidIn: Cents;
  kind: ContributionKind;
}

/** Die Gründung. Den Tag der Beurkundung führen die Stammdaten als `foundedOn`. */
export interface Foundation {
  id: number;
  /** Tag der Eintragung. Leer heißt: noch Vorgesellschaft. */
  registeredOn: string;
  registerCourt: string;
  registerNumber: string;
  shareCapital: Cents;
  /**
   * Gründungsaufwand laut Gesellschaftsvertrag. Er begrenzt die
   * Vorbelastungshaftung: was die Satzung der Gesellschaft auferlegt, ist
   * zulässig von ihr getragen.
   */
  foundationCostCap: Cents;
  shareholders: Shareholder[];
}

/** Was eine Rechtsform vor der Anmeldung verlangt. */
export interface FoundationRules {
  legalForm: string;
  minShareCapital: Cents;
  paidInPerShareQuota: number;
  paidInFloor: Cents;
  paidInFloorIsFullCapital: boolean;
  cashOnly: boolean;
  legalReserve: boolean;
  reference: string;
  note: string;
}

/** Der Anteil eines Gesellschafters an der Unterbilanzhaftung. */
export interface UnterbilanzShare {
  shareholderId: number;
  name: string;
  shareCapital: Cents;
  amount: Cents;
}

/**
 * Die Vorbelastungsrechnung: was die Gesellschafter der Gesellschaft schulden,
 * weil ihr Reinvermögen hinter dem Stammkapital zurückbleibt.
 */
export interface Unterbilanz {
  asOf: string;
  /** Erst ab der Eintragung steht die Zahl endgültig fest. */
  isFinal: boolean;
  shareCapital: Cents;
  assets: Cents;
  liabilities: Cents;
  netAssets: Cents;
  /** Rohe Unterdeckung, davon durch die Satzungsklausel gedeckt, und der Rest. */
  shortfall: Cents;
  covered: Cents;
  amount: Cents;
  shares: UnterbilanzShare[];
}

export interface ShareholderCheck {
  shareholderId: number;
  name: string;
  kind: ContributionKind;
  shareCapital: Cents;
  requiredPaidIn: Cents;
  paidIn: Cents;
  isSatisfied: boolean;
}

/** Reicht die geleistete Einlage für die Anmeldung zum Handelsregister? */
export interface AnmeldungCheck {
  legalForm: string;
  minShareCapital: Cents;
  shareCapital: Cents;
  requiredPaidIn: Cents;
  actualPaidIn: Cents;
  isSatisfied: boolean;
  /** Was fehlt, je Befund ein Satz mit Fundstelle. Leer heißt: es passt. */
  findings: string[];
  reference: string;
  shareholders: ShareholderCheck[];
}

/** Alles, was die Gründungsansicht braucht — in einem Aufruf. */
export interface FoundationState {
  hasFoundation: boolean;
  legalForm: string;
  rules: FoundationRules;
  foundation?: Foundation;
  stage: FoundationStage;
  anmeldung?: AnmeldungCheck;
  unterbilanz?: Unterbilanz;
  postingsBooked: boolean;
}

// -------------------------------------------------------------
// Jahresabschluss: Geschäftsjahr, Saldenvortrag, Abschlussstand
// (internal/domain/fiscalyear.go, internal/service/closing_service.go)
// -------------------------------------------------------------

/**
 * Die vier Stände sind Vorgänge mit verschiedenen Beteiligten: Aufstellung
 * durch die Geschäftsführung (§ 242, § 264 Abs. 1 HGB), Feststellung durch
 * die Gesellschafter (§ 42a Abs. 2 GmbHG), Offenlegung gegenüber dem
 * Bundesanzeiger (§ 325 HGB).
 */
export type FiscalYearStatus = 'open' | 'prepared' | 'adopted' | 'disclosed';

export interface FiscalYearCandidate {
  year: number;
  startDate: string;
  endDate: string;
  /** Grund, aus dem sich das Jahr nicht anlegen lässt; fehlt, wenn es anlegbar ist. */
  blocked?: string;
}

export interface FiscalYearCandidates {
  next: FiscalYearCandidate;
  previous: FiscalYearCandidate;
}

/** Das Geschäftsjahr als Entität: Zeitraum, Rumpfjahr, Abschlussstand. */
export interface FiscalYear {
  year: number;
  startDate: string;
  endDate: string;
  /** Rumpfgeschäftsjahr (§ 8b EStDV): kürzer als zwölf Monate. */
  isShort: boolean;
  status: FiscalYearStatus;
  preparedOn?: string;
  adoptedOn?: string;
  disclosedOn?: string;
  /** Welcher Gesellschafterbeschluss den Abschluss festgestellt hat. */
  adoptionNote?: string;
  disclosureNote?: string;
  /** Zeitpunkt des letzten Saldenvortrags in dieses Jahr. */
  carriedForwardAt?: string;
  /**
   * Die durchschnittliche Zahl der Arbeitnehmer ist das dritte Merkmal des
   * § 267 Abs. 1 HGB. Aus der Buchführung lässt sie sich nicht ableiten,
   * deshalb wird sie erfasst und nicht gerechnet.
   */
  averageEmployees: number;
  /**
   * Der Gesamtumsatz des vorangegangenen Kalenderjahres. Die Übergangsfrist
   * des § 27 Abs. 38 Nr. 2 UStG richtet sich nach ihm: bis 800.000 € darf 2027
   * noch eine sonstige Rechnung ausgestellt werden. Null heißt „nicht erfasst".
   */
  priorYearRevenue: Cents;
  createdAt: string;
}

/** Alles, was die Abschlussansicht eines Jahres braucht — in einem Aufruf. */
export interface ClosingState {
  legalReserve?: { applies: boolean; year: number; date: string; netIncome: number; lossCarryForward: number; required: number; booked: number; difference: number };
  year: number;
  fiscalYear: FiscalYear;
  /** Erträge minus Aufwendungen der GuV-Konten; abgeleitet, nicht gebucht. */
  netIncome: Cents;
  /** Ohne Jahres-Festschreibung lässt sich der Abschluss nicht feststellen. */
  hasYearCommitment: boolean;
  committedUntil?: string;
  nextYear: number;
  carriedForward: boolean;
  /** Falsch, sobald im abgelaufenen Jahr nach dem Vortrag noch gebucht wurde. */
  carryForwardCurrent: boolean;
  carriedForwardAt?: string;
  /** Leer, wenn das Jahr offengelegt ist. */
  nextStatus?: FiscalYearStatus;
  canAdopt: boolean;
  blocker?: string;
}

/** Vortragsart: je eine Buchung gegen 9000, 9008 und 9009. */
export type CarryForwardKind = 'sachkonto' | 'debitor' | 'kreditor';

/**
 * Eine Zeile der Vortragsvorschau. Alle Beträge sind vorzeichenbehaftet in
 * Soll-Richtung: positiv ist ein Sollsaldo, negativ ein Habensaldo.
 */
export interface CarryForwardRow {
  account: string;
  name: string;
  kind: CarryForwardKind;
  closingBalance: Cents;
  carried: Cents;
  difference: Cents;
  /** Zahl der offenen Posten hinter einem Personenkonto. */
  openItems?: number;
  includesNetIncome?: boolean;
}

/** Der Stand des Saldenvortrags in ein Geschäftsjahr. */
export interface CarryForwardPreview {
  resultToCarry: Cents;
  fromYear: number;
  toYear: number;
  /** Erster Tag des neuen Jahres, sonst der erste nicht festgeschriebene Tag. */
  bookingDate: string;
  deferred: boolean;
  rows: CarryForwardRow[];
  netIncome: Cents;
  resultAccount: string;
  resultAccountName: string;
  alreadyCarried: boolean;
  needsCorrection: boolean;
  /** Vortragswerte ohne zurücknehmbare Buchung: ein Lauf würde sie verdoppeln. */
  irreversible?: boolean;
  /** Das Vorjahr selbst hat keinen Saldenvortrag, obwohl es einen bräuchte. */
  priorYearNotCarried?: boolean;
  /** Zahl der Buchungen, die ein Lauf erzeugt; höchstens drei. */
  entries: number;
  /** Probe auf die Bilanzidentität: Summe aller Vortragswerte, muss null sein. */
  balanceDifference: Cents;
  isBalanced: boolean;
  /**
   * Die Auflösungen der Rechnungsabgrenzung, die der Vortrag im neuen Jahr
   * gleich mitbucht (§ 250 HGB). Sie gehören in die Vorschau: sonst gäbe der
   * Anwender Buchungen frei, die ihm niemand genannt hat.
   */
  accrualReleases: AccrualReleaseDue[];
}

// -------------------------------------------------------------
// Jahresabschluss: Bilanz und Gewinn- und Verlustrechnung
// (internal/domain/statement.go, internal/domain/sizeclass.go,
//  internal/accounting/statement_mapping.go, internal/ebilanz/assign.go)
//
// Die Gliederung nach den §§ 266 und 275 HGB entsteht vollständig im Backend.
// Hier stehen nur die Formen, in denen sie ankommt: die Ansicht zeigt Zeilen
// an und rechnet an keiner Stelle nach — sonst gäbe es zwei Bilanzen.
// -------------------------------------------------------------

/**
 * Die Gliederungstiefe ist eine Rechtsfolge der Größenklasse und kein
 * Anzeigegeschmack: § 266 Abs. 1 Satz 3 HGB erlaubt der kleinen Gesellschaft
 * Buchstaben und römische Ziffern, Satz 4 der Kleinstgesellschaft allein die
 * Buchstaben.
 */
export type StatementDepth = 'full' | 'short' | 'letters';

/** Abschnitt, in dem eine Gliederungsposition steht. */
export type StatementSection = 'aktiva' | 'passiva' | 'guv' | 'statistisch';

/** Ein Konto unter einer Gliederungsposition — der Weg zurück zum Kontoblatt. */
export interface StatementAccount {
  number: string;
  name: string;
  positionId: string;
  /** Bezeichnung der SKR04-Position, nicht die der Gliederungszeile. */
  position: string;
  /** Erklärt eine Zuordnung, die dem Kontonamen widerspricht. */
  note?: string;
  amount: Cents;
  priorAmount: Cents;
}

/** Eine Gliederungsposition mit ihrem Wert. */
export interface StatementLine {
  /** Stabiler Schlüssel, z. B. "aktiva.A.II.3". */
  key: string;
  /** Ordnungszahl des Gesetzes: "A.", "II.", "3.", "a)". */
  ordinal: string;
  label: string;
  /** 1 Buchstabe, 2 römische Ziffer, 3 arabische Ziffer. */
  level: number;
  section: StatementSection;
  note?: string;
  /** Zwischensumme der Staffel (§ 275 Abs. 2 Nr. 15 und 17 HGB). */
  isSubtotal: boolean;
  /** Auffangposition ("sonstige …"). */
  isFallback: boolean;
  /**
   * Posten ohne Betrag in beiden Jahren; § 265 Abs. 8 HGB lässt ihn entfallen.
   * Die Entscheidung fällt im Backend, damit Ansicht, PDF und CSV dieselben
   * Zeilen zeigen.
   */
  omitted: boolean;
  amount: Cents;
  priorAmount: Cents;
  accounts?: StatementAccount[];
  /** Davon-Vermerke für Posten, die die Gliederungstiefe hier zusammenfasst (§ 42 Abs. 3 GmbHG). */
  remarks?: StatementRemark[];
}

/** Ein Davon-Vermerk unter einer Gliederungsposition. */
export interface StatementRemark {
  key: string;
  label: string;
  amount: Cents;
  priorAmount: Cents;
  reference?: string;
}

/** Was in einer Auffangposition gelandet ist. */
export interface FallbackCount {
  key: string;
  label: string;
  accounts: number;
  amount: Cents;
}

/** Ein Konto, das wegen seines Vorzeichens auf der Gegenposition steht. */
export interface SignSwitch {
  account: string;
  name: string;
  from: string;
  to: string;
  label: string;
  amount: Cents;
}

/** Was die Gliederung nicht oder nur mit Vorbehalt einordnen konnte. */
export interface AssignmentReport {
  /** Konten mit Saldo ohne Gliederungsposition; sie stehen in "Nicht zugeordnet". */
  unassigned: StatementAccount[];
  /** Saldo gegen die Richtung der Position, ohne Gegenposition. */
  wrongSign: StatementAccount[];
  signSwitches: SignSwitch[];
  fallbacks: FallbackCount[];
}

/** Die Gliederung eines Geschäftsjahres mit Vorjahresspalte. */
export interface Statement {
  fiscalYear: number;
  priorYear: number;
  hasPrior: boolean;
  depth: StatementDepth;
  assets: StatementLine[];
  liabilities: StatementLine[];
  income: StatementLine[];
  /** Konten der Klasse 9 — weder Bilanz noch GuV, aber sichtbar. */
  statistical: StatementLine[];
  assignment: AssignmentReport;

  totalAssets: Cents;
  totalAssetsPrior: Cents;
  totalLiabilities: Cents;
  totalLiabilitiesPrior: Cents;
  /** Bilanzsumme des § 267 Abs. 4a HGB: Posten A bis E der Aktivseite. */
  balanceSheetTotal: Cents;
  balanceSheetTotalPrior: Cents;

  netIncome: Cents;
  netIncomePrior: Cents;
  /** Nummer 1 der Staffel — das Merkmal "Umsatzerlöse" des § 267 HGB. */
  revenue: Cents;
  revenuePrior: Cents;
}

/** Eine Zeile der Restlaufzeitengliederung. */
export interface MaturityRow {
  key: string;
  label: string;
  total: Cents;
  upToOneYear: Cents;
  overOneYear: Cents;
  overFiveYears: Cents;
  items: number;
  /** Posten ohne Fälligkeit: ohne sie gibt es keine Restlaufzeit. */
  undated: Cents;
  note?: string;
}

/** Angabe unter der Bilanz nach § 268 Abs. 4 und 5 HGB. */
export interface MaturityTable {
  closingDate: string;
  rows: MaturityRow[];
  reference: string;
}

/** Ein Termin des Jahresabschlusses mit seiner Norm. */
export interface Deadline {
  key: string;
  title: string;
  dueDate: string;
  period: string;
  reference: string;
  description: string;
  fiscalYear: number;
  isDone: boolean;
  doneOn?: string;
  /** Stand im Pflichtenkalender: open, prepared, done, overdue, waived. */
  status?: string;
  /** Der Mensch kann die Pflicht in der App bestätigen. */
  confirmable?: boolean;
}

/** Die Pflichtangaben des § 264 Abs. 1a HGB im Kopf des Abschlusses. */
export interface StatementHeader {
  /** Der erfasste Name. */
  companyName: string;
  /**
   * Die Firma, unter der das Unternehmen auftritt: um die Rechtsform ergänzt,
   * wo sie im Namen fehlt, und bis zur Eintragung um „i. G.".
   */
  firmName: string;
  legalForm: string;
  seat: string;
  registerCourt: string;
  registerNumber: string;
  fiscalYear: number;
  startDate: string;
  closingDate: string;
  priorYear: number;
  isShortYear: boolean;
  reference: string;
  /** Pflichtangaben, die in den Einstellungen fehlen. */
  missing: string[];
}

/** Größenklasse nach den §§ 267, 267a HGB. */
export type SizeClassKind = 'micro' | 'small' | 'medium' | 'large';

/** Die drei Merkmale des § 267 Abs. 1 HGB zu einem Stichtag. */
export interface SizeCriteria {
  balanceSheetTotal: Cents;
  revenue: Cents;
  employees: number;
}

/** Die Schwellenwerte einer Fassung, datiert nach dem Beginn des Jahres. */
export interface SizeThresholdSet {
  validFrom: string;
  reference: string;
  micro: SizeCriteria;
  small: SizeCriteria;
  medium: SizeCriteria;
}

/** Die Beurteilung eines einzelnen Abschlussstichtags (§ 267 Abs. 4 HGB). */
export interface SizeAssessment {
  year: number;
  closingDate: string;
  criteria: SizeCriteria;
  class: SizeClassKind;
  /** Die Merkmale, die für die Klasse sprechen — zwei von drei genügen. */
  met: string[];
  thresholds: SizeThresholdSet;
}

/** Die Folgen der Größenklasse, je mit ihrer Norm. */
export interface SizeObligations {
  depth: StatementDepth;
  depthReference: string;
  notesRequired: boolean;
  notesReference: string;
  managementReport: boolean;
  managementReportReference: string;
  auditRequired: boolean;
  auditReference: string;
  preparationMonths: number;
  preparationReference: string;
  disclosureMonths: number;
  disclosureReference: string;
  disclosureScope: string;
  disclosureScopeReference: string;
}

/** Die Einordnung eines Geschäftsjahres samt Begründung. */
export interface SizeClass {
  year: number;
  closingDate: string;
  class: SizeClassKind;
  criteria: SizeCriteria;
  current: SizeAssessment;
  prior?: SizeAssessment;
  /** Die Stichtage, die in die Zweijahresregel eingegangen sind. */
  history?: SizeAssessment[];
  /** § 267 Abs. 4 Satz 2 HGB: bei Neugründung gilt schon der erste Stichtag. */
  isFirstYear: boolean;
  reason: string;
  obligations: SizeObligations;
}

/** Der Jahresabschluss, wie ihn die Ansicht zeigt. */
export interface FinancialStatement {
  header: StatementHeader;
  statement: Statement;
  sizeClass: SizeClass;
  maturities: MaturityTable;
  /** Der Anhang gehört zum Abschluss und entsteht mit ihm, nicht daneben. */
  notes: StatementNotes;
  deadlines: Deadline[];
}

/** Ein Konto mit Saldo, seine Gliederungsposition und sein Taxonomie-Element. */
export interface MappingRow {
  account: string;
  name: string;
  balance: Cents;
  positionKey: string;
  positionLabel: string;
  element: string;
  /** Auffangposition der Gliederungsposition statt einer Kontenregel. */
  fallback: boolean;
  /** Benennt, was fehlt, wenn etwas fehlt. */
  finding?: string;
}

/** Ein Befund der Prüfung vor dem E-Bilanz-Export. */
export interface EBilanzFinding {
  severity: 'blocking' | 'warning';
  code: string;
  message: string;
  element?: string;
  accounts?: string[];
}

/** Der Zuordnungsbericht vor dem E-Bilanz-Export. */
/** Eine Abweichung der Überleitungsrechnung, steuerlich minus handelsrechtlich. */
export interface TransferRow {
  reason: string;
  account: string;
  element: string;
  current: number;
  prior: number;
}

export interface MappingReport {
  fiscalYear: number;
  taxonomyVersion: string;
  taxonomyDate: string;
  taxonomyNote: string;
  /** HAOE: Einheitsbilanz; HGBM: Handelsbilanz mit Überleitungsrechnung. */
  accountingStandard: string;
  rows: MappingRow[];
  /**
   * Abweichungen der Steuerbilanz bei HGBM. Die XBRL-Instanz enthält sie als
   * Überleitungsrechnung, die Saldenliste nicht.
   */
  transfers: TransferRow[];
  /** Blockierende Befunde, je Zeile mit den betroffenen Konten. */
  blocking: MappingRow[];
  fallbacks: FallbackCount[];
  findings: EBilanzFinding[];
  canExport: boolean;
}

// -------------------------------------------------------------------------
// Umsatzsteuer-Voranmeldung, Zusammenfassende Meldung, Prüfläufe
// -------------------------------------------------------------------------

/** Länge eines Voranmeldungszeitraums (§ 18 Abs. 2 UStG). */
export type VatPeriodType = 'month' | 'quarter' | 'year' | 'unknown' | 'none';

/**
 * Zwei Stände, mehr gibt es nicht: Buchfink übermittelt nicht selbst, und ein
 * „übermittelt, aber ohne Ticket" wäre eine Behauptung ohne Nachweis.
 */
export type VatReturnStatus = 'draft' | 'submitted';

/** Ein Zeitraum, wie ihn das Backend benennt: „März 2026", „2026-Q1". */
export interface VatPeriod {
  key: string;
  type: VatPeriodType;
  label: string;
  from: string;
  to: string;
  year: number;
}

/**
 * Eine Zeile des Vordrucks USt 1 A. `hasBase` und `hasTax` sagen, welche Felder
 * der Vordruck in dieser Zeile kennt; `taxCode` benennt die zweite Kennziffer,
 * wenn der Steuerbetrag unter einer eigenen steht (35/36, 46/47).
 */
export interface VatReturnLine {
  code: string;
  label: string;
  reference?: string;
  hasBase: boolean;
  base: Cents;
  hasTax: boolean;
  taxCode?: string;
  tax: Cents;
  /** Die aus der Bemessungsgrundlage errechnete Steuer — zum Vergleich. */
  expectedTax: Cents;
  /** Die Buchungen hinter der Kennziffer (Drill-down). */
  entryIds?: number[];
}

/** Eine Buchung, deren Voranmeldungszeitraum bereits übermittelt ist. */
export interface VatLateEntry {
  entryId: number;
  entryNumber: string;
  bookingDate: string;
  /** Der Zeitraum, in den die Buchung gehört. */
  periodKey: string;
  description: string;
  code: string;
  base: Cents;
  tax: Cents;
}

/** Die Umsatzsteuer-Voranmeldung eines Zeitraums, mit Übermittlungsprotokoll. */
export interface VatReturn {
  id: number;
  fiscalYear: number;
  periodType: VatPeriodType;
  periodKey: string;
  periodFrom: string;
  periodTo: string;
  /** Setzt die Kennziffer 10 des Vordrucks: berichtigte Anmeldung. */
  isCorrection: boolean;
  correctsId?: number;
  status: VatReturnStatus;
  submittedAt?: string;
  transferTicket?: string;
  submissionNote?: string;
  /** Kennziffer 83; negativ heißt Überschuss zugunsten des Unternehmers. */
  payable: Cents;
  dueDate?: string;
  programVersion?: string;
  figures: VatReturnLine[];
  lateEntries: VatLateEntry[];
  createdAt: string;
}

/** S: sonstige Leistung nach § 3a Abs. 2 UStG — die einzige Meldeart in Buchfink. */
export type ZMLineKind = 'S';

/** Eine Meldezeile: je USt-IdNr. ein Betrag. */
export interface ZMLine {
  id: number;
  zmReturnId: number;
  countryCode: string;
  vatId: string;
  kind: ZMLineKind;
  amount: Cents;
  contactId: number;
  contactName?: string;
  entryIds?: number[];
}

/** Ein meldepflichtiger Umsatz, dessen Meldezeitraum übermittelt ist. */
export interface ZMLateEntry {
  entryId: number;
  entryNumber: string;
  periodKey: string;
  date: string;
  vatId?: string;
  amount: Cents;
}

/**
 * Die Abstimmung gegen die Kennziffer 21 der Voranmeldungen. Sie wird nicht
 * gespeichert: sie beschreibt den heutigen Stand beider Meldungen.
 */
export interface ZMReconciliation {
  scopeKey?: string;
  scopeLabel?: string;
  servicesZm: Cents;
  servicesVat: Cents;
  vatReturnsFound: number;
}

/** Die Zusammenfassende Meldung eines Meldezeitraums (§ 18a UStG). */
export interface ZMReturn {
  id: number;
  fiscalYear: number;
  periodType: VatPeriodType;
  periodKey: string;
  periodFrom: string;
  periodTo: string;
  isCorrection: boolean;
  correctsId?: number;
  status: VatReturnStatus;
  submittedAt?: string;
  transferTicket?: string;
  submissionNote?: string;
  dueDate?: string;
  totalServices: Cents;
  lines: ZMLine[];
  reconciliation?: ZMReconciliation;
  /** Was die Bestätigung verhindert — allen voran eine fehlende USt-IdNr. */
  findings?: string[];
  lateEntries?: ZMLateEntry[];
  createdAt: string;
}

/** Ein Meldezeitraum mit Fälligkeit und Stand. */
export interface ZMPeriodStatus extends VatPeriod {
  dueDate: string;
  status: VatReturnStatus;
  returnId?: number;
  /** Ohne Festschreibung ist die Bestätigung gesperrt — wie bei der Voranmeldung. */
  committed: boolean;
  total: Cents;
  submittedAt?: string;
  isOverdue: boolean;
}

/**
 * Das Gewicht eines Befundes. `blocking` verhindert die Festschreibung,
 * `warning` nicht — eine dritte Stufe stünde nur in der Liste herum.
 */
export type CheckSeverity = 'blocking' | 'warning';

/** Ein einzelner Befund eines Prüflaufs. */
export interface CheckFinding {
  id: number;
  checkRunId: number;
  rule: string;
  severity: CheckSeverity;
  /** Bezugsobjekt, damit die Ansicht einen Weg dorthin anbieten kann. */
  objectType?: string;
  objectId?: string;
  objectName?: string;
  message: string;
  reference?: string;
}

/**
 * Die Abstände Belegdatum → Erfassung → Festschreibung als Kennzahlen eines
 * Prüflaufs (Entscheidung 4, GoBD Rz. 47). Spiegelt `domain.CheckTimeliness`.
 *
 * Median und nicht Mittelwert: ein einzelner nachgetragener Altbeleg zöge den
 * Mittelwert so weit hoch, dass die Kennzahl über den Regelfall nichts mehr
 * sagt; das Maximum steht daneben, damit der Ausreißer sichtbar bleibt.
 */
export interface CheckTimeliness {
  /** Buchungen, deren Erfassungsabstand gemessen werden konnte. */
  measuredEntries: number;
  captureDaysMedian: number;
  captureDaysMax: number;
  /** Die Erfassungsfrist, gegen die gemessen wurde (Vorgabe zehn Tage). */
  captureLimitDays: number;
  lateEntries: number;
  committedEntries: number;
  commitDaysMedian: number;
  commitDaysMax: number;
  uncommittedEntries: number;
}

/** Ein Prüflauf über einen Zeitraum bis zu einem Stichtag (GoBD Rz. 34 ff.). */
export interface CheckRun {
  id: number;
  fiscalYear: number;
  cutoffDate: string;
  periodType?: string;
  checkedEntries: number;
  checkedReceipts: number;
  checkedBankTx: number;
  /**
   * Das Backend liefert die Kennzahlen; die Ansicht erlaubt auch gespeicherte
   * Prüfläufe ohne Kennzahlen.
   */
  timeliness?: CheckTimeliness;
  /** Die Begründung, mit der blockierende Befunde übergangen wurden. */
  overrideReason?: string;
  findings: CheckFinding[];
  createdAt: string;
}

// -------------------------------------------------------------------------
// Datenüberlassung, Sicherung und Prüfermodus

/** Der Umfang eines Exports. Spiegelt `export.Kind` (internal/export). */
export type ExportKind = 'z3' | 'archive' | 'audit_package' | 'journal' | 'key_directory';

export interface ExportTableInfo {
  name: string;
  file: string;
  rows: number;
}

/** Eine erzeugte Datei mit ihrer Prüfsumme — ein Datenträger kann unterwegs
 *  beschädigt werden, und der Empfänger soll das bemerken können. */
export interface ExportFileInfo {
  path: string;
  sha256: string;
  bytes: number;
}

export interface ExportResult {
  kind: ExportKind;
  dir: string;
  tenantName: string;
  fiscalYear: number;
  from?: string;
  to?: string;
  createdAt: string;
  programVersion: string;
  /** Die Fassung des Beschreibungsstandards, nach dem index.xml aufgebaut ist. */
  standardVersion: string;
  tables: ExportTableInfo[];
  files: ExportFileInfo[];
  /** Mitgegebene Belegdateien. */
  receiptFiles: number;
  /** Hinweise, die der Export nicht selbst beheben kann. */
  notes: string[];
}

export interface FileCheckIssue {
  receiptNumber?: string;
  fileName: string;
  path: string;
  /** `missing` heißt: Datei fehlt. `damaged` heißt: Prüfsumme stimmt nicht. */
  reason: string;
  message: string;
}

/**
 * Der Belegprüflauf über alle Dateien. Die Hash-Chain sichert die Buchungen,
 * nicht die Dateien: ob die Datei noch die gebuchte ist, sagt erst der
 * Vergleich mit ihrer Prüfsumme (GoBD Rz. 110).
 */
export interface FileCheckResult {
  checked: number;
  intact: number;
  damaged: number;
  missing: number;
  issues: FileCheckIssue[];
  isValid: boolean;
  message: string;
  checkedAt: string;
}

/** Der Anlass eines Sicherungslaufs. */
export type BackupKind = 'manual' | 'automatic' | 'verify' | 'restore';

export interface BackupRun {
  id: number;
  kind: BackupKind;
  startedAt: string;
  finishedAt: string;
  /** Pfad der erzeugten oder geprüften ZIP-Datei. */
  target: string;
  fileCount: number;
  bytes: number;
  success: boolean;
  message: string;
  programVersion?: string;
  createdAt: string;
}

// -------------------------------------------------------------------------
// Abschlussbausteine: Schritte, Rechnungsabgrenzung, Rückstellungen,
// Umsatzsteuer-Verrechnung, Steuerrückstellung, Ergebnisverwendung, Anhang
// -------------------------------------------------------------------------

/** Die Bausteine des Jahresabschlusses in ihrer fachlichen Reihenfolge. */
export type ClosingStepKey =
  | 'depreciation'
  /** Wertaufholung prüfen: ist der Grund einer Abwertung weggefallen (§ 253 Abs. 5 HGB)? */
  | 'write_up'
  /** Fremdwährungsbewertung zum Stichtag (§ 256a HGB). */
  | 'currency_valuation'
  /** Wertberichtigung auf Forderungen (§ 253 Abs. 4 HGB). */
  | 'receivable_allowance'
  /** Unfertige Leistungen zu Herstellungskosten (§ 255 Abs. 2 HGB). */
  | 'work_in_progress'
  | 'accruals'
  | 'provisions'
  | 'vat_settlement'
  /** Vorsteuerberichtigung nach § 15a UStG; der Betrag geht in Kennziffer 64. */
  | 'input_tax_correction'
  | 'tax_provision'
  | 'check_run'
  | 'statement'
  | 'adoption'
  | 'disclosure'
  | 'appropriation';

/** Übersprungen ist etwas anderes als offen: es ist eine Aussage mit Grund. */
export type ClosingStepState = 'open' | 'done' | 'skipped';

export interface ClosingStepView {
  key: ClosingStepKey;
  order: number;
  label: string;
  /** Was der Schritt tut und warum er an dieser Stelle steht. */
  hint: string;
  /** Der Zustand folgt aus den Daten, statt vom Anwender gesetzt zu werden. */
  automatic: boolean;
  /** Die Normen; nur im Detaildialog. */
  reference?: string;
  state: ClosingStepState;
  reason?: string;
  changedOn?: string;
  /** Woran der Zustand liegt: „3 Posten gebildet", „AfA für 2 Anlagegüter offen". */
  detail?: string;
}

export interface ClosingSteps {
  fiscalYear: number;
  cutoff: string;
  steps: ClosingStepView[];
  /** Weder erledigt noch übersprungen. */
  openCount: number;
  /** Erledigte Schritte — gerechnet im Backend, nicht in der Ansicht. */
  doneCount: number;
  /** Bewusst übergangene Schritte; sie zählen zum Fortschritt. */
  skippedCount: number;
  /** Die Zahl aller Bausteine („Schritt 4 von 11"). */
  total: number;
  /**
   * Ob eine Entscheidung des geführten Weges noch zurückzunehmen ist — ein
   * übersprungener Baustein also wieder offen werden darf. Nach der
   * Festschreibung und nach der Feststellung nicht mehr; dann sagt
   * `reopenBlocker`, warum.
   */
  reopenable: boolean;
  reopenBlocker?: string;
}

/** Die drei Fälle des § 250 HGB. */
export type AccrualKind = 'active' | 'passive' | 'disagio';

/** Verteilungsverfahren; gilt für den ganzen Mandanten (§ 252 Abs. 1 Nr. 6 HGB). */
export type AccrualMethod = 'monthly' | 'daily';

/** Auflösungstakt: eine Buchung je Geschäftsjahr oder eine je Monat. */
export type AccrualReleaseCycle = 'yearly' | 'monthly';

/**
 * Die Einstellungen, die die Abschlussbausteine steuern
 * (internal/service/closing_settings.go).
 *
 * Sie stehen getrennt von `CompanySettings`, weil sie die Buchführung
 * beschreiben: der Hebesatz gehört zur Gemeinde, die Abgrenzungsmethode zur
 * Art, wie abgegrenzt wird.
 */
export interface ClosingSettings {
  /** Hebesatz der Gemeinde in Prozent (400 = 400 %), § 16 GewStG. */
  tradeTaxRatePercent: number;
  accrualMethod: AccrualMethod;
  /** Vorschlagsschwelle in Cent; nur für den Vorschlag, nicht für die Pflicht. */
  accrualThreshold: Cents;
  accrualRelease: AccrualReleaseCycle;
}

/** Eine geplante oder gebuchte Auflösung eines Abgrenzungspostens. */
export interface AccrualRelease {
  id: number;
  accrualId: number;
  fiscalYear: number;
  date: string;
  amount: Cents;
  journalEntryId?: number;
}

/**
 * Eine im Zieljahr fällige Auflösung, wie der Saldenvortrag sie mitbucht
 * (`service.AccrualReleaseDue`). Sie steht in der Vortragsvorschau, weil der
 * Vortrag sonst mehr täte, als er ankündigt.
 */
export interface AccrualReleaseDue {
  accrualId: number;
  releaseId: number;
  kind: AccrualKind;
  text: string;
  /** Das Aufwands- oder Ertragskonto, auf das die Auflösung zurückfließt. */
  account: string;
  date: string;
  amount: Cents;
}

export interface Accrual {
  id: number;
  fiscalYear: number;
  kind: AccrualKind;
  /** Die Buchung, aus der der Posten entstanden ist. */
  sourceEntryId?: number;
  text: string;
  totalAmount: Cents;
  /** Der Teil nach dem Stichtag — der abgegrenzte Betrag. */
  deferredAmount: Cents;
  startDate: string;
  endDate: string;
  cutoffDate: string;
  account: string;
  method: AccrualMethod;
  /** Leer heißt: nur vorgeschlagen, nicht gebildet. */
  formationEntryId?: number;
  releases: AccrualRelease[];
  createdAt: string;
}

/** Der Rückstellungsgrund nach § 249 HGB. */
export type ProvisionKind =
  | 'uncertain_liability'
  | 'pending_loss'
  | 'deferred_maintenance'
  | 'warranty_without_obligation'
  | 'tax_income'
  | 'tax_trade'
  | 'closing_costs'
  | 'retention_costs'
  | 'personnel'
  | 'pension';

/** Die fünf Spalten des Rückstellungsspiegels. */
export type ProvisionMovementKind =
  | 'formation'
  | 'increase'
  | 'consumption'
  | 'release'
  | 'unwinding';

export interface ProvisionMovement {
  id: number;
  provisionId: number;
  kind: ProvisionMovementKind;
  date: string;
  fiscalYear: number;
  /** Immer positiv; die Richtung folgt aus der Art. */
  amount: Cents;
  /** Bei der Auflösung Pflicht (§ 249 Abs. 2 Satz 2 HGB). */
  reason?: string;
  journalEntryId?: number;
  entryNumber?: string;
  createdAt: string;
}

export interface Provision {
  id: number;
  fiscalYear: number;
  kind: ProvisionKind;
  text: string;
  /** Erfüllungsbetrag nach § 253 Abs. 1 Satz 2 HGB. */
  settlementAmount: Cents;
  expectedDate: string;
  /** Abgezinster Wert zum Stichtag; gleich dem Erfüllungsbetrag ohne Abzinsung. */
  discountedAmount: Cents;
  /** Der verwendete Satz in Millionsteln: 1,50 % sind 15000. */
  discountRateMicros?: number;
  balanceAccount: string;
  expenseAccount: string;
  reason: string;
  /** Gesetzt, sobald die Rückstellung erledigt ist. */
  settledOn?: string;
  movements: ProvisionMovement[];
  createdAt: string;
}

export interface ProvisionMirrorRow {
  kind: ProvisionKind;
  label: string;
  account: string;
  opening: Cents;
  additions: Cents;
  used: Cents;
  released: Cents;
  unwinding: Cents;
  closing: Cents;
}

/** Der Rückstellungsspiegel des Anhangs (§ 285 HGB). */
export interface ProvisionMirror {
  fiscalYear: number;
  rows: ProvisionMirrorRow[];
  total: ProvisionMirrorRow;
}

export interface VatSettlementRow {
  account: string;
  accountName: string;
  /** Saldo in Soll-Richtung: Vorsteuer positiv, Umsatzsteuer negativ. */
  balance: Cents;
}

/** Die Jahresverrechnung der Umsatzsteuer zum Bilanzstichtag. */
export interface VatSettlement {
  fiscalYear: number;
  cutoff: string;
  rows: VatSettlementRow[];
  inputTax: Cents;
  outputTax: Cents;
  prepaid: Cents;
  /** Zahllast auf 3841 oder Erstattung auf 1420 — immer nur eines von beiden. */
  payable: Cents;
  refund: Cents;
  lines: JournalLine[];
  bookingDate: string;
  explanation: string;
}

/** Was in die Steuerrückstellung eingeht. */
export interface TaxProvisionInput {
  profitBeforeTax: Cents;
  nonDeductible: Cents;
  /** Hebesatz der Gemeinde in Prozent: 400 sind 400 %. */
  tradeTaxRatePercent: number;
  prepaidCorporate: Cents;
  prepaidTrade: Cents;
  date: string;
}

/** Das Ergebnis der Rechnung, Schritt für Schritt. */
export interface TaxProvisionResult {
  taxableIncome: Cents;
  corporateTax: Cents;
  solidarity: Cents;
  /** Auf volle 100 Euro abgerundeter Gewerbeertrag (§ 11 Abs. 1 Satz 3 GewStG). */
  tradeIncome: Cents;
  tradeBase: Cents;
  tradeTax: Cents;
  incomeProvision: Cents;
  tradeProvision: Cents;
  /** Überzahlungen; sie werden ausgewiesen und nicht gebucht. */
  incomeRefund: Cents;
  tradeRefund: Cents;
  ratesUsed: string;
}

/**
 * Die Vorschau erbt die Felder der Rechnung: Go bettet `TaxProvisionResult`
 * ohne eigenen Namen ein, seine Felder stehen deshalb unmittelbar im JSON.
 */
export interface TaxProvisionPreview extends TaxProvisionResult {
  fiscalYear: number;
  cutoff: string;
  input: TaxProvisionInput;
  lines: JournalLine[];
  explanation: string;
  /** Sagt ausdrücklich, dass die Rechnung eine Schätzung ist. */
  warning: string;
}

/** Der Beschluss über die Ergebnisverwendung (§ 29 GmbHG). */
export interface Appropriation {
  year: number;
  decisionDate: string;
  text?: string;
  receiptId?: number;
  /** Das verwendbare Ergebnis, wie es auf dem Vortragskonto stand. */
  netIncome: Cents;
  legalReserve: Cents;
  otherReserves: Cents;
  distribution: Cents;
  withholdingTax: Cents;
  solidarityOnWithholding: Cents;
  /** Der Rest auf neue Rechnung; er erzeugt keine Buchung. */
  carryForward: Cents;
  journalEntryId?: number;
  createdAt: string;
}

export interface AppropriationRequest {
  decisionDate: string;
  text: string;
  legalReserve: Cents;
  otherReserves: Cents;
  distribution: Cents;
  receiptId?: number;
}

export interface AppropriationPreview {
  year: number;
  bookingYear: number;
  netIncome: Cents;
  appropriation: Appropriation;
  lines: JournalLine[];
  bookingDate: string;
  /** Der Jahresüberschuss des verwendeten Jahres, ohne frühere Vorträge. */
  yearResult: Cents;
  /** Bereits im Abschluss des Gewinnjahres gebuchte Pflichtzuführung. */
  reservedInClosing: Cents;
  requiredLegalReserve: Cents;
  explanation: string;
  warnings: string[];
}

/** Ein Abschnitt des Anhangs. */
export type NotesSection =
  | 'board_loans'
  | 'additional'
  | 'methods'
  | 'board'
  | 'subsequent'
  | 'commitments'
  | 'contingent'
  | 'investments'
  | 'appropriation';

export interface NotesSectionDefinition {
  section: NotesSection;
  label: string;
  hint: string;
  /** Die Vorschrift, aus der die Angabe folgt. */
  basis: string;
}

/** Ein Abschnitt mit seinem Freitext. */
export interface NotesSectionText extends NotesSectionDefinition {
  text: string;
}

/** Eine Position der Überleitungsrechnung von der Handels- zur Steuerbilanz. */
export interface ReconciliationRow {
  position: string;
  basis: string;
  commercial: Cents;
  tax: Cents;
  /** Steuerlich minus handelsrechtlich. */
  difference: Cents;
  explanation: string;
}

/** Die Überleitung Handelsbilanz → Steuerbilanz (§ 60 Abs. 2 EStDV). */
export interface Reconciliation {
  fiscalYear: number;
  cutoff: string;
  rows: ReconciliationRow[];
  /** Summe der Differenzen: um so viel weicht das steuerliche Eigenkapital ab. */
  equityEffect: Cents;
  note: string;
}

/** Der Anhang: Freitexte, Rückstellungsspiegel, Überleitung. */
export interface StatementNotes {
  belowBalance?: NotesSectionText[];
  missing?: string[];
  texts: NotesSectionText[];
  provisionMirror: ProvisionMirror;
  reconciliation: Reconciliation;
  reference: string;
}

/** Ein Jahr im Verzeichnis: handelsrechtliche und steuerliche Abschreibung. */
export interface TaxElectionYear {
  fiscalYear: number;
  commercial: Cents;
  tax: Cents;
  difference: Cents;
}

/** Ein Wirtschaftsgut mit steuerlichem Wahlrecht (§ 5 Abs. 1 Satz 2 EStG). */
export interface TaxElectionRow {
  assetId: number;
  inventoryNumber: string;
  name: string;
  acquisitionDate: string;
  cost: Cents;
  /** Die Vorschrift, auf die sich das Wahlrecht stützt. */
  provision: string;
  reason?: string;
  years: TaxElectionYear[];
  totalCommercial: Cents;
  totalTax: Cents;
  totalDifference: Cents;
  bookValue: Cents;
  taxBookValue: Cents;
}

export interface TaxElectionRegister {
  fiscalYear: number;
  rows: TaxElectionRow[];
  totalDifference: Cents;
  totalBookValue: Cents;
  totalTaxBookValue: Cents;
  note: string;
}

// -------------------------------------------------------------------------
// Steuerliche Nebenpflichten: Vorsteuerberichtigung § 15a UStG,
// Bestätigung der USt-IdNr., nicht abziehbare Betriebsausgaben, Fremdwährung,
// Anlagen
// -------------------------------------------------------------------------

/** Der bestätigte Verwendungsanteil eines Jahres samt seiner Berichtigung. */
export interface InputTaxUsage {
  correctionId: number;
  fiscalYear: number;
  /** Anteil der Verwendung für zum Vorsteuerabzug berechtigende Umsätze, in Promille. */
  permille: number;
  /** Ohne Bestätigung ist der Anteil ein Vorschlag und wird nicht gebucht. */
  confirmed: boolean;
  /** Berichtigungsbetrag mit Vorzeichen: positiv, wo Vorsteuer hinzukommt. */
  amount: Cents;
  reason?: string;
  entryId?: number;
  bookedOn?: string;
  updatedAt: string;
}

/** Ein Wirtschaftsgut im Verzeichnis nach § 15a UStG. */
export interface InputTaxCorrection {
  id: number;
  assetId?: number;
  receiptId?: number;
  entryId?: number;
  label: string;
  /** Das Konto entscheidet über die Beweglichkeit und damit über den Zeitraum. */
  account?: string;
  acquisitionDate: string;
  netAmount: Cents;
  inputTaxAmount: Cents;
  /** Der Anteil, mit dem die Vorsteuer beim Zugang gezogen wurde, in Promille. */
  originalPermille: number;
  /** Grundstück oder Gebäude: zehn Jahre statt fünf (§ 15a Abs. 1 UStG). */
  immovable: boolean;
  correctionPeriodYears: number;
  firstFiscalYear: number;
  lastFiscalYear: number;
  /** Gesetzt heißt: der Eintrag ist vorzeitig abgeschlossen. */
  closedReason?: string;
  closedOn?: string;
  note?: string;
  createdAt: string;
  updatedAt: string;
  usages: InputTaxUsage[];
}

/** Die Bewertung eines Jahres: ob zu berichtigen ist und mit welchem Betrag. */
export interface InputTaxCorrectionAssessment {
  amount: Cents;
  /** Ist null, nennt `reason` die Bagatellgrenze des § 44 UStDV. */
  required: boolean;
  /** Erst bei der Steuerberechnung für das Kalenderjahr (§ 44 Abs. 3 UStDV). */
  deferToAnnual: boolean;
  account?: string;
  reason: string;
}

export interface InputTaxCorrectionRow {
  correction: InputTaxCorrection;
  /** Fällt das Jahr in den Berichtigungszeitraum? */
  inPeriod: boolean;
  /** Der Anteil, mit dem gerechnet wurde — bestätigt oder als Vorschlag. */
  permille: number;
  confirmed: boolean;
  assessment: InputTaxCorrectionAssessment;
  booked: boolean;
  entryNumber?: string;
}

export interface InputTaxCorrectionYear {
  fiscalYear: number;
  bookingDate: string;
  rows: InputTaxCorrectionRow[];
  /** Summe der zu buchenden Berichtigungen mit Vorzeichen. */
  totalAmount: Cents;
  /** Solange ein Verwendungsanteil unbestätigt ist, wird nicht gebucht. */
  unconfirmed: number;
  note: string;
}

/**
 * Das Ergebnis einer Bestätigungsanfrage. „unavailable" ist ein fehlender
 * Befund, kein negatives Ergebnis — die beiden dürfen nicht dasselbe
 * bedeuten.
 */
export type VatIDCheckStatus = 'valid' | 'invalid' | 'unavailable';

/**
 * Die Rückmeldung zu einem Feld der qualifizierten Abfrage: A stimmt überein,
 * B stimmt nicht überein, C nicht angefragt, D vom Mitgliedstaat nicht
 * mitgeteilt.
 */
export type VatIDFieldResult = 'A' | 'B' | 'C' | 'D';

/** Eine Bestätigungsanfrage beim Bundeszentralamt für Steuern (§ 18e UStG). */
export interface VatIDCheck {
  id: number;
  contactId: number;
  /** Die geprüfte Nummer in der Schreibweise, in der sie abgefragt wurde. */
  vatId: string;
  /** Die eigene USt-IdNr.; ohne sie ist die Abfrage keine qualifizierte. */
  ownVatId?: string;
  /** Zeitpunkt der Abfrage als RFC3339; die Frist richtet sich danach. */
  checkedAt: string;
  status: VatIDCheckStatus;
  resultCode?: string;
  resultText?: string;
  /** Die Abfrage-Identifikationsnummer: der Beleg gegenüber der Finanzverwaltung. */
  requestId?: string;
  nameResult?: VatIDFieldResult;
  cityResult?: VatIDFieldResult;
  postalCodeResult?: VatIDFieldResult;
  streetResult?: VatIDFieldResult;
  /** Die Antwort, wie sie kam (GoBD Rz. 130). */
  rawResponse?: string;
  /** Wohin gefragt wurde. */
  endpoint?: string;
  createdAt: string;
}

/** Der Stand der Bestätigung, ohne dass dafür gefragt würde. */
export interface VatIDStatus {
  contactId: number;
  vatId: string;
  /** Liegt eine gültige Bestätigung innerhalb der Frist vor? */
  confirmed: boolean;
  latest?: VatIDCheck;
  /** Die Frist in Tagen, mit der Buchfink rechnet. */
  validityDays: number;
  note: string;
}

/** Die Aufzeichnung zu einem Geschenk (§ 4 Abs. 7 EStG). */
export interface GiftRecord {
  id: number;
  entryId: number;
  fiscalYear: number;
  date: string;
  recipientContactId?: number;
  recipientName: string;
  occasion?: string;
  /** Der Nettobetrag, an dem die Freigrenze gemessen wird. */
  netAmount: Cents;
  /** Als nicht abziehbar gebucht, weil die Freigrenze gerissen ist. */
  nonDeductible?: boolean;
  account?: string;
}

/** Eine Kategorie der beschränkt abziehbaren Betriebsausgaben (§ 4 Abs. 5 EStG). */
export interface NonDeductibleCategory {
  key: string;
  label: string;
  reference: string;
  /** Was in keiner Höhe abziehbar ist, hat kein abziehbares Konto. */
  deductibleAccount?: string;
  nonDeductibleAccount?: string;
  note: string;
}

/** Eine Kategorie mit den Summen eines Geschäftsjahres. */
export interface NonDeductibleCategoryRow extends NonDeductibleCategory {
  deductibleAmount: Cents;
  nonDeductibleAmount: Cents;
  total: Cents;
  count: number;
}

export interface GiftBookingRow {
  entryId: number;
  entryNumber: string;
  date: string;
  netAmount: Cents;
  account: string;
  deductible: boolean;
  occasion?: string;
}

export interface GiftRecipientRow {
  recipientKey: string;
  recipientName: string;
  contactId?: number;
  total: Cents;
  /** Ist die Freigrenze gerissen? Dann ist alles an diesen Empfänger nicht abziehbar. */
  overLimit: boolean;
  /** Buchungen, die noch abziehbar stehen, obwohl die Freigrenze gerissen ist. */
  toRebook: GiftBookingRow[];
  bookings: GiftBookingRow[];
  note: string;
}

export interface NonDeductibleReport {
  fiscalYear: number;
  /** Die Freigrenze je Empfänger und Wirtschaftsjahr. */
  giftLimit: Cents;
  categories: NonDeductibleCategoryRow[];
  recipients: GiftRecipientRow[];
  note: string;
}

/** Ein Devisenkurs eines Tages. */
export interface ExchangeRate {
  id: number;
  currency: string;
  date: string;
  /** 1 EUR = rateMicros / 1.000.000 Einheiten der Fremdwährung ({@link RATE_SCALE}). */
  rateMicros: number;
  /** Ein Kurs ohne Quelle ist eine Behauptung. */
  source: string;
  /** Von Hand erfasst — zulässig, aber als solcher erkennbar. */
  manual: boolean;
  createdAt: string;
}

/** Ein Umsatzsteuer-Umrechnungskurs des BMF (§ 16 Abs. 6 UStG). */
export interface VatExchangeRate {
  /** Der Monat im Format JJJJ-MM. */
  month: string;
  currency: string;
  rateMicros: number;
  source: string;
  updatedAt: string;
}

/** Ergebnis des CSV-Imports der Umsatzsteuer-Umrechnungskurse. */
export interface VatRateImport {
  imported: number;
  skipped: number;
  /** Je übergangener Zeile ein Satz mit der Zeilennummer. */
  problems: string[];
}

export interface ForeignCurrencyValuationItem {
  /** „open_item" ist eine Forderung oder Verbindlichkeit, „bank" ein Guthaben. */
  kind: string;
  entryId?: number;
  entryNumber: string;
  account: string;
  contactId?: number;
  description: string;
  currency: string;
  dueDate?: string;
  foreignAmount: Cents;
  /** Der bisher gebuchte Eurobetrag und sein Wert zum Stichtagskurs. */
  bookValue: Cents;
  valueAtCutoff: Cents;
  difference: Cents;
  /** Wirkt die Änderung als Ertrag? Bei einer Verbindlichkeit kehrt sich das um. */
  gain: boolean;
  /** Ein Gewinn aus einem langfristigen Posten wird nicht gebucht (§ 256a HGB). */
  recognised: boolean;
  amount: Cents;
  reason: string;
  rateMicros: number;
}

export interface ForeignCurrencyValuation {
  fiscalYear: number;
  cutoff: string;
  /** Der erste Tag des Folgejahres: dort wird die Bewertung wieder aufgelöst. */
  reversalDate: string;
  items: ForeignCurrencyValuationItem[];
  totalGain: Cents;
  totalLoss: Cents;
  note: string;
  /** Nach dem Buchen belegt. */
  entryNumber?: string;
  reversalEntryNumber?: string;
}

export interface WriteUpImpairment {
  fiscalYear: number;
  date: string;
  amount: Cents;
  reason: string;
}

export interface WriteUpCandidate {
  assetId: number;
  inventoryNumber: string;
  name: string;
  account: string;
  bookValue: Cents;
  /** Der Buchwert ohne die außerplanmäßige Abschreibung: die Obergrenze. */
  continuedCost: Cents;
  /** Der Spielraum: fortgeführte Anschaffungskosten minus Buchwert. */
  maxWriteUp: Cents;
  impairments: WriteUpImpairment[];
  /** Für dieses Jahr bestätigt, dass der Grund fortbesteht. */
  confirmed: boolean;
  confirmedNote?: string;
  note: string;
}

export interface WriteUpReport {
  fiscalYear: number;
  candidates: WriteUpCandidate[];
  /** Anlagegüter, für die weder zugeschrieben noch bestätigt wurde. */
  open: number;
  note: string;
}

// -------------------------------------------------------------------------
// Nachweise: Änderungsprotokoll, Versionen, Aufbewahrung,
// Verfahrensdokumentation, Datenübernahme, Altersstruktur
// -------------------------------------------------------------------------

/**
 * Die Aufbewahrungsklasse eines Objekts. Drei Klassen statt einer Frist je
 * Objektart, weil § 257 Abs. 4 HGB und § 147 Abs. 3 AO diese Staffelung
 * kennen. Leer heißt: keine Frist zugeordnet.
 */
export type RetentionClass = 'books' | 'vouchers' | 'letters' | '';

/**
 * Die Aufbewahrungsklassen in Klartext.
 *
 * Sie stehen hier und nicht je Seite: die Belegliste, die Fristenübersicht und
 * das Löschkonzept benennen dieselbe Klasse, und drei Fassungen desselben Worts
 * laufen auseinander. Die Jahreszahl steht nicht dabei — sie richtet sich nach
 * dem Entstehungsjahr und kommt aus dem Backend.
 */
export const RETENTION_CLASS_LABELS: Record<RetentionClass, string> = {
  books: 'Handelsbücher und Abschlüsse',
  vouchers: 'Buchungsbelege und Rechnungen',
  letters: 'Handelsbriefe und sonstige Unterlagen',
  '': 'Ohne Aufbewahrungsklasse',
};

/**
 * Eine Zeile der Versionshistorie (UNV-06).
 *
 * Sie kommt zerlegt aus dem Backend: die Historie liegt dort als Markdown-Datei
 * im Programm, und das Format einer Datei kennt das Paket, das sie hält.
 */
export interface ChangelogEntry {
  /** Die Bezeichnung der Fassung, „v0.1". */
  version: string;
  date: string;
  /** Wofür die Fassung steht; kann leer sein. */
  summary: string;
  changes: string[];
}

/** Ein Lauf der Schemaanpassung (UNV-06). */
export interface SchemaMigration {
  id: number;
  runAt: string;
  appVersion: string;
  /** 0 heißt: eine neue oder eine noch nicht protokollierte Datei. */
  fromVersion: number;
  toVersion: number;
  /** Die betroffenen Tabellen, durch Komma getrennt. */
  tables: string;
  /** „ok" oder „fehlgeschlagen". */
  result: string;
  message: string;
  actor?: string;
}

/** Auf welchem Weg Daten in diese Installation gekommen sind. */
export type MigrationKind = 'restore' | 'open';

/** Das Protokoll einer Datenübernahme mit ihren Zählungen (ARC-05). */
export interface MigrationRecord {
  id: number;
  kind: MigrationKind;
  source: string;
  target: string;
  runAt: string;
  appVersion: string;
  actor: string;
  journalEntries: number;
  journalLines: number;
  receipts: number;
  contacts: number;
  accounts: number;
  invoices: number;
  fixedAssets: number;
  auditEntries: number;
  debitTotal: Cents;
  creditTotal: Cents;
  /** Die Kettenprüfung unmittelbar nach der Übernahme. */
  chainValid: boolean;
  chainMessage: string;
}

/** Eine abgelegte Fassung der Verfahrensdokumentation (PRF-03). */
export interface ProcedureDocumentation {
  id: number;
  /** Die fortlaufende Fassungsbezeichnung, z. B. „2026-09-05-1". */
  version: string;
  createdAt: string;
  fiscalYear: number;
  companyName: string;
  appVersion: string;
  ruleVersion: string;
  actor: string;
  storedPath?: string;
  fileName: string;
  sha256: string;
  size: number;
  /** Leer, wenn der PDF-Satz nicht möglich war; das Markdown gilt trotzdem. */
  pdfFileName?: string;
  pdfStoredPath?: string;
  pdfSha256?: string;
  pdfSize?: number;
}

/** Die erzeugte Fassung mit ihrem Text. */
export interface ProcDocResult {
  document: ProcedureDocumentation;
  /** Sagt, warum kein PDF entstanden ist; leer heißt: es ist entstanden. */
  pdfNote?: string;
  markdown: string;
  message: string;
}

/**
 * Die unternehmensindividuellen Teile der Verfahrensdokumentation. Sie sind
 * Freitext, weil nur das Unternehmen weiß, wer scannt, prüft und vertritt.
 */
export interface OrganisationTexts {
  responsibilities: string;
  receiptFlow: string;
  scanning: string;
  approval: string;
  substitution: string;
  backup: string;
  notes: string;
}

/** Die Hinweise zu Speicherort und Steuerfällen. */
export interface ComplianceHints {
  /** Belegt, wenn der Datenordner in einem Synchronisationsordner liegt. */
  cloudWarning: string;
  dataDir: string;
  taxCaseHints: string[];
  systemChangeDate: string;
  systemChangeNote: string;
}

/** Das Ergebnis einer Sperre nach einem Löschverlangen. */
export interface BlockContactResult {
  contact: Contact;
  /** Der Antworttext an die betroffene Person, mit den Normen dazu. */
  answer: string;
}

// ---------------------------------------------------------------------------
// Die Bedienung: Aufgabenliste, Monatsabschluss, Zuordnungsvorschlag
// und Prüfpfad.
// ---------------------------------------------------------------------------

/**
 * Die drei Gruppen der Aufgabenliste (`domain.TaskGroup`).
 *
 * „overdue" ist eine verstrichene Frist, „open" die laufende Arbeit ohne Frist,
 * „upcoming" die Frist der nächsten dreißig Tage.
 */
export type TaskGroup = 'overdue' | 'open' | 'upcoming';

/**
 * Das Navigationsziel einer Aufgabe (`domain.TaskTarget`): Bereich, Reiter und
 * Kennung des Objekts, wie in `navigation.ts`.
 */
export interface TaskTarget {
  area: string;
  tab?: string;
  id?: string;
}

/** Eine Zeile der Aufgabenliste (`domain.Task`). */
export interface Task {
  key: string;
  group: TaskGroup;
  title: string;
  /** Der eine Satz, warum die Aufgabe besteht. */
  why: string;
  /** Die Norm — nur für die zweite Erklärstufe, nie im Titel. */
  reference?: string;
  target: TaskTarget;
  /** Anzahl und Betrag sind der Kontext; 0 heißt: für diese Aufgabe ohne Aussage. */
  count: number;
  amount: Cents;
  dueDate?: string;
}

/** Die Aufgabenliste in ihren drei Gruppen (`domain.TaskList`). */
export interface TaskList {
  /** Der Tag, gegen den die Fristen gerechnet wurden. */
  today: string;
  overdue: Task[];
  open: Task[];
  upcoming: Task[];
}

/** Der Stand eines Schrittes im Monatsabschluss (`service.MonthCloseStepState`). */
export type MonthCloseStepState = 'done' | 'open' | 'blocked' | 'not_applicable';

/** Ein Schritt des Monatsabschlusses (`service.MonthCloseStep`). */
export interface MonthCloseStep {
  number: number;
  key: string;
  title: string;
  state: MonthCloseStepState;
  note: string;
}

/**
 * Der Stand eines Monats in seinen drei Schritten (`service.MonthCloseState`).
 *
 * Die Reihenfolge steht im Backend und wird hier nicht nachgerechnet: erst der
 * Prüfbericht, dann die Festschreibung, dann die Bestätigung der Voranmeldung.
 */
export interface MonthCloseState {
  month: string;
  label: string;
  from: string;
  to: string;
  fiscalYear: number;
  steps: MonthCloseStep[];
  findings: CheckFinding[];
  /** Zahl der Befunde, die die Festschreibung verhindern. */
  blocking: number;
  committed: boolean;
  committedTo?: string;
  vatApplies: boolean;
  vatPeriodKey?: string;
  vatPeriodLabel?: string;
  vatStatus?: VatReturnStatus;
  vatDueDate?: string;
  vatSubmittedAt?: string;
  vatReturnId?: number;
}

/** Die Art eines Zuordnungsvorschlags (`service.SuggestionKind`). */
export type SuggestionKind = 'open_item' | 'collective' | 'rule' | 'credit_card';

/** Ein Vorschlag zu einem Bankumsatz (`service.BankSuggestion`). */
export interface BankSuggestion {
  kind: SuggestionKind;
  score: number;
  /** Die Merkmale, auf denen der Vorschlag beruht, in der Reihenfolge ihres Gewichts. */
  reasons: string[];
  label: string;
  contactId?: number;
  contactName?: string;
  amount: Cents;
  items: OpenItem[];
  counterAccount?: string;
  postingGroup?: string;
  ruleId?: number;
  /** Abrechnungsstichtag einer Kreditkartenabrechnung. */
  statementDate?: string;
}

/** Die Vorschlagsliste zu einem Bankumsatz, die beste zuerst. */
export interface BankSuggestions {
  bankTxId: number;
  amount: Cents;
  suggestions: BankSuggestion[];
  /** Steht dabei, wenn es nichts vorzuschlagen gab. */
  note?: string;
}

/** Eine gelernte Zuordnung wiederkehrender Umsätze (`domain.BankRule`). */
export interface BankRule {
  id: number;
  pattern: string;
  label: string;
  counterAccount: string;
  postingGroup?: string;
  /** Aus einem Geldeingang gelernt; ein Vorschlag mit vertauschter Richtung ist keiner. */
  moneyIn: boolean;
  hits: number;
  lastUsedAt?: string;
  createdAt: string;
  updatedAt: string;
}

// -------------------------------------------------------------------------
// Laufende Buchhaltung — Handbuchung mit Beleg, Eigenbeleg,
// Beanstandungsliste, eigene Konten, Fristen, Steuersätze, Journalfilter
// -------------------------------------------------------------------------

/** Eine abgelöste Aufbewahrungsfrist (`accounting.RetentionPreviousTerm`). */
export interface RetentionPreviousTerm {
  years: number;
  replacedFrom: string;
  note: string;
}

/** Die Frist einer Aufbewahrungsklasse (`accounting.RetentionClassRules`). */
export interface RetentionClassRules {
  class: RetentionClass;
  label: string;
  years: number;
  legalBasis: string;
  /** Die Objektarten, die in diese Klasse fallen. */
  kinds: string[];
  previous?: RetentionPreviousTerm;
}

/**
 * Die Fristentabelle mit ihrem Rechtsstand (`accounting.RetentionRules`).
 * Anzeige, keine Einstellung: eine Frist ist keine Wahl des Anwenders.
 */
export interface RetentionRules {
  version: string;
  validFrom: string;
  source: string;
  note: string;
  classes: RetentionClassRules[];
}

/** Der Vorschlag für den Voranmeldungszeitraum (`service.VatPeriodProposal`). */
export interface VatPeriodProposal {
  year: number;
  basedOnYear: number;
  /** Die Summe der Kennziffer 83 des Vorjahres. */
  priorYearTax: Cents;
  current: VatPeriodType;
  proposed: VatPeriodType;
  /** Weicht der Vorschlag vom eingestellten Zeitraum ab? */
  changes: boolean;
  /** Liegt für jeden Zeitraum des Vorjahres eine übermittelte Anmeldung vor? */
  complete: boolean;
  missingPeriods: number;
  reference: string;
  note: string;
}

/**
 * Die Einschränkungen der Journalansicht (`accounting.JournalFilter`).
 *
 * Alle Felder sind freiwillig. `amountFrom`, `amountTo` und `hasReceipt`
 * unterscheiden „nicht gesetzt" von „null" bzw. „nein": ein Betragsfilter „von
 * 0 €" ist etwas anderes als kein Betragsfilter.
 */
export interface JournalFilter {
  from?: string;
  to?: string;
  account?: string;
  counterAccount?: string;
  amountFrom?: Cents;
  amountTo?: Cents;
  taxKey?: string;
  actor?: string;
  /** Erfassungsweg: `ui`, `system` oder `agent:<kurzname>`. */
  channel?: string;
  hasReceipt?: boolean;
  text?: string;
}

/** Eine gefilterte Journalzeile mit dem Kopf ihrer Buchung (`accounting.JournalFilterRow`). */
export interface JournalFilterRow {
  entryId: number;
  entryNumber: string;
  bookingDate: string;
  documentDate: string;
  documentNumber?: string;
  description: string;
  kind: EntryKind;
  actor?: string;
  channel?: string;
  receiptId?: number;
  position: number;
  account: string;
  accountName?: string;
  side: Side;
  amount: Cents;
  taxKey?: string;
  taxBase?: Cents;
  text?: string;
}

/** Die gefilterte Menge mit ihrer Summenzeile (`accounting.JournalFilterResult`). */
export interface JournalFilterResult {
  rows: JournalFilterRow[];
  rowCount: number;
  entryCount: number;
  totalDebit: Cents;
  totalCredit: Cents;
  /** Soll minus Haben der gefilterten Menge; nicht notwendig null. */
  balance: Cents;
}

export interface BankAccount {
  iban: string;
  name: string;
  ledgerAccount: string;
  currency: string;
  bic?: string;
  bankName?: string;
  /** Das Konto, dessen Verbindung auf Rechnungen steht. */
  isInvoiceAccount?: boolean;
}

export interface Country {
  code: string;
  name: string;
}

/** Ein Gesellschafter der aktuellen Gesellschafterliste. */
export interface CompanyShareholder {
  id?: number;
  name: string;
  shareCapital: Cents;
}
export interface BankImportPreview {
  accounts: BankAccount[];
  transactions: number;
}

/** Zustand des MCP-Servers für verbundene Apps. */
export interface MCPServerStatus {
  running: boolean;
  port: number;
  /** Adresse, die Apps eintragen, z. B. http://127.0.0.1:7465/mcp. */
  url: string;
  /** Warum der Server nicht läuft, etwa ein belegter Port; sonst leer. */
  error: string;
  /** BUCHFINK_MCP_PORT setzt den Port; der eingestellte gilt erst ohne die Variable. */
  fromEnvironment: boolean;
}

/** Ein Mandant, für den eine App zugelassen ist. */
export interface ConnectedAppTenant {
  id: string;
  name: string;
  scopes: string[];
  grantedAt: string;
}

/** Eine App, der in Buchfink zugestimmt wurde. */
export interface ConnectedApp {
  clientId: string;
  /** Name, den die App bei der Anmeldung angab. */
  name: string;
  /** Erfassungsweg an jeder Buchung dieser App, z. B. agent:claude-code. */
  channel: string;
  approvedAt: string;
  /** Leer: nie benutzt. */
  lastUsedAt: string;
  tenants: ConnectedAppTenant[];
  /** Keine gültige Zulassung mehr. */
  revoked: boolean;
}

/**
 * Eine App bittet um Zugriff. Der Browser zeigt denselben Abgleichcode; die
 * Entscheidung geht mit der Nonce an ResolveAppRequest.
 */
export interface AppConsentRequest {
  id: string;
  nonce: string;
  clientName: string;
  clientId: string;
  clientRegisteredAt: string;
  knownSince: string | null;
  redirectUri: string;
  redirectTo: string;
  scopes: string[];
  tenantId: string;
  tenantName: string;
  userCode: string;
  expiresAt: string;
}

/** Eine Zeile der Vorschau: ein Feld heute und wie vorgeschlagen (`service.MasterDataChangeLine`). */
export interface MasterDataChangeLine {
  area: string;
  field: string;
  before: string;
  after: string;
}

/**
 * Ein Vorschlag eines Agenten für die Stammdaten mit Vorschau gegen den
 * heutigen Stand (`service.MasterDataProposalView`). Übernommen wird nur mit
 * dem Fingerabdruck der Vorschau, die der Mensch gesehen hat.
 */
export interface MasterDataProposal {
  id: number;
  status: 'offen' | 'uebernommen' | 'abgelehnt';
  reason: string;
  receiptId?: number;
  receiptNumber?: string;
  channel: string;
  createdAt: string;
  lines: MasterDataChangeLine[];
  fingerprint: string;
  /** Warum sich der Vorschlag nicht übernehmen lässt. */
  problem?: string;
}

// --- Übersicht: Pflichtenkalender und Buchungen der Apps ------------------

/** Stand einer Pflicht im Pflichtenkalender (`domain.ObligationStatus`). */
export type ObligationStatus = 'open' | 'prepared' | 'done' | 'overdue' | 'waived';

/** Woher der Stand kommt: aus den Daten, aus einer Bestätigung oder aus beidem. */
export type ObligationEvidenceSource = 'data' | 'human' | 'either';

/** Wer die Pflicht trägt (`domain.ObligationResponsibility`). */
export type ObligationResponsibility = 'agent_prepares' | 'human_confirms' | 'tax_advisor';

/** Art eines Nachweises (`domain.EvidenceKind`). */
export type EvidenceKind =
  | 'transfer_ticket'
  | 'file_number'
  | 'order_number'
  | 'booking'
  | 'commitment'
  | 'resolution'
  | 'status'
  | 'confirmation';

/** Der Nachweis einer erledigten Pflicht. */
export interface ObligationEvidence {
  date: string;
  kind: EvidenceKind;
  reference?: string;
  note?: string;
  /** Aus den Daten abgeleitet, nicht bestätigt. */
  automatic: boolean;
  channel?: string;
  actor?: string;
}

/** Die Meldung eines Agenten oder Menschen, dass die Pflicht vorbereitet ist. */
export interface ObligationPreparation {
  at: string;
  note?: string;
  /** `ui` oder `agent:<kurzname>`. */
  channel: string;
  actor: string;
}

/**
 * Eine Pflicht aus dem Pflichtenkalender mit der Stelle der Oberfläche, an der
 * sie bearbeitet wird (`wailsbridge.ObligationView`).
 */
export interface Obligation {
  /** Pflichtart und Zeitraum, etwa „ustva.2026-Q1“. */
  key: string;
  kind: string;
  title: string;
  description: string;
  /** Die Rechtsgrundlage, nur für die Erklärstufe. */
  basis: string;
  channel: string;
  responsibility: ObligationResponsibility;
  evidenceSource: ObligationEvidenceSource;
  skillRef: string;
  periodKey: string;
  periodLabel: string;
  periodFrom?: string;
  periodTo?: string;
  fiscalYear: number;
  /** Gesetzliches Fristende; `dueDate` ist das nach Wochenende und Feiertag verschobene. */
  statutoryDate: string;
  dueDate: string;
  shiftReason?: string;
  /** „Unverzüglich“: die Frist beginnt am Fälligkeitstag. */
  immediate?: boolean;
  status: ObligationStatus;
  proof?: ObligationEvidence;
  preparation?: ObligationPreparation;
  waivedReason?: string;
  hints: string[];
  target: TaskTarget;
}

/** Der Nachweis, mit dem der Mensch eine Pflicht bestätigt (`service.ConfirmInput`). */
export interface ObligationConfirmation {
  date: string;
  kind: EvidenceKind;
  reference?: string;
  note?: string;
}

/** Eine Buchung einer verbundenen App (`wailsbridge.AgentBooking`). */
export interface AgentBooking extends JournalEntry {
  /** Eine Generalumkehr hebt die Buchung auf. */
  reversed: boolean;
}

/** Die Buchungen der Apps, die noch nicht festgeschrieben sind. */
export interface AgentBookings {
  /** Die Festschreibung mit dem jüngsten Stichtag; fehlt ohne Festschreibung. */
  lastCommit?: Festschreibung;
  /** Alle solchen Buchungen; `entries` enthält die jüngsten. */
  total: number;
  entries: AgentBooking[];
}

// -------------------------------------------------------------------------
// Jahreserklärungen (internal/declaration). Die Schlüssel sind die des
// JSON-Exports buchfink.erklaerung/1 und deshalb deutsch.

export type DeclarationKind = 'ust_jahr' | 'kst' | 'gewst';

export type DeclarationStatus = 'entwurf' | 'exportiert' | 'uebermittelt';

export type DeclarationLineStatus = 'auto' | 'vorgeschlagen' | 'eingabe' | 'offen' | 'entfaellt';

export type DeclarationOwner = 'buecher' | 'agent_rueckfrage' | 'mensch';

export type DeclarationUnit =
  | 'euro_voll'
  | 'euro_cent'
  | 'text'
  | 'ja_nein'
  | 'datum'
  | 'prozent'
  | 'schluessel';

/** Ein Wert: Betrag in Cent, Text, ja/nein oder Datum. */
export interface DeclarationValue {
  cent?: Cents;
  betrag?: string;
  text?: string;
  ja?: boolean;
  datum?: string;
}

export interface DeclarationAccountPart {
  konto: string;
  bezeichnung: string;
  saldo: Cents;
  anteil_zaehler: number;
  anteil_nenner: number;
  beitrag: Cents;
}

export interface DeclarationOrigin {
  art:
    | 'konten'
    | 'buchungen'
    | 'rechnung'
    | 'stammdaten'
    | 'eingabe'
    | 'voranmeldungen'
    | 'anlagenkartei'
    | 'abschluss'
    | 'keine';
  konten?: DeclarationAccountPart[];
  buchungen?: number[];
  aus_zeilen?: string[];
  rechnung?: string;
  eingabe?: string;
  erlaeuterung?: string;
}

export interface DeclarationLine {
  schluessel: string;
  zeile?: string;
  kz?: string;
  bezeichnung: string;
  wert: DeclarationValue;
  einheit: DeclarationUnit;
  rundung: string;
  herkunft: DeclarationOrigin;
  status: DeclarationLineStatus;
  zustaendig: DeclarationOwner;
  eingabe_moeglich: boolean;
  pflicht: boolean;
  vorlaeufig?: boolean;
  rechtsgrundlage?: string[];
  pruefvermerk: { am_vordruck_geprueft: boolean; quelle: string };
  hinweise?: string[];
}

export interface DeclarationSummary {
  zeilen: number;
  mit_wert: number;
  offen: number;
  vorgeschlagen: number;
  eingaben: number;
}

export interface DeclarationSection {
  id: string;
  titel: string;
  zeilen: DeclarationLine[];
  uebersicht: DeclarationSummary;
}

export type DeclarationRequirement = 'pflicht' | 'bei_bedarf' | 'entfaellt' | 'kontrolle';

export interface DeclarationForm {
  id: string;
  titel: string;
  erforderlich: DeclarationRequirement;
  grund?: string;
  abschnitte: DeclarationSection[];
  uebersicht: DeclarationSummary;
}

export type DeclarationInputSource = 'bescheid' | 'beschluss' | 'beleg' | 'mensch' | 'buecher';

export interface DeclarationInput {
  schluessel: string;
  wert: DeclarationValue;
  quelle: DeclarationInputSource;
  notiz?: string;
  erfassungsweg: string;
  erfasst_von: string;
  erfasst_am: string;
  bestaetigt: boolean;
  bestaetigt_von?: string;
  bestaetigt_am?: string;
}

export interface DeclarationFinding {
  schwere: 'blockiert' | 'warnung' | 'hinweis';
  code: string;
  zeile?: string;
  text: string;
  rechtsgrundlage?: string;
}

export interface DeclarationSubmission {
  am: string;
  transferticket: string;
  zahlen_hash: string;
  bearbeiter: string;
  begruendung?: string;
}

export interface DeclarationReconciliationRow {
  kz_voranmeldung: string;
  bezeichnung: string;
  feld: string;
  jahr: Cents;
  voranmeldungen: Cents;
  differenz: Cents;
  grund?: string;
}

export interface Declaration {
  art: DeclarationKind;
  titel: string;
  zeitraum: { jahr: number; von: string; bis: string };
  vordruckjahr: number;
  formulare: DeclarationForm[];
  abstimmung?: DeclarationReconciliationRow[];
  eingaben: DeclarationInput[];
  befunde: DeclarationFinding[];
  status: DeclarationStatus;
  grundlage: {
    abschlussstand: string;
    buchungen: number;
    zeilenkatalog: string;
    gerechnet_am: string;
  };
  zahlen_hash: string;
  uebermittlung?: DeclarationSubmission;
}

export interface DeclarationPendingLine {
  schluessel: string;
  formular: string;
  bezeichnung: string;
  status: DeclarationLineStatus;
  zustaendig: DeclarationOwner;
  wert: DeclarationValue;
  hinweise?: string[];
}

export interface DeclarationOverview {
  art: DeclarationKind;
  titel: string;
  jahr: number;
  status: DeclarationStatus;
  zahlen_hash: string;
  abgabereif: boolean;
  /** Die Hauptzahlen; `kurz` steht über der Kennzahl, `bezeichnung` ist der amtliche Wortlaut. */
  ergebnis: { schluessel: string; bezeichnung: string; kurz?: string; wert: DeclarationValue }[];
  formulare: {
    id: string;
    titel: string;
    erforderlich: DeclarationRequirement;
    grund?: string;
    uebersicht: DeclarationSummary;
  }[];
  zu_klaeren: DeclarationPendingLine[];
  befunde_blockierend: number;
  befunde_warnung: number;
  befunde_hinweis: number;
  uebermittlung?: DeclarationSubmission;
}

/** Was GetDeclaration liefert: Übersicht und vollständige Erklärung. */
export interface DeclarationView {
  overview: DeclarationOverview;
  declaration: Declaration;
}

export interface DeclarationInputRequest {
  kind: DeclarationKind;
  year: number;
  key: string;
  value: DeclarationValue;
  source: DeclarationInputSource;
  note?: string;
}

export interface DeclarationSubmissionRequest {
  kind: DeclarationKind;
  year: number;
  submittedAt: string;
  transferTicket: string;
  figuresHash: string;
  overrideReason?: string;
}

/** Eine geschriebene Exportdatei einer Erklärung. */
export interface DeclarationExport {
  pfad: string;
  sha256: string;
  zahlen_hash: string;
  status: DeclarationStatus;
  befunde_blockierend: number;
  stand_id: number;
}

/** Eine festgehaltene Übermittlung oder ein Export. */
export interface DeclarationRecord {
  id: number;
  kind: string;
  year: number;
  status: string;
  figuresHash: string;
  submittedAt?: string;
  transferTicket?: string;
  createdAt: string;
}

/** Eine KI-App, zu der es eine Anleitung gibt (internal/wailsbridge/company_service.go). */
export type AgentClient = 'claude-code' | 'claude-desktop' | 'chatgpt-desktop' | 'opencode' | 'codex' | 'pi';

// -------------------------------------------------------------
// Gesellschafterdarlehen (internal/domain/shareholder.go,
// internal/service/shareholder_service.go)
// -------------------------------------------------------------

/** Wer wem leiht: der Gesellschafter der GmbH oder die GmbH dem Gesellschafter. */
export type LoanDirection = 'to_company' | 'to_shareholder';

export type LoanDayCount = 'act/365' | 'act/360' | '30/360';

/** Ein Darlehensvertrag zwischen der GmbH und einem Gesellschafter. */
export interface ShareholderLoan {
  id: number;
  contactId: number;
  direction: LoanDirection;
  principal: Cents;
  /** Zinssatz in Hundertstel Prozent: 350 = 3,50 %. */
  interestRateBp: number;
  dayCount: LoanDayCount;
  contractDate: string;
  startDate: string;
  /** Leer: ohne bestimmte Laufzeit. */
  maturityDate?: string;
  account: string;
  interestAccount: string;
  collateral?: string;
  /** Rangrücktritt vereinbart. */
  subordinated: boolean;
  note?: string;
}

/** Ein Darlehen mit gebuchtem Stand und den Befunden zum Fremdvergleich. */
export interface ShareholderLoanView {
  loan: ShareholderLoan;
  contactName: string;
  outstanding: Cents;
  firstMovement?: string;
  interestBookedThrough?: string;
  findings: CheckFinding[];
}

/** Ein neuer oder berichtigter Vertrag; leere Konten leitet Buchfink ab. */
export interface ShareholderLoanRequest {
  contactId: number;
  direction: LoanDirection;
  principal: Cents;
  interestRateBp: number;
  dayCount?: LoanDayCount;
  contractDate?: string;
  startDate: string;
  maturityDate?: string;
  collateral?: string;
  subordinated?: boolean;
  note?: string;
}

/** Art einer Datei im Eingang, wie Buchfink sie erkennt (`inbox.Kind`). */
export type InboxKind = 'beleg' | 'e_rechnung' | 'kontoauszug' | 'nicht_unterstuetzt';

/** Der Eingang des geöffneten Mandanten mit den Ordnern der verbundenen Apps (`wailsbridge.InboxInfo`). */
export interface InboxInfo {
  path: string;
  /** Der Eingang ist der voreingestellte im Datenordner. */
  default: boolean;
  count: number;
  /** Alter der ältesten Datei in Tagen; 0 ohne Datei. */
  oldestDays: number;
  readFolders: string[];
  exportFolder: string;
}

/** Eine Datei im Eingang (`wailsbridge.InboxFile`). */
export interface InboxFile {
  path: string;
  name: string;
  size: number;
  /** RFC 3339 */
  modifiedAt: string;
  kind: InboxKind;
  /** Derselbe Inhalt liegt schon als dieser Beleg im Archiv. */
  receiptId?: number;
  /** Das Geschäftsjahr dieses Belegs; die Belegliste zeigt nur eines. */
  receiptYear?: number;
}

/** Eine Datei, die nicht in den Eingang kam, mit Grund. */
export interface InboxSkipped {
  name: string;
  reason: string;
}

/** Das Ergebnis, wenn Dateien in den Eingang gelegt werden (`wailsbridge.InboxAdded`). */
export interface InboxAdded {
  names: string[];
  skipped: InboxSkipped[];
  error?: string;
}

/** Ein Synchronisationsdienst über einem Ordner und der Hinweis dazu (`wailsbridge.FolderCheck`). */
export interface FolderCheck {
  /** Name des Dienstes, etwa „iCloud Drive“; leer ohne Dienst. */
  sync: string;
  warning: string;
}

/** Die Regeln für verbundene Apps, wie gespeichert; 0 und leer heißen Voreinstellung (`domain.AgentSettings`). */
export interface AgentSettings {
  /** Die Direktgrenze brutto in Cent. */
  directMaxGross: Cents;
  /** Frühere Buchungen mit der Standardgruppe, ab der direkt gebucht wird. */
  directMinBookings: number;
  extraReadFolders: string[];
  exportFolder: string;
}

/** Die Regeln für verbundene Apps: gespeichert und wie sie gelten (`wailsbridge.AgentRules`). */
export interface AgentRules {
  stored: AgentSettings;
  directMaxGross: Cents;
  directMinBookings: number;
  extraReadFolders: string[];
  exportFolder: string;
  /** Die Voreinstellungen, für die ein leeres Feld steht. */
  defaults: { directMaxGross: Cents; directMinBookings: number };
}

/** Die grobe Art einer Änderung in einem Mandanten (`core.ChangeKind`). */
export type ChangeKind = 'buchungen' | 'belege' | 'bank' | 'stammdaten' | 'abschluss' | 'eingang';

/** Die Änderungen eines Mandanten seit dem letzten Ereignis (`wailsbridge.ChangeEvent`). */
export interface ChangeEvent {
  tenantId: string;
  /** Die Erfassungswege: `ui`, `system`, `agent:<kurzname>`. */
  channels: string[];
  kinds: ChangeKind[];
}
