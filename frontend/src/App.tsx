import { useEffect, useRef, useState, type ReactNode } from 'react';
import { Toaster } from 'sonner';
import { Sidebar } from './components/Sidebar';
import { Header } from './components/Header';
import { SetupAssistantScreen } from './components/SetupAssistantScreen';
import { PostingLockProvider, WriteLockProvider } from './components/WriteLock';
import { AppConsentDialog } from './components/AppConsentDialog';
import { IntegrityCheckResult, CompanySettings, AppConfig, TenantConfig, FiscalYear } from './types';
import { Api } from './services/api';
import { formatDate } from './utils/formatters';
import { Button, Notice, PAGE_CONTAINER, cn, toast } from './components/ui';
import { NavigationProvider, useHashRoute, type Route } from './navigation';
import { WorkspaceProvider, type Workspace } from './workspace';
import { OverviewArea } from './pages/overview/OverviewArea';
import { BooksArea } from './pages/books/BooksArea';
import { ClosingsArea } from './pages/closings/ClosingsArea';
import { CompanyArea } from './pages/company/CompanyArea';
import { SettingsArea } from './pages/settings/SettingsArea';

export function App() {
  const currentCalendarYear = new Date().getFullYear();
  const [appConfig, setAppConfig] = useState<AppConfig | null>(null);
  const [tenants, setTenants] = useState<TenantConfig[]>([]);
  const [activeTenant, setActiveTenant] = useState<TenantConfig | null>(null);
  const [route, setRoute] = useHashRoute();
  const [currentYear, setCurrentYear] = useState<number>(currentCalendarYear);
  const [availableYears, setAvailableYears] = useState<number[]>([currentCalendarYear]);
  // Ab der Feststellung nimmt ein Geschäftsjahr keine Buchung mehr an. Der
  // Abschlussstand gehört deshalb in die Kopfzeile und nicht erst in die
  // Fehlermeldung beim Buchen (§11.5).
  const [fiscalYears, setFiscalYears] = useState<FiscalYear[]>([]);
  const [companySettings, setCompanySettings] = useState<CompanySettings | null>(null);
  const [integrity, setIntegrity] = useState<IntegrityCheckResult | null>(null);
  const [integrityError, setIntegrityError] = useState<string | null>(null);
  const [isCheckingIntegrity, setIsCheckingIntegrity] = useState(false);
  const [loadingConfig, setLoadingConfig] = useState(true);
  const [isAddingTenant, setIsAddingTenant] = useState(false);
  const [isMobileSidebarOpen, setIsMobileSidebarOpen] = useState(false);
  const mainRef = useRef<HTMLElement>(null);

  useEffect(() => {
    void bootstrapApp();
  }, []);

  // Ein anderer Bereich oder Reiter beginnt oben; eine geöffnete Seitenleiste
  // im selben Reiter lässt die Liste stehen, wo sie war.
  useEffect(() => {
    mainRef.current?.scrollTo({ top: 0 });
  }, [route.area, route.tab]);

  const bootstrapApp = async () => {
    setLoadingConfig(true);
    try {
      const [cfg, tenantList, currentActive] = await Promise.all([
        Api.getAppConfig(),
        Api.getTenants(),
        Api.getActiveTenant(),
      ]);
      setAppConfig(cfg);
      setTenants(tenantList);
      setActiveTenant(currentActive);

      if (cfg.isConfigured) {
        await loadActiveFiscalYearData();
      }
    } catch (e) {
      console.error('Error bootstrapping Buchfink:', e);
    } finally {
      setLoadingConfig(false);
    }
  };

  const loadActiveFiscalYearData = async () => {
    try {
      const [year, years, settings, currentActive] = await Promise.all([
        Api.getFiscalYear(),
        Api.getAvailableFiscalYears(),
        Api.getCompanySettings(),
        Api.getActiveTenant(),
      ]);
      setCurrentYear(year);
      setAvailableYears(years.length > 0 ? years : [year]);
      setCompanySettings(settings);
      if (currentActive) {
        setActiveTenant(currentActive);
      }
      await refreshFiscalYears();
      await refreshIntegrity();
    } catch (e) {
      console.error('Error loading fiscal year data:', e);
    }
  };

  /**
   * Die Abschlussstände der Geschäftsjahre. Eigener Fehlerpfad: fehlen sie,
   * fehlt nur das Schloss, die übrige Ansicht bleibt benutzbar.
   */
  const refreshFiscalYears = async () => {
    try {
      setFiscalYears(await Api.getFiscalYears());
    } catch (e) {
      console.error('Error loading fiscal years:', e);
      setFiscalYears([]);
    }
  };

  /**
   * Nach einer Änderung am Abschluss: der Saldenvortrag legt das Zieljahr an.
   * Ohne die neu gelesene Liste stünde das neue Jahr erst nach einem Neustart
   * zur Auswahl.
   */
  const refreshFiscalYearSelection = async () => {
    await refreshFiscalYears();
    try {
      const years = await Api.getAvailableFiscalYears();
      if (years.length > 0) {
        setAvailableYears(years);
      }
    } catch (e) {
      console.error('Error loading available fiscal years:', e);
    }
  };

  /**
   * Ein Geschäftsjahr von Hand anlegen und darauf umschalten. Welche Jahre sich
   * anlegen lassen, entscheidet das Backend.
   */
  const handleCreateFiscalYear = async (year: number) => {
    try {
      await Api.createFiscalYear(year);
      await loadActiveFiscalYearData();
      toast.success(`Geschäftsjahr ${year} angelegt.`);
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    }
  };

  const refreshTenants = async () => {
    try {
      const [tenantList, currentActive] = await Promise.all([
        Api.getTenants(),
        Api.getActiveTenant(),
      ]);
      setTenants(tenantList);
      setActiveTenant(currentActive);
    } catch (e) {
      console.error('Error refreshing tenants:', e);
    }
  };

  const handleSwitchTenant = async (tenantId: string) => {
    try {
      await Api.switchTenant(tenantId);
      await refreshTenants();
      await loadActiveFiscalYearData();
      setRoute({ area: 'overview' });
      toast.success('Mandant gewechselt.');
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    }
  };

  const refreshIntegrity = async () => {
    setIsCheckingIntegrity(true);
    try {
      setIntegrity(await Api.verifyIntegrity());
      setIntegrityError(null);
    } catch (e) {
      setIntegrity(null);
      setIntegrityError(e instanceof Error ? e.message : String(e));
    } finally {
      setIsCheckingIntegrity(false);
    }
  };

  const reloadSettings = async () => {
    try {
      setCompanySettings(await Api.getCompanySettings());
    } catch (e) {
      console.error('Error loading company settings:', e);
    }
  };

  const handleYearChange = async (year: number) => {
    setCurrentYear(year);
    await Api.setFiscalYear(year);
    await loadActiveFiscalYearData();
  };

  /**
   * Wechselt die Ansicht. Nennt das Ziel ein Geschäftsjahr, wird zuerst
   * umgeschaltet: eine Aufgabe zum Vorjahresabschluss landete sonst auf
   * derselben Seite mit den Zahlen des laufenden Jahres.
   */
  const navigate = (next: Route) => {
    if (next.year && next.year !== currentYear) {
      void handleYearChange(next.year);
    }
    setRoute(next);
    setIsMobileSidebarOpen(false);
  };

  const handleSetupCompleted = async () => {
    setIsAddingTenant(false);
    await bootstrapApp();
    setRoute({ area: 'overview' });
  };

  // Abgeschlossen ist ein Jahr ab der Feststellung: JournalService.Post weist
  // dann jede Buchung mit Datum in diesem Jahr ab (§11.5).
  const closedYears = fiscalYears
    .filter((fy) => fy.status === 'adopted' || fy.status === 'disclosed')
    .map((fy) => fy.year);
  const yearClosed = closedYears.includes(currentYear);

  // Der Zustimmungsdialog steht über jedem Schirm: eine App kann sich auch
  // während der Einrichtung melden, wenn dort Claude verbunden wird.
  const shell = (content: ReactNode) => (
    <>
      <Toaster
        position="bottom-right"
        richColors
        closeButton
        toastOptions={{ className: 'font-sans text-body' }}
      />
      <AppConsentDialog />
      {content}
    </>
  );

  if (loadingConfig) {
    return (
      <div className="h-screen flex items-center justify-center bg-shell text-shell-text-muted text-body">
        Buchfink wird gestartet …
      </div>
    );
  }

  // Die Onboarding-Weiche: ohne eingerichteten Mandanten und beim Anlegen
  // eines weiteren steht die Einrichtung vor dem Arbeitsbereich.
  if (!appConfig?.isConfigured || isAddingTenant) {
    return shell(
      <SetupAssistantScreen
        onSetupCompleted={handleSetupCompleted}
        onCancel={() => setIsAddingTenant(false)}
        isAdditionalTenant={Boolean(appConfig?.isConfigured)}
      />,
    );
  }

  const workspace: Workspace = {
    appConfig,
    setAppConfig,
    settings: companySettings,
    reloadSettings,
    tenants,
    activeTenant,
    switchTenant: handleSwitchTenant,
    refreshTenants,
    addTenant: () => setIsAddingTenant(true),
    reload: bootstrapApp,
    year: currentYear,
    availableYears,
    closedYears,
    refreshFiscalYears: refreshFiscalYearSelection,
    integrity,
    integrityError,
    isCheckingIntegrity,
    refreshIntegrity,
  };

  return shell(
    // Der Prüfermodus sperrt jede schreibende Bedienung der Anwendung; jeder
    // schreibende Knopf liest ihn über `useWriteLock`. Das festgestellte
    // Geschäftsjahr sperrt die Erfassung; Buchungsansichten lesen es über
    // `usePostingLock` (§11.5).
    <WriteLockProvider
      readOnly={appConfig.readOnly ?? false}
      until={appConfig.readOnlyUntil}
      reason={appConfig.readOnlyReason}
    >
      <PostingLockProvider closedYear={yearClosed ? currentYear : undefined}>
        <WorkspaceProvider value={workspace}>
          <NavigationProvider value={{ route, navigate }}>
            <div className="flex h-screen bg-paper text-ink overflow-hidden">
              <Sidebar
                currentArea={route.area}
                onSelectArea={(area) => navigate({ area } as Route)}
                integrity={integrity}
                integrityError={integrityError}
                isCheckingIntegrity={isCheckingIntegrity}
                onOpenAudit={() => navigate({ area: 'settings', tab: 'audit' })}
                isOpenMobile={isMobileSidebarOpen}
                onCloseMobile={() => setIsMobileSidebarOpen(false)}
              />

              <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
                <Header
                  settings={companySettings}
                  tenants={tenants}
                  activeTenant={activeTenant}
                  onSwitchTenant={(id) => void handleSwitchTenant(id)}
                  onManageTenants={() => navigate({ area: 'settings', tab: 'tenants' })}
                  onAddTenant={() => setIsAddingTenant(true)}
                  currentYear={currentYear}
                  availableYears={availableYears}
                  closedYears={closedYears}
                  onYearChange={handleYearChange}
                  onCreateFiscalYear={handleCreateFiscalYear}
                  onToggleMobileSidebar={() => setIsMobileSidebarOpen(!isMobileSidebarOpen)}
                />

                {/* Ein abgeschlossenes Geschäftsjahr hat einen anderen Grund:
                    `sunken` statt `paper`, in jeder Ansicht (§11.5). */}
                <main
                  ref={mainRef}
                  className={cn('flex-1 overflow-y-auto', yearClosed ? 'bg-sunken' : 'bg-paper')}
                >
                  {/* Der Prüfermodus sperrt die ganze Anwendung; der Hinweis
                      steht deshalb über jedem Bereich (§11.5). */}
                  {appConfig.readOnly && (
                    <div className={cn(PAGE_CONTAINER, 'pt-8')}>
                      <Notice
                        text={`Prüfermodus bis ${formatDate(appConfig.readOnlyUntil)} — die Buchführung nimmt bis dahin keine Änderung auf.`}
                        action={
                          <Button
                            variant="secondary"
                            size="sm"
                            onClick={() =>
                              navigate({ area: 'settings', tab: 'audit', id: 'betriebspruefung' })
                            }
                          >
                            Zum Prüfermodus
                          </Button>
                        }
                      />
                    </div>
                  )}
                  {route.area === 'overview' && <OverviewArea />}
                  {route.area === 'books' && <BooksArea />}
                  {route.area === 'closings' && <ClosingsArea />}
                  {route.area === 'company' && <CompanyArea />}
                  {route.area === 'settings' && <SettingsArea />}
                </main>
              </div>
            </div>
          </NavigationProvider>
        </WorkspaceProvider>
      </PostingLockProvider>
    </WriteLockProvider>,
  );
}

export default App;
