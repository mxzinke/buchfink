import React, { useEffect, useState } from 'react';
import { Button, Dialog, Notice, RadioGroup } from './ui';

/**
 * Wie weit ein Geschäftsjahr in die Zukunft reichen darf.
 *
 * Ein Jahr im Voraus ist die Grenze: Zum Jahreswechsel wird im neuen Jahr
 * gebucht, bevor das alte festgestellt ist — dafür muss es anzulegen sein.
 * Alles darüber ist kein Anwendungsfall, sondern ein Vertipper, und er
 * hinterlässt ein leeres Geschäftsjahr, das sich nicht mehr entfernen lässt.
 */
export const MAX_YEARS_AHEAD = 1;

export type FiscalYearDirection = 'next' | 'previous';

export interface FiscalYearDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** Die bereits angelegten Geschäftsjahre. */
  availableYears: number[];
  onCreate: (year: number) => void;
}

/**
 * Ein Geschäftsjahr anzulegen ist kein Klick.
 *
 * Bis hierher legte das Pluszeichen in der Kopfzeile das Folgejahr sofort an —
 * ohne Rückfrage und ohne Grenze nach oben. Ein Geschäftsjahr ist aber eine
 * Entität mit Zeitraum, aus der Buchungen, Vorträge und Abschlüsse hängen; ein
 * versehentlich angelegtes Jahr 2031 steht danach in jeder Auswahl. Deshalb
 * zuerst die Richtung, dann die Bestätigung (§8.2).
 */
export const FiscalYearDialog: React.FC<FiscalYearDialogProps> = ({
  open,
  onOpenChange,
  availableYears,
  onCreate,
}) => {
  const calendarYear = new Date().getFullYear();
  const years = availableYears.length > 0 ? availableYears : [calendarYear];
  const nextYear = Math.max(...years) + 1;
  const previousYear = Math.min(...years) - 1;
  // Das Folgejahr schließt an das zuletzt erfasste an. Liegt dieses schon
  // weiter in der Zukunft, als die Grenze erlaubt, gibt es nichts anzulegen.
  const nextAllowed = nextYear <= calendarYear + MAX_YEARS_AHEAD;

  const [direction, setDirection] = useState<FiscalYearDirection>(
    nextAllowed ? 'next' : 'previous',
  );

  // Beim Öffnen steht die Auswahl wieder auf dem Regelfall: das Folgejahr ist
  // der Grund, aus dem dieser Dialog fast immer aufgeht.
  useEffect(() => {
    if (open) setDirection(nextAllowed ? 'next' : 'previous');
  }, [open, nextAllowed]);

  const target = direction === 'next' ? nextYear : previousYear;

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title="Geschäftsjahr anlegen"
      width="max-w-lg"
      footer={
        <>
          <Button variant="secondary" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button
            variant="primary"
            disabled={direction === 'next' && !nextAllowed}
            onClick={() => {
              onOpenChange(false);
              onCreate(target);
            }}
          >
            Geschäftsjahr {target} anlegen
          </Button>
        </>
      }
    >
      <RadioGroup<FiscalYearDirection>
        value={direction}
        onValueChange={setDirection}
        options={[
          {
            value: 'next',
            label: `Kommendes Geschäftsjahr ${nextYear}`,
            hint: nextAllowed
              ? 'Es beginnt am Tag nach dem Ende des zuletzt erfassten Jahres und dauert zwölf Monate.'
              : `Weiter als bis ${calendarYear + MAX_YEARS_AHEAD} lässt sich nicht vorausbuchen.`,
            disabled: !nextAllowed,
          },
          {
            value: 'previous',
            label: `Vergangenes Geschäftsjahr ${previousYear}`,
            hint: `Es endet am Tag vor dem Beginn von ${Math.min(...years)} — für Eröffnungswerte und die Übernahme aus einem Altsystem.`,
          },
        ]}
      />

      <p className="text-body text-ink-muted mt-5">
        {direction === 'next'
          ? `Buchfink legt das Geschäftsjahr ${target} an und schaltet die Ansicht darauf um. Gebucht wird danach in ${target}, bis Sie das Jahr in der Kopfzeile wechseln.`
          : `Buchfink legt das Geschäftsjahr ${target} vor dem bisher ersten Jahr an und schaltet die Ansicht darauf um. Der Saldenvortrag nach ${Math.min(...years)} ist danach neu zu buchen.`}
      </p>

      {/* Ein angelegtes Geschäftsjahr bleibt: es steht im Protokoll, und die
          Buchungen, Vorträge und Abschlüsse daran hängen an seinem Zeitraum. */}
      <Notice
        className="mt-4"
        text="Ein angelegtes Geschäftsjahr lässt sich nicht wieder entfernen."
      />
    </Dialog>
  );
};
