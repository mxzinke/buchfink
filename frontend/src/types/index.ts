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
  | 'reverse_charge'
  | 'intra_community_acquisition'
  | 'intra_community_supply'
  | 'export'
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
export type Settlement = 'open' | 'paid';
export type AccountType = 'asset' | 'liability' | 'equity' | 'revenue' | 'expense' | 'statistical';

// -------------------------------------------------------------------------

export interface TenantConfig {
  id: string;
  name: string;
  dataDir: string;
  createdAt: string;
  /** Kennung im Schlüsselbund. Leer heißt: dieselbe wie `id`. */
  keyId?: string;
  /** Zielordner der Sicherung. Leer heißt: keine Sicherung eingerichtet. */
  backupDir?: string;
  lastBackupAt?: string;
  /** Letzter Tag des Prüfermodus (JJJJ-MM-TT). Leer heißt: aus. */
  readOnlyUntil?: string;
  readOnlyReason?: string;
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

/**
 * Die Prüfung des 15-%-Rahmens des § 6 Abs. 1 Nr. 1a EStG zu einer
 * Instandsetzung an einem Gebäude.
 */
export interface NearAcquisitionCheck {
  applicable: boolean;
  periodEnd?: string;
  limit: Cents;
  spent: Cents;
  planned: Cents;
  exceeded: boolean;
  note: string;
}

/**
 * Das Ergebnis einer Erhaltungsaufwandsbuchung: die Buchung und, bei einem
 * Gebäude in den ersten drei Jahren, die Prüfung des 15-%-Rahmens.
 */
export interface MaintenanceResult {
  entry: JournalEntry;
  nearAcquisition?: NearAcquisitionCheck;
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
  reversalOfId?: number;
  reversalReason?: string;
  currency: string;
  exchangeRateMicros: number;
  exchangeRateSource?: string;
  exchangeRateDate?: string;
  postingRuleVersion?: string;
  lines: JournalLine[];
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

export interface DifferenceKindInfo {
  kind: DifferenceKind;
  label: string;
  hint: string;
  /**
   * Zu dieser Differenzart fließt kein Geld. Sie steht in derselben Auswahl,
   * geht aber über „Forderung ausbuchen" und nicht über den Zahlungsausgleich.
   */
  withoutPayment?: boolean;
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
   * auf den Aufwand; die Vorsteuer hängt an der Zahlung (§ 15 Abs. 1 Satz 1
   * Nr. 1 Satz 3 UStG).
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
}

/**
 * Der Buchungssatz, wie das Backend ihn berechnet — ohne ihn zu schreiben.
 *
 * Die Oberfläche zeigt diese Zahlen an und rechnet sie nicht selbst nach. Eine
 * zweite Steuerrechnung im Frontend wäre eine zweite Wahrheit, die auseinander
 * läuft, sobald ein Steuerfall dazukommt.
 */
export interface PostingPreview {
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
  /** Der Tageskurs, mit dem der Aufwand bewertet wird. */
  rate: ExchangeRate;
  /**
   * Der Durchschnittskurs, mit dem die Bemessungsgrundlage der Umsatzsteuer
   * gerechnet wird (§ 16 Abs. 6 UStG). Fehlt er, bleibt es beim Tageskurs.
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
export type ReceiptKind = 'invoice' | 'statement' | 'self_issued' | 'other';

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

export interface ValidationResult {
  ruleset: string;
  version: string;
  format: string;
  profile: string;
  coverage: 'partial';
  findings?: ValidationFinding[];
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
  /** Die unstrukturierte Anschrift aus der Zeit vor der strukturierten. */
  address: string;
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
 * Ein Versandweg mit seiner Beschriftung (domain.InvoiceSentViaOption).
 *
 * Die Wörter kommen aus dem Backend und nicht aus der Seite: Wertelisten mit
 * fester Bedeutung haben eine Quelle, sonst heißt derselbe Weg an zwei Stellen
 * verschieden.
 */
export interface InvoiceSentViaOption {
  via: InvoiceSentVia;
  label: string;
}

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
  /** Leer heißt „Rechnung": Bestandsdaten aus der Zeit vor der Dokumentart. */
  kind: InvoiceKind;
  terms: PaymentTerms;
  /** Kleinbetragsrechnung nach § 33 UStDV: verkürzte Angaben, kein Empfänger nötig. */
  smallAmount: boolean;
  /** Zahlungsmittelkonto einer Rechnung ohne Empfänger; leer heißt Kasse. */
  paymentAccount?: string;
  /** Das Format, in dem dieses Dokument erzeugt wurde. */
  eInvoiceProfile?: EInvoiceProfile;
  /**
   * Der Grund, mit dem eine steuerfreie ig. Lieferung ohne Bestätigung der
   * USt-IdNr. ausgestellt wurde. Er steht an der Rechnung, weil die Frage nach
   * der Befreiung später an ihr gestellt wird.
   */
  vatIdOverrideReason?: string;
  /**
   * Wer den Gegenstand befördert hat. Leer wird als Regelfall gelesen —
   * Beförderung durch den Lieferer; „customer" ist der Abholfall und verlangt
   * zusätzlich die Gelangensbestätigung.
   */
  transportKind?: TransportKind;
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
  /** So viele davon tragen ein Dokument. */
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

/**
 * Ein Rechnungsverbund: ein Auftrag, abgerechnet in Abschlägen und einer
 * Schlussrechnung.
 */
export interface InvoiceGroup {
  id: number;
  fiscalYear: number;
  contactId: number;
  title: string;
  /** Der vereinbarte Gesamtbetrag netto — Obergrenze der Abschläge. */
  totalNet: Cents;
  taxRate: TaxRate;
  closed: boolean;
  finalInvoiceId?: number;
  advances: AdvanceItem[];
  /** Der Stand des Verbunds, im Backend gerechnet (domain.GroupProgress). */
  progress: GroupProgress;
  createdAt: string;
}

/**
 * Abgerechnet, vereinnahmt und offen zu einem Verbund.
 *
 * Die Summen kommen aus dem Backend und werden in der Oberfläche nicht
 * nachgerechnet: welche Abschläge mitzählen, ist eine fachliche Regel
 * (stornierte fallen heraus, vereinnahmt zählt erst mit dem Zahlungsdatum) und
 * gehört an eine einzige Stelle.
 */
export interface GroupProgress {
  agreedNet: Cents;
  billedNet: Cents;
  receivedNet: Cents;
  receivedTax: Cents;
  receivedGross: Cents;
  openNet: Cents;
  closed: boolean;
}

export interface AdvanceGroupRequest {
  contactId: number;
  title: string;
  totalNet: Cents;
  taxRate: TaxRate;
}

export interface AdvanceInvoiceRequest {
  groupId: number;
  date: string;
  description: string;
  net: Cents;
  /** Der Vereinnahmungszeitpunkt, sofern er beim Ausstellen feststeht. */
  paymentReceivedAt?: string;
}

export interface SettleAdvanceRequest {
  /** Die Abschlagsrechnung, auf die das Geld eingegangen ist. */
  advanceId: number;
  /** Der Bankumsatz, aus dem die Vereinnahmung stammt. */
  bankTxId?: number;
  paymentDate: string;
  paymentAccount: string;
}

export interface RefundAdvanceRequest {
  advanceId: number;
  refundDate: string;
  paymentAccount: string;
  reason: string;
}

export interface FinalInvoiceRequest {
  groupId: number;
  date: string;
  serviceDateFrom: string;
  serviceDateTo: string;
  items: InvoiceItem[];
  terms: PaymentTerms;
}

/** Wofür angezahlt wurde — die Angabe entscheidet über den Bilanzposten. */
export type AdvanceTarget = 'inventory' | 'tangible' | 'intangible';

export interface AdvanceTargetOption {
  key: AdvanceTarget;
  label: string;
  account: string;
}

/** Eine geleistete, noch nicht verrechnete Anzahlung an einen Lieferanten. */
export interface VendorAdvance {
  id: number;
  contactId: number;
  receiptId: number;
  entryId: number;
  documentNumber: string;
  account: string;
  target: AdvanceTarget;
  netAmount: Cents;
  taxAmount: Cents;
  grossAmount: Cents;
  taxRate: TaxRate;
  paidAt: string;
  settledByEntryId?: number;
}

/** Die Ausbuchung eines uneinbringlichen Postens; die Begründung ist Pflicht. */
export interface WriteOffRequest {
  openItemEntryId: number;
  /** Bruttobetrag; null heißt der ganze offene Betrag. */
  amount: Cents;
  date: string;
  reason: string;
}

export interface VatFigure {
  rate: TaxRate;
  net: Cents;
  tax: Cents;
}

export interface VatSummary {
  fiscalYear: number;
  periodFrom: string;
  periodTo: string;
  taxableRevenue: VatFigure[] | null;
  exemptRevenue: Cents;
  intraCommunitySupply: Cents;
  export: Cents;
  reverseChargeSupply: Cents;
  outputTax: Cents;
  reverseChargeTax: Cents;
  reverseChargeBase: Cents;
  intraCommunityAcquisitionTax: Cents;
  intraCommunityAcquisitionBase: Cents;
  totalOwedTax: Cents;
  inputTax: Cents;
  payable: Cents;
}

export interface CashflowDataPoint {
  month: string;
  label: string;
  inflow: Cents;
  outflow: Cents;
  net: Cents;
}

export interface FinancialSummary {
  totalRevenue: Cents;
  totalExpenses: Cents;
  netIncome: Cents;
  bankBalance: Cents;
  openReceivables: Cents;
  openPayables: Cents;
  cashflowHistory: CashflowDataPoint[] | null;
}

/**
 * Woran die Kette zerbrochen ist: `linkage` heißt, eine Buchung wurde
 * eingefügt oder entfernt; `content` heißt, eine Buchung wurde verändert.
 */
export type IntegrityBreakReason = 'linkage' | 'content';

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
  /** Die geprüften Geschäftsjahre, aufsteigend. Jedes trägt eine eigene Kette. */
  fiscalYears: number[];
  /** Alle Brüche, nicht nur der erste. Leer heißt: unversehrt. */
  breaks: IntegrityBreak[];
}

export interface CompanySettings {
  companyName: string;
  legalForm: string;
  fiscalYear: number;
  fiscalYearStartMonth: number;
  taxNumber: string;
  vatId: string;
  taxOffice: string;
  iban: string;
  bic: string;
  bankName: string;
  street: string;
  zipCity: string;
  country: string;
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
   * Sitz, Registergericht und Registernummer sind die Pflichtangaben des
   * § 264 Abs. 1a HGB auf jedem Jahresabschluss. Sie standen bisher nur an der
   * Gründung und fehlten damit jedem Mandanten ohne Gründungsweg.
   */
  seat: string;
  registerCourt: string;
  registerNumber: string;
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
  /** Nachfrist für die Festschreibung des Vormonats; 0 heißt Monatsende. */
  commitGraceDays: number;
  /**
   * Legt die Anlegerstellung für § 20 InvStG ausdrücklich fest — normalerweise
   * leer, weil sie aus der Rechtsform folgt.
   *
   * Gebraucht wird sie in zwei Fällen: bei einer Personengesellschaft, wo
   * § 20 Abs. 3a InvStG auf den Gesellschafter abstellt, und bei den Ausnahmen
   * des § 20 Abs. 1 Sätze 4 und 5 — Lebens- und Krankenversicherer,
   * Kreditinstitute mit Handelsbestand, Pensionsfonds.
   */
  investorOverride: InvestorType;
}

/** Eine Rechtsform aus dem Katalog, mit dem, was sie steuerlich nach sich zieht. */
export interface LegalFormInfo {
  name: string;
  /** Die abgeleitete Anlegerstellung. Leer heißt: aus der Rechtsform folgt sie nicht. */
  investor: InvestorType;
  note: string;
}

export interface AuditLogEntry {
  id: number;
  timestamp: string;
  action: string;
  entityType: string;
  entityId: string;
  details: string;
  previousHash?: string;
  entryHash?: string;
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
// SKR04-Katalog (statisch, aus skr04_2026.json)

export interface SKR04Position {
  id: string;
  name: string;
  statement_type: string;
  balance_side: string;
  hgb_code: string;
  group: string;
  main_group: string;
  account_type: string;
  kontenklasse: { number: number; name: string };
  account_numbers: string[];
  accounts_count: number;
}

export interface SKR04Metadata {
  title: string;
  subtitle: string;
  validity_from: string;
  version: string;
  article_number: string;
  source_file: string;
  description: string;
  generated_at: string;
}

export interface SKR04Legend {
  hauptfunktionen: Record<string, string>;
  zusatzfunktionen: Record<string, string>;
  abschlusszweck: Record<string, string>;
  programmverbindung: Record<string, string>;
  footnotes: Record<string, string>;
}

export interface SKR04Statistics {
  total_accounts: number;
  active_accounts: number;
  reserved_accounts: number;
  range_accounts: number;
  total_positions: number;
  accounts_by_type: Record<string, number>;
  accounts_by_kontenklasse: Record<string, number>;
  positions_by_side: Record<string, number>;
}

export interface SKR04Catalog {
  metadata: SKR04Metadata;
  legend: SKR04Legend;
  statistics: SKR04Statistics;
  positions: SKR04Position[];
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
export type InvestorType = '' | 'basic' | 'individual_business' | 'corporate' | 'mixed';

/** Was zu einem Anlagegut abgelegt wird, ohne gebucht zu werden. */
export type AssetDocumentKind =
  | 'contract'
  | 'invoice'
  | 'valuation'
  | 'registration'
  | 'insurance'
  | 'maintenance'
  | 'statement'
  | 'photo'
  | 'other';

export interface AssetDocument {
  id: number;
  assetId: number;
  kind: AssetDocumentKind;
  title?: string;
  fileName: string;
  mimeType: string;
  size: number;
  sha256: string;
  storedPath: string;
  documentDate?: string;
  /** Tag, an dem das Dokument abläuft — eine Police, eine Frist. */
  validUntil?: string;
  note?: string;
  createdAt: string;
}

export interface AssetDocumentKindInfo {
  kind: AssetDocumentKind;
  label: string;
}

export interface ExpiringAssetDocument {
  assetId: number;
  inventoryNumber: string;
  assetName: string;
  documentId: number;
  kind: AssetDocumentKind;
  kindLabel: string;
  title: string;
  validUntil: string;
}

export type AssetMovementKind =
  | 'transfer'
  | 'acquisition'
  | 'subsequent_cost'
  | 'cost_reduction'
  | 'depreciation'
  | 'special_depreciation'
  | 'impairment'
  | 'write_up'
  | 'maintenance'
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

/** Ein Stück in der Skalierung von {@link Units}. */
export const UNIT_SCALE = 10000;

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
  /** Betrag einer Bewegung ohne Buchwertänderung (Erhaltungsaufwand). */
  expenseAmount?: Cents;
  /** Aufwandskonto einer Bewegung ohne Buchwertänderung (Erhaltungsaufwand). */
  expenseAccount?: string;
  /**
   * Instandsetzung oder Modernisierung im Sinne des § 6 Abs. 1 Nr. 1a EStG.
   * Nur solcher Aufwand zählt in den 15-%-Rahmen der ersten drei Jahre.
   */
  isModernisation?: boolean;
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
  documents?: AssetDocument[];

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

export interface AssetSummary {
  fiscalYear: number;
  count: number;
  cost: Cents;
  accumulated: Cents;
  bookValue: Cents;
  yearAmount: Cents;
  dueAmount: Cents;
  specialDue: Cents;
  dueCount: number;
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
  /** Die Sätze, die zu genau diesem Anlagegut gehören — vom Backend gerechnet. */
  notes: string[];
}

export interface AssetAccountInfo {
  number: string;
  name: string;
  class: AssetClass;
  group: string;
  hint?: string;
  /** Anlagen im Bau und geleistete Anzahlungen: von hier wird umgebucht. */
  inProgress?: boolean;
  /** Grund und Boden und alles, was darauf steht — keine degressive AfA, keine Sonderabschreibung. */
  immovable?: boolean;
  /**
   * Ein Gebäude, das Wohnzwecken dient. § 7 Abs. 4 EStG macht den Satz beim
   * Wohngebäude an der Fertigstellung fest, beim Betriebsgebäude am Bauantrag.
   */
  residential?: boolean;
  depreciationAccount?: string;
  depreciable: boolean;
  defaultUsefulLifeMonths?: number;
  usefulLifeSource?: string;
  /**
   * Eine Abweichung vom Vorschlag dieses Kontos ist zu begründen. Gesetzt für
   * die Konten mit dem Wahlrecht des BMF-Schreibens vom 22.02.2022; das
   * Backend verlangt dort dasselbe.
   */
  usefulLifeReasonRequired?: boolean;
}

export type AcquisitionOption = 'immediate' | 'pool' | 'activate';

export interface AcquisitionAdvice {
  recommended: AcquisitionOption;
  allowed: AcquisitionOption[];
  reason: string;
  poolNote?: string;
  limits: {
    immediate: Cents;
    recordFrom: Cents;
    poolLowerLimit: Cents;
    poolUpperLimit: Cents;
  };
}

/**
 * Ein Zeitfenster, in dem die degressive AfA zulässig ist. Die Schlüssel sind
 * kleingeschrieben, seit die Regeln aus der Ressource `afa_rules.json` kommen
 * und das Go-Struct dieselben JSON-Namen trägt wie die Datei.
 */
export interface DegressiveWindow {
  from: string;
  until: string;
  /** Vielfaches des linearen Satzes in Tausendsteln: 3000 ist das Dreifache. */
  factorPermille: number;
  maxPermille: number;
  source: string;
}

export interface AssetMethodInfo {
  method: DepreciationMethod;
  label: string;
  classes: AssetClass[];
  hint: string;
}

export interface AssetRules {
  fiscalYear: number;
  gwgImmediateLimit: Cents;
  gwgRecordFrom: Cents;
  poolLowerLimit: Cents;
  poolUpperLimit: Cents;
  poolYears: number;
  degressiveWindows: DegressiveWindow[];
  /** Das Zeitfenster der Staffel des § 7 Abs. 2a EStG für E-Fahrzeuge. */
  electricVehicleWindows: ElectricVehicleWindow[];
  /** Höchstsatz der Sonderabschreibung in Promille (§ 7g Abs. 5 EStG). */
  specialMaxPermille: number;
  /** Begünstigungszeitraum in Jahren: das Anschaffungsjahr und die vier folgenden. */
  specialPeriodYears: number;
  /**
   * Warum der Investitionsabzugsbetrag des § 7g Abs. 1 EStG in Buchfink nicht
   * vorkommt: er wird außerhalb der Bilanz vorgenommen.
   */
  investmentDeductionNote: string;
  methods: AssetMethodInfo[];
}

export interface DepreciationDue {
  assetId: number;
  inventoryNumber: string;
  name: string;
  account: string;
  expenseAccount: string;
  method: string;
  rateLabel: string;
  months: number;
  planned: Cents;
  booked: Cents;
  due: Cents;
  specialAccount?: string;
  specialPlanned: Cents;
  specialBooked: Cents;
  specialDue: Cents;
  bookValueBefore: Cents;
  bookValueAfter: Cents;
  note?: string;
}

export interface DepreciationRun {
  fiscalYear: number;
  bookingDate: string;
  due: DepreciationDue[];
  total: Cents;
  missingPriorYears?: number[];
}

export interface DepreciationResult {
  entries: JournalEntry[];
  /** Summe der gebuchten Abschreibung, ohne die Sonderabschreibung. */
  total: Cents;
  skipped?: string[];
  /**
   * Anlagegüter, bei denen der Lauf nur einen steuerlichen Wert festgehalten
   * hat: die Sonderabschreibung des § 7g Abs. 5 EStG wird seit dem BilMoG
   * nicht mehr in der Handelsbilanz gebucht.
   */
  taxOnly?: string[];
  taxOnlyTotal?: Cents;
}

export interface DisposalAccounts {
  revenue?: string;
  bookValue: string;
  explanation: string;
}

export interface DisposalRequest {
  assetId: number;
  date: string;
  kind: DisposalKind;
  proceeds: Cents;
  /** Teil der Anschaffungskosten, der abgeht. Leer = alles. Nur bei Finanzanlagen. */
  costShare?: Cents;
  /** Derselbe Teilabgang in Stück. Hat Vorrang vor costShare. */
  quantity?: Units;
  taxTreatment?: TaxTreatment;
  taxRate?: TaxRate;
  settlement: Settlement;
  paymentAccount?: string;
  contactId?: number;
  note?: string;
}

export interface DisposalPreview {
  catchUpAmount: Cents;
  /** Im Abgangsjahr noch offene Sonderabschreibung, mit demselben Beleg nachgeholt. */
  specialCatchUp: Cents;
  catchUpLines?: JournalLine[];
  partial: boolean;
  costShare: Cents;
  /** Abgehende Stückzahl und der Bestand danach. */
  quantityShare?: Units;
  unitsRemaining?: Units;
  depreciationShare: Cents;
  bookValue: Cents;
  /** Buchgewinn positiv, Buchverlust negativ. */
  result: Cents;
  isGain: boolean;
  accounts: DisposalAccounts;
  lines: JournalLine[];
  gross: Cents;
  tax: Cents;
  /** Steuerliche Nebenrechnung eines Investmentanteils — sie ändert die Buchung nicht. */
  investment?: InvestmentTaxNote;
  warnings?: string[];
}

/** Die Teilfreistellung eines Fonds für einen Anleger (§ 20 InvStG). */
export interface PartialExemption {
  /** Steuerfreier Anteil in Promille: 800 sind 80 %. */
  permille: number;
  determined: boolean;
  source: string;
  explanation: string;
}

/** Was das InvStG neben der Buchung aus einem Betrag macht. */
export interface InvestmentTaxNote {
  fundClass: FundClass;
  fundClassLabel: string;
  exemption: PartialExemption;
  /** Steht, wenn sich kein Satz bestimmen lässt — und sagt warum. */
  exemptionError?: string;
  grossAmount: Cents;
  vorabpauschalen: Cents;
  exemptAmount: Cents;
  taxableAmount: Cents;
  explanation: string;
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

export interface InvestmentRules {
  fundClasses: { class: FundClass; label: string }[];
  investorTypes: { type: InvestorType; label: string }[];
  investorType: InvestorType;
  investorLabel: string;
  /** Woher die Anlegerstellung kommt: aus der Rechtsform oder aus einer Festlegung. */
  investorReason: string;
  legalForm: string;
  exemptions: {
    class: FundClass;
    label: string;
    permille: number;
    source?: string;
    explanation?: string;
    problem?: string;
  }[];
}

export interface DisposalResult {
  catchUpEntry?: JournalEntry;
  disposalEntry?: JournalEntry;
  asset: FixedAsset;
  message: string;
}

/** Was der Devisenkassamittelkurs eines Stichtags für eine Finanzanlage bedeutet. */
export interface CurrencyValuation {
  currency: string;
  foreignAmount: Cents;
  /** Anschaffungskurs, abgeleitet aus Fremdbetrag und Euro-Anschaffungskosten. */
  acquisitionRate: number;
  ratePerEuro: number;
  valueAtRate: Cents;
  bookValue: Cents;
  /** Greift § 256a Satz 2 HGB — Restlaufzeit höchstens ein Jahr, kein Deckel nach oben? */
  shortTerm: boolean;
  /** Unterschied zum Buchwert: negativ, wo der Kurs gefallen ist. */
  difference: Cents;
  /** Was daraus folgt — und mit welchem Betrag er tatsächlich gebucht werden dürfte. */
  proposal: 'impairment' | 'write_up' | 'none';
  proposedAmount: Cents;
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

export interface Foundation {
  id: number;
  /** Tag der notariellen Beurkundung. Mit ihm entsteht die Vorgesellschaft. */
  notarizedOn: string;
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

/** Eine Pflicht aus der Gründung, mit Frist und Erledigung. */
export interface FoundationDuty {
  key: string;
  title: string;
  /** Leer, wo das Gesetz „unverzüglich" sagt statt einer Tagesfrist. */
  dueDate: string;
  deadline: string;
  reference: string;
  description: string;
  doneOn: string;
  isDone: boolean;
}

/** Ein Buchungsvorschlag der Gründung, vor der Freigabe. */
export interface FoundationPosting {
  title: string;
  date: string;
  description: string;
  reference: string;
  lines: JournalLine[];
  amount: Cents;
}

export interface FoundationPostingPreview {
  postings: FoundationPosting[];
  total: Cents;
  alreadyBooked: boolean;
  /** Was nicht gebucht wird, und warum. */
  skipped: string[];
}

/** Alles, was die Gründungsansicht braucht — in einem Aufruf. */
export interface FoundationState {
  /** Falsch, wenn die Rechtsform keine Kapitalgesellschaft ist. */
  applies: boolean;
  hasFoundation: boolean;
  legalForm: string;
  rules: FoundationRules;
  foundation?: Foundation;
  stage: FoundationStage;
  anmeldung?: AnmeldungCheck;
  unterbilanz?: Unterbilanz;
  duties: FoundationDuty[];
  postingsBooked: boolean;
}

// -------------------------------------------------------------
// Jahresabschluss: Geschäftsjahr, Saldenvortrag, Abschlussstand
// (internal/domain/fiscalyear.go, internal/service/closing_service.go)
// -------------------------------------------------------------

/**
 * Die vier Stände sind keine Abstufungen derselben Sache, sondern Vorgänge mit
 * verschiedenen Beteiligten: Aufstellung durch die Geschäftsführung (§ 242,
 * § 264 Abs. 1 HGB), Feststellung durch die Gesellschafter (§ 42a Abs. 2
 * GmbHG), Offenlegung gegenüber dem Bundesanzeiger (§ 325 HGB).
 */
export type FiscalYearStatus = 'open' | 'prepared' | 'adopted' | 'disclosed';

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
  /** Zeitpunkt des letzten Saldenvortrags in dieses Jahr. */
  carriedForwardAt?: string;
  /**
   * Die durchschnittliche Zahl der Arbeitnehmer ist das dritte Merkmal des
   * § 267 Abs. 1 HGB. Aus der Buchführung lässt sie sich nicht ableiten,
   * deshalb wird sie erfasst und nicht gerechnet.
   */
  averageEmployees: number;
  /**
   * Der Gesamtumsatz des vorangegangenen Kalenderjahres. An ihm hängt die
   * Übergangsfrist des § 27 Abs. 38 Nr. 2 UStG: bis 800.000 € darf 2027 noch
   * eine sonstige Rechnung ausgestellt werden. Null heißt „nicht erfasst".
   */
  priorYearRevenue: Cents;
  createdAt: string;
}

/** Alles, was die Abschlussansicht eines Jahres braucht — in einem Aufruf. */
export interface ClosingState {
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
  /** Das Vorjahr selbst trägt keinen Saldenvortrag, obwohl es einen bräuchte. */
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
//  internal/ebilanz/mapping.go)
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
}

/** Die Pflichtangaben des § 264 Abs. 1a HGB im Kopf des Abschlusses. */
export interface StatementHeader {
  companyName: string;
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
  verified: boolean;
  /** Benennt, was fehlt, wenn etwas fehlt. */
  finding?: string;
}

/** Der Zuordnungsbericht vor dem E-Bilanz-Export. */
export interface MappingReport {
  fiscalYear: number;
  taxonomyVersion: string;
  taxonomyDate: string;
  taxonomyNote: string;
  rows: MappingRow[];
  /** Konten ohne Gliederungsposition oder ohne Taxonomie-Element. */
  blocking: MappingRow[];
  fallbacks: FallbackCount[];
  /** Elemente, deren Name noch gegen die amtliche Taxonomie zu prüfen ist. */
  unverified: number;
  canExport: boolean;
}

// -------------------------------------------------------------------------
// Umsatzsteuer-Voranmeldung, Zusammenfassende Meldung, Prüfläufe
// -------------------------------------------------------------------------

/** Länge eines Voranmeldungszeitraums (§ 18 Abs. 2 UStG). */
export type VatPeriodType = 'month' | 'quarter' | 'year';

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

/** Ein Zeitraum mit Fälligkeit, Festschreibungsstand und Stand der Anmeldung. */
export interface VatPeriodStatus extends VatPeriod {
  dueDate: string;
  status: VatReturnStatus;
  returnId?: number;
  /** Ohne Festschreibung ist die Bestätigung der Übermittlung gesperrt. */
  committed: boolean;
  payable: Cents;
  submittedAt?: string;
  isOverdue: boolean;
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

/** Eine Anmeldung des Vorjahres im Vorschlag zur Sondervorauszahlung. */
export interface SpecialPrepaymentPeriod {
  periodKey: string;
  periodLabel: string;
  returnId: number;
  submittedAt: string;
  prepayment: Cents;
}

/** Ein Elftel der Vorauszahlungen des Vorjahres (§ 47 Abs. 1 UStDV). */
export interface SpecialPrepaymentSuggestion {
  year: number;
  basedOnYear: number;
  amount: Cents;
  prepaymentSum: Cents;
  periods: SpecialPrepaymentPeriod[];
  /** Liegt für jeden Zeitraum des Vorjahres eine übermittelte Anmeldung vor? */
  complete: boolean;
  account: string;
  note: string;
}

/** L: ig. Lieferung, S: sonstige Leistung § 3a Abs. 2, D: Dreiecksgeschäft. */
export type ZMLineKind = 'L' | 'S' | 'D';

/** Eine Meldezeile: je USt-IdNr. und Art ein Betrag. */
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
  kind: ZMLineKind;
  amount: Cents;
}

/**
 * Die Abstimmung gegen die Kennziffern 41 und 21 der Voranmeldungen. Sie wird
 * nicht gespeichert: sie beschreibt den heutigen Stand beider Meldungen.
 */
export interface ZMReconciliation {
  scopeKey?: string;
  scopeLabel?: string;
  suppliesZm: Cents;
  suppliesVat: Cents;
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
  totalSupplies: Cents;
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

/** Ein Prüflauf über einen Zeitraum bis zu einem Stichtag (GoBD Rz. 34 ff.). */
export interface CheckRun {
  id: number;
  fiscalYear: number;
  cutoffDate: string;
  periodType?: string;
  checkedEntries: number;
  checkedReceipts: number;
  checkedBankTx: number;
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
  /** Mitgegebene Originaldateien. */
  receiptFiles: number;
  documentFiles: number;
  /** Hinweise, die der Export nicht selbst beheben kann. */
  notes: string[];
}

/** Eine Zeile des Schlüsselverzeichnisses (GoBD Rz. 95). */
export interface KeyDirectoryEntry {
  category: string;
  key: string;
  label: string;
  description: string;
}

export interface FileCheckIssue {
  /** `receipt` für eine Belegdatei, `document` für ein Anlagendokument. */
  kind: string;
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
// Abschlussbausteine: Schritte, Rechnungsabgrenzung, Rückstellungen, Vorräte,
// Umsatzsteuer-Verrechnung, Steuerrückstellung, Ergebnisverwendung, Anhang
// -------------------------------------------------------------------------

/** Die Bausteine des Jahresabschlusses in ihrer fachlichen Reihenfolge. */
export type ClosingStepKey =
  | 'depreciation'
  /** Wertaufholung prüfen: ist der Grund einer Abwertung weggefallen (§ 253 Abs. 5 HGB)? */
  | 'write_up'
  /** Fremdwährungsbewertung zum Stichtag (§ 256a HGB). */
  | 'currency_valuation'
  | 'accruals'
  | 'provisions'
  | 'inventory'
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
 * Sie stehen getrennt von `CompanySettings`, weil sie nicht den Rechtsträger
 * beschreiben, sondern die Buchführung: der Hebesatz gehört zur Gemeinde, die
 * Abgrenzungsmethode zur Art, wie abgegrenzt wird.
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
 * Eine Auflösung aus der Vorschau. Sie trägt keine Kennung, weil sie noch
 * nicht existiert — deshalb ein eigener Typ und nicht `AccrualRelease` mit
 * optionalen Feldern, die in der Vorschau nie gesetzt sind.
 */
export interface AccrualReleasePlanItem {
  fiscalYear: number;
  date: string;
  amount: Cents;
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

/** Eine Buchung, deren Leistung über den Bilanzstichtag hinausreicht. */
export interface AccrualProposalItem {
  entryId: number;
  entryNumber: string;
  bookingDate: string;
  description: string;
  kind: AccrualKind;
  account: string;
  accountName: string;
  serviceFrom: string;
  serviceTo: string;
  totalAmount: Cents;
  deferredAmount: Cents;
  /** Unter der Vorschlagsschwelle; angezeigt wird der Posten trotzdem. */
  belowThreshold: boolean;
  alreadyBooked: boolean;
}

export interface AccrualProposal {
  fiscalYear: number;
  cutoff: string;
  method: AccrualMethod;
  threshold: Cents;
  items: AccrualProposalItem[];
  /** Was die Schwelle bedeutet und was nicht. */
  note: string;
}

export interface AccrualRequest {
  fiscalYear: number;
  kind: AccrualKind;
  sourceEntryId?: number;
  text: string;
  totalAmount: Cents;
  startDate: string;
  endDate: string;
  account: string;
  /** Überschreibt den gerechneten Anteil. Null heißt: rechnen. */
  deferredAmount?: Cents;
}

export interface AccrualPreview {
  accrual: Accrual;
  lines: JournalLine[];
  releases: AccrualReleasePlanItem[];
  bookingDate: string;
  explanation: string;
  warnings: string[];
}

export interface AccrualReportRow {
  accrualId: number;
  kind: AccrualKind;
  kindLabel: string;
  text: string;
  account: string;
  startDate: string;
  endDate: string;
  deferredAmount: Cents;
  released: Cents;
  remaining: Cents;
  /** Restlaufzeit ab dem Stichtag in Kalendertagen. */
  remainingDays: number;
}

export interface AccrualReport {
  cutoff: string;
  rows: AccrualReportRow[];
  totalActive: Cents;
  totalPassive: Cents;
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

/** Ein Satz der Abzinsungszinssatzverordnung, wie ihn die Bundesbank meldet. */
export interface DiscountRate {
  /** Monat der Veröffentlichung als JJJJ-MM. */
  month: string;
  /** Restlaufzeit in Jahren (1 bis 50). */
  years: number;
  /** Zinssatz in Millionsteln: 1,50 % sind 15000. */
  rateMicros: number;
  /** Mittelungsdauer: sieben Jahre, für Altersversorgung zehn. */
  average: number;
  updatedAt?: string;
}

export interface ProvisionRequest {
  /** Bei einer Zuführung gesetzt, bei der Bildung leer. */
  provisionId?: number;
  fiscalYear: number;
  kind: ProvisionKind;
  text: string;
  amount: Cents;
  expectedOn: string;
  reason: string;
  /** Überschreiben den Vorschlag aus der Art. */
  balanceAccount?: string;
  expenseAccount?: string;
  /** Leer heißt Bilanzstichtag. */
  date?: string;
}

export interface ProvisionPreview {
  provision: Provision;
  lines: JournalLine[];
  settlementAmount: Cents;
  /** Der gebuchte Betrag — bei Abzinsung der Barwert. */
  amount: Cents;
  discounted: boolean;
  discountYears: number;
  discountRate?: string;
  /** Monat der Zinstabelle, mit der gerechnet wurde. */
  discountMonth?: string;
  /** Der steuerliche Wert (5,5 %, § 6 Abs. 1 Nr. 3a EStG); nicht gebucht. */
  taxAmount: Cents;
  bookingDate: string;
  bookingYear: number;
  explanation: string;
  findings: string[];
  isIncrease: boolean;
}

export interface ProvisionChangeRequest {
  provisionId: number;
  amount: Cents;
  date: string;
  reason: string;
  /** Nimmt beim Verbrauch ohne Rechnung die Zahlung auf. */
  paymentAccount?: string;
}

/** Der Inventurwert eines Vorratskontos zum Bilanzstichtag (§ 240 HGB). */
export interface InventoryCount {
  id: number;
  fiscalYear: number;
  account: string;
  amount: Cents;
  /** Buchwert vor der Abschlussbuchung. */
  bookValue: Cents;
  countedOn: string;
  method: string;
  /** Die Inventurliste im Belegspeicher — Pflicht. */
  receiptId?: number;
  journalEntryId?: number;
  createdAt: string;
}

export interface InventoryAccount {
  account: string;
  accountName: string;
  group: string;
  /** Gegenkonto der Bestandsveränderung. */
  changeAccount: string;
  changeAccountName: string;
  bookValue: Cents;
  counted: Cents;
  countedAt?: string;
  booked: boolean;
}

export interface InventoryOverview {
  fiscalYear: number;
  cutoff: string;
  accounts: InventoryAccount[];
  note: string;
}

export interface InventoryRequest {
  fiscalYear: number;
  account: string;
  amount: Cents;
  countedOn: string;
  method: string;
  /** Die Inventurliste im Belegspeicher — Pflicht. */
  receiptId: number;
}

export interface InventoryPreview {
  account: string;
  accountName: string;
  changeAccount: string;
  bookValue: Cents;
  counted: Cents;
  change: Cents;
  lines: JournalLine[];
  bookingDate: string;
  explanation: string;
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

export interface TaxProvisionRequest {
  fiscalYear: number;
  incomeProvision: Cents;
  tradeProvision: Cents;
  reason: string;
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
  /** Die Pflichtrücklage der UG (§ 5a Abs. 3 GmbHG); sonst null. */
  requiredLegalReserve: Cents;
  explanation: string;
  warnings: string[];
}

/** Ein Abschnitt des Anhangs. */
export type NotesSection =
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

/** Eine Sonderabschreibung, die noch als Buchung im Journal steht. */
export interface LegacySpecialDepreciation {
  assetId: number;
  inventoryNumber: string;
  name: string;
  fiscalYear: number;
  date: string;
  amount: Cents;
  expenseAccount: string;
  entryNumber?: string;
}

export interface LegacySpecialDepreciationNotice {
  rows: LegacySpecialDepreciation[];
  total: Cents;
  note: string;
}

// -------------------------------------------------------------------------
// Steuerliche Nebenpflichten (Welle 5c): Vorsteuerberichtigung § 15a UStG,
// Bestätigung der USt-IdNr., Belegnachweis der ig. Lieferung, nicht abziehbare
// Betriebsausgaben, Fremdwährung, Anlagen
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

export interface RegisterInputTaxRequest {
  assetId?: number;
  receiptId?: number;
  entryId?: number;
  label: string;
  account?: string;
  acquisitionDate: string;
  netAmount: Cents;
  inputTaxAmount: Cents;
  /** Null wird als volle Verwendung gelesen. */
  originalPermille?: number;
  immovable?: boolean;
  note?: string;
}

export interface SaveInputTaxUsageRequest {
  correctionId: number;
  fiscalYear: number;
  permille: number;
  reason?: string;
}

/**
 * Das Ergebnis einer Bestätigungsanfrage. „unavailable" ist kein negatives
 * Ergebnis, sondern gar keins — die beiden dürfen nicht dasselbe bedeuten.
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
  /** Zeitpunkt der Abfrage als RFC3339; an ihm hängt die Frist. */
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
  /** Wohin gefragt wurde — die Adresse ist eine Einstellung. */
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

/** Die Belegarten des Nachweises einer ig. Lieferung. */
export type EvidenceKind =
  | 'cmr_frachtbrief'
  | 'konnossement'
  | 'luftfrachtrechnung'
  | 'spediteurbescheinigung'
  | 'versicherungspolice'
  | 'bankbeleg'
  | 'behoerdliche_bestaetigung'
  | 'lagerbescheinigung'
  | 'gelangensbestaetigung'
  | 'rechnungsdoppel'
  | 'tracking_protokoll'
  | 'sonstiges';

/**
 * Die Systematik des Art. 45a MwStVO: „a" sind Beförderungsbelege, „b" die
 * sonstigen Belege, "" trägt die Vermutung nicht.
 */
export type EvidenceGroup = 'a' | 'b' | '';

/** Wer den Gegenstand befördert hat. Leer heißt Regelfall — der Lieferer. */
export type TransportKind = 'supplier' | 'customer' | '';

export interface EvidenceKindInfo {
  kind: EvidenceKind;
  label: string;
  group: EvidenceGroup;
  hint?: string;
}

/** Ob der Belegnachweis trägt, und woran es sonst liegt. */
export interface EvidenceStatus {
  fulfilled: boolean;
  /** Die Vorschrift, auf die sich das Ergebnis stützt. */
  basis?: string;
  reason: string;
  /** Was noch fehlt; leer, wenn nichts fehlt. */
  missing: string[];
  groupACount: number;
  groupBCount: number;
}

/** Ein Nachweisbeleg an einer Rechnung. */
export interface SupplyEvidence {
  id: number;
  invoiceId: number;
  kind: EvidenceKind;
  /** Zwei Belege desselben Ausstellers sind ein Beleg mit zwei Blättern. */
  issuer: string;
  /** Weder Lieferer noch Erwerber — die Bedingung des Art. 45a MwStVO. */
  independent: boolean;
  date: string;
  /** Der Beleg, unter dem die Datei im Belegspeicher liegt. */
  receiptId?: number;
  note?: string;
  createdAt: string;
}

export interface SupplyEvidenceView {
  invoiceId: number;
  invoiceNumber: string;
  date: string;
  contactName: string;
  transport: TransportKind;
  items: SupplyEvidence[];
  status: EvidenceStatus;
  kinds: EvidenceKindInfo[];
}

export interface SupplyEvidenceRequest {
  invoiceId: number;
  kind: EvidenceKind;
  issuer: string;
  independent: boolean;
  date: string;
  receiptId?: number;
  note?: string;
  /** Leer heißt: keine Änderung an dem, was an der Rechnung steht. */
  transport?: TransportKind;
  /** Die Datei auf der Platte; sie geht in den Belegspeicher wie jede andere. */
  filePath?: string;
}

export interface SupplyEvidenceReportRow {
  invoiceId: number;
  invoiceNumber: string;
  date: string;
  contactName: string;
  netAmount: Cents;
  evidenceCount: number;
  transport: TransportKind;
  status: EvidenceStatus;
}

export interface SupplyEvidenceReport {
  fiscalYear: number;
  rows: SupplyEvidenceReportRow[];
  /** Zahl der Lieferungen ohne vollständigen Nachweis. */
  incomplete: number;
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

export interface RebookGiftsRequest {
  fiscalYear: number;
  recipientKey: string;
  /** Leer heißt: der Tag der Korrektur. */
  date?: string;
  /** Pflicht: eine Umbuchung ohne Grund ist im Journal später nicht erklärbar. */
  reason: string;
}

export interface GiftRebooking {
  recipientName: string;
  /** Die Buchungsnummern der Stornos und der Neubuchungen. */
  reversals: string[];
  rebookings: string[];
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

export interface VatRateImport {
  imported: number;
  skipped: number;
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

/** Ein Satz Wertgrenzen aus der Ressource `afa_rules.json`. */
export interface AfaParameterSet {
  validFrom: string;
  note?: string;
  gwgImmediateLimit: Cents;
  gwgRecordThreshold: Cents;
  poolLowerLimit: Cents;
  poolUpperLimit: Cents;
  poolYears: number;
}

/** Das Zeitfenster der Staffel für E-Fahrzeuge (§ 7 Abs. 2a EStG). */
export interface ElectricVehicleWindow {
  from: string;
  until: string;
  /** Die Sätze in Promille der Anschaffungskosten, Jahr für Jahr. */
  permillePerYear: number[];
  source: string;
  note: string;
}

/** Ein fester Gebäudesatz (§ 7 Abs. 4 EStG). */
export interface BuildingRate {
  key: string;
  residential: boolean;
  /** Leer heißt: kein unteres Ende — der Eintrag fängt die früheren Fälle auf. */
  referenceFrom?: string;
  permille: number;
  source: string;
  label: string;
  note?: string;
}

/** Die Abschreibungsregeln aus der Ressource — dieselbe Datei, aus der gerechnet wird. */
export interface AfaRules {
  version: string;
  source: string;
  note: string;
  investmentDeductionNote: string;
  parameterSets: AfaParameterSet[];
  degressiveWindows: DegressiveWindow[];
  electricVehicleWindows: ElectricVehicleWindow[];
  buildingRates: BuildingRate[];
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

export interface PoolConsistencyRow {
  assetId: number;
  inventoryNumber: string;
  name: string;
  acquisitionDate: string;
  cost: Cents;
  method: DepreciationMethod;
}

/** Die Einheitlichkeit des Wahlrechts nach § 6 Abs. 2a Satz 5 EStG. */
export interface PoolConsistencyReport {
  fiscalYear: number;
  lowerLimit: Cents;
  upperLimit: Cents;
  pooled: PoolConsistencyRow[];
  immediate: PoolConsistencyRow[];
  consistent: boolean;
  note: string;
}

export interface CapitalizeNearAcquisitionCostRequest {
  assetId: number;
  /** Leer heißt: das Ende des Dreijahreszeitraums. */
  date?: string;
  /** Pflicht: die Umbuchung verteilt einen sofort abgezogenen Aufwand. */
  reason: string;
}

/** Eine Freistellungsbescheinigung § 48b EStG, die abläuft oder abgelaufen ist. */
export interface ExemptionCertificateWarning {
  contactId: number;
  name: string;
  number: string;
  validUntil: string;
  /** „expiring" oder „expired". */
  state: string;
  note: string;
}

/**
 * Die Adressen der beiden Netzdienste: Bundeszentralamt für Steuern und
 * Kursdienst. Ein leerer Wert bedeutet die jeweilige Voreinstellung.
 */
export interface ServiceEndpoints {
  vatIdEndpoint: string;
  vatIdDefault: string;
  exchangeRateEndpoint: string;
  exchangeRateDefault: string;
}
