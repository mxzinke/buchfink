import React, { useEffect, useState } from 'react';
import { AlertTriangle } from 'lucide-react';
import type { JournalEntry, JournalLine, ReceiptPreview } from '../../types';
import { formatCents } from '../../utils/formatters';
import { Notice, cn, type Status } from '../../components/ui';

/** Die Fehlermeldung eines Bridge-Aufrufs als Text. */
export function errorText(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}

/** Heute als JJJJ-MM-TT in der Ortszeit. */
export function todayISO(): string {
  const now = new Date();
  const pad = (n: number) => String(n).padStart(2, '0');
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`;
}

/** Bruttobetrag einer Buchung, gemessen an der Sollseite. */
export function grossOf(entry: JournalEntry): number {
  return entry.lines.filter((line) => line.side === 'S').reduce((sum, line) => sum + line.amount, 0);
}

/**
 * Der Zustand einer Buchung im Vokabular des §11.3: storniert, wenn sie eine
 * Generalumkehr ist oder eine hat; festgeschrieben, sobald die Festschreibung
 * sie erfasst hat; sonst gebucht.
 */
export function entryStatus(entry: JournalEntry, reversed: Set<number>): Status {
  if (entry.kind === 'reversal' || reversed.has(entry.id)) return 'storniert';
  if (entry.committedAt) return 'festgeschrieben';
  return 'gebucht';
}

/** Die Buchungen, zu denen es eine Generalumkehr gibt. */
export function reversedIds(entries: JournalEntry[]): Set<number> {
  return new Set(entries.map((entry) => entry.reversalOfId).filter((id): id is number => Boolean(id)));
}

/**
 * Der Buchungssatz: Soll links, Haben rechts, Summe mit Doppellinie (§11.1).
 * Steht in Seitenleiste und Dialog und bekommt dort keine eigene Fläche.
 */
export const PostingLines: React.FC<{
  lines: JournalLine[];
  currency?: string;
  className?: string;
}> = ({ lines, currency = 'EUR', className }) => {
  const debit = lines.filter((l) => l.side === 'S').reduce((s, l) => s + l.amount, 0);
  const credit = lines.filter((l) => l.side === 'H').reduce((s, l) => s + l.amount, 0);
  return (
    <table className={cn('w-full text-body', className)}>
      <thead>
        <tr className="[&>th]:h-7 [&>th]:text-label [&>th]:font-medium [&>th]:text-ink-subtle [&>th]:border-b [&>th]:border-line-strong">
          <th className="text-left w-16">Konto</th>
          <th className="text-left">Bezeichnung</th>
          <th className="text-right w-28">Soll</th>
          <th className="text-right w-28">Haben</th>
        </tr>
      </thead>
      <tbody>
        {lines.map((line, index) => (
          <tr key={`${line.position}-${index}`} className="[&>td]:h-8 [&>td]:border-b [&>td]:border-line">
            <td className="code-num text-caption text-ink-muted">{line.account}</td>
            <td className="text-ink truncate max-w-0" title={line.accountName}>
              {line.accountName || '—'}
            </td>
            <td className="text-right num">{line.side === 'S' ? formatCents(line.amount, currency) : ''}</td>
            <td className="text-right num">{line.side === 'H' ? formatCents(line.amount, currency) : ''}</td>
          </tr>
        ))}
        <tr className="[&>td]:h-9 [&>td]:rule-total [&>td]:font-semibold">
          <td />
          <td>Summe</td>
          <td className="text-right num">{formatCents(debit, currency)}</td>
          <td className="text-right num">{formatCents(credit, currency)}</td>
        </tr>
      </tbody>
    </table>
  );
};

/** Eine Zwischenüberschrift in Seitenleiste und Dialog. */
export const PanelHeading: React.FC<{ children: React.ReactNode; className?: string }> = ({
  children,
  className,
}) => <h3 className={cn('text-label text-ink-muted mb-2', className)}>{children}</h3>;

/**
 * Die Vorschau eines Belegs oder Rechnungsdokuments. Das Dokument ist ein
 * Fremdkörper in der Oberfläche und bekommt deshalb eine eigene Fläche (§6.2,
 * Fall 3).
 */
export const DocumentPreview: React.FC<{
  /** Lädt die Vorschau; ändert sich der Schlüssel, wird neu geladen. */
  load: () => Promise<ReceiptPreview>;
  loadKey: string;
  /** Ersetzt die Fehlermeldung, etwa beim Kontoauszug in XML. */
  fallback?: string;
  className?: string;
}> = ({ load, loadKey, fallback, className }) => {
  const [preview, setPreview] = useState<ReceiptPreview | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setPreview(null);
    setError(null);
    load()
      .then((p) => {
        if (!cancelled) setPreview(p);
      })
      .catch((e) => {
        if (!cancelled) setError(errorText(e));
      });
    return () => {
      cancelled = true;
    };
    // Die Ladefunktion entsteht bei jedem Rendern neu; maßgeblich ist der Schlüssel.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [loadKey]);

  return (
    <div className={className}>
      <div className="rounded-card border border-line bg-sunken min-h-[16rem] flex items-center justify-center p-3">
        {error ? (
          <div className="text-center max-w-sm">
            <AlertTriangle className="w-5 h-5 mx-auto text-attention" strokeWidth={1.5} />
            <p className="text-body text-ink-muted mt-2">{fallback ?? error}</p>
          </div>
        ) : !preview ? (
          <span className="text-body text-ink-subtle">Vorschau wird geladen …</span>
        ) : preview.mimeType === 'application/pdf' ? (
          <iframe
            title="Belegvorschau"
            src={preview.dataUrl}
            className="w-full h-[28rem] rounded-control bg-surface"
          />
        ) : (
          <img src={preview.dataUrl} alt="Beleg" className="max-h-[28rem] rounded-control" />
        )}
      </div>
      {preview && !preview.intact && (
        <Notice
          tone="negative"
          className="mt-3"
          text="Die Datei passt nicht mehr zu ihrer Prüfsumme. Sie wurde nach dem Ablegen verändert."
        />
      )}
    </div>
  );
};
