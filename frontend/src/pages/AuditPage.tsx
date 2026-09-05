import React, { useEffect, useState } from 'react';
import { AlertCircle, ChevronDown, ChevronRight, Clock, Lock, ShieldCheck } from 'lucide-react';
import { AuditLogEntry, CheckRun, Festschreibung, IntegrityCheckResult } from '../types';
import { Api } from '../services/api';
import { formatDate } from '../utils/formatters';
import {
  Button,
  EmptyState,
  HelpPopover,
  PageHeader,
  Section,
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
} from '../components/ui';

function formatMoment(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;
  return new Intl.DateTimeFormat('de-DE', { dateStyle: 'short', timeStyle: 'short' }).format(date);
}

export const AuditPage: React.FC = () => {
  const [logs, setLogs] = useState<AuditLogEntry[]>([]);
  const [integrity, setIntegrity] = useState<IntegrityCheckResult | null>(null);
  const [commitments, setCommitments] = useState<Festschreibung[]>([]);
  // Die Prüfläufe gehören hierher und nicht in die Fristenansicht: dort werden
  // sie ausgelöst, hier bleiben sie nachlesbar — samt der Begründung, mit der
  // ein blockierender Befund übergangen wurde (GoBD Rz. 34 ff.).
  const [checkRuns, setCheckRuns] = useState<CheckRun[]>([]);
  const [expandedRun, setExpandedRun] = useState<number | null>(null);
  const [loading, setLoading] = useState(true);
  const [isVerifying, setIsVerifying] = useState(false);
  const [verifyingId, setVerifyingId] = useState<number | null>(null);

  useEffect(() => {
    void loadData();
  }, []);

  async function loadData() {
    setLoading(true);
    try {
      const [logList, result, festschreibungen, runs] = await Promise.all([
        Api.getAuditLogs(),
        Api.verifyIntegrity(),
        Api.getFestschreibungen(),
        // Jahr 0 heißt: das aktive Geschäftsjahr.
        Api.getCheckRuns(0),
      ]);
      setLogs(logList);
      setIntegrity(result);
      setCommitments(festschreibungen);
      setCheckRuns(runs);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  async function handleReverify() {
    setIsVerifying(true);
    try {
      const result = await Api.verifyIntegrity();
      setIntegrity(result);
      if (result.isValid) toast.success(`${result.checkedEntries} Buchungen unverändert.`);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setIsVerifying(false);
    }
  }

  async function handleVerifyCommitment(id: number) {
    setVerifyingId(id);
    try {
      const result = await Api.verifyFestschreibung(id);
      if (result?.isValid) toast.success('Festschreibung gültig.');
      else toast.error(result?.message ?? 'Festschreibung ungültig.');
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setVerifyingId(null);
    }
  }

  const broken = integrity !== null && !integrity.isValid;

  return (
    <div className="max-w-[1200px] mx-auto px-8 py-8">
      <PageHeader
        title="Sicherheit & Protokoll"
        context="Änderungsprotokoll und Schutz vor nachträglicher Veränderung"
        action={
          <Button
            variant="secondary"
            loading={isVerifying}
            onClick={handleReverify}
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

      {loading ? (
        <div className="mt-8">
          <SkeletonRows rows={6} />
        </div>
      ) : (
        <>
          {integrity && (
            <div className="mt-6">
              <StatRow>
                <Stat
                  label="Zustand der Kette"
                  value={broken ? 'Verletzt' : 'Unverändert'}
                  context={`Geprüft am ${formatMoment(integrity.checkedAt)}`}
                  tone={broken ? 'negative' : 'positive'}
                />
                <Stat
                  label="Geprüfte Buchungen"
                  value={`${integrity.checkedEntries} von ${integrity.totalEntries}`}
                  context="Hash-Kette vollständig durchlaufen"
                />
                <Stat
                  label="Festgeschriebene Zeiträume"
                  value={String(commitments.length)}
                  context="Abschluss über Steuerfristen"
                />
              </StatRow>
            </div>
          )}

          <Section
            title="Festgeschriebene Zeiträume"
            context="Abgeschlossen und von einem Zeitstempeldienst beglaubigt"
            action={
              <HelpPopover label="Erklärung zur Festschreibung">
                Beim Festschreiben werden die Buchungen eines Zeitraums verbindlich abgeschlossen,
                spätere Änderungen sind nur noch per Storno möglich. Zusätzlich beglaubigt ein
                unabhängiger Zeitstempeldienst nach RFC 3161 den Stand. Festgeschrieben wird unter
                Steuerfristen.
              </HelpPopover>
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
                              ? `${formatMoment(fs.tsaGenTime!)} · ${fs.tsaName}`
                              : 'Ausstehend, wird nachgeholt'}
                          </span>
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

          <Section
            title="Prüfläufe"
            context="Der Bericht, der vor jeder Festschreibung läuft"
            action={
              <HelpPopover label="Erklärung zu den Prüfläufen">
                Der Prüflauf sagt vor der Festschreibung, was danach nicht mehr zu ändern wäre:
                Buchungen ohne Beleg, nicht zugeordnete Bankumsätze, Salden auf den Interimskonten.
                Blockierende Befunde verhindern die Festschreibung; übergangen werden sie nur mit
                einer Begründung, und die steht dann hier.
              </HelpPopover>
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
                    const blocking = run.findings.filter((f) => f.severity === 'blocking');
                    const warnings = run.findings.filter((f) => f.severity === 'warning');
                    const open = expandedRun === run.id;
                    return (
                      <React.Fragment key={run.id}>
                        <Tr>
                          <Td className="text-ink-subtle num">
                            <button
                              type="button"
                              onClick={() => setExpandedRun(open ? null : run.id)}
                              aria-expanded={open}
                              disabled={run.findings.length === 0}
                              className="inline-flex items-center gap-1 text-ink-muted
                                         hover:text-ink transition-colors duration-120 ease-quiet
                                         disabled:text-ink-faint"
                            >
                              {open ? (
                                <ChevronDown className="w-3.5 h-3.5" strokeWidth={1.5} />
                              ) : (
                                <ChevronRight className="w-3.5 h-3.5" strokeWidth={1.5} />
                              )}
                              {formatMoment(run.createdAt)}
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
                        {open &&
                          run.findings.map((finding) => (
                            <Tr key={finding.id}>
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
                                  <span className="text-caption text-ink-subtle"> · {finding.reference}</span>
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

          <Section title="Änderungsprotokoll" context="Chronologischer Verlauf aller Vorgänge">
            {logs.length === 0 ? (
              <EmptyState title="Noch keine Einträge" />
            ) : (
              <Table density="kompakt">
                <Thead sticky>
                  <Tr>
                    <Th className="w-44">Zeitpunkt</Th>
                    <Th className="w-32">Aktion</Th>
                    <Th className="w-36">Bereich</Th>
                    <Th>Beschreibung</Th>
                  </Tr>
                </Thead>
                <Tbody>
                  {logs.map((entry) => (
                    <Tr key={entry.id}>
                      <Td className="text-ink-subtle num">{formatMoment(entry.timestamp)}</Td>
                      <Td>
                        <span className="inline-flex items-center h-5 px-2 rounded-control border border-line-strong text-caption text-ink-muted">
                          {entry.action}
                        </span>
                      </Td>
                      <Td className="text-ink-muted">{entry.entityType}</Td>
                      <Td className="whitespace-normal">{entry.details}</Td>
                    </Tr>
                  ))}
                </Tbody>
              </Table>
            )}
          </Section>
        </>
      )}
    </div>
  );
};
