import { useState, useEffect } from 'react';
import { Sidebar, TabType } from './components/Sidebar';
import { Header } from './components/Header';
import { StartupScreen } from './components/StartupScreen';
import { OnboardingModal } from './components/OnboardingModal';
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
import { IntegrityCheckResult, CompanySettings } from './types';
import { Api } from './services/api';

export function App() {
  const [currentTab, setCurrentTab] = useState<TabType>('welcome');
  const [currentYear, setCurrentYear] = useState<number>(2024);
  const [companySettings, setCompanySettings] = useState<CompanySettings | null>(null);
  const [integrity, setIntegrity] = useState<IntegrityCheckResult | null>(null);
  const [isOnboardingOpen, setIsOnboardingOpen] = useState(false);
  const [isCheckingIntegrity, setIsCheckingIntegrity] = useState(false);

  useEffect(() => {
    loadInitialData();
  }, [currentYear]);

  const loadInitialData = async () => {
    try {
      const [year, settings] = await Promise.all([
        Api.getFiscalYear(),
        Api.getCompanySettings(),
      ]);
      setCurrentYear(year);
      setCompanySettings(settings);
      await refreshIntegrity();
    } catch (e) {
      console.error('Error loading initial data:', e);
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
    await loadInitialData();
  };

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
      {/* Grouped Sidebar Navigation */}
      <Sidebar
        currentTab={currentTab}
        onSelectTab={setCurrentTab}
        settings={companySettings}
        onOpenOnboarding={() => setIsOnboardingOpen(true)}
      />

      {/* Main Content Area */}
      <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
        {currentTab !== 'welcome' && (
          <Header
            currentYear={currentYear}
            onYearChange={handleYearChange}
            integrity={integrity}
            onRefreshIntegrity={refreshIntegrity}
            isCheckingIntegrity={isCheckingIntegrity}
          />
        )}

        <main className={`flex-1 overflow-y-auto ${currentTab === 'welcome' ? 'bg-stone-950' : 'bg-[#FAF9F6]'}`}>
          {renderContent()}
        </main>
      </div>

      {/* Onboarding Wizard Modal */}
      <OnboardingModal
        isOpen={isOnboardingOpen}
        onClose={() => setIsOnboardingOpen(false)}
        initialSettings={companySettings}
        onCompleted={loadInitialData}
      />
    </div>
  );
}

export default App;
