import { useState, useEffect } from 'react';
import { Lock } from 'lucide-react';
import { Toaster } from 'sonner';
import { Sidebar, TabType } from './components/Sidebar';
import { Header } from './components/Header';
import { StartupScreen } from './components/StartupScreen';
import { SetupAssistantScreen } from './components/SetupAssistantScreen';
import { RecoveryScreen } from './components/RecoveryScreen';
import { DashboardPage } from './pages/DashboardPage';
import { AccountsPage } from './pages/AccountsPage';
import { JournalPage } from './pages/JournalPage';
import { AssetsPage } from './pages/AssetsPage';
import { BankImportPage } from './pages/BankImportPage';
import { InvoicesPage } from './pages/InvoicesPage';
import { ReceiptsPage } from './pages/ReceiptsPage';
import { ContactsPage } from './pages/ContactsPage';
import { ReportsPage } from './pages/ReportsPage';
import { ClosingPage } from './pages/ClosingPage';
import { DeadlinesPage } from './pages/DeadlinesPage';
import { EBilanzPage } from './pages/EBilanzPage';
import { AuditPage } from './pages/AuditPage';
import { SettingsPage } from './pages/SettingsPage';
import { IntegrityCheckResult, CompanySettings, AppConfig, TenantConfig, FiscalYear } from './types';
import { Api } from './services/api';
import { toast } from './components/ui';

/**
 * Ansichten, in denen im gesperrten Geschäftsjahr erfasst würde. Nur sie tragen
 * den Hinweisstreifen; eine Auswertung ist auch im abgeschlossenen Jahr das,
 * was sie sein soll (§11.5).
 */
const POSTING_TABS: TabType[] = ['journal', 'bank', 'receipts', 'invoices', 'assets'];

export function App() {
  const currentCalendarYear = new Date().getFullYear(); // e.g. 2026
  const [appConfig, setAppConfig] = useState<AppConfig | null>(null);
  const [tenants, setTenants] = useState<TenantConfig[]>([]);
  const [activeTenant, setActiveTenant] = useState<TenantConfig | null>(null);
  const [currentTab, setCurrentTab] = useState<TabType>('welcome');
  const [currentYear, setCurrentYear] = useState<number>(currentCalendarYear);
  const [availableYears, setAvailableYears] = useState<number[]>([currentCalendarYear]);
  // Ab der Feststellung nimmt ein Geschäftsjahr keine Buchung mehr an. Der
  // Abschlussstand gehört deshalb in die Kopfzeile und ins Journal und nicht
  // erst in die Fehlermeldung beim Buchen (§11.5).
  const [fiscalYears, setFiscalYears] = useState<FiscalYear[]>([]);
  const [companySettings, setCompanySettings] = useState<CompanySettings | null>(null);
  const [integrity, setIntegrity] = useState<IntegrityCheckResult | null>(null);
  const [isCheckingIntegrity, setIsCheckingIntegrity] = useState(false);
  const [loadingConfig, setLoadingConfig] = useState(true);
  const [isLocked, setIsLocked] = useState(false);
  const [isAddingTenant, setIsAddingTenant] = useState(false);
  const [isMobileSidebarOpen, setIsMobileSidebarOpen] = useState(false);

  useEffect(() => {
    bootstrapApp();
  }, []);

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

      // A configured tenant whose keychain secret is missing on this machine
      // must be recovered before any encrypted data can be read.
      const locked = cfg.isConfigured ? await Api.isLocked() : false;
      setIsLocked(locked);

      if (cfg.isConfigured && !locked) {
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
   * Nach einer Änderung am Abschluss: der Saldenvortrag legt das Zieljahr an und
   * bucht hinein, `GetAvailableFiscalYears` kennt es danach. Ohne die neu
   * gelesene Liste stünde das neue Jahr erst nach einem Neustart zur Auswahl.
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
   * Ein Geschäftsjahr von Hand anlegen: die Entität beginnt am Tag nach dem Ende
   * des Vorjahres, auch nach einem Rumpfgeschäftsjahr, und die Ansicht schaltet
   * auf sie um.
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
      toast.success('Mandant gewechselt.');
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    }
  };

  const refreshIntegrity = async () => {
    setIsCheckingIntegrity(true);
    try {
      const res = await Api.verifyIntegrity();
      setIntegrity(res);
    } catch (e) {
      console.error('Error verifying integrity:', e);
    } finally {
      setIsCheckingIntegrity(false);
    }
  };

  const handleYearChange = async (year: number) => {
    setCurrentYear(year);
    await Api.setFiscalYear(year);
    await loadActiveFiscalYearData();
  };

  const handleSetupCompleted = async () => {
    setIsAddingTenant(false);
    await bootstrapApp();
    setCurrentTab('dashboard');
  };

  // Loading Screen while reading initial config
  if (loadingConfig) {
    return (
      <div className="h-screen flex items-center justify-center bg-shell text-shell-text-muted text-body">
        Buchfink wird gestartet …
      </div>
    );
  }

  // If not configured yet, show the full-screen Setup Assistant
  if (!appConfig?.isConfigured || isAddingTenant) {
    return (
      <SetupAssistantScreen
        onSetupCompleted={handleSetupCompleted}
        onCancel={() => setIsAddingTenant(false)}
        isAdditionalTenant={Boolean(appConfig?.isConfigured)}
      />
    );
  }

  // Configured but locked on this machine: require recovery before the app opens.
  if (isLocked) {
    return (
      <RecoveryScreen
        activeTenant={activeTenant}
        onRecovered={async () => {
          setIsLocked(false);
          await bootstrapApp();
          setCurrentTab('dashboard');
        }}
      />
    );
  }

  // Abgeschlossen ist ein Jahr ab der Feststellung: JournalService.Post weist
  // dann jede Buchung mit Datum in diesem Jahr ab (§11.5).
  const closedYears = fiscalYears
    .filter((fy) => fy.status === 'adopted' || fy.status === 'disclosed')
    .map((fy) => fy.year);
  const yearClosed = closedYears.includes(currentYear);

  const renderContent = () => {
    switch (currentTab) {
      case 'welcome':
        return (
          <StartupScreen
            settings={companySettings}
            tenants={tenants}
            activeTenant={activeTenant}
            onSwitchTenant={handleSwitchTenant}
            onRefreshTenants={refreshTenants}
            onAddTenant={() => setIsAddingTenant(true)}
            onStartDashboard={() => setCurrentTab('dashboard')}
            onNavigate={setCurrentTab}
          />
        );
      case 'dashboard':
        return <DashboardPage onNavigate={setCurrentTab} />;
      case 'accounts':
        return <AccountsPage />;
      case 'journal':
        return <JournalPage closedYear={closedYears.includes(currentYear) ? currentYear : undefined} />;
      case 'assets':
        return <AssetsPage />;
      case 'bank':
        return <BankImportPage />;
      case 'receipts':
        return <ReceiptsPage />;
      case 'invoices':
        return <InvoicesPage />;
      case 'contacts':
        return <ContactsPage />;
      case 'reports':
        return <ReportsPage />;
      case 'closing':
        // Die Abschlussansicht folgt dem Jahr aus der Kopfzeile; sie zeigt
        // Stand und Vortrag genau eines Geschäftsjahres.
        return <ClosingPage year={currentYear} onFiscalYearChanged={refreshFiscalYearSelection} />;
      case 'deadlines':
        return <DeadlinesPage />;
      case 'ebilanz':
        return <EBilanzPage />;
      case 'audit':
        return <AuditPage />;
      case 'settings':
        return <SettingsPage />;
      default:
        return <DashboardPage onNavigate={setCurrentTab} />;
    }
  };

  return (
    <div className="flex h-screen bg-paper text-ink overflow-hidden">
      <Toaster
        position="bottom-right"
        richColors
        closeButton
        toastOptions={{
          className: 'font-sans text-body',
        }}
      />
      {/* Grouped Sidebar Navigation */}
      <Sidebar
        currentTab={currentTab}
        onSelectTab={(tab) => {
          setCurrentTab(tab);
          setIsMobileSidebarOpen(false);
        }}
        settings={companySettings}
        integrity={integrity}
        onRefreshIntegrity={refreshIntegrity}
        isCheckingIntegrity={isCheckingIntegrity}
        isOpenMobile={isMobileSidebarOpen}
        onCloseMobile={() => setIsMobileSidebarOpen(false)}
      />

      {/* Main Content Area */}
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        {currentTab !== 'welcome' && (
          <Header
            currentYear={currentYear}
            availableYears={availableYears}
            closedYears={closedYears}
            onYearChange={handleYearChange}
            onCreateFiscalYear={handleCreateFiscalYear}
            tenants={tenants}
            activeTenant={activeTenant}
            onSwitchTenant={handleSwitchTenant}
            onOpenNewTenantModal={() => setIsAddingTenant(true)}
            onToggleMobileSidebar={() => setIsMobileSidebarOpen(!isMobileSidebarOpen)}
          />
        )}

        {/* Ein abgeschlossenes Geschäftsjahr trägt einen anderen Grund: `sunken`
            statt `paper`, und zwar in jeder Ansicht und nicht nur im Journal —
            die Sperre gilt dem Jahr, nicht der Seite (§11.5). */}
        <main
          className={`flex-1 overflow-y-auto ${
            currentTab === 'welcome' ? 'bg-shell-deep' : yearClosed ? 'bg-sunken' : 'bg-paper'
          }`}
        >
          {/* Der Hinweisstreifen steht über dem Inhalt, damit jede Ansicht mit
              Erfassung ihn zeigt und keine ihn vergisst. */}
          {yearClosed && POSTING_TABS.includes(currentTab) && (
            <div className="max-w-[1200px] mx-auto px-8 pt-8">
              <div className="flex items-start gap-2.5 rounded-control border border-line bg-surface px-4 py-3">
                <Lock className="w-4 h-4 mt-0.5 shrink-0 text-ink-faint" strokeWidth={1.5} />
                <p className="text-body text-ink-muted">
                  {`Geschäftsjahr ${currentYear} ist abgeschlossen. Buchungen sind nur im laufenden Jahr möglich.`}
                </p>
              </div>
            </div>
          )}
          {renderContent()}
        </main>
      </div>
    </div>
  );
}

export default App;
