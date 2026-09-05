import React from 'react';
import { Progress as Base } from '@base-ui/react/progress';
import { AlertTriangle, OctagonAlert } from 'lucide-react';
import { toast as sonner } from 'sonner';
import { cn } from './cn';

export interface NoticeProps {
  /**
   * `attention` (Bernstein) heißt: etwas ist zu klären, die Ansicht bleibt
   * benutzbar. `negative` (Rosé) heißt: das Backend hat abgelehnt (§10).
   */
  tone?: 'attention' | 'negative';
  /** Ein Satz (§15.1). Was länger ist, gehört hinter ein Erklärzeichen. */
  text: string;
  /** Die eine Aktion, die weiterhilft — etwa „Erneut aufstellen". */
  action?: React.ReactNode;
  className?: string;
}

/**
 * Hinweisfläche nach §10 und §11.4: ein Satz auf der Fläche, kein Toast. Der
 * Befund bleibt stehen, bis er behoben ist.
 *
 * Sie steht als Baustein hier und nicht als Klassenkette in einer Seite: sonst
 * trägt jede Seite ihre eigene Kopie desselben Musters, und die Kopien laufen
 * auseinander.
 */
export const Notice: React.FC<NoticeProps> = ({ tone = 'attention', text, action, className }) => {
  const negative = tone === 'negative';
  const Icon = negative ? OctagonAlert : AlertTriangle;
  return (
    <div
      className={cn(
        'flex items-start gap-3 rounded-control border px-4 py-3',
        negative
          ? 'border-negative-line bg-negative-soft'
          : 'border-attention-line bg-attention-soft',
        className,
      )}
      role={negative ? 'alert' : 'status'}
    >
      <Icon
        className={cn('w-4 h-4 mt-0.5 shrink-0', negative ? 'text-negative-text' : 'text-attention-text')}
        strokeWidth={1.5}
      />
      <p className="text-body text-ink-muted flex-1">{text}</p>
      {action && <div className="shrink-0">{action}</div>}
    </div>
  );
};

/**
 * Ladezustand nach §8.4: unter 200 ms nichts, darüber Skelettzeilen in der Form
 * des erwarteten Inhalts. Kein Spinner, der die Seite ersetzt.
 */
export const Skeleton: React.FC<{ className?: string; style?: React.CSSProperties }> = ({
  className,
  style,
}) => (
  <span
    className={cn('block h-2.5 rounded-[3px] bg-sunken', className)}
    style={style}
    aria-hidden="true"
  />
);

export interface SkeletonRowsProps {
  rows?: number;
  /** Breiten je Zeile in Prozent, wiederholt sich. */
  widths?: number[];
  className?: string;
}

export const SkeletonRows: React.FC<SkeletonRowsProps> = ({
  rows = 5,
  widths = [100, 88, 94, 72],
  className,
}) => (
  // Breite als Inline-Stil: eine zusammengesetzte Klasse wie `w-[88%]` sieht
  // Tailwind beim Bauen nicht und würde nicht erzeugt.
  <div className={cn('flex flex-col gap-3', className)} role="status" aria-label="Wird geladen">
    {Array.from({ length: rows }, (_, i) => (
      <Skeleton key={i} style={{ width: `${widths[i % widths.length]}%` }} />
    ))}
  </div>
);

export interface ProgressProps {
  /** 0 bis 100, oder null für unbestimmten Fortschritt. */
  value: number | null;
  label: string;
  /** Etwa "128 von 312 Umsätzen". */
  detail?: string;
  className?: string;
}

/**
 * Für Vorgänge über 10 Sekunden: Import, XBRL-Export. Immer mit Anzahl, damit
 * man abschätzen kann, ob sich das Warten lohnt (§8.4).
 */
export const Progress: React.FC<ProgressProps> = ({ value, label, detail, className }) => (
  <Base.Root value={value} className={cn('flex flex-col gap-2', className)}>
    <div className="flex items-baseline justify-between gap-4">
      <Base.Label className="text-label text-ink-muted">{label}</Base.Label>
      {detail && <span className="text-caption text-ink-subtle num">{detail}</span>}
    </div>
    <Base.Track className="h-1.5 w-full overflow-hidden rounded-full bg-sunken">
      <Base.Indicator className="h-full bg-accent transition-all duration-180 ease-quiet" />
    </Base.Track>
  </Base.Root>
);

/**
 * Rückmeldung nach §8.5: Toast nur für abgeschlossene Aktionen, deren Ergebnis
 * man nicht ohnehin sieht. Vier Sekunden, unten rechts.
 *
 * `undo` setzt §8.2 um: Was rückgängig gemacht werden kann, wird ohne Rückfrage
 * ausgeführt und bleibt acht Sekunden lang zurückholbar.
 */
export const toast = {
  success: (message: string) => sonner.success(message, { duration: 4000 }),
  error: (message: string) => sonner.error(message, { duration: 6000 }),
  info: (message: string) => sonner(message, { duration: 4000 }),
  undo: (message: string, onUndo: () => void) =>
    sonner(message, {
      duration: 8000,
      action: { label: 'Rückgängig', onClick: onUndo },
    }),
};
