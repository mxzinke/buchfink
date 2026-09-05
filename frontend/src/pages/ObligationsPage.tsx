import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Archive, CheckCircle2, CircleAlert, Download, Plus, RefreshCw, Trash2 } from 'lucide-react';
import { Api } from '../services/api';
import { useWriteLock } from '../components/WriteLock';
import type { NavigateFn } from '../components/Sidebar';
import type {
  AfaRules,
  Contact,
  EvidenceKind,
  EvidenceKindInfo,
  EvidenceStatus,
  ExchangeRate,
  ExemptionCertificateWarning,
  ForeignCurrencyValuation,
  GiftRecipientRow,
  InputTaxCorrectionRow,
  InputTaxCorrectionYear,
  NonDeductibleReport,
  PoolConsistencyReport,
  ServiceEndpoints,
  SupplyEvidenceReport,
  SupplyEvidenceView,
  TransportKind,
  VatExchangeRate,
  VatIDCheck,
  VatIDCheckStatus,
  VatIDFieldResult,
  VatIDStatus,
  WriteUpReport,
} from '../types';
import {
  formatCents,
  formatCentsPlain,
  formatDate,
  formatDateTime,
  formatExchangeRate,
  formatPermille,
  parseCents,
  parseExchangeRate,
} from '../utils/formatters';
import {
  Button,
  Checkbox,
  Dialog,
  EmptyState,
  Field,
  FieldRow,
  FieldValue,
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
  TabPanel,
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
 * Die steuerlichen Nebenpflichten (Welle 5c).
 *
 * Sechs Pflichten, die neben der Buchung stehen und keine eigene Buchung sind:
 * das Verzeichnis nach § 15a UStG, die Bestätigung der USt-IdNr., der
 * Belegnachweis der innergemeinschaftlichen Lieferung, die Aufzeichnung der
 * beschränkt abziehbaren Betriebsausgaben, die Kurse der Fremdwährung und die
 * beiden Anlagenberichte zu Wertaufholung und Sammelposten.
 *
 * Sie stehen in einer Ansicht und nicht verteilt über sechs, weil sie eine
 * gemeinsame Eigenschaft haben: Wer sie versäumt, merkt es nicht beim Buchen,
 * sondern in der Prüfung. Eine Liste, die man ansehen kann, ist die einzige
 * Stelle, an der sie überhaupt auffallen.
 */

type TabKey = 'inputtax' | 'vatid' | 'evidence' | 'nondeductible' | 'currency' | 'assets';

export interface ObligationsPageProps {
  /** Das Geschäftsjahr aus der Kopfzeile. */
  year: number;
  /**
   * Der Reiter, mit dem die Seite öffnet. Aus der Schrittliste des Abschlusses
   * und vom Kontakt führt ein Verweis hierher; ohne dieses Ziel landete er auf
   * dem ersten Reiter, und die benannte Arbeit wäre wieder zu suchen.
   */
  initialTab?: string;
  /**
   * Rechnung, deren Nachweisbelege der Reiter „Belegnachweis" gleich aufschlägt.
   * Der Weg von der Rechnung hierher meint eine bestimmte Lieferung; ohne sie
   * müsste der Anwender sie in der Liste erneut suchen.
   */
  initialInvoiceId?: number;
  onNavigate?: NavigateFn;
}

const TAB_KEYS: TabKey[] = [
  'inputtax',
  'vatid',
  'evidence',
  'nondeductible',
  'currency',
  'assets',
];

function isTabKey(value: string | undefined): value is TabKey {
  return value !== undefined && (TAB_KEYS as string[]).includes(value);
}

function message(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

/** Der heutige Tag als ISO-Datum — der Vorschlag jedes Datumsfeldes hier. */
function today(): string {
  return new Date().toISOString().slice(0, 10);
}

export const ObligationsPage: React.FC<ObligationsPageProps> = ({
  year,
  initialTab,
  initialInvoiceId,
  onNavigate,
}) => {
  const [tab, setTab] = useState<TabKey>(isTabKey(initialTab) ? initialTab : 'inputtax');

  // Ein zweiter Verweis auf dieselbe Seite soll den Reiter wechseln und nicht
  // ins Leere gehen: die Seite bleibt montiert, wenn nur der Parameter wechselt.
  useEffect(() => {
    if (isTabKey(initialTab)) setTab(initialTab);
  }, [initialTab]);

  return (
    <div className="max-w-[1200px] mx-auto px-8 py-8">
      <PageHeader
        title="Steuerliche Nebenpflichten"
        context={`Geschäftsjahr ${year} · Verzeichnisse, Nachweise und Kurse`}
      />

      <Tabs<TabKey>
        value={tab}
        onValueChange={setTab}
        className="mt-6"
        items={[
          { value: 'inputtax', label: 'Vorsteuer § 15a' },
          { value: 'vatid', label: 'USt-IdNr.' },
          { value: 'evidence', label: 'Belegnachweis' },
          { value: 'nondeductible', label: 'Nicht abziehbar' },
          { value: 'currency', label: 'Fremdwährung' },
          { value: 'assets', label: 'Anlagen' },
        ]}
      >
        {/* Jede Sicht lädt für sich. Alle sechs auf einmal zu holen hieße, für
            eine Frage sechs Auswertungen zu rechnen. */}
        <TabPanel value="inputtax">
          {tab === 'inputtax' && <InputTaxPanel year={year} />}
        </TabPanel>
        <TabPanel value="vatid">{tab === 'vatid' && <VatIDPanel onNavigate={onNavigate} />}</TabPanel>
        <TabPanel value="evidence">
          {tab === 'evidence' && <EvidencePanel year={year} initialInvoiceId={initialInvoiceId} />}
        </TabPanel>
        <TabPanel value="nondeductible">
          {tab === 'nondeductible' && <NonDeductiblePanel year={year} />}
        </TabPanel>
        <TabPanel value="currency">{tab === 'currency' && <CurrencyPanel year={year} />}</TabPanel>
        <TabPanel value="assets">
          {tab === 'assets' && <AssetObligationsPanel year={year} />}
        </TabPanel>
      </Tabs>
    </div>
  );
};

// -------------------------------------------------------------------------
// Vorsteuerberichtigung nach § 15a UStG
// -------------------------------------------------------------------------

/**
 * Das Verzeichnis und der Jahreslauf.
 *
 * Der Verwendungsanteil je Wirtschaftsgut wird vorgelegt und nicht übernommen:
 * eine ungeprüfte Übernahme wäre die Behauptung, jemand habe hingesehen.
 * Deshalb bucht der Lauf erst, wenn kein Anteil mehr offen ist.
 */
const InputTaxPanel: React.FC<{ year: number }> = ({ year }) => {
  const lock = useWriteLock();
  const [view, setView] = useState<InputTaxCorrectionYear | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [drafts, setDrafts] = useState<Record<number, string>>({});
  const [closing, setClosing] = useState<InputTaxCorrectionRow | null>(null);
  const [closeReason, setCloseReason] = useState('');
  const [adding, setAdding] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      setView(await Api.getInputTaxCorrections(year));
    } catch (e) {
      setView(null);
      setError(message(e));
    } finally {
      setLoading(false);
    }
  }, [year]);

  useEffect(() => {
    void load();
    setDrafts({});
  }, [load]);

  const rows = view?.rows ?? [];
  const inPeriod = rows.filter((row) => row.inPeriod);
  // Das Backend lehnt die zweite Buchung ab; ein Knopf, der nur noch in einen
  // Fehler führt, ist keine Aktion mehr.
  const allBooked = inPeriod.length > 0 && inPeriod.every((row) => row.booked);

  async function saveUsage(row: InputTaxCorrectionRow) {
    const raw = drafts[row.correction.id] ?? String(row.permille);
    const permille = Number.parseInt(raw, 10);
    if (!Number.isFinite(permille) || permille < 0 || permille > 1000) {
      setError('Der Verwendungsanteil steht in Promille und liegt zwischen 0 und 1000.');
      return;
    }
    setBusy(true);
    setError('');
    try {
      setView(
        await Api.saveInputTaxUsage({
          correctionId: row.correction.id,
          fiscalYear: year,
          permille,
        }),
      );
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  async function book() {
    setBusy(true);
    setError('');
    try {
      const result = await Api.bookInputTaxCorrection(year);
      setView(result);
      toast.success(`Vorsteuerberichtigung gebucht: ${formatCents(result.totalAmount)}.`);
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  async function closeEntry() {
    if (!closing) return;
    if (!closeReason.trim()) {
      setError('Ohne Grund wird ein Eintrag des Verzeichnisses nicht abgeschlossen.');
      return;
    }
    setBusy(true);
    try {
      await Api.closeInputTaxCorrection(closing.correction.id, closeReason.trim(), today());
      setClosing(null);
      setCloseReason('');
      // Kein Toast: der geschlossene Dialog ist die Rückmeldung, der neue Stand
      // steht danach in der Tabelle (Gestaltungskonzept 8.5).
      await load();
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  if (loading) return <SkeletonRows rows={8} />;

  return (
    <>
      {error && <Notice tone="negative" text={error} className="mb-6" />}

      {view && view.unconfirmed > 0 && (
        <Notice
          className="mb-6"
          text={`${view.unconfirmed} Wirtschaftsgüter warten auf die Bestätigung ihres Verwendungsanteils; bis dahin wird nichts gebucht.`}
        />
      )}

      <StatRow>
        <Stat label="Im Verzeichnis" value={String(rows.length)} context="Wirtschaftsgüter" />
        <Stat
          label="Im Berichtigungszeitraum"
          value={String(inPeriod.length)}
          context={`Geschäftsjahr ${year}`}
        />
        <Stat
          label="Anteil offen"
          value={String(view?.unconfirmed ?? 0)}
          context="ohne Bestätigung"
          tone={(view?.unconfirmed ?? 0) > 0 ? 'negative' : 'positive'}
        />
        <Stat
          label="Berichtigung"
          value={formatCents(view?.totalAmount ?? 0)}
          context="Kennziffer 64"
          tone={(view?.totalAmount ?? 0) < 0 ? 'negative' : 'neutral'}
        />
      </StatRow>

      <Section
        title="Wirtschaftsgüter im Verzeichnis"
        context={`Buchungstag ${formatDate(view?.bookingDate ?? '')}`}
        className="mt-8"
        divider={false}
        action={
          <div className="flex items-center gap-2">
            <HelpPopover label="Erklärung zur Vorsteuerberichtigung">
              {view?.note ||
                'Ändert sich die Verwendung eines Wirtschaftsguts innerhalb des Berichtigungszeitraums, ist der Vorsteuerabzug anteilig zu berichtigen (§ 15a UStG). Die Bagatellgrenzen des § 44 UStDV nimmt Buchfink dabei selbst an.'}
            </HelpPopover>
            <Button
              variant="secondary"
              icon={<Plus className="w-4 h-4" strokeWidth={1.5} />}
              onClick={() => setAdding(true)}
              disabled={lock.locked}
              title={lock.hint}
            >
              Wirtschaftsgut aufnehmen
            </Button>
            <Button
              variant="primary"
              onClick={book}
              loading={busy}
              disabled={lock.locked || busy || allBooked || (view?.unconfirmed ?? 0) > 0}
              title={
                lock.locked
                  ? lock.hint
                  : allBooked
                    ? 'bereits gebucht — Storno über das Journal'
                    : (view?.unconfirmed ?? 0) > 0
                      ? 'Zuerst ist jeder Verwendungsanteil zu bestätigen'
                      : undefined
              }
            >
              Berichtigung buchen
            </Button>
          </div>
        }
      >
        {rows.length === 0 ? (
          <EmptyState
            title="Kein Wirtschaftsgut im Verzeichnis"
            description="Aufgenommen wird, wessen Vorsteuer die Grenze des § 44 Abs. 1 UStDV übersteigt."
          />
        ) : (
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th>Wirtschaftsgut</Th>
                <Th className="w-28">Anschaffung</Th>
                <Th numeric className="w-32">
                  Vorsteuer
                </Th>
                <Th numeric className="w-24">
                  Ursprung
                </Th>
                <Th className="w-24">Zeitraum</Th>
                <Th className="w-40">Anteil {year}</Th>
                <Th numeric className="w-32">
                  Berichtigung
                </Th>
                <Th className="w-40">Stand</Th>
              </Tr>
            </Thead>
            <Tbody>
              {rows.map((row) => {
                const id = row.correction.id;
                const status = row.booked ? 'gebucht' : row.confirmed ? 'zugeordnet' : 'offen';
                return (
                  <Tr key={id}>
                    <Td className="whitespace-normal">
                      {row.correction.label}
                      {row.correction.account && (
                        <span className="ml-2 code-num text-caption text-ink-muted">
                          {row.correction.account}
                        </span>
                      )}
                    </Td>
                    <Td>{formatDate(row.correction.acquisitionDate)}</Td>
                    <Td numeric>{formatCents(row.correction.inputTaxAmount)}</Td>
                    <Td numeric>{formatPermille(row.correction.originalPermille)}</Td>
                    <Td className="num">
                      {row.correction.firstFiscalYear}–{row.correction.lastFiscalYear}
                    </Td>
                    <Td>
                      {row.inPeriod && !row.booked ? (
                        <div className="flex items-center gap-2">
                          <Input
                            type="number"
                            min={0}
                            max={1000}
                            step={1}
                            align="right"
                            className="w-20"
                            aria-label={`Verwendungsanteil ${row.correction.label} in Promille`}
                            value={drafts[id] ?? String(row.permille)}
                            onChange={(e) =>
                              setDrafts((prev) => ({ ...prev, [id]: e.target.value }))
                            }
                          />
                          <Button
                            variant="secondary"
                            size="sm"
                            onClick={() => saveUsage(row)}
                            disabled={lock.locked || busy}
                            title={lock.hint}
                          >
                            Bestätigen
                          </Button>
                        </div>
                      ) : (
                        <span className="num">{formatPermille(row.permille)}</span>
                      )}
                    </Td>
                    <Td numeric>
                      {row.assessment.required ? formatCents(row.assessment.amount) : '—'}
                    </Td>
                    <Td>
                      <div className="flex items-center gap-1">
                        <StatusBadge status={status} />
                        <HelpPopover label={`Bewertung ${row.correction.label}`}>
                          {row.assessment.reason}
                        </HelpPopover>
                        {!row.correction.closedReason && (
                          <Button
                            variant="quiet"
                            size="sm"
                            iconOnly
                            title="Eintrag abschließen"
                            aria-label="Eintrag abschließen"
                            onClick={() => {
                              setClosing(row);
                              setCloseReason('');
                            }}
                            disabled={lock.locked}
                          >
                            {/* Kein Papierkorb: der Eintrag bleibt mit Grund im
                                Verzeichnis stehen, er wird nur nicht mehr
                                fortgeschrieben. */}
                            <Archive className="w-3.5 h-3.5" strokeWidth={1.5} />
                          </Button>
                        )}
                      </div>
                    </Td>
                  </Tr>
                );
              })}
            </Tbody>
          </Table>
        )}
      </Section>

      <Dialog
        open={Boolean(closing)}
        onOpenChange={(open) => !open && setClosing(null)}
        title="Eintrag abschließen"
        width="max-w-lg"
        footer={
          <>
            <Button variant="secondary" onClick={() => setClosing(null)}>
              Abbrechen
            </Button>
            <Button variant="primary" onClick={closeEntry} loading={busy} disabled={busy}>
              Abschließen
            </Button>
          </>
        }
      >
        <Field
          label="Grund"
          hint="Abgang, Entnahme oder Fehleintrag"
          explain="Ein abgeschlossener Eintrag wird in den Folgejahren nicht mehr berichtigt. Der Grund bleibt im Verzeichnis stehen, damit später erkennbar ist, warum der Zeitraum vorzeitig endete."
        >
          <Textarea
            value={closeReason}
            onChange={(e) => setCloseReason(e.target.value)}
            placeholder="Verkauft am 30.06., Eintrag endet mit dem Abgang"
          />
        </Field>
      </Dialog>

      <RegisterInputTaxDialog
        open={adding}
        onOpenChange={setAdding}
        onSaved={() => {
          setAdding(false);
          void load();
        }}
      />
    </>
  );
};

/** Ein Wirtschaftsgut von Hand ins Verzeichnis aufnehmen. */
const RegisterInputTaxDialog: React.FC<{
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSaved: () => void;
}> = ({ open, onOpenChange, onSaved }) => {
  const [label, setLabel] = useState('');
  const [account, setAccount] = useState('');
  const [date, setDate] = useState(today());
  const [net, setNet] = useState('');
  const [inputTax, setInputTax] = useState('');
  const [permille, setPermille] = useState('1000');
  const [immovable, setImmovable] = useState(false);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  async function save() {
    const netAmount = parseCents(net);
    const inputTaxAmount = parseCents(inputTax);
    if (!label.trim() || netAmount === null || inputTaxAmount === null) {
      setError('Bezeichnung, Nettobetrag und Vorsteuer gehören zu jedem Eintrag.');
      return;
    }
    setBusy(true);
    setError('');
    try {
      await Api.registerInputTaxCorrection({
        label: label.trim(),
        account: account.trim(),
        acquisitionDate: date,
        netAmount,
        inputTaxAmount,
        // `|| 1000` hätte hier einen ursprünglichen Anteil von 0 ‰ still zu
        // voller Verwendung gemacht — und 0 ‰ ist der klassische Fall der
        // Berichtigung nach oben: angeschafft ohne Abzugsberechtigung, später
        // für abzugsberechtigende Umsätze verwendet.
        originalPermille: Number.isNaN(Number.parseInt(permille, 10))
          ? 1000
          : Number.parseInt(permille, 10),
        immovable,
      });
      setLabel('');
      setAccount('');
      setNet('');
      setInputTax('');
      onSaved();
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title="Wirtschaftsgut aufnehmen"
      footer={
        <>
          <Button variant="secondary" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button variant="primary" onClick={save} loading={busy} disabled={busy}>
            Aufnehmen
          </Button>
        </>
      }
    >
      {error && <Notice tone="negative" text={error} className="mb-4" />}
      <div className="space-y-4">
        <Field
          label="Bezeichnung"
          explain="Aktivierte Anlagegüter nimmt Buchfink selbst auf. Von Hand kommt hier hinein, was kein Anlagegut ist und trotzdem ins Verzeichnis gehört — eine Großreparatur an einem Gebäude etwa (§ 15a Abs. 3 UStG)."
        >
          <Input value={label} onChange={(e) => setLabel(e.target.value)} />
        </Field>
        <FieldRow>
          <Field label="Konto" optional className="w-40" hint="Entscheidet über den Zeitraum">
            <Input
              value={account}
              onChange={(e) => setAccount(e.target.value)}
              className="code-num"
            />
          </Field>
          <Field label="Anschaffung" className="w-44">
            <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
          </Field>
          <Field label="Anteil in Promille" className="w-44">
            <Input
              type="number"
              min={0}
              max={1000}
              align="right"
              value={permille}
              onChange={(e) => setPermille(e.target.value)}
            />
          </Field>
        </FieldRow>
        <FieldRow>
          <Field label="Nettobetrag" className="w-48">
            <Input align="right" value={net} onChange={(e) => setNet(e.target.value)} />
          </Field>
          <Field label="Vorsteuer" className="w-48">
            <Input align="right" value={inputTax} onChange={(e) => setInputTax(e.target.value)} />
          </Field>
        </FieldRow>
        <Checkbox
          label="Grundstück oder Gebäude"
          hint="Zehn Jahre statt fünf (§ 15a UStG)"
          checked={immovable}
          onCheckedChange={(checked) => setImmovable(Boolean(checked))}
        />
      </div>
    </Dialog>
  );
};

// -------------------------------------------------------------------------
// Bestätigung der USt-IdNr.
// -------------------------------------------------------------------------

const VAT_ID_STATUS_LABEL: Record<VatIDCheckStatus, string> = {
  valid: 'gültig',
  invalid: 'ungültig',
  unavailable: 'nicht erreichbar',
};

const VAT_ID_FIELD_LABEL: Record<VatIDFieldResult, string> = {
  A: 'stimmt überein',
  B: 'stimmt nicht überein',
  C: 'nicht angefragt',
  D: 'vom Mitgliedstaat nicht mitgeteilt',
};

function fieldLabel(result?: VatIDFieldResult): string {
  return result ? VAT_ID_FIELD_LABEL[result] : '—';
}

/**
 * Die qualifizierte Bestätigungsanfrage nach § 18e UStG samt ihrem Verlauf.
 *
 * Der Verlauf steht neben dem aktuellen Stand, weil die Bestätigung ein Beleg
 * ist und kein Zustand: Was zählt, ist die Abfrage zum Zeitpunkt der Lieferung,
 * nicht die von heute.
 */
const VatIDPanel: React.FC<{ onNavigate?: NavigateFn }> = ({ onNavigate }) => {
  const lock = useWriteLock();
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [contactId, setContactId] = useState<number>(0);
  const [status, setStatus] = useState<VatIDStatus | null>(null);
  const [checks, setChecks] = useState<VatIDCheck[]>([]);
  const [warnings, setWarnings] = useState<ExemptionCertificateWarning[]>([]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    void (async () => {
      setLoading(true);
      try {
        const [list, expiring] = await Promise.all([
          Api.getContacts(),
          Api.getExemptionCertificateWarnings(today()),
        ]);
        setContacts(list);
        setWarnings(expiring);
        const first = list.find((contact) => Boolean(contact.vatId));
        if (first) setContactId(first.id);
      } catch (e) {
        setError(message(e));
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  const loadContact = useCallback(async (id: number) => {
    if (!id) {
      setStatus(null);
      setChecks([]);
      return;
    }
    try {
      const [state, history] = await Promise.all([Api.getVatIDStatus(id), Api.getVatIDChecks(id)]);
      setStatus(state);
      setChecks(history);
      setError('');
    } catch (e) {
      setStatus(null);
      setChecks([]);
      setError(message(e));
    }
  }, []);

  useEffect(() => {
    void loadContact(contactId);
  }, [contactId, loadContact]);

  async function runCheck() {
    setBusy(true);
    setError('');
    try {
      // Das Ergebnis steht nach dem Neuladen im Stand-Feld und in der Verlaufstabelle.
      await Api.checkVatID(contactId);
      await loadContact(contactId);
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  const withVatId = useMemo(
    () => contacts.filter((contact) => Boolean(contact.vatId)),
    [contacts],
  );

  if (loading) return <SkeletonRows rows={6} />;

  return (
    <>
      {error && <Notice tone="negative" text={error} className="mb-6" />}

      <Section
        title="Bestätigungsanfrage"
        context="Qualifizierte Abfrage beim Bundeszentralamt für Steuern"
        divider={false}
        action={
          <HelpPopover label="Erklärung zur Bestätigungsanfrage">
            {status?.note ||
              'Eine steuerfreie innergemeinschaftliche Lieferung setzt eine im Zeitpunkt der Lieferung gültige USt-IdNr. des Abnehmers voraus (§ 6a Abs. 1 Satz 1 Nr. 4 UStG). Buchfink hebt jede Antwort samt Abfrage-Identifikationsnummer auf; sie ist der Beleg gegenüber der Finanzverwaltung.'}
          </HelpPopover>
        }
      >
        {withVatId.length === 0 ? (
          <EmptyState
            title="Kein Geschäftspartner mit USt-IdNr."
            description="Abgefragt wird die Nummer, die am Geschäftspartner steht."
            action={
              onNavigate && (
                <Button variant="secondary" onClick={() => onNavigate('contacts')}>
                  Zu den Kontakten
                </Button>
              )
            }
          />
        ) : (
          <>
            <FieldRow>
              <Field label="Geschäftspartner" className="w-96">
                <Select<number>
                  value={contactId}
                  onValueChange={setContactId}
                  aria-label="Geschäftspartner der Bestätigungsanfrage"
                  items={withVatId.map((contact) => ({
                    value: contact.id,
                    label: `${contact.name} · ${contact.vatId}`,
                  }))}
                />
              </Field>
              <Field label="Stand" className="w-64">
                <FieldValue>
                  {status ? (
                    <span
                      className={cn(
                        'inline-flex items-center gap-1.5',
                        status.confirmed ? 'text-positive-text' : 'text-attention-text',
                      )}
                    >
                      {status.confirmed ? (
                        <CheckCircle2 className="w-4 h-4 shrink-0" strokeWidth={1.5} />
                      ) : (
                        <CircleAlert className="w-4 h-4 shrink-0" strokeWidth={1.5} />
                      )}
                      {status.confirmed
                        ? `bestätigt, gültig ${status.validityDays} Tage`
                        : 'keine gültige Bestätigung'}
                    </span>
                  ) : (
                    '—'
                  )}
                </FieldValue>
              </Field>
              <Field label="Abfrage" className="w-52">
                <Button
                  variant="primary"
                  icon={<RefreshCw className="w-4 h-4" strokeWidth={1.5} />}
                  onClick={runCheck}
                  loading={busy}
                  disabled={lock.locked || busy || !contactId}
                  title={lock.hint}
                >
                  Bestätigung abfragen
                </Button>
              </Field>
            </FieldRow>

            <div className="mt-6">
              {checks.length === 0 ? (
                <EmptyState
                  title="Noch keine Abfrage"
                  description="Die erste Abfrage legt den Beleg an, auf den sich die Steuerbefreiung stützt."
                />
              ) : (
                <Table density="kompakt">
                  <Thead>
                    <Tr>
                      <Th className="w-44">Zeitpunkt</Th>
                      <Th className="w-44">USt-IdNr.</Th>
                      <Th className="w-32">Ergebnis</Th>
                      <Th className="w-28">Code</Th>
                      <Th>Name</Th>
                      <Th>Ort</Th>
                      <Th>PLZ</Th>
                      <Th>Straße</Th>
                      <Th className="w-40">Abfrage-ID</Th>
                    </Tr>
                  </Thead>
                  <Tbody>
                    {checks.map((check) => (
                      <Tr key={check.id}>
                        <Td className="num">{formatDateTime(check.checkedAt)}</Td>
                        <Td code>{check.vatId}</Td>
                        <Td
                          className={
                            check.status === 'valid'
                              ? 'text-positive-text'
                              : check.status === 'invalid'
                                ? 'text-negative-text'
                                : 'text-attention-text'
                          }
                        >
                          {VAT_ID_STATUS_LABEL[check.status]}
                        </Td>
                        <Td code>{check.resultCode || '—'}</Td>
                        <Td className="text-ink-muted">{fieldLabel(check.nameResult)}</Td>
                        <Td className="text-ink-muted">{fieldLabel(check.cityResult)}</Td>
                        <Td className="text-ink-muted">{fieldLabel(check.postalCodeResult)}</Td>
                        <Td className="text-ink-muted">{fieldLabel(check.streetResult)}</Td>
                        <Td code>{check.requestId || '—'}</Td>
                      </Tr>
                    ))}
                  </Tbody>
                </Table>
              )}
            </div>
          </>
        )}
      </Section>

      <Section
        title="Freistellungsbescheinigungen"
        context={`${warnings.length} laufen ab oder sind abgelaufen`}
        action={
          <HelpPopover label="Erklärung zur Freistellungsbescheinigung">
            Wer eine Bauleistung bezieht, hat nach § 48 EStG 15 % der Gegenleistung einzubehalten
            — es sei denn, der Leistende legt eine gültige Freistellungsbescheinigung nach § 48b
            EStG vor. Den Steuerabzug selbst rechnet Buchfink nicht; die Bescheinigung wird
            geführt und ihre Frist überwacht.
          </HelpPopover>
        }
      >
        {warnings.length === 0 ? (
          <EmptyState
            title="Keine Bescheinigung läuft ab"
            description="Gemeldet wird 30 Tage vor dem letzten Gültigkeitstag."
          />
        ) : (
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th>Geschäftspartner</Th>
                <Th className="w-44">Nummer</Th>
                <Th className="w-32">Gültig bis</Th>
                <Th className="w-32">Stand</Th>
                <Th>Hinweis</Th>
              </Tr>
            </Thead>
            <Tbody>
              {warnings.map((warning) => (
                <Tr key={`${warning.contactId}-${warning.validUntil}`}>
                  <Td>{warning.name}</Td>
                  <Td code>{warning.number || '—'}</Td>
                  <Td className="num">{formatDate(warning.validUntil)}</Td>
                  <Td
                    className={
                      warning.state === 'expired' ? 'text-negative-text' : 'text-attention-text'
                    }
                  >
                    {warning.state === 'expired' ? 'abgelaufen' : 'läuft ab'}
                  </Td>
                  <Td className="text-ink-muted whitespace-normal">{warning.note}</Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>

      <EndpointSection field="vatId" title="Adresse des Bundeszentralamts" />
    </>
  );
};

// -------------------------------------------------------------------------
// Belegnachweis der innergemeinschaftlichen Lieferung
// -------------------------------------------------------------------------

const TRANSPORT_ITEMS: { value: TransportKind; label: string }[] = [
  { value: 'supplier', label: 'Beförderung durch den Lieferer' },
  { value: 'customer', label: 'Abholung durch den Abnehmer' },
];

/** Der Nachweisstand als Wort mit Zeichen — Farbe steht nie allein (§3.4). */
const EvidenceMark: React.FC<{ status: EvidenceStatus }> = ({ status }) => (
  <span
    className={cn(
      'inline-flex items-center gap-1.5',
      status.fulfilled ? 'text-positive-text' : 'text-attention-text',
    )}
  >
    {status.fulfilled ? (
      <CheckCircle2 className="w-3.5 h-3.5 shrink-0" strokeWidth={1.5} />
    ) : (
      <CircleAlert className="w-3.5 h-3.5 shrink-0" strokeWidth={1.5} />
    )}
    {status.fulfilled ? 'Nachweis vollständig' : 'Nachweis unvollständig'}
  </span>
);

const EvidencePanel: React.FC<{ year: number; initialInvoiceId?: number }> = ({
  year,
  initialInvoiceId,
}) => {
  const lock = useWriteLock();
  const [report, setReport] = useState<SupplyEvidenceReport | null>(null);
  const [kinds, setKinds] = useState<EvidenceKindInfo[]>([]);
  const [selected, setSelected] = useState<number>(0);
  const [view, setView] = useState<SupplyEvidenceView | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  // Die Felder des neuen Nachweisbelegs.
  const [kind, setKind] = useState<EvidenceKind>('cmr_frachtbrief');
  const [issuer, setIssuer] = useState('');
  const [independent, setIndependent] = useState(true);
  const [date, setDate] = useState(today());
  const [filePath, setFilePath] = useState('');

  const loadReport = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [rows, kindList] = await Promise.all([
        Api.getSupplyEvidenceReport(year),
        Api.getSupplyEvidenceKinds(),
      ]);
      setReport(rows);
      setKinds(kindList);
    } catch (e) {
      setReport(null);
      setError(message(e));
    } finally {
      setLoading(false);
    }
  }, [year]);

  useEffect(() => {
    void loadReport();
    setSelected(0);
    setView(null);
  }, [loadReport]);

  const open = useCallback(async (invoiceId: number) => {
    setSelected(invoiceId);
    try {
      setView(await Api.getSupplyEvidence(invoiceId));
      setError('');
    } catch (e) {
      setView(null);
      setError(message(e));
    }
  }, []);

  // Kommt der Anwender von einer bestimmten Rechnung, liegen deren Belege sofort
  // offen. Die Liste bleibt daneben stehen — der Weg zurück zu den anderen
  // Lieferungen ist damit nicht verstellt.
  useEffect(() => {
    if (initialInvoiceId) void open(initialInvoiceId);
  }, [initialInvoiceId, open]);

  // Die Beförderungsart wird gespeichert und nicht nur angezeigt: an ihr hängt
  // die Bewertung des Nachweises, und beim Abholfall die Gelangensbestätigung.
  // Eine Auswahl, die nach dem Neuladen wieder auf dem alten Wert stünde, sähe
  // aus wie eine Einstellung und wäre keine.
  async function setTransport(transport: TransportKind) {
    if (!view) return;
    setBusy(true);
    try {
      setView(await Api.setSupplyTransport(view.invoiceId, transport));
      await loadReport();
      setError('');
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  async function pickFile() {
    try {
      const paths = await Api.selectReceiptFiles('Nachweisbeleg auswählen');
      if (paths.length > 0) setFilePath(paths[0]);
    } catch (e) {
      setError(message(e));
    }
  }

  async function addEvidence() {
    if (!view) return;
    if (!issuer.trim()) {
      setError('Ohne Aussteller lässt sich die Unabhängigkeit der Belege nicht beurteilen.');
      return;
    }
    setBusy(true);
    setError('');
    try {
      setView(
        await Api.addSupplyEvidence({
          invoiceId: view.invoiceId,
          kind,
          issuer: issuer.trim(),
          independent,
          date,
          filePath: filePath || undefined,
          transport: view.transport,
        }),
      );
      setIssuer('');
      setFilePath('');
      await loadReport();
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  async function removeEvidence(evidenceId: number) {
    if (!view) return;
    setBusy(true);
    try {
      setView(await Api.removeSupplyEvidence(view.invoiceId, evidenceId, view.transport));
      await loadReport();
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  if (loading) return <SkeletonRows rows={8} />;

  const rows = report?.rows ?? [];
  const kindLabel = (value: EvidenceKind) =>
    kinds.find((info) => info.kind === value)?.label ?? value;

  return (
    <>
      {error && <Notice tone="negative" text={error} className="mb-6" />}

      {report && report.incomplete > 0 && (
        <Notice
          className="mb-6"
          text={`${report.incomplete} steuerfreie Lieferungen haben noch keinen vollständigen Belegnachweis.`}
        />
      )}

      <StatRow>
        <Stat label="Steuerfreie Lieferungen" value={String(rows.length)} context={`${year}`} />
        <Stat
          label="Ohne vollständigen Nachweis"
          value={String(report?.incomplete ?? 0)}
          context="§§ 17a ff. UStDV"
          tone={(report?.incomplete ?? 0) > 0 ? 'negative' : 'positive'}
        />
        <Stat
          label="Betroffenes Entgelt"
          value={formatCents(
            rows.filter((row) => !row.status.fulfilled).reduce((sum, row) => sum + row.netAmount, 0),
          )}
          context="ohne Nachweis"
        />
      </StatRow>

      <Section
        title="Innergemeinschaftliche Lieferungen"
        context="Eine Zeile öffnet die Belege dieser Lieferung"
        className="mt-8"
        divider={false}
        action={
          <HelpPopover label="Erklärung zum Belegnachweis">
            {report?.note ||
              'Die Steuerbefreiung setzt den Belegnachweis voraus. Die Vermutung des § 17a UStDV greift bei zwei einander nicht widersprechenden Belegen aus Gruppe a von zwei unabhängigen Parteien oder bei einem Beleg aus a und einem aus b; im Abholfall kommt die Gelangensbestätigung hinzu.'}
          </HelpPopover>
        }
      >
        {rows.length === 0 ? (
          <EmptyState
            title="Keine steuerfreie ig. Lieferung"
            description="Aufgeführt wird, was mit Steuerfall ig. Lieferung ausgestellt wurde."
          />
        ) : (
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th className="w-40">Rechnung</Th>
                <Th className="w-28">Datum</Th>
                <Th>Empfänger</Th>
                <Th numeric className="w-32">
                  Netto
                </Th>
                <Th numeric className="w-20">
                  Belege
                </Th>
                <Th className="w-32">Beförderung</Th>
                <Th className="w-56">Nachweis</Th>
              </Tr>
            </Thead>
            <Tbody>
              {rows.map((row) => (
                <Tr
                  key={row.invoiceId}
                  variant={row.invoiceId === selected ? 'selected' : 'default'}
                  className="cursor-pointer"
                  onClick={() => open(row.invoiceId)}
                >
                  <Td code>{row.invoiceNumber}</Td>
                  <Td>{formatDate(row.date)}</Td>
                  <Td>{row.contactName}</Td>
                  <Td numeric>{formatCents(row.netAmount)}</Td>
                  <Td numeric>{row.evidenceCount}</Td>
                  <Td className="text-ink-muted">
                    {row.transport === 'customer' ? 'Abholung' : 'Lieferer'}
                  </Td>
                  <Td>
                    <div className="flex items-center gap-1">
                      <EvidenceMark status={row.status} />
                      <HelpPopover label={`Bewertung ${row.invoiceNumber}`}>
                        {row.status.reason}
                      </HelpPopover>
                    </div>
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>

      {view && (
        <Section
          title={`Belege zu ${view.invoiceNumber}`}
          context={`${view.contactName} · ${formatDate(view.date)}`}
          action={
            <HelpPopover label="Bewertung des Nachweises">
              {`${view.status.reason}${view.status.basis ? ` (${view.status.basis})` : ''}`}
            </HelpPopover>
          }
        >
          <FieldRow className="mb-5">
            <Field
              label="Beförderung"
              className="w-72"
              hint="Der Abholfall braucht die Gelangensbestätigung"
              help="Die Auswahl wird an der Rechnung gespeichert; sie entscheidet über die Bewertung des Nachweises nach § 17a UStDV."
            >
              <Select<TransportKind>
                value={view.transport || 'supplier'}
                onValueChange={setTransport}
                items={TRANSPORT_ITEMS}
                disabled={lock.locked || busy}
                aria-label="Wer den Gegenstand befördert hat"
              />
            </Field>
            <Field label="Stand" className="w-64">
              <FieldValue>
                <EvidenceMark status={view.status} />
              </FieldValue>
            </Field>
            <Field label="Unabhängige Belege" className="w-52">
              <FieldValue>
                <span className="num">
                  {view.status.groupACount} aus a · {view.status.groupBCount} aus b
                </span>
              </FieldValue>
            </Field>
          </FieldRow>

          {view.items.length > 0 && (
            <Table density="kompakt">
              <Thead>
                <Tr>
                  <Th>Belegart</Th>
                  <Th className="w-16">Gruppe</Th>
                  <Th>Aussteller</Th>
                  <Th className="w-32">Unabhängig</Th>
                  <Th className="w-28">Datum</Th>
                  <Th className="w-24">Beleg</Th>
                  <Th className="w-16" />
                </Tr>
              </Thead>
              <Tbody>
                {view.items.map((item) => (
                  <Tr key={item.id}>
                    <Td>{kindLabel(item.kind)}</Td>
                    <Td className="text-ink-muted">
                      {kinds.find((info) => info.kind === item.kind)?.group || '—'}
                    </Td>
                    <Td>{item.issuer}</Td>
                    <Td className={item.independent ? undefined : 'text-attention-text'}>
                      {item.independent ? 'ja' : 'nein'}
                    </Td>
                    <Td>{formatDate(item.date)}</Td>
                    <Td code>{item.receiptId ? String(item.receiptId) : '—'}</Td>
                    <Td>
                      <Button
                        variant="quiet"
                        size="sm"
                        iconOnly
                        title="Nachweisbeleg entfernen"
                        aria-label="Nachweisbeleg entfernen"
                        onClick={() => removeEvidence(item.id)}
                        disabled={lock.locked || busy}
                      >
                        <Trash2 className="w-3.5 h-3.5" strokeWidth={1.5} />
                      </Button>
                    </Td>
                  </Tr>
                ))}
              </Tbody>
            </Table>
          )}

          <FieldRow className="mt-5 items-end">
            <Field label="Belegart" className="w-72">
              <Select<EvidenceKind>
                value={kind}
                onValueChange={setKind}
                aria-label="Art des Nachweisbelegs"
                items={kinds.map((info) => ({
                  value: info.kind,
                  label: info.group ? `${info.label} (${info.group})` : info.label,
                }))}
              />
            </Field>
            <Field label="Aussteller" className="w-64">
              <Input value={issuer} onChange={(e) => setIssuer(e.target.value)} />
            </Field>
            <Field label="Datum" className="w-40">
              <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
            </Field>
            <Field label="Datei" optional className="w-72" hint="Sie geht in den Belegspeicher">
              <div className="flex items-center gap-2">
                <Input
                  value={filePath}
                  onChange={(e) => setFilePath(e.target.value)}
                  placeholder="Pfad der Datei"
                />
                <Button variant="secondary" onClick={pickFile} disabled={lock.locked}>
                  Wählen
                </Button>
              </div>
            </Field>
          </FieldRow>

          <div className="mt-4 flex items-center gap-6">
            <Checkbox
              label="Aussteller ist unabhängig"
              hint="Weder Lieferer noch Erwerber (Art. 45a MwStVO)"
              checked={independent}
              onCheckedChange={(checked) => setIndependent(Boolean(checked))}
            />
            <Button
              variant="primary"
              icon={<Plus className="w-4 h-4" strokeWidth={1.5} />}
              onClick={addEvidence}
              loading={busy}
              disabled={lock.locked || busy}
              title={lock.hint}
            >
              Nachweis ablegen
            </Button>
          </div>
        </Section>
      )}
    </>
  );
};

// -------------------------------------------------------------------------
// Nicht abziehbare Betriebsausgaben
// -------------------------------------------------------------------------

const NonDeductiblePanel: React.FC<{ year: number }> = ({ year }) => {
  const lock = useWriteLock();
  const [report, setReport] = useState<NonDeductibleReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);
  const [rebooking, setRebooking] = useState<GiftRecipientRow | null>(null);
  const [reason, setReason] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      setReport(await Api.getNonDeductibleReport(year));
    } catch (e) {
      setReport(null);
      setError(message(e));
    } finally {
      setLoading(false);
    }
  }, [year]);

  useEffect(() => {
    void load();
  }, [load]);

  async function rebook() {
    if (!rebooking) return;
    if (!reason.trim()) {
      setError('Eine Umbuchung ohne Grund ist im Journal später nicht mehr erklärbar.');
      return;
    }
    setBusy(true);
    try {
      const result = await Api.rebookGiftsForRecipient({
        fiscalYear: year,
        recipientKey: rebooking.recipientKey,
        reason: reason.trim(),
      });
      setRebooking(null);
      setReason('');
      await load();
      toast.success(
        `${result.reversals.length} Buchungen storniert, ${result.rebookings.length} neu gebucht.`,
      );
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  if (loading) return <SkeletonRows rows={8} />;

  const categories = report?.categories ?? [];
  const recipients = report?.recipients ?? [];
  const overLimit = recipients.filter((row) => row.overLimit);
  const toRebook = recipients.filter((row) => row.toRebook.length > 0);

  return (
    <>
      {error && <Notice tone="negative" text={error} className="mb-6" />}

      {toRebook.length > 0 && (
        <Notice
          className="mb-6"
          text={`${toRebook.length} Empfänger haben die Freigrenze überschritten; frühere Geschenke stehen noch abziehbar.`}
        />
      )}

      <StatRow>
        <Stat
          label="Freigrenze"
          value={formatCents(report?.giftLimit ?? 0)}
          context="je Empfänger und Jahr"
        />
        <Stat
          label="Nicht abziehbar"
          value={formatCents(
            categories.reduce((sum, category) => sum + category.nonDeductibleAmount, 0),
          )}
          context="§ 4 Abs. 5 EStG"
        />
        <Stat
          label="Über der Freigrenze"
          value={String(overLimit.length)}
          context="Empfänger"
          tone={overLimit.length > 0 ? 'negative' : 'positive'}
        />
      </StatRow>

      <Section
        title="Kategorien"
        context={`Geschäftsjahr ${year}`}
        className="mt-8"
        divider={false}
      >
        {categories.length === 0 ? (
          <EmptyState
            title="Keine beschränkt abziehbare Ausgabe"
            description="Aufgeführt wird, was auf den Konten des § 4 Abs. 5 EStG gebucht ist."
          />
        ) : (
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th>Kategorie</Th>
                <Th className="w-52">Vorschrift</Th>
                <Th numeric className="w-32">
                  Abziehbar
                </Th>
                <Th numeric className="w-36">
                  Nicht abziehbar
                </Th>
                <Th numeric className="w-32">
                  Summe
                </Th>
                <Th numeric className="w-24">
                  Buchungen
                </Th>
              </Tr>
            </Thead>
            <Tbody>
              {categories.map((category) => (
                <Tr key={category.key}>
                  <Td>
                    <div className="flex items-center gap-1">
                      {category.label}
                      <HelpPopover label={`Erklärung zu ${category.label}`}>
                        {category.note}
                      </HelpPopover>
                    </div>
                  </Td>
                  <Td className="text-ink-muted">{category.reference}</Td>
                  <Td numeric>{formatCents(category.deductibleAmount)}</Td>
                  <Td numeric>{formatCents(category.nonDeductibleAmount)}</Td>
                  <Td numeric>{formatCents(category.total)}</Td>
                  <Td numeric>{category.count}</Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>

      <Section
        title="Geschenke je Empfänger"
        context="Die Freigrenze läuft je Empfänger und Wirtschaftsjahr"
        action={
          <HelpPopover label="Erklärung zur Freigrenze">
            {report?.note ||
              'Eine Freigrenze ist kein Freibetrag: Wird sie überschritten, sind sämtliche Geschenke an diesen Empfänger nicht abziehbar, und mit ihnen entfällt der Vorsteuerabzug (§ 15 Abs. 1a UStG).'}
          </HelpPopover>
        }
      >
        {recipients.length === 0 ? (
          <EmptyState
            title="Kein Geschenk erfasst"
            description="Jedes Geschenk trägt seinen Empfänger; ohne ihn wird nicht gebucht."
          />
        ) : (
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th>Empfänger</Th>
                <Th numeric className="w-32">
                  Summe netto
                </Th>
                <Th className="w-40">Freigrenze</Th>
                <Th numeric className="w-28">
                  Umzubuchen
                </Th>
                <Th className="w-40" />
              </Tr>
            </Thead>
            <Tbody>
              {recipients.map((row) => (
                <Tr key={row.recipientKey}>
                  <Td>
                    <div className="flex items-center gap-1">
                      {row.recipientName}
                      <HelpPopover label={`Geschenke an ${row.recipientName}`}>
                        {row.note}
                      </HelpPopover>
                    </div>
                  </Td>
                  <Td numeric>{formatCents(row.total)}</Td>
                  <Td>
                    <span
                      className={cn(
                        'inline-flex items-center gap-1.5',
                        row.overLimit ? 'text-negative-text' : 'text-positive-text',
                      )}
                    >
                      {row.overLimit ? (
                        <CircleAlert className="w-3.5 h-3.5 shrink-0" strokeWidth={1.5} />
                      ) : (
                        <CheckCircle2 className="w-3.5 h-3.5 shrink-0" strokeWidth={1.5} />
                      )}
                      {row.overLimit ? 'überschritten' : 'eingehalten'}
                    </span>
                  </Td>
                  <Td numeric>{row.toRebook.length || '—'}</Td>
                  <Td>
                    {row.toRebook.length > 0 && (
                      <Button
                        variant="secondary"
                        size="sm"
                        onClick={() => {
                          setRebooking(row);
                          setReason('');
                        }}
                        disabled={lock.locked || busy}
                        title={lock.hint}
                      >
                        Umbuchen
                      </Button>
                    )}
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>

      <Dialog
        open={Boolean(rebooking)}
        onOpenChange={(open) => !open && setRebooking(null)}
        title="Geschenke umbuchen"
        width="max-w-lg"
        footer={
          <>
            <Button variant="secondary" onClick={() => setRebooking(null)}>
              Abbrechen
            </Button>
            <Button variant="primary" onClick={rebook} loading={busy} disabled={busy}>
              Storno und Neubuchung
            </Button>
          </>
        }
      >
        {rebooking && (
          <div className="space-y-4">
            <FieldRow>
              <Field label="Empfänger" className="w-64">
                <FieldValue>{rebooking.recipientName}</FieldValue>
              </Field>
              <Field label="Betroffene Buchungen" className="w-40">
                <FieldValue>
                  <span className="num">{rebooking.toRebook.length}</span>
                </FieldValue>
              </Field>
              <Field label="Summe" className="w-40">
                <FieldValue>
                  <span className="num">
                    {formatCents(
                      rebooking.toRebook.reduce((sum, booking) => sum + booking.netAmount, 0),
                    )}
                  </span>
                </FieldValue>
              </Field>
            </FieldRow>
            <Field
              label="Grund"
              explain="Die Umbuchung nimmt jede abziehbar gebuchte Zuwendung an diesen Empfänger zurück und bucht sie auf das nicht abziehbare Konto — mit ihr entfällt der Vorsteuerabzug. Storno und Neubuchung stehen danach beide im Journal."
            >
              <Textarea
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                placeholder="Freigrenze mit dem Geschenk vom 12.11. überschritten"
              />
            </Field>
          </div>
        )}
      </Dialog>
    </>
  );
};

// -------------------------------------------------------------------------
// Fremdwährung
// -------------------------------------------------------------------------

const CurrencyPanel: React.FC<{ year: number }> = ({ year }) => {
  const lock = useWriteLock();
  const [currency, setCurrency] = useState('USD');
  const [date, setDate] = useState(today());
  const [rate, setRate] = useState<ExchangeRate | null>(null);
  const [history, setHistory] = useState<ExchangeRate[]>([]);
  const [vatRates, setVatRates] = useState<VatExchangeRate[]>([]);
  const [valuation, setValuation] = useState<ForeignCurrencyValuation | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  // Ein Kurs von Hand — ohne Netz gibt es keinen anderen.
  const [manualRate, setManualRate] = useState('');
  const [manualSource, setManualSource] = useState('');
  const [importPath, setImportPath] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [rates, vat, preview] = await Promise.all([
        Api.getExchangeRates(currency, `${year}-01-01`, `${year}-12-31`),
        Api.getVatExchangeRates(`${year}-01`, `${year}-12`),
        Api.previewCurrencyValuation(year),
      ]);
      setHistory(rates);
      setVatRates(vat);
      setValuation(preview);
    } catch (e) {
      setError(message(e));
    } finally {
      setLoading(false);
    }
  }, [currency, year]);

  useEffect(() => {
    void load();
  }, [load]);

  async function fetchRate() {
    setBusy(true);
    setError('');
    try {
      const result = await Api.getExchangeRate(currency.trim().toUpperCase(), date);
      setRate(result);
      await load();
    } catch (e) {
      setRate(null);
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  async function saveManual() {
    const micros = parseExchangeRate(manualRate);
    if (micros === null || !manualSource.trim()) {
      setError('Ein Kurs von Hand braucht seinen Wert und seine Quelle.');
      return;
    }
    setBusy(true);
    setError('');
    try {
      const saved = await Api.saveExchangeRate({
        currency: currency.trim().toUpperCase(),
        date,
        rateMicros: micros,
        source: manualSource.trim(),
        manual: true,
      });
      setRate(saved);
      setManualRate('');
      setManualSource('');
      await load();
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  async function importVatRates() {
    if (!importPath.trim()) {
      setError('Der Pfad der CSV-Datei fehlt.');
      return;
    }
    setBusy(true);
    setError('');
    try {
      const result = await Api.importVatExchangeRatesCSV(importPath.trim());
      setImportPath('');
      await load();
      toast.success(`${result.imported} Kurse übernommen, ${result.skipped} übergangen.`);
      if (result.problems.length > 0) setError(result.problems.join(' · '));
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  async function bookValuation() {
    setBusy(true);
    setError('');
    try {
      const result = await Api.bookCurrencyValuation(year);
      setValuation(result);
      toast.success(`Bewertung gebucht: ${result.entryNumber ?? ''}`.trim());
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  if (loading) return <SkeletonRows rows={8} />;

  const items = valuation?.items ?? [];
  // Eine gebuchte Bewertung trägt ihre Buchungsnummer; ein zweiter Lauf wird vom
  // Backend abgewiesen.
  const booked = Boolean(valuation?.entryNumber);

  return (
    <>
      {error && <Notice tone="negative" text={error} className="mb-6" />}

      <Section
        title="Tageskurs"
        context="Referenzkurs der Europäischen Zentralbank"
        divider={false}
        action={
          <HelpPopover label="Erklärung zum Kurs">
            Buchfink rät keinen Kurs. Liegt für den Tag keiner vor und ist der Kursdienst nicht
            erreichbar, wird der Kurs von Hand erfasst — mit seiner Quelle, weil er über den
            Aufwand entscheidet.
          </HelpPopover>
        }
      >
        <FieldRow className="items-end">
          <Field label="Währung" className="w-32" hint="ISO 4217">
            <Input
              value={currency}
              onChange={(e) => setCurrency(e.target.value.toUpperCase())}
              className="code-num"
            />
          </Field>
          <Field label="Tag" className="w-44">
            <Input type="date" value={date} onChange={(e) => setDate(e.target.value)} />
          </Field>
          <Field label="Abfrage" className="w-40">
            <Button variant="secondary" onClick={fetchRate} loading={busy} disabled={busy}>
              Kurs holen
            </Button>
          </Field>
          <Field label="Ergebnis" className="w-72">
            <FieldValue>
              {rate ? (
                <span className="num">
                  1 € = {formatExchangeRate(rate.rateMicros)} {rate.currency}
                  <span className="ml-2 text-caption text-ink-subtle">
                    {rate.source}
                    {rate.manual ? ' · von Hand' : ''}
                  </span>
                </span>
              ) : (
                '—'
              )}
            </FieldValue>
          </Field>
        </FieldRow>

        <FieldRow className="mt-4 items-end">
          <Field label="Kurs von Hand" className="w-44" hint="Etwa 1,0874">
            <Input
              align="right"
              value={manualRate}
              onChange={(e) => setManualRate(e.target.value)}
            />
          </Field>
          <Field label="Quelle" className="w-80">
            <Input
              value={manualSource}
              onChange={(e) => setManualSource(e.target.value)}
              placeholder="Bundesbank, Kursblatt vom 02.01."
            />
          </Field>
          <Field label="Übernahme" className="w-40">
            <Button
              variant="secondary"
              onClick={saveManual}
              disabled={lock.locked || busy}
              title={lock.hint}
            >
              Kurs erfassen
            </Button>
          </Field>
        </FieldRow>

        <div className="mt-6">
          {history.length === 0 ? (
            <EmptyState
              title="Kein Kurs im Geschäftsjahr"
              description={`Für ${currency} ist in ${year} noch kein Kurs gespeichert.`}
            />
          ) : (
            <Table density="kompakt">
              <Thead>
                <Tr>
                  <Th className="w-32">Tag</Th>
                  <Th className="w-24">Währung</Th>
                  <Th numeric className="w-40">
                    Kurs je Euro
                  </Th>
                  <Th className="w-32">Herkunft</Th>
                  <Th>Quelle</Th>
                </Tr>
              </Thead>
              <Tbody>
                {history.map((row) => (
                  <Tr key={row.id}>
                    <Td className="num">{formatDate(row.date)}</Td>
                    <Td code>{row.currency}</Td>
                    <Td numeric>{formatExchangeRate(row.rateMicros)}</Td>
                    <Td className="text-ink-muted">{row.manual ? 'von Hand' : 'Kursdienst'}</Td>
                    <Td className="text-ink-muted whitespace-normal">{row.source}</Td>
                  </Tr>
                ))}
              </Tbody>
            </Table>
          )}
        </div>
      </Section>

      <Section
        title="Umsatzsteuer-Umrechnungskurse"
        context="Monatliche Durchschnittskurse des BMF"
        action={
          <HelpPopover label="Erklärung zum Umsatzsteuerkurs">
            Für die Umsatzsteuer gilt der monatliche Durchschnittskurs, den das
            Bundesministerium der Finanzen veröffentlicht (§ 16 Abs. 6 UStG). Liegt er vor,
            rechnet Buchfink die Bemessungsgrundlage damit; der Aufwand bleibt beim Tageskurs,
            und die Differenz ist Kursaufwand oder Kursertrag.
          </HelpPopover>
        }
      >
        <FieldRow className="mb-5 items-end">
          <Field
            label="CSV-Datei"
            className="w-[28rem]"
            hint="Monat, Währung, Kurs"
            explain="Die Liste des Bundesministeriums der Finanzen erscheint monatlich. Der Import liest sie als CSV; was er nicht zuordnen kann, meldet er, statt es stillschweigend zu übergehen."
          >
            <Input
              value={importPath}
              onChange={(e) => setImportPath(e.target.value)}
              placeholder="Pfad der CSV-Datei"
            />
          </Field>
          <Field label="Import" className="w-40">
            <Button
              variant="secondary"
              icon={<Download className="w-4 h-4" strokeWidth={1.5} />}
              onClick={importVatRates}
              disabled={lock.locked || busy}
              title={lock.hint}
            >
              Einlesen
            </Button>
          </Field>
        </FieldRow>

        {vatRates.length === 0 ? (
          <EmptyState
            title="Kein Durchschnittskurs erfasst"
            description="Ohne ihn rechnet Buchfink die Bemessungsgrundlage mit dem Tageskurs."
          />
        ) : (
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th className="w-28">Monat</Th>
                <Th className="w-24">Währung</Th>
                <Th numeric className="w-40">
                  Kurs je Euro
                </Th>
                <Th>Quelle</Th>
              </Tr>
            </Thead>
            <Tbody>
              {vatRates.map((row) => (
                <Tr key={`${row.month}-${row.currency}`}>
                  <Td className="num">{row.month}</Td>
                  <Td code>{row.currency}</Td>
                  <Td numeric>{formatExchangeRate(row.rateMicros)}</Td>
                  <Td className="text-ink-muted whitespace-normal">{row.source}</Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>

      <Section
        title="Stichtagsbewertung"
        context={`Stichtag ${formatDate(valuation?.cutoff ?? '')} · Auflösung ${formatDate(valuation?.reversalDate ?? '')}`}
        action={
          <div className="flex items-center gap-2">
            <HelpPopover label="Erklärung zur Stichtagsbewertung">
              {valuation?.note ||
                'Posten in Fremdwährung werden zum Devisenkassamittelkurs des Stichtags bewertet. Bei einer Restlaufzeit bis zu einem Jahr wirken Gewinn und Verlust erfolgswirksam (§ 256a Satz 2 HGB), darüber nur der Verlust. Die Bewertung wird am ersten Tag des Folgejahres wieder aufgelöst.'}
            </HelpPopover>
            <Button
              variant="primary"
              onClick={bookValuation}
              loading={busy}
              disabled={lock.locked || busy || booked || items.length === 0}
              title={
                lock.locked
                  ? lock.hint
                  : booked
                    ? 'bereits gebucht — Storno über das Journal'
                    : items.length === 0
                      ? 'Nichts zu bewerten'
                      : undefined
              }
            >
              Bewertung buchen
            </Button>
          </div>
        }
      >
        <StatRow className="mb-5">
          <Stat
            label="Erträge"
            value={formatCents(valuation?.totalGain ?? 0)}
            context="unrealisiert, Konto 4840"
          />
          <Stat
            label="Aufwendungen"
            value={formatCents(valuation?.totalLoss ?? 0)}
            context="unrealisiert, Konto 6880"
          />
          <Stat
            label="Buchung"
            value={valuation?.entryNumber || '—'}
            context={valuation?.reversalEntryNumber ? `Auflösung ${valuation.reversalEntryNumber}` : 'noch nicht gebucht'}
          />
        </StatRow>

        {items.length === 0 ? (
          <EmptyState
            title="Kein Posten in Fremdwährung"
            description="Bewertet werden offene Posten und Bankguthaben, die nicht auf Euro lauten."
          />
        ) : (
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th className="w-24">Art</Th>
                <Th className="w-32">Buchung</Th>
                <Th>Bezeichnung</Th>
                <Th className="w-20">Währung</Th>
                <Th numeric className="w-32">
                  Fremdbetrag
                </Th>
                <Th numeric className="w-32">
                  Buchwert
                </Th>
                <Th numeric className="w-32">
                  Stichtag
                </Th>
                <Th numeric className="w-32">
                  Buchung
                </Th>
              </Tr>
            </Thead>
            <Tbody>
              {items.map((item, index) => (
                <Tr key={`${item.entryNumber}-${item.account}-${index}`}>
                  <Td className="text-ink-muted">{item.kind === 'bank' ? 'Bank' : 'Posten'}</Td>
                  <Td code>{item.entryNumber || '—'}</Td>
                  <Td className="whitespace-normal">
                    <div className="flex items-center gap-1">
                      {item.description}
                      <HelpPopover label={`Bewertung ${item.entryNumber || item.account}`}>
                        {item.reason}
                      </HelpPopover>
                    </div>
                  </Td>
                  <Td code>{item.currency}</Td>
                  <Td numeric>{formatCentsPlain(item.foreignAmount)}</Td>
                  <Td numeric>{formatCents(item.bookValue)}</Td>
                  <Td numeric>{formatCents(item.valueAtCutoff)}</Td>
                  <Td numeric className={item.recognised ? undefined : 'text-ink-subtle'}>
                    {item.recognised ? formatCents(item.amount) : '—'}
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>

      <EndpointSection field="exchangeRate" title="Adresse des Kursdienstes" />
    </>
  );
};

// -------------------------------------------------------------------------
// Anlagen: Wertaufholung, Sammelposten, Regelsätze
// -------------------------------------------------------------------------

const AssetObligationsPanel: React.FC<{ year: number }> = ({ year }) => {
  const lock = useWriteLock();
  const [writeUps, setWriteUps] = useState<WriteUpReport | null>(null);
  const [pool, setPool] = useState<PoolConsistencyReport | null>(null);
  const [rules, setRules] = useState<AfaRules | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [confirming, setConfirming] = useState<number>(0);
  const [note, setNote] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [report, consistency, afa] = await Promise.all([
        Api.getWriteUpReport(year),
        Api.getPoolConsistencyReport(year),
        Api.getAfaRules(),
      ]);
      setWriteUps(report);
      setPool(consistency);
      setRules(afa);
    } catch (e) {
      setError(message(e));
    } finally {
      setLoading(false);
    }
  }, [year]);

  useEffect(() => {
    void load();
  }, [load]);

  async function confirm() {
    if (!note.trim()) {
      setError('Ohne Begründung ist die Bestätigung nur eine Behauptung.');
      return;
    }
    setBusy(true);
    try {
      setWriteUps(await Api.confirmImpairmentPersists(confirming, year, note.trim()));
      setConfirming(0);
      setNote('');
      // Kein Toast: der geschlossene Dialog ist die Rückmeldung, die Zeile
      // trägt den Stand danach selbst (Gestaltungskonzept 8.5).
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  if (loading) return <SkeletonRows rows={8} />;

  const candidates = writeUps?.candidates ?? [];
  const pooled = pool?.pooled ?? [];
  const immediate = pool?.immediate ?? [];

  return (
    <>
      {error && <Notice tone="negative" text={error} className="mb-6" />}

      {(writeUps?.open ?? 0) > 0 && (
        <Notice
          className="mb-6"
          text={`${writeUps?.open} Anlagegüter mit außerplanmäßiger Abschreibung sind für ${year} weder zugeschrieben noch bestätigt.`}
        />
      )}
      {pool && !pool.consistent && (
        <Notice
          className="mb-6"
          text="Das Wahlrecht zwischen Sammelposten und Sofortabzug ist im Jahr uneinheitlich ausgeübt (§ 6 Abs. 2a Satz 5 EStG)."
        />
      )}

      <Section
        title="Wertaufholung"
        context={`Geschäftsjahr ${year} · § 253 Abs. 5 Satz 1 HGB`}
        divider={false}
        action={
          <HelpPopover label="Erklärung zur Wertaufholung">
            {writeUps?.note ||
              'Ist der Grund einer außerplanmäßigen Abschreibung weggefallen, ist zuzuschreiben — bis höchstens zu den fortgeführten Anschaffungskosten. Das ist ein Gebot und kein Wahlrecht; besteht der Grund fort, wird das festgehalten.'}
          </HelpPopover>
        }
      >
        {candidates.length === 0 ? (
          <EmptyState
            title="Keine außerplanmäßige Abschreibung offen"
            description="Geprüft wird jedes Anlagegut, dessen Buchwert unter den fortgeführten Kosten liegt."
          />
        ) : (
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th className="w-28">Inventar</Th>
                <Th>Anlagegut</Th>
                <Th numeric className="w-32">
                  Buchwert
                </Th>
                <Th numeric className="w-40">
                  Fortgeführte Kosten
                </Th>
                <Th numeric className="w-36">
                  Spielraum
                </Th>
                <Th className="w-52">Stand</Th>
              </Tr>
            </Thead>
            <Tbody>
              {candidates.map((candidate) => (
                <Tr key={candidate.assetId}>
                  <Td code>{candidate.inventoryNumber}</Td>
                  <Td className="whitespace-normal">
                    <div className="flex items-center gap-1">
                      {candidate.name}
                      <HelpPopover label={`Abwertungen ${candidate.name}`}>
                        {candidate.impairments
                          .map(
                            (impairment) =>
                              `${formatDate(impairment.date)}: ${formatCents(impairment.amount)} — ${impairment.reason}`,
                          )
                          .join(' · ') || candidate.note}
                      </HelpPopover>
                    </div>
                  </Td>
                  <Td numeric>{formatCents(candidate.bookValue)}</Td>
                  <Td numeric>{formatCents(candidate.continuedCost)}</Td>
                  <Td numeric>{formatCents(candidate.maxWriteUp)}</Td>
                  <Td>
                    {candidate.confirmed ? (
                      <span className="inline-flex items-center gap-1.5 text-positive-text">
                        <CheckCircle2 className="w-3.5 h-3.5 shrink-0" strokeWidth={1.5} />
                        Grund besteht fort
                      </span>
                    ) : (
                      <Button
                        variant="secondary"
                        size="sm"
                        onClick={() => {
                          setConfirming(candidate.assetId);
                          setNote('');
                        }}
                        disabled={lock.locked || busy}
                        title={lock.hint}
                      >
                        Grund besteht fort
                      </Button>
                    )}
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>

      <Section
        title="Sammelposten und Sofortabzug"
        context={`Zugänge zwischen ${formatCents(pool?.lowerLimit ?? 0)} und ${formatCents(pool?.upperLimit ?? 0)}`}
        action={
          <HelpPopover label="Erklärung zur Einheitlichkeit">
            {pool?.note ||
              'Wird für ein Wirtschaftsjahr ein Sammelposten gebildet, gilt das Wahlrecht für alle Zugänge dieses Jahres in diesem Wertbereich einheitlich (§ 6 Abs. 2a Satz 5 EStG). Beides nebeneinander ist unzulässig.'}
          </HelpPopover>
        }
      >
        {pooled.length === 0 && immediate.length === 0 ? (
          <EmptyState
            title="Kein Zugang in diesem Wertbereich"
            description="Die Frage nach der Einheitlichkeit stellt sich dann nicht."
          />
        ) : (
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th className="w-28">Inventar</Th>
                <Th>Anlagegut</Th>
                <Th className="w-32">Anschaffung</Th>
                <Th numeric className="w-32">
                  Kosten
                </Th>
                <Th className="w-40">Behandlung</Th>
              </Tr>
            </Thead>
            <Tbody>
              {[...pooled, ...immediate].map((row) => (
                <Tr key={`${row.method}-${row.assetId}`}>
                  <Td code>{row.inventoryNumber}</Td>
                  <Td>{row.name}</Td>
                  <Td>{formatDate(row.acquisitionDate)}</Td>
                  <Td numeric>{formatCents(row.cost)}</Td>
                  <Td className="text-ink-muted">
                    {row.method === 'pool' ? 'Sammelposten' : 'Sofortabzug'}
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>

      <Section
        title="Abschreibungsregeln"
        context={rules ? `Stand ${rules.version} · ${rules.source}` : 'aus der Ressource'}
        action={
          <HelpPopover label="Erklärung zu den Regelsätzen">
            {rules?.investmentDeductionNote || rules?.note || 'Die Sätze stehen als Ressource neben dem Programm.'}
          </HelpPopover>
        }
      >
        <Table density="kompakt">
          <Thead>
            <Tr>
              <Th className="w-56">Regel</Th>
              <Th className="w-56">Zeitraum</Th>
              <Th>Satz</Th>
              <Th className="w-64">Fundstelle</Th>
            </Tr>
          </Thead>
          <Tbody>
            {(rules?.degressiveWindows ?? []).map((window) => (
              <Tr key={`degressive-${window.from}`}>
                <Td>Degressive AfA</Td>
                <Td className="num">
                  {formatDate(window.from)} – {formatDate(window.until)}
                </Td>
                <Td className="num">
                  das {(window.factorPermille / 1000).toLocaleString('de-DE')}-fache, höchstens{' '}
                  {formatPermille(window.maxPermille)}
                </Td>
                <Td className="text-ink-muted whitespace-normal">{window.source}</Td>
              </Tr>
            ))}
            {(rules?.electricVehicleWindows ?? []).map((window) => (
              <Tr key={`electric-${window.from}`}>
                <Td>E-Fahrzeug-Staffel</Td>
                <Td className="num">
                  {formatDate(window.from)} – {formatDate(window.until)}
                </Td>
                <Td className="num">
                  {window.permillePerYear.map((value) => formatPermille(value)).join(' · ')}
                </Td>
                <Td className="text-ink-muted whitespace-normal">{window.source}</Td>
              </Tr>
            ))}
            {(rules?.buildingRates ?? []).map((rate) => (
              <Tr key={`building-${rate.key}`}>
                <Td>{rate.label}</Td>
                <Td className="num">
                  {rate.referenceFrom ? `ab ${formatDate(rate.referenceFrom)}` : 'davor'}
                </Td>
                <Td className="num">{formatPermille(rate.permille)}</Td>
                <Td className="text-ink-muted whitespace-normal">{rate.source}</Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      </Section>

      <Dialog
        open={confirming > 0}
        onOpenChange={(open) => !open && setConfirming(0)}
        title="Grund der Abwertung besteht fort"
        width="max-w-lg"
        footer={
          <>
            <Button variant="secondary" onClick={() => setConfirming(0)}>
              Abbrechen
            </Button>
            <Button variant="primary" onClick={confirm} loading={busy} disabled={busy}>
              Festhalten
            </Button>
          </>
        }
      >
        <Field
          label="Begründung"
          explain="Die Bestätigung tritt an die Stelle der Zuschreibung. Sie gilt für dieses Geschäftsjahr; im nächsten wird die Frage erneut gestellt, weil sich der Grund bis dahin erledigt haben kann."
        >
          <Textarea
            value={note}
            onChange={(e) => setNote(e.target.value)}
            placeholder="Der Mietvertrag ist weiter gekündigt, die Fläche steht leer"
          />
        </Field>
      </Dialog>
    </>
  );
};

// -------------------------------------------------------------------------
// Adressen der Netzdienste
// -------------------------------------------------------------------------

/**
 * Die Adresse eines der beiden Netzdienste, dort wo er gebraucht wird.
 *
 * Beide zusammen in den Einstellungen zu führen wäre die zweite Stelle, an der
 * man nach ihnen sucht: Die Adresse des Bundeszentralamts gehört neben die
 * Bestätigungsanfrage, die des Kursdienstes neben die Kurse.
 */
const EndpointSection: React.FC<{ field: 'vatId' | 'exchangeRate'; title: string }> = ({
  field,
  title,
}) => {
  const lock = useWriteLock();
  const [endpoints, setEndpoints] = useState<ServiceEndpoints | null>(null);
  const [value, setValue] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    void (async () => {
      try {
        const current = await Api.getServiceEndpoints();
        setEndpoints(current);
        setValue(field === 'vatId' ? current.vatIdEndpoint : current.exchangeRateEndpoint);
      } catch (e) {
        setError(message(e));
      }
    })();
  }, [field]);

  async function save() {
    if (!endpoints) return;
    setBusy(true);
    setError('');
    try {
      const next: ServiceEndpoints =
        field === 'vatId'
          ? { ...endpoints, vatIdEndpoint: value.trim() }
          : { ...endpoints, exchangeRateEndpoint: value.trim() };
      const saved = await Api.saveServiceEndpoints(next);
      setEndpoints(saved);
      setValue(field === 'vatId' ? saved.vatIdEndpoint : saved.exchangeRateEndpoint);
      toast.success('Adresse gespeichert.');
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  const fallback =
    field === 'vatId' ? endpoints?.vatIdDefault ?? '' : endpoints?.exchangeRateDefault ?? '';

  return (
    <Section
      title={title}
      context="Leer heißt: die Voreinstellung gilt"
      action={
        <HelpPopover label="Erklärung zur Adresse">
          Wechselt die Stelle ihre Schnittstelle, soll das keine neue Programmfassung nötig
          machen. Die Abfrage trägt die USt-IdNr. und den Namen des Geschäftspartners; sie geht
          über https oder gar nicht.
        </HelpPopover>
      }
    >
      {error && <Notice tone="negative" text={error} className="mb-4" />}
      <FieldRow className="items-end">
        <Field label="Adresse" className="w-[36rem]" hint={fallback ? `Voreinstellung: ${fallback}` : undefined}>
          <Input value={value} onChange={(e) => setValue(e.target.value)} placeholder={fallback} />
        </Field>
        <Field label="Übernahme" className="w-40">
          <Button
            variant="secondary"
            onClick={save}
            loading={busy}
            disabled={lock.locked || busy || !endpoints}
            title={lock.hint}
          >
            Speichern
          </Button>
        </Field>
      </FieldRow>
    </Section>
  );
};

export default ObligationsPage;
