import { todayISO } from "../utils/calendar";
import React, { useState } from 'react';
import { Calendar, Check, ChevronDown, Lock, Menu as MenuIcon, Plus } from 'lucide-react';
import { CompanySettings, TenantConfig } from '../types';
import { useWriteLock } from './WriteLock';
import { FiscalYearDialog } from './FiscalYearDialog';
import { Button, Menu, MenuGroup, MenuItem, MenuSeparator, cn } from './ui';

interface HeaderProps {
  settings: CompanySettings | null;
  tenants: TenantConfig[];
  activeTenant: TenantConfig | null;
  onSwitchTenant: (tenantId: string) => void;
  onManageTenants: () => void;
  onAddTenant: () => void;
  currentYear: number;
  availableYears: number[];
  /**
   * Geschäftsjahre ab der Feststellung. Sie nehmen keine Buchung mehr an, und
   * das steht neben der Jahreszahl, nicht erst in der Ablehnung (§11.5).
   */
  closedYears?: number[];
  onYearChange: (year: number) => void;
  /** Legt ein Geschäftsjahr an und schaltet auf es um. */
  onCreateFiscalYear?: (year: number) => void;
  onToggleMobileSidebar?: () => void;
}

/**
 * Die Firma, unter der das Unternehmen auftritt. Bis zur Eintragung mit dem
 * Zusatz „i. G."; der Zustand kommt abgeleitet aus dem Backend.
 */
function firmNameOf(settings: CompanySettings | null, tenant: TenantConfig | null): string {
  const name = settings?.companyName?.trim();
  if (!name) return tenant?.name ?? 'Buchfink';
  return settings?.inGruendung ? `${name} i. G.` : name;
}

/**
 * Die Mandantenwahl steht an der Firma. Mit einem Mandanten gibt es nichts zu
 * wählen; dann steht dort nur der Name, und weitere Mandanten legt man in den
 * Einstellungen an.
 */
const TenantMenu: React.FC<{
  name: string;
  tenants: TenantConfig[];
  activeTenant: TenantConfig | null;
  onSwitchTenant: (tenantId: string) => void;
  onManageTenants: () => void;
  onAddTenant: () => void;
}> = ({ name, tenants, activeTenant, onSwitchTenant, onManageTenants, onAddTenant }) => (
  <Menu
    align="start"
    className="min-w-[16rem] max-w-[20rem]"
    trigger={
      <button
        type="button"
        title="Mandant wechseln"
        aria-label={`Mandant ${name} wechseln`}
        className="flex items-center gap-1.5 min-w-0 h-8 px-2 -mx-2 rounded-control text-left
                   transition-colors duration-120 ease-quiet hover:bg-sunken"
      >
        <span className="text-body font-semibold text-ink truncate">{name}</span>
        <ChevronDown className="w-3.5 h-3.5 shrink-0 text-ink-faint" strokeWidth={1.5} />
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
    <MenuItem onClick={onManageTenants}>Mandanten verwalten</MenuItem>
    <MenuItem onClick={onAddTenant} className="text-accent-text font-medium">
      <Plus className="w-3.5 h-3.5 shrink-0" strokeWidth={1.5} />
      Neuen Mandanten anlegen …
    </MenuItem>
  </Menu>
);

export const Header: React.FC<HeaderProps> = ({
  settings,
  tenants,
  activeTenant,
  onSwitchTenant,
  onManageTenants,
  onAddTenant,
  currentYear,
  availableYears,
  closedYears = [],
  onYearChange,
  onCreateFiscalYear,
  onToggleMobileSidebar,
}) => {
  const currentCalendarYear = Number(todayISO().slice(0, 4));
  // Ein Geschäftsjahr anzulegen ist eine Änderung an den Büchern und im
  // Prüfermodus gesperrt; das Umschalten der Ansicht bleibt möglich.
  const writeLock = useWriteLock();
  const [creating, setCreating] = useState(false);
  const name = firmNameOf(settings, activeTenant);

  return (
    <header
      className="sticky top-0 z-30 h-14 shrink-0 flex items-center justify-between gap-4
                 px-4 sm:px-8 border-b border-line bg-paper/90 backdrop-blur-md
                 select-none window-drag"
    >
      <div className="flex items-center gap-2 min-w-0 window-no-drag">
        {onToggleMobileSidebar && (
          <Button
            variant="quiet"
            size="sm"
            iconOnly
            onClick={onToggleMobileSidebar}
            title="Navigation öffnen"
            aria-label="Navigation öffnen"
            className="md:hidden"
          >
            <MenuIcon className="w-5 h-5" strokeWidth={1.5} />
          </Button>
        )}
        {tenants.length > 1 ? (
          <TenantMenu
            name={name}
            tenants={tenants}
            activeTenant={activeTenant}
            onSwitchTenant={onSwitchTenant}
            onManageTenants={onManageTenants}
            onAddTenant={onAddTenant}
          />
        ) : (
          <span className="text-body font-semibold text-ink truncate">{name}</span>
        )}
      </div>

      {/* Geschäftsjahr: eine Auswahl, deshalb Himmelblau und nicht Tinte (§3.3). */}
      <div
        className="flex items-center gap-1 p-0.5 rounded-control border border-line bg-surface
                   overflow-x-auto max-w-[220px] sm:max-w-none window-no-drag"
      >
        <Calendar className="w-3.5 h-3.5 shrink-0 ml-1.5 mr-0.5 text-ink-faint" strokeWidth={1.5} />
        {availableYears.map((year) => {
          const isSelected = currentYear === year;
          const isClosed = closedYears.includes(year);
          return (
            <button
              key={year}
              type="button"
              onClick={() => onYearChange(year)}
              aria-pressed={isSelected}
              aria-label={[
                `Geschäftsjahr ${year}`,
                isClosed ? 'abgeschlossen' : '',
                year === currentCalendarYear ? 'aktuelles Kalenderjahr' : '',
              ]
                .filter(Boolean)
                .join(', ')}
              title={
                isClosed
                  ? `Geschäftsjahr ${year} ist abgeschlossen. Buchungen sind nur im laufenden Jahr möglich.`
                  : `Ansicht auf Geschäftsjahr ${year} filtern`
              }
              className={cn(
                'flex items-center gap-1.5 h-7 px-2.5 shrink-0 rounded-[4px] text-label num',
                'transition-colors duration-120 ease-quiet',
                // Die Leiste scrollt seitlich und schneidet einen äußeren Umriss ab.
                'focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-accent',
                isSelected
                  ? 'bg-accent-soft text-accent-text font-semibold'
                  : 'text-ink-subtle hover:bg-sunken hover:text-ink',
              )}
            >
              {year}
              {isClosed && (
                <Lock
                  className="w-3 h-3 shrink-0 text-ink-faint"
                  strokeWidth={1.5}
                  aria-hidden="true"
                />
              )}
              {year === currentCalendarYear && !isSelected && (
                <span
                  className="mark-diamond bg-ink-faint scale-75"
                  title="Aktuelles Kalenderjahr"
                  aria-hidden="true"
                />
              )}
            </button>
          );
        })}
        {onCreateFiscalYear && (
          <button
            type="button"
            onClick={() => setCreating(true)}
            disabled={writeLock.locked}
            title={writeLock.hint ?? 'Geschäftsjahr anlegen'}
            aria-label="Geschäftsjahr anlegen"
            className={cn(
              'flex items-center h-7 px-2 shrink-0 rounded-[4px] text-ink-faint',
              'focus-visible:outline focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-accent',
              'transition-colors duration-120 ease-quiet hover:bg-sunken hover:text-ink',
              'disabled:cursor-not-allowed disabled:hover:bg-transparent disabled:hover:text-ink-faint',
            )}
          >
            <Plus className="w-3.5 h-3.5" strokeWidth={1.5} />
          </button>
        )}
      </div>

      {onCreateFiscalYear && (
        <FiscalYearDialog open={creating} onOpenChange={setCreating} onCreate={onCreateFiscalYear} />
      )}
    </header>
  );
};
