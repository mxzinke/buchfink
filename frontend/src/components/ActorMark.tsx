import React from 'react';
import { Tooltip } from '@base-ui/react/tooltip';
import { Cog, Sparkles, User } from 'lucide-react';
import { POPUP, cn } from './ui';
import { actorKind, actorLabel } from '../utils/actor';

const ICON = { agent: Sparkles, system: Cog, person: User };

export interface ActorMarkProps {
  actor?: string;
  channel?: string;
  className?: string;
}

/**
 * Wer erfasst hat, als kleines Zeichen in der Zeile: Person, verbundene App
 * oder Programmlauf. Weg und Bearbeiter nennt der Tooltip. Ein Klick auf das
 * Zeichen öffnet nicht die Zeile, in der es steht.
 */
export const ActorMark: React.FC<ActorMarkProps> = ({ actor, channel, className }) => {
  const kind = actorKind(channel);
  const Icon = ICON[kind];
  const label = actorLabel(actor, channel);
  return (
    <Tooltip.Root>
      <Tooltip.Trigger
        delay={300}
        render={
          <span
            role="img"
            aria-label={label}
            tabIndex={0}
            onClick={(event) => event.stopPropagation()}
            className={cn(
              'inline-flex h-5 w-5 items-center justify-center rounded-control align-middle',
              'text-ink-subtle transition-colors duration-120 ease-quiet hover:text-ink-muted',
              'focus-visible:outline focus-visible:outline-2 focus-visible:outline-accent',
              className,
            )}
          />
        }
      >
        <Icon className="w-3.5 h-3.5" strokeWidth={1.5} aria-hidden="true" />
      </Tooltip.Trigger>
      <Tooltip.Portal>
        <Tooltip.Positioner sideOffset={6} className="z-50">
          <Tooltip.Popup className={cn(POPUP, 'px-2.5 py-1.5 text-caption text-ink-muted')}>
            {label}
          </Tooltip.Popup>
        </Tooltip.Positioner>
      </Tooltip.Portal>
    </Tooltip.Root>
  );
};
