import React from 'react';
import { Building2, Calendar, Check, ChevronDown, Lock, Menu as MenuIcon, Plus } from 'lucide-react';
import { TenantConfig } from '../types';
import { useWriteLock } from './WriteLock';
import { Button, Menu, MenuGroup, MenuItem, MenuSeparator, cn } from './ui';

interface HeaderProps {
  currentYear: number;
  availableYears: number[];
  /**
   * Geschäftsjahre ab der Feststellung. Sie nehmen keine Buchung mehr an, und
   * das steht neben der Jahreszahl, nicht erst in der Ablehnung (§11.5).
   */
  closedYears?: number[];
  onYearChange: (year: number) => void;
  /**
   * Legt das nächste Geschäftsjahr als Entität an und schaltet auf es um. Ohne
   * diesen Weg entstünde ein Jahr erst mit der ersten Buchung oder mit dem
   * Saldenvortrag — also nie, bevor man darin arbeiten will.
   */
  onCreateFiscalYear?: (year: number) => void;
  tenants?: TenantConfig[];
  activeTenant?: TenantConfig | null;
  onSwitchTenant?: (tenantId: string) => void;
  onOpenNewTenantModal?: () => void;
  onToggleMobileSidebar?: () => void;
}

export const Header: React.FC<HeaderProps> = ({
  currentYear,
  availableYears,
  closedYears = [],
  onYearChange,
  onCreateFiscalYear,
  tenants = [],
  activeTenant,
  onSwitchTenant,
  onOpenNewTenantModal,
  onToggleMobileSidebar,
}) => {
  const currentCalendarYear = new Date().getFullYear();
  const nextYear = availableYears.length > 0 ? Math.max(...availableYears) + 1 : currentCalendarYear;
  // Ein Geschäftsjahr anzulegen ist eine Änderung an den Büchern und im
  // Prüfermodus gesperrt; das Umschalten der Ansicht bleibt möglich.
  const writeLock = useWriteLock();

  return (
    <header
      className="sticky top-0 z-30 h-14 shrink-0 flex items-center justify-between gap-2
                 px-4 sm:px-8 border-b border-line bg-paper/90 backdrop-blur-md
                 select-none window-drag"
    >
      <div className="flex items-center gap-2 sm:gap-3 min-w-0 window-no-drag">
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

        {/* Geschäftsjahr: eine Auswahl, deshalb Himmelblau und nicht Tinte (§3.3). */}
        <div className="flex items-center gap-1 p-0.5 rounded-control border border-line bg-surface
                        overflow-x-auto max-w-[220px] sm:max-w-none">
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
                title={
                  isClosed
                    ? `Geschäftsjahr ${year} ist abgeschlossen. Buchungen sind nur im laufenden Jahr möglich.`
                    : `Ansicht auf Geschäftsjahr ${year} filtern`
                }
                className={cn(
                  'flex items-center gap-1.5 h-7 px-2.5 shrink-0 rounded-[4px] text-label num',
                  'transition-colors duration-120 ease-quiet',
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
                    aria-label="abgeschlossen"
                  />
                )}
                {year === currentCalendarYear && !isSelected && (
                  <span
                    className="mark-diamond bg-ink-faint scale-75"
                    title="Aktuelles Kalenderjahr"
                  />
                )}
              </button>
            );
          })}
          {onCreateFiscalYear && (
            <button
              type="button"
              onClick={() => onCreateFiscalYear(nextYear)}
              disabled={writeLock.locked}
              title={writeLock.hint ?? `Geschäftsjahr ${nextYear} anlegen`}
              aria-label={`Geschäftsjahr ${nextYear} anlegen`}
              className={cn(
                'flex items-center h-7 px-2 shrink-0 rounded-[4px] text-ink-faint',
                'transition-colors duration-120 ease-quiet hover:bg-sunken hover:text-ink',
                'disabled:cursor-not-allowed disabled:hover:bg-transparent disabled:hover:text-ink-faint',
              )}
            >
              <Plus className="w-3.5 h-3.5" strokeWidth={1.5} />
            </button>
          )}
        </div>
      </div>

      <div className="shrink-0 window-no-drag">
        <Menu
          trigger={
            <Button
              variant="secondary"
              size="sm"
              title="Mandanten wechseln"
              icon={<Building2 className="w-3.5 h-3.5 text-ink-faint" strokeWidth={1.5} />}
            >
              <span className="max-w-[100px] sm:max-w-[160px] truncate text-ink">
                {activeTenant?.name || 'Mandant'}
              </span>
              <ChevronDown className="w-3 h-3 text-ink-faint" strokeWidth={1.5} />
            </Button>
          }
        >
          <MenuGroup label={`Mandanten (${tenants.length})`}>
            {tenants.map((tenant) => {
              const isActive = tenant.id === activeTenant?.id;
              return (
                <MenuItem
                  key={tenant.id}
                  onClick={() => onSwitchTenant?.(tenant.id)}
                  className="h-auto py-2 items-start"
                >
                  <span className="min-w-0 flex-1">
                    <span className={cn('block truncate', isActive && 'font-semibold text-accent-text')}>
                      {tenant.name}
                    </span>
                    <span className="block text-caption text-ink-subtle truncate">
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

          {onOpenNewTenantModal && (
            <>
              <MenuSeparator />
              <MenuItem onClick={onOpenNewTenantModal} className="text-accent-text font-medium">
                <Plus className="w-3.5 h-3.5 shrink-0" strokeWidth={1.5} />
                Neuen Mandanten anlegen …
              </MenuItem>
            </>
          )}
        </Menu>
      </div>
    </header>
  );
};
