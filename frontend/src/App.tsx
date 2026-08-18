import { useState, useEffect } from 'react';
import { Sidebar, TabType } from './components/Sidebar';
import { Header } from './components/Header';
import { StartupScreen } from './components/StartupScreen';
import { SetupAssistantScreen } from './components/SetupAssistantScreen';
import { DashboardPage } from './pages/DashboardPage';
import { AccountsPage } from './pages/AccountsPage';
import { JournalPage } from './pages/JournalPage';
import { BankImportPage } from './pages/BankImportPage';
import { InvoicesPage } from './pages/InvoicesPage';
import { ContactsPage } from './pages/ContactsPage';
import { ReportsPage } from './pages/ReportsPage';
import { EBilanzPage } from './pages/EBilanzPage';
import { AuditPage } from './pages/AuditPage';
import { SettingsPage } from './pages/SettingsPage';
import { IntegrityCheckResult, CompanySettings, AppConfig } from './types';
import { Api } from './services/api';

export function App() {
  const currentCalendarYear = new Date().getFullYear(); // e.g. 2026
  const [appConfig, setAppConfig] = useState<AppConfig | null>(null);
  const [currentTab, setCurrentTab] = useState<TabType>('welcome');
  const [currentYear, setCurrentYear] = useState<number>(currentCalendarYear);
  const [availableYears, setAvailableYears] = useState<number[]>([currentCalendarYear]);
  const [companySettings, setCompanySettings] = useState<CompanySettings | null>(null);
  const [integrity, setIntegrity] = useState<IntegrityCheckResult | null>(null);
  const [isCheckingIntegrity, setIsCheckingIntegrity] = useState(false);
  const [loadingConfig, setLoadingConfig] = useState(true);

  useEffect(() => {
    bootstrapApp();
  }, []);

  const bootstrapApp = async () => {
    setLoadingConfig(true);
    try {
      const cfg = await Api.getAppConfig();
      setAppConfig(cfg);

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
      const [year, years, settings] = await Promise.all([
        Api.getFiscalYear(),
        Api.getAvailableFiscalYears(),
        Api.getCompanySettings(),
      ]);
      setCurrentYear(year);
      setAvailableYears(years.length > 0 ? years : [year]);
      setCompanySettings(settings);
      await refreshIntegrity();
    } catch (e) {
      console.error('Error loading fiscal year data:', e);
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

  const handleCreateYear = async (year: number) => {
    await Api.createFiscalYear(year);
    await handleYearChange(year);
  };

  const handleSetupCompleted = async () => {
    await bootstrapApp();
    setCurrentTab('dashboard');
  };

  // Loading Screen while reading initial config
  if (loadingConfig) {
    return (
      <div className="h-screen flex items-center justify-center bg-stone-950 text-stone-400 text-xs font-mono">
        Buchfink wird gestartet...
      </div>
    );
  }

  // If not configured yet, show the full-screen Setup Assistant
  if (!appConfig?.isConfigured) {
    return <SetupAssistantScreen onSetupCompleted={handleSetupCompleted} />;
  }

  const renderContent = () => {
    switch (currentTab) {
      case 'welcome':
        return (
          <StartupScreen
            settings={companySettings}
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
      case 'bank':
        return <BankImportPage />;
      case 'invoices':
        return <InvoicesPage />;
      case 'contacts':
        return <ContactsPage />;
      case 'reports':
        return <ReportsPage />;
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
    <div className="flex h-screen bg-[#FAF9F6] text-stone-900 overflow-hidden font-sans">
      {/* Grouped Sidebar Navigation with bottom GoBD badge */}
      <Sidebar
        currentTab={currentTab}
        onSelectTab={setCurrentTab}
        settings={companySettings}
        integrity={integrity}
        onRefreshIntegrity={refreshIntegrity}
        isCheckingIntegrity={isCheckingIntegrity}
      />

      {/* Main Content Area */}
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        {currentTab !== 'welcome' && (
          <Header
            currentYear={currentYear}
            availableYears={availableYears}
            onYearChange={handleYearChange}
            onCreateYear={handleCreateYear}
          />
        )}

        <main className={`flex-1 overflow-y-auto ${currentTab === 'welcome' ? 'bg-stone-950' : 'bg-[#FAF9F6]'}`}>
          {renderContent()}
        </main>
      </div>
    </div>
  );
}

export default App;
