import React from 'react';
import { Collapsible } from '@base-ui/react/collapsible';
import { ChevronRight } from 'lucide-react';
import { cn } from './cn';

export interface DisclosureProps {
  /** Anker, über den ein Ziel den Abschnitt aufklappt und anspringt. */
  id?: string;
  title: string;
  /** Eine Zeile, die auch eingeklappt sichtbar bleibt: Anzahl, Stand, Betrag. */
  context?: string;
  /** Rechts im Kopf, etwa ein StatusBadge oder ein Betrag. */
  meta?: React.ReactNode;
  open?: boolean;
  defaultOpen?: boolean;
  onOpenChange?: (open: boolean) => void;
  /**
   * `section`: ein eingeklappter Abschnitt der Seite, mit Haarlinie darüber.
   * `row`: eine Zeile einer Liste, deren Einzelheiten darunter aufgehen.
   */
  variant?: 'section' | 'row';
  className?: string;
  children: React.ReactNode;
}

const PANEL =
  'h-[var(--collapsible-panel-height)] overflow-hidden transition-[height] duration-180 ease-quiet ' +
  'data-[starting-style]:h-0 data-[ending-style]:h-0';

/**
 * Aufklappen statt Wegnavigieren: der Kopf zeigt das Nötige, der Inhalt geht
 * erst auf Klick auf. Der Inhalt wird erst beim Aufklappen gerendert, eine
 * eingeklappte Ansicht lädt also nichts.
 *
 * Bewegung nur über die Höhe, 180 ms (§7).
 */
export const Disclosure: React.FC<DisclosureProps> = ({
  id,
  title,
  context,
  meta,
  open,
  defaultOpen,
  onOpenChange,
  variant = 'section',
  className,
  children,
}) => {
  const section = variant === 'section';
  return (
    <Collapsible.Root
      open={open}
      defaultOpen={defaultOpen}
      onOpenChange={(next) => onOpenChange?.(next)}
      className={cn(section ? 'mt-8 pt-6 border-t border-line' : 'border-b border-line', className)}
      render={<section id={id} />}
    >
      <Collapsible.Trigger
        className={cn(
          'group w-full flex items-start gap-3 text-left rounded-control',
          'transition-colors duration-120 ease-quiet',
          section ? 'py-2' : 'px-2 py-3 hover:bg-sunken',
          'focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent',
        )}
      >
        <ChevronRight
          className={cn(
            'shrink-0 text-ink-faint transition-transform duration-180 ease-quiet',
            'group-data-[panel-open]:rotate-90',
            section ? 'w-4 h-4 mt-0.5' : 'w-3.5 h-3.5 mt-[3px]',
          )}
          strokeWidth={1.5}
          aria-hidden="true"
        />
        <span className="min-w-0 flex-1">
          <span className={cn('block text-ink', section ? 'text-heading' : 'text-body font-medium')}>
            {title}
          </span>
          {context && <span className="block text-caption text-ink-subtle mt-0.5">{context}</span>}
        </span>
        {meta && <span className="shrink-0">{meta}</span>}
      </Collapsible.Trigger>
      <Collapsible.Panel className={PANEL}>
        <div className={section ? 'pt-4' : 'px-2 pb-4 pl-[30px]'}>{children}</div>
      </Collapsible.Panel>
    </Collapsible.Root>
  );
};
