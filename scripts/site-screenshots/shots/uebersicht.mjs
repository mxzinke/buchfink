/** Aufnahmen der Übersicht: Startseite, Einzelheiten einer Pflicht, Nachweis. */
export default [
  {
    file: 'uebersicht.png',
    go: async ({ page, nav }) => {
      await nav('Übersicht');
      await page.getByRole('heading', { level: 1, name: 'Übersicht' }).waitFor();
      await page.getByText('Gewerbesteuer-Vorauszahlung 15.08.2026').waitFor();
      await page.getByText('Monatsfestschreibung Juli 2026').waitFor();
      await page.getByText('B-2026-0055').first().waitFor();
      await page.waitForTimeout(400);
    },
  },
  {
    // Eine von Claude vorbereitete Pflicht in der Seitenleiste.
    file: 'uebersicht-pflicht.png',
    go: async ({ page }) => {
      await page.getByRole('button', { name: 'Lohnsteuer-Anmeldung August 2026', exact: true }).click();
      await page.getByRole('dialog', { name: 'Lohnsteuer-Anmeldung August 2026' }).waitFor();
      await page.getByText('Lohnjournal August gebucht', { exact: false }).waitFor();
      await page.waitForTimeout(400);
    },
  },
  {
    // Bestätigen mit Nachweis: Transferticket, Tag und Notiz.
    file: 'uebersicht-nachweis.png',
    go: async ({ page }) => {
      await page.getByRole('button', { name: 'Als erledigt bestätigen', exact: true }).click();
      await page.getByRole('dialog', { name: 'Als erledigt bestätigen' }).waitFor();
      await page.waitForTimeout(400);
    },
    after: async ({ escape }) => {
      await escape();
      await escape();
    },
  },
];
