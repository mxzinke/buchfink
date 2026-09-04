import React from 'react';
import { Lock } from 'lucide-react';
import { cn } from './cn';

/**
 * Das Status-Vokabular aus §11.3, abschließend. Ein Wort pro Zustand, in der
 * ganzen Anwendung dasselbe. Synonyme wie erledigt oder fertig gibt es nicht,
 * und weil die Beschriftung hier im Typ steckt, kann auch keins entstehen.
 */
export type Status =
  | 'entwurf'
  | 'offen'
  | 'teilweiseAusgeglichen'
  | 'zugeordnet'
  | 'gebucht'
  | 'ausgeglichen'
  | 'festgeschrieben'
  | 'ueberfaellig'
  | 'storniert'
  | 'fehlerhaft'
  // Die Stände des Jahresabschlusses. Sie beschreiben nicht dieselbe Sache wie
  // „Festgeschrieben": festgeschrieben ist der Zeitraum, festgestellt ist der
  // Abschluss, und beschlossen haben ihn die Gesellschafter (§ 42a Abs. 2
  // GmbHG). Ein gemeinsames Wort für beides würde den Beschluss unterschlagen.
  | 'aufgestellt'
  | 'festgestellt'
  | 'offengelegt';

type Tone = 'neutral' | 'attention' | 'attentionOutline' | 'positive' | 'negative';

const TONE: Record<Tone, { badge: string; mark: string }> = {
  neutral: { badge: 'bg-transparent text-ink-subtle border-line-strong', mark: 'bg-ink-faint' },
  attention: {
    badge: 'bg-attention-soft text-attention-text border-attention-line',
    mark: 'bg-attention',
  },
  attentionOutline: {
    badge: 'bg-transparent text-attention-text border-attention-line',
    mark: 'bg-attention',
  },
  positive: {
    badge: 'bg-positive-soft text-positive-text border-positive-line',
    mark: 'bg-positive',
  },
  negative: {
    badge: 'bg-negative-soft text-negative-text border-negative-line',
    mark: 'bg-negative',
  },
};

const STATUS: Record<Status, { label: string; tone: Tone; lock?: boolean }> = {
  entwurf: { label: 'Entwurf', tone: 'neutral' },
  offen: { label: 'Offen', tone: 'attention' },
  teilweiseAusgeglichen: { label: 'Teilweise ausgeglichen', tone: 'attentionOutline' },
  zugeordnet: { label: 'Zugeordnet', tone: 'neutral' },
  gebucht: { label: 'Gebucht', tone: 'positive' },
  ausgeglichen: { label: 'Ausgeglichen', tone: 'positive' },
  festgeschrieben: { label: 'Festgeschrieben', tone: 'positive', lock: true },
  ueberfaellig: { label: 'Überfällig', tone: 'negative' },
  storniert: { label: 'Storniert', tone: 'negative' },
  fehlerhaft: { label: 'Fehlerhaft', tone: 'negative' },
  aufgestellt: { label: 'Aufgestellt', tone: 'neutral' },
  // Ab der Feststellung nimmt das Geschäftsjahr keine Buchung mehr auf; das
  // Schloss steht für dieselbe Sperre wie neben der Jahreszahl (§11.5).
  festgestellt: { label: 'Festgestellt', tone: 'positive', lock: true },
  offengelegt: { label: 'Offengelegt', tone: 'positive', lock: true },
};

export interface StatusBadgeProps {
  status: Status;
  className?: string;
}

/** Der Marker ist eine Raute, kein Punkt. Vier definierte Kanten, eine Achse. */
export const StatusBadge: React.FC<StatusBadgeProps> = ({ status, className }) => {
  const { label, tone, lock } = STATUS[status];
  const style = TONE[tone];

  return (
    <span
      className={cn(
        'inline-flex items-center gap-1.5 h-5 px-2 rounded-control border',
        'text-caption font-medium whitespace-nowrap',
        style.badge,
        className,
      )}
    >
      {lock ? (
        <Lock className="w-3 h-3 shrink-0" strokeWidth={1.5} aria-hidden="true" />
      ) : (
        <span className={cn('mark-diamond', style.mark)} aria-hidden="true" />
      )}
      {label}
    </span>
  );
};
