import { createContext, useContext } from 'react';
import type {
  AppConfig,
  CompanySettings,
  IntegrityCheckResult,
  TenantConfig,
} from './types';

/**
 * Der Zustand, den App.tsx für den ganzen Arbeitsbereich hält: Mandant,
 * Geschäftsjahr, Stammdaten, Konfiguration und Integritätsstand. Die Bereiche
 * lesen ihn hier, statt ihn als Props durch jede Ebene zu reichen.
 */
export interface Workspace {
  appConfig: AppConfig;
  setAppConfig: (config: AppConfig) => void;
  settings: CompanySettings | null;
  /** Nach dem Speichern der Stammdaten: die Kopfzeile zeigt die Firma. */
  reloadSettings: () => Promise<void>;
  tenants: TenantConfig[];
  activeTenant: TenantConfig | null;
  switchTenant: (tenantId: string) => Promise<void>;
  refreshTenants: () => Promise<void>;
  /** Öffnet die Einrichtung für einen weiteren Mandanten. */
  addTenant: () => void;
  /** Nach dem Entfernen des geöffneten Mandanten oder einer Wiederherstellung. */
  reload: () => Promise<void>;
  /** Das Geschäftsjahr aus der Kopfzeile. */
  year: number;
  availableYears: number[];
  /** Festgestellte und offengelegte Jahre; sie nehmen keine Buchung mehr an. */
  closedYears: number[];
  /** Nach Feststellung oder Saldenvortrag: die Jahresauswahl ändert sich. */
  refreshFiscalYears: () => Promise<void>;
  integrity: IntegrityCheckResult | null;
  isCheckingIntegrity: boolean;
  refreshIntegrity: () => Promise<void>;
}

const WorkspaceContext = createContext<Workspace | null>(null);

export const WorkspaceProvider = WorkspaceContext.Provider;

export function useWorkspace(): Workspace {
  const value = useContext(WorkspaceContext);
  if (!value) throw new Error('useWorkspace braucht den WorkspaceProvider aus App.tsx.');
  return value;
}
