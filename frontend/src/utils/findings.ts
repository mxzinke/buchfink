/**
 * Wohin ein Befund des Prüflaufs führt.
 *
 * Ein Befund ohne Adresse ist eine Hausaufgabe ohne Ort: „Beleg 2026-014 ist
 * nicht gebucht" sagt, was zu tun ist, aber nicht, wo. Die Zuordnung steht
 * hier, weil Monatsabschluss, Festschreibung und Übersicht sie brauchen.
 */

import type { Route } from '../navigation';
import type { CheckFinding } from '../types';

/**
 * Die Beschriftung des Knopfes, der zum Ziel führt. Sie nennt das Ziel, damit
 * man vor dem Klick weiß, wo man landet.
 */
export function targetLabel(route: Route | null): string {
  if (!route) return 'Öffnen';
  switch (route.area) {
    case 'books': {
      // Mit Kennung führt der Knopf zu einem Objekt, ohne Kennung zur Liste.
      const one = Boolean(route.id);
      switch (route.tab) {
        case 'journal':
          return one ? 'Zur Buchung' : 'Zum Journal';
        case 'receipts':
          return one ? 'Zum Beleg' : 'Zu den Belegen';
        case 'bank':
          return one ? 'Zum Bankumsatz' : 'Zu den Bankumsätzen';
        case 'accounts':
          return one ? 'Zum Konto' : 'Zu den Konten';
        case 'invoices':
          return one ? 'Zur Rechnung' : 'Zu den Rechnungen';
        case 'assets':
          return one ? 'Zum Anlagegut' : 'Zu den Anlagen';
        case 'statements':
          return 'Zur Bilanz';
        default:
          return 'Zu den Büchern';
      }
    }
    case 'closings':
      switch (route.tab) {
        case 'month':
          return 'Zum Monatsabschluss';
        case 'declarations':
          return 'Zur Erklärung';
        default:
          return 'Zum Jahresabschluss';
      }
    case 'company':
      if (route.tab === 'contacts') return route.id ? 'Zum Kontakt' : 'Zu den Kontakten';
      return 'Zu den Stammdaten';
    case 'settings':
      switch (route.tab) {
        case 'backup':
          return 'Zur Sicherung';
        case 'inbox':
          return 'Zum Eingang';
        case 'audit':
          return 'Zur Prüfung';
        default:
          return 'Zu den Einstellungen';
      }
    default:
      return 'Öffnen';
  }
}

/**
 * Die Stelle, an der ein Befund zu beheben ist; null heißt: keine bekannte.
 *
 * Mit der Kennung des Objekts, damit die Ansicht es gleich aufschlägt und man
 * es nicht in der Liste suchen muss.
 */
export function findingRoute(finding: CheckFinding): Route | null {
  switch (finding.objectType) {
    case 'JOURNAL_ENTRY':
      return { area: 'books', tab: 'journal', id: finding.objectName || undefined };
    case 'RECEIPT': {
      // Der Beleg selbst und nicht ein Zustandsfilter: „Leistungsnachweis
      // fehlt" trifft gerade den gebuchten Beleg, den die Liste „zu buchen"
      // nicht zeigt.
      const receiptId = Number(finding.objectId);
      return {
        area: 'books',
        tab: 'receipts',
        id: Number.isInteger(receiptId) && receiptId > 0 ? String(receiptId) : undefined,
      };
    }
    case 'BANK_TX':
      return { area: 'books', tab: 'bank', id: finding.objectId || undefined };
    case 'ACCOUNT':
      return { area: 'books', tab: 'accounts', id: finding.objectId || undefined };
    case 'VAT_PERIOD':
      return { area: 'closings', tab: 'month' };
    default:
      return null;
  }
}
