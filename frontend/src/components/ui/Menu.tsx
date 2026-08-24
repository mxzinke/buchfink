import React from 'react';
import { Menu as Base } from '@base-ui/react/menu';
import { Check } from 'lucide-react';
import { cn } from './cn';
import { POPUP, POPUP_ITEM } from './popup';

export interface MenuProps {
  /** Der auslösende Button. Muss ein einzelnes Element sein. */
  trigger: React.ReactElement;
  children: React.ReactNode;
  align?: 'start' | 'center' | 'end';
  className?: string;
}

/**
 * Aufklappmenü für Aktionen und Wechsel, etwa den Mandanten in der Kopfzeile.
 * Tastaturführung, Typeahead und das Schließen bei Klick daneben kommen von
 * Base UI.
 */
export const Menu: React.FC<MenuProps> = ({ trigger, children, align = 'end', className }) => (
  <Base.Root>
    <Base.Trigger render={trigger} />
    <Base.Portal>
      <Base.Positioner sideOffset={4} align={align} className="z-50">
        <Base.Popup className={cn(POPUP, 'py-1 min-w-56 max-h-80 overflow-auto', className)}>
          {children}
        </Base.Popup>
      </Base.Positioner>
    </Base.Portal>
  </Base.Root>
);

export const MenuItem: React.FC<React.ComponentProps<typeof Base.Item>> = ({
  className,
  ...props
}) => <Base.Item className={cn(POPUP_ITEM, className)} {...props} />;

export interface MenuGroupProps {
  /** Gruppenlabel, sparsam einsetzen. Ein Menü ohne Gruppen braucht keins. */
  label?: string;
  children: React.ReactNode;
  className?: string;
}

/**
 * Die Beschriftung gehört zwingend in eine Gruppe, sonst findet Base UI keinen
 * Kontext und wirft. Deshalb gibt es beides nur zusammen.
 */
export const MenuGroup: React.FC<MenuGroupProps> = ({ label, children, className }) => (
  <Base.Group className={className}>
    {label && (
      <Base.GroupLabel className="px-3 pt-2 pb-1 text-overline text-ink-subtle">
        {label}
      </Base.GroupLabel>
    )}
    {children}
  </Base.Group>
);

export const MenuSeparator: React.FC<{ className?: string }> = ({ className }) => (
  <div className={cn('my-1 h-px bg-line', className)} />
);

export interface MenuCheckItemProps
  extends React.ComponentProps<typeof Base.CheckboxItem> {
  children: React.ReactNode;
}

export const MenuCheckItem: React.FC<MenuCheckItemProps> = ({ className, children, ...props }) => (
  <Base.CheckboxItem className={cn(POPUP_ITEM, className)} {...props}>
    <Base.CheckboxItemIndicator className="shrink-0 text-accent-text data-[unchecked]:invisible">
      <Check className="w-3.5 h-3.5" strokeWidth={1.5} />
    </Base.CheckboxItemIndicator>
    {children}
  </Base.CheckboxItem>
);
