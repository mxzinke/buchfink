/**
 * Der Reiter, unter dem eine Pflicht steht. Spiegelt `obligationTarget` in
 * internal/service/task_service.go, damit ein Verweis aus der Übersicht
 * dort landet, wo die Pflicht erledigt wird.
 */
export function obligationTab(kind: string): 'month' | 'year' | 'declarations' | 'company' {
  switch (kind) {
    case 'kst-erklaerung':
    case 'gewst-erklaerung':
    case 'ust-erklaerung':
    case 'e-bilanz':
    case 'e-bilanz-eroeffnung':
    case 'kst-vorauszahlung':
    case 'gewst-vorauszahlung':
      return 'declarations';
    case 'abschlussbuchungen':
    case 'aufstellung':
    case 'feststellung':
    case 'ergebnisverwendung':
    case 'offenlegung':
    case 'kapitalertragsteuer':
      return 'year';
    case 'transparenzregister':
    case 'gesellschafterliste':
    case 'steuerliche-erfassung':
    case 'gewerbeanmeldung':
    case 'eroeffnungsbilanz':
      return 'company';
    default:
      return 'month';
  }
}

/**
 * Die Pflichten, die zu den Stammdaten gehören, nach dem Block, der sie zeigt.
 * Die Eröffnungsbilanz steht bei der Gründung im Block der Gesellschafter.
 */
export const COMPANY_OBLIGATIONS = {
  firma: ['steuerliche-erfassung', 'gewerbeanmeldung'],
  gesellschafter: ['transparenzregister', 'gesellschafterliste', 'eroeffnungsbilanz'],
} as const;
