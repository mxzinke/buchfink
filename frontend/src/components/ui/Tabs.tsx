import React from 'react';
import { Tabs as Base } from '@base-ui/react/tabs';
import { Separator as BaseSeparator } from '@base-ui/react/separator';
import { cn } from './cn';

export interface TabItem<T> {
  value: T;
  label: string;
  /** Zähler rechts der Beschriftung, etwa die Anzahl offener Posten. */
  count?: number;
  disabled?: boolean;
}

export interface TabsProps<T> {
  items: TabItem<T>[];
  value?: T;
  defaultValue?: T;
  onValueChange?: (value: T) => void;
  className?: string;
  children?: React.ReactNode;
}

/**
 * Wechsel zwischen gleichrangigen Sichten auf denselben Datenbestand, etwa
 * Bilanz und GuV. Kein Ersatz für Navigation: Was in der Seitenspalte steht,
 * wird nicht zusätzlich zum Reiter.
 *
 * Der aktive Reiter trägt eine Leiste in Himmelblau, dieselbe Markierung wie
 * der aktive Eintrag in der Navigation (§12).
 */
export function Tabs<T extends string | number>({
  items,
  value,
  defaultValue,
  onValueChange,
  className,
  children,
}: TabsProps<T>) {
  return (
    <Base.Root
      value={value}
      defaultValue={defaultValue}
      onValueChange={(next) => onValueChange?.(next as T)}
      className={className}
    >
      <Base.List className="relative flex items-center gap-6 border-b border-line">
        {items.map((item) => (
          <Base.Tab
            key={String(item.value)}
            value={item.value}
            disabled={item.disabled}
            className={cn(
              'relative -mb-px flex items-center gap-2 pb-2.5 pt-1 text-label',
              'text-ink-subtle transition-colors duration-120 ease-quiet',
              'hover:text-ink data-[selected]:text-ink data-[selected]:font-semibold',
              'data-[disabled]:text-ink-faint data-[disabled]:cursor-not-allowed',
            )}
          >
            {item.label}
            {item.count !== undefined && (
              <span className="rounded-full bg-sunken px-1.5 text-caption text-ink-subtle num">
                {item.count}
              </span>
            )}
          </Base.Tab>
        ))}
        <Base.Indicator
          className="absolute bottom-0 left-0 h-0.5 bg-accent transition-all duration-180 ease-quiet"
          style={{
            width: 'var(--active-tab-width)',
            transform: 'translateX(var(--active-tab-left))',
          }}
        />
      </Base.List>
      {children}
    </Base.Root>
  );
}

export const TabPanel: React.FC<React.ComponentProps<typeof Base.Panel>> = ({
  className,
  ...props
}) => <Base.Panel className={cn('pt-6 outline-none', className)} {...props} />;

export interface SeparatorProps {
  orientation?: 'horizontal' | 'vertical';
  className?: string;
}

/** Haarlinie. Das leiseste Trennmittel nach dem Weißraum (§6.4). */
export const Separator: React.FC<SeparatorProps> = ({ orientation = 'horizontal', className }) => (
  <BaseSeparator
    orientation={orientation}
    className={cn(
      'bg-line shrink-0',
      orientation === 'horizontal' ? 'h-px w-full' : 'w-px self-stretch',
      className,
    )}
  />
);
