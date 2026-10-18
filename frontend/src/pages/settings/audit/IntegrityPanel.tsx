import React, { useEffect, useState } from 'react';
import { ChevronDown, ChevronRight, Clock, Lock, OctagonAlert, ShieldCheck } from 'lucide-react';
import { Api } from '../../../services/api';
import type {
  CheckRun,
  CheckTimeliness,
  Festschreibung,
  FileCheckResult,
  IntegrityCheckResult,
} from '../../../types';
import { formatDate, formatDateTime } from '../../../utils/formatters';
import {
  Button,
  Help,
  Notice,
  SkeletonRows,
  Stat,
  StatRow,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  cn,
  toast,
} from '../../../components/ui';

function message(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

function days(count: number): string {
  return count === 1 ? '1 Tag' : `${count} Tage`;
}

/** Die Abstände Belegdatum → Erfassung → Festschreibung eines Prüflaufs. */
function timelinessLines(t: CheckTimeliness): string[] {
  return [
    t.measuredEntries === 0
      ? 'Erfassung: nicht messbar, keine Buchung hatte ein Belegdatum'
      : `Erfassung: Median ${days(t.captureDaysMedian)}, höchstens ${days(t.captureDaysMax)}, ` +
        `${t.lateEntries} über der Frist von ${days(t.captureLimitDays)}`,
    t.committedEntries === 0
      ? `Festschreibung: noch keine Buchung festgeschrieben, ${t.uncommittedEntries} offen`
      : `Festschreibung: Median ${days(t.commitDaysMedian)}, höchstens ${days(t.commitDaysMax)}, ` +
        `${t.uncommittedEntries} offen`,
  ];
}

const SubHeading: React.FC<{ title: string; context?: string; action?: React.ReactNode }> = ({
  title,
  context,
  action,
}) => (
  <div className="mt-8 mb-3 flex items-end justify-between gap-4">
    <div>
      <h4 className="text-label text-ink">{title}</h4>
      {context && <p className="text-caption text-ink-subtle mt-0.5">{context}</p>}
    </div>
    {action}
  </div>
);

export interface IntegrityPanelProps {
  integrity: IntegrityCheckResult | null;
  /** Die letzte Prüfung ist fehlgeschlagen. */
  integrityError?: string | null;
  isChecking: boolean;
  onRefresh: () => void;
  /** Die Prüfregel einer Aufgabe: der jüngste Lauf mit diesem Befund geht auf. */
  initialRule?: string;
}

/**
 * Ist an den Daten seit ihrer Erfassung etwas verändert worden? Die Kette der
 * Buchungen und des Protokolls, die Belegdateien, die Festschreibungen und die
 * Prüfläufe vor jeder Festschreibung.
 */
export const IntegrityPanel: React.FC<IntegrityPanelProps> = ({
  integrity,
  integrityError,
  isChecking,
  onRefresh,
  initialRule,
}) => {
  const [commitments, setCommitments] = useState<Festschreibung[] | null>(null);
  const [runs, setRuns] = useState<CheckRun[]>([]);
  const [expandedRun, setExpandedRun] = useState<number | null>(null);
  const [verifyingId, setVerifyingId] = useState<number | null>(null);
  // Der Belegprüflauf öffnet jede Datei einzeln und läuft nur auf Knopfdruck.
  const [fileCheck, setFileCheck] = useState<FileCheckResult | null>(null);
  const [checkingFiles, setCheckingFiles] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    let active = true;
    Promise.all([Api.getFestschreibungen(), Api.getCheckRuns(0)])
      .then(([festschreibungen, checkRuns]) => {
        if (!active) return;
        setCommitments(festschreibungen);
        setRuns(checkRuns);
        if (initialRule) {
          const hit = checkRuns.find((run) =>
            (run.findings ?? []).some((finding) => finding.rule === initialRule),
          );
          if (hit) setExpandedRun(hit.id);
        }
      })
      .catch((e) => {
        if (!active) return;
        setCommitments([]);
        setError(message(e));
      });
    return () => {
      active = false;
    };
  }, [initialRule]);

  async function checkFiles() {
    setCheckingFiles(true);
    setError('');
    try {
      const result = await Api.verifyReceiptFiles();
      setFileCheck(result);
      if (result.isValid) toast.success(`${result.checked} Dateien unverändert.`);
    } catch (e) {
      setError(message(e));
    } finally {
      setCheckingFiles(false);
    }
  }

  async function verifyCommitment(id: number) {
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

  return (
    <div>
      {broken && integrity && <Notice tone="negative" className="mb-5" text={integrity.message} />}
      {error && <Notice tone="negative" className="mb-5" text={error} />}

      {!integrity && integrityError && !isChecking && (
        <Notice
          tone="negative"
          className="mb-5"
          text={`Die Prüfung ist fehlgeschlagen. ${integrityError}`}
        />
      )}

      {integrity ? (
        <StatRow>
          <Stat
            label="Buchungen"
            value={broken ? 'Verletzt' : 'Unverändert'}
            context={`${integrity.checkedEntries} von ${integrity.totalEntries} geprüft am ${formatDateTime(integrity.checkedAt)}`}
            tone={broken ? 'negative' : 'positive'}
          />
          <Stat
            label="Änderungsprotokoll"
            value={integrity.auditChain ? (integrity.auditChain.isValid ? 'Unverändert' : 'Verletzt') : '—'}
            context={
              integrity.auditChain
                ? `${integrity.auditChain.checkedEntries} Einträge nachgerechnet`
                : 'nicht mitgeprüft'
            }
            tone={integrity.auditChain ? (integrity.auditChain.isValid ? 'positive' : 'negative') : 'neutral'}
          />
          <Stat
            label="Belegdateien"
            value={fileCheck ? (fileCheck.isValid ? 'Unverändert' : 'Beschädigt') : '—'}
            context={
              fileCheck
                ? `${fileCheck.intact} von ${fileCheck.checked} unversehrt`
                : 'noch nicht geprüft'
            }
            tone={fileCheck ? (fileCheck.isValid ? 'positive' : 'negative') : 'neutral'}
          />
        </StatRow>
      ) : integrityError && !isChecking ? null : (
        <SkeletonRows rows={2} />
      )}

      <div className="mt-5 flex flex-wrap gap-2">
        <Button
          variant="secondary"
          size="sm"
          loading={isChecking}
          icon={<ShieldCheck className="w-4 h-4" strokeWidth={1.5} />}
          onClick={onRefresh}
        >
          Buchungen prüfen
        </Button>
        <Button
          variant="secondary"
          size="sm"
          loading={checkingFiles}
          title="Öffnet jede abgelegte Datei; bei vielen Belegen dauert das einen Moment."
          icon={<ShieldCheck className="w-4 h-4" strokeWidth={1.5} />}
          onClick={() => void checkFiles()}
        >
          Belegdateien prüfen
        </Button>
      </div>

      {fileCheck && fileCheck.issues.length > 0 && (
        <>
          <SubHeading title="Beschädigte oder fehlende Dateien" />
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th className="w-32">Beleg</Th>
                <Th>Datei</Th>
                <Th className="w-32">Befund</Th>
              </Tr>
            </Thead>
            <Tbody>
              {fileCheck.issues.map((issue) => (
                <Tr key={`${issue.path}-${issue.fileName}`}>
                  <Td code>{issue.receiptNumber || '—'}</Td>
                  <Td className="text-ink-muted">{issue.fileName}</Td>
                  <Td className="text-negative-text">
                    <span className="flex items-center gap-1.5">
                      <OctagonAlert className="w-3.5 h-3.5 shrink-0" strokeWidth={1.5} />
                      {issue.reason === 'missing' ? 'Fehlt' : 'Beschädigt'}
                    </span>
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        </>
      )}

      {integrity && integrity.breaks.length > 0 && (
        <>
          <SubHeading
            title="Abweichende Buchungen"
            context="Erwarteter und tatsächlicher Hash, zum Nachrechnen außerhalb von Buchfink"
          />
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th className="w-20">Jahr</Th>
                <Th className="w-36">Buchung</Th>
                <Th className="w-40">Befund</Th>
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
                    {issue.reason === 'linkage'
                      ? 'Kette unterbrochen'
                      : issue.reason === 'receipt'
                        ? 'Beleg verändert'
                        : 'Buchung verändert'}
                  </Td>
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
        </>
      )}

      <SubHeading
        title="Festgeschriebene Zeiträume"
        context="Mit Zeitstempel eines unabhängigen Dienstes"
      />
      {commitments === null ? (
        <SkeletonRows rows={3} />
      ) : commitments.length === 0 ? (
        <p className="text-body text-ink-muted">Noch kein Zeitraum festgeschrieben.</p>
      ) : (
        <Table density="kompakt">
          <Thead>
            <Tr>
              <Th>Zeitraum</Th>
              <Th className="w-28">Stichtag</Th>
              <Th numeric className="w-24">Buchungen</Th>
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
                      {confirmed ? `${formatDateTime(fs.tsaGenTime!)} · ${fs.tsaName}` : 'Ausstehend, wird nachgeholt'}
                    </span>
                    {fs.timeDriftNote && <span className="mt-1 flex text-attention-text">{fs.timeDriftNote}</span>}
                  </Td>
                  <Td className="pl-0">
                    <Button
                      variant="quiet"
                      size="sm"
                      loading={verifyingId === fs.id}
                      onClick={() => void verifyCommitment(fs.id)}
                      className="opacity-0 transition-opacity duration-120 ease-quiet group-hover:opacity-100 focus-visible:opacity-100"
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

      <SubHeading title="Prüfläufe" context="Der Bericht vor jeder Festschreibung, mit übergangenen Befunden" />
      {runs.length === 0 ? (
        <p className="text-body text-ink-muted">Ein Lauf entsteht mit der ersten Festschreibung.</p>
      ) : (
        <Table density="kompakt">
          <Thead>
            <Tr>
              <Th className="w-48">Zeitpunkt</Th>
              <Th className="w-28">Stichtag</Th>
              <Th className="w-44">Befunde</Th>
              <Th>Übergangen mit Begründung</Th>
            </Tr>
          </Thead>
          <Tbody>
            {runs.map((run) => {
              const findings = run.findings ?? [];
              const blocking = findings.filter((f) => f.severity === 'blocking');
              const details = run.timeliness ? timelinessLines(run.timeliness) : [];
              const open = expandedRun === run.id;
              const empty = findings.length === 0 && details.length === 0;
              return (
                <React.Fragment key={run.id}>
                  <Tr>
                    <Td className="text-ink-subtle num">
                      <button
                        type="button"
                        onClick={() => setExpandedRun(open ? null : run.id)}
                        aria-expanded={open}
                        disabled={empty}
                        title={empty ? 'Zu diesem Lauf sind weder Befunde noch Kennzahlen gespeichert.' : undefined}
                        className="inline-flex items-center gap-1 text-ink-muted hover:text-ink transition-colors duration-120 ease-quiet disabled:text-ink-faint"
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
                    <Td className={cn('text-caption', blocking.length > 0 ? 'text-negative-text' : 'text-ink-muted')}>
                      {blocking.length} blockierend · {findings.length - blocking.length}{' '}
                      {findings.length - blocking.length === 1 ? 'Hinweis' : 'Hinweise'}
                    </Td>
                    <Td className="whitespace-normal text-ink-muted">{run.overrideReason || '—'}</Td>
                  </Tr>
                  {open && details.length > 0 && (
                    <Tr>
                      <Td />
                      <Td className="text-ink-subtle">Abstände</Td>
                      <Td colSpan={2} className="whitespace-normal text-ink-muted">
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
                        className={initialRule && finding.rule === initialRule ? 'bg-attention-soft' : undefined}
                      >
                        <Td />
                        <Td className={finding.severity === 'blocking' ? 'text-negative-text' : 'text-attention-text'}>
                          {finding.severity === 'blocking' ? 'Blockierend' : 'Hinweis'}
                        </Td>
                        <Td colSpan={2} className="whitespace-normal text-ink-muted">
                          {finding.message}
                          {finding.reference && (
                            <Help
                              summary="Auf dieser Vorschrift beruht der Prüfhinweis."
                              label="Hintergrund zum Prüfhinweis"
                            >
                              {finding.reference}
                            </Help>
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
    </div>
  );
};
