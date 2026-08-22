// SPDX-License-Identifier: EUPL-1.2

import React, { useState } from 'react';
import { Calendar, Building2, ChevronDown, Plus, Check, Menu } from 'lucide-react';
import { TenantConfig } from '../types';

interface HeaderProps {
  currentYear: number;
  availableYears: number[];
  onYearChange: (year: number) => void;
  tenants?: TenantConfig[];
  activeTenant?: TenantConfig | null;
  onSwitchTenant?: (tenantId: string) => void;
  onOpenNewTenantModal?: () => void;
  onToggleMobileSidebar?: () => void;
}

export const Header: React.FC<HeaderProps> = ({
  currentYear,
  availableYears,
  onYearChange,
  tenants = [],
  activeTenant,
  onSwitchTenant,
  onOpenNewTenantModal,
  onToggleMobileSidebar,
}) => {
  const [isTenantDropdownOpen, setIsTenantDropdownOpen] = useState(false);
  const currentCalendarYear = new Date().getFullYear();

  return (
    <header className="h-14 border-b border-stone-200/80 bg-[#FAF8F5]/90 backdrop-blur-md px-3 sm:px-6 flex items-center justify-between select-none sticky top-0 z-30 window-drag gap-2">
      {/* Left: Mobile Menu Trigger + Fiscal Year Filter */}
      <div className="flex items-center gap-2 sm:gap-3 window-no-drag min-w-0">
        {onToggleMobileSidebar && (
          <button
            onClick={onToggleMobileSidebar}
            className="md:hidden p-1.5 text-stone-600 hover:text-stone-900 rounded-lg hover:bg-stone-200/60 transition-colors shrink-0"
            title="Navigation öffnen"
          >
            <Menu className="w-5 h-5" />
          </button>
        )}

        <span className="text-xs font-semibold text-stone-500 uppercase tracking-wider hidden lg:inline whitespace-nowrap">
          Filter Geschäftsjahr:
        </span>

        <div className="flex items-center gap-1 bg-stone-200/50 p-1 rounded-lg border border-stone-200 text-xs overflow-x-auto max-w-[200px] sm:max-w-none">
          <Calendar className="w-3.5 h-3.5 text-stone-500 ml-1 mr-0.5 shrink-0" />
          {availableYears.map((year) => {
            const isSelected = currentYear === year;
            const isCurrentYear = year === currentCalendarYear;

            return (
              <button
                key={year}
                onClick={() => onYearChange(year)}
                className={`px-2 sm:px-2.5 py-0.5 rounded-md font-medium transition-all flex items-center gap-1 text-[11px] sm:text-xs shrink-0 cursor-pointer ${
                  isSelected
                    ? 'bg-amber-700 text-white shadow-2xs font-semibold'
                    : 'text-stone-600 hover:text-stone-900 hover:bg-stone-200/60'
                }`}
                title={`Ansicht auf Geschäftsjahr ${year} filtern`}
              >
                <span>{year}</span>
                {isCurrentYear && !isSelected && (
                  <span className="w-1.5 h-1.5 rounded-full bg-emerald-500" title="Aktuelles Kalenderjahr" />
                )}
              </button>
            );
          })}
        </div>
      </div>

      {/* Right: Tenant Quick Switcher Dropdown */}
      <div className="relative window-no-drag shrink-0">
        <button
          onClick={() => setIsTenantDropdownOpen(!isTenantDropdownOpen)}
          className="flex items-center gap-1.5 sm:gap-2 px-2.5 sm:px-3 py-1.5 rounded-lg bg-stone-100 hover:bg-stone-200/80 border border-stone-200/80 text-xs text-stone-700 font-medium transition-colors cursor-pointer"
          title="Mandanten wechseln"
        >
          <Building2 className="w-3.5 h-3.5 text-amber-700 shrink-0" />
          <span className="font-semibold text-stone-900 max-w-[100px] sm:max-w-[160px] truncate">
            {activeTenant?.name || 'Mandant'}
          </span>
          <ChevronDown className="w-3 h-3 text-stone-400 shrink-0" />
        </button>

        {isTenantDropdownOpen && (
          <>
            <div
              className="fixed inset-0 z-40"
              onClick={() => setIsTenantDropdownOpen(false)}
            />
            <div className="absolute right-0 mt-1.5 w-64 bg-white rounded-xl shadow-xl border border-stone-200 py-1.5 z-50 text-xs animate-in fade-in">
              <div className="px-3 py-1.5 text-[10px] font-bold uppercase tracking-wider text-stone-400 border-b border-stone-100">
                Mandanten ({tenants.length})
              </div>

              <div className="max-h-56 overflow-y-auto py-1">
                {tenants.map((t) => {
                  const isActive = t.id === activeTenant?.id;
                  return (
                    <button
                      key={t.id}
                      onClick={() => {
                        onSwitchTenant?.(t.id);
                        setIsTenantDropdownOpen(false);
                      }}
                      className={`w-full px-3 py-2 text-left flex items-center justify-between hover:bg-amber-50/50 transition-colors cursor-pointer ${
                        isActive ? 'bg-amber-50/80 font-semibold text-amber-900' : 'text-stone-700'
                      }`}
                    >
                      <div className="min-w-0 pr-2">
                        <div className="truncate text-xs">{t.name}</div>
                        <div className="text-[10px] text-stone-400 truncate font-mono">
                          {t.dataDir}
                        </div>
                      </div>
                      {isActive && <Check className="w-3.5 h-3.5 text-amber-700 shrink-0" />}
                    </button>
                  );
                })}
              </div>

              {onOpenNewTenantModal && (
                <div className="border-t border-stone-100 pt-1 mt-1">
                  <button
                    onClick={() => {
                      setIsTenantDropdownOpen(false);
                      onOpenNewTenantModal();
                    }}
                    className="w-full px-3 py-2 text-left text-xs font-semibold text-amber-800 hover:bg-amber-50 flex items-center gap-1.5 transition-colors cursor-pointer"
                  >
                    <Plus className="w-3.5 h-3.5" />
                    Neuen Mandanten anlegen...
                  </button>
                </div>
              )}
            </div>
          </>
        )}
      </div>
    </header>
  );
};
