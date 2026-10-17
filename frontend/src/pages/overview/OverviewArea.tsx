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
import { monthToClose, planOverview, todayISO } from '../../utils/overview';
import { useNavigation } from '../../navigation';
import { useWorkspace } from '../../workspace';
import { Notice, PageFrame } from '../../components/ui';
import { DutiesSection } from './DutiesSection';
import { MonthCloseSection } from './MonthCloseSection';
import { AgentBookingsSection } from './AgentBookingsSection';
import { AppsSection } from './AppsSection';
import { ObligationPanel } from './ObligationPanel';

/** So viele Buchungen der Apps zeigt die Übersicht; der Rest steht im Journal. */
const AGENT_BOOKINGS_SHOWN = 8;

function message(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

/**
 * Die Startseite: was jetzt zu tun ist, der anstehende Monatsabschluss, was
 * die verbundenen Apps seit der letzten Festschreibung gebucht haben, und ob
 * sie verbunden sind. Gebucht wird hier nichts; bestätigt wird nur, was allein
 * der Mensch bestätigen kann. Eine Pflicht öffnet sich über das Ziel
 * `#/overview/<schlüssel>` in der Seitenleiste.
 */
export const OverviewArea: React.FC = () => {
  const { route, navigate } = useNavigation();
  const { year, activeTenant } = useWorkspace();
  const obligationKey = route.area === 'overview' ? route.id : undefined;

  const [tasks, setTasks] = useState<TaskList | null>(null);
  const [obligations, setObligations] = useState<Obligation[] | null>(null);
  const [error, setError] = useState('');
  const [bookings, setBookings] = useState<AgentBookings | null>(null);
  const [apps, setApps] = useState<ConnectedApp[] | null>(null);
  const [server, setServer] = useState<MCPServerStatus | null>(null);
  const [monthState, setMonthState] = useState<MonthCloseState | null>(null);
  const [monthError, setMonthError] = useState('');

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
      // Die übrigen Abschnitte bleiben benutzbar; der Fehler steht oben.
      setTasks({ today: todayISO(), overdue: [], open: [], upcoming: [] });
      setObligations([]);
      setError(message(e));
    }
    Api.getAgentBookings(AGENT_BOOKINGS_SHOWN)
      .then(setBookings)
      .catch(() => setBookings({ total: 0, entries: [] }));
  }, []);

  useEffect(() => {
    void load();
  }, [load, year, activeTenant?.id]);

  useEffect(() => {
    Promise.all([Api.getConnectedApps(), Api.getMCPServerStatus()])
      .then(([nextApps, status]) => {
        setApps(nextApps.filter((app) => !app.revoked));
        setServer(status);
      })
      .catch(() => setApps([]));
  }, []);

  const plan = useMemo(
    () => (tasks && obligations ? planOverview(tasks, obligations) : null),
    [tasks, obligations],
  );
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

  const closePanel = useCallback(() => navigate({ area: 'overview' }), [navigate]);
  const openPanel = useCallback((key: string) => navigate({ area: 'overview', id: key }), [navigate]);

  return (
    <PageFrame
      title="Übersicht"
      context={
        tasks ? `Stand ${formatDate(tasks.today)} · Geschäftsjahr ${year}` : `Geschäftsjahr ${year}`
      }
      notice={error ? <Notice tone="negative" text={error} /> : undefined}
    >
      <DutiesSection plan={plan} onOpen={openPanel} onNavigate={navigate} />
      <MonthCloseSection month={month} state={monthState} error={monthError} onNavigate={navigate} />
      <AgentBookingsSection bookings={bookings} apps={apps ?? []} onNavigate={navigate} />
      <AppsSection apps={apps} status={server} tenantId={activeTenant?.id} onNavigate={navigate} />

      <ObligationPanel
        obligationKey={obligationKey}
        onClose={closePanel}
        onChanged={() => void load()}
        apps={apps ?? []}
      />
    </PageFrame>
  );
};
