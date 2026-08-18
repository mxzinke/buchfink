import React, { useEffect, useState, useMemo } from 'react';
import {
  Search,
  Filter,
  BookOpen,
  Scale,
  ArrowLeft,
  ChevronRight,
  ChevronDown,
  Download,
  CheckCircle2,
  AlertCircle,
  Layers,
  FileText,
  RefreshCw,
  Eye,
  Tag,
  ShieldCheck,
  Building2,
  SlidersHorizontal,
  FolderTree,
  TrendingUp,
  Folder,
  FolderOpen,
  EyeOff,
  ListTree,
} from 'lucide-react';
import {
  Account,
  AccountLedger,
  SuSaOverview,
  SKR04Catalog,
} from '../types';
import { Api } from '../services/api';
import { formatCurrency, formatPercent } from '../utils/formatters';
import { HelpTooltip } from '../components/HelpTooltip';

type ViewMode = 'list' | 'susa' | 'hierarchy' | 'detail';

interface TreeNodeData {
  id: string;
  name: string;
  hgbCode?: string;
  level: number;
  type: 'root' | 'main_group' | 'group' | 'position';
  statementType: string;
  balanceSide: string;
  totalDebit: number;
  totalCredit: number;
  balance: number;
  accountsCount: number;
  accounts: Account[];
  children?: TreeNodeData[];
}

export const AccountsPage: React.FC = () => {
  const [viewMode, setViewMode] = useState<ViewMode>('list');
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [susaOverview, setSusaOverview] = useState<SuSaOverview | null>(null);
  const [catalog, setCatalog] = useState<SKR04Catalog | null>(null);
  const [selectedAccountNumber, setSelectedAccountNumber] = useState<string | null>(null);
  const [selectedLedger, setSelectedLedger] = useState<AccountLedger | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadingLedger, setLoadingLedger] = useState(false);

  // Filters for Accounts List
  const [search, setSearch] = useState('');
  const [typeFilter, setTypeFilter] = useState<string>('all');
  const [classFilter, setClassFilter] = useState<string>('all');
  const [activityFilter, setActivityFilter] = useState<'all' | 'active_balance' | 'has_turnover'>('all');
  const [hideReserved, setHideReserved] = useState(true);
  const [groupByClass, setGroupByClass] = useState(true);
  const [collapsedListClasses, setCollapsedListClasses] = useState<Record<number, boolean>>({});

  // Filters for SuSa
  const [susaSearch, setSusaSearch] = useState('');
  const [susaOnlyWithTurnover, setSusaOnlyWithTurnover] = useState(true);
  const [collapsedClasses, setCollapsedClasses] = useState<Record<number, boolean>>({});

  // Filters for Hierarchy Tree View
  const [hierarchySection, setHierarchySection] = useState<'aktiva' | 'passiva' | 'guv' | 'statistisch'>('aktiva');
  const [hierarchySearch, setHierarchySearch] = useState('');
  const [hierarchyOnlyBebucht, setHierarchyOnlyBebucht] = useState(false);
  const [expandedNodes, setExpandedNodes] = useState<Record<string, boolean>>({
    'aktiva.anlagevermoegen': true,
    'aktiva.umlaufvermoegen': true,
    'passiva.eigenkapital': true,
    'passiva.verbindlichkeiten': true,
    'guv.umsatzerloese': true,
    'guv.materialaufwand': true,
    'guv.personalaufwand': true,
  });

  // Filters for Ledger Detail
  const [ledgerSearch, setLedgerSearch] = useState('');
  const [ledgerDirFilter, setLedgerDirFilter] = useState<'all' | 'SOLL' | 'HABEN'>('all');

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    setLoading(true);
    try {
      const [accList, susa, cat] = await Promise.all([
        Api.getAccounts(),
        Api.getSuSaOverview(),
        Api.getSKR04Catalog(),
      ]);
      setAccounts(accList);
      setSusaOverview(susa);
      setCatalog(cat);
    } catch (e) {
      console.error('Fehler beim Laden der Konten:', e);
    } finally {
      setLoading(false);
    }
  };

  const openAccountDetail = async (accNumber: string) => {
    setSelectedAccountNumber(accNumber);
    setViewMode('detail');
    setLoadingLedger(true);
    try {
      const ledger = await Api.getAccountLedger(accNumber);
      setSelectedLedger(ledger);
    } catch (e) {
      console.error('Fehler beim Laden des Kontoblatts:', e);
    } finally {
      setLoadingLedger(false);
    }
  };

  const navigateAccount = (direction: 'prev' | 'next') => {
    if (!selectedAccountNumber || accounts.length === 0) return;
    const currentIndex = accounts.findIndex((a) => a.number === selectedAccountNumber);
    if (currentIndex === -1) return;
    let nextIndex = direction === 'next' ? currentIndex + 1 : currentIndex - 1;
    if (nextIndex < 0) nextIndex = accounts.length - 1;
    if (nextIndex >= accounts.length) nextIndex = 0;
    openAccountDetail(accounts[nextIndex].number);
  };

  const toggleClassCollapse = (clsNum: number) => {
    setCollapsedClasses((prev) => ({
      ...prev,
      [clsNum]: !prev[clsNum],
    }));
  };

  const toggleListClassCollapse = (clsNum: number) => {
    setCollapsedListClasses((prev) => ({
      ...prev,
      [clsNum]: !prev[clsNum],
    }));
  };

  const expandAllListClasses = () => {
    setCollapsedListClasses({});
  };

  const collapseAllListClasses = () => {
    const next: Record<number, boolean> = {};
    for (let i = 0; i <= 9; i++) {
      next[i] = true;
    }
    setCollapsedListClasses(next);
  };

  const toggleNodeExpand = (nodeId: string) => {
    setExpandedNodes((prev) => ({
      ...prev,
      [nodeId]: !prev[nodeId],
    }));
  };

  const expandAllHierarchy = () => {
    const next: Record<string, boolean> = {};
    const expandTree = (nodes: TreeNodeData[]) => {
      for (const n of nodes) {
        next[n.id] = true;
        if (n.children) expandTree(n.children);
      }
    };
    if (hierarchyTree) expandTree([hierarchyTree]);
    setExpandedNodes(next);
  };

  const collapseAllHierarchy = () => {
    setExpandedNodes({});
  };

  // Map accounts by number for fast hierarchy matching
  const accountsMap = useMemo(() => {
    const map = new Map<string, Account>();
    for (const a of accounts) {
      map.set(a.number, a);
    }
    return map;
  }, [accounts]);

  // Class descriptions
  const classNames: Record<number, string> = {
    0: 'Klasse 0: Anlagevermögenskonten',
    1: 'Klasse 1: Umlaufvermögenskonten',
    2: 'Klasse 2: Eigenkapital- & Fremdkapitalkonten',
    3: 'Klasse 3: Fremdkapitalkonten (Verbindlichkeiten)',
    4: 'Klasse 4: Betriebliche Erträge',
    5: 'Klasse 5: Betriebliche Aufwendungen (Material / Fremdleistungen)',
    6: 'Klasse 6: Betriebliche Aufwendungen (Personal / AfA / Sonstige)',
    7: 'Klasse 7: Weitere Erträge & Aufwendungen (Finanzen / Steuern)',
    8: 'Klasse 8: Freie Kontenklasse / Sonderkonten',
    9: 'Klasse 9: Vortrags-, Kapital- & statistische Konten',
  };

  // Filtered accounts for List view
  const filteredAccounts = useMemo(() => {
    return accounts.filter((acc) => {
      if (hideReserved && acc.isReserved) {
        return false;
      }

      const q = search.trim().toLowerCase();
      const matchesSearch =
        !q ||
        acc.number.toLowerCase().includes(q) ||
        acc.name.toLowerCase().includes(q) ||
        (acc.category && acc.category.toLowerCase().includes(q)) ||
        (acc.subcategory && acc.subcategory.toLowerCase().includes(q)) ||
        (acc.posten && acc.posten.toLowerCase().includes(q)) ||
        (acc.hgbCode && acc.hgbCode.toLowerCase().includes(q)) ||
        (acc.description && acc.description.toLowerCase().includes(q));

      const matchesType = typeFilter === 'all' || acc.type === typeFilter;
      const matchesClass =
        classFilter === 'all' || acc.kontenklasse?.toString() === classFilter;

      let matchesActivity = true;
      if (activityFilter === 'active_balance') {
        matchesActivity = acc.balance !== 0;
      } else if (activityFilter === 'has_turnover') {
        matchesActivity = (acc.debitSum || 0) > 0 || (acc.creditSum || 0) > 0 || (acc.bookingsCount || 0) > 0;
      }

      return matchesSearch && matchesType && matchesClass && matchesActivity;
    });
  }, [accounts, search, typeFilter, classFilter, activityFilter, hideReserved]);

  // Group filtered accounts by Kontenklasse
  const accountsByClass = useMemo(() => {
    const groups: {
      kontenklasse: number;
      name: string;
      accounts: Account[];
      totalDebit: number;
      totalCredit: number;
      balance: number;
      subcategories: { name: string; accounts: Account[] }[];
    }[] = [];

    for (let k = 0; k <= 9; k++) {
      const classAccs = filteredAccounts.filter((a) => (a.kontenklasse ?? 0) === k);
      if (classAccs.length === 0) continue;

      let totalDebit = 0;
      let totalCredit = 0;
      let balance = 0;

      // Group by subcategory
      const subMap = new Map<string, Account[]>();
      for (const a of classAccs) {
        totalDebit += a.debitSum || 0;
        totalCredit += a.creditSum || 0;
        balance += a.balance || 0;

        const subName = a.subcategory || a.category || 'Standardkonten';
        if (!subMap.has(subName)) {
          subMap.set(subName, []);
        }
        subMap.get(subName)!.push(a);
      }

      const subcategories = Array.from(subMap.entries()).map(([name, accList]) => ({
        name,
        accounts: accList,
      }));

      groups.push({
        kontenklasse: k,
        name: classNames[k] || `Klasse ${k}`,
        accounts: classAccs,
        totalDebit,
        totalCredit,
        balance,
        subcategories,
      });
    }

    return groups;
  }, [filteredAccounts]);

  // Build the hierarchical HGB tree structure with live balances
  const hierarchyTree = useMemo<TreeNodeData | null>(() => {
    if (!catalog?.hierarchy) return null;

    const findAccountsForNumbers = (numbers: string[]): Account[] => {
      const result: Account[] = [];
      for (const num of numbers) {
        const direct = accountsMap.get(num);
        if (direct) {
          result.push(direct);
        } else {
          const found = accounts.find(
            (a) =>
              a.number === num ||
              (a.isRange && a.rangeStart && a.rangeEnd && num >= a.rangeStart && num <= a.rangeEnd)
          );
          if (found && !result.includes(found)) {
            result.push(found);
          }
        }
      }
      return result;
    };

    const buildTreeBranch = (
      rawDict: Record<string, any>,
      sectionKey: string,
      rootName: string,
      statementType: string,
      balanceSide: string
    ): TreeNodeData => {
      const rootNode: TreeNodeData = {
        id: sectionKey,
        name: rootName,
        level: 0,
        type: 'root',
        statementType,
        balanceSide,
        totalDebit: 0,
        totalCredit: 0,
        balance: 0,
        accountsCount: 0,
        accounts: [],
        children: [],
      };

      for (const [mainKey, mainVal] of Object.entries(rawDict)) {
        if (!mainVal || typeof mainVal !== 'object') continue;
        const mainNodeId = `${sectionKey}.${mainKey}`;
        const mainNodeName = mainVal.name || mainKey;
        const groups = mainVal.groups || {};

        const mainGroupNode: TreeNodeData = {
          id: mainNodeId,
          name: mainNodeName,
          level: 1,
          type: 'main_group',
          statementType,
          balanceSide,
          totalDebit: 0,
          totalCredit: 0,
          balance: 0,
          accountsCount: 0,
          accounts: [],
          children: [],
        };

        for (const [grpKey, grpVal] of Object.entries(groups)) {
          if (!grpVal || typeof grpVal !== 'object') continue;
          const grpNodeId = `${mainNodeId}.${grpKey}`;
          const grpNodeName = (grpVal as any).name || grpKey;
          const positions = (grpVal as any).positions || {};

          const groupNode: TreeNodeData = {
            id: grpNodeId,
            name: grpNodeName,
            level: 2,
            type: 'group',
            statementType,
            balanceSide,
            totalDebit: 0,
            totalCredit: 0,
            balance: 0,
            accountsCount: 0,
            accounts: [],
            children: [],
          };

          for (const [posKey, posVal] of Object.entries(positions)) {
            if (!posVal || typeof posVal !== 'object') continue;
            const p = posVal as any;
            const posNodeId = p.position_id || `${grpNodeId}.${posKey}`;
            const posName = p.name || posKey;
            const hgbCode = p.hgb_code || '';
            const accNumbers: string[] = p.account_numbers || [];

            const matchedAccs = findAccountsForNumbers(accNumbers);
            let posDebit = 0;
            let posCredit = 0;
            let posBalance = 0;

            for (const a of matchedAccs) {
              posDebit += a.debitSum || 0;
              posCredit += a.creditSum || 0;
              posBalance += a.balance || 0;
            }

            const posNode: TreeNodeData = {
              id: posNodeId,
              name: posName,
              hgbCode,
              level: 3,
              type: 'position',
              statementType,
              balanceSide,
              totalDebit: posDebit,
              totalCredit: posCredit,
              balance: posBalance,
              accountsCount: matchedAccs.length,
              accounts: matchedAccs,
            };

            groupNode.children?.push(posNode);
            groupNode.totalDebit += posDebit;
            groupNode.totalCredit += posCredit;
            groupNode.balance += posBalance;
            groupNode.accountsCount += matchedAccs.length;
          }

          mainGroupNode.children?.push(groupNode);
          mainGroupNode.totalDebit += groupNode.totalDebit;
          mainGroupNode.totalCredit += groupNode.totalCredit;
          mainGroupNode.balance += groupNode.balance;
          mainGroupNode.accountsCount += groupNode.accountsCount;
        }

        rootNode.children?.push(mainGroupNode);
        rootNode.totalDebit += mainGroupNode.totalDebit;
        rootNode.totalCredit += mainGroupNode.totalCredit;
        rootNode.balance += mainGroupNode.balance;
        rootNode.accountsCount += mainGroupNode.accountsCount;
      }

      return rootNode;
    };

    const hier = catalog.hierarchy;
    if (hierarchySection === 'aktiva' && hier.bilanz?.aktiva) {
      return buildTreeBranch(hier.bilanz.aktiva, 'aktiva', 'Aktiva (Bilanz)', 'Bilanz', 'Aktiva');
    }
    if (hierarchySection === 'passiva' && hier.bilanz?.passiva) {
      return buildTreeBranch(hier.bilanz.passiva, 'passiva', 'Passiva (Bilanz)', 'Bilanz', 'Passiva');
    }
    if (hierarchySection === 'guv' && hier.guv) {
      return buildTreeBranch(hier.guv, 'guv', 'Gewinn- und Verlustrechnung (GuV)', 'GuV', 'GuV');
    }
    if (hierarchySection === 'statistisch' && hier.statistisch) {
      return buildTreeBranch(hier.statistisch, 'statistisch', 'Statistische Posten & Vortragskonten', 'Statistisch', 'Statistisch');
    }

    return null;
  }, [catalog, hierarchySection, accounts, accountsMap]);

  // Filtered entries for Detail Ledger
  const filteredLedgerEntries = useMemo(() => {
    if (!selectedLedger) return [];
    return selectedLedger.entries.filter((entry) => {
      const q = ledgerSearch.trim().toLowerCase();
      const matchesSearch =
        !q ||
        entry.booking.bookingNumber.toLowerCase().includes(q) ||
        entry.booking.description.toLowerCase().includes(q) ||
        entry.counterAccount.toLowerCase().includes(q) ||
        entry.counterName.toLowerCase().includes(q) ||
        (entry.booking.receiptNumber && entry.booking.receiptNumber.toLowerCase().includes(q));

      const matchesDir =
        ledgerDirFilter === 'all' || entry.direction.includes(ledgerDirFilter);

      return matchesSearch && matchesDir;
    });
  }, [selectedLedger, ledgerSearch, ledgerDirFilter]);

  // Export List or SuSa to CSV
  const exportAccountsCSV = () => {
    const headers = [
      'Kontonummer',
      'Kontenbezeichnung',
      'Kontenklasse',
      'Kategorie',
      'Unterkategorie',
      'Kontenart',
      'Bilanz-/GuV-Posten',
      'HGB-Code',
      'Steuersatz',
      'Hauptfunktion',
      'Umsatz Soll (EUR)',
      'Umsatz Haben (EUR)',
      'Saldo (EUR)',
      'Anzahl Buchungen',
    ];

    const rows = filteredAccounts.map((a) => [
      `"${a.number}"`,
      `"${a.name.replace(/"/g, '""')}"`,
      `"Klasse ${a.kontenklasse ?? 0}"`,
      `"${(a.category || '').replace(/"/g, '""')}"`,
      `"${(a.subcategory || '').replace(/"/g, '""')}"`,
      `"${a.type}"`,
      `"${(a.posten || '').replace(/"/g, '""')}"`,
      `"${a.hgbCode || ''}"`,
      `${a.taxRate || 0}`,
      `"${a.hauptfunktion || ''}"`,
      `${(a.debitSum || 0).toFixed(2)}`,
      `${(a.creditSum || 0).toFixed(2)}`,
      `${(a.balance || 0).toFixed(2)}`,
      `${a.bookingsCount || 0}`,
    ]);

    const csvContent = 'data:text/csv;charset=utf-8,\uFEFF' + [headers.join(';'), ...rows.map((r) => r.join(';'))].join('\n');
    const encodedUri = encodeURI(csvContent);
    const link = document.createElement('a');
    link.setAttribute('href', encodedUri);
    link.setAttribute('download', `SKR04_Kontenuebersicht_${new Date().toISOString().slice(0, 10)}.csv`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  const exportLedgerCSV = () => {
    if (!selectedLedger) return;
    const headers = [
      'Datum',
      'Buchungsnummer',
      'Buchungstext',
      'Gegenkonto',
      'Gegenkonto-Name',
      'Richtung',
      'Soll (EUR)',
      'Haben (EUR)',
      'Laufender Saldo (EUR)',
      'Steuerschlüssel',
      'Belegnummer',
      'Storno',
    ];

    const rows = selectedLedger.entries.map((e) => [
      `"${e.booking.date}"`,
      `"${e.booking.bookingNumber}"`,
      `"${e.booking.description.replace(/"/g, '""')}"`,
      `"${e.counterAccount}"`,
      `"${e.counterName.replace(/"/g, '""')}"`,
      `"${e.direction}"`,
      `${e.debitAmount.toFixed(2)}`,
      `${e.creditAmount.toFixed(2)}`,
      `${e.runningBalance.toFixed(2)}`,
      `"${e.booking.taxCode || ''}"`,
      `"${e.booking.receiptNumber || ''}"`,
      `"${e.booking.isStorno ? 'JA' : 'NEIN'}"`,
    ]);

    const csvContent =
      'data:text/csv;charset=utf-8,\uFEFF' +
      [
        `"Kontoblatt: ${selectedLedger.account.number} - ${selectedLedger.account.name}"`,
        `"Geschäftsjahr: ${selectedLedger.fiscalYear}"`,
        '',
        headers.join(';'),
        ...rows.map((r) => r.join(';')),
      ].join('\n');

    const encodedUri = encodeURI(csvContent);
    const link = document.createElement('a');
    link.setAttribute('href', encodedUri);
    link.setAttribute('download', `Kontoblatt_${selectedLedger.account.number}_${new Date().toISOString().slice(0, 10)}.csv`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  const getTypeBadge = (type: string) => {
    switch (type) {
      case 'asset':
        return <span className="px-2 py-0.5 rounded text-[10px] font-medium bg-emerald-100 text-emerald-800 border border-emerald-200/50 whitespace-nowrap">Aktiva / Vermögen</span>;
      case 'liability':
        return <span className="px-2 py-0.5 rounded text-[10px] font-medium bg-sky-100 text-sky-800 border border-sky-200/50 whitespace-nowrap">Passiva / Verb.</span>;
      case 'equity':
        return <span className="px-2 py-0.5 rounded text-[10px] font-medium bg-indigo-100 text-indigo-800 border border-indigo-200/50 whitespace-nowrap">Eigenkapital</span>;
      case 'revenue':
        return <span className="px-2 py-0.5 rounded text-[10px] font-medium bg-amber-100 text-amber-800 border border-amber-200/50 whitespace-nowrap">Ertrag / Einnahmen</span>;
      case 'expense':
        return <span className="px-2 py-0.5 rounded text-[10px] font-medium bg-rose-100 text-rose-800 border border-rose-200/50 whitespace-nowrap">Aufwand / Ausgaben</span>;
      case 'statistical':
        return <span className="px-2 py-0.5 rounded text-[10px] font-medium bg-stone-100 text-stone-700 border border-stone-200 whitespace-nowrap">Statistisch / Vortrag</span>;
      default:
        return null;
    }
  };

  const getStatementBadge = (stmt?: string, side?: string) => {
    if (!stmt) return null;
    if (stmt === 'Bilanz') {
      return (
        <span className="px-1.5 py-0.5 rounded text-[9px] font-mono font-medium bg-stone-100 text-stone-700 border border-stone-200 whitespace-nowrap">
          Bilanz ({side || 'HGB'})
        </span>
      );
    }
    if (stmt === 'GuV') {
      return (
        <span className="px-1.5 py-0.5 rounded text-[9px] font-mono font-medium bg-amber-50 text-amber-800 border border-amber-200/60 whitespace-nowrap">
          GuV (§ 275 HGB)
        </span>
      );
    }
    return (
      <span className="px-1.5 py-0.5 rounded text-[9px] font-mono text-stone-500 bg-stone-50 whitespace-nowrap">
        Statistik
      </span>
    );
  };

  // Helper to render Account Row in table
  const renderAccountRow = (acc: Account) => {
    const breadcrumb = [acc.category, acc.subcategory, acc.posten].filter(Boolean).join(' ▸ ');

    return (
      <tr
        key={acc.number}
        onClick={() => openAccountDetail(acc.number)}
        className="hover:bg-amber-50/40 transition-colors cursor-pointer group"
      >
        <td className="py-2.5 sm:py-3 px-3 sm:px-4 font-mono font-bold text-amber-800">
          {acc.number}
          {acc.isRange && (
            <span className="block text-[9px] font-normal text-stone-400">
              Bereich
            </span>
          )}
        </td>
        <td className="py-2.5 sm:py-3 px-3 sm:px-4 max-w-sm">
          <div className="font-semibold text-stone-900 group-hover:text-amber-900 flex items-center gap-1.5">
            <span>{acc.name}</span>
            {acc.bookingsCount ? (
              <span className="px-1.5 py-0.2 rounded text-[9px] font-mono bg-stone-100 text-stone-700 font-bold border border-stone-200 shrink-0">
                {acc.bookingsCount} {acc.bookingsCount === 1 ? 'Buchung' : 'Buchungen'}
              </span>
            ) : null}
          </div>
          {/* Detailed Context Breadcrumb to clarify duplicates (e.g. Darlehen, Privatentnahmen) */}
          <div className="text-[11px] text-stone-500 mt-0.5 flex flex-wrap items-center gap-1">
            {acc.hgbCode && (
              <span className="px-1 py-0.2 rounded text-[9px] font-mono font-semibold bg-stone-100 text-stone-600 border border-stone-200">
                {acc.hgbCode}
              </span>
            )}
            <span className="truncate max-w-md text-stone-500" title={breadcrumb}>
              {breadcrumb || acc.name}
            </span>
          </div>
        </td>
        <td className="py-2.5 sm:py-3 px-3 sm:px-4 max-w-xs hidden md:table-cell">
          <div className="font-medium text-stone-800 truncate">
            {acc.subcategory || acc.category || 'Standard'}
          </div>
          <div className="text-[11px] text-stone-400 truncate">
            {acc.posten || acc.statementType || ''}
          </div>
        </td>
        <td className="py-2.5 sm:py-3 px-3 sm:px-4 hidden sm:table-cell">{getTypeBadge(acc.type)}</td>
        <td className="py-2.5 sm:py-3 px-3 sm:px-4 text-center hidden lg:table-cell">
          {acc.taxRate > 0 ? (
            <span className="font-mono font-medium text-stone-700">
              {formatPercent(acc.taxRate)}
            </span>
          ) : acc.hauptfunktion ? (
            <span className="px-1.5 py-0.5 rounded text-[10px] font-mono bg-stone-100 text-stone-600 border border-stone-200">
              {acc.hauptfunktion}
            </span>
          ) : (
            <span className="text-stone-300">—</span>
          )}
        </td>
        <td className="py-2.5 sm:py-3 px-3 sm:px-4 text-right font-mono text-stone-700 hidden sm:table-cell">
          {acc.debitSum ? formatCurrency(acc.debitSum) : '—'}
        </td>
        <td className="py-2.5 sm:py-3 px-3 sm:px-4 text-right font-mono text-stone-700 hidden sm:table-cell">
          {acc.creditSum ? formatCurrency(acc.creditSum) : '—'}
        </td>
        <td
          className={`py-2.5 sm:py-3 px-3 sm:px-4 text-right font-mono font-bold ${
            acc.balance !== 0
              ? acc.balance > 0
                ? 'text-stone-900'
                : 'text-rose-700'
              : 'text-stone-400 font-normal'
          }`}
        >
          {formatCurrency(acc.balance)}
        </td>
        <td className="py-2.5 sm:py-3 px-3 sm:px-4 text-center">
          <button
            onClick={(e) => {
              e.stopPropagation();
              openAccountDetail(acc.number);
            }}
            className="inline-flex items-center gap-1 px-2 py-1 bg-stone-100 hover:bg-amber-100 text-stone-700 hover:text-amber-900 rounded text-[11px] font-medium transition-colors cursor-pointer"
          >
            <Eye className="w-3 h-3" />
            <span className="hidden sm:inline">Kontoblatt</span>
          </button>
        </td>
      </tr>
    );
  };

  // ---------------------------------------------------------------------------
  // DETAIL VIEW (KONTOBLATT / LEDGER)
  // ---------------------------------------------------------------------------
  if (viewMode === 'detail') {
    const acc = selectedLedger?.account;
    return (
      <div className="p-4 sm:p-6 md:p-8 max-w-7xl mx-auto space-y-4 sm:space-y-6">
        {/* Top Back Navigation Bar */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 pb-2 border-b border-stone-200">
          <div className="flex flex-wrap items-center gap-2 sm:gap-3">
            <button
              onClick={() => setViewMode('list')}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-white border border-stone-200 rounded-lg text-xs font-medium text-stone-700 hover:bg-stone-50 transition-colors shadow-2xs cursor-pointer"
            >
              <ArrowLeft className="w-3.5 h-3.5" />
              Zurück zur Kontenübersicht
            </button>
            <span className="text-stone-300 hidden sm:inline">|</span>
            <div className="flex items-center gap-1 text-xs text-stone-500">
              <button
                onClick={() => navigateAccount('prev')}
                className="px-2 py-1 rounded hover:bg-stone-200 text-stone-600 transition-colors font-medium text-[11px] cursor-pointer"
                title="Vorheriges Konto"
              >
                ◀ Vorheriges
              </button>
              <button
                onClick={() => navigateAccount('next')}
                className="px-2 py-1 rounded hover:bg-stone-200 text-stone-600 transition-colors font-medium text-[11px] cursor-pointer"
                title="Nächstes Konto"
              >
                Nächstes ▶
              </button>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={exportLedgerCSV}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-white border border-stone-200 rounded-lg text-xs font-medium text-stone-700 hover:bg-stone-50 transition-colors shadow-2xs cursor-pointer"
            >
              <Download className="w-3.5 h-3.5 text-stone-500" />
              Kontoblatt exportieren (CSV)
            </button>
          </div>
        </div>

        {/* Account Master Data Header */}
        {loadingLedger || !acc ? (
          <div className="bg-white p-8 rounded-xl border border-stone-200/80 shadow-xs text-center text-stone-400">
            <RefreshCw className="w-6 h-6 animate-spin mx-auto mb-2 text-amber-600" />
            Kontodaten und Buchungen werden geladen...
          </div>
        ) : (
          <>
            <div className="bg-white p-4 sm:p-6 rounded-2xl border border-stone-200/80 shadow-xs space-y-4">
              <div className="flex flex-col md:flex-row md:items-start justify-between gap-4">
                <div className="space-y-1.5">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="text-xl sm:text-2xl font-mono font-bold text-amber-800 bg-amber-50 px-2.5 py-0.5 rounded-lg border border-amber-200/60">
                      {acc.number}
                    </span>
                    <h1 className="text-xl sm:text-2xl font-bold text-stone-900 tracking-tight">
                      {acc.name}
                    </h1>
                    {getTypeBadge(acc.type)}
                    {getStatementBadge(acc.statementType, acc.balanceSide)}
                  </div>

                  <p className="text-xs text-stone-600 leading-relaxed max-w-4xl pt-1">
                    {acc.description || 'Offizielles DATEV SKR04 Konto für die ordnungsmäßige Buchführung.'}
                  </p>
                </div>

                <div className="text-left md:text-right shrink-0 bg-stone-50 p-3 rounded-xl border border-stone-200/60">
                  <div className="text-[11px] font-medium text-stone-500 uppercase tracking-wider">
                    Aktueller Saldo (GJ {selectedLedger.fiscalYear})
                  </div>
                  <div
                    className={`text-xl sm:text-2xl font-mono font-bold mt-0.5 ${
                      acc.balance >= 0 ? 'text-stone-900' : 'text-rose-600'
                    }`}
                  >
                    {formatCurrency(acc.balance)}
                  </div>
                  <div className="text-[10px] text-stone-400 mt-0.5">
                    {acc.type === 'asset' || acc.type === 'expense'
                      ? acc.balance >= 0
                        ? 'Sollsaldo (Aktiva/Aufwand)'
                        : 'Habensaldo'
                      : acc.balance >= 0
                      ? 'Habensaldo (Passiva/Ertrag)'
                      : 'Sollsaldo'}
                  </div>
                </div>
              </div>

              {/* Account Meta Badges */}
              <div className="pt-3 border-t border-stone-100 grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-2 sm:gap-3 text-xs">
                <div className="bg-stone-50/70 p-2.5 rounded-lg border border-stone-100">
                  <div className="text-[10px] text-stone-400 uppercase font-semibold flex items-center gap-1">
                    <Layers className="w-3 h-3 text-stone-400" /> Kontenklasse
                  </div>
                  <div className="font-semibold text-stone-800 mt-0.5 truncate">
                    {acc.kontenklasseName || `Klasse ${acc.kontenklasse ?? 0}`}
                  </div>
                </div>

                <div className="bg-stone-50/70 p-2.5 rounded-lg border border-stone-100">
                  <div className="text-[10px] text-stone-400 uppercase font-semibold flex items-center gap-1">
                    <Building2 className="w-3 h-3 text-stone-400" /> Bilanz-/GuV-Gliederung
                  </div>
                  <div className="font-semibold text-stone-800 mt-0.5 truncate" title={acc.posten}>
                    {acc.hgbCode ? `${acc.hgbCode} • ${acc.posten || acc.category}` : acc.posten || acc.category || 'Standard'}
                  </div>
                </div>

                <div className="bg-stone-50/70 p-2.5 rounded-lg border border-stone-100">
                  <div className="text-[10px] text-stone-400 uppercase font-semibold flex items-center gap-1">
                    <Tag className="w-3 h-3 text-stone-400" /> Steuerfunktion
                  </div>
                  <div className="font-semibold text-stone-800 mt-0.5 truncate">
                    {acc.hauptfunktion ? `${acc.hauptfunktion} (${acc.hauptfunktionDesc || 'Automatisch'})` : acc.taxRate > 0 ? `${formatPercent(acc.taxRate)} USt/VSt` : 'Ohne USt'}
                  </div>
                </div>

                <div className="bg-stone-50/70 p-2.5 rounded-lg border border-stone-100">
                  <div className="text-[10px] text-stone-400 uppercase font-semibold flex items-center gap-1">
                    <ShieldCheck className="w-3 h-3 text-emerald-600" /> Abschlusszweck
                  </div>
                  <div className="font-semibold text-stone-800 mt-0.5 truncate">
                    {acc.abschlusszweck ? `${acc.abschlusszweck} (Handels-/Steuerbilanz)` : 'HB / SB / EÜR'}
                  </div>
                </div>
              </div>
            </div>

            {/* KPI Cards */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-2 sm:gap-4">
              <div className="bg-white p-3 sm:p-4 rounded-xl border border-stone-200/80 shadow-2xs">
                <div className="text-[11px] sm:text-xs text-stone-500 font-medium">Umsatz SOLL</div>
                <div className="text-base sm:text-lg font-mono font-bold text-stone-900 mt-1">
                  {formatCurrency(selectedLedger.totalDebit)}
                </div>
                <div className="text-[10px] text-stone-400 mt-0.5 hidden sm:block">Summe aller Soll-Buchungen</div>
              </div>

              <div className="bg-white p-3 sm:p-4 rounded-xl border border-stone-200/80 shadow-2xs">
                <div className="text-[11px] sm:text-xs text-stone-500 font-medium">Umsatz HABEN</div>
                <div className="text-base sm:text-lg font-mono font-bold text-stone-900 mt-1">
                  {formatCurrency(selectedLedger.totalCredit)}
                </div>
                <div className="text-[10px] text-stone-400 mt-0.5 hidden sm:block">Summe aller Haben-Buchungen</div>
              </div>

              <div className="bg-white p-3 sm:p-4 rounded-xl border border-stone-200/80 shadow-2xs">
                <div className="text-[11px] sm:text-xs text-stone-500 font-medium">Endsaldo</div>
                <div className="text-base sm:text-lg font-mono font-bold text-amber-800 mt-1">
                  {formatCurrency(selectedLedger.closingBalance)}
                </div>
                <div className="text-[10px] text-stone-400 mt-0.5 hidden sm:block">Per Stichtag</div>
              </div>

              <div className="bg-white p-3 sm:p-4 rounded-xl border border-stone-200/80 shadow-2xs">
                <div className="text-[11px] sm:text-xs text-stone-500 font-medium">Buchungen</div>
                <div className="text-base sm:text-lg font-mono font-bold text-stone-900 mt-1">
                  {selectedLedger.bookingsCount}
                </div>
                <div className="text-[10px] text-stone-400 mt-0.5 hidden sm:block">Im Geschäftsjahr {selectedLedger.fiscalYear}</div>
              </div>
            </div>

            {/* Bookings Table Filter & Search */}
            <div className="bg-white p-3 sm:p-4 rounded-xl border border-stone-200/80 shadow-xs flex flex-col md:flex-row items-stretch md:items-center justify-between gap-3">
              <div className="relative flex-1">
                <Search className="w-4 h-4 text-stone-400 absolute left-3 top-1/2 -translate-y-1/2" />
                <input
                  type="text"
                  placeholder="Buchungen in diesem Konto durchsuchen..."
                  value={ledgerSearch}
                  onChange={(e) => setLedgerSearch(e.target.value)}
                  className="w-full pl-9 pr-4 py-2 text-xs bg-stone-50 border border-stone-200 rounded-lg focus:outline-hidden focus:ring-2 focus:ring-amber-500/20 focus:border-amber-500"
                />
              </div>

              <div className="flex items-center gap-1.5 overflow-x-auto text-xs">
                <span className="text-stone-400 flex items-center gap-1 text-[11px] mr-1">
                  <Filter className="w-3 h-3" /> Richtung:
                </span>
                {[
                  { id: 'all', label: 'Alle' },
                  { id: 'SOLL', label: 'Nur SOLL' },
                  { id: 'HABEN', label: 'Nur HABEN' },
                ].map((tab) => (
                  <button
                    key={tab.id}
                    onClick={() => setLedgerDirFilter(tab.id as any)}
                    className={`px-3 py-1.5 rounded-lg font-medium whitespace-nowrap transition-colors cursor-pointer ${
                      ledgerDirFilter === tab.id
                        ? 'bg-amber-700 text-white shadow-2xs'
                        : 'bg-stone-100 text-stone-600 hover:bg-stone-200/70'
                    }`}
                  >
                    {tab.label}
                  </button>
                ))}
              </div>
            </div>

            {/* Bookings Journal on this account */}
            <div className="bg-white rounded-xl border border-stone-200/80 shadow-xs overflow-hidden">
              <div className="px-4 py-3 bg-stone-50/80 border-b border-stone-200 flex items-center justify-between">
                <span className="text-xs font-bold text-stone-700 uppercase tracking-wider flex items-center gap-1.5">
                  <FileText className="w-3.5 h-3.5 text-amber-700" />
                  Kontoblatt & Einzelbuchungen ({filteredLedgerEntries.length})
                </span>
                <span className="text-[11px] text-stone-400 hidden sm:inline">
                  GoBD-konform chronologisch sortiert
                </span>
              </div>

              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs min-w-[700px]">
                  <thead className="bg-stone-50/50 border-b border-stone-200 text-stone-500 font-medium">
                    <tr>
                      <th className="py-2.5 px-3 sm:px-4 w-24">Datum</th>
                      <th className="py-2.5 px-3 sm:px-4 w-32">Buchungs-Nr.</th>
                      <th className="py-2.5 px-3 sm:px-4">Buchungstext & Beleg</th>
                      <th className="py-2.5 px-3 sm:px-4">Gegenkonto</th>
                      <th className="py-2.5 px-3 sm:px-4 text-center w-20">Richtung</th>
                      <th className="py-2.5 px-3 sm:px-4 text-right w-24 sm:w-28">Soll (€)</th>
                      <th className="py-2.5 px-3 sm:px-4 text-right w-24 sm:w-28">Haben (€)</th>
                      <th className="py-2.5 px-3 sm:px-4 text-right w-28 sm:w-32">Laufender Saldo</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-stone-100">
                    {filteredLedgerEntries.length === 0 ? (
                      <tr>
                        <td colSpan={8} className="py-12 text-center text-stone-400">
                          <BookOpen className="w-8 h-8 text-stone-300 mx-auto mb-2" />
                          Keine Buchungen für dieses Konto vorhanden.
                        </td>
                      </tr>
                    ) : (
                      filteredLedgerEntries.map((e, idx) => (
                        <tr
                          key={e.booking.id || idx}
                          className={`hover:bg-amber-50/30 transition-colors ${
                            e.booking.isStorno ? 'opacity-60 bg-rose-50/20' : ''
                          }`}
                        >
                          <td className="py-2.5 sm:py-3 px-3 sm:px-4 font-mono text-stone-600 whitespace-nowrap">
                            {e.booking.date}
                          </td>
                          <td className="py-2.5 sm:py-3 px-3 sm:px-4 font-mono font-medium text-stone-900 whitespace-nowrap">
                            {e.booking.bookingNumber}
                            {e.booking.isStorno && (
                              <span className="ml-1 px-1 py-0.2 rounded text-[9px] bg-rose-100 text-rose-800 font-semibold">
                                STORNO
                              </span>
                            )}
                          </td>
                          <td className="py-2.5 sm:py-3 px-3 sm:px-4">
                            <div className="font-medium text-stone-900">{e.booking.description}</div>
                            {e.booking.receiptNumber && (
                              <div className="text-[11px] text-stone-400 mt-0.5 flex items-center gap-1">
                                <FileText className="w-3 h-3 text-stone-400" /> Beleg: {e.booking.receiptNumber}
                              </div>
                            )}
                          </td>
                          <td className="py-2.5 sm:py-3 px-3 sm:px-4">
                            <button
                              onClick={() => openAccountDetail(e.counterAccount)}
                              className="text-left group cursor-pointer"
                              title="Gegenkonto öffnen"
                            >
                              <span className="font-mono font-bold text-amber-800 group-hover:underline">
                                {e.counterAccount}
                              </span>
                              <div className="text-[11px] text-stone-500 truncate max-w-[180px]">
                                {e.counterName || 'Gegenkonto'}
                              </div>
                            </button>
                          </td>
                          <td className="py-2.5 sm:py-3 px-3 sm:px-4 text-center">
                            <span
                              className={`px-2 py-0.5 rounded text-[10px] font-bold ${
                                e.direction === 'SOLL'
                                  ? 'bg-amber-100 text-amber-900'
                                  : 'bg-stone-200 text-stone-800'
                              }`}
                            >
                              {e.direction}
                            </span>
                          </td>
                          <td className="py-2.5 sm:py-3 px-3 sm:px-4 text-right font-mono text-stone-900">
                            {e.debitAmount > 0 ? formatCurrency(e.debitAmount) : '—'}
                          </td>
                          <td className="py-2.5 sm:py-3 px-3 sm:px-4 text-right font-mono text-stone-900">
                            {e.creditAmount > 0 ? formatCurrency(e.creditAmount) : '—'}
                          </td>
                          <td className="py-2.5 sm:py-3 px-3 sm:px-4 text-right font-mono font-bold text-stone-900 bg-stone-50/50">
                            {formatCurrency(e.runningBalance)}
                          </td>
                        </tr>
                      ))
                    )}
                  </tbody>
                  {filteredLedgerEntries.length > 0 && (
                    <tfoot className="bg-stone-50 font-bold border-t border-stone-200">
                      <tr>
                        <td colSpan={5} className="py-3 px-3 sm:px-4 text-stone-700">
                          Summen ({filteredLedgerEntries.length} Buchungen):
                        </td>
                        <td className="py-3 px-3 sm:px-4 text-right font-mono text-stone-900">
                          {formatCurrency(selectedLedger.totalDebit)}
                        </td>
                        <td className="py-3 px-3 sm:px-4 text-right font-mono text-stone-900">
                          {formatCurrency(selectedLedger.totalCredit)}
                        </td>
                        <td className="py-3 px-3 sm:px-4 text-right font-mono text-amber-800">
                          {formatCurrency(selectedLedger.closingBalance)}
                        </td>
                      </tr>
                    </tfoot>
                  )}
                </table>
              </div>
            </div>
          </>
        )}
      </div>
    );
  }

  // ---------------------------------------------------------------------------
  // MAIN OVERVIEW (KONTENLISTE, SUSA & HGB-STRUKTURBAUM)
  // ---------------------------------------------------------------------------
  return (
    <div className="p-3 sm:p-6 md:p-8 max-w-7xl mx-auto space-y-4 sm:space-y-6">
      {/* Header */}
      <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-4">
        <div>
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="text-xl sm:text-2xl font-bold text-stone-900 tracking-tight flex items-center">
              Kontenübersicht
              <HelpTooltip
                title="Kontenübersicht (SKR04 2026)"
                content="Offizieller DATEV SKR04 Kontenrahmen nach BilRUG. Bietet vollständige Erfassung aller Konten, Bilanz- und GuV-Zuordnungen, hierarchische HGB-Struktur, Kontoblätter und eine Soll/Haben-Übersicht (SuSa)."
              />
            </h2>
            <span className="px-2 py-0.5 rounded-full text-[10px] sm:text-[11px] font-mono font-semibold bg-amber-100 text-amber-900 border border-amber-200/60">
              DATEV SKR04 2026
            </span>
          </div>
          <p className="text-xs text-stone-500 mt-1">
            Vollständiger Kontenrahmen mit HGB-Bilanzzuordnung, Gliederungsbaum und Summen- & Saldenliste (SuSa)
          </p>
        </div>

        {/* View Mode Switcher (Tabs) */}
        <div className="flex flex-wrap items-center gap-2">
          <div className="flex flex-wrap items-center bg-stone-100 p-1 rounded-xl border border-stone-200/80 gap-1">
            <button
              onClick={() => setViewMode('list')}
              className={`flex items-center gap-1.5 px-2.5 sm:px-3 py-1.5 rounded-lg text-xs font-semibold transition-all cursor-pointer ${
                viewMode === 'list'
                  ? 'bg-white text-stone-900 shadow-xs'
                  : 'text-stone-600 hover:text-stone-900'
              }`}
            >
              <BookOpen className="w-3.5 h-3.5 text-amber-700" />
              Kontenliste ({filteredAccounts.length})
            </button>
            <button
              onClick={() => setViewMode('susa')}
              className={`flex items-center gap-1.5 px-2.5 sm:px-3 py-1.5 rounded-lg text-xs font-semibold transition-all cursor-pointer ${
                viewMode === 'susa'
                  ? 'bg-white text-stone-900 shadow-xs'
                  : 'text-stone-600 hover:text-stone-900'
              }`}
            >
              <Scale className="w-3.5 h-3.5 text-amber-700" />
              Soll / Haben (SuSa)
            </button>
            <button
              onClick={() => setViewMode('hierarchy')}
              className={`flex items-center gap-1.5 px-2.5 sm:px-3 py-1.5 rounded-lg text-xs font-semibold transition-all cursor-pointer ${
                viewMode === 'hierarchy'
                  ? 'bg-white text-stone-900 shadow-xs'
                  : 'text-stone-600 hover:text-stone-900'
              }`}
            >
              <FolderTree className="w-3.5 h-3.5 text-amber-700" />
              HGB-Strukturbaum
            </button>
          </div>

          <button
            onClick={exportAccountsCSV}
            className="inline-flex items-center gap-1.5 px-3 py-2 bg-white border border-stone-200 rounded-xl text-xs font-medium text-stone-700 hover:bg-stone-50 transition-colors shadow-2xs cursor-pointer ml-auto sm:ml-0"
            title="Kontenliste exportieren"
          >
            <Download className="w-3.5 h-3.5 text-stone-500" />
            CSV
          </button>
        </div>
      </div>

      {/* ------------------------------------------------------------- */}
      {/* TAB 1: KONTENLISTE (MIT KLASSEN- & UNTERGRUPPEN-STRUKTUR)      */}
      {/* ------------------------------------------------------------- */}
      {viewMode === 'list' && (
        <>
          {/* Filters Bar */}
          <div className="bg-white p-3 sm:p-4 rounded-xl border border-stone-200/80 shadow-xs space-y-3">
            <div className="flex flex-col md:flex-row items-stretch md:items-center justify-between gap-3">
              <div className="relative flex-1">
                <Search className="w-4 h-4 text-stone-400 absolute left-3 top-1/2 -translate-y-1/2" />
                <input
                  type="text"
                  placeholder="Konto suchen nach Nummer, Name, Unterkategorie, Posten oder HGB-Code..."
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  className="w-full pl-9 pr-4 py-2 text-xs bg-stone-50 border border-stone-200 rounded-lg focus:outline-hidden focus:ring-2 focus:ring-amber-500/20 focus:border-amber-500"
                />
              </div>

              {/* Status / Activity Filter */}
              <div className="flex items-center gap-1.5 overflow-x-auto text-xs pb-1 md:pb-0">
                <span className="text-stone-400 flex items-center gap-1 text-[11px] mr-1 shrink-0">
                  <SlidersHorizontal className="w-3 h-3" /> Status:
                </span>
                {[
                  { id: 'all', label: 'Alle' },
                  { id: 'has_turnover', label: 'Mit Umsatz' },
                  { id: 'active_balance', label: 'Mit Saldo != 0' },
                ].map((tab) => (
                  <button
                    key={tab.id}
                    onClick={() => setActivityFilter(tab.id as any)}
                    className={`px-2.5 sm:px-3 py-1.5 rounded-lg font-medium whitespace-nowrap transition-colors cursor-pointer text-[11px] sm:text-xs shrink-0 ${
                      activityFilter === tab.id
                        ? 'bg-amber-700 text-white shadow-2xs'
                        : 'bg-stone-100 text-stone-600 hover:bg-stone-200/70'
                    }`}
                  >
                    {tab.label}
                  </button>
                ))}
              </div>
            </div>

            {/* Structure Controls & Quick Jump Bar */}
            <div className="flex flex-wrap items-center justify-between gap-2 pt-2 border-t border-stone-100 text-xs">
              <div className="flex flex-wrap items-center gap-2">
                {/* Toggle Grouping */}
                <button
                  onClick={() => setGroupByClass(!groupByClass)}
                  className={`flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-[11px] font-medium transition-colors cursor-pointer ${
                    groupByClass
                      ? 'bg-amber-100 text-amber-900 border border-amber-200/60 font-semibold'
                      : 'bg-stone-100 text-stone-600 hover:bg-stone-200'
                  }`}
                >
                  <ListTree className="w-3.5 h-3.5 text-amber-800" />
                  Nach Klassen gruppieren
                </button>

                {/* Hide / Show Reserved */}
                <label className="flex items-center gap-1.5 text-[11px] text-stone-600 bg-stone-50 px-2.5 py-1 rounded-lg border border-stone-200 cursor-pointer select-none">
                  <input
                    type="checkbox"
                    checked={hideReserved}
                    onChange={(e) => setHideReserved(e.target.checked)}
                    className="rounded text-amber-700 focus:ring-amber-500"
                  />
                  <EyeOff className="w-3 h-3 text-stone-400" />
                  <span>Reservierte Konten ausblenden ({accounts.filter((a) => a.isReserved).length})</span>
                </label>
              </div>

              {/* Class Quick Jump Chips */}
              <div className="flex items-center gap-1 overflow-x-auto text-[11px]">
                <span className="text-stone-400 mr-0.5 hidden sm:inline">Klassen:</span>
                {[0, 1, 2, 3, 4, 5, 6, 7, 8, 9].map((k) => (
                  <button
                    key={k}
                    onClick={() => {
                      setClassFilter(classFilter === k.toString() ? 'all' : k.toString());
                    }}
                    className={`px-2 py-0.5 rounded font-mono font-medium transition-colors cursor-pointer ${
                      classFilter === k.toString()
                        ? 'bg-amber-800 text-white shadow-2xs font-bold'
                        : 'bg-stone-100 text-stone-600 hover:bg-stone-200'
                    }`}
                    title={classNames[k]}
                  >
                    Kl.{k}
                  </button>
                ))}
                {classFilter !== 'all' && (
                  <button
                    onClick={() => setClassFilter('all')}
                    className="px-1.5 py-0.5 text-[10px] text-stone-400 hover:text-stone-700 font-medium ml-1 cursor-pointer"
                  >
                    Reset
                  </button>
                )}
              </div>
            </div>

            {/* Type / Bereich Filter Bar */}
            <div className="flex flex-wrap items-center gap-1.5 pt-2 border-t border-stone-100 text-xs">
              <span className="text-stone-400 text-[11px] mr-1">Bereich:</span>
              {[
                { id: 'all', label: 'Alle Bereiche' },
                { id: 'asset', label: 'Vermögen (Aktiva)' },
                { id: 'liability', label: 'Verbindlichkeiten' },
                { id: 'equity', label: 'Eigenkapital' },
                { id: 'revenue', label: 'Erträge' },
                { id: 'expense', label: 'Aufwendungen' },
                { id: 'statistical', label: 'Statistisch' },
              ].map((tab) => (
                <button
                  key={tab.id}
                  onClick={() => setTypeFilter(tab.id)}
                  className={`px-2 py-0.5 rounded-md text-[11px] font-medium transition-colors cursor-pointer ${
                    typeFilter === tab.id
                      ? 'bg-stone-800 text-white shadow-2xs font-semibold'
                      : 'bg-stone-100 text-stone-600 hover:bg-stone-200'
                  }`}
                >
                  {tab.label}
                </button>
              ))}
            </div>

            {/* Expand / Collapse All for Grouped View */}
            {groupByClass && (
              <div className="flex items-center justify-between text-[11px] text-stone-500 pt-1">
                <span>
                  {accountsByClass.length} Kontenklassen dargestellt ({filteredAccounts.length} Konten gesamt)
                </span>
                <div className="flex items-center gap-2">
                  <button
                    onClick={expandAllListClasses}
                    className="text-stone-600 hover:text-stone-900 font-medium cursor-pointer"
                  >
                    Alle Klassen aufklappen
                  </button>
                  <span className="text-stone-300">•</span>
                  <button
                    onClick={collapseAllListClasses}
                    className="text-stone-600 hover:text-stone-900 font-medium cursor-pointer"
                  >
                    Alle Klassen zuklappen
                  </button>
                </div>
              </div>
            )}
          </div>

          {/* Grouped Structure Table vs Flat Table */}
          {groupByClass ? (
            <div className="space-y-4">
              {loading ? (
                <div className="bg-white p-12 rounded-xl border border-stone-200/80 shadow-xs text-center text-stone-400">
                  <RefreshCw className="w-6 h-6 animate-spin mx-auto mb-2 text-amber-600" />
                  Kontenübersicht wird geladen...
                </div>
              ) : accountsByClass.length === 0 ? (
                <div className="bg-white p-12 rounded-xl border border-stone-200/80 shadow-xs text-center text-stone-400">
                  Keine Konten gefunden für die gewählten Filterkriterien.
                </div>
              ) : (
                accountsByClass.map((group) => {
                  const isCollapsed = collapsedListClasses[group.kontenklasse];

                  return (
                    <div
                      key={group.kontenklasse}
                      className="bg-white rounded-xl border border-stone-200/80 shadow-xs overflow-hidden"
                    >
                      {/* Kontenklasse Section Header Banner */}
                      <div
                        onClick={() => toggleListClassCollapse(group.kontenklasse)}
                        className="p-3 sm:p-4 bg-stone-50/90 hover:bg-stone-100/90 transition-colors flex flex-col md:flex-row md:items-center justify-between gap-2 sm:gap-3 cursor-pointer select-none border-b border-stone-200"
                      >
                        <div className="flex items-center gap-2">
                          {isCollapsed ? (
                            <ChevronRight className="w-4 h-4 text-stone-400 shrink-0" />
                          ) : (
                            <ChevronDown className="w-4 h-4 text-stone-400 shrink-0" />
                          )}
                          <div>
                            <div className="font-bold text-stone-900 text-xs sm:text-sm flex items-center gap-2">
                              <span>{group.name}</span>
                              <span className="px-2 py-0.2 rounded-full text-[10px] font-mono bg-amber-100 text-amber-900 border border-amber-200/50">
                                {group.accounts.length} {group.accounts.length === 1 ? 'Konto' : 'Konten'}
                              </span>
                            </div>
                            <div className="text-[10px] sm:text-[11px] text-stone-400 mt-0.5">
                              {group.subcategories.length} Unterkategorien: {group.subcategories.map((s) => s.name).join(', ')}
                            </div>
                          </div>
                        </div>

                        <div className="flex flex-wrap items-center gap-2 sm:gap-4 text-xs font-mono font-medium justify-between md:justify-end">
                          <div className="text-right">
                            <span className="text-[9px] sm:text-[10px] text-stone-400 block font-sans">Umsatz Soll</span>
                            <span className="font-bold text-stone-900 text-[11px] sm:text-xs">
                              {formatCurrency(group.totalDebit)}
                            </span>
                          </div>
                          <div className="text-right">
                            <span className="text-[9px] sm:text-[10px] text-stone-400 block font-sans">Umsatz Haben</span>
                            <span className="font-bold text-stone-900 text-[11px] sm:text-xs">
                              {formatCurrency(group.totalCredit)}
                            </span>
                          </div>
                          <div className="text-right pl-2 border-l border-stone-200">
                            <span className="text-[9px] sm:text-[10px] text-stone-400 block font-sans">Klassensaldo</span>
                            <span
                              className={`font-bold text-[11px] sm:text-xs ${
                                group.balance !== 0
                                  ? group.balance > 0
                                    ? 'text-amber-900'
                                    : 'text-rose-700'
                                  : 'text-stone-400'
                              }`}
                            >
                              {formatCurrency(group.balance)}
                            </span>
                          </div>
                        </div>
                      </div>

                      {/* Konten Table inside this class */}
                      {!isCollapsed && (
                        <div className="overflow-x-auto">
                          <table className="w-full text-left text-xs min-w-[650px]">
                            <thead className="bg-stone-50/50 border-b border-stone-100 text-stone-400 font-medium">
                              <tr>
                                <th className="py-2.5 px-3 sm:px-4 w-24 sm:w-28">Kontonr.</th>
                                <th className="py-2.5 px-3 sm:px-4">Kontenbezeichnung & HGB-Pfad</th>
                                <th className="py-2.5 px-3 sm:px-4 hidden md:table-cell">Unterkategorie</th>
                                <th className="py-2.5 px-3 sm:px-4 hidden sm:table-cell">Bereich</th>
                                <th className="py-2.5 px-3 sm:px-4 text-center hidden lg:table-cell">Steuer</th>
                                <th className="py-2.5 px-3 sm:px-4 text-right hidden sm:table-cell">Umsatz Soll</th>
                                <th className="py-2.5 px-3 sm:px-4 text-right hidden sm:table-cell">Umsatz Haben</th>
                                <th className="py-2.5 px-3 sm:px-4 text-right">Saldo</th>
                                <th className="py-2.5 px-3 sm:px-4 text-center w-20 sm:w-24">Aktion</th>
                              </tr>
                            </thead>
                            <tbody className="divide-y divide-stone-100">
                              {group.accounts.map((acc) => renderAccountRow(acc))}
                            </tbody>
                          </table>
                        </div>
                      )}
                    </div>
                  );
                })
              )}
            </div>
          ) : (
            /* Flat Table when not grouped */
            <div className="bg-white rounded-xl border border-stone-200/80 shadow-xs overflow-hidden">
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs min-w-[650px]">
                  <thead className="bg-stone-50 border-b border-stone-200 text-stone-500 font-medium">
                    <tr>
                      <th className="py-2.5 sm:py-3 px-3 sm:px-4 w-24 sm:w-28">Kontonr.</th>
                      <th className="py-2.5 sm:py-3 px-3 sm:px-4">Kontenbezeichnung & HGB-Pfad</th>
                      <th className="py-2.5 sm:py-3 px-3 sm:px-4 hidden md:table-cell">Klasse & Unterkategorie</th>
                      <th className="py-2.5 sm:py-3 px-3 sm:px-4 hidden sm:table-cell">Bereich</th>
                      <th className="py-2.5 sm:py-3 px-3 sm:px-4 text-center hidden lg:table-cell">Steuer</th>
                      <th className="py-2.5 sm:py-3 px-3 sm:px-4 text-right hidden sm:table-cell">Umsatz Soll</th>
                      <th className="py-2.5 sm:py-3 px-3 sm:px-4 text-right hidden sm:table-cell">Umsatz Haben</th>
                      <th className="py-2.5 sm:py-3 px-3 sm:px-4 text-right">Saldo</th>
                      <th className="py-2.5 sm:py-3 px-3 sm:px-4 text-center w-20 sm:w-24">Aktion</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-stone-100">
                    {loading ? (
                      <tr>
                        <td colSpan={9} className="py-12 text-center text-stone-400">
                          <RefreshCw className="w-6 h-6 animate-spin mx-auto mb-2 text-amber-600" />
                          Kontenübersicht wird geladen...
                        </td>
                      </tr>
                    ) : filteredAccounts.length === 0 ? (
                      <tr>
                        <td colSpan={9} className="py-12 text-center text-stone-400">
                          Keine Konten gefunden für die gewählten Filterkriterien.
                        </td>
                      </tr>
                    ) : (
                      filteredAccounts.map((acc) => renderAccountRow(acc))
                    )}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </>
      )}

      {/* ------------------------------------------------------------- */}
      {/* TAB 2: SOLL / HABEN ÜBERSICHT (SUSA)                          */}
      {/* ------------------------------------------------------------- */}
      {viewMode === 'susa' && susaOverview && (
        <div className="space-y-4 sm:space-y-6">
          {/* SuSa Summary & GoBD Balance Banner */}
          <div className="bg-white p-4 sm:p-6 rounded-2xl border border-stone-200/80 shadow-xs space-y-4">
            <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
              <div className="space-y-1">
                <div className="flex items-center gap-2">
                  <h3 className="text-base sm:text-lg font-bold text-stone-900">
                    Summen- und Saldenliste (SuSa)
                  </h3>
                  <span className="px-2 py-0.5 rounded text-[10px] font-mono bg-stone-100 text-stone-700 font-bold border border-stone-200">
                    GJ {susaOverview.fiscalYear}
                  </span>
                </div>
                <p className="text-xs text-stone-500">
                  Gegenüberstellung aller Soll- und Haben-Umsätze sowie der Endsalden nach Kontenklassen 0 bis 9.
                </p>
              </div>

              {/* Balance Verification Check */}
              <div
                className={`flex items-center gap-2.5 px-3.5 py-2 rounded-xl border ${
                  susaOverview.isBalanced
                    ? 'bg-emerald-50/80 border-emerald-200 text-emerald-900'
                    : 'bg-rose-50 border-rose-200 text-rose-900'
                }`}
              >
                {susaOverview.isBalanced ? (
                  <CheckCircle2 className="w-5 h-5 text-emerald-600 shrink-0" />
                ) : (
                  <AlertCircle className="w-5 h-5 text-rose-600 shrink-0" />
                )}
                <div>
                  <div className="text-xs font-bold">
                    {susaOverview.isBalanced
                      ? 'Bilanzgleichgewicht ausgeglichen'
                      : 'Ungleichgewicht festgestellt'}
                  </div>
                  <div className="text-[11px] opacity-80">
                    Summe Soll = Summe Haben (Differenz: {formatCurrency(susaOverview.difference)})
                  </div>
                </div>
              </div>
            </div>

            {/* KPI Totals */}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-2 sm:gap-3 pt-3 border-t border-stone-100">
              <div className="bg-stone-50 p-2.5 sm:p-3 rounded-xl border border-stone-100">
                <div className="text-[10px] sm:text-[11px] text-stone-500 font-medium uppercase">
                  Gesamtsumme SOLL
                </div>
                <div className="text-base sm:text-lg font-mono font-bold text-stone-900 mt-0.5">
                  {formatCurrency(susaOverview.totalDebit)}
                </div>
              </div>

              <div className="bg-stone-50 p-2.5 sm:p-3 rounded-xl border border-stone-100">
                <div className="text-[10px] sm:text-[11px] text-stone-500 font-medium uppercase">
                  Gesamtsumme HABEN
                </div>
                <div className="text-base sm:text-lg font-mono font-bold text-stone-900 mt-0.5">
                  {formatCurrency(susaOverview.totalCredit)}
                </div>
              </div>

              <div className="bg-stone-50 p-2.5 sm:p-3 rounded-xl border border-stone-100">
                <div className="text-[10px] sm:text-[11px] text-stone-500 font-medium uppercase">
                  Summe SOLL-Salden
                </div>
                <div className="text-base sm:text-lg font-mono font-bold text-stone-900 mt-0.5">
                  {formatCurrency(susaOverview.totalSaldoDebit)}
                </div>
              </div>

              <div className="bg-stone-50 p-2.5 sm:p-3 rounded-xl border border-stone-100">
                <div className="text-[10px] sm:text-[11px] text-stone-500 font-medium uppercase">
                  Summe HABEN-Salden
                </div>
                <div className="text-base sm:text-lg font-mono font-bold text-stone-900 mt-0.5">
                  {formatCurrency(susaOverview.totalSaldoCredit)}
                </div>
              </div>
            </div>
          </div>

          {/* SuSa Filter Controls */}
          <div className="bg-white p-3 sm:p-4 rounded-xl border border-stone-200/80 shadow-xs flex flex-col md:flex-row items-stretch md:items-center justify-between gap-3">
            <div className="relative flex-1">
              <Search className="w-4 h-4 text-stone-400 absolute left-3 top-1/2 -translate-y-1/2" />
              <input
                type="text"
                placeholder="SuSa filtern nach Kontonummer oder Kontoname..."
                value={susaSearch}
                onChange={(e) => setSusaSearch(e.target.value)}
                className="w-full pl-9 pr-4 py-2 text-xs bg-stone-50 border border-stone-200 rounded-lg focus:outline-hidden focus:ring-2 focus:ring-amber-500/20 focus:border-amber-500"
              />
            </div>

            <div className="flex items-center gap-2">
              <label className="flex items-center gap-2 text-xs text-stone-700 bg-stone-50 px-3 py-2 rounded-lg border border-stone-200 cursor-pointer select-none">
                <input
                  type="checkbox"
                  checked={susaOnlyWithTurnover}
                  onChange={(e) => setSusaOnlyWithTurnover(e.target.checked)}
                  className="rounded text-amber-700 focus:ring-amber-500"
                />
                <span className="font-medium">Nur bebuchte Konten anzeigen</span>
              </label>
            </div>
          </div>

          {/* Grouped Classes Tables */}
          <div className="space-y-3 sm:space-y-4">
            {susaOverview.classes.map((cls) => {
              const matchingAccounts = cls.accounts.filter((a) => {
                const q = susaSearch.trim().toLowerCase();
                const matchesSearch =
                  !q ||
                  a.number.toLowerCase().includes(q) ||
                  a.name.toLowerCase().includes(q) ||
                  (a.posten && a.posten.toLowerCase().includes(q));

                if (susaOnlyWithTurnover) {
                  const hasUmsatz = (a.debitSum || 0) > 0 || (a.creditSum || 0) > 0 || (a.bookingsCount || 0) > 0;
                  return matchesSearch && hasUmsatz;
                }
                return matchesSearch;
              });

              if (susaOnlyWithTurnover && matchingAccounts.length === 0) {
                return null;
              }

              const isCollapsed = collapsedClasses[cls.kontenklasse];

              return (
                <div
                  key={cls.kontenklasse}
                  className="bg-white rounded-xl border border-stone-200/80 shadow-xs overflow-hidden"
                >
                  {/* Class Header Bar */}
                  <div
                    onClick={() => toggleClassCollapse(cls.kontenklasse)}
                    className="p-3 sm:p-4 bg-stone-50/80 hover:bg-stone-100/80 transition-colors flex flex-col md:flex-row md:items-center justify-between gap-2 sm:gap-3 cursor-pointer select-none border-b border-stone-200"
                  >
                    <div className="flex items-center gap-2">
                      {isCollapsed ? (
                        <ChevronRight className="w-4 h-4 text-stone-400 shrink-0" />
                      ) : (
                        <ChevronDown className="w-4 h-4 text-stone-400 shrink-0" />
                      )}
                      <div>
                        <div className="font-bold text-stone-900 text-xs sm:text-sm">
                          {cls.kontenklasseName}
                        </div>
                        <div className="text-[10px] sm:text-[11px] text-stone-400">
                          {matchingAccounts.length} {matchingAccounts.length === 1 ? 'Konto' : 'Konten'}
                        </div>
                      </div>
                    </div>

                    <div className="flex flex-wrap items-center gap-2 sm:gap-4 text-xs font-mono font-medium justify-between sm:justify-end">
                      <div className="text-right">
                        <span className="text-[9px] sm:text-[10px] text-stone-400 block font-sans">Soll</span>
                        <span className="font-bold text-stone-900 text-[11px] sm:text-xs">
                          {formatCurrency(cls.totalDebit)}
                        </span>
                      </div>
                      <div className="text-right">
                        <span className="text-[9px] sm:text-[10px] text-stone-400 block font-sans">Haben</span>
                        <span className="font-bold text-stone-900 text-[11px] sm:text-xs">
                          {formatCurrency(cls.totalCredit)}
                        </span>
                      </div>
                      <div className="text-right pl-2 border-l border-stone-200">
                        <span className="text-[9px] sm:text-[10px] text-stone-400 block font-sans">Saldo Soll</span>
                        <span className="font-bold text-amber-900 text-[11px] sm:text-xs">
                          {formatCurrency(cls.totalSaldoDebit)}
                        </span>
                      </div>
                      <div className="text-right">
                        <span className="text-[9px] sm:text-[10px] text-stone-400 block font-sans">Saldo Haben</span>
                        <span className="font-bold text-amber-900 text-[11px] sm:text-xs">
                          {formatCurrency(cls.totalSaldoCredit)}
                        </span>
                      </div>
                    </div>
                  </div>

                  {/* Class Accounts Table */}
                  {!isCollapsed && (
                    <div className="overflow-x-auto">
                      <table className="w-full text-left text-xs min-w-[650px]">
                        <thead className="bg-stone-50/50 border-b border-stone-100 text-stone-400 font-medium">
                          <tr>
                            <th className="py-2 px-3 sm:px-4 w-24 sm:w-28">Kontonr.</th>
                            <th className="py-2 px-3 sm:px-4">Kontenbezeichnung</th>
                            <th className="py-2 px-3 sm:px-4 hidden md:table-cell">Bilanz-/GuV-Posten</th>
                            <th className="py-2 px-3 sm:px-4 text-right w-24 sm:w-28">Umsatz Soll</th>
                            <th className="py-2 px-3 sm:px-4 text-right w-24 sm:w-28">Umsatz Haben</th>
                            <th className="py-2 px-3 sm:px-4 text-right w-24 sm:w-28">Saldo Soll</th>
                            <th className="py-2 px-3 sm:px-4 text-right w-24 sm:w-28">Saldo Haben</th>
                            <th className="py-2 px-3 sm:px-4 text-center w-16">Details</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-stone-100">
                          {matchingAccounts.length === 0 ? (
                            <tr>
                              <td colSpan={8} className="py-6 text-center text-stone-400">
                                Keine Konten in dieser Klasse gefunden.
                              </td>
                            </tr>
                          ) : (
                            matchingAccounts.map((acc) => {
                              const deb = acc.debitSum || 0;
                              const cred = acc.creditSum || 0;
                              const saldoDebit = deb > cred ? deb - cred : 0;
                              const saldoCredit = cred > deb ? cred - deb : 0;

                              return (
                                <tr
                                  key={acc.number}
                                  onClick={() => openAccountDetail(acc.number)}
                                  className="hover:bg-amber-50/40 transition-colors cursor-pointer group"
                                >
                                  <td className="py-2.5 px-3 sm:px-4 font-mono font-bold text-amber-800">
                                    {acc.number}
                                  </td>
                                  <td className="py-2.5 px-3 sm:px-4 font-medium text-stone-900 group-hover:text-amber-900">
                                    {acc.name}
                                  </td>
                                  <td className="py-2.5 px-3 sm:px-4 text-stone-500 truncate max-w-xs hidden md:table-cell">
                                    {acc.posten || acc.category || '—'}
                                  </td>
                                  <td className="py-2.5 px-3 sm:px-4 text-right font-mono text-stone-900">
                                    {deb > 0 ? formatCurrency(deb) : '—'}
                                  </td>
                                  <td className="py-2.5 px-3 sm:px-4 text-right font-mono text-stone-900">
                                    {cred > 0 ? formatCurrency(cred) : '—'}
                                  </td>
                                  <td className="py-2.5 px-3 sm:px-4 text-right font-mono font-bold text-stone-900 bg-stone-50/30">
                                    {saldoDebit > 0 ? formatCurrency(saldoDebit) : '—'}
                                  </td>
                                  <td className="py-2.5 px-3 sm:px-4 text-right font-mono font-bold text-stone-900 bg-stone-50/30">
                                    {saldoCredit > 0 ? formatCurrency(saldoCredit) : '—'}
                                  </td>
                                  <td className="py-2.5 px-3 sm:px-4 text-center">
                                    <button
                                      onClick={(e) => {
                                        e.stopPropagation();
                                        openAccountDetail(acc.number);
                                      }}
                                      className="p-1 rounded hover:bg-amber-100 text-stone-400 hover:text-amber-900 transition-colors cursor-pointer"
                                      title="Kontoblatt ansehen"
                                    >
                                      <Eye className="w-3.5 h-3.5" />
                                    </button>
                                  </td>
                                </tr>
                              );
                            })
                          )}
                        </tbody>
                        <tfoot className="bg-stone-50/70 border-t border-stone-200 text-xs font-bold font-mono">
                          <tr>
                            <td colSpan={2} className="py-2.5 px-3 sm:px-4 text-stone-700 font-sans">
                              Zwischensumme {cls.kontenklasseName}:
                            </td>
                            <td className="hidden md:table-cell"></td>
                            <td className="py-2.5 px-3 sm:px-4 text-right text-stone-900">
                              {formatCurrency(cls.totalDebit)}
                            </td>
                            <td className="py-2.5 px-3 sm:px-4 text-right text-stone-900">
                              {formatCurrency(cls.totalCredit)}
                            </td>
                            <td className="py-2.5 px-3 sm:px-4 text-right text-amber-900 bg-stone-100/50">
                              {formatCurrency(cls.totalSaldoDebit)}
                            </td>
                            <td className="py-2.5 px-3 sm:px-4 text-right text-amber-900 bg-stone-100/50">
                              {formatCurrency(cls.totalSaldoCredit)}
                            </td>
                            <td></td>
                          </tr>
                        </tfoot>
                      </table>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* ------------------------------------------------------------- */}
      {/* TAB 3: HGB-STRUKTURBAUM (BILANZ & GUV GLIEDERUNG)             */}
      {/* ------------------------------------------------------------- */}
      {viewMode === 'hierarchy' && hierarchyTree && (
        <div className="space-y-4 sm:space-y-6">
          {/* Section Selection Bar */}
          <div className="bg-white p-3 sm:p-4 rounded-xl border border-stone-200/80 shadow-xs space-y-3">
            <div className="flex flex-col md:flex-row items-stretch md:items-center justify-between gap-3">
              <div className="flex items-center gap-1.5 overflow-x-auto text-xs pb-1 md:pb-0">
                {[
                  { id: 'aktiva', label: 'Aktiva (Bilanz)', icon: <Building2 className="w-3.5 h-3.5" /> },
                  { id: 'passiva', label: 'Passiva (Bilanz)', icon: <Layers className="w-3.5 h-3.5" /> },
                  { id: 'guv', label: 'GuV (Ertrag & Aufwand)', icon: <TrendingUp className="w-3.5 h-3.5" /> },
                  { id: 'statistisch', label: 'Statistisch & Vorträge', icon: <FolderTree className="w-3.5 h-3.5" /> },
                ].map((sec) => (
                  <button
                    key={sec.id}
                    onClick={() => setHierarchySection(sec.id as any)}
                    className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg font-semibold whitespace-nowrap transition-colors cursor-pointer text-xs ${
                      hierarchySection === sec.id
                        ? 'bg-amber-700 text-white shadow-2xs'
                        : 'bg-stone-100 text-stone-700 hover:bg-stone-200/70'
                    }`}
                  >
                    {sec.icon}
                    {sec.label}
                  </button>
                ))}
              </div>

              <div className="flex items-center gap-2">
                <button
                  onClick={expandAllHierarchy}
                  className="px-2.5 py-1 text-[11px] bg-stone-100 hover:bg-stone-200 text-stone-700 rounded-lg font-medium transition-colors cursor-pointer"
                >
                  Alle aufklappen
                </button>
                <button
                  onClick={collapseAllHierarchy}
                  className="px-2.5 py-1 text-[11px] bg-stone-100 hover:bg-stone-200 text-stone-700 rounded-lg font-medium transition-colors cursor-pointer"
                >
                  Alle zuklappen
                </button>
              </div>
            </div>

            {/* Hierarchy Search & Bebucht-Filter */}
            <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-3 pt-2 border-t border-stone-100">
              <div className="relative flex-1">
                <Search className="w-4 h-4 text-stone-400 absolute left-3 top-1/2 -translate-y-1/2" />
                <input
                  type="text"
                  placeholder="HGB-Gliederung durchsuchen (Posten, HGB-Code, Konto)..."
                  value={hierarchySearch}
                  onChange={(e) => setHierarchySearch(e.target.value)}
                  className="w-full pl-9 pr-4 py-1.5 text-xs bg-stone-50 border border-stone-200 rounded-lg focus:outline-hidden focus:ring-2 focus:ring-amber-500/20 focus:border-amber-500"
                />
              </div>

              <label className="flex items-center gap-2 text-xs text-stone-700 bg-stone-50 px-3 py-1.5 rounded-lg border border-stone-200 cursor-pointer select-none shrink-0">
                <input
                  type="checkbox"
                  checked={hierarchyOnlyBebucht}
                  onChange={(e) => setHierarchyOnlyBebucht(e.target.checked)}
                  className="rounded text-amber-700 focus:ring-amber-500"
                />
                <span className="font-medium">Nur Posten mit Saldo anzeigen</span>
              </label>
            </div>
          </div>

          {/* Root Section Summary Banner */}
          <div className="bg-white p-4 sm:p-5 rounded-xl border border-stone-200/80 shadow-xs flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <div>
              <div className="text-xs text-stone-400 font-semibold uppercase tracking-wider">
                Gesamtsumme {hierarchyTree.name}
              </div>
              <div className="text-xl sm:text-2xl font-mono font-bold text-stone-900 mt-0.5">
                {formatCurrency(hierarchyTree.balance)}
              </div>
            </div>

            <div className="flex items-center gap-4 text-xs font-mono">
              <div className="bg-stone-50 p-2 rounded-lg border border-stone-100">
                <span className="text-[10px] text-stone-400 block">Summe Soll</span>
                <span className="font-bold text-stone-900">{formatCurrency(hierarchyTree.totalDebit)}</span>
              </div>
              <div className="bg-stone-50 p-2 rounded-lg border border-stone-100">
                <span className="text-[10px] text-stone-400 block">Summe Haben</span>
                <span className="font-bold text-stone-900">{formatCurrency(hierarchyTree.totalCredit)}</span>
              </div>
            </div>
          </div>

          {/* Render Main Groups & Subgroups Tree */}
          <div className="space-y-3 sm:space-y-4">
            {hierarchyTree.children?.map((mainGroup) => {
              const isMainExpanded = expandedNodes[mainGroup.id];

              return (
                <div
                  key={mainGroup.id}
                  className="bg-white rounded-xl border border-stone-200/80 shadow-xs overflow-hidden"
                >
                  {/* Main Group Header (e.g. A. Anlagevermögen) */}
                  <div
                    onClick={() => toggleNodeExpand(mainGroup.id)}
                    className="p-3 sm:p-4 bg-stone-100/70 hover:bg-stone-100 transition-colors flex items-center justify-between cursor-pointer select-none border-b border-stone-200"
                  >
                    <div className="flex items-center gap-2.5 min-w-0">
                      {isMainExpanded ? (
                        <FolderOpen className="w-4 h-4 text-amber-700 shrink-0" />
                      ) : (
                        <Folder className="w-4 h-4 text-stone-400 shrink-0" />
                      )}
                      <div className="min-w-0">
                        <div className="font-bold text-stone-900 text-xs sm:text-sm truncate">
                          {mainGroup.name}
                        </div>
                        <div className="text-[10px] sm:text-[11px] text-stone-400">
                          {mainGroup.children?.length || 0} Untergruppen • {mainGroup.accountsCount} Konten
                        </div>
                      </div>
                    </div>

                    <div className="text-right font-mono shrink-0 pl-2">
                      <span className="text-xs sm:text-sm font-bold text-stone-900">
                        {formatCurrency(mainGroup.balance)}
                      </span>
                    </div>
                  </div>

                  {/* Subgroups & Positions */}
                  {isMainExpanded && (
                    <div className="divide-y divide-stone-100">
                      {mainGroup.children?.map((subGroup) => {
                        const isSubExpanded = expandedNodes[subGroup.id];

                        return (
                          <div key={subGroup.id} className="bg-stone-50/30">
                            {/* Subgroup Header (e.g. II. Sachanlagen) */}
                            <div
                              onClick={() => toggleNodeExpand(subGroup.id)}
                              className="px-4 py-2.5 sm:px-6 hover:bg-amber-50/30 transition-colors flex items-center justify-between cursor-pointer select-none"
                            >
                              <div className="flex items-center gap-2 min-w-0">
                                {isSubExpanded ? (
                                  <ChevronDown className="w-3.5 h-3.5 text-stone-400 shrink-0" />
                                ) : (
                                  <ChevronRight className="w-3.5 h-3.5 text-stone-400 shrink-0" />
                                )}
                                <span className="font-semibold text-xs text-stone-800 truncate">
                                  {subGroup.name}
                                </span>
                              </div>

                              <span className="text-xs font-mono font-semibold text-stone-800 shrink-0 pl-2">
                                {formatCurrency(subGroup.balance)}
                              </span>
                            </div>

                            {/* Position Nodes & Contained Accounts */}
                            {isSubExpanded && (
                              <div className="px-4 pb-3 sm:px-6 space-y-2">
                                {subGroup.children?.map((pos) => {
                                  if (hierarchyOnlyBebucht && pos.balance === 0 && pos.totalDebit === 0 && pos.totalCredit === 0) {
                                    return null;
                                  }

                                  const q = hierarchySearch.trim().toLowerCase();
                                  if (
                                    q &&
                                    !pos.name.toLowerCase().includes(q) &&
                                    !(pos.hgbCode && pos.hgbCode.toLowerCase().includes(q)) &&
                                    !pos.accounts.some((a) => a.number.includes(q) || a.name.toLowerCase().includes(q))
                                  ) {
                                    return null;
                                  }

                                  return (
                                    <div
                                      key={pos.id}
                                      className="bg-white rounded-lg border border-stone-200/80 p-3 shadow-2xs space-y-2"
                                    >
                                      {/* Position Title & Code */}
                                      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-1 pb-1 border-b border-stone-100">
                                        <div className="flex items-center gap-1.5 min-w-0">
                                          {pos.hgbCode && (
                                            <span className="px-1.5 py-0.2 rounded text-[9px] font-mono font-bold bg-amber-50 text-amber-900 border border-amber-200/60 shrink-0">
                                              {pos.hgbCode}
                                            </span>
                                          )}
                                          <span className="font-medium text-xs text-stone-900 truncate">
                                            {pos.name}
                                          </span>
                                        </div>
                                        <div className="text-xs font-mono font-bold text-amber-900 shrink-0">
                                          Saldo: {formatCurrency(pos.balance)}
                                        </div>
                                      </div>

                                      {/* Accounts List for this Position */}
                                      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-1.5 pt-1">
                                        {pos.accounts.map((acc) => (
                                          <div
                                            key={acc.number}
                                            onClick={() => openAccountDetail(acc.number)}
                                            className="p-2 rounded-lg bg-stone-50/80 hover:bg-amber-50/60 border border-stone-100 transition-colors flex items-center justify-between cursor-pointer group"
                                          >
                                            <div className="min-w-0 pr-1">
                                              <div className="flex items-center gap-1">
                                                <span className="font-mono font-bold text-[11px] text-amber-800 group-hover:underline">
                                                  {acc.number}
                                                </span>
                                                {acc.hauptfunktion && (
                                                  <span className="text-[8px] font-mono px-1 rounded bg-stone-200/60 text-stone-600">
                                                    {acc.hauptfunktion}
                                                  </span>
                                                )}
                                              </div>
                                              <div className="text-[10px] text-stone-700 font-medium truncate max-w-[140px]">
                                                {acc.name}
                                              </div>
                                            </div>

                                            <div className="text-right font-mono shrink-0">
                                              <div className="text-[10px] font-bold text-stone-900">
                                                {formatCurrency(acc.balance)}
                                              </div>
                                              <div className="text-[8px] text-stone-400">
                                                {acc.bookingsCount || 0} Buchungen
                                              </div>
                                            </div>
                                          </div>
                                        ))}
                                      </div>
                                    </div>
                                  );
                                })}
                              </div>
                            )}
                          </div>
                        );
                      })}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
};
