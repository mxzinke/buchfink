import React, { useCallback, useEffect, useState } from 'react';
import { AREA_TABS, TAB_LABELS, activeTab, useNavigation, type TabOf } from '../../navigation';
import { useWorkspace } from '../../workspace';
import { useLiveReload } from '../../live';
import { Api } from '../../services/api';
import { ClosedYearNotice } from '../../components/WriteLock';
import { PageFrame, type FrameTab } from '../../components/ui';
import { AccountsTab } from './AccountsTab';
import { AssetsTab } from './AssetsTab';
import { BankTab } from './BankTab';
import { InvoicesTab } from './InvoicesTab';
import { JournalTab } from './JournalTab';
import { ReceiptsTab, needsBooking } from './ReceiptsTab';
import { StatementsTab } from './StatementsTab';

type BooksTab = TabOf<'books'>;

/** Reiter, in denen eine Notfallmaske bucht; nur sie zeigen die Sperre des Geschäftsjahres. */
const POSTING_TABS: BooksTab[] = ['receipts', 'bank', 'invoices'];

/** Was an den Reitern als Zahl steht: die Belege und Umsätze, die noch zu buchen sind. */
interface Counts {
  receipts?: number;
  bank?: number;
}

/**
 * Die Bücher: Belege, Bank, Rechnungen, Journal, Konten, Anlagen, Bilanz und
 * GuV. Jeder Reiter zeigt eine Liste mit Zustand; ein Eintrag öffnet seine
 * Einzelheiten in der Seitenleiste. Die Kennung im Ziel schlägt ihn auf.
 */
export const BooksArea: React.FC = () => {
  const { route, navigate } = useNavigation();
  const { year, closedYears } = useWorkspace();
  const tab = activeTab(route, 'books');
  const [counts, setCounts] = useState<Counts>({});

  const count = useCallback(async () => {
    try {
      const [receipts, transactions] = await Promise.all([Api.getReceipts(''), Api.getBankTransactions()]);
      setCounts({
        receipts: receipts.filter(needsBooking).length,
        bank: transactions.filter((t) => t.matchStatus === 'unmatched').length,
      });
    } catch {
      setCounts({});
    }
  }, []);

  // Neu gezählt wird beim Wechsel des Reiters, dort wurde gerade gebucht, und
  // wenn Claude Belege oder Umsätze bucht.
  useEffect(() => {
    void count();
  }, [count, year, tab]);

  useLiveReload(count, ['belege', 'bank', 'buchungen']);

  const tabs: FrameTab<BooksTab>[] = AREA_TABS.books.map((value) => ({
    value,
    label: TAB_LABELS.books[value],
    count: value === 'receipts' || value === 'bank' ? counts[value] || undefined : undefined,
  }));

  return (
    <PageFrame
      title="Bücher"
      context={`Geschäftsjahr ${year}`}
      tabs={tabs}
      tab={tab}
      onTabChange={(next) => navigate({ area: 'books', tab: next })}
      notice={POSTING_TABS.includes(tab) && closedYears.includes(year) ? <ClosedYearNotice /> : undefined}
    >
      {tab === 'receipts' && <ReceiptsTab />}
      {tab === 'bank' && <BankTab />}
      {tab === 'invoices' && <InvoicesTab />}
      {tab === 'journal' && <JournalTab />}
      {tab === 'accounts' && <AccountsTab />}
      {tab === 'assets' && <AssetsTab />}
      {tab === 'statements' && <StatementsTab />}
    </PageFrame>
  );
};
