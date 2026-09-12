import React, { useEffect, useState } from 'react';
import { Code, Download } from 'lucide-react';
import { Api } from '../services/api';
import type { MappingReport } from '../types';
import { formatCents } from '../utils/formatters';
import {
  Button,
  Notice,
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
  toast,
} from '../components/ui';

/**
 * E-Bilanz nach § 5b EStG.
 *
 * Der Zuordnungsbericht läuft vor der Erzeugung und nicht danach: eine
 * Instanz, in der ein Konto auf einer Sammelposition verschwunden ist, lässt
 * sich beim Finanzamt nicht zurückholen. Blockierende Befunde verhindern
 * deshalb die Datei — die Bilanzansicht zeigt dieselben Konten weiter an.
 */

export interface EBilanzPageProps {
  /** Das Geschäftsjahr aus der Kopfzeile. */
  year: number;
}

export const EBilanzPage: React.FC<EBilanzPageProps> = ({ year }) => {
  const [xbrlContent, setXbrlContent] = useState('');
  const [report, setReport] = useState<MappingReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [showRawXML, setShowRawXML] = useState(false);

  useEffect(() => {
    void load();
  }, [year]);

  async function load() {
    setLoading(true);
    setXbrlContent('');
    try {
      const mapping = await Api.getEBilanzMappingReport(year);
      setReport({
        ...mapping,
        rows: mapping.rows ?? [],
        blocking: mapping.blocking ?? [],
        fallbacks: mapping.fallbacks ?? [],
      });
      // Ohne vollständige Zuordnung wird gar nicht erst erzeugt: der Aufruf
      // schlüge mit derselben Begründung fehl, die schon im Bericht steht.
      if (mapping.canExport) {
        setXbrlContent(await Api.exportEBilanzXBRL());
      }
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }

  function handleDownloadXBRL() {
    const blob = new Blob([xbrlContent], { type: 'application/xml;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `e-bilanz-${year}.xbrl`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
    toast.success('E-Bilanz-Datei gespeichert.');
  }

  return (
    <div className="max-w-[1200px] mx-auto px-8 py-8">
      <PageHeader helpSummary="Die E-Bilanz ist die elektronische Bilanz fürs Finanzamt. Buchfinks Export ist noch ungeprüft."
        title="E-Bilanz"
        context={`Geschäftsjahr ${year} · Export noch ungeprüft`}
        explain={
          <>
            Bilanz und Gewinn- und Verlustrechnung sind nach amtlich vorgeschriebenem Datensatz
            elektronisch an das Finanzamt zu übermitteln (§ 5b EStG). Buchfink erzeugt den
            vorläufigen Datensatz aus den Buchungen dieses Geschäftsjahres.
            Die Zuordnung zum amtlichen Format ist noch ungeprüft. Vor einer Einreichung
            müssen Format und Inhalt geprüft und mit einem geeigneten Programm übermittelt werden.
          </>
        }
        action={
          <div className="flex items-center gap-2">
            <Button
              variant="secondary"
              icon={<Code className="w-4 h-4" strokeWidth={1.5} />}
              onClick={() => setShowRawXML((v) => !v)}
            >
              {showRawXML ? 'Rohdaten ausblenden' : 'Rohdaten anzeigen'}
            </Button>
            <Button
              variant="primary"
              icon={<Download className="w-4 h-4" strokeWidth={1.5} />}
              onClick={handleDownloadXBRL}
              disabled={!xbrlContent}
              title={
                xbrlContent
                  ? undefined
                  : 'Die Datei entsteht erst, wenn jedes Konto mit Saldo zugeordnet ist'
              }
            >
              Datei herunterladen
            </Button>
          </div>
        }
      />

      {loading ? (
        <div className="mt-8">
          <SkeletonRows rows={8} />
        </div>
      ) : (
        <>
          {report && report.blocking.length > 0 && (
            <Notice
              className="mt-6"
              text={`${report.blocking.length} Konten mit Saldo haben keine Zuordnung; die Datei wird erst nach ihrer Klärung erzeugt.`}
            />
          )}

          {report && (
            <div className="mt-8">
              <StatRow>
                <Stat
                  label="Konten mit Saldo"
                  value={String(report.rows.length)}
                  context={`Geschäftsjahr ${report.fiscalYear}`}
                />
                <Stat
                  label="Ohne Zuordnung"
                  value={String(report.blocking.length)}
                  context="verhindern die Erzeugung"
                  tone={report.blocking.length > 0 ? 'negative' : 'positive'}
                />
                <Stat
                  label="Auffangpositionen"
                  value={String(report.fallbacks.length)}
                  context={'Sammelposten „sonstige …" der Gliederung'}
                />
                <Stat
                  label="Elemente ungeprüft"
                  value={String(report.unverified)}
                  context={`Taxonomie ${report.taxonomyVersion}`}
                />
              </StatRow>
            </div>
          )}

          {report && report.blocking.length > 0 && (
            <Section
              title="Konten ohne Zuordnung"
              context="Jede Zeile nennt, was fehlt"
              className="mt-8"
              divider={false}
            >
              <Table density="kompakt">
                <Thead>
                  <Tr>
                    <Th className="w-28">Konto</Th>
                    <Th className="w-64">Bezeichnung</Th>
                    <Th>Befund</Th>
                  </Tr>
                </Thead>
                <Tbody>
                  {report.blocking.map((row) => (
                    <Tr key={row.account}>
                      <Td code>{row.account}</Td>
                      <Td>{row.name}</Td>
                      <Td className="text-ink-muted whitespace-normal">{row.finding}</Td>
                    </Tr>
                  ))}
                </Tbody>
              </Table>
            </Section>
          )}

          <Section helpSummary="Hier sehen Sie, wie die Konten dem elektronischen Bilanzformat zugeordnet werden."
            title="Zuordnung der Konten"
            context={`SKR04 über die Gliederung auf die Taxonomie ${report?.taxonomyVersion ?? ''}`}
            divider={Boolean(report && report.blocking.length > 0)}
            className="mt-8"
            explain={
              <>
                {report?.taxonomyNote ??
                'Die Datei enthält Bilanz, Gewinn- und Verlustrechnung, Kontennachweis und Anlagenspiegel. Die Zuordnung ist ungeprüft. Buchfink übermittelt sie nicht an das Finanzamt.'}
              </>
            }
          >
            <Table density="kompakt">
              <Thead>
                <Tr>
                  <Th className="w-24">Konto</Th>
                  <Th className="w-56">Bezeichnung</Th>
                  <Th numeric className="w-36">
                    Saldo
                  </Th>
                  <Th>Gliederungsposition</Th>
                  <Th className="w-56">Taxonomie-Element</Th>
                  <Th className="w-24">Geprüft</Th>
                </Tr>
              </Thead>
              <Tbody>
                {(report?.rows ?? []).map((row) => (
                  <Tr key={row.account}>
                    <Td code>{row.account}</Td>
                    <Td>{row.name}</Td>
                    <Td numeric>{formatCents(row.balance)}</Td>
                    <Td className="text-ink-muted whitespace-normal">{row.positionLabel || '—'}</Td>
                    <Td code>{row.element || '—'}</Td>
                    <Td className={row.verified ? undefined : 'text-attention-text'}>
                      {row.element ? (row.verified ? 'Ja' : 'Nein') : '—'}
                    </Td>
                  </Tr>
                ))}
              </Tbody>
            </Table>
          </Section>

          {report && report.fallbacks.length > 0 && (
            <Section
              title="Auffangpositionen"
              context="Was hier steht, ist zugeordnet, aber nicht benannt"
            >
              <Table density="kompakt">
                <Thead>
                  <Tr>
                    <Th>Position</Th>
                    <Th numeric className="w-28">
                      Konten
                    </Th>
                    <Th numeric className="w-40">
                      Betrag
                    </Th>
                  </Tr>
                </Thead>
                <Tbody>
                  {report.fallbacks.map((fallback) => (
                    <Tr key={fallback.key}>
                      <Td className="whitespace-normal">{fallback.label}</Td>
                      <Td numeric>{fallback.accounts}</Td>
                      <Td numeric>{formatCents(fallback.amount)}</Td>
                    </Tr>
                  ))}
                </Tbody>
              </Table>
            </Section>
          )}
        </>
      )}

      {showRawXML && (
        <Section title="Rohdaten" context={`${xbrlContent.length.toLocaleString('de-DE')} Zeichen`}>
          {/* Technische Ausgabe, deshalb Monospace und die dunkle Fläche (§4.1). */}
          <pre className="rounded-card border border-shell-line bg-shell text-shell-text
                          font-mono text-caption leading-relaxed p-5 max-h-96 overflow-auto">
            {xbrlContent}
          </pre>
        </Section>
      )}
    </div>
  );
};
