import React from 'react';
import { Check, ChevronsUpDown, LayoutGrid, Plus } from 'lucide-react';
import { CompanySettings, TenantConfig } from '../types';
import { GermanFlag } from './GermanFlag';
import { Menu, MenuGroup, MenuItem, MenuSeparator, cn } from './ui';

export interface TenantSwitcherProps {
  tenants: TenantConfig[];
  activeTenant: TenantConfig | null;
  settings: CompanySettings | null;
  onSwitchTenant: (tenantId: string) => void;
  onAddTenant: () => void;
  /** Zurück auf die Übersicht aller Mandanten. */
  onShowAll: () => void;
}

/**
 * Der Mandant wird dort gewechselt, wo sein Name steht.
 *
 * Bis hierher war der Weg der Startbildschirm: die Ansicht verlassen, in der
 * Übersicht den anderen Mandanten öffnen, zurück in die Buchhaltung. Ein
 * Mandantenwechsel ist aber keine Reise, sondern eine Auswahl — und eine
 * Auswahl gehört an die Stelle, an der der gewählte Wert steht.
 */
/**
 * Die Firma, unter der das Unternehmen auftritt.
 *
 * Bis zur Eintragung ins Handelsregister mit dem Zusatz „i. G." — der Zustand
 * kommt abgeleitet aus dem Backend und steht nirgends in der Datenbank. Die
 * Kopfzeile zeigt ihn, weil sie den ganzen Tag im Blick ist: wer den Zusatz
 * dort sieht, vergisst nicht, dass die Haftungsbeschränkung noch nicht greift.
 */
function firmNameOf(settings?: CompanySettings | null): string {
  const name = settings?.companyName?.trim();
  if (!name) return '';
  return settings?.inGruendung ? `${name} i. G.` : name;
}

export const TenantSwitcher: React.FC<TenantSwitcherProps> = ({
  tenants,
  activeTenant,
  settings,
  onSwitchTenant,
  onAddTenant,
  onShowAll,
}) => (
  <Menu
    align="start"
    className="min-w-[16rem] max-w-[20rem]"
    trigger={
      <button
        type="button"
        title="Mandant wechseln"
        aria-label={`Mandant ${firmNameOf(settings) || activeTenant?.name || 'Buchfink'} wechseln`}
        className="flex items-center gap-3 w-full min-w-0 p-1.5 -m-1.5 rounded-control text-left
                   transition-colors duration-120 ease-quiet hover:bg-shell-raised window-no-drag"
      >
        <span className="relative shrink-0">
          <img
            src="/buchfink-logo.svg"
            alt=""
            className="w-8 h-8 rounded-control bg-white/10 p-0.5 border border-white/10"
          />
          <span className="absolute -bottom-1 -right-1">
            <GermanFlag className="w-3.5 h-2.5 border border-shell" />
          </span>
        </span>
        <span className="min-w-0 flex-1">
          <span className="block text-body font-semibold text-white truncate">
            {firmNameOf(settings) || activeTenant?.name || 'Buchfink'}
          </span>
          <span className="block text-caption text-shell-text-muted truncate">
            Geschäftsjahr {settings?.fiscalYear || new Date().getFullYear()}
          </span>
        </span>
        <ChevronsUpDown className="w-3.5 h-3.5 shrink-0 text-shell-text-muted" strokeWidth={1.5} />
      </button>
    }
  >
    <MenuGroup label={`Mandanten · ${tenants.length}`}>
      {tenants.map((tenant) => {
        const isActive = tenant.id === activeTenant?.id;
        return (
          <MenuItem
            key={tenant.id}
            onClick={() => !isActive && onSwitchTenant(tenant.id)}
            className="h-auto py-2 items-start"
          >
            <span className="min-w-0 flex-1">
              <span className={cn('block truncate', isActive && 'font-semibold text-accent-text')}>
                {tenant.name}
              </span>
              <span className="block code-num text-caption text-ink-subtle truncate">
                {tenant.dataDir}
              </span>
            </span>
            {isActive && (
              <Check className="w-3.5 h-3.5 shrink-0 mt-0.5 text-accent-text" strokeWidth={1.5} />
            )}
          </MenuItem>
        );
      })}
    </MenuGroup>

    <MenuSeparator />
    <MenuItem onClick={onShowAll}>
      <LayoutGrid className="w-3.5 h-3.5 shrink-0 text-ink-faint" strokeWidth={1.5} />
      Alle Mandanten
    </MenuItem>
    <MenuItem onClick={onAddTenant} className="text-accent-text font-medium">
      <Plus className="w-3.5 h-3.5 shrink-0" strokeWidth={1.5} />
      Neuen Mandanten anlegen …
    </MenuItem>
  </Menu>
);
