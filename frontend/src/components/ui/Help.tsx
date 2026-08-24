import React from 'react';
import { Tooltip } from '@base-ui/react/tooltip';
import { Popover } from '@base-ui/react/popover';
import { cn } from './cn';
import { POPUP, TOOLTIP_POPUP } from './popup';

/**
 * Die drei Stufen der Erklärung aus §15.2. Ausgelöst wird immer bewusst, nie
 * automatisch: keine Tour, kein Popover beim ersten Besuch.
 *
 * Das Erklärzeichen ist ein Fragezeichen mit 24 px Klickfeld. Es steht hinter
 * der Beschriftung, nie davor.
 */
const MARK =
  'inline-flex items-center justify-center w-6 h-6 shrink-0 align-middle ' +
  'text-caption font-semibold leading-none text-ink-faint ' +
  'transition-colors duration-120 ease-quiet hover:text-ink-muted data-[popup-open]:text-ink-muted';

/** Verzögerung, damit der Zeiger beim Überstreichen nichts auslöst. */
const HOVER_DELAY_MS = 400;

export interface HelpTooltipProps {
  /** Ein Satz. Was länger ist, gehört ins Popover. */
  content: string;
  /** Beschriftung für Screenreader, etwa "Erklärung zu Einnahmen". */
  label: string;
  className?: string;
}

export const HelpTooltip: React.FC<HelpTooltipProps> = ({ content, label, className }) => (
  <Tooltip.Provider delay={HOVER_DELAY_MS}>
    <Tooltip.Root>
      <Tooltip.Trigger aria-label={label} className={cn(MARK, className)}>
        ?
      </Tooltip.Trigger>
      <Tooltip.Portal>
        <Tooltip.Positioner sideOffset={6} className="z-50">
          <Tooltip.Popup className={TOOLTIP_POPUP}>{content}</Tooltip.Popup>
        </Tooltip.Positioner>
      </Tooltip.Portal>
    </Tooltip.Root>
  </Tooltip.Provider>
);

export interface HelpPopoverProps {
  /** Bis drei Sätze. */
  children: React.ReactNode;
  label: string;
  /** Sprung in die dritte Stufe. Die Beschriftung ist immer "Mehr dazu". */
  onMore?: () => void;
  className?: string;
}

export const HelpPopover: React.FC<HelpPopoverProps> = ({
  children,
  label,
  onMore,
  className,
}) => (
  <Popover.Root>
    <Popover.Trigger aria-label={label} className={cn(MARK, className)}>
      ?
    </Popover.Trigger>
    <Popover.Portal>
      <Popover.Positioner sideOffset={6} className="z-50">
        <Popover.Popup className={cn(POPUP, 'p-4 max-w-[340px] text-left')}>
          <Popover.Description className="text-body text-ink-muted">
            {children}
          </Popover.Description>
          {onMore && (
            <Popover.Close
              onClick={onMore}
              className="mt-3 text-label font-semibold text-accent-text hover:text-accent transition-colors duration-120 ease-quiet"
            >
              Mehr dazu
            </Popover.Close>
          )}
        </Popover.Popup>
      </Popover.Positioner>
    </Popover.Portal>
  </Popover.Root>
);

export interface InfoPopoverProps {
  trigger: React.ReactNode;
  children: React.ReactNode;
  className?: string;
}

/** Popover an einem eigenen Auslöser, etwa einem Button in einer Tabellenzeile. */
export const InfoPopover: React.FC<InfoPopoverProps> = ({ trigger, children, className }) => (
  <Popover.Root>
    <Popover.Trigger render={trigger as React.ReactElement} />
    <Popover.Portal>
      <Popover.Positioner sideOffset={6} className="z-50">
        <Popover.Popup className={cn(POPUP, 'p-4 max-w-[340px]', className)}>
          {children}
        </Popover.Popup>
      </Popover.Positioner>
    </Popover.Portal>
  </Popover.Root>
);
