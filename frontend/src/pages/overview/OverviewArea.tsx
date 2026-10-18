import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Api } from '../../services/api';
import type {
  AgentBookings,
  ConnectedApp,
  MCPServerStatus,
  MonthCloseState,
  Obligation,
  TaskList,
} from '../../types';
import { formatDate } from '../../utils/formatters';
import { agentWorkSummary, monthToClose, planOverview, todayISO } from '../../utils/overview';
import { profileGaps, profileTask } from '../../utils/profileGaps';
import { useNavigation } from '../../navigation';
import { useWorkspace } from '../../workspace';
import { useLiveReload } from '../../live';
import { PageFrame } from '../../components/ui';
import { DutiesSection } from './DutiesSection';
import { MonthCloseSection } from './MonthCloseSection';
import { AgentBookingsSection } from './AgentBookingsSection';
import { AppsSection } from './AppsSection';
import { InboxSection } from './InboxSection';
import { ObligationPanel } from './ObligationPanel';

/** So viele Buchungen der Apps zeigt die Übersicht; der Rest steht im Journal. */
const AGENT_BOOKINGS_SHOWN = 8;

function message(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

/** Ein Ladefehler bleibt im Abschnitt stehen und tritt an die Stelle des Leerzustands. */
function loadFailed(error: unknown): string {
  return `Konnte nicht geladen werden. ${message(error)}`;
}

/**
 * Die Startseite: was jetzt zu tun ist, was im Eingang liegt, der anstehende
 * Monatsabschluss, was die verbundenen Apps seit der letzten Festschreibung
 * gebucht haben, und ob sie verbunden sind. Gebucht wird hier nichts;
 * bestätigt wird nur, was allein der Mensch bestätigen kann. Eine Pflicht
 * öffnet sich über das Ziel `#/overview/<schlüssel>` in der Seitenleiste.
 * Ändert ein Agent etwas, lädt die Seite neu, ohne den Stand zu verlieren.
 */
export const OverviewArea: React.FC = () => {
  const { route, navigate } = useNavigation();
  const { year, activeTenant, settings, availableYears } = useWorkspace();
  const obligationKey = route.area === 'overview' ? route.id : undefined;

  const [tasks, setTasks] = useState<TaskList | null>(null);
  const [obligations, setObligations] = useState<Obligation[] | null>(null);
  const [error, setError] = useState('');
  const [bookings, setBookings] = useState<AgentBookings | null>(null);
  const [bookingsError, setBookingsError] = useState('');
  const [apps, setApps] = useState<ConnectedApp[] | null>(null);
  const [appsError, setAppsError] = useState('');
  const [server, setServer] = useState<MCPServerStatus | null>(null);
  const [monthState, setMonthState] = useState<MonthCloseState | null>(null);
  const [monthError, setMonthError] = useState('');
  const [profileFacts, setProfileFacts] = useState<{ bankAccounts: number; hasFoundation: boolean } | null>(
    null,
  );

  const load = useCallback(async () => {
    try {
      const [nextTasks, nextObligations] = await Promise.all([
        Api.getTasks(),
        Api.getNextObligations(),
      ]);
      setTasks(nextTasks);
      setObligations(nextObligations);
      setError('');
    } catch (e) {
      // Kein Leerzustand: „Nichts fällig" wäre eine Entwarnung, die niemand gegeben hat.
      setTasks(null);
      setObligations(null);
      setError(loadFailed(e));
    }
    Api.getAgentBookings(AGENT_BOOKINGS_SHOWN)
      .then((next) => {
        setBookings(next);
        setBookingsError('');
      })
      .catch((e) => {
        setBookings(null);
        setBookingsError(loadFailed(e));
      });
  }, []);

  useEffect(() => {
    void load();
  }, [load, year, activeTenant?.id]);

  // Für die Aufgabe „Stammdaten vervollständigen“: was die Einstellungen allein nicht wissen.
  useEffect(() => {
    let current = true;
    Promise.all([Api.getBankAccounts(), Api.getFoundationState().catch(() => null)])
      .then(([banks, foundation]) => {
        if (!current) return;
        setProfileFacts({
          bankAccounts: banks.length,
          hasFoundation: Boolean(foundation?.hasFoundation && foundation.foundation),
        });
      })
      .catch(() => current && setProfileFacts(null));
    return () => {
      current = false;
    };
  }, [activeTenant?.id, settings]);

  useEffect(() => {
    Promise.all([Api.getConnectedApps(), Api.getMCPServerStatus()])
      .then(([nextApps, status]) => {
        setApps(nextApps.filter((app) => !app.revoked));
        setServer(status);
        setAppsError('');
      })
      .catch((e) => setAppsError(loadFailed(e)));
  }, []);

  const plan = useMemo(
    () => (tasks && obligations ? planOverview(tasks, obligations) : null),
    [tasks, obligations],
  );
  const profile = useMemo(() => {
    if (!settings || !profileFacts) return null;
    const firstYear = availableYears.length ? Math.min(...availableYears) : year;
    return profileTask(profileGaps(settings, { ...profileFacts, firstYear }));
  }, [settings, profileFacts, availableYears, year]);
  const agentWork = useMemo(() => agentWorkSummary(plan?.agent ?? []), [plan]);
  const month = useMemo(
    () => (tasks && obligations ? monthToClose(obligations, tasks.today) : ''),
    [tasks, obligations],
  );

  useEffect(() => {
    if (!month) return;
    let current = true;
    setMonthState(null);
    setMonthError('');
    Api.getMonthCloseState(month)
      .then((state) => current && setMonthState(state))
      .catch((e) => current && setMonthError(message(e)));
    return () => {
      current = false;
    };
  }, [month]);

  // Ein Agent bucht, legt ab oder bereitet vor: die Seite liest Aufgaben,
  // Buchungen und Monatsabschluss neu und behält, was sie zeigt, bis der neue
  // Stand da ist.
  useLiveReload(() => {
    void load();
    if (month) {
      Api.getMonthCloseState(month)
        .then((state) => {
          setMonthState(state);
          setMonthError('');
        })
        .catch(() => undefined);
    }
  });

  const closePanel = useCallback(() => navigate({ area: 'overview' }), [navigate]);
  const openPanel = useCallback((key: string) => navigate({ area: 'overview', id: key }), [navigate]);

  return (
    <PageFrame
      title="Übersicht"
      context={`Stand ${formatDate(tasks?.today ?? todayISO())} · Geschäftsjahr ${year}`}
    >
      <DutiesSection plan={plan} extra={profile ? [profile] : []} error={error} onOpen={openPanel} onNavigate={navigate} />
      <InboxSection tenantId={activeTenant?.id} onNavigate={navigate} />
      {!error && (
        <MonthCloseSection month={month} state={monthState} error={monthError} onNavigate={navigate} />
      )}
      <AgentBookingsSection
        bookings={bookings}
        open={agentWork}
        error={bookingsError}
        apps={apps ?? []}
        onNavigate={navigate}
      />
      <AppsSection
        apps={apps}
        error={appsError}
        status={server}
        tenantId={activeTenant?.id}
        onNavigate={navigate}
      />

      <ObligationPanel
        obligationKey={obligationKey}
        onClose={closePanel}
        onChanged={() => void load()}
        apps={apps ?? []}
      />
    </PageFrame>
  );
};
