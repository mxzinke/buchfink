import { useState, useEffect } from 'react';
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
import { DeadlinesPage } from './pages/DeadlinesPage';
import { EBilanzPage } from './pages/EBilanzPage';
import { AuditPage } from './pages/AuditPage';
import { SettingsPage } from './pages/SettingsPage';
import { IntegrityCheckResult, CompanySettings, AppConfig, TenantConfig } from './types';
import { Api } from './services/api';
import { toast } from './components/ui';

export function App() {
  const currentCalendarYear = new Date().getFullYear(); // e.g. 2026
  const [appConfig, setAppConfig] = useState<AppConfig | null>(null);
  const [tenants, setTenants] = useState<TenantConfig[]>([]);
  const [activeTenant, setActiveTenant] = useState<TenantConfig | null>(null);
  const [currentTab, setCurrentTab] = useState<TabType>('welcome');
  const [currentYear, setCurrentYear] = useState<number>(currentCalendarYear);
  const [availableYears, setAvailableYears] = useState<number[]>([currentCalendarYear]);
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
      await refreshIntegrity();
    } catch (e) {
      console.error('Error loading fiscal year data:', e);
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
        return <JournalPage />;
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
            onYearChange={handleYearChange}
            tenants={tenants}
            activeTenant={activeTenant}
            onSwitchTenant={handleSwitchTenant}
            onOpenNewTenantModal={() => setIsAddingTenant(true)}
            onToggleMobileSidebar={() => setIsMobileSidebarOpen(!isMobileSidebarOpen)}
          />
        )}

        <main
          className={`flex-1 overflow-y-auto ${
            currentTab === 'welcome' ? 'bg-shell-deep' : 'bg-paper'
          }`}
        >
          {renderContent()}
        </main>
      </div>
    </div>
  );
}

export default App;
