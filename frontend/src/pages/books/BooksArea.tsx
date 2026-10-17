import React from 'react';
import { AREA_TABS, TAB_LABELS, activeTab, useNavigation, type TabOf } from '../../navigation';
import { useWorkspace } from '../../workspace';
import { ClosedYearNotice } from '../../components/WriteLock';
import { Disclosure, PageFrame, type FrameTab } from '../../components/ui';
import { AccountsPage } from '../AccountsPage';
import { AdvancesPage } from '../AdvancesPage';
import { AssetsPage } from '../AssetsPage';
import { BankImportPage } from '../BankImportPage';
import { InvoicesPage } from '../InvoicesPage';
import { JournalPage } from '../JournalPage';
import { ReceiptsPage } from '../ReceiptsPage';
import { ReportsPage } from '../ReportsPage';

type BooksTab = TabOf<'books'>;

const TABS: FrameTab<BooksTab>[] = AREA_TABS.books.map((value) => ({
  value,
  label: TAB_LABELS.books[value],
}));

/** Reiter, in denen gebucht wird; nur sie zeigen die Sperre des Geschäftsjahres. */
const POSTING_TABS: BooksTab[] = ['receipts', 'bank', 'invoices', 'journal', 'assets'];

/**
 * Die Bücher: Belege, Bank, Rechnungen, Journal, Konten, Anlagen, Bilanz und
 * GuV. Die Kennung im Ziel schlägt das Objekt im Reiter auf: Beleg-ID,
 * Buchungsnummer oder Kontonummer.
 */
export const BooksArea: React.FC = () => {
  const { route, navigate } = useNavigation();
  const { year, closedYears } = useWorkspace();
  const tab = activeTab(route, 'books');
  const id = route.area === 'books' ? route.id : undefined;
  const receiptId = Number(id);

  return (
    <PageFrame
      title="Bücher"
      context={`Geschäftsjahr ${year}`}
      tabs={TABS}
      tab={tab}
      onTabChange={(next) => navigate({ area: 'books', tab: next })}
      notice={
        POSTING_TABS.includes(tab) && closedYears.includes(year) ? <ClosedYearNotice /> : undefined
      }
    >
      {tab === 'receipts' && (
        <ReceiptsPage
          initialReceiptId={Number.isInteger(receiptId) && receiptId > 0 ? receiptId : undefined}
          onNavigate={navigate}
        />
      )}
      {tab === 'bank' && <BankImportPage />}
      {tab === 'invoices' && (
        <>
          <InvoicesPage onNavigate={navigate} />
          {/* Abschlagsrechnungen und Schlussrechnung, bis der Reiter sie selbst zeigt. */}
          <Disclosure title="Anzahlungen" context="Abschlagsrechnungen und Schlussrechnungen">
            <AdvancesPage />
          </Disclosure>
        </>
      )}
      {tab === 'journal' && <JournalPage initialSearch={id} onNavigate={navigate} />}
      {tab === 'accounts' && <AccountsPage initialAccount={id} onNavigate={navigate} />}
      {tab === 'assets' && <AssetsPage />}
      {tab === 'statements' && <ReportsPage year={year} onNavigate={navigate} />}
    </PageFrame>
  );
};
