/**
 * Aufnahmen der Übersicht: Startseite, Einzelheiten einer Pflicht, Nachweis,
 * der Eingang und Dateien, die ins Fenster gezogen werden.
 */

/**
 * Spielt das Ziehen von Dateien über das Fenster nach. Die Wails-Laufzeit hört
 * auf dieselben DOM-Ereignisse wie unter Windows und setzt die Klasse, an der
 * die Ablagezone hängt.
 */
function drag(page, type) {
  return page.evaluate((eventType) => {
    const data = new DataTransfer();
    data.items.add(new File(['%PDF-1.7'], 'Hanse Cloud Rechnung September.pdf', { type: 'application/pdf' }));
    const target = document.querySelector('[data-file-drop-target]');
    const box = target.getBoundingClientRect();
    target.dispatchEvent(
      new DragEvent(eventType, {
        bubbles: true,
        cancelable: true,
        dataTransfer: data,
        clientX: box.left + box.width / 2,
        clientY: box.top + box.height / 2,
      }),
    );
  }, type);
}

export default [
  {
    file: 'uebersicht.png',
    go: async ({ page, nav }) => {
      await nav('Übersicht');
      await page.getByRole('heading', { level: 1, name: 'Übersicht' }).waitFor();
      await page.getByText('Gewerbesteuer-Vorauszahlung 15.08.2026').waitFor();
      await page.getByText('Monatsfestschreibung Juli 2026').waitFor();
      await page.getByText('B-2026-0180').first().waitFor();
      await page.waitForTimeout(400);
    },
  },
  {
    // Eine von Claude vorbereitete Pflicht in der Seitenleiste.
    file: 'uebersicht-pflicht.png',
    go: async ({ page }) => {
      await page.getByRole('button', { name: 'Lohnsteuer-Anmeldung August 2026', exact: true }).click();
      await page.getByRole('dialog', { name: 'Lohnsteuer-Anmeldung August 2026' }).waitFor();
      await page.getByText('Lohnabrechnung August abgelegt', { exact: false }).waitFor();
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
  {
    // Der Eingang mit einer Datei, die seit 19 Tagen liegt, und der Liste der Dateien.
    file: 'uebersicht-eingang.png',
    go: async ({ page, open }) => {
      await open('#/overview');
      await page.getByRole('heading', { level: 2, name: 'Eingang' }).waitFor();
      await page.getByText('Sagen Sie Claude', { exact: false }).waitFor();
      await page.getByRole('button', { name: /^Dateien im Eingang/ }).click();
      await page.getByText('Kontoauszug DKB 2026-08-24.xml').waitFor();
      await page.locator('#eingang').evaluate((section) => section.scrollIntoView({ block: 'start' }));
      await page.waitForTimeout(400);
    },
  },
  {
    // Während des Ziehens steht die Ablagezone über dem Fenster.
    file: 'uebersicht-ziehen.png',
    go: async ({ page }) => {
      await drag(page, 'dragenter');
      await drag(page, 'dragover');
      await page.locator('.drop-zone').waitFor({ state: 'visible' });
      await page.waitForTimeout(200);
    },
    after: ({ page }) => drag(page, 'drop'),
  },
  {
    // Nach dem Ablegen meldet das Backend, was im Eingang liegt und was nicht.
    file: 'uebersicht-abgelegt.png',
    go: async ({ page }) => {
      await page.evaluate(() =>
        window._wails.dispatchWailsEvent({
          name: 'buchfink:eingang-abgelegt',
          data: {
            names: ['Hanse Cloud Rechnung September.pdf'],
            skipped: [{ name: 'Notizen Steuerberatung.docx', reason: 'kein Beleg (PDF, Bild oder XML)' }],
          },
        }),
      );
      await page.getByText('liegt jetzt im Eingang', { exact: false }).waitFor();
      await page.waitForTimeout(500);
    },
    // Die Meldung stünde sonst noch in der nächsten Aufnahme.
    after: async ({ page }) => {
      await page.evaluate(() =>
        document.querySelectorAll('[data-sonner-toast] [data-close-button]').forEach((button) => button.click()),
      );
      await page.waitForTimeout(400);
    },
  },
];
