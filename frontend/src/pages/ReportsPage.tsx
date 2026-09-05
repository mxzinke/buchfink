import React, { useEffect, useState } from 'react';
import { Download, Table2 } from 'lucide-react';
import { FinancialStatement, StatementNotes } from '../types';
import { Api } from '../services/api';
import type { NavigateFn } from '../components/Sidebar';
import { downloadBlob } from '../utils/download';
import { formatCents, formatDate } from '../utils/formatters';
import {
  DepthChoice,
  StatementTab,
  StatementView,
} from '../components/StatementView';
import {
  Button,
  EmptyState,
  HelpPopover,
  HelpTooltip,
  Notice,
  PageHeader,
  Section,
  SkeletonRows,
  TabPanel,
  Table,
  Tabs,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  toast,
} from '../components/ui';

/**
 * Auswertungen: Bilanz und Gewinn- und Verlustrechnung.
 *
 * Die Zahlen kommen fertig aus dem Backend. Bilanz und GuV entstanden bis
 * Welle 2 hier durch Filtern nach Kontenklasse und waren damit eine zweite,
 * abweichende Wahrheit; jetzt liest die Ansicht die Gliederung nach den
 * §§ 266 und 275 HGB aus `GetStatement` und rechnet nichts nach.
 *
 * Die Umsatzsteuer stand hier bis Welle 3 als fünfter Reiter mit vier
 * Kennziffern, gerechnet nach Buchungsdatum. Sie ist eine eigene Ansicht
 * geworden: der Voranmeldungszeitraum folgt dem Leistungsdatum (§ 13 Abs. 1
 * UStG), und eine Anmeldung ist eine Entität mit Übermittlungsprotokoll und
 * keine Auswertung.
 */

/**
 * Der Anhang steht neben Bilanz und GuV, weil er zu ihnen gehört: § 264 Abs. 1
 * HGB macht ihn zum dritten Bestandteil des Abschlusses. Geschrieben wird er
 * unter „Abschlussbausteine" — hier steht er so, wie er in die Offenlegung
 * geht.
 */
type Tab = StatementTab | 'anhang';

/**
 * Der Anhang: die Freitexte, der Rückstellungsspiegel und die Überleitung zur
 * Steuerbilanz.
 *
 * Alle drei kommen fertig aus `GetStatement` — der Anhang entsteht mit dem
 * Abschluss und nicht daneben. Diese Ansicht zeigt sie und rechnet nichts nach;
 * insbesondere summiert sie den Spiegel nicht selbst, sondern nimmt die Zeile
 * `total`, die das Backend gerechnet hat.
 */
const NotesPanel: React.FC<{ notes: StatementNotes }> = ({ notes }) => {
  // Die Listen kommen aus dem Backend leer und nicht als null. Der Standardwert
  // steht trotzdem hier: eine fehlende Liste nähme im Render den ganzen Baum
  // mit, und der Anhang ist die Stelle, an der am ehesten nichts erfasst ist.
  const texts = notes?.texts ?? [];
  const mirror = notes?.provisionMirror;
  const mirrorRows = mirror?.rows ?? [];
  const reconciliation = notes?.reconciliation;
  const reconciliationRows = reconciliation?.rows ?? [];
  const written = texts.filter((entry) => entry.text.trim() !== '');

  return (
    <>
      <Section
        title="Angaben im Anhang"
        context={notes?.reference}
        divider={false}
        action={
          <HelpPopover label="Erklärung zum Anhang">
            Der Anhang erläutert Bilanz und Gewinn- und Verlustrechnung (§§ 284, 285 HGB).
            Kleinstkapitalgesellschaften dürfen ihn nach § 264 Abs. 1 Satz 5 HGB weglassen, wenn sie
            die Angaben unter der Bilanz machen. Geschrieben werden die Texte unter
            „Abschlussbausteine"; hier stehen sie so, wie sie in die Offenlegung gehen.
          </HelpPopover>
        }
      >
        {written.length === 0 ? (
          <EmptyState
            title="Noch kein Anhangtext erfasst"
            description="Die Abschnitte stehen unter „Abschlussbausteine“ zum Ausfüllen bereit."
          />
        ) : (
          <div className="max-w-3xl">
            {written.map((entry) => (
              <div key={entry.section} className="mb-6">
                <h3 className="text-label text-ink">
                  <span className="inline-flex items-center gap-1.5">
                    {entry.label}
                    {entry.hint && (
                      <HelpTooltip label={`Erklärung zu ${entry.label}`} content={entry.hint} />
                    )}
                  </span>
                </h3>
                <p className="text-caption text-ink-subtle mt-0.5">{entry.basis}</p>
                {/* Der Freitext behält seine Absätze: der Anwender hat sie gesetzt. */}
                <p className="text-body text-ink mt-2 whitespace-pre-wrap">{entry.text}</p>
              </div>
            ))}
          </div>
        )}
      </Section>

      <Section
        title="Rückstellungsspiegel"
        context="Anfangsbestand, Zuführung, Verbrauch, Auflösung, Aufzinsung, Endbestand"
        action={
          <HelpPopover label="Erklärung zum Rückstellungsspiegel">
            Der Spiegel zeigt je Art der Rückstellung, wie sich der Bestand im Geschäftsjahr
            entwickelt hat. Er geht per Definition auf: Anfangsbestand plus Zuführung und Aufzinsung
            minus Verbrauch und Auflösung ergibt den Endbestand.
          </HelpPopover>
        }
      >
        {mirrorRows.length === 0 ? (
          <EmptyState
            title="Keine Rückstellungen im Geschäftsjahr"
            description="Ohne Rückstellung bleibt der Spiegel leer; das ist keine fehlende Angabe."
          />
        ) : (
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th>Art</Th>
                <Th className="w-20">Konto</Th>
                <Th numeric className="w-32">Anfangsbestand</Th>
                <Th numeric className="w-28">Zuführung</Th>
                <Th numeric className="w-28">Verbrauch</Th>
                <Th numeric className="w-28">Auflösung</Th>
                <Th numeric className="w-28">Aufzinsung</Th>
                <Th numeric className="w-32">Endbestand</Th>
              </Tr>
            </Thead>
            <Tbody>
              {mirrorRows.map((row) => (
                <Tr key={`${row.kind}-${row.account}`}>
                  <Td className="whitespace-normal">{row.label}</Td>
                  <Td code>{row.account}</Td>
                  <Td numeric>{formatCents(row.opening)}</Td>
                  <Td numeric>{formatCents(row.additions)}</Td>
                  <Td numeric>{formatCents(row.used)}</Td>
                  <Td numeric>{formatCents(row.released)}</Td>
                  <Td numeric>{formatCents(row.unwinding)}</Td>
                  <Td numeric>{formatCents(row.closing)}</Td>
                </Tr>
              ))}
              {mirror?.total && (
                <Tr variant="sum">
                  <Td>Summe</Td>
                  <Td />
                  <Td numeric>{formatCents(mirror.total.opening)}</Td>
                  <Td numeric>{formatCents(mirror.total.additions)}</Td>
                  <Td numeric>{formatCents(mirror.total.used)}</Td>
                  <Td numeric>{formatCents(mirror.total.released)}</Td>
                  <Td numeric>{formatCents(mirror.total.unwinding)}</Td>
                  <Td numeric>{formatCents(mirror.total.closing)}</Td>
                </Tr>
              )}
            </Tbody>
          </Table>
        )}
      </Section>

      <Section
        title="Überleitung zur Steuerbilanz"
        context={
          reconciliation?.cutoff
            ? `Stichtag ${formatDate(reconciliation.cutoff)}`
            : 'Wo Handels- und Steuerbilanz auseinanderfallen'
        }
        action={
          <HelpPopover label="Erklärung zur Überleitung">
            § 60 Abs. 2 EStDV verlangt, die Handelsbilanz durch Zusätze oder Anmerkungen an die
            steuerlichen Vorschriften anzupassen, wo beide auseinanderfallen. In Buchfink sind das
            die Sonderabschreibung nach § 7g Abs. 5 EStG und die abweichende Abzinsung der
            Rückstellungen mit 5,5 % (§ 6 Abs. 1 Nr. 3a Buchst. e EStG).
          </HelpPopover>
        }
      >
        {reconciliationRows.length === 0 ? (
          <EmptyState
            title="Keine Abweichung zwischen Handels- und Steuerbilanz"
            description="Ohne steuerliches Wahlrecht und ohne abgezinste Rückstellung stimmen beide überein."
          />
        ) : (
          <Table density="kompakt">
            <Thead>
              <Tr>
                <Th>Position</Th>
                <Th numeric className="w-40">Handelsbilanz</Th>
                <Th numeric className="w-40">Steuerbilanz</Th>
                <Th numeric className="w-40">Differenz</Th>
                <Th className="w-64">Rechtsgrundlage</Th>
              </Tr>
            </Thead>
            <Tbody>
              {reconciliationRows.map((row) => (
                <Tr key={row.position}>
                  <Td className="whitespace-normal">
                    <span className="inline-flex items-center gap-1.5">
                      {row.position}
                      {row.explanation && (
                        <HelpTooltip
                          label={`Erklärung zu ${row.position}`}
                          content={row.explanation}
                        />
                      )}
                    </span>
                  </Td>
                  <Td numeric>{formatCents(row.commercial)}</Td>
                  <Td numeric>{formatCents(row.tax)}</Td>
                  <Td numeric>{formatCents(row.difference)}</Td>
                  <Td className="text-ink-muted whitespace-normal">{row.basis}</Td>
                </Tr>
              ))}
              <Tr variant="sum">
                <Td>Wirkung auf das Eigenkapital</Td>
                <Td />
                <Td />
                <Td numeric>{formatCents(reconciliation?.equityEffect ?? 0)}</Td>
                <Td />
              </Tr>
            </Tbody>
          </Table>
        )}
        {reconciliation?.note && (
          <p className="text-caption text-ink-subtle mt-3">{reconciliation.note}</p>
        )}
      </Section>
    </>
  );
};

export interface ReportsPageProps {
  /** Das Geschäftsjahr aus der Kopfzeile. Der Abschluss folgt ihm. */
  year: number;
  /** Weg von der Gliederungszeile über das Konto ins Kontoblatt (GOB-02). */
  onNavigate?: NavigateFn;
}

export const ReportsPage: React.FC<ReportsPageProps> = ({ year, onNavigate }) => {
  const [tab, setTab] = useState<Tab>('bilanz');
  const [statement, setStatement] = useState<FinancialStatement | null>(null);
  const [depth, setDepth] = useState<DepthChoice>('auto');
  const [loadingStatement, setLoadingStatement] = useState(true);
  const [statementError, setStatementError] = useState<string | null>(null);
  const [exporting, setExporting] = useState<'pdf' | 'csv' | null>(null);

  useEffect(() => {
    void loadStatement();
    // Die Tiefe ist ein Parameter des Aufbaus, kein Filter der Ansicht: eine
    // verkürzte Bilanz wird gebaut und nicht ausgeblendet.
  }, [year, depth]);

  async function loadStatement() {
    setLoadingStatement(true);
    try {
      setStatement(await Api.getStatement(year, depth === 'auto' ? '' : depth));
      setStatementError(null);
    } catch (e) {
      // Ein Befund aus dem Aufstellen — etwa eine Bilanz, die nicht aufgeht —
      // ist ein Fehler und kein Leerzustand: er gehört als Hinweisfläche über
      // die Ansicht (§10), damit der Satz des Backends lesbar bleibt.
      setStatement(null);
      setStatementError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoadingStatement(false);
    }
  }

  async function handleExport(kind: 'pdf' | 'csv') {
    setExporting(kind);
    try {
      if (kind === 'pdf') {
        const base64 = await Api.exportStatementPDF(year);
        downloadBlob(
          `jahresabschluss-${year}.pdf`,
          new Blob([bufferFromBase64(base64)], { type: 'application/pdf' }),
        );
      } else {
        const csv = await Api.exportStatementCSV(year);
        // Das Byte-Order-Mark steht davor, damit die Tabellenkalkulation die
        // Umlaute als UTF-8 liest und nicht als Zeichensalat.
        downloadBlob(
          `jahresabschluss-${year}.csv`,
          new Blob([`﻿${csv}`], { type: 'text/csv;charset=utf-8' }),
        );
      }
      toast.success(kind === 'pdf' ? 'Abschluss als PDF gespeichert.' : 'Gliederung als CSV gespeichert.');
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setExporting(null);
    }
  }

  /** Bilanz, Staffel, Angaben und Größenklasse — vier Sichten auf einen Aufbau. */
  const statementPanel = (view: StatementTab) => {
    if (loadingStatement) return <SkeletonRows rows={10} />;
    if (!statement) {
      // Der Fehler steht als Hinweisfläche über den Reitern; der Leerzustand
      // gilt allein dem Fall, dass es nichts zu zeigen gibt.
      return statementError ? null : (
        <EmptyState
          title={`Kein Abschluss für ${year}`}
          description="Das Backend hat keine Gliederung geliefert."
        />
      );
    }
    return (
      <StatementView
        data={statement}
        view={view}
        depth={depth}
        onDepthChange={setDepth}
        onNavigate={onNavigate}
      />
    );
  };

  return (
    <div className="max-w-[1200px] mx-auto px-8 py-8">
      <PageHeader
        title="Auswertungen"
        context={`Geschäftsjahr ${year} · berechnet aus den erfassten Buchungen`}
        action={
          <div className="flex gap-2">
              <Button
                variant="secondary"
                icon={<Table2 className="w-4 h-4" strokeWidth={1.5} />}
                onClick={() => handleExport('csv')}
                loading={exporting === 'csv'}
                disabled={!statement || exporting !== null}
              >
                Als CSV
              </Button>
              <Button
                variant="primary"
                icon={<Download className="w-4 h-4" strokeWidth={1.5} />}
                onClick={() => handleExport('pdf')}
                loading={exporting === 'pdf'}
                disabled={!statement || exporting !== null}
            >
              Als PDF
            </Button>
          </div>
        }
      />

      {statementError && (
        <Notice
          className="mt-6"
          tone="negative"
          text={statementError}
          action={
            <Button variant="secondary" size="sm" onClick={() => void loadStatement()}>
              Erneut aufstellen
            </Button>
          }
        />
      )}

      <Tabs<Tab>
        items={[
          { value: 'bilanz', label: 'Bilanz' },
          { value: 'guv', label: 'Gewinn- und Verlustrechnung' },
          { value: 'angaben', label: 'Angaben unter der Bilanz' },
          { value: 'anhang', label: 'Anhang' },
          { value: 'klasse', label: 'Größenklasse und Fristen' },
        ]}
        value={tab}
        onValueChange={setTab}
        className="mt-6"
      >
        <TabPanel value="bilanz">{statementPanel('bilanz')}</TabPanel>
        <TabPanel value="guv">{statementPanel('guv')}</TabPanel>
        <TabPanel value="angaben">{statementPanel('angaben')}</TabPanel>
        <TabPanel value="anhang">
          {loadingStatement ? (
            <SkeletonRows rows={8} />
          ) : statement ? (
            <NotesPanel notes={statement.notes} />
          ) : statementError ? null : (
            <EmptyState
              title={`Kein Anhang für ${year}`}
              description="Der Anhang entsteht mit dem Abschluss; ohne Gliederung gibt es ihn nicht."
            />
          )}
        </TabPanel>
        <TabPanel value="klasse">{statementPanel('klasse')}</TabPanel>

      </Tabs>

      {/* Zwei Auswertungen der Welle 5c stehen auf der Seite „Nebenpflichten",
          weil sie zu Verzeichnissen gehören, die dort geführt werden. Der
          Verweis steht hier, weil man einen Bericht unter „Auswertungen"
          sucht. */}
      {onNavigate && (
        <Section
          title="Weitere Auswertungen"
          context="Steuerliche Nebenpflichten"
          className="mt-8"
        >
          <div className="flex flex-col gap-3">
            <p className="text-body text-ink-muted">
              Die nicht abziehbaren Betriebsausgaben je Kategorie (§ 4 Abs. 5 EStG) und der
              Belegnachweis der steuerfreien innergemeinschaftlichen Lieferungen (§§ 17a bis 17c
              UStDV) stehen dort, wo auch die Verzeichnisse dazu geführt werden.
            </p>
            <div className="flex gap-2">
              <Button
                variant="secondary"
                size="sm"
                onClick={() => onNavigate('obligations', { obligationsTab: 'nondeductible' })}
              >
                Nicht abziehbare Betriebsausgaben
              </Button>
              <Button
                variant="secondary"
                size="sm"
                onClick={() => onNavigate('obligations', { obligationsTab: 'evidence' })}
              >
                Belegnachweis ig. Lieferungen
              </Button>
            </div>
          </div>
        </Section>
      )}
    </div>
  );
};

// -------------------------------------------------------------------------

/** Base64 aus der Bridge in Bytes — das PDF kommt wie der Rechnungsexport. */
function bufferFromBase64(base64: string): ArrayBuffer {
  const binary = atob(base64);
  const buffer = new ArrayBuffer(binary.length);
  const bytes = new Uint8Array(buffer);
  for (let i = 0; i < binary.length; i++) bytes[i] = binary.charCodeAt(i);
  return buffer;
}

