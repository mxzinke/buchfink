import React, { useCallback, useEffect, useState } from 'react';
import {
  ChevronDown,
  ChevronRight,
  FileText,
  Pause,
  Play,
  RefreshCw,
  ShieldCheck,
  Trash2,
} from 'lucide-react';
import { Api } from '../services/api';
import { useWriteLock } from '../components/WriteLock';
import type { NavigateFn } from '../components/Sidebar';
import { RETENTION_CLASS_LABELS } from '../types';
import type {
  AuditChainResult,
  AuditFilter,
  AuditLogEntry,
  ComplianceHints,
  MigrationRecord,
  OrganisationTexts,
  ProcDocResult,
  ProcedureDocumentation,
  RetentionHold,
  RetentionHoldReason,
  RetentionOverview,
  RetentionYear,
  SchemaMigration,
} from '../types';
import {
  formatBytes,
  formatCents,
  formatDate,
  formatDateOfTimestamp,
  formatDateTime,
  formatShortHash,
} from '../utils/formatters';
import {
  Button,
  ConfirmDialog,
  Dialog,
  EmptyState,
  Field,
  FieldRow,
  HelpPopover,
  Input,
  Notice,
  PageHeader,
  Section,
  Select,
  SkeletonRows,
  Stat,
  StatRow,
  Table,
  TabPanel,
  Tabs,
  Tbody,
  Td,
  Textarea,
  Th,
  Thead,
  Tr,
  toast,
} from '../components/ui';

/**
 * Die Nachweise (Welle 6).
 *
 * Vier Sichten, die eine Frage beantworten: woran erkennt ein Prüfer, dass
 * diese Buchführung ordnungsmäßig ist. Das Änderungsprotokoll sagt, wer wann
 * was geändert hat; die Versionen sagen, womit gebucht wurde und was am Schema
 * geschah; die Aufbewahrung sagt, wie lange die Daten zu halten sind; die
 * Verfahrensdokumentation beschreibt das Verfahren selbst.
 *
 * Sie stehen zusammen und nicht bei der jeweiligen Arbeit, weil sie niemand
 * beim Buchen braucht und alle zusammen, sobald jemand fragt.
 */

type TabKey = 'protokoll' | 'versionen' | 'aufbewahrung' | 'verfahren';

const TAB_KEYS: TabKey[] = ['protokoll', 'versionen', 'aufbewahrung', 'verfahren'];

function isTabKey(value: string | undefined): value is TabKey {
  return value !== undefined && (TAB_KEYS as string[]).includes(value);
}

function message(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

/**
 * Ein Befund in Worten: das Ergebnis eines Laufs, der Stand einer Frist.
 *
 * Bewusst kein StatusBadge. Dessen Vokabular (§11.3) ist abschließend und
 * beschreibt Zustände von Buchungen, Rechnungen und Perioden; ein Schemalauf,
 * eine Kettenprüfung und eine Aufbewahrungsfrist haben keinen davon. Ein
 * Abzeichen mit fremder Beschriftung gäbe ihnen die Bedeutung des fremden
 * Zustands mit: Bernstein „Offen" heißt „noch zu erledigen", das Schloss der
 * Festschreibung heißt „nimmt nichts mehr an" — beides trifft auf eine laufende
 * Aufbewahrungsfrist nicht zu.
 */
const VERDICT_TONE = {
  neutral: 'text-ink-muted',
  attention: 'text-attention-text',
  positive: 'text-positive-text',
  negative: 'text-negative-text',
} as const;

const Verdict: React.FC<{
  text: string;
  tone?: keyof typeof VERDICT_TONE;
}> = ({ text, tone = 'neutral' }) => (
  <span className={`text-body whitespace-nowrap ${VERDICT_TONE[tone]}`}>{text}</span>
);

export interface NachweisePageProps {
  /** Das Geschäftsjahr aus der Kopfzeile — der Vorschlag der Fristenansicht. */
  year: number;
  /** Der Reiter, mit dem die Seite öffnet. */
  initialTab?: string;
  onNavigate?: NavigateFn;
}

export const NachweisePage: React.FC<NachweisePageProps> = ({ year, initialTab, onNavigate }) => {
  const [tab, setTab] = useState<TabKey>(isTabKey(initialTab) ? initialTab : 'protokoll');

  // Ein zweiter Verweis auf dieselbe Seite soll den Reiter wechseln: die Seite
  // bleibt montiert, wenn nur der Parameter wechselt.
  useEffect(() => {
    if (isTabKey(initialTab)) setTab(initialTab);
  }, [initialTab]);

  return (
    <div className="max-w-[1200px] mx-auto px-8 py-8">
      <PageHeader
        title="Nachweise"
        context="Protokoll, Fristen und Verfahrensdokumentation"
        action={
          onNavigate && (
            <Button variant="quiet" onClick={() => onNavigate('dataaccess')}>
              Datenüberlassung
            </Button>
          )
        }
      />

      <Tabs<TabKey>
        value={tab}
        onValueChange={setTab}
        className="mt-6"
        items={[
          { value: 'protokoll', label: 'Änderungsprotokoll' },
          { value: 'versionen', label: 'Versionen & Übernahmen' },
          { value: 'aufbewahrung', label: 'Aufbewahrung' },
          { value: 'verfahren', label: 'Verfahrensdokumentation' },
        ]}
      >
        {/* Jede Sicht lädt für sich: das Protokoll ist lang, die Fristen
            rechnen über alle Jahre, und niemand braucht beides auf einmal. */}
        <TabPanel value="protokoll">{tab === 'protokoll' && <ProtocolPanel />}</TabPanel>
        <TabPanel value="versionen">{tab === 'versionen' && <VersionsPanel />}</TabPanel>
        <TabPanel value="aufbewahrung">
          {tab === 'aufbewahrung' && <RetentionPanel year={year} />}
        </TabPanel>
        <TabPanel value="verfahren">{tab === 'verfahren' && <ProcDocPanel />}</TabPanel>
      </Tabs>
    </div>
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
 * Die geänderten Felder aus dem Protokoll.
 *
 * Der Eintrag trägt sie als JSON-Text, weil er verschlüsselt gespeichert wird
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
  // Protokoll nach und hängt am Filter nicht — sie beim Anwenden jedes Filters
  // mitlaufen zu lassen, macht jede Filteränderung so langsam wie das ganze
  // Protokoll lang ist. Sie läuft einmal beim Öffnen und auf Knopfdruck.
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
    // Der Zustand der Kette beim Öffnen der Seite. Schlägt er fehl, bleibt die
    // Liste trotzdem stehen: das Protokoll zu lesen ist auch dann möglich.
    Api.verifyAuditChain()
      .then(setChain)
      .catch((e) => {
        setChain(null);
        setError(message(e));
      });
  }, [load]);

  const filtered =
    filter.action !== undefined ||
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

      <StatRow>
        <Stat
          label="Protokolleinträge"
          value={String(chain?.totalEntries ?? 0)}
          context="seit Anlage des Mandanten"
        />
        <Stat
          label="Nachgerechnet"
          value={String(chain?.checkedEntries ?? 0)}
          context="Einträge der Kette"
        />
        {/* Ohne Ergebnis steht hier ein Strich und nicht „Unversehrt": eine
            Kette, die nicht nachgerechnet wurde, ist nicht geprüft. */}
        <Stat
          label="Kette"
          value={chain === null ? '—' : chain.isValid ? 'Unversehrt' : 'Gebrochen'}
          context={breaks.length > 0 ? `${breaks.length} Brüche` : 'ohne Befund'}
          tone={chain === null ? 'neutral' : chain.isValid ? 'positive' : 'negative'}
        />
        <Stat
          label="Zuletzt geprüft"
          value={chain?.checkedAt ? formatDateOfTimestamp(chain.checkedAt) : '—'}
          context={chain?.checkedAt ? formatDateTime(chain.checkedAt) : 'noch nicht nachgerechnet'}
        />
      </StatRow>

      {chain && !chain.isValid && <Notice tone="negative" text={chain.message} className="mt-6" />}

      {breaks.length > 0 && (
        <Section title="Brüche der Protokollkette" context="Erwarteter und tatsächlicher Hash">
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

      <Section
        title="Änderungsprotokoll"
        context={`${entries.length} Einträge · neueste zuerst`}
        action={
          <div className="flex items-center gap-2">
            <HelpPopover label="Erklärung zum Änderungsprotokoll">
              Jeder Eintrag hält Vorher und Nachher der geänderten Felder fest und ist mit seinem
              Vorgänger verkettet (GoBD Rz. 34). Wird ein Eintrag verändert oder entfernt, bricht
              die Kette an dieser Stelle. Die Prüfung läuft nach einem Bruch weiter, damit nicht die
              erste Änderung jede spätere verdeckt.
            </HelpPopover>
            <Button
              variant="secondary"
              loading={verifying}
              onClick={() => void reverify()}
              icon={<ShieldCheck className="w-4 h-4" strokeWidth={1.5} />}
            >
              Kette prüfen
            </Button>
          </div>
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
        <Field label="Vorgang">
          <Select<string>
            items={[
              { value: '', label: 'Alle' },
              ...Object.entries(ACTION_LABELS).map(([value, label]) => ({ value, label })),
            ]}
            value={draft.action ?? ''}
            onValueChange={(value) => setDraft({ ...draft, action: value || undefined })}
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
// Versionshistorie, Schemaänderungen und Datenübernahmen
// -------------------------------------------------------------------------

const MIGRATION_KIND_LABELS: Record<string, string> = {
  import: 'Mandant übernommen',
  restore: 'Aus Sicherung wiederhergestellt',
  open: 'Vorhandene Datei geöffnet',
};

const VersionsPanel: React.FC = () => {
  const lock = useWriteLock();
  const [changelog, setChangelog] = useState('');
  const [schema, setSchema] = useState<SchemaMigration[]>([]);
  const [records, setRecords] = useState<MigrationRecord[]>([]);
  const [hints, setHints] = useState<ComplianceHints | null>(null);
  const [changeDate, setChangeDate] = useState('');
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [log, migrations, migrationRecords, complianceHints] = await Promise.all([
        Api.getChangeLog(),
        Api.getSchemaMigrations(),
        Api.getMigrationRecords(),
        Api.getComplianceHints(),
      ]);
      setChangelog(log);
      setSchema(migrations);
      setRecords(migrationRecords);
      setHints(complianceHints);
      setChangeDate(complianceHints?.systemChangeDate ?? '');
    } catch (e) {
      setError(message(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function saveChangeDate() {
    setBusy(true);
    setError('');
    try {
      await Api.setSystemChangeDate(changeDate);
      await load();
      toast.success('Umstellungszeitpunkt gespeichert.');
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

      <Section
        title="Umstellung aus einem Altsystem"
        context={hints?.systemChangeNote || 'Kein Umstellungszeitpunkt hinterlegt'}
        divider={false}
        action={
          <HelpPopover label="Erklärung zur Fünfjahresfrist">
            Wer die Buchführung aus einem anderen System übernimmt, hält den Umstellungszeitpunkt
            fest. Ab ihm läuft die Fünfjahresfrist des § 147 Abs. 6 Satz 6 AO: so lange muss das
            Altsystem für den Datenzugriff verfügbar bleiben.
          </HelpPopover>
        }
      >
        <FieldRow>
          <Field label="Umstellungszeitpunkt" optional>
            <Input
              type="date"
              value={changeDate}
              onChange={(e) => setChangeDate(e.target.value)}
              disabled={lock.locked}
            />
          </Field>
          <div className="flex items-end pb-0.5">
            <Button
              variant="secondary"
              loading={busy}
              disabled={lock.locked}
              title={lock.hint}
              onClick={() => void saveChangeDate()}
            >
              Speichern
            </Button>
          </div>
        </FieldRow>
      </Section>

      <Section title="Versionshistorie" context="Jede Fassung mit Datum und Änderung">
        {changelog.trim() === '' ? (
          <EmptyState title="Keine Versionshistorie hinterlegt" />
        ) : (
          <pre className="max-h-96 overflow-y-auto rounded-control border border-line bg-surface p-4 text-caption text-ink-muted whitespace-pre-wrap">
            {changelog}
          </pre>
        )}
      </Section>

      <Section
        title="Schemaänderungen"
        context="Jeder Lauf mit Programmfassung und betroffenen Tabellen"
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

      <Section
        title="Datenübernahmen"
        context="Zählungen und Kettenprüfung zum Zeitpunkt der Übernahme"
        action={
          <HelpPopover label="Erklärung zur Abstimmung">
            Jede Übernahme zählt Buchungen, Belege und Stammdaten und stellt Soll und Haben
            gegenüber. Stimmen sie nicht überein, fehlt etwas — eine halb kopierte Datei fiele
            sonst erst auf, wenn die Bilanz nicht mehr aufgeht (ARC-05).
          </HelpPopover>
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
    </>
  );
};

// -------------------------------------------------------------------------
// Aufbewahrung: Fristen, Aussetzung, Löschung
// -------------------------------------------------------------------------

const HOLD_REASONS: { value: RetentionHoldReason; label: string }[] = [
  { value: 'audit', label: 'Außenprüfung' },
  { value: 'appeal', label: 'Rechtsbehelfsverfahren' },
  { value: 'other', label: 'Sonstiger Grund' },
];

function holdReasonLabel(reason: RetentionHoldReason): string {
  return HOLD_REASONS.find((entry) => entry.value === reason)?.label ?? 'Sonstiger Grund';
}

const RetentionPanel: React.FC<{ year: number }> = ({ year }) => {
  const lock = useWriteLock();
  const [overview, setOverview] = useState<RetentionOverview | null>(null);
  const [holds, setHolds] = useState<RetentionHold[]>([]);
  const [expired, setExpired] = useState<RetentionYear[]>([]);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  // Die Aussetzung, die Aufhebung und die Löschung führen je einen eigenen
  // Dialog: sie fragen Verschiedenes, und ein gemeinsamer Dialog mit drei
  // Zuständen wäre an jeder Stelle halb falsch.
  //
  // Jeder Dialog führt seinen eigenen Fehlerzustand. Der Fehler der Seite
  // stünde hinter dem Backdrop des offenen Dialogs, und der Dialog bliebe
  // ohne Meldung stehen — ein Klick, der sichtbar nichts bewirkt (§10.4).
  const [holdYear, setHoldYear] = useState<number | null>(null);
  const [holdReason, setHoldReason] = useState<RetentionHoldReason>('audit');
  const [holdText, setHoldText] = useState('');
  const [holdError, setHoldError] = useState('');
  const [releasing, setReleasing] = useState<RetentionHold | null>(null);
  const [releaseText, setReleaseText] = useState('');
  const [releaseError, setReleaseError] = useState('');
  const [deleting, setDeleting] = useState<RetentionYear | null>(null);
  const [confirmation, setConfirmation] = useState('');
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [deleteError, setDeleteError] = useState('');
  // Das Löschprotokoll bleibt stehen, bis es weggeklickt wird. Es ist ein
  // mehrzeiliges Ergebnis mit Zählungen und dem Pfad des Archivs — also etwas,
  // das weiterverwendet wird, und kein Vollzug, den man ohnehin sieht (§10.4).
  // Als Toast wäre der Archivpfad nach vier Sekunden nur noch im
  // Änderungsprotokoll zu finden.
  const [deleteReport, setDeleteReport] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [view, holdList, expiredYears] = await Promise.all([
        Api.getRetentionOverview(0),
        Api.getRetentionHolds(),
        Api.getExpiredObjects(),
      ]);
      setOverview(view);
      setHolds(holdList);
      setExpired(expiredYears);
    } catch (e) {
      setOverview(null);
      setHolds([]);
      setExpired([]);
      setError(message(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const years = overview?.years ?? [];
  const activeHolds = holds.filter((hold) => !hold.releasedAt);
  const suggestedYear = years.some((row) => row.fiscalYear === year)
    ? year
    : (years[0]?.fiscalYear ?? year);

  async function saveHold() {
    if (holdYear === null) return;
    if (holdReason === 'other' && holdText.trim() === '') {
      setHoldError('Ein sonstiger Grund ist zu beschreiben — sonst lässt sich später nicht beurteilen, ob er noch gilt.');
      return;
    }
    setBusy(true);
    setHoldError('');
    try {
      await Api.setRetentionHold(holdYear, holdReason, holdText.trim());
      setHoldYear(null);
      setHoldText('');
      await load();
    } catch (e) {
      setHoldError(message(e));
    } finally {
      setBusy(false);
    }
  }

  async function release() {
    if (!releasing) return;
    if (releaseText.trim() === '') {
      setReleaseError('Zur Aufhebung gehört ein Grund; er bleibt im Protokoll stehen.');
      return;
    }
    setBusy(true);
    setReleaseError('');
    try {
      await Api.releaseRetentionHold(releasing.id, releaseText.trim());
      setReleasing(null);
      setReleaseText('');
      await load();
    } catch (e) {
      setReleaseError(message(e));
    } finally {
      setBusy(false);
    }
  }

  async function archiveAndDelete() {
    if (!deleting) return;
    setBusy(true);
    setDeleteError('');
    try {
      const result = await Api.archiveAndDeleteFiscalYear(deleting.fiscalYear, confirmation.trim());
      const deletedYear = deleting.fiscalYear;
      setDeleting(null);
      setConfirmation('');
      await load();
      setDeleteReport(result.message);
      // Der Toast sagt nur, dass es geschehen ist; was geschehen ist, steht in
      // der Hinweisfläche darunter.
      toast.success(`Geschäftsjahr ${deletedYear} archiviert und gelöscht.`);
    } catch (e) {
      // Der Dialog bleibt mit seiner Eingabe stehen: die Ablehnung des
      // Backends — noch laufende Frist, gescheiterter Archivexport — nennt
      // einen Grund, der hier zu lesen und nicht wegzuklicken ist.
      setDeleteError(message(e));
    } finally {
      setBusy(false);
    }
  }

  if (loading) return <SkeletonRows rows={8} />;

  return (
    <>
      {error && <Notice tone="negative" text={error} className="mb-6" />}

      <StatRow>
        <Stat label="Geschäftsjahre" value={String(years.length)} context="mit Daten" />
        {/* Die abgelaufenen Objekte stehen als Jahreszahlen dabei und nicht
            nur als Anzahl: „3 abgelaufen" verlangt eine zweite Frage, „2013,
            2014, 2015" beantwortet sie gleich mit. */}
        <Stat
          label="Frist abgelaufen"
          value={String(expired.length)}
          context={
            // Bei null abgelaufenen Jahren beschreibt „löschbar nach Archiv"
            // nichts — es gibt nichts zu löschen.
            expired.length > 0 ? expired.map((row) => row.fiscalYear).join(', ') : 'keines'
          }
        />
        <Stat
          label="Fristen ausgesetzt"
          value={String(activeHolds.length)}
          context="§ 147 Abs. 3 Satz 5 AO"
        />
        <Stat
          label="Stichtag"
          value={formatDate(overview?.today ?? '')}
          context="Grundlage der Fristen"
        />
      </StatRow>

      {/* Der Umstellungszeitpunkt gehört auf die Fristenseite (Entscheidung 13):
          die Fünfjahresfrist des § 147 Abs. 6 Satz 6 AO ist eine
          Aufbewahrungsfrist und keine Frage der Versionshistorie. Wer hier
          liest, wann gelöscht werden darf, muss sie sehen. */}
      {overview?.systemChangeNote && (
        <Notice className="mt-6" text={overview.systemChangeNote} />
      )}

      {/* Das Löschprotokoll: Zählungen und Archivpfad bleiben stehen, bis sie
          zur Kenntnis genommen sind. */}
      {deleteReport && (
        <Notice
          className="mt-6"
          text={deleteReport}
          action={
            <Button variant="quiet" size="sm" onClick={() => setDeleteReport(null)}>
              Verstanden
            </Button>
          }
        />
      )}

      <Section
        title="Fristen je Geschäftsjahr"
        context="Es gilt die längste Frist des Jahres, weil das Löschen das ganze Jahr trifft"
        action={
          <div className="flex items-center gap-2">
            <HelpPopover label="Erklärung zu den Aufbewahrungsfristen">
              Die Frist beginnt mit dem Schluss des Kalenderjahres, in dem die letzte Eintragung
              gemacht oder der Beleg entstanden ist (§ 257 Abs. 5 HGB, § 147 Abs. 4 AO). Handelsbücher
              und Abschlüsse sind zehn Jahre aufzubewahren, Buchungsbelege und Rechnungen seit 2025
              acht, Handelsbriefe sechs. Gelöscht werden darf ab dem Tag nach dem Fristende.
            </HelpPopover>
            <Button
              variant="secondary"
              disabled={lock.locked || years.length === 0}
              // Ein gesperrter Knopf ohne Erklärung versteckt seinen Grund
              // (§10.4): der Prüfermodus ist der eine, ein Mandant ohne
              // Geschäftsjahr mit Daten der andere.
              title={
                lock.hint ??
                (years.length === 0
                  ? 'Es gibt noch kein Geschäftsjahr mit Daten, dessen Frist ausgesetzt werden könnte.'
                  : undefined)
              }
              onClick={() => {
                setHoldYear(suggestedYear);
                setHoldReason('audit');
                setHoldText('');
                setHoldError('');
              }}
              icon={<Pause className="w-4 h-4" strokeWidth={1.5} />}
            >
              Frist aussetzen
            </Button>
          </div>
        }
      >
        {years.length === 0 ? (
          <EmptyState
            title="Noch kein Geschäftsjahr mit Daten"
            description="Die Fristen entstehen mit der ersten Buchung."
          />
        ) : (
          <Table density="kompakt">
            <Thead sticky>
              <Tr>
                <Th className="w-24" numeric>
                  Jahr
                </Th>
                <Th numeric className="w-28">
                  Buchungen
                </Th>
                <Th numeric className="w-24">
                  Belege
                </Th>
                <Th className="w-72">Fristen</Th>
                <Th className="w-36">Löschbar ab</Th>
                <Th className="w-40">Stand</Th>
                <Th className="w-44" />
              </Tr>
            </Thead>
            <Tbody>
              {years.map((row) => (
                <Tr key={row.fiscalYear}>
                  <Td numeric className="num">
                    {row.fiscalYear}
                  </Td>
                  <Td numeric className="num">
                    {row.counts.journalEntries}
                  </Td>
                  <Td numeric className="num">
                    {row.counts.receipts}
                  </Td>
                  <Td className="whitespace-normal text-ink-muted">
                    {row.classes.map((entry) => (
                      <span key={entry.class} className="block text-caption">
                        {RETENTION_CLASS_LABELS[entry.class]} · {entry.years} Jahre · bis{' '}
                        {formatDate(entry.retentionEnd)}
                      </span>
                    ))}
                  </Td>
                  <Td className="num text-ink-muted">{formatDate(row.earliestDeletion)}</Td>
                  <Td>
                    {row.hold ? (
                      <Verdict text="Frist ausgesetzt" tone="attention" />
                    ) : row.deletable ? (
                      <Verdict text="Frist abgelaufen" />
                    ) : (
                      <Verdict text="Aufzubewahren" />
                    )}
                  </Td>
                  <Td>
                    <Button
                      variant="quiet"
                      size="sm"
                      disabled={lock.locked || !row.deletable}
                      title={
                        lock.hint ??
                        (row.deletable
                          ? undefined
                          : row.hold
                            ? `Die Frist ist seit dem ${formatDateOfTimestamp(row.hold.setAt)} ausgesetzt (${holdReasonLabel(row.hold.reason)}).`
                            : `Löschbar ab ${formatDate(row.earliestDeletion)}.`)
                      }
                      onClick={() => {
                        setDeleting(row);
                        setConfirmation('');
                        setDeleteError('');
                      }}
                      icon={<Trash2 className="w-3.5 h-3.5" strokeWidth={1.5} />}
                    >
                      Archivieren & löschen
                    </Button>
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>

      <Section title="Ausgesetzte Fristen" context="Gesetzte und aufgehobene Aussetzungen">
        {holds.length === 0 ? (
          <EmptyState
            title="Keine Frist ausgesetzt"
            description="Eine Aussetzung hält die Frist an, solange ein Verfahren läuft."
          />
        ) : (
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th numeric className="w-20">
                  Jahr
                </Th>
                <Th className="w-44">Grund</Th>
                <Th>Erläuterung</Th>
                <Th className="w-52">Gesetzt</Th>
                <Th className="w-52">Aufgehoben</Th>
                <Th className="w-28" />
              </Tr>
            </Thead>
            <Tbody>
              {holds.map((hold) => (
                // Eine aufgehobene Aussetzung bleibt stehen: dass eine Frist
                // einmal ausgesetzt war, gehört zur Geschichte der Daten.
                <Tr key={hold.id}>
                  <Td numeric className="num">
                    {hold.fiscalYear}
                  </Td>
                  <Td className="text-ink-muted">{holdReasonLabel(hold.reason)}</Td>
                  <Td className="whitespace-normal">
                    {hold.description || '—'}
                    {hold.affectedNote && (
                      <span className="block text-caption text-ink-subtle">
                        {hold.affectedNote}
                      </span>
                    )}
                  </Td>
                  <Td className="text-ink-subtle num">
                    {formatDateTime(hold.setAt)}
                    <span className="block text-caption">{hold.setBy}</span>
                  </Td>
                  <Td className="text-ink-subtle num">
                    {hold.releasedAt ? (
                      <>
                        {formatDateTime(hold.releasedAt)}
                        <span className="block text-caption">{hold.releasedBy}</span>
                      </>
                    ) : (
                      '—'
                    )}
                  </Td>
                  <Td>
                    {!hold.releasedAt && (
                      <Button
                        variant="quiet"
                        size="sm"
                        disabled={lock.locked}
                        title={lock.hint}
                        onClick={() => {
                          setReleasing(hold);
                          setReleaseText('');
                          setReleaseError('');
                        }}
                        icon={<Play className="w-3.5 h-3.5" strokeWidth={1.5} />}
                      >
                        Aufheben
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
        title="Löschkonzept"
        context="Datenkategorie, Frist und Rechtsgrundlage"
        action={
          <HelpPopover label="Erklärung zum Löschkonzept">
            Das Konzept steht im Programmcode und nicht in einem Dokument daneben: es ist zugleich
            der Abschnitt der Verfahrensdokumentation und die Regel, nach der die Löschfunktion
            entscheidet. Zwei Fassungen desselben Konzepts laufen auseinander.
          </HelpPopover>
        }
      >
        {(overview?.concept ?? []).length === 0 ? (
          <EmptyState title="Kein Löschkonzept hinterlegt" />
        ) : (
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th className="w-80">Datenkategorie</Th>
                <Th className="w-56">Klasse</Th>
                <Th numeric className="w-20">
                  Jahre
                </Th>
                <Th>Rechtsgrundlage</Th>
              </Tr>
            </Thead>
            <Tbody>
              {(overview?.concept ?? []).map((row) => (
                <Tr key={row.category}>
                  <Td className="whitespace-normal">{row.category}</Td>
                  <Td className="text-ink-muted">{RETENTION_CLASS_LABELS[row.class]}</Td>
                  <Td numeric className="num">
                    {row.years}
                  </Td>
                  <Td className="whitespace-normal text-ink-subtle">{row.legalBasis}</Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>

      {/* Frist aussetzen */}
      <Dialog
        open={holdYear !== null}
        onOpenChange={(open) => !open && setHoldYear(null)}
        title="Aufbewahrungsfrist aussetzen"
        width="max-w-lg"
        footer={
          <>
            <Button variant="secondary" onClick={() => setHoldYear(null)}>
              Abbrechen
            </Button>
            <Button variant="primary" loading={busy} onClick={() => void saveHold()}>
              Frist aussetzen
            </Button>
          </>
        }
      >
        {holdError && <Notice tone="negative" text={holdError} className="mb-4" />}

        <FieldRow>
          <Field label="Geschäftsjahr">
            <Select<number>
              items={years.map((row) => ({ value: row.fiscalYear, label: String(row.fiscalYear) }))}
              value={holdYear ?? suggestedYear}
              onValueChange={(value) => setHoldYear(value)}
              aria-label="Geschäftsjahr"
            />
          </Field>
          <Field label="Grund" help="Die Frist läuft nicht ab, solange der Grund besteht.">
            <Select<RetentionHoldReason>
              items={HOLD_REASONS}
              value={holdReason}
              onValueChange={(value) => setHoldReason(value)}
              aria-label="Grund"
            />
          </Field>
        </FieldRow>
        <Field
          label="Erläuterung"
          className="mt-4"
          optional={holdReason !== 'other'}
          hint={holdReason === 'other' ? 'Bei sonstigem Grund Pflicht' : undefined}
        >
          <Textarea
            rows={3}
            value={holdText}
            onChange={(e) => setHoldText(e.target.value)}
          />
        </Field>
      </Dialog>

      {/* Aussetzung aufheben */}
      <Dialog
        open={releasing !== null}
        onOpenChange={(open) => !open && setReleasing(null)}
        title="Aussetzung aufheben"
        width="max-w-lg"
        footer={
          <>
            <Button variant="secondary" onClick={() => setReleasing(null)}>
              Abbrechen
            </Button>
            <Button variant="primary" loading={busy} onClick={() => void release()}>
              Aufheben
            </Button>
          </>
        }
      >
        {releaseError && <Notice tone="negative" text={releaseError} className="mb-4" />}

        <Field label="Grund der Aufhebung" hint="Bleibt im Protokoll stehen">
          <Textarea
            rows={3}
            value={releaseText}
            onChange={(e) => setReleaseText(e.target.value)}
          />
        </Field>
      </Dialog>

      {/* Archivieren und löschen: erst die ausgeschriebene Jahreszahl, dann
          die Rückfrage. Der Vorgang lässt sich nicht rückgängig machen.

          Während die Rückfrage steht, weicht dieser Dialog: zwei Overlays
          übereinander verdecken einander, und die Rückfrage ist ein
          AlertDialog, der sich bewusst nicht durch einen Klick daneben
          schließen lässt (§8.2). Wird sie abgebrochen, steht dieser Dialog
          samt Eingabe wieder da, weil `deleting` gesetzt bleibt. */}
      <Dialog
        open={deleting !== null && !confirmDelete}
        onOpenChange={(open) => !open && setDeleting(null)}
        title={`Geschäftsjahr ${deleting?.fiscalYear ?? ''} archivieren und löschen`}
        width="max-w-xl"
        footer={
          <>
            <Button variant="secondary" onClick={() => setDeleting(null)}>
              Abbrechen
            </Button>
            <Button
              variant="danger"
              loading={busy}
              disabled={confirmation.trim() !== String(deleting?.fiscalYear ?? '')}
              title={
                confirmation.trim() === String(deleting?.fiscalYear ?? '')
                  ? undefined
                  : 'Die Jahreszahl als Bestätigung eingeben.'
              }
              onClick={() => setConfirmDelete(true)}
            >
              Archivieren und löschen
            </Button>
          </>
        }
      >
        {/* Die Ablehnung des Backends steht über der Eingabe, die sie
            betrifft, und über der Ankündigung des Vorgangs. */}
        {deleteError && <Notice tone="negative" text={deleteError} className="mb-4" />}

        <Notice
          tone="negative"
          text={`${deleting?.counts.journalEntries ?? 0} Buchungen, ${deleting?.counts.receipts ?? 0} Belege und ${deleting?.counts.invoices ?? 0} Rechnungen dieses Jahres werden nach dem Archivexport gelöscht.`}
        />
        <Field
          label="Bestätigung"
          className="mt-4"
          hint="Die Jahreszahl eingeben"
          help="Ein Klick allein ist bei einem unumkehrbaren Vorgang keine Bestätigung."
        >
          <Input
            value={confirmation}
            onChange={(e) => setConfirmation(e.target.value)}
            placeholder={String(deleting?.fiscalYear ?? '')}
          />
        </Field>
      </Dialog>

      {/* Die zweite Rückfrage vor einem unumkehrbaren Schritt ist ein
          ConfirmDialog und kein Dialog: er lässt sich nur über eine der beiden
          Antworten schließen, nicht durch einen Klick daneben (§8.2). Der Text
          nennt die Folge, nicht die Aktion. */}
      <ConfirmDialog
        open={confirmDelete}
        onOpenChange={setConfirmDelete}
        title="Löschung wirklich ausführen?"
        description={`Buchfink erstellt zuerst den Archivexport des Geschäftsjahres ${deleting?.fiscalYear ?? ''} und löscht danach seine Daten. Der Vorgang lässt sich nicht rückgängig machen; das Löschprotokoll bleibt.`}
        confirmLabel="Endgültig löschen"
        destructive
        onConfirm={() => void archiveAndDelete()}
      />
    </>
  );
};

// -------------------------------------------------------------------------
// Verfahrensdokumentation
// -------------------------------------------------------------------------

/** Die Freitextfelder der Organisationsanweisung in fester Reihenfolge. */
const ORG_FIELDS: { key: keyof OrganisationTexts; label: string; hint: string }[] = [
  { key: 'responsibilities', label: 'Verantwortung', hint: 'Wer führt die Buchführung' },
  { key: 'receiptFlow', label: 'Belegfluss', hint: 'Eingang, Erfassung, Prüfung' },
  { key: 'scanning', label: 'Einscannen', hint: 'Gerät, Auflösung, Ablage' },
  { key: 'approval', label: 'Freigabe', hint: 'Wer bucht, wer schreibt fest' },
  { key: 'substitution', label: 'Vertretung', hint: 'Im Verhinderungsfall' },
  { key: 'backup', label: 'Sicherung', hint: 'Wer überwacht, wo liegen Kopien' },
  { key: 'notes', label: 'Weiteres', hint: 'Alles Übrige' },
];

const ProcDocPanel: React.FC = () => {
  const lock = useWriteLock();
  const [documents, setDocuments] = useState<ProcedureDocumentation[]>([]);
  const [texts, setTexts] = useState<OrganisationTexts | null>(null);
  const [hints, setHints] = useState<ComplianceHints | null>(null);
  const [preview, setPreview] = useState<ProcDocResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [docs, orgTexts, complianceHints] = await Promise.all([
        Api.getProcedureDocumentations(),
        Api.getOrganisationTexts(),
        Api.getComplianceHints(),
      ]);
      setDocuments(docs);
      setTexts(orgTexts);
      setHints(complianceHints);
    } catch (e) {
      setError(message(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const latest = documents[0];
  const taxCaseHints = hints?.taxCaseHints ?? [];

  async function generate() {
    setBusy(true);
    setError('');
    try {
      const result = await Api.generateProcedureDocumentation();
      setPreview(result);
      await load();
      toast.success(result.message || `Fassung ${result.document.version} erzeugt.`);
    } catch (e) {
      setError(message(e));
    } finally {
      setBusy(false);
    }
  }

  async function saveTexts() {
    if (!texts) return;
    setSaving(true);
    setError('');
    try {
      await Api.saveOrganisationTexts(texts);
      toast.success('Organisationsanweisung gespeichert.');
    } catch (e) {
      setError(message(e));
    } finally {
      setSaving(false);
    }
  }

  if (loading) return <SkeletonRows rows={8} />;

  return (
    <>
      {error && <Notice tone="negative" text={error} className="mb-6" />}
      {hints?.cloudWarning && <Notice text={hints.cloudWarning} className="mb-6" />}
      {hints?.legalFormNote && <Notice text={hints.legalFormNote} className="mb-6" />}

      <StatRow>
        <Stat
          label="Fassungen"
          value={String(documents.length)}
          context="im Belegspeicher abgelegt"
        />
        <Stat
          label="Letzte Fassung"
          value={latest?.version ?? '—'}
          context={latest ? formatDateOfTimestamp(latest.createdAt) : 'noch keine erzeugt'}
        />
        <Stat
          label="Programmfassung"
          value={latest?.appVersion ?? '—'}
          context={latest ? `Regelwerk ${latest.ruleVersion}` : 'aus der letzten Fassung'}
        />
        <Stat label="Speicherort" value="Lokaler Datenordner" context={hints?.dataDir ?? '—'} />
      </StatRow>

      <Section
        title="Fassungen"
        context="Zu jedem Geschäftsjahr gilt die Fassung, die damals galt (GoBD Rz. 151)"
        action={
          <div className="flex items-center gap-2">
            <HelpPopover label="Erklärung zur Verfahrensdokumentation">
              Buchfink erzeugt die Verfahrensdokumentation aus dem System: Unternehmensdaten,
              Nummernkreise, Regelwerks- und Programmfassung kommen aus der Datenbank, die
              Beschreibung des Verfahrens aus dem Programm. Jede Fassung wird abgelegt und bleibt
              über die Aufbewahrungsfrist nachlesbar.
            </HelpPopover>
            {/* Genau eine Primäraktion (§10.4): solange keine Fassung
                besteht, trägt sie der Leerzustand. */}
            {documents.length > 0 && (
              <Button
                variant="primary"
                loading={busy}
                disabled={lock.locked}
                title={lock.hint}
                onClick={() => void generate()}
                icon={<RefreshCw className="w-4 h-4" strokeWidth={1.5} />}
              >
                Neue Fassung erzeugen
              </Button>
            )}
          </div>
        }
      >
        {preview?.pdfNote && <Notice text={preview.pdfNote} className="mb-5" />}

        {documents.length === 0 ? (
          <EmptyState
            icon={<FileText className="w-6 h-6" strokeWidth={1.5} />}
            title="Noch keine Fassung erzeugt"
            description="Die erste Fassung entsteht aus den heutigen Stammdaten und Einstellungen."
            action={
              <Button
                variant="primary"
                loading={busy}
                disabled={lock.locked}
                title={lock.hint}
                onClick={() => void generate()}
              >
                Fassung erzeugen
              </Button>
            }
          />
        ) : (
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th className="w-36">Fassung</Th>
                <Th className="w-52">Erzeugt</Th>
                <Th numeric className="w-20">
                  Jahr
                </Th>
                <Th className="w-24">Programm</Th>
                <Th className="w-24">Regelwerk</Th>
                <Th numeric className="w-24">
                  Größe
                </Th>
                <Th className="w-40">Prüfsumme</Th>
                <Th>Ablage im Datenordner</Th>
              </Tr>
            </Thead>
            <Tbody>
              {documents.map((doc) => (
                <Tr key={doc.id}>
                  <Td className="num">{doc.version}</Td>
                  <Td className="text-ink-subtle num">
                    {formatDateTime(doc.createdAt)}
                    <span className="block text-caption">{doc.actor}</span>
                  </Td>
                  <Td numeric className="num">
                    {doc.fiscalYear}
                  </Td>
                  <Td className="num text-ink-muted">{doc.appVersion}</Td>
                  <Td className="num text-ink-muted">{doc.ruleVersion}</Td>
                  <Td numeric className="num">
                    {formatBytes(doc.size)}
                  </Td>
                  <Td className="code-num text-ink-subtle">{formatShortHash(doc.sha256)}</Td>
                  {/* Der Pfad und nicht nur der Dateiname: ohne ihn ist eine
                      frühere Fassung erzeugt und nicht wiederzufinden — und zu
                      jedem Geschäftsjahr gilt die Fassung, die damals galt
                      (GoBD Rz. 151). Herausgeben lässt sie sich über das
                      Prüferpaket, lesen im Datenordner. */}
                  <Td className="whitespace-normal break-all text-caption text-ink-subtle">
                    {doc.storedPath || doc.fileName}
                    {doc.pdfStoredPath && (
                      <span className="mt-0.5 block">{doc.pdfStoredPath}</span>
                    )}
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>

      {preview?.markdown && (
        <Section title="Erzeugte Fassung" context={preview.document.fileName}>
          <pre className="max-h-[32rem] overflow-y-auto rounded-control border border-line bg-surface p-4 text-caption text-ink-muted whitespace-pre-wrap">
            {preview.markdown}
          </pre>
        </Section>
      )}

      <Section
        title="Organisationsanweisung"
        context="Die Teile, die nur das Unternehmen kennt"
        action={
          <Button
            variant="secondary"
            loading={saving}
            disabled={lock.locked || !texts}
            title={lock.hint}
            onClick={() => void saveTexts()}
          >
            Texte speichern
          </Button>
        }
      >
        {texts && (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
            {ORG_FIELDS.map((entry) => (
              <Field key={entry.key} label={entry.label} hint={entry.hint}>
                <Textarea
                  rows={4}
                  value={texts[entry.key]}
                  disabled={lock.locked}
                  onChange={(e) => setTexts({ ...texts, [entry.key]: e.target.value })}
                />
              </Field>
            ))}
          </div>
        )}
      </Section>

      <Section
        title="Grenzen des Funktionsumfangs"
        context="Steuerfälle, die Buchfink nicht abbildet"
      >
        {taxCaseHints.length === 0 ? (
          <EmptyState title="Keine Einschränkung hinterlegt" />
        ) : (
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th>Hinweis</Th>
              </Tr>
            </Thead>
            <Tbody>
              {taxCaseHints.map((hint) => (
                <Tr key={hint}>
                  <Td className="whitespace-normal text-ink-muted">{hint}</Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>
    </>
  );
};

export default NachweisePage;
