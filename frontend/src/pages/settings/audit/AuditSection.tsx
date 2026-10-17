import React, { useEffect, useState } from 'react';
import { useWorkspace } from '../../../workspace';
import { formatDate, formatDateTime } from '../../../utils/formatters';
import { InvoiceNumberRange } from '../../../components/InvoiceNumberRange';
import { Disclosure } from '../../../components/ui';
import { DataExportPanel } from './DataExportPanel';
import { ExaminerPanel } from './ExaminerPanel';
import { HistoryPanel } from './HistoryPanel';
import { IntegrityPanel } from './IntegrityPanel';
import { ProcDocPanel } from './ProcDocPanel';
import { ProtocolPanel } from './ProtocolPanel';

/** Die Zeilen des Abschnitts; ihr Schlüssel ist zugleich die Kennung im Ziel. */
const ROWS = [
  'integritaet',
  'protokoll',
  'pruefermodus',
  'datenueberlassung',
  'verfahrensdokumentation',
  'nummernkreise',
  'systemhistorie',
] as const;

type Row = (typeof ROWS)[number];

/** Die frühere Seite „Betriebsprüfung": Prüfermodus und Datenüberlassung. */
const TAX_AUDIT = 'betriebspruefung';

function isRow(value: string): value is Row {
  return (ROWS as readonly string[]).includes(value);
}

/** Welche Zeile ein Ziel aufschlägt; jede andere Kennung ist eine Prüfregel. */
function rowsFor(target: string | undefined): Row[] {
  if (!target) return [];
  if (target === TAX_AUDIT) return ['pruefermodus', 'datenueberlassung'];
  if (isRow(target)) return [target];
  return ['integritaet'];
}

/** Der Anker der ersten Zeile, die ein Ziel aufschlägt. */
export function auditAnchor(target: string | undefined): string | undefined {
  const [first] = rowsFor(target);
  return first ? `pruefung-${first}` : undefined;
}

const Meta: React.FC<{ tone?: 'positive' | 'negative' | 'attention'; children: React.ReactNode }> = ({
  tone,
  children,
}) => (
  <span
    className={
      tone === 'positive'
        ? 'text-caption text-positive-text'
        : tone === 'negative'
          ? 'text-caption text-negative-text'
          : tone === 'attention'
            ? 'text-caption text-attention-text'
            : 'text-caption text-ink-subtle'
    }
  >
    {children}
  </span>
);

/**
 * Prüfung & GoBD: was ein Prüfer sehen will und was der Mensch für ihn tut.
 * Jede Zeile zeigt eingeklappt ihren Stand und lädt erst beim Aufklappen.
 */
export const AuditSection: React.FC<{ target?: string }> = ({ target }) => {
  const workspace = useWorkspace();
  const [open, setOpen] = useState<Row[]>(() => rowsFor(target));
  const rule = target && target !== TAX_AUDIT && !isRow(target) ? target : undefined;

  useEffect(() => {
    const wanted = rowsFor(target);
    if (wanted.length > 0) setOpen((prev) => Array.from(new Set([...prev, ...wanted])));
  }, [target]);

  const toggle = (row: Row) => (next: boolean) =>
    setOpen((prev) => (next ? [...prev, row] : prev.filter((r) => r !== row)));

  const { integrity, isCheckingIntegrity, appConfig } = workspace;
  const broken = integrity !== null && !integrity.isValid;

  return (
    <div className="border-t border-line">
      <Disclosure
        id="pruefung-integritaet"
        variant="row"
        title="Integrität"
        context={
          integrity
            ? `Buchungen, Protokoll und Festschreibungen · geprüft am ${formatDateTime(integrity.checkedAt)}`
            : 'Buchungen, Protokoll und Festschreibungen'
        }
        meta={
          isCheckingIntegrity ? (
            <Meta>Prüfung läuft</Meta>
          ) : integrity ? (
            <Meta tone={broken ? 'negative' : 'positive'}>{broken ? 'Verletzt' : 'Unverändert'}</Meta>
          ) : undefined
        }
        open={open.includes('integritaet')}
        onOpenChange={toggle('integritaet')}
      >
        <IntegrityPanel
          integrity={integrity}
          isChecking={isCheckingIntegrity}
          onRefresh={() => void workspace.refreshIntegrity()}
          initialRule={rule}
        />
      </Disclosure>

      <Disclosure
        id="pruefung-protokoll"
        variant="row"
        title="Änderungsprotokoll"
        context="Wer wann was geändert hat, mit Vorher und Nachher"
        open={open.includes('protokoll')}
        onOpenChange={toggle('protokoll')}
      >
        <ProtocolPanel />
      </Disclosure>

      <Disclosure
        id="pruefung-pruefermodus"
        variant="row"
        title="Prüfermodus"
        context="Sperrt jede Änderung während einer Betriebsprüfung, auch für verbundene Apps"
        meta={
          appConfig.readOnly ? (
            <Meta tone="attention">Bis {formatDate(appConfig.readOnlyUntil ?? '')}</Meta>
          ) : (
            <Meta>Aus</Meta>
          )
        }
        open={open.includes('pruefermodus')}
        onOpenChange={toggle('pruefermodus')}
      >
        <ExaminerPanel appConfig={appConfig} onAppConfigChange={workspace.setAppConfig} />
      </Disclosure>

      <Disclosure
        id="pruefung-datenueberlassung"
        variant="row"
        title="Datenüberlassung"
        context="Die Buchführung eines Jahres für die Software des Finanzamts"
        open={open.includes('datenueberlassung')}
        onOpenChange={toggle('datenueberlassung')}
      >
        <DataExportPanel year={workspace.year} availableYears={workspace.availableYears} />
      </Disclosure>

      <Disclosure
        id="pruefung-verfahrensdokumentation"
        variant="row"
        title="Verfahrensdokumentation"
        context="Wie die Buchführung zustande kommt, als Dokument je Fassung"
        open={open.includes('verfahrensdokumentation')}
        onOpenChange={toggle('verfahrensdokumentation')}
      >
        <ProcDocPanel />
      </Disclosure>

      <Disclosure
        id="pruefung-nummernkreise"
        variant="row"
        title="Nummernkreise"
        context={`Vergebene Rechnungsnummern ohne Dokument im Jahr ${workspace.year}`}
        open={open.includes('nummernkreise')}
        onOpenChange={toggle('nummernkreise')}
      >
        <InvoiceNumberRange key={workspace.year} year={workspace.year} />
      </Disclosure>

      <Disclosure
        id="pruefung-systemhistorie"
        variant="row"
        title="Systemhistorie"
        context="Schemaläufe, Datenübernahmen und Programmfassungen"
        open={open.includes('systemhistorie')}
        onOpenChange={toggle('systemhistorie')}
      >
        <HistoryPanel />
      </Disclosure>
    </div>
  );
};
