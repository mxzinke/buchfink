import React, { useCallback, useEffect, useState } from 'react';
import { ChevronDown, ChevronRight, ShieldCheck } from 'lucide-react';
import { Api } from '../../../services/api';
import type { AuditChainResult, AuditFilter, AuditLogEntry } from '../../../types';
import { formatDateTime, formatShortHash } from '../../../utils/formatters';
import { channelNote } from '../../../utils/actor';
import {
  Button,
  EmptyState,
  Field,
  Input,
  Notice,
  Select,
  SkeletonRows,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  toast,
} from '../../../components/ui';

function message(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

/** Die Vorgangsarten des Protokolls in Klartext. */
const ACTION_LABELS: Record<string, string> = {
  CREATE: 'Angelegt',
  UPDATE: 'Geändert',
  STORNO: 'Storniert',
  IMPORT: 'Übernommen',
  INTEGRITY_CHECK: 'Geprüft',
  EXPORT: 'Ausgegeben',
  COMMIT: 'Festgeschrieben',
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
 * Der Eintrag enthält sie als JSON-Text, weil die Struktur je Objektart eine
 * andere ist. Ein defekter Text darf die Zeile nicht mitreißen — dann bleibt die
 * Aufklappung eben leer.
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

/**
 * Das Änderungsprotokoll mit Vorher und Nachher je Eintrag. Jeder Eintrag ist
 * mit seinem Vorgänger verkettet; die Prüfung rechnet die Kette nach.
 */
export const ProtocolPanel: React.FC = () => {
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

      <div className="mb-5 flex flex-wrap items-center justify-between gap-3">
        <p className="text-caption text-ink-subtle">
          {chain
            ? `${entries.length} Einträge · Kette am ${formatDateTime(chain.checkedAt)} nachgerechnet`
            : `${entries.length} Einträge · neueste zuerst`}
        </p>
        <Button
          variant="secondary"
          size="sm"
          loading={verifying}
          onClick={() => void reverify()}
          icon={<ShieldCheck className="w-4 h-4" strokeWidth={1.5} />}
        >
          Kette prüfen
        </Button>
      </div>

      {breaks.length > 0 && (
        <div className="mb-6">
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
                  <Td className="text-ink-muted">{entry.reason === 'linkage' ? 'Verkettung' : 'Inhalt'}</Td>
                  <Td className="whitespace-normal">{entry.message}</Td>
                  <Td className="code-num text-ink-subtle">{formatShortHash(entry.expectedHash)}</Td>
                  <Td className="code-num text-ink-subtle">{formatShortHash(entry.actualHash)}</Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        </div>
      )}

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
                    <Td className="text-ink-subtle">
                      {entry.actor || '—'}
                      {channelNote(entry.channel) && (
                        <span className="block text-caption">{channelNote(entry.channel)}</span>
                      )}
                    </Td>
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

