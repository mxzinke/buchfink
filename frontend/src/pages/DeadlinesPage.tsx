import React, { useEffect, useState } from 'react';
import { Lock } from 'lucide-react';
import { CompanySettings, Festschreibung, FoundationState } from '../types';
import { Api } from '../services/api';
import { formatDate } from '../utils/formatters';
import { FoundationDutyResetDialog, FoundationSection } from '../components/FoundationSection';
import {
  Button,
  Checkbox,
  ConfirmDialog,
  EmptyState,
  HelpPopover,
  HelpTooltip,
  PageHeader,
  Section,
  Stat,
  StatRow,
  StatusBadge,
  Table,
  Tabs,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  cn,
  toast,
} from '../components/ui';

/**
 * Steuerfristen.
 *
 * Die Termine entstehen aus der Mandantenkonfiguration: Der Rhythmus der
 * Voranmeldung entscheidet, ob monatlich oder quartalsweise abzugeben ist.
 * Abgehakt wird lokal — es ist eine Merkhilfe, keine Meldung ans Finanzamt.
 */

/** Die Kategorie steht als Wort, nicht als Farbe (§3.4). */
const CATEGORY_LABELS: Record<DeadlineItem['category'], string> = {
  gruendung: 'Gründung',
  ust: 'Umsatzsteuer',
  income: 'Ertragsteuer',
  trade: 'Gewerbesteuer',
  annual: 'Jahresabschluss',
};

type Filter = 'all' | 'open' | 'overdue' | 'done';

interface CommittablePeriod {
  type: 'month' | 'quarter';
  label: string;
  cutoff: string; // YYYY-MM-DD, last day of the period
}

interface DeadlineItem {
  id: string;
  title: string;
  category: 'gruendung' | 'ust' | 'income' | 'trade' | 'annual';
  dueDate: string; // YYYY-MM-DD; bei „unverzüglich" leer
  quarter: 0 | 1 | 2 | 3 | 4 | 5; // 0: Gründung, 1-4: Quartale, 5: Jahresabschluss
  description: string;
  isImportant?: boolean;
  /**
   * Bei einer Gründungspflicht steht hier ihr Schlüssel. Sie wird dann nicht im
   * `localStorage` abgehakt, sondern mit Datum in der Datenbank quittiert: dass
   * der Fragebogen übermittelt wurde, ist eine Tatsache über das Unternehmen
   * und keine Merkhilfe.
   */
  dutyKey?: string;
  /** Der Wortlaut, wo das Gesetz keine Tagesfrist nennt. */
  deadlineText?: string;
  doneOn?: string;
}

export const DeadlinesPage: React.FC = () => {
  const [settings, setSettings] = useState<CompanySettings | null>(null);
  const [completedDeadlines, setCompletedDeadlines] = useState<string[]>([]);
  const [activeFilter, setActiveFilter] = useState<'all' | 'open' | 'overdue' | 'done'>('all');
  const [festschreibungen, setFestschreibungen] = useState<Festschreibung[]>([]);
  const [committingCutoff, setCommittingCutoff] = useState<string | null>(null);
  const [foundation, setFoundation] = useState<FoundationState | null>(null);
  const [resettingDuty, setResettingDuty] = useState<DeadlineItem | null>(null);

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
      setFestschreibungen(await Api.getFestschreibungen());
      setFoundation(await Api.getFoundationState());
    } catch (e) {
      console.error(e);
    }
  };

  /**
   * Eine Gründungspflicht wird mit ihrem Datum quittiert, nicht mit einem Haken.
   * Zurückgenommen wird sie über die Rückfrage — ein Klick, der ein Datum
   * löscht, gehört nicht in eine Tabellenzeile.
   */
  const toggleDuty = async (item: DeadlineItem) => {
    if (!item.dutyKey) return;
    if (item.doneOn) {
      setResettingDuty(item);
      return;
    }
    try {
      await Api.completeFoundationDuty(item.dutyKey, new Date().toISOString().slice(0, 10));
      setFoundation(await Api.getFoundationState());
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    }
  };

  const resetDuty = async (item: DeadlineItem) => {
    if (!item.dutyKey) return;
    try {
      await Api.completeFoundationDuty(item.dutyKey, '');
      setFoundation(await Api.getFoundationState());
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    }
  };

  // The committable accounting periods of the fiscal year, derived from the VAT
  // reporting cadence, limited to periods that have already ended.
  const getCommittablePeriods = (): CommittablePeriod[] => {
    const y = currentYear;
    const todayIso = new Date().toISOString().slice(0, 10);
    const monthNames = ['Januar', 'Februar', 'März', 'April', 'Mai', 'Juni', 'Juli', 'August', 'September', 'Oktober', 'November', 'Dezember'];
    const lastDay = (year: number, month1: number) => new Date(year, month1, 0).getDate(); // month1 is 1-based
    const periods: CommittablePeriod[] = [];

    if ((settings?.vatPeriod || 'quarter') === 'month') {
      for (let m = 1; m <= 12; m++) {
        const cutoff = `${y}-${String(m).padStart(2, '0')}-${String(lastDay(y, m)).padStart(2, '0')}`;
        periods.push({ type: 'month', label: `${monthNames[m - 1]} ${y}`, cutoff });
      }
    } else {
      const quarters: [string, number][] = [['Q1', 3], ['Q2', 6], ['Q3', 9], ['Q4', 12]];
      for (const [label, endMonth] of quarters) {
        const cutoff = `${y}-${String(endMonth).padStart(2, '0')}-${String(lastDay(y, endMonth)).padStart(2, '0')}`;
        periods.push({ type: 'quarter', label: `${label} ${y}`, cutoff });
      }
    }
    return periods.filter((p) => p.cutoff <= todayIso);
  };

  const committedCutoffs = new Set(festschreibungen.map((f) => f.cutoffDate));
  const latestCommittedCutoff = festschreibungen.reduce((max, f) => (f.cutoffDate > max ? f.cutoffDate : max), '');

  const handleCommitPeriod = async (p: CommittablePeriod) => {
    setCommittingCutoff(p.cutoff);
    try {
      const rec = await Api.commitPeriod(p.type, p.label, p.cutoff);
      if (rec) {
        toast.success(
          rec.timestampStatus === 'confirmed'
            ? `${p.label} festgeschrieben, beglaubigt durch ${rec.tsaName}.`
            : `${p.label} festgeschrieben. Der Zeitstempel wird nachgeholt, sobald wieder Netz da ist.`,
        );
        setFestschreibungen(await Api.getFestschreibungen());
      }
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setCommittingCutoff(null);
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

    // -------------------------------------------------------------
    // 2. QUARTAL (April – Juni)
    // -------------------------------------------------------------
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

  /**
   * Die Pflichten aus der Gründung, als Termine derselben Liste.
   *
   * Sie stehen in einem eigenen Abschnitt vor dem ersten Quartal und tragen ihr
   * Erledigungsdatum statt eines Hakens. Fristen ohne Tagesangabe — das Gesetz
   * sagt dort „unverzüglich" — bekommen keine erfundene: sie sortieren ans Ende
   * ihres Abschnitts und nennen den Wortlaut.
   */
  const foundationDeadlines = (): DeadlineItem[] =>
    (foundation?.duties ?? []).map((duty) => ({
      id: `gruendung_${duty.key}`,
      title: duty.title,
      category: 'gruendung' as const,
      dueDate: duty.dueDate,
      quarter: 0 as const,
      description: `${duty.description} Frist: ${duty.deadline} (${duty.reference}).`,
      isImportant: true,
      dutyKey: duty.key,
      deadlineText: duty.deadline,
      doneOn: duty.doneOn,
    })).sort((a, b) => {
      if (a.dueDate === b.dueDate) return 0;
      if (!a.dueDate) return 1;
      if (!b.dueDate) return -1;
      return a.dueDate.localeCompare(b.dueDate);
    });

  const allDeadlines = [...foundationDeadlines(), ...generateDeadlines()];
  const [confirming, setConfirming] = useState<CommittablePeriod | null>(null);

  const today = new Date();
  today.setHours(0, 0, 0, 0);

  /** Tage bis zur Fälligkeit. Negativ heißt überfällig. */
  const daysUntil = (iso: string) => {
    const due = new Date(iso);
    due.setHours(0, 0, 0, 0);
    return Math.ceil((due.getTime() - today.getTime()) / (1000 * 60 * 60 * 24));
  };

  /**
   * Erledigt ist eine Gründungspflicht, wenn ihr Datum in der Datenbank steht;
   * ein gewöhnlicher Steuertermin, wenn der Haken im `localStorage` liegt.
   */
  const isDone = (item: DeadlineItem) =>
    item.dutyKey ? Boolean(item.doneOn) : completedDeadlines.includes(item.id);

  const statusOf = (item: DeadlineItem) => {
    if (isDone(item)) return 'done';
    // Ohne Tagesfrist gibt es kein Überfällig — „unverzüglich" lässt sich nicht
    // in Tagen messen.
    if (!item.dueDate) return 'open';
    const diff = daysUntil(item.dueDate);
    if (diff < 0) return 'overdue';
    if (diff <= 30) return 'upcoming';
    return 'open';
  };

  const doneCount = allDeadlines.filter(isDone).length;
  const openCount = allDeadlines.length - doneCount;
  const overdueCount = allDeadlines.filter((d) => statusOf(d) === 'overdue').length;
  const upcomingCount = allDeadlines.filter((d) => statusOf(d) === 'upcoming').length;

  const filtered = allDeadlines.filter((item) => {
    const status = statusOf(item);
    if (activeFilter === 'open' && status === 'done') return false;
    if (activeFilter === 'overdue' && status !== 'overdue') return false;
    if (activeFilter === 'done' && status !== 'done') return false;
    return true;
  });

  const sections: { key: number; title: string }[] = [
    { key: 0, title: 'Gründung und Anmeldung' },
    { key: 1, title: `1. Quartal ${currentYear} · Januar bis März` },
    { key: 2, title: `2. Quartal ${currentYear} · April bis Juni` },
    { key: 3, title: `3. Quartal ${currentYear} · Juli bis September` },
    { key: 4, title: `4. Quartal ${currentYear} · Oktober bis Dezember` },
    { key: 5, title: 'Jahresabschluss und Steuererklärungen' },
  ];

  const periods = getCommittablePeriods();

  return (
    <div className="max-w-[1200px] mx-auto px-8 py-8">
      <PageHeader
        title={`Steuerfristen ${currentYear}`}
        context="Voranmeldungen, Vorauszahlungen und Jahreserklärungen"
        action={
          <HelpPopover label="Erklärung zu den Steuerfristen">
            Die Termine ergeben sich aus dem Rhythmus der Voranmeldung. Abgehakt wird lokal, als
            Merkhilfe. Bei Überweisung an das Finanzamt gilt die Zahlungsschonfrist von drei Tagen
            nach § 240 Abs. 3 AO; fällt ein Fälligkeitstag auf ein Wochenende oder einen Feiertag,
            verschiebt er sich auf den nächsten Werktag.
          </HelpPopover>
        }
      />

      <div className="mt-6">
        <StatRow>
          <Stat label="Termine" value={String(allDeadlines.length)} context={`im Jahr ${currentYear}`} />
          <Stat
            label="Überfällig"
            value={String(overdueCount)}
            context={overdueCount > 0 ? 'ohne Haken in der Vergangenheit' : 'keine Rückstände'}
            tone={overdueCount > 0 ? 'negative' : 'neutral'}
          />
          <Stat
            label="Demnächst fällig"
            value={String(upcomingCount)}
            context="in den nächsten 30 Tagen"
          />
          <Stat label="Erledigt" value={String(doneCount)} context={`${openCount} noch offen`} />
        </StatRow>
      </div>

      {foundation?.applies && foundation.hasFoundation && foundation.stage === 'vorgesellschaft' && (
        <FoundationSection state={foundation} onChanged={loadSettings} />
      )}

      <Section
        title="Zeiträume festschreiben"
        context="Abschluss passend zum Rhythmus der Voranmeldung"
        action={
          <HelpPopover label="Erklärung zur Festschreibung">
            Ein festgeschriebener Zeitraum nimmt keine rückdatierten Buchungen mehr an, Korrekturen
            laufen ab dann über den Storno. Zusätzlich beglaubigt ein unabhängiger Zeitstempeldienst
            den Stand — übertragen wird dabei nur eine Prüfsumme, keine Buchung.
          </HelpPopover>
        }
      >
        {periods.length === 0 ? (
          <EmptyState
            icon={<Lock className="w-6 h-6" strokeWidth={1.5} />}
            title="Noch kein abgeschlossener Zeitraum"
            description="Zeiträume lassen sich festschreiben, sobald sie beendet sind."
          />
        ) : (
          <Table>
            <Thead>
              <Tr>
                <Th>Zeitraum</Th>
                <Th className="w-40">Stichtag</Th>
                <Th className="w-56" aria-label="Zustand" />
              </Tr>
            </Thead>
            <Tbody>
              {periods.map((period) => {
                const committed = committedCutoffs.has(period.cutoff);
                const busy = committingCutoff === period.cutoff;
                // Festgeschrieben wird der Reihe nach. Ein Zeitraum mit Lücke
                // davor wäre kein Abschluss.
                const isNext = !committed && period.cutoff > latestCommittedCutoff;
                return (
                  <Tr key={period.cutoff}>
                    <Td>{period.label}</Td>
                    <Td className="num text-ink-subtle">{formatDate(period.cutoff)}</Td>
                    <Td className="text-right">
                      {committed ? (
                        <StatusBadge status="festgeschrieben" />
                      ) : (
                        <Button
                          variant="secondary"
                          size="sm"
                          loading={busy}
                          disabled={!isNext}
                          title={!isNext ? 'Frühere Zeiträume zuerst festschreiben' : undefined}
                          onClick={() => setConfirming(period)}
                          icon={<Lock className="w-3.5 h-3.5" strokeWidth={1.5} />}
                        >
                          Festschreiben
                        </Button>
                      )}
                    </Td>
                  </Tr>
                );
              })}
            </Tbody>
          </Table>
        )}
      </Section>

      <Section title="Termine" context="Abhaken merkt sich Buchfink lokal">
        <Tabs
          items={[
            { value: 'all' as Filter, label: 'Alle', count: allDeadlines.length },
            { value: 'open' as Filter, label: 'Offen', count: openCount },
            { value: 'overdue' as Filter, label: 'Überfällig', count: overdueCount },
            { value: 'done' as Filter, label: 'Erledigt', count: doneCount },
          ]}
          value={activeFilter}
          onValueChange={setActiveFilter}
          className="mb-6"
        />

        {filtered.length === 0 ? (
          <EmptyState title="Kein Termin in dieser Auswahl" />
        ) : (
          <Table>
            <Thead>
              <Tr>
                <Th className="w-10" aria-label="Erledigt" />
                <Th>Termin</Th>
                <Th className="w-40">Art</Th>
                <Th className="w-32">Fällig am</Th>
                <Th className="w-52">Zustand</Th>
              </Tr>
            </Thead>
            <Tbody>
              {sections.map((section) => {
                const items = filtered.filter((d) => d.quarter === section.key);
                if (items.length === 0) return null;
                return (
                  <React.Fragment key={section.key}>
                    <Tr>
                      <Td colSpan={5} className="text-overline uppercase text-ink-subtle">
                        {section.title}
                      </Td>
                    </Tr>
                    {items.map((item) => {
                      const status = statusOf(item);
                      const done = status === 'done';
                      const diff = item.dueDate ? daysUntil(item.dueDate) : null;
                      return (
                        <Tr key={item.id}>
                          <Td className="pr-0">
                            <Checkbox
                              checked={done}
                              onCheckedChange={() =>
                                item.dutyKey ? void toggleDuty(item) : toggleDeadline(item.id)
                              }
                              label={
                                <span className="sr-only">
                                  {done ? 'Als offen markieren' : 'Als erledigt abhaken'}
                                </span>
                              }
                            />
                          </Td>
                          <Td className="max-w-[30rem]">
                            <span className="flex items-center gap-1">
                              {/* Eine erledigte Gründungspflicht wird nicht
                                  durchgestrichen: sie ist ein Vorgang mit Datum
                                  und bleibt lesbar (§11.2). */}
                              <span
                                className={cn(
                                  'truncate',
                                  done && 'text-ink-subtle',
                                  done && !item.dutyKey && 'line-through',
                                )}
                              >
                                {item.title}
                              </span>
                              <HelpTooltip
                                label={`Erklärung zu ${item.title}`}
                                content={item.description}
                              />
                              {item.isImportant && !done && (
                                <span className="text-caption text-attention-text">wichtig</span>
                              )}
                            </span>
                          </Td>
                          <Td className="text-ink-muted">{CATEGORY_LABELS[item.category]}</Td>
                          <Td className={cn(item.dueDate ? 'num text-ink-subtle' : 'text-ink-subtle')}>
                            {item.dueDate ? formatDate(item.dueDate) : (item.deadlineText ?? '—')}
                          </Td>
                          <Td>
                            {done ? (
                              <span className="text-caption text-ink-subtle">
                                {item.doneOn ? `Erledigt am ${formatDate(item.doneOn)}` : 'Erledigt'}
                              </span>
                            ) : diff === null ? (
                              <span className="text-caption text-ink-subtle">Offen</span>
                            ) : diff < 0 ? (
                              <span className="flex items-center gap-2">
                                <StatusBadge status="ueberfaellig" />
                                <span className="text-caption text-ink-subtle num">
                                  seit {Math.abs(diff)} Tagen
                                </span>
                              </span>
                            ) : diff === 0 ? (
                              <span className="text-caption text-attention-text">Heute fällig</span>
                            ) : diff <= 30 ? (
                              <span className="text-caption text-attention-text num">
                                in {diff} Tagen
                              </span>
                            ) : (
                              <span className="text-caption text-ink-subtle">Offen</span>
                            )}
                          </Td>
                        </Tr>
                      );
                    })}
                  </React.Fragment>
                );
              })}
            </Tbody>
          </Table>
        )}
      </Section>

      <FoundationDutyResetDialog
        title={resettingDuty?.title ?? null}
        onOpenChange={(next) => !next && setResettingDuty(null)}
        onConfirm={() => {
          const item = resettingDuty;
          setResettingDuty(null);
          if (item) void resetDuty(item);
        }}
      />

      <ConfirmDialog
        open={confirming !== null}
        onOpenChange={(next) => !next && setConfirming(null)}
        title={`${confirming?.label ?? ''} festschreiben`}
        description="Danach nimmt der Zeitraum keine rückdatierten Buchungen mehr an. Korrekturen laufen ab dann über den Storno."
        confirmLabel="Festschreiben"
        onConfirm={() => {
          const period = confirming;
          setConfirming(null);
          if (period) void handleCommitPeriod(period);
        }}
      />
    </div>
  );
};
