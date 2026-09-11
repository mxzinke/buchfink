import React, { useCallback, useEffect, useState } from 'react';
import {
  AlertCircle,
  ChevronDown,
  ChevronRight,
  Clock,
  ExternalLink,
  Lock,
  OctagonAlert,
  ShieldCheck,
} from 'lucide-react';
import {
  AuditChainResult,
  AuditFilter,
  AuditLogEntry,
  ChangelogEntry,
  CheckRun,
  CheckTimeliness,
  Festschreibung,
  FileCheckResult,
  IntegrityCheckResult,
  MigrationRecord,
  SchemaMigration,
} from '../types';
import { Api } from '../services/api';
import {
  formatCents,
  formatDate,
  formatDateTime,
  formatShortHash,
} from '../utils/formatters';
import {
  Button,
  EmptyState,
  Field,
  Input,
  Help,
  Notice,
  PageHeader,
  Section,
  Select,
  SkeletonRows,
  Stat,
  StatRow,
  TabPanel,
  Table,
  Tabs,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  cn,
  toast,
} from '../components/ui';

/**
 * Die Prüfübersicht (Welle 9).
 *
 * Eine Frage: ist an dieser Buchführung etwas nicht in Ordnung. Sie hat keinen
 * Eintrag in der Navigation, sondern hängt am Zustandsanzeiger in der Fußzeile
 * der Navigation — dort steht die Antwort ohnehin dauerhaft (§11.4), und wer
 * sie genauer wissen will, klickt sie an. Ein Navigationseintrag daneben hätte
 * eine Seite beworben, die im Normalfall nur bestätigt, was der Anzeiger schon
 * sagt.
 *
 * Drei Reiter: die Prüfungen mit ihrem Befund, das Änderungsprotokoll mit
 * Vorher und Nachher, und die Geschichte des Systems selbst — Fassungen,
 * Schemaläufe, Übernahmen. Auch die letzte beantwortet dieselbe Frage: ein
 * fehlgeschlagener Schemalauf und eine Übernahme mit gebrochener Kette sind
 * Befunde und keine Chronik.
 */

type TabKey = 'pruefungen' | 'protokoll' | 'historie';

function message(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

export interface AuditPageProps {
  /**
   * Der Zustand der Kette, wie ihn die Anwendung führt. Er kommt von oben und
   * wird hier nicht ein zweites Mal geholt: der Anzeiger in der Fußzeile und
   * diese Seite dürfen nicht Verschiedenes behaupten.
   */
  integrity: IntegrityCheckResult | null;
  isChecking: boolean;
  onRefreshIntegrity: () => void;
  /**
   * Die Prüfregel, deren Befunde aus der Aufgabenliste gemeint sind. Der
   * jüngste Lauf mit einem solchen Befund wird aufgeklappt und die Zeilen
   * werden hervorgehoben: sonst käme die Aufgabe „Befunde klären" auf einer
   * Liste zusammengeklappter Läufe an.
   */
  initialRule?: string;
}

export const AuditPage: React.FC<AuditPageProps> = ({
  integrity,
  isChecking,
  onRefreshIntegrity,
  initialRule,
}) => {
  // Der Reiter ist Zustand dieser Seite und kein Navigationsziel: jeder Weg
  // hierher — der Zustandsanzeiger, eine Aufgabe mit Prüfregel — meint die
  // Prüfungen. Ein Parameter, den keine Stelle setzt, wäre nur ein Versprechen.
  const [tab, setTab] = useState<TabKey>('pruefungen');

  // Liegt noch kein Ergebnis vor — die Prüfung beim Start ist fehlgeschlagen
  // oder läuft noch nicht —, wird sie hier angestoßen. Eine Prüfübersicht ohne
  // die Auskunft, wegen der sie geöffnet wurde, wäre eine leere Seite.
  useEffect(() => {
    if (integrity === null && !isChecking) onRefreshIntegrity();
    // Nur beim Öffnen: sonst liefe die Prüfung nach jedem gescheiterten
    // Versuch erneut, und zwar in einer Schleife.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const broken = integrity !== null && !integrity.isValid;

  return (
    <div className="max-w-[1200px] mx-auto px-8 py-8">
      <PageHeader
        title="Prüfübersicht"
        context="Ist an den Daten seit ihrer Erfassung etwas verändert worden?"
        action={
          <Button
            variant="secondary"
            loading={isChecking}
            onClick={onRefreshIntegrity}
            icon={<ShieldCheck className="w-4 h-4" strokeWidth={1.5} />}
          >
            Daten jetzt prüfen
          </Button>
        }
      />

      {/* Ein Integritätsbruch darf laut werden, sonst reicht die Kennzahl (§11.4). */}
      {broken && integrity && (
        <div className="mt-6 flex items-start gap-2.5 rounded-control border border-negative-line bg-negative-soft px-4 py-3">
          <AlertCircle className="w-4 h-4 mt-0.5 shrink-0 text-negative" strokeWidth={1.5} />
          <p className="text-body text-negative-text">{integrity.message}</p>
        </div>
      )}

      <Tabs<TabKey>
        value={tab}
        onValueChange={setTab}
        className="mt-6"
        items={[
          { value: 'pruefungen', label: 'Prüfungen' },
          { value: 'protokoll', label: 'Änderungsprotokoll' },
          { value: 'historie', label: 'Systemhistorie' },
        ]}
      >
        {/* Jede Sicht lädt für sich: das Protokoll ist lang, der Belegprüflauf
            öffnet jede Datei einzeln, und niemand braucht beides auf einmal. */}
        <TabPanel value="pruefungen">
          {tab === 'pruefungen' && <ChecksPanel integrity={integrity} initialRule={initialRule} />}
        </TabPanel>
        <TabPanel value="protokoll">{tab === 'protokoll' && <ProtocolPanel />}</TabPanel>
        <TabPanel value="historie">{tab === 'historie' && <HistoryPanel />}</TabPanel>
      </Tabs>
    </div>
  );
};

// -------------------------------------------------------------------------
// Die Prüfungen: Kette, Belegdateien, Festschreibung, Prüfläufe
// -------------------------------------------------------------------------

/** „1 Tag“ und nicht „1 Tage“: die Kennzahl steht in einem Satz. */
function days(count: number): string {
  return count === 1 ? '1 Tag' : `${count} Tage`;
}

/** Belegdatei oder Anlagendokument — beide haben dieselbe Prüfsumme. */
function issueKindLabel(kind: string): string {
  return kind === 'document' ? 'Anlagendokument' : 'Belegdatei';
}

function issueReasonLabel(reason: string): string {
  return reason === 'missing' ? 'Fehlt' : 'Beschädigt';
}

/**
 * Die Abstände Belegdatum → Erfassung → Festschreibung als ein Satz
 * (Entscheidung 4). Sie beantworten die Frage der GoBD Rz. 47 — „wie lange
 * liegen Belege, bevor sie gebucht werden" —, die an einer einzelnen
 * Journalzeile nicht zu beantworten ist.
 */
function timelinessLines(t: CheckTimeliness): string[] {
  const lines: string[] = [];
  lines.push(
    t.measuredEntries === 0
      ? 'Erfassung: nicht messbar, keine Buchung trug ein Belegdatum'
      : `Erfassung: Median ${days(t.captureDaysMedian)}, höchstens ${days(t.captureDaysMax)}, ` +
          `${t.lateEntries} über der Frist von ${days(t.captureLimitDays)} ` +
          `(${t.measuredEntries} gemessen)`,
  );
  lines.push(
    t.committedEntries === 0
      ? `Festschreibung: noch keine Buchung festgeschrieben, ${t.uncommittedEntries} offen`
      : `Festschreibung: Median ${days(t.commitDaysMedian)}, höchstens ${days(t.commitDaysMax)}, ` +
          `${t.uncommittedEntries} offen (${t.committedEntries} festgeschrieben)`,
  );
  return lines;
}

const ChecksPanel: React.FC<{
  integrity: IntegrityCheckResult | null;
  initialRule?: string;
}> = ({ integrity, initialRule }) => {
  const [commitments, setCommitments] = useState<Festschreibung[]>([]);
  // Die Prüfläufe gehören hierher und nicht in die Fristenansicht: dort werden
  // sie ausgelöst, hier bleiben sie nachlesbar — samt der Begründung, mit der
  // ein blockierender Befund übergangen wurde (GoBD Rz. 34 ff.).
  const [checkRuns, setCheckRuns] = useState<CheckRun[]>([]);
  const [expandedRun, setExpandedRun] = useState<number | null>(null);
  const [loading, setLoading] = useState(true);
  const [verifyingId, setVerifyingId] = useState<number | null>(null);
  // Der Belegprüflauf läuft auf Knopfdruck und nicht beim Öffnen: Er öffnet
  // jede abgelegte Datei einzeln und dauert bei einem vollen Geschäftsjahr
  // spürbar länger als die übrigen Abfragen dieser Seite.
  const [fileCheck, setFileCheck] = useState<FileCheckResult | null>(null);
  const [checkingFiles, setCheckingFiles] = useState(false);
  const [fileCheckError, setFileCheckError] = useState('');

  useEffect(() => {
    void loadData();
  }, []);

  async function loadData() {
    setLoading(true);
    try {
      const [festschreibungen, runs] = await Promise.all([
        Api.getFestschreibungen(),
        // Jahr 0 heißt: das aktive Geschäftsjahr.
        Api.getCheckRuns(0),
      ]);
      setCommitments(festschreibungen);
      setCheckRuns(runs);
      // Kommt der Aufruf aus einer Aufgabe, wird der jüngste Lauf mit einem
      // Befund dieser Regel aufgeklappt. Die Läufe stehen in absteigender
      // Reihenfolge, der erste Treffer ist deshalb der jüngste.
      if (initialRule) {
        const hit = runs.find((run) =>
          (run.findings ?? []).some((finding) => finding.rule === initialRule),
        );
        if (hit) setExpandedRun(hit.id);
      }
    } catch (e) {
      toast.error(message(e));
    } finally {
      setLoading(false);
    }
  }

  async function handleCheckFiles() {
    setCheckingFiles(true);
    setFileCheckError('');
    try {
      const result = await Api.verifyReceiptFiles();
      setFileCheck(result);
      if (result.isValid) toast.success(`${result.checked} Dateien unverändert.`);
    } catch (e) {
      // Der Fehler bleibt am Abschnitt stehen: Der Lauf dauert, und wer daneben
      // weiterliest, hätte einen Toast samt Grund verpasst (§10.4).
      setFileCheckError(message(e));
    } finally {
      setCheckingFiles(false);
    }
  }

  async function handleVerifyCommitment(id: number) {
    setVerifyingId(id);
    try {
      const result = await Api.verifyFestschreibung(id);
      if (result?.isValid) toast.success('Festschreibung gültig.');
      else toast.error(result?.message ?? 'Festschreibung ungültig.');
    } catch (e) {
      toast.error(message(e));
    } finally {
      setVerifyingId(null);
    }
  }

  const broken = integrity !== null && !integrity.isValid;

  if (loading) return <SkeletonRows rows={6} />;

  return (
    <>
      {integrity && (
        <StatRow>
          <Stat
            label="Zustand der Kette"
            value={broken ? 'Verletzt' : 'Unverändert'}
            context={`Geprüft am ${formatDateTime(integrity.checkedAt)}`}
            tone={broken ? 'negative' : 'positive'}
          />
          <Stat
            label="Geprüfte Buchungen"
            value={`${integrity.checkedEntries} von ${integrity.totalEntries}`}
            context={
              integrity.fiscalYears.length > 0
                ? `Geschäftsjahre ${integrity.fiscalYears.join(', ')}`
                : 'Hash-Kette vollständig durchlaufen'
            }
          />
          {/* Die Protokollkette richtet sich nach demselben Prüflauf: eine
              Journalkette, die hält, während sich Protokolleinträge entfernen
              ließen, beantwortet die Frage nur halb (UNV-03). Fehlt das
              Ergebnis, steht hier ein Strich statt einer Behauptung. */}
          <Stat
            label="Änderungsprotokoll"
            value={
              integrity.auditChain
                ? integrity.auditChain.isValid
                  ? 'Unverändert'
                  : 'Verletzt'
                : '—'
            }
            context={
              integrity.auditChain
                ? `${integrity.auditChain.checkedEntries} Einträge nachgerechnet`
                : 'nicht mitgeprüft'
            }
            tone={
              integrity.auditChain
                ? integrity.auditChain.isValid
                  ? 'positive'
                  : 'negative'
                : 'neutral'
            }
          />
          <Stat
            label="Festgeschriebene Zeiträume"
            value={String(commitments.length)}
            context="Abschluss über Steuerfristen"
          />
        </StatRow>
      )}

      <Section helpSummary="Die Prüfung zeigt, ob alle abgelegten Dateien vorhanden und unverändert sind."
        title="Belege prüfen"
        context="Sind alle Belege noch da und unverändert?"
        explain={
          <>
            Der Lauf öffnet jede abgelegte Rechnung und jedes Dokument und vergleicht sie mit
            dem Stand beim Ablegen. So fällt auf, wenn eine Datei fehlt oder die Festplatte sie
            beschädigt hat, bevor der Prüfer sie sehen will.
          </>
        }
        action={
          <Button
            variant="secondary"
            loading={checkingFiles}
            onClick={() => void handleCheckFiles()}
            icon={<ShieldCheck className="w-4 h-4" strokeWidth={1.5} />}
          >
            Dateien prüfen
          </Button>
        }
      >
        {fileCheckError && <Notice tone="negative" className="mb-5" text={fileCheckError} />}

        {!fileCheck ? (
          // Kein Leerzustand mit Kasten: Hier fehlen keine Daten, es ist
          // nur noch nichts gelaufen — und der Kasten wäre das Größte auf
          // einer Seite, auf der er meistens so steht.
          <p className="text-body text-ink-subtle">
            Noch nicht geprüft. Der Lauf dauert bei vielen Belegen einen Moment.
          </p>
        ) : (
          <>
            <StatRow>
              <Stat
                label="Geprüfte Dateien"
                value={String(fileCheck.checked)}
                context={`Zuletzt ${formatDateTime(fileCheck.checkedAt)}`}
              />
              <Stat
                label="Unversehrt"
                value={String(fileCheck.intact)}
                tone={fileCheck.isValid ? 'positive' : 'neutral'}
              />
              <Stat
                label="Beschädigt oder fehlend"
                value={String(fileCheck.damaged + fileCheck.missing)}
                context={`${fileCheck.damaged} beschädigt · ${fileCheck.missing} fehlend`}
                tone={fileCheck.isValid ? 'neutral' : 'negative'}
              />
            </StatRow>

            {fileCheck.issues.length > 0 && (
              <div className="mt-6">
                <Table density="kompakt">
                  <Thead>
                    <Tr>
                      <Th className="w-36">Art</Th>
                      <Th className="w-32">Beleg</Th>
                      <Th>Datei</Th>
                      <Th className="w-32">Befund</Th>
                    </Tr>
                  </Thead>
                  <Tbody>
                    {fileCheck.issues.map((issue) => (
                      <Tr key={`${issue.path}-${issue.fileName}`}>
                        <Td className="text-ink-muted">{issueKindLabel(issue.kind)}</Td>
                        <Td code>{issue.receiptNumber || '—'}</Td>
                        <Td className="text-ink-muted">{issue.fileName}</Td>
                        <Td className="text-negative-text">
                          <span className="flex items-center gap-1.5">
                            <OctagonAlert className="w-3.5 h-3.5 shrink-0" strokeWidth={1.5} />
                            {issueReasonLabel(issue.reason)}
                          </span>
                        </Td>
                      </Tr>
                    ))}
                  </Tbody>
                </Table>
              </div>
            )}
          </>
        )}
      </Section>

      {/* Jeder Bruch mit erwartetem und tatsächlichem Hash: erst damit lässt
          sich außerhalb von Buchfink nachrechnen, welche Seite recht hat
          (UNV-01). */}
      {integrity && integrity.breaks.length > 0 && (
        <Section helpSummary="Hier sehen Sie Buchungen, deren gespeicherter Prüfwert nicht mehr stimmt."
          title="Abweichende Buchungen"
          context="Erwarteter und tatsächlicher Hash je Bruch der Kette"
          explain={
            <>
              Jede Buchung hat den Hash ihres Vorgängers und einen eigenen über ihre Felder.
              Weicht der Vorgängerhash ab, wurde eine Buchung eingefügt oder entfernt; weicht
              der eigene ab, wurde die Buchung selbst verändert. Wie der Wert gebildet wird,
              steht in der Feldbeschreibung der Datenüberlassung.
            </>
          }
        >
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th className="w-28">Jahr</Th>
                <Th className="w-40">Buchung</Th>
                <Th className="w-44">Befund</Th>
                <Th>Erwartet</Th>
                <Th>Tatsächlich</Th>
              </Tr>
            </Thead>
            <Tbody>
              {integrity.breaks.map((issue) => (
                <Tr key={`${issue.entryId}-${issue.reason}`}>
                  <Td className="num text-ink-subtle">{issue.fiscalYear}</Td>
                  <Td code>{issue.entryNumber || issue.entryId}</Td>
                  <Td className="text-negative-text" title={issue.message}>
                    {issue.reason === 'linkage' ? 'Kette unterbrochen' : 'Buchung verändert'}
                  </Td>
                  {/* Der volle Hash, nicht die Kurzform: nachrechnen lässt
                      sich die Kette nur mit allen 64 Zeichen, und zwar
                      außerhalb von Buchfink (UNV-01). Er bricht um, statt
                      abgeschnitten zu werden. */}
                  <Td className="font-mono text-caption text-ink-muted break-all whitespace-normal">
                    {issue.expectedHash || '—'}
                  </Td>
                  <Td className="font-mono text-caption text-ink-muted break-all whitespace-normal">
                    {issue.actualHash || '—'}
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        </Section>
      )}

      <Section helpSummary="Abgeschlossene Zeiträume sind für weitere rückdatierte Buchungen gesperrt."
        title="Festgeschriebene Zeiträume"
        context="Abgeschlossen und von einem Zeitstempeldienst beglaubigt"
        explain={
          <>
            Beim Festschreiben werden die Buchungen eines Zeitraums verbindlich abgeschlossen,
            spätere Änderungen sind nur noch per Storno möglich. Zusätzlich beglaubigt ein
            unabhängiger Zeitstempeldienst nach RFC 3161 den Stand. Festgeschrieben wird unter
            Steuerfristen.
          </>
        }
      >
        {commitments.length === 0 ? (
          <EmptyState
            icon={<Lock className="w-6 h-6" strokeWidth={1.5} />}
            title="Noch kein Zeitraum festgeschrieben"
            description="Nach der Umsatzsteuer-Voranmeldung lässt sich der jeweilige Zeitraum unter Steuerfristen abschließen."
          />
        ) : (
          <Table>
            <Thead>
              <Tr>
                <Th>Zeitraum</Th>
                <Th>Stichtag</Th>
                <Th numeric>Buchungen</Th>
                <Th>Zeitstempel</Th>
                <Th className="w-24" aria-label="Aktionen" />
              </Tr>
            </Thead>
            <Tbody>
              {commitments.map((fs) => {
                const confirmed = fs.timestampStatus === 'confirmed' && fs.tsaGenTime;
                return (
                  <Tr key={fs.id} className="group">
                    <Td>
                      <span className="flex items-center gap-2">
                        <Lock className="w-3.5 h-3.5 shrink-0 text-ink-faint" strokeWidth={1.5} />
                        {fs.periodLabel}
                      </span>
                    </Td>
                    <Td className="text-ink-subtle num">{formatDate(fs.cutoffDate)}</Td>
                    <Td numeric>{fs.entryCount}</Td>
                    <Td className={cn('text-caption', confirmed ? 'text-ink-muted' : 'text-attention-text')}>
                      <span className="flex items-center gap-1.5">
                        <Clock className="w-3.5 h-3.5 shrink-0" strokeWidth={1.5} />
                        {confirmed
                          ? `${formatDateTime(fs.tsaGenTime!)} · ${fs.tsaName}`
                          : 'Ausstehend, wird nachgeholt'}
                      </span>
                      {/* Die beglaubigte Zeit steht hier, um mit der
                          Systemzeit verglichen zu werden (Entscheidung 8).
                          Wich sie beim Festschreiben ab, ist das der
                          einzige Ort, an dem der Befund später auffällt. */}
                      {fs.timeDriftNote && (
                        <span className="mt-1 flex text-attention-text">{fs.timeDriftNote}</span>
                      )}
                    </Td>
                    <Td className="pl-0">
                      <Button
                        variant="quiet"
                        size="sm"
                        loading={verifyingId === fs.id}
                        onClick={() => handleVerifyCommitment(fs.id)}
                        className="opacity-0 transition-opacity duration-120 ease-quiet
                                   group-hover:opacity-100 focus-visible:opacity-100"
                      >
                        Prüfen
                      </Button>
                    </Td>
                  </Tr>
                );
              })}
            </Tbody>
          </Table>
        )}
      </Section>

      <Section helpSummary="Vor dem Abschluss zeigt Buchfink offene Fragen und Fehler in Ihrer Buchhaltung."
        title="Prüfläufe"
        context="Der Bericht, der vor jeder Festschreibung läuft"
        explain={
          <>
            Der Prüflauf sagt vor der Festschreibung, was danach nicht mehr zu ändern wäre:
            Buchungen ohne Beleg, nicht zugeordnete Bankumsätze, Salden auf den Interimskonten.
            Blockierende Befunde verhindern die Festschreibung; übergangen werden sie nur mit
            einer Begründung, und die steht dann hier.
          </>
        }
      >
        {checkRuns.length === 0 ? (
          <EmptyState
            title="Noch kein Prüflauf"
            description="Ein Lauf entsteht mit der Festschreibung eines Zeitraums unter Steuerfristen."
          />
        ) : (
          <Table>
            <Thead>
              <Tr>
                <Th className="w-44">Zeitpunkt</Th>
                <Th className="w-32">Stichtag</Th>
                <Th numeric className="w-56">Geprüft</Th>
                <Th className="w-44">Befunde</Th>
                <Th>Übergangen mit Begründung</Th>
              </Tr>
            </Thead>
            <Tbody>
              {checkRuns.map((run) => {
                // Der Standardwert doppelt die Normalisierung in api.ts:
                // eine fehlende Befundliste aus dem Backend bräche schon
                // beim ersten `filter` die ganze Tabelle.
                const findings = run.findings ?? [];
                const blocking = findings.filter((f) => f.severity === 'blocking');
                const warnings = findings.filter((f) => f.severity === 'warning');
                // Aufklappbar ist ein Lauf auch ohne Befund: die Abstände
                // Belegdatum → Erfassung → Festschreibung stehen dort.
                const details = run.timeliness ? timelinessLines(run.timeliness) : [];
                const open = expandedRun === run.id;
                return (
                  <React.Fragment key={run.id}>
                    <Tr>
                      <Td className="text-ink-subtle num">
                        <button
                          type="button"
                          onClick={() => setExpandedRun(open ? null : run.id)}
                          aria-expanded={open}
                          disabled={findings.length === 0 && details.length === 0}
                          title={
                            findings.length === 0 && details.length === 0
                              ? 'Zu diesem Lauf sind weder Befunde noch Kennzahlen gespeichert.'
                              : undefined
                          }
                          className="inline-flex items-center gap-1 text-ink-muted
                                     hover:text-ink transition-colors duration-120 ease-quiet
                                     disabled:text-ink-faint"
                        >
                          {open ? (
                            <ChevronDown className="w-3.5 h-3.5" strokeWidth={1.5} />
                          ) : (
                            <ChevronRight className="w-3.5 h-3.5" strokeWidth={1.5} />
                          )}
                          {formatDateTime(run.createdAt)}
                        </button>
                      </Td>
                      <Td className="text-ink-subtle num">{formatDate(run.cutoffDate)}</Td>
                      <Td numeric className="text-ink-muted">
                        {run.checkedEntries} · {run.checkedReceipts} · {run.checkedBankTx}
                      </Td>
                      <Td
                        className={cn(
                          'text-caption',
                          blocking.length > 0 ? 'text-negative-text' : 'text-ink-muted',
                        )}
                      >
                        {blocking.length} blockierend · {warnings.length} Hinweise
                      </Td>
                      <Td className="whitespace-normal text-ink-muted">
                        {run.overrideReason || '—'}
                      </Td>
                    </Tr>
                    {open && details.length > 0 && (
                      <Tr>
                        <Td />
                        <Td className="text-ink-subtle">Abstände</Td>
                        <Td colSpan={3} className="whitespace-normal text-ink-muted">
                          {details.map((line) => (
                            <span key={line} className="block">
                              {line}
                            </span>
                          ))}
                        </Td>
                      </Tr>
                    )}
                    {open &&
                      findings.map((finding) => (
                        <Tr
                          key={finding.id}
                          className={
                            initialRule && finding.rule === initialRule
                              ? 'bg-attention-soft'
                              : undefined
                          }
                        >
                          <Td />
                          <Td
                            className={
                              finding.severity === 'blocking'
                                ? 'text-negative-text'
                                : 'text-attention-text'
                            }
                          >
                            {finding.severity === 'blocking' ? 'Blockierend' : 'Hinweis'}
                          </Td>
                          <Td colSpan={3} className="whitespace-normal text-ink-muted">
                            {finding.message}
                            {finding.reference && (
                              <Help summary="Hier erfahren Sie, auf welcher Anforderung dieser Prüfhinweis beruht." label="Hintergrund zum Prüfhinweis">{finding.reference}</Help>
                            )}
                          </Td>
                        </Tr>
                      ))}
                  </React.Fragment>
                );
              })}
            </Tbody>
          </Table>
        )}
      </Section>
    </>
  );
};

// -------------------------------------------------------------------------
// Änderungsprotokoll mit Vorher/Nachher und Kettenprüfung
// -------------------------------------------------------------------------

/** Die Vorgangsarten des Protokolls in Klartext. */
const ACTION_LABELS: Record<string, string> = {
  CREATE: 'Angelegt',
  UPDATE: 'Geändert',
  STORNO: 'Storniert',
  IMPORT: 'Übernommen',
  INTEGRITY_CHECK: 'Geprüft',
  EXPORT: 'Ausgegeben',
};

function actionLabel(action: string): string {
  return ACTION_LABELS[action] ?? action;
}

/**
 * Dieselben Vorgangsarten im Filter, mit einem anderen Namen für EXPORT.
 *
 * Ein Lesezugriff auf personenbezogene Daten — Datenüberlassung, Prüferpaket,
 * Herausgabe einzelner Dateien — wird als Ausgabe protokolliert. Das Ein- und
 * Ausschalten des Prüfermodus steht dagegen als Änderung am Bereich READ_ONLY
 * und ist über den Bereichsfilter zu finden.
 * In der Zeile heißt das „Ausgegeben"; gesucht wird danach aber unter dem
 * Namen, den Art. 30 DSGVO und die Verfahrensdokumentation dafür verwenden:
 * Zugriffe (QUE-02 K2).
 */
const FILTER_ACTION_LABELS: Record<string, string> = {
  ...ACTION_LABELS,
};

/** Der Wert im Vorgangsfilter, der nicht auf eine Vorgangsart, sondern auf AuditFilter.access abbildet. */
const ACCESS_FILTER = 'access';

/**
 * Die geänderten Felder aus dem Protokoll.
 *
 * Der Eintrag enthält sie als JSON-Text, weil er verschlüsselt gespeichert wird
 * und die Struktur je Objektart eine andere ist. Ein defekter Text darf die
 * Zeile nicht mitreißen — dann bleibt die Aufklappung eben leer.
 */
function parseChange(json: string | undefined): Record<string, unknown> {
  if (!json) return {};
  try {
    const parsed: unknown = JSON.parse(json);
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>;
    }
  } catch {
    return {};
  }
  return {};
}

/**
 * Vereinigt die bekannten Werte mit den neu gesehenen, ohne Leereintrag und in
 * fester Reihenfolge — sonst sprängen die Einträge einer Auswahlliste bei
 * jedem Laden.
 */
function union(known: string[], seen: string[]): string[] {
  const all = new Set(known);
  for (const value of seen) {
    if (value.trim() !== '') all.add(value);
  }
  return Array.from(all).sort();
}

/** Ein Wert aus dem Protokoll, lesbar gemacht. Leer heißt: nicht belegt. */
function changeValue(value: unknown): string {
  if (value === undefined || value === null || value === '') return '—';
  if (typeof value === 'boolean') return value ? 'ja' : 'nein';
  if (typeof value === 'number' || typeof value === 'string') return String(value);
  return JSON.stringify(value);
}

const ProtocolPanel: React.FC = () => {
  const [entries, setEntries] = useState<AuditLogEntry[]>([]);
  const [chain, setChain] = useState<AuditChainResult | null>(null);
  const [filter, setFilter] = useState<AuditFilter>({});
  // Die Auswahllisten wachsen mit dem, was schon einmal geladen war, und
  // schrumpfen nie: sonst verschwände der eben gewählte Bereich aus seiner
  // eigenen Auswahl, sobald der Filter greift.
  const [seenTypes, setSeenTypes] = useState<string[]>([]);
  const [seenActors, setSeenActors] = useState<string[]>([]);
  const [expanded, setExpanded] = useState<number | null>(null);
  const [loading, setLoading] = useState(true);
  const [verifying, setVerifying] = useState(false);
  const [error, setError] = useState('');

  // Nur die gefilterten Einträge: die Kettenprüfung rechnet das gesamte
  // Protokoll nach und richtet sich nicht nach dem Filter — sie beim Anwenden
  // jedes Filters mitlaufen zu lassen, macht jede Filteränderung so langsam wie
  // das ganze Protokoll lang ist. Sie läuft einmal beim Öffnen und auf
  // Knopfdruck.
  const load = useCallback(async (active: AuditFilter) => {
    setLoading(true);
    setError('');
    try {
      const logs = await Api.getAuditLogsFiltered(500, active);
      setEntries(logs);
      setSeenTypes((known) => union(known, logs.map((entry) => entry.entityType)));
      setSeenActors((known) => union(known, logs.map((entry) => entry.actor ?? '')));
    } catch (e) {
      setEntries([]);
      setError(message(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load({});
    // Der Zustand der Kette beim Öffnen des Reiters. Schlägt er fehl, bleibt
    // die Liste trotzdem stehen: das Protokoll zu lesen ist auch dann möglich.
    Api.verifyAuditChain()
      .then(setChain)
      .catch((e) => {
        setChain(null);
        setError(message(e));
      });
  }, [load]);

  const filtered =
    filter.action !== undefined ||
    filter.access !== undefined ||
    filter.entityType !== undefined ||
    filter.actor !== undefined ||
    filter.from !== undefined ||
    filter.to !== undefined;

  async function reverify() {
    setVerifying(true);
    try {
      const result = await Api.verifyAuditChain();
      setChain(result);
      if (result.isValid) toast.success(`${result.checkedEntries} Protokolleinträge unverändert.`);
      else toast.error(result.message);
    } catch (e) {
      setError(message(e));
    } finally {
      setVerifying(false);
    }
  }

  function applyFilter(next: AuditFilter) {
    setFilter(next);
    setExpanded(null);
    void load(next);
  }

  const breaks = chain?.breaks ?? [];

  return (
    <>
      {error && <Notice tone="negative" text={error} className="mb-6" />}
      {chain && !chain.isValid && <Notice tone="negative" text={chain.message} className="mb-6" />}

      {/* Der Zustand der Kette steht als Kennzahl unter „Prüfungen" und hier
          nur, wo er hingehört: bei den Brüchen selbst. Dieselbe Zahl an zwei
          Stellen läuft auseinander, sobald eine der beiden nachgerechnet
          wurde. */}
      {breaks.length > 0 && (
        <Section
          title="Brüche der Protokollkette"
          divider={false}
          context="Erwarteter und tatsächlicher Hash"
        >
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th className="w-20">Eintrag</Th>
                <Th className="w-32">Art</Th>
                <Th>Befund</Th>
                <Th className="w-40">Erwartet</Th>
                <Th className="w-40">Tatsächlich</Th>
              </Tr>
            </Thead>
            <Tbody>
              {breaks.map((entry) => (
                <Tr key={entry.entryId}>
                  <Td className="num">{entry.entryId}</Td>
                  <Td className="text-ink-muted">
                    {entry.reason === 'linkage' ? 'Verkettung' : 'Inhalt'}
                  </Td>
                  <Td className="whitespace-normal">{entry.message}</Td>
                  <Td className="code-num text-ink-subtle">{formatShortHash(entry.expectedHash)}</Td>
                  <Td className="code-num text-ink-subtle">{formatShortHash(entry.actualHash)}</Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        </Section>
      )}

      <Section helpSummary="Hier können Sie nachvollziehen, wer welche Daten geändert hat."
        title="Änderungsprotokoll"
        divider={breaks.length > 0}
        context={
          chain
            ? `${entries.length} Einträge · Kette am ${formatDateTime(chain.checkedAt)} nachgerechnet`
            : `${entries.length} Einträge · neueste zuerst`
        }
        explain={
          <>
            Jeder Eintrag hält Vorher und Nachher der geänderten Felder fest und ist mit seinem
            Vorgänger verkettet (GoBD Rz. 34). Wird ein Eintrag verändert oder entfernt, bricht
            die Kette an dieser Stelle. Die Prüfung läuft nach einem Bruch weiter, damit nicht die
            erste Änderung jede spätere verdeckt.
          </>
        }
        action={
          <Button
            variant="secondary"
            loading={verifying}
            onClick={() => void reverify()}
            icon={<ShieldCheck className="w-4 h-4" strokeWidth={1.5} />}
          >
            Kette prüfen
          </Button>
        }
      >
        <ProtocolFilter
          filter={filter}
          entityTypes={seenTypes}
          actors={seenActors}
          onApply={applyFilter}
        />

        {loading ? (
          <SkeletonRows rows={10} />
        ) : entries.length === 0 ? (
          <EmptyState
            variant={filtered ? 'gefiltert' : 'leer'}
            title={filtered ? 'Keine Einträge für diesen Filter' : 'Noch keine Einträge'}
            description={
              filtered ? undefined : 'Das Protokoll füllt sich mit der ersten Änderung.'
            }
            action={
              filtered ? (
                <Button variant="secondary" onClick={() => applyFilter({})}>
                  Filter zurücksetzen
                </Button>
              ) : undefined
            }
          />
        ) : (
          <Table density="kompakt">
            <Thead sticky>
              <Tr>
                <Th className="w-8" aria-label="Vorher und Nachher" />
                <Th className="w-52">Zeitpunkt</Th>
                <Th className="w-28">Vorgang</Th>
                <Th className="w-32">Bereich</Th>
                <Th>Beschreibung</Th>
                <Th className="w-44">Bearbeiter</Th>
                <Th className="w-24">Fassung</Th>
              </Tr>
            </Thead>
            <Tbody>
              {entries.map((entry) => {
                const before = parseChange(entry.before);
                const after = parseChange(entry.after);
                const fields = Array.from(
                  new Set([...Object.keys(before), ...Object.keys(after)]),
                ).sort();
                const open = expanded === entry.id;

                return (
                  <React.Fragment key={entry.id}>
                    <Tr>
                      <Td>
                        {/* Ein echter Knopf und keine anklickbare Zeile: die
                            Aufklappung muss mit der Tastatur erreichbar sein
                            und ihren Zustand ansagen (§14). */}
                        <button
                          type="button"
                          onClick={() => setExpanded(open ? null : entry.id)}
                          aria-expanded={open}
                          aria-label={open ? 'Änderung zuklappen' : 'Änderung aufklappen'}
                          disabled={fields.length === 0}
                          title={
                            fields.length === 0
                              ? 'Dieser Vorgang hat kein Feld geändert.'
                              : undefined
                          }
                          className="text-ink-faint hover:text-ink transition-colors duration-120
                                     ease-quiet disabled:opacity-0"
                        >
                          {open ? (
                            <ChevronDown className="w-3.5 h-3.5" strokeWidth={1.5} />
                          ) : (
                            <ChevronRight className="w-3.5 h-3.5" strokeWidth={1.5} />
                          )}
                        </button>
                      </Td>
                      <Td className="text-ink-subtle num">{formatDateTime(entry.timestamp)}</Td>
                      <Td>
                        <span className="inline-flex items-center h-5 px-2 rounded-control border border-line-strong text-caption text-ink-muted">
                          {actionLabel(entry.action)}
                        </span>
                      </Td>
                      <Td className="text-ink-muted">{entry.entityType}</Td>
                      <Td className="whitespace-normal">{entry.details}</Td>
                      <Td className="text-ink-subtle">{entry.actor || '—'}</Td>
                      <Td className="text-ink-subtle num">{entry.appVersion || '—'}</Td>
                    </Tr>

                    {open && fields.length > 0 && (
                      <Tr>
                        <Td colSpan={7} className="bg-sunken">
                          <table className="w-full text-body">
                            <thead>
                              <tr className="text-label text-ink-subtle text-left">
                                <th className="w-52 font-medium py-1">Feld</th>
                                <th className="font-medium py-1">Vorher</th>
                                <th className="font-medium py-1">Nachher</th>
                              </tr>
                            </thead>
                            <tbody>
                              {fields.map((field) => (
                                <tr key={field} className="align-top">
                                  <td className="py-1 pr-4 text-ink-subtle">{field}</td>
                                  <td className="py-1 pr-4 text-ink-muted break-all">
                                    {changeValue(before[field])}
                                  </td>
                                  <td className="py-1 text-ink break-all">
                                    {changeValue(after[field])}
                                  </td>
                                </tr>
                              ))}
                            </tbody>
                          </table>
                        </Td>
                      </Tr>
                    )}
                  </React.Fragment>
                );
              })}
            </Tbody>
          </Table>
        )}
      </Section>
    </>
  );
};

/**
 * Der Filter über das Protokoll.
 *
 * Er schränkt im Backend ein und nicht in der Ansicht: das Protokoll wächst mit
 * jeder Buchung, jedem Export und jeder Sicherung, und zehntausend Zeilen zu
 * laden, um darin fünf zu suchen, wäre der falsche Weg.
 */
const ProtocolFilter: React.FC<{
  filter: AuditFilter;
  /** Die Bereiche und Bearbeiter, die im Protokoll vorkommen. */
  entityTypes: string[];
  actors: string[];
  onApply: (filter: AuditFilter) => void;
}> = ({ filter, entityTypes, actors, onApply }) => {
  const [draft, setDraft] = useState<AuditFilter>(filter);

  useEffect(() => {
    setDraft(filter);
  }, [filter]);

  function trimmed(value: string | undefined): string | undefined {
    const text = (value ?? '').trim();
    return text === '' ? undefined : text;
  }

  return (
    <div className="mb-5 flex flex-wrap items-end gap-3">
      <div className="w-44">
        <Field helpSummary="Filtern Sie die Liste nach der Art des protokollierten Vorgangs."
          label="Vorgang"
          explain={
            '„Zugriffe“ sind die Lesezugriffe auf personenbezogene Daten: Datenüberlassung, ' +
            'Prüferpaket, die Herausgabe einzelner Dateien und der Prüfermodus.'
          }
        >
          <Select<string>
            items={[
              { value: '', label: 'Alle' },
              { value: ACCESS_FILTER, label: 'Zugriffe' },
              ...Object.entries(FILTER_ACTION_LABELS).map(([value, label]) => ({ value, label })),
            ]}
            value={draft.access ? ACCESS_FILTER : (draft.action ?? '')}
            onValueChange={(value) =>
              setDraft({
                ...draft,
                access: value === ACCESS_FILTER ? true : undefined,
                action: value && value !== ACCESS_FILTER ? value : undefined,
              })
            }
            aria-label="Vorgang"
          />
        </Field>
      </div>
      <div className="w-52">
        <Field label="Bereich">
          <Select<string>
            items={[
              { value: '', label: 'Alle' },
              ...entityTypes.map((value) => ({ value, label: value })),
            ]}
            value={draft.entityType ?? ''}
            onValueChange={(value) => setDraft({ ...draft, entityType: value || undefined })}
            aria-label="Bereich"
          />
        </Field>
      </div>
      <div className="w-56">
        <Field label="Bearbeiter">
          <Select<string>
            items={[
              { value: '', label: 'Alle' },
              ...actors.map((value) => ({ value, label: value })),
            ]}
            value={draft.actor ?? ''}
            onValueChange={(value) => setDraft({ ...draft, actor: value || undefined })}
            aria-label="Bearbeiter"
          />
        </Field>
      </div>
      <div className="w-40">
        <Field label="Von">
          <Input
            type="date"
            value={draft.from ?? ''}
            onChange={(e) => setDraft({ ...draft, from: e.target.value })}
          />
        </Field>
      </div>
      <div className="w-40">
        <Field label="Bis">
          <Input
            type="date"
            value={draft.to ?? ''}
            onChange={(e) => setDraft({ ...draft, to: e.target.value })}
          />
        </Field>
      </div>
      <div className="flex items-center gap-2 pb-0.5">
        <Button
          variant="secondary"
          onClick={() =>
            onApply({
              action: trimmed(draft.action),
              access: draft.access ? true : undefined,
              entityType: trimmed(draft.entityType),
              actor: trimmed(draft.actor),
              from: trimmed(draft.from),
              to: trimmed(draft.to),
            })
          }
        >
          Filtern
        </Button>
        <Button variant="quiet" onClick={() => onApply({})}>
          Zurücksetzen
        </Button>
      </div>
    </div>
  );
};

// -------------------------------------------------------------------------
// Systemhistorie: Fassungen, Schemaläufe, Übernahmen
// -------------------------------------------------------------------------

const MIGRATION_KIND_LABELS: Record<string, string> = {
  import: 'Mandant übernommen',
  restore: 'Aus Sicherung wiederhergestellt',
  open: 'Vorhandene Datei geöffnet',
};

/**
 * Ein Befund in Worten: das Ergebnis eines Laufs.
 *
 * Bewusst kein StatusBadge. Dessen Vokabular (§11.3) ist abschließend und
 * beschreibt Zustände von Buchungen, Rechnungen und Perioden; ein Schemalauf
 * und eine Kettenprüfung haben keinen davon.
 */
const VERDICT_TONE = {
  neutral: 'text-ink-muted',
  positive: 'text-positive-text',
  negative: 'text-negative-text',
} as const;

const Verdict: React.FC<{ text: string; tone?: keyof typeof VERDICT_TONE }> = ({
  text,
  tone = 'neutral',
}) => <span className={`text-body whitespace-nowrap ${VERDICT_TONE[tone]}`}>{text}</span>;

const HistoryPanel: React.FC = () => {
  const [changelog, setChangelog] = useState<ChangelogEntry[]>([]);
  const [schema, setSchema] = useState<SchemaMigration[]>([]);
  const [records, setRecords] = useState<MigrationRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [openingReleases, setOpeningReleases] = useState(false);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [log, migrations, migrationRecords] = await Promise.all([
        Api.getChangeLog(),
        Api.getSchemaMigrations(),
        Api.getMigrationRecords(),
      ]);
      setChangelog(log);
      setSchema(migrations);
      setRecords(migrationRecords);
    } catch (e) {
      setError(message(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  /**
   * Der Weg nach draußen. Scheitert er — kein Browser, kein Netz —, bleibt der
   * Grund am Abschnitt stehen: die Historie daneben ist die Antwort, die auch
   * ohne Netz gilt.
   */
  async function openReleases() {
    setOpeningReleases(true);
    setError('');
    try {
      await Api.openReleasesPage();
    } catch (e) {
      setError(message(e));
    } finally {
      setOpeningReleases(false);
    }
  }

  if (loading) return <SkeletonRows rows={8} />;

  return (
    <>
      {error && <Notice tone="negative" text={error} className="mb-6" />}

      <Section helpSummary="Hier sehen Sie, ob die Anpassung Ihrer Datenbank an die Programmversion gelungen ist."
        title="Schemaänderungen"
        divider={false}
        context="Jeder Lauf mit Programmfassung und betroffenen Tabellen"
        explain={
          <>
            Bei jedem Programmstart bringt Buchfink die Datenbank auf den Stand der laufenden
            Fassung. Ein fehlgeschlagener Lauf steht hier — er ist der Grund, aus dem eine
            Auswertung anders aussehen kann als am Tag zuvor.
          </>
        }
      >
        {schema.length === 0 ? (
          <EmptyState title="Noch kein Lauf protokolliert" />
        ) : (
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th className="w-52">Zeitpunkt</Th>
                <Th className="w-24">Fassung</Th>
                <Th className="w-28" numeric>
                  Schema
                </Th>
                <Th className="w-28">Ergebnis</Th>
                <Th>Tabellen</Th>
              </Tr>
            </Thead>
            <Tbody>
              {schema.map((run) => (
                <Tr key={run.id}>
                  <Td className="text-ink-subtle num">{formatDateTime(run.runAt)}</Td>
                  <Td className="num">{run.appVersion}</Td>
                  <Td numeric className="num">
                    {run.fromVersion} → {run.toVersion}
                  </Td>
                  <Td>
                    <Verdict
                      text={run.result === 'ok' ? 'Ausgeführt' : 'Fehlgeschlagen'}
                      tone={run.result === 'ok' ? 'neutral' : 'negative'}
                    />
                  </Td>
                  <Td className="whitespace-normal text-ink-muted">{run.tables || run.message}</Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>

      <Section helpSummary="Hier sehen Sie, welche Daten übernommen wurden und ob die Summen übereinstimmen."
        title="Datenübernahmen"
        context="Zählungen und Kettenprüfung zum Zeitpunkt der Übernahme"
        explain={
          <>
            Jede Übernahme zählt Buchungen, Belege und Stammdaten und stellt Soll und Haben
            gegenüber. Stimmen sie nicht überein, fehlt etwas — eine halb kopierte Datei fiele
            sonst erst auf, wenn die Bilanz nicht mehr aufgeht (ARC-05).
          </>
        }
      >
        {records.length === 0 ? (
          <EmptyState title="Keine Übernahme protokolliert" />
        ) : (
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th className="w-52">Zeitpunkt</Th>
                <Th className="w-56">Art</Th>
                <Th numeric className="w-24">
                  Buchungen
                </Th>
                <Th numeric className="w-24">
                  Belege
                </Th>
                <Th numeric className="w-32">
                  Soll
                </Th>
                <Th numeric className="w-32">
                  Haben
                </Th>
                <Th className="w-28">Kette</Th>
              </Tr>
            </Thead>
            <Tbody>
              {records.map((record) => (
                <Tr key={record.id}>
                  <Td className="text-ink-subtle num">{formatDateTime(record.runAt)}</Td>
                  <Td className="text-ink-muted">
                    {MIGRATION_KIND_LABELS[record.kind] ?? record.kind}
                  </Td>
                  <Td numeric className="num">
                    {record.journalEntries}
                  </Td>
                  <Td numeric className="num">
                    {record.receipts}
                  </Td>
                  <Td numeric className="num">
                    {formatCents(record.debitTotal)}
                  </Td>
                  <Td numeric className="num">
                    {formatCents(record.creditTotal)}
                  </Td>
                  <Td>
                    <Verdict
                      text={record.chainValid ? 'Unversehrt' : 'Gebrochen'}
                      tone={record.chainValid ? 'positive' : 'negative'}
                    />
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>

      <Section helpSummary="Hier sehen Sie, was sich zwischen den Programmversionen geändert hat."
        title="Versionshistorie"
        context="Jede Fassung des Programms mit Datum und Änderung"
        explain={
          <>
            Die Historie steckt im Programm und reist mit ihm: sie steht auch dort zur Verfügung,
            wo eine Sicherung auf einem fremden Rechner geprüft wird, und liegt jedem Prüferpaket
            bei. Ausführlicher — mit den einzelnen Änderungen je Fassung — steht sie im Netz.
          </>
        }
        action={
          <Button
            variant="quiet"
            loading={openingReleases}
            onClick={() => void openReleases()}
            icon={<ExternalLink className="w-4 h-4" strokeWidth={1.5} />}
          >
            Fassungen im Netz
          </Button>
        }
      >
        {changelog.length === 0 ? (
          <EmptyState title="Keine Versionshistorie hinterlegt" />
        ) : (
          // Keine Tabelle: eine Fassung ist eine Überschrift mit Fließtext
          // darunter und keine Zeile mit Spalten. In eine Zelle gepresst
          // stünde der längste Satz der Anwendung in einer Spalte von
          // Handbreite.
          <div className="space-y-8 max-w-3xl">
            {changelog.map((entry) => (
              <div key={`${entry.version}-${entry.date}`}>
                <h3 className="text-label text-ink">
                  {entry.version}
                  <span className="ml-2 font-normal text-ink-subtle num">{entry.date}</span>
                </h3>
                {entry.summary && (
                  <p className="mt-1.5 text-body text-ink-muted">{entry.summary}</p>
                )}
                <ul className="mt-3 space-y-1.5">
                  {entry.changes.map((change) => (
                    <li key={change} className="flex gap-2.5 text-body text-ink-muted">
                      {/* Dieselbe Raute wie im Zustandsanzeiger; andere
                          Aufzählungszeichen gibt es nicht (§10.4). */}
                      <span className="mark-diamond bg-line-strong mt-2 shrink-0" />
                      <span>{change}</span>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        )}
      </Section>
    </>
  );
};

export default AuditPage;
