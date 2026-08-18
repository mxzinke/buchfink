import React, { useEffect, useState } from 'react';
import {
  Calendar,
  Check,
  Clock,
  AlertTriangle,
  ReceiptText,
  Building2,
  FileSpreadsheet,
  CalendarDays,
} from 'lucide-react';
import { CompanySettings } from '../types';
import { Api } from '../services/api';
import { formatDate } from '../utils/formatters';
import { HelpTooltip } from '../components/HelpTooltip';

interface DeadlineItem {
  id: string;
  title: string;
  category: 'ust' | 'income' | 'trade' | 'annual';
  dueDate: string; // YYYY-MM-DD
  quarter: 1 | 2 | 3 | 4 | 5; // 1-4: Quarters, 5: Annual/Abschluss
  description: string;
  isImportant?: boolean;
}

export const DeadlinesPage: React.FC = () => {
  const [settings, setSettings] = useState<CompanySettings | null>(null);
  const [completedDeadlines, setCompletedDeadlines] = useState<string[]>([]);
  const [activeFilter, setActiveFilter] = useState<'all' | 'open' | 'overdue' | 'done'>('all');

  const currentYear = settings?.fiscalYear || new Date().getFullYear();

  useEffect(() => {
    loadSettings();
  }, []);

  const loadSettings = async () => {
    try {
      const cfg = await Api.getCompanySettings();
      setSettings(cfg);
      const year = cfg.fiscalYear || new Date().getFullYear();
      try {
        const saved = localStorage.getItem(`buchfink_deadlines_${year}`);
        if (saved) {
          setCompletedDeadlines(JSON.parse(saved));
        }
      } catch (e) {
        console.error(e);
      }
    } catch (e) {
      console.error(e);
    }
  };

  const toggleDeadline = (id: string) => {
    const year = settings?.fiscalYear || new Date().getFullYear();
    setCompletedDeadlines((prev) => {
      const updated = prev.includes(id) ? prev.filter((item) => item !== id) : [...prev, id];
      try {
        localStorage.setItem(`buchfink_deadlines_${year}`, JSON.stringify(updated));
      } catch (e) {
        console.error(e);
      }
      return updated;
    });
  };

  // Generate deadlines based on client configuration
  const generateDeadlines = (): DeadlineItem[] => {
    const y = currentYear;
    const nextY = y + 1;
    const list: DeadlineItem[] = [];

    const isSmallBusiness = settings?.isSmallBusiness;
    const vatPeriod = settings?.vatPeriod || 'quarter';

    // -------------------------------------------------------------
    // 1. QUARTAL (Januar – März)
    // -------------------------------------------------------------
    // Gewerbesteuer Q1
    list.push({
      id: `gewst_q1_${y}`,
      title: `Gewerbesteuer-Vorauszahlung (1. Quartal)`,
      category: 'trade',
      dueDate: `${y}-02-15`,
      quarter: 1,
      description: `1. Quartalsrate der Gewerbesteuer an die zuständige Stadt- oder Gemeindekasse.`,
    });

    // KSt / ESt Q1
    list.push({
      id: `est_q1_${y}`,
      title: `Körperschaft- / Einkommensteuer (1. Quartal)`,
      category: 'income',
      dueDate: `${y}-03-10`,
      quarter: 1,
      description: `1. Quartals-Vorauszahlung auf die Ertragsteuern an das Finanzamt.`,
    });

    // UStVA in Q1
    if (!isSmallBusiness) {
      if (vatPeriod === 'month') {
        list.push(
          {
            id: `dauerfrist_${y}`,
            title: `1/11 Sondervorauszahlung (Dauerfristverlängerung)`,
            category: 'ust',
            dueDate: `${y}-02-10`,
            quarter: 1,
            description: `Zahlung der 1/11 Sondervorauszahlung für die Fristverlängerung bei monatlicher Abgabe.`,
          },
          {
            id: `ustva_m1_${y}`,
            title: `USt-Voranmeldung Januar ${y}`,
            category: 'ust',
            dueDate: `${y}-02-10`,
            quarter: 1,
            description: `Monatliche Voranmeldung und Zahlung der Umsatzsteuer für Januar.`,
            isImportant: true,
          },
          {
            id: `ustva_m2_${y}`,
            title: `USt-Voranmeldung Februar ${y}`,
            category: 'ust',
            dueDate: `${y}-03-10`,
            quarter: 1,
            description: `Monatliche Voranmeldung und Zahlung der Umsatzsteuer für Februar.`,
            isImportant: true,
          }
        );
      }
    }

    // -------------------------------------------------------------
    // 2. QUARTAL (April – Juni)
    // -------------------------------------------------------------
    if (!isSmallBusiness) {
      if (vatPeriod === 'quarter') {
        list.push({
          id: `ustva_q1_${y}`,
          title: `USt-Voranmeldung 1. Quartal ${y}`,
          category: 'ust',
          dueDate: `${y}-04-10`,
          quarter: 2,
          description: `Voranmeldung für Januar bis März ${y} (bzw. 10.05. mit Dauerfristverlängerung).`,
          isImportant: true,
        });
      } else if (vatPeriod === 'month') {
        list.push(
          {
            id: `ustva_m3_${y}`,
            title: `USt-Voranmeldung März ${y}`,
            category: 'ust',
            dueDate: `${y}-04-10`,
            quarter: 2,
            description: `Monatliche Voranmeldung und Zahlung der Umsatzsteuer für März.`,
            isImportant: true,
          },
          {
            id: `ustva_m4_${y}`,
            title: `USt-Voranmeldung April ${y}`,
            category: 'ust',
            dueDate: `${y}-05-10`,
            quarter: 2,
            description: `Monatliche Voranmeldung und Zahlung der Umsatzsteuer für April.`,
            isImportant: true,
          },
          {
            id: `ustva_m5_${y}`,
            title: `USt-Voranmeldung Mai ${y}`,
            category: 'ust',
            dueDate: `${y}-06-10`,
            quarter: 2,
            description: `Monatliche Voranmeldung und Zahlung der Umsatzsteuer für Mai.`,
            isImportant: true,
          }
        );
      }
    }

    // Gewerbesteuer Q2
    list.push({
      id: `gewst_q2_${y}`,
      title: `Gewerbesteuer-Vorauszahlung (2. Quartal)`,
      category: 'trade',
      dueDate: `${y}-05-15`,
      quarter: 2,
      description: `2. Quartalsrate der Gewerbesteuer an die Stadt-/Gemeindekasse.`,
    });

    // KSt / ESt Q2
    list.push({
      id: `est_q2_${y}`,
      title: `Körperschaft- / Einkommensteuer (2. Quartal)`,
      category: 'income',
      dueDate: `${y}-06-10`,
      quarter: 2,
      description: `2. Quartals-Vorauszahlung auf die Ertragsteuern.`,
    });

    // -------------------------------------------------------------
    // 3. QUARTAL (Juli – September)
    // -------------------------------------------------------------
    if (!isSmallBusiness) {
      if (vatPeriod === 'quarter') {
        list.push({
          id: `ustva_q2_${y}`,
          title: `USt-Voranmeldung 2. Quartal ${y}`,
          category: 'ust',
          dueDate: `${y}-07-10`,
          quarter: 3,
          description: `Voranmeldung für April bis Juni ${y} (bzw. 10.08. mit Dauerfristverlängerung).`,
          isImportant: true,
        });
      } else if (vatPeriod === 'month') {
        list.push(
          {
            id: `ustva_m6_${y}`,
            title: `USt-Voranmeldung Juni ${y}`,
            category: 'ust',
            dueDate: `${y}-07-10`,
            quarter: 3,
            description: `Monatliche Voranmeldung und Zahlung der Umsatzsteuer für Juni.`,
            isImportant: true,
          },
          {
            id: `ustva_m7_${y}`,
            title: `USt-Voranmeldung Juli ${y}`,
            category: 'ust',
            dueDate: `${y}-08-10`,
            quarter: 3,
            description: `Monatliche Voranmeldung und Zahlung der Umsatzsteuer für Juli.`,
            isImportant: true,
          },
          {
            id: `ustva_m8_${y}`,
            title: `USt-Voranmeldung August ${y}`,
            category: 'ust',
            dueDate: `${y}-09-10`,
            quarter: 3,
            description: `Monatliche Voranmeldung und Zahlung der Umsatzsteuer für August.`,
            isImportant: true,
          }
        );
      }
    }

    // Gewerbesteuer Q3
    list.push({
      id: `gewst_q3_${y}`,
      title: `Gewerbesteuer-Vorauszahlung (3. Quartal)`,
      category: 'trade',
      dueDate: `${y}-08-15`,
      quarter: 3,
      description: `3. Quartalsrate der Gewerbesteuer an die Stadt-/Gemeindekasse.`,
    });

    // KSt / ESt Q3
    list.push({
      id: `est_q3_${y}`,
      title: `Körperschaft- / Einkommensteuer (3. Quartal)`,
      category: 'income',
      dueDate: `${y}-09-10`,
      quarter: 3,
      description: `3. Quartals-Vorauszahlung auf die Ertragsteuern.`,
    });

    // -------------------------------------------------------------
    // 4. QUARTAL (Oktober – Dezember & Jan Folgejahr)
    // -------------------------------------------------------------
    if (!isSmallBusiness) {
      if (vatPeriod === 'quarter') {
        list.push(
          {
            id: `ustva_q3_${y}`,
            title: `USt-Voranmeldung 3. Quartal ${y}`,
            category: 'ust',
            dueDate: `${y}-10-10`,
            quarter: 4,
            description: `Voranmeldung für Juli bis September ${y} (bzw. 10.11. mit Dauerfristverlängerung).`,
            isImportant: true,
          },
          {
            id: `ustva_q4_${y}`,
            title: `USt-Voranmeldung 4. Quartal ${y}`,
            category: 'ust',
            dueDate: `${nextY}-01-10`,
            quarter: 4,
            description: `Voranmeldung für Oktober bis Dezember ${y} (bzw. 10.02.${nextY} mit Dauerfristverlängerung).`,
            isImportant: true,
          }
        );
      } else if (vatPeriod === 'month') {
        list.push(
          {
            id: `ustva_m9_${y}`,
            title: `USt-Voranmeldung September ${y}`,
            category: 'ust',
            dueDate: `${y}-10-10`,
            quarter: 4,
            description: `Monatliche Voranmeldung und Zahlung der Umsatzsteuer für September.`,
            isImportant: true,
          },
          {
            id: `ustva_m10_${y}`,
            title: `USt-Voranmeldung Oktober ${y}`,
            category: 'ust',
            dueDate: `${y}-11-10`,
            quarter: 4,
            description: `Monatliche Voranmeldung und Zahlung der Umsatzsteuer für Oktober.`,
            isImportant: true,
          },
          {
            id: `ustva_m11_${y}`,
            title: `USt-Voranmeldung November ${y}`,
            category: 'ust',
            dueDate: `${y}-12-10`,
            quarter: 4,
            description: `Monatliche Voranmeldung und Zahlung der Umsatzsteuer für November.`,
            isImportant: true,
          },
          {
            id: `ustva_m12_${y}`,
            title: `USt-Voranmeldung Dezember ${y}`,
            category: 'ust',
            dueDate: `${nextY}-01-10`,
            quarter: 4,
            description: `Monatliche Voranmeldung und Zahlung der Umsatzsteuer für Dezember.`,
            isImportant: true,
          }
        );
      }
    }

    // Gewerbesteuer Q4
    list.push({
      id: `gewst_q4_${y}`,
      title: `Gewerbesteuer-Vorauszahlung (4. Quartal)`,
      category: 'trade',
      dueDate: `${y}-11-15`,
      quarter: 4,
      description: `4. Quartalsrate der Gewerbesteuer an die Stadt-/Gemeindekasse.`,
    });

    // KSt / ESt Q4
    list.push({
      id: `est_q4_${y}`,
      title: `Körperschaft- / Einkommensteuer (4. Quartal)`,
      category: 'income',
      dueDate: `${y}-12-10`,
      quarter: 4,
      description: `4. Quartals-Vorauszahlung auf die Ertragsteuern.`,
    });

    // -------------------------------------------------------------
    // 5. JAHRESABSCHLUSS & STEUERERKLÄRUNGEN (Folgejahr)
    // -------------------------------------------------------------
    list.push(
      {
        id: `annual_ust_${y}`,
        title: `Umsatzsteuer-Jahreserklärung ${y}`,
        category: 'annual',
        dueDate: `${nextY}-07-31`,
        quarter: 5,
        description: `Gesetzliche Abgabefrist der USt-Jahreserklärung (bei Beauftragung eines Steuerberaters: Ende Februar ${nextY + 1}).`,
        isImportant: true,
      },
      {
        id: `annual_ebilanz_${y}`,
        title: `E-Bilanz & Ertragsteuererklärungen ${y}`,
        category: 'annual',
        dueDate: `${nextY}-07-31`,
        quarter: 5,
        description: `Elektronische Übermittlung der Bilanz, GuV und Körperschaft-/Gewerbesteuererklärung an das Finanzamt.`,
        isImportant: true,
      }
    );

    return list.sort((a, b) => a.dueDate.localeCompare(b.dueDate));
  };

  const allDeadlines = generateDeadlines();

  // Status calculation helpers
  const today = new Date();
  today.setHours(0, 0, 0, 0);

  const getDeadlineStatus = (dueDateStr: string, isCompleted: boolean) => {
    if (isCompleted) return 'done';
    const due = new Date(dueDateStr);
    due.setHours(0, 0, 0, 0);
    const diffDays = Math.ceil((due.getTime() - today.getTime()) / (1000 * 60 * 60 * 24));

    if (diffDays < 0) return 'overdue';
    if (diffDays <= 30) return 'upcoming';
    return 'open';
  };

  // Summary counts
  const overdueCount = allDeadlines.filter(
    (d) => getDeadlineStatus(d.dueDate, completedDeadlines.includes(d.id)) === 'overdue'
  ).length;

  const upcomingCount = allDeadlines.filter(
    (d) => getDeadlineStatus(d.dueDate, completedDeadlines.includes(d.id)) === 'upcoming'
  ).length;

  const doneCount = allDeadlines.filter((d) => completedDeadlines.includes(d.id)).length;
  const openCount = allDeadlines.length - doneCount;

  // Filtered list
  const filteredList = allDeadlines.filter((item) => {
    const isCompleted = completedDeadlines.includes(item.id);
    const status = getDeadlineStatus(item.dueDate, isCompleted);

    if (activeFilter === 'open' && isCompleted) return false;
    if (activeFilter === 'overdue' && status !== 'overdue') return false;
    if (activeFilter === 'done' && !isCompleted) return false;
    return true;
  });

  // Grouping sections
  const sections: { key: number; title: string; subtitle: string }[] = [
    { key: 1, title: '1. Quartal (Januar – März)', subtitle: `Laufende Vorauszahlungen Q1 ${currentYear}` },
    { key: 2, title: '2. Quartal (April – Juni)', subtitle: `Laufende Vorauszahlungen Q2 ${currentYear}` },
    { key: 3, title: '3. Quartal (Juli – September)', subtitle: `Laufende Vorauszahlungen Q3 ${currentYear}` },
    { key: 4, title: '4. Quartal (Oktober – Dezember)', subtitle: `Laufende Vorauszahlungen Q4 ${currentYear}` },
    {
      key: 5,
      title: 'Jahresabschluss & Steuererklärungen',
      subtitle: `Fristen für die Jahressteuererklärungen und E-Bilanz ${currentYear}`,
    },
  ];

  const getCategoryIcon = (category: string) => {
    switch (category) {
      case 'ust':
        return <ReceiptText className="w-4 h-4 text-amber-700" />;
      case 'income':
        return <Building2 className="w-4 h-4 text-sky-700" />;
      case 'trade':
        return <Building2 className="w-4 h-4 text-stone-700" />;
      case 'annual':
        return <FileSpreadsheet className="w-4 h-4 text-emerald-700" />;
      default:
        return <Calendar className="w-4 h-4 text-stone-600" />;
    }
  };

  const renderStatusBadge = (dueDateStr: string, isCompleted: boolean) => {
    if (isCompleted) {
      return (
        <span className="px-2.5 py-1 rounded-full text-xs font-semibold bg-emerald-100 text-emerald-800 flex items-center gap-1">
          <Check className="w-3 h-3" /> Erledigt
        </span>
      );
    }

    const due = new Date(dueDateStr);
    due.setHours(0, 0, 0, 0);
    const diffDays = Math.ceil((due.getTime() - today.getTime()) / (1000 * 60 * 60 * 24));

    if (diffDays < 0) {
      return (
        <span className="px-2.5 py-1 rounded-full text-xs font-bold bg-rose-100 text-rose-800 border border-rose-200 flex items-center gap-1">
          <AlertTriangle className="w-3 h-3" /> Überfällig ({Math.abs(diffDays)} Tage)
        </span>
      );
    } else if (diffDays === 0) {
      return (
        <span className="px-2.5 py-1 rounded-full text-xs font-bold bg-rose-100 text-rose-800 border border-rose-200 animate-pulse">
          Heute fällig
        </span>
      );
    } else if (diffDays <= 30) {
      return (
        <span className="px-2.5 py-1 rounded-full text-xs font-semibold bg-amber-100 text-amber-900 border border-amber-200">
          In {diffDays} Tagen ({formatDate(dueDateStr)})
        </span>
      );
    } else {
      return (
        <span className="px-2.5 py-1 rounded-full text-xs font-medium bg-stone-100 text-stone-600">
          Fällig am {formatDate(dueDateStr)}
        </span>
      );
    }
  };

  return (
    <div className="p-8 max-w-6xl mx-auto space-y-7">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2.5">
            <h2 className="text-2xl font-bold text-stone-900 tracking-tight flex items-center gap-2">
              <CalendarDays className="w-6 h-6 text-amber-700" />
              Steuerfristen {currentYear}
              <HelpTooltip
                title="Steuerfristen"
                content="Gesetzliche Fristen für Umsatzsteuer-Voranmeldungen, Ertragsteuer-Vorauszahlungen und Jahressteuererklärungen. Nicht abgehakte, vergangene Fristen werden als überfällig markiert."
              />
            </h2>
            {settings?.isSmallBusiness && (
              <span className="px-2.5 py-0.5 rounded-full text-[11px] font-semibold bg-amber-100 text-amber-800">
                § 19 UStG (UStVA befreit)
              </span>
            )}
          </div>
          <p className="text-xs text-stone-500 mt-1">
            Übersichtliche Quartals- und Jahresfristen &bull; Termine einfach abhaken und im Blick behalten
          </p>
        </div>

        {/* Quick Filter Switcher */}
        <div className="flex bg-stone-100 p-1 rounded-xl border border-stone-200 text-xs">
          <button
            onClick={() => setActiveFilter('all')}
            className={`px-3 py-1.5 rounded-lg font-medium transition-all ${
              activeFilter === 'all'
                ? 'bg-amber-700 text-white shadow-xs font-semibold'
                : 'text-stone-600 hover:text-stone-900'
            }`}
          >
            Alle ({allDeadlines.length})
          </button>
          <button
            onClick={() => setActiveFilter('open')}
            className={`px-3 py-1.5 rounded-lg font-medium transition-all ${
              activeFilter === 'open'
                ? 'bg-amber-700 text-white shadow-xs font-semibold'
                : 'text-stone-600 hover:text-stone-900'
            }`}
          >
            Offen ({openCount})
          </button>
          {overdueCount > 0 && (
            <button
              onClick={() => setActiveFilter('overdue')}
              className={`px-3 py-1.5 rounded-lg font-medium transition-all flex items-center gap-1 ${
                activeFilter === 'overdue'
                  ? 'bg-rose-700 text-white shadow-xs font-semibold'
                  : 'text-rose-700 hover:bg-rose-50'
              }`}
            >
              <AlertTriangle className="w-3 h-3" />
              Überfällig ({overdueCount})
            </button>
          )}
          <button
            onClick={() => setActiveFilter('done')}
            className={`px-3 py-1.5 rounded-lg font-medium transition-all ${
              activeFilter === 'done'
                ? 'bg-emerald-700 text-white shadow-xs font-semibold'
                : 'text-stone-600 hover:text-stone-900'
            }`}
          >
            Erledigt ({doneCount})
          </button>
        </div>
      </div>

      {/* KPI Overview Strip */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <div className="bg-white p-4 rounded-xl border border-stone-200/80 shadow-xs space-y-1">
          <span className="text-[11px] uppercase tracking-wider text-stone-500 font-medium">
            Gesamte Termine
          </span>
          <div className="text-xl font-bold font-mono text-stone-900">{allDeadlines.length}</div>
          <span className="text-[11px] text-stone-400">Für {currentYear}</span>
        </div>

        <div
          className={`p-4 rounded-xl border shadow-xs space-y-1 ${
            overdueCount > 0
              ? 'bg-rose-50/70 border-rose-200 text-rose-950'
              : 'bg-white border-stone-200/80 text-stone-900'
          }`}
        >
          <span className="text-[11px] uppercase tracking-wider font-medium opacity-80">
            Überfällig
          </span>
          <div
            className={`text-xl font-bold font-mono ${
              overdueCount > 0 ? 'text-rose-700' : 'text-stone-900'
            }`}
          >
            {overdueCount}
          </div>
          <span className="text-[11px] opacity-70">
            {overdueCount > 0 ? 'Dringend prüfen' : 'Keine Rückstände'}
          </span>
        </div>

        <div className="bg-white p-4 rounded-xl border border-stone-200/80 shadow-xs space-y-1">
          <span className="text-[11px] uppercase tracking-wider text-amber-800 font-medium">
            Demnächst fällig
          </span>
          <div className="text-xl font-bold font-mono text-amber-800">{upcomingCount}</div>
          <span className="text-[11px] text-stone-400">In den nächsten 30 Tagen</span>
        </div>

        <div className="bg-white p-4 rounded-xl border border-stone-200/80 shadow-xs space-y-1">
          <span className="text-[11px] uppercase tracking-wider text-emerald-800 font-medium">
            Erledigt
          </span>
          <div className="text-xl font-bold font-mono text-emerald-700">{doneCount}</div>
          <span className="text-[11px] text-stone-400">Abgehakt</span>
        </div>
      </div>

      {/* Grouped by Quarter Sections */}
      <div className="space-y-6">
        {sections.map((section) => {
          const sectionDeadlines = filteredList.filter((d) => d.quarter === section.key);
          if (sectionDeadlines.length === 0) return null;

          return (
            <div
              key={section.key}
              className="bg-white rounded-2xl border border-stone-200/80 shadow-xs overflow-hidden"
            >
              {/* Section Header */}
              <div className="p-4 bg-stone-50/80 border-b border-stone-200/80 flex items-center justify-between">
                <div>
                  <h3 className="text-sm font-bold text-stone-900 flex items-center gap-2">
                    <span>{section.title}</span>
                  </h3>
                  <p className="text-[11px] text-stone-500 mt-0.5">{section.subtitle}</p>
                </div>
                <span className="text-xs font-mono text-stone-400">
                  {sectionDeadlines.length} {sectionDeadlines.length === 1 ? 'Termin' : 'Termine'}
                </span>
              </div>

              {/* Deadlines List */}
              <div className="divide-y divide-stone-100">
                {sectionDeadlines.map((item) => {
                  const isCompleted = completedDeadlines.includes(item.id);
                  const status = getDeadlineStatus(item.dueDate, isCompleted);
                  const isOverdue = status === 'overdue';

                  return (
                    <div
                      key={item.id}
                      className={`p-4 transition-colors flex flex-col sm:flex-row sm:items-center justify-between gap-3 ${
                        isCompleted
                          ? 'bg-stone-50/40 opacity-60'
                          : isOverdue
                          ? 'bg-rose-50/30'
                          : 'hover:bg-amber-50/20'
                      }`}
                    >
                      {/* Checkbox and Title */}
                      <div className="flex items-start gap-3.5 min-w-0">
                        <button
                          type="button"
                          onClick={() => toggleDeadline(item.id)}
                          className={`mt-0.5 w-5 h-5 rounded-md border flex items-center justify-center transition-colors shrink-0 ${
                            isCompleted
                              ? 'bg-emerald-600 border-emerald-600 text-white'
                              : isOverdue
                              ? 'border-rose-400 hover:border-rose-600 bg-white'
                              : 'border-stone-300 hover:border-amber-600 bg-white'
                          }`}
                          title={isCompleted ? 'Als offen markieren' : 'Als erledigt abhaken'}
                        >
                          {isCompleted && <Check className="w-3.5 h-3.5" />}
                        </button>

                        <div className="space-y-0.5 min-w-0">
                          <div className="flex items-center gap-2">
                            <span className="shrink-0">{getCategoryIcon(item.category)}</span>
                            <h4
                              className={`text-xs font-bold truncate ${
                                isCompleted
                                  ? 'line-through text-stone-500'
                                  : isOverdue
                                  ? 'text-rose-950 font-semibold'
                                  : 'text-stone-900'
                              }`}
                            >
                              {item.title}
                            </h4>
                            {item.isImportant && !isCompleted && !isOverdue && (
                              <span className="px-1.5 py-0.2 rounded text-[10px] font-semibold bg-amber-100 text-amber-900 border border-amber-200">
                                Wichtig
                              </span>
                            )}
                          </div>
                          <p className="text-xs text-stone-500 leading-snug">{item.description}</p>
                        </div>
                      </div>

                      {/* Status Badge */}
                      <div className="sm:text-right shrink-0 pl-8 sm:pl-0">
                        {renderStatusBadge(item.dueDate, isCompleted)}
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          );
        })}
      </div>

      {/* Info Tip Footer */}
      <div className="p-4 bg-stone-50 rounded-xl border border-stone-200/80 text-xs text-stone-600 space-y-1">
        <div className="font-semibold text-stone-800 flex items-center gap-1.5">
          <Clock className="w-3.5 h-3.5 text-amber-700" />
          Zahlungsschonfrist nach § 240 Abs. 3 AO:
        </div>
        <p className="text-[11px] leading-relaxed text-stone-500">
          Bei Banküberweisungen an das Finanzamt gilt eine gesetzliche Zahlungsschonfrist von 3 Tagen nach Fälligkeit. Fällt ein Fälligkeitstag auf ein Wochenende oder einen gesetzlichen Feiertag, verschiebt sich die Frist automatisch auf den nächsten Werktag.
        </p>
      </div>
    </div>
  );
};
