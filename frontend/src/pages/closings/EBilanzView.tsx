import React, { useCallback, useEffect, useState } from 'react';
import { FileDown } from 'lucide-react';
import { Api } from '../../services/api';
import { useWriteLock } from '../../components/WriteLock';
import { useLatestRequest, useLiveReload } from '../../live';
import { formatCents } from '../../utils/formatters';
import type { MappingReport, Obligation, Reconciliation, TaxElectionRegister } from '../../types';
import {
  Button,
  Disclosure,
  Help,
  Notice,
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
} from '../../components/ui';
import { ObligationDecisionDialog, ObligationState, isManual, message } from './shared';

/**
 * Die E-Bilanz: Prüfergebnis der Zuordnung, Export als XBRL oder als Summen-
 * und Saldenliste und die Bestätigung der Übermittlung. Mein ELSTER hat für
 * die E-Bilanz kein Formular; übermittelt wird die XBRL-Datei über einen
 * E-Bilanz-Dienst wie ebilanzplus.de oder ebilanzonline.de. Die Summen- und
 * Saldenliste ist der Ausweichweg, wenn der Dienst die XBRL-Datei nicht
 * annimmt: eBilanz+ liest sie als Summen- und Saldenliste ein.
 *
 * Im Geschäftsjahr einer Gründung in Buchfink steht darüber die E-Bilanz der
 * Eröffnungsbilanz (§ 5b Abs. 1 Satz 4 EStG); `opening` ist ihre Pflicht.
 */
export const EBilanzView: React.FC<{
  year: number;
  obligation?: Obligation;
  opening?: Obligation;
  onObligationsChanged: () => Promise<void>;
}> = ({ year, obligation, opening, onObligationsChanged }) => {
  const writeLock = useWriteLock();
  const [report, setReport] = useState<MappingReport | null>(null);
  const [error, setError] = useState('');
  const [saving, setSaving] = useState<'xbrl' | 'susa_csv' | 'opening' | null>(null);
  const [deciding, setDeciding] = useState<Obligation | null>(null);

  const begin = useLatestRequest();
  const load = useCallback(async () => {
    const latest = begin();
    try {
      const next = await Api.getEBilanzMappingReport(year);
      if (!latest()) return;
      setReport(next);
      setError('');
    } catch (e) {
      if (!latest()) return;
      setReport(null);
      setError(message(e));
    }
  }, [year, begin]);

  useEffect(() => {
    void load();
  }, [load]);

  // Ordnet Claude ein Konto zu oder bucht den Abschluss, gibt die Ansicht den
  // Export frei, ohne dass jemand sie verlässt.
  useLiveReload(load, ['buchungen', 'abschluss', 'stammdaten']);

  async function save(format: 'xbrl' | 'susa_csv') {
    setSaving(format);
    try {
      const path = await Api.saveEBilanzAs(year, format);
      if (path) toast.success(`E-Bilanz gespeichert unter ${path}`);
    } catch (e) {
      setError(message(e));
    } finally {
      setSaving(null);
    }
  }

  async function saveOpening() {
    setSaving('opening');
    try {
      const path = await Api.saveOpeningEBilanzAs(year);
      if (path) toast.success(`Eröffnungsbilanz gespeichert unter ${path}`);
    } catch (e) {
      setError(message(e));
    } finally {
      setSaving(null);
    }
  }

  const rows = report?.rows ?? [];
  const transfers = report?.accountingStandard === 'HGBM' ? (report.transfers ?? []) : [];
  const blocking = report?.blocking ?? [];
  const warnings = (report?.findings ?? []).filter((f) => f.severity !== 'blocking');
  const fallbacks = report?.fallbacks ?? [];

  return (
    <Section
      title={`E-Bilanz ${year}`}
      context={
        report
          ? `Taxonomie ${report.taxonomyVersion} · ${report.accountingStandard === 'HGBM' ? 'Handelsbilanz mit Überleitung' : 'Einheitsbilanz'}`
          : undefined
      }
      action={
        <div className="flex flex-wrap items-center justify-end gap-2">
          <Button
            variant="quiet"
            loading={saving === 'susa_csv'}
            disabled={!report?.canExport || saving !== null}
            title={report && !report.canExport ? 'Erst wenn jedes Konto zugeordnet ist, entsteht die Datei.' : undefined}
            onClick={() => void save('susa_csv')}
          >
            Als Saldenliste (CSV)
          </Button>
          <Button
            variant="primary"
            icon={<FileDown className="w-4 h-4" strokeWidth={1.5} />}
            loading={saving === 'xbrl'}
            disabled={!report?.canExport || saving !== null}
            title={report && !report.canExport ? 'Erst wenn jedes Konto zugeordnet ist, entsteht die Datei.' : undefined}
            onClick={() => void save('xbrl')}
          >
            XBRL exportieren
          </Button>
        </div>
      }
    >
      {error && <Notice tone="negative" className="mb-6" text={error} />}
      {opening && (
        <div className="mb-6 flex flex-wrap items-start gap-4 rounded-control border border-line bg-surface px-4 py-3">
          <div className="flex-1 min-w-0">
            <p className="text-body text-ink flex items-center flex-wrap gap-x-1">
              {opening.title}
              <Help
                summary="Zur Gründung geht auch die Eröffnungsbilanz an das Finanzamt, über denselben Dienst wie die E-Bilanz des Jahres."
                label="Erklärung zur E-Bilanz der Eröffnungsbilanz"
              >
                Bei Eröffnung des Betriebs ist auch die Eröffnungsbilanz nach amtlichem Datensatz zu
                übermitteln (§ 5b Abs. 1 Satz 4 EStG). Buchfink erzeugt die Bilanz auf den Tag der
                Beurkundung aus den Gründungsbuchungen, ohne Gewinn- und Verlustrechnung. Eine eigene
                Frist nennt das Gesetz nicht; Buchfink setzt die Frist der E-Bilanz des ersten
                Geschäftsjahres an. Mit dem Transferticket aus dem Übermittlungsprotokoll des Dienstes
                bestätigen Sie die Übermittlung hier.
              </Help>
            </p>
            <p className="text-caption text-ink-subtle">Bilanz auf den Tag der Gründung, ohne Gewinn- und Verlustrechnung</p>
          </div>
          <div className="shrink-0 flex items-center gap-3">
            <ObligationState obligation={opening} />
            <Button
              variant="secondary"
              size="sm"
              icon={<FileDown className="w-4 h-4" strokeWidth={1.5} />}
              loading={saving === 'opening'}
              disabled={saving !== null}
              onClick={() => void saveOpening()}
            >
              Als XBRL exportieren
            </Button>
            {isManual(opening) && (
              <Button
                variant="secondary"
                size="sm"
                disabled={writeLock.locked}
                title={writeLock.hint}
                onClick={() => setDeciding(opening)}
              >
                Übermittlung bestätigen
              </Button>
            )}
          </div>
        </div>
      )}
      {!report ? (
        !error && <SkeletonRows rows={6} />
      ) : (
        <div className="flex flex-col gap-6">
          <StatRow>
            <Stat label="Konten mit Saldo" value={String(rows.length)} />
            <Stat
              label="Befunde"
              value={String(blocking.length)}
              context="verhindern die Datei"
              tone={blocking.length > 0 ? 'negative' : 'positive'}
            />
            <Stat label="Auffangpositionen" value={String(fallbacks.length)} />
          </StatRow>

          {blocking.length > 0 && (
            <div>
              <h3 className="text-label text-ink-muted mb-2">Was die Datei verhindert</h3>
              <ul className="flex flex-col">
                {blocking.map((row, index) => (
                  <li key={`${row.account}-${index}`} className="flex items-start gap-3 py-2 border-t border-line first:border-t-0">
                    <span className="w-24 shrink-0 code-num text-caption text-ink-muted">{row.account || '—'}</span>
                    <span className="flex-1 text-body text-ink-muted">
                      {row.name ? `${row.name}: ` : ''}
                      {row.finding}
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          )}

          <div className="flex items-start gap-4 rounded-control border border-line bg-surface px-4 py-3">
            <p className="flex-1 text-body text-ink-muted flex items-center flex-wrap gap-x-1">
              Übermitteln Sie die Datei über einen Dienst wie ebilanzonline.de oder ebilanzplus.de.
              <Help summary="Mein ELSTER hat kein Formular für die E-Bilanz. Die Datei geht über einen E-Bilanz-Dienst an das Finanzamt." label="Erklärung zur Übermittlung der E-Bilanz">
                Bilanz und Gewinn- und Verlustrechnung sind nach amtlichem Datensatz zu übermitteln (§ 5b
                EStG). Buchfink erzeugt den Datensatz nach der amtlichen Taxonomie und übermittelt ihn
                nicht selbst. Das Übermittlungsprotokoll des Dienstes nennt das Transferticket; damit
                bestätigen Sie die Übermittlung hier. Nimmt der Dienst die XBRL-Datei nicht an, lesen
                Sie bei eBilanz+ die Saldenliste als Summen- und Saldenliste ein. Sie füllt nur Bilanz
                und Gewinn- und Verlustrechnung; die übrigen Angaben ergänzen Sie im Dienst. Weicht die
                Steuerbilanz ab, enthält die Saldenliste die Handelsbilanz, und die Überleitungsrechnung
                tragen Sie im Dienst nach (§ 5b Abs. 1 Satz 2 EStG).
              </Help>
            </p>
            {obligation && (
              <div className="shrink-0 flex items-center gap-3">
                <ObligationState obligation={obligation} />
                {isManual(obligation) && (
                  <Button
                    variant="secondary"
                    size="sm"
                    disabled={writeLock.locked}
                    title={writeLock.hint}
                    onClick={() => setDeciding(obligation)}
                  >
                    Übermittlung bestätigen
                  </Button>
                )}
              </div>
            )}
          </div>

          {transfers.length > 0 && (
            <Notice
              text={`Die Saldenliste enthält die Handelsbilanz. Die Steuerbilanz weicht an ${
                transfers.length === 1 ? 'einer Stelle' : `${transfers.length} Stellen`
              } ab (${transfers
                .map((t) => `Konto ${t.account}: ${formatCents(t.current)}`)
                .join(', ')}). Die XBRL-Datei enthält diese Überleitung. Nehmen Sie die Saldenliste, tragen Sie die Überleitung im Dienst nach; die Einzelheiten stehen unten unter „Überleitung zur Steuerbilanz“.`}
            />
          )}

          {warnings.length > 0 && (
            <Disclosure title="Hinweise" context={`${warnings.length} Hinweise zur Zuordnung`} className="mt-0 pt-4">
              <ul className="flex flex-col">
                {warnings.map((finding, index) => (
                  <li key={`${finding.code}-${index}`} className="py-2 border-t border-line first:border-t-0 text-body text-ink-muted">
                    {finding.message}
                  </li>
                ))}
              </ul>
            </Disclosure>
          )}

          <Disclosure
            title="Zuordnung der Konten"
            context={`${rows.length} Konten auf Positionen der Taxonomie`}
            className="mt-0 pt-4"
          >
            <Table density="kompakt">
              <Thead>
                <Tr>
                  <Th className="w-20">Konto</Th>
                  <Th>Bezeichnung</Th>
                  <Th numeric className="w-32">
                    Saldo
                  </Th>
                  <Th>Position</Th>
                </Tr>
              </Thead>
              <Tbody>
                {rows.map((row) => (
                  <Tr key={row.account}>
                    <Td code>{row.account}</Td>
                    <Td className="whitespace-normal">{row.name}</Td>
                    <Td numeric>{formatCents(row.balance)}</Td>
                    <Td className={cn('whitespace-normal', row.fallback ? 'text-attention-text' : 'text-ink-muted')}>
                      {row.positionLabel || '—'}
                      {row.fallback ? ' · Auffangposition' : ''}
                    </Td>
                  </Tr>
                ))}
              </Tbody>
            </Table>
          </Disclosure>

          {fallbacks.length > 0 && (
            <Disclosure title="Auffangpositionen" context="zugeordnet, aber nicht einzeln benannt" className="mt-0 pt-4">
              <Table density="kompakt">
                <Thead>
                  <Tr>
                    <Th>Position</Th>
                    <Th numeric className="w-24">
                      Konten
                    </Th>
                    <Th numeric className="w-36">
                      Betrag
                    </Th>
                  </Tr>
                </Thead>
                <Tbody>
                  {fallbacks.map((fallback) => (
                    <Tr key={fallback.key}>
                      <Td className="whitespace-normal">{fallback.label}</Td>
                      <Td numeric>{fallback.accounts}</Td>
                      <Td numeric>{formatCents(fallback.amount)}</Td>
                    </Tr>
                  ))}
                </Tbody>
              </Table>
            </Disclosure>
          )}

          <TaxBalanceKartei year={year} />
        </div>
      )}

      <ObligationDecisionDialog
        obligation={deciding}
        onClose={() => setDeciding(null)}
        onDone={onObligationsChanged}
      />
    </Section>
  );
};

/**
 * Überleitung zur Steuerbilanz und das Verzeichnis der steuerlichen
 * Wahlrechte. Beides bucht Claude; die App zeigt es zum Nachlesen.
 */
const TaxBalanceKartei: React.FC<{ year: number }> = ({ year }) => {
  const [open, setOpen] = useState(false);
  const [data, setData] = useState<[Reconciliation, TaxElectionRegister] | null>(null);
  const [error, setError] = useState('');
  useEffect(() => {
    if (!open || data) return;
    Promise.all([Api.getReconciliation(year), Api.getTaxElectionRegister(year)])
      .then(setData)
      .catch((e) => setError(message(e)));
  }, [open, data, year]);
  const [reconciliation, register] = data ?? [null, null];
  return (
    <Disclosure
      title="Überleitung zur Steuerbilanz"
      context="Abweichungen und steuerliche Wahlrechte"
      open={open}
      onOpenChange={setOpen}
      className="mt-0 pt-4"
    >
      {error ? (
        <Notice tone="negative" text={error} />
      ) : !data ? (
        <SkeletonRows rows={3} />
      ) : (
        <div className="flex flex-col gap-5">
          {reconciliation && reconciliation.rows.length === 0 ? (
            <p className="text-body text-ink-subtle">Handels- und Steuerbilanz stimmen überein.</p>
          ) : (
            <Table density="kompakt">
              <Thead>
                <Tr>
                  <Th>Position</Th>
                  <Th numeric className="w-32">
                    Handelsbilanz
                  </Th>
                  <Th numeric className="w-32">
                    Steuerbilanz
                  </Th>
                  <Th numeric className="w-32">
                    Differenz
                  </Th>
                </Tr>
              </Thead>
              <Tbody>
                {(reconciliation?.rows ?? []).map((row) => (
                  <Tr key={row.position}>
                    <Td className="whitespace-normal">{row.position}</Td>
                    <Td numeric className="text-ink-muted">
                      {formatCents(row.commercial)}
                    </Td>
                    <Td numeric>{formatCents(row.tax)}</Td>
                    <Td numeric>{formatCents(row.difference)}</Td>
                  </Tr>
                ))}
              </Tbody>
            </Table>
          )}
          {register && register.rows.length > 0 && (
            <div>
              <h4 className="text-label text-ink-muted mb-1.5">Steuerliche Wahlrechte</h4>
              <Table density="kompakt">
                <Tbody>
                  {register.rows.map((row) => (
                    <Tr key={row.assetId}>
                      <Td>{`${row.inventoryNumber} · ${row.name}`}</Td>
                      <Td className="text-caption text-ink-subtle">{row.provision}</Td>
                      <Td numeric className="w-32">
                        {formatCents(row.totalDifference)}
                      </Td>
                    </Tr>
                  ))}
                </Tbody>
              </Table>
            </div>
          )}
        </div>
      )}
    </Disclosure>
  );
};
