import React, { useEffect, useState } from 'react';
import { Download, Table2 } from 'lucide-react';
import { FinancialStatement } from '../types';
import { Api } from '../services/api';
import type { NavigateFn } from '../components/Sidebar';
import { downloadBlob } from '../utils/download';
import {
  DepthChoice,
  StatementTab,
  StatementView,
} from '../components/StatementView';
import {
  Button,
  EmptyState,
  Notice,
  PageHeader,
  SkeletonRows,
  TabPanel,
  Tabs,
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

type Tab = StatementTab;

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
          { value: 'klasse', label: 'Größenklasse und Fristen' },
        ]}
        value={tab}
        onValueChange={setTab}
        className="mt-6"
      >
        <TabPanel value="bilanz">{statementPanel('bilanz')}</TabPanel>
        <TabPanel value="guv">{statementPanel('guv')}</TabPanel>
        <TabPanel value="angaben">{statementPanel('angaben')}</TabPanel>
        <TabPanel value="klasse">{statementPanel('klasse')}</TabPanel>

      </Tabs>
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

