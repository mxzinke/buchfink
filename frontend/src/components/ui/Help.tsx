import React, { useState, useId } from 'react';
import { Popover } from '@base-ui/react/popover';
import { Tooltip } from '@base-ui/react/tooltip';
import { cn } from './cn';
import { POPUP } from './popup';
import { Dialog } from './Dialog';

const MARK =
  'inline-flex items-center justify-center w-6 h-6 -my-1 shrink-0 align-middle ' +
  'text-caption font-semibold leading-none text-ink-subtle ' +
  'transition-colors duration-120 ease-quiet hover:text-ink-muted focus-visible:outline focus-visible:outline-2 focus-visible:outline-accent';

export interface HelpProps {
  /** Kurze Erklärung ohne Fachjargon, Quellen oder Bedienelemente. */
  summary: string;
  /** Ausführliche Erklärung für den Dialog. Normen werden dort verlinkt. */
  children?: React.ReactNode;
  label: string;
  /** Öffnet einen eigenen Detaildialog, etwa mit einer Berechnungstabelle. */
  onMore?: () => void;
  className?: string;
}

/** Das Fragezeichen erklärt kurz; der getrennte Button öffnet die Details. */
export const Help: React.FC<HelpProps> = ({ summary, children, label, onMore, className }) => {
  const tooltipId = useId();
  const [tooltipOpen, setTooltipOpen] = useState(false);
  const [detailsOpen, setDetailsOpen] = useState(false);
  const hasDetails = Boolean(children || onMore);

  return (
    <span className={cn('inline-flex items-center shrink-0 align-middle', className)}>
      <Tooltip.Root open={tooltipOpen} onOpenChange={setTooltipOpen}>
        <Tooltip.Trigger
          aria-label={label}
          aria-describedby={tooltipOpen ? tooltipId : undefined}
          className={MARK}
          delay={300}
          closeDelay={200}
          closeOnClick={false}
          onFocus={() => setTooltipOpen(true)}
          onBlur={() => setTooltipOpen(false)}
          onClick={() => setTooltipOpen(true)}
        >
          ?
        </Tooltip.Trigger>
        <Tooltip.Portal>
          <Tooltip.Positioner sideOffset={6} className="z-50">
            <Tooltip.Popup id={tooltipId} role="tooltip" className={cn(POPUP, 'px-3 py-2 max-w-[300px] text-left text-body font-normal text-ink-muted')}>
              {summary}
            </Tooltip.Popup>
          </Tooltip.Positioner>
        </Tooltip.Portal>
      </Tooltip.Root>
      {hasDetails && (
        <button
          type="button"
          className="ml-1 text-caption font-normal text-ink-subtle underline underline-offset-2 hover:text-ink focus-visible:outline focus-visible:outline-2 focus-visible:outline-accent"
          aria-label={`Mehr erfahren: ${label}`}
          aria-haspopup="dialog"
          onClick={() => {
            setTooltipOpen(false);
            if (onMore) onMore();
            else setDetailsOpen(true);
          }}
        >
          Mehr erfahren
        </button>
      )}
      {!onMore && hasDetails && (
        <Dialog open={detailsOpen} onOpenChange={setDetailsOpen} title={label}>
          <div className="text-body font-normal text-ink-muted space-y-3">
            <p>{summary}</p>
            <div>{children}</div>
          </div>
        </Dialog>
      )}
    </span>
  );
};

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
