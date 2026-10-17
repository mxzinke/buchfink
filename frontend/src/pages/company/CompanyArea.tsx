import React, { useCallback, useEffect, useState } from 'react';
import {
  AREA_TABS,
  TAB_LABELS,
  activeTab,
  useAnchorScroll,
  useNavigation,
  type TabOf,
} from '../../navigation';
import { useWorkspace } from '../../workspace';
import { Api } from '../../services/api';
import type { Account, BankAccount } from '../../types';
import { BankAccountsPanel } from '../../components/BankAccountsPanel';
import { PageFrame, SkeletonRows, toast, type FrameTab } from '../../components/ui';
import { ContactsPage } from '../ContactsPage';
import { SettingsPage } from '../SettingsPage';

type CompanyTab = TabOf<'company'>;

const TABS: FrameTab<CompanyTab>[] = AREA_TABS.company.map((value) => ({
  value,
  label: TAB_LABELS.company[value],
}));

/**
 * Das Unternehmen: Stammdaten, die nur der Mensch pflegt, und die Kontakte.
 * Die Kennung im Reiter „Stammdaten" ist der Block, zu dem gesprungen wird.
 */
export const CompanyArea: React.FC = () => {
  const { route, navigate } = useNavigation();
  const { year, activeTenant, reload, reloadSettings } = useWorkspace();
  const tab = activeTab(route, 'company');
  const id = route.area === 'company' ? route.id : undefined;

  useAnchorScroll(tab === 'profile' ? id : undefined);

  return (
    <PageFrame
      title="Unternehmen"
      context="Gilt über alle Geschäftsjahre"
      tabs={TABS}
      tab={tab}
      onTabChange={(next) => navigate({ area: 'company', tab: next })}
    >
      {tab === 'profile' && (
        <>
          <SettingsPage
            key={activeTenant?.id}
            part="company"
            year={year}
            onNavigate={navigate}
            onTenantDeleted={reload}
            onSaved={reloadSettings}
          />
          <CompanyBankAccounts />
        </>
      )}
      {tab === 'contacts' && <ContactsPage onNavigate={navigate} />}
    </PageFrame>
  );
};

/** Die Bankkonten des Unternehmens und das Konto, das auf Rechnungen steht. */
const CompanyBankAccounts: React.FC = () => {
  const [accounts, setAccounts] = useState<BankAccount[] | null>(null);
  const [paymentAccounts, setPaymentAccounts] = useState<Account[]>([]);

  const load = useCallback(async () => {
    try {
      const [banks, payment] = await Promise.all([Api.getBankAccounts(), Api.getPaymentAccounts()]);
      setAccounts(banks);
      setPaymentAccounts(payment);
    } catch (e) {
      setAccounts([]);
      toast.error(e instanceof Error ? e.message : String(e));
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <div id="bankkonten" className="max-w-[900px] mt-8 border-t border-line">
      {accounts === null ? (
        <div className="mt-8">
          <SkeletonRows rows={3} />
        </div>
      ) : (
        <BankAccountsPanel accounts={accounts} paymentAccounts={paymentAccounts} onChanged={load} />
      )}
    </div>
  );
};
