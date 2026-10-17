import { createContext, useCallback, useContext, useEffect, useState } from 'react';

/**
 * Die Navigation: fünf Bereiche, je Bereich feste Reiter.
 *
 * Ein Ziel besteht aus Bereich, Reiter und der Kennung eines Objekts. Was eine
 * Ansicht darüber hinaus braucht (Filter, Unteransicht), hält sie selbst. Das
 * Geschäftsjahr steht nicht im Ziel, es gilt für die ganze Oberfläche und wird
 * in der Kopfzeile umgeschaltet; `year` ist nur der Auftrag, vorher dorthin zu
 * wechseln.
 *
 * Die Bereiche und Reiter stehen ebenso in internal/domain/task.go, weil die
 * Aufgabenliste ihre Ziele im Backend bildet.
 */

export const AREA_TABS = {
  overview: [],
  books: ['receipts', 'bank', 'invoices', 'journal', 'accounts', 'assets', 'statements'],
  closings: ['month', 'year', 'declarations'],
  company: ['profile', 'contacts'],
  settings: ['apps', 'backup', 'tenants', 'audit', 'advanced'],
} as const satisfies Record<string, readonly string[]>;

export type Area = keyof typeof AREA_TABS;

export type TabOf<A extends Area> = (typeof AREA_TABS)[A][number];

/**
 * Ein Navigationsziel.
 *
 * `id` ist die Kennung des Objekts, das die Ansicht aufschlägt. Was sie
 * bedeutet, hängt am Reiter: Belegnummer im Journal, Kontonummer bei den
 * Konten, Beleg-ID bei den Belegen, Schlüssel einer Pflicht oder eines
 * Abschlussschritts bei den Abschlüssen, Block der Stammdaten beim Unternehmen.
 */
export type Route = {
  [A in Area]: { area: A; tab?: TabOf<A>; id?: string; year?: number };
}[Area];

export type NavigateFn = (route: Route) => void;

export const AREA_LABELS: Record<Area, string> = {
  overview: 'Übersicht',
  books: 'Bücher',
  closings: 'Abschlüsse',
  company: 'Unternehmen',
  settings: 'Einstellungen',
};

export const TAB_LABELS: { [A in Area]: Record<TabOf<A>, string> } = {
  overview: {},
  books: {
    receipts: 'Belege',
    bank: 'Bank',
    invoices: 'Rechnungen',
    journal: 'Journal',
    accounts: 'Konten',
    assets: 'Anlagen',
    statements: 'Bilanz & GuV',
  },
  closings: { month: 'Monat', year: 'Jahr', declarations: 'Erklärungen' },
  company: { profile: 'Stammdaten', contacts: 'Kontakte' },
  settings: {
    apps: 'Apps verbinden',
    backup: 'Sicherung',
    tenants: 'Mandanten',
    audit: 'Prüfung & GoBD',
    advanced: 'Erweitert',
  },
};

export const START_ROUTE: Route = { area: 'overview' };

function isArea(value: string): value is Area {
  return Object.prototype.hasOwnProperty.call(AREA_TABS, value);
}

/**
 * Prüft ein Ziel, das von außen kommt, etwa aus der Aufgabenliste des
 * Backends. Ein unbekannter Bereich ergibt null; ein fehlender oder
 * unbekannter Reiter wird der erste des Bereichs. Eine Kennung ohne gültigen
 * Reiter fällt weg, weil erst der Reiter sagt, was sie bedeutet.
 */
export function toRoute(area: string, tab?: string, id?: string): Route | null {
  if (!isArea(area)) return null;
  const tabs: readonly string[] = AREA_TABS[area];
  const route = { area } as { area: Area; tab?: string; id?: string };
  if (tabs.length === 0) return route as Route;
  const known = tab !== undefined && tabs.includes(tab);
  route.tab = known ? tab : tabs[0];
  if (id && known) route.id = id;
  return route as Route;
}

/** Der Reiter, den ein Bereich zeigt; ohne Angabe sein erster. */
export function activeTab<A extends Exclude<Area, 'overview'>>(route: Route, area: A): TabOf<A> {
  if (route.area === area && route.tab) return route.tab as TabOf<A>;
  return AREA_TABS[area][0] as TabOf<A>;
}

/** `#/books/journal/B-2026-0055` */
export function formatRoute(route: Route): string {
  const parts: string[] = [route.area];
  if (route.tab) parts.push(route.tab);
  if (route.tab && route.id) parts.push(encodeURIComponent(route.id));
  return `#/${parts.join('/')}`;
}

export function parseRoute(hash: string): Route {
  const [area = '', tab, id] = hash.replace(/^#\/?/, '').split('/');
  return toRoute(area, tab, id ? decodeURIComponent(id) : undefined) ?? START_ROUTE;
}

export function sameRoute(a: Route, b: Route): boolean {
  return a.area === b.area && a.tab === b.tab && (a.id ?? '') === (b.id ?? '');
}

/**
 * Das Ziel steht im Hash der Adresse. So führen Zurück und Vor der WebView
 * durch die besuchten Ansichten, und ein Screenshot-Lauf kann eine Ansicht
 * direkt aufrufen.
 */
export function useHashRoute(): [Route, (route: Route) => void] {
  const [route, setRoute] = useState<Route>(() => parseRoute(window.location.hash));

  useEffect(() => {
    const onChange = () => setRoute(parseRoute(window.location.hash));
    window.addEventListener('hashchange', onChange);
    return () => window.removeEventListener('hashchange', onChange);
  }, []);

  const go = useCallback((next: Route) => {
    const normalized = toRoute(next.area, next.tab, next.id) ?? START_ROUTE;
    const hash = formatRoute(normalized);
    if (window.location.hash !== hash) window.location.hash = hash;
    setRoute(normalized);
  }, []);

  return [route, go];
}

interface NavigationValue {
  route: Route;
  navigate: NavigateFn;
}

const NavigationContext = createContext<NavigationValue | null>(null);

export const NavigationProvider = NavigationContext.Provider;

/** Das aktuelle Ziel und der Weg zu einem anderen, für jede Ansicht im Arbeitsbereich. */
export function useNavigation(): NavigationValue {
  const value = useContext(NavigationContext);
  if (!value) throw new Error('useNavigation braucht den NavigationProvider aus App.tsx.');
  return value;
}

/**
 * Springt zu einem Abschnitt der Seite, sobald ein Ziel ihn nennt; `key`
 * wiederholt den Sprung für ein anderes Objekt im selben Abschnitt. Ein
 * zweiter Sprung folgt, wenn darüberliegende Abschnitte nachgeladen und den
 * Anker verschoben haben.
 */
export function useAnchorScroll(anchor: string | undefined, key?: string): void {
  useEffect(() => {
    if (!anchor) return;
    const jump = () => document.getElementById(anchor)?.scrollIntoView({ block: 'start' });
    const timers = [window.setTimeout(jump, 80), window.setTimeout(jump, 600)];
    return () => timers.forEach((timer) => window.clearTimeout(timer));
  }, [anchor, key]);
}

/**
 * Das Objekt, das ein Reiter aufgeschlagen hat, für Seitenleiste oder
 * aufgeklappte Zeile. Öffnen und Schließen laufen über das Ziel, damit ein
 * Verweis von außen dieselbe Ansicht ergibt und Zurück sie wieder schließt.
 */
export function useRouteObject<A extends Exclude<Area, 'overview'>>(
  area: A,
  tab: TabOf<A>,
): { id: string | undefined; open: (id: string) => void; close: () => void } {
  const { route, navigate } = useNavigation();
  const here = route.area === area && activeTab(route, area) === tab;
  const open = useCallback((id: string) => navigate({ area, tab, id } as Route), [navigate, area, tab]);
  const close = useCallback(() => navigate({ area, tab } as Route), [navigate, area, tab]);
  return { id: here ? route.id : undefined, open, close };
}
