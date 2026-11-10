/** Gesetzeskürzel und ihre Verzeichnisse bei Gesetze im Internet. */
export const STATUTES: Record<string, string> = {
  AO: 'ao_1977', HGB: 'hgb', EStG: 'estg', EStDV: 'estdv_1955',
  UStG: 'ustg_1980', UStDV: 'ustdv_1980', GmbHG: 'gmbhg', AktG: 'aktg',
  BGB: 'bgb', KStG: 'kstg_1977', GewStG: 'gewstg', InvStG: 'invstg_2018',
  SolZG: 'solzg_1995', InsO: 'inso', GwG: 'gwg_2017', GewO: 'gewo',
};

/** Erfasst nur Normverweise, keine normalen Sätze zwischen Paragraph und Kürzel. */
export const REFERENCE_SOURCE =
  `§{1,2}\\s*(\\d+[a-z]?)(?:\\s*(?:Abs\\.|Absatz|Nr\\.|Satz|Sätze|S\\.|Buchst\\.|bis|und|ff\\.|[,–-]|\\d+[a-z]?|[a-e](?![a-z])|[IVX]+(?![a-z])))*\\s+(${Object.keys(STATUTES).join('|')})\\b` +
  `|GoBD(?:\\s+Rz\\.?\\s*\\d+(?:\\s*ff\\.)?)?|Art\\.\\s*(\\d+)(?:\\s*(?:Abs\\.|Buchst\\.|\\d+|[a-z](?![a-z])))*\\s+DSGVO`;

/**
 * Trennt die Normen aus einem Text, den das Backend liefert, etwa
 * „Verpflichtungen … (§ 249 HGB)“ oder „Beteiligungserwerb nach § 8c KStG“.
 * Die Arbeitsansicht zeigt `text`; die Normen gehören in den Detaildialog
 * (§15.2 der Gestaltung). Eine Norm mitten im Satz bleibt stehen, weil der Satz
 * ohne sie nicht mehr stimmt.
 */
export function splitNorms(value: string): { text: string; norms: string[] } {
  const norms: string[] = [];
  const only = new RegExp(`^(?:\\s*(?:${REFERENCE_SOURCE})\\s*(?:[,;]|und|i\\.\\s?V\\.\\s?m\\.)?)+\\s*$`);
  const any = new RegExp(REFERENCE_SOURCE, 'g');
  let text = value.replace(/\s*\(([^()]*)\)/g, (whole, inner: string) => {
    if (!only.test(inner)) return whole;
    norms.push(...(inner.match(any) ?? []));
    return '';
  });
  text = text.replace(new RegExp(`\\s+(?:nach|gemäß|gem\\.)\\s+((?:${REFERENCE_SOURCE}))`, 'g'), (_whole, norm: string) => {
    norms.push(norm);
    return '';
  });
  return { text: text.replace(/\s+([.,;:])/g, '$1').trim(), norms };
}
