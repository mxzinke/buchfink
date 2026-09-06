/**
 * Wohin ein Befund des Prüflaufs führt.
 *
 * Ein Befund ohne Adresse ist eine Hausaufgabe ohne Ort: „Beleg 2026-014 ist
 * nicht gebucht" sagt, was zu tun ist, aber nicht, wo. Die Zuordnung steht hier
 * und nicht in den Ansichten, weil sie inzwischen an drei Stellen gebraucht
 * wird — Fristenliste, Monatsabschluss und Aufgabenliste — und drei Kopien
 * auseinanderlaufen, sobald das Backend einen Objekttyp ergänzt.
 */

import type { NavigationParams, TabType } from '../components/Sidebar';
import type { CheckFinding } from '../types';

/**
 * Die Beschriftung des Knopfes, der zum Ziel führt.
 *
 * Der Knopf nennt das Ziel, statt auf sich selbst zu zeigen: „Zum Beleg" sagt
 * vor dem Klick, wo man landet, „Öffnen" erst danach. Für Reiter ohne
 * naheliegenden Namen bleibt „Öffnen" der Rückfall.
 */
export function targetLabel(tab: TabType | null): string {
  switch (tab) {
    case 'journal':
      return 'Zur Buchung';
    case 'receipts':
      return 'Zum Beleg';
    case 'bank':
      return 'Zum Bankumsatz';
    case 'accounts':
      return 'Zum Konto';
    case 'invoices':
      return 'Zur Rechnung';
    case 'advances':
      return 'Zur Anzahlung';
    case 'assets':
      return 'Zum Anlagegut';
    case 'contacts':
      return 'Zum Kontakt';
    case 'vat':
      return 'Zur Voranmeldung';
    case 'deadlines':
      return 'Zur Frist';
    case 'closing':
    case 'closingmodules':
      return 'Zum Jahresabschluss';
    case 'obligations':
      return 'Zu den Nebenpflichten';
    case 'audit':
      return 'Zur Prüfübersicht';
    case 'backup':
      return 'Zur Datensicherung';
    case 'taxaudit':
      return 'Zur Betriebsprüfung';
    case 'settings':
      return 'Zu den Einstellungen';
    default:
      return 'Öffnen';
  }
}

/** Die Ansicht, in der ein Befund zu beheben ist; null heißt: keine bekannte. */
export function findingTarget(objectType?: string): TabType | null {
  switch (objectType) {
    case 'JOURNAL_ENTRY':
      return 'journal';
    case 'RECEIPT':
      return 'receipts';
    case 'BANK_TX':
      return 'bank';
    case 'ACCOUNT':
      return 'accounts';
    case 'VAT_PERIOD':
      return 'vat';
    default:
      return null;
  }
}

/**
 * Der Parameter, mit dem die Zielansicht öffnet.
 *
 * Ohne ihn landete der Anwender auf einer Liste und müsste die Buchung, von der
 * er kommt, dort erneut suchen.
 */
export function findingParams(finding: CheckFinding): NavigationParams {
  switch (finding.objectType) {
    case 'JOURNAL_ENTRY':
      return { entryNumber: finding.objectName };
    case 'ACCOUNT':
      return { account: finding.objectId };
    case 'RECEIPT': {
      // Der Beleg selbst und nicht der Zustandsfilter: „Leistungsnachweis
      // fehlt" trifft gerade den gebuchten Beleg (siehe checkServiceProof in
      // internal/service/check_service.go), und die Liste „zu buchen" zeigt
      // ihn nicht — der Sprung endete auf einer Liste ohne den gemeinten
      // Beleg. Die Belegseite stellt den Filter selbst auf „alle", sobald sie
      // einen Beleg aufschlägt. Nur wenn der Befund keine Beleg-Kennung hat,
      // bleibt der Filter die beste verfügbare Auskunft.
      const receiptId = Number(finding.objectId);
      if (Number.isInteger(receiptId) && receiptId > 0) return { receiptId };
      return { receiptStatus: 'filed' };
    }
    default:
      return {};
  }
}
