/**
 * Dateien speichern.
 *
 * Buchfink gibt Dateien über einen Anker aus, den es selbst anlegt und gleich
 * wieder entfernt. Der Weg stand bis Welle 3 zweimal im Code — einmal für den
 * Jahresabschluss, einmal für die Umsatzsteuer —, und zwei Fassungen desselben
 * Wegs laufen früher oder später auseinander.
 */

/** Speichert einen Blob unter dem angegebenen Namen. */
export function downloadBlob(name: string, blob: Blob): void {
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = name;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

/**
 * Speichert Text als CSV. Das Byte-Order-Mark steht davor, damit die
 * Tabellenkalkulation die Umlaute als UTF-8 liest und nicht als Zeichensalat.
 */
export function downloadCSV(name: string, content: string): void {
  downloadBlob(name, new Blob([`﻿${content}`], { type: 'text/csv;charset=utf-8' }));
}
