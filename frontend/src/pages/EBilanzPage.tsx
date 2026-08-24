import React, { useEffect, useState } from 'react';
import { Code, Download } from 'lucide-react';
import { Api } from '../services/api';
import {
  Button,
  HelpPopover,
  PageHeader,
  Section,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  toast,
} from '../components/ui';

/** Auszug der Zuordnung. Die vollständige Abbildung liegt im Backend. */
const MAPPING = [
  { account: '1800', name: 'Bankkonto', position: 'Umlaufvermögen / Liquide Mittel' },
  { account: '1200', name: 'Forderungen LuL', position: 'Forderungen aus Lieferungen und Leistungen' },
  { account: '4400', name: 'Erlöse 19 % USt', position: 'Umsatzerlöse / Betriebliche Erträge' },
  { account: '6500', name: 'Büromiete', position: 'Raumkosten / Betriebliche Aufwendungen' },
];

export const EBilanzPage: React.FC = () => {
  const [xbrlContent, setXbrlContent] = useState('');
  const [showRawXML, setShowRawXML] = useState(false);

  useEffect(() => {
    void loadXBRL();
  }, []);

  async function loadXBRL() {
    try {
      setXbrlContent(await Api.exportEBilanzXBRL());
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    }
  }

  function handleDownloadXBRL() {
    const blob = new Blob([xbrlContent], { type: 'application/xml;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = 'e-bilanz-export.xbrl';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
    toast.success('E-Bilanz-Datei gespeichert.');
  }

  return (
    <div className="max-w-[1200px] mx-auto px-8 py-8">
      <PageHeader
        title="E-Bilanz"
        context="Export nach amtlicher Taxonomie, § 5b EStG"
        action={
          <div className="flex gap-2">
            <Button
              variant="secondary"
              icon={<Code className="w-4 h-4" strokeWidth={1.5} />}
              onClick={() => setShowRawXML((v) => !v)}
            >
              {showRawXML ? 'Rohdaten ausblenden' : 'Rohdaten anzeigen'}
            </Button>
            <Button
              variant="primary"
              icon={<Download className="w-4 h-4" strokeWidth={1.5} />}
              onClick={handleDownloadXBRL}
              disabled={!xbrlContent}
              title={xbrlContent ? undefined : 'Die Datei wird noch erzeugt'}
            >
              Datei herunterladen
            </Button>
          </div>
        }
      />

      <Section
        title="Zuordnung der Standardkonten"
        context="SKR04 auf die Positionen der amtlichen Taxonomie"
        divider={false}
        className="mt-8"
        action={
          <HelpPopover label="Erklärung zur E-Bilanz">
            Die erzeugte XBRL-Datei enthält Bilanz, GuV und den Kontennachweis. Sie lässt sich in
            Mein ELSTER hochladen oder an die Steuerberatung übergeben. Eine direkte Übermittlung
            aus Buchfink heraus gibt es bewusst nicht.
          </HelpPopover>
        }
      >
        <Table>
          <Thead>
            <Tr>
              <Th className="w-28">Konto</Th>
              <Th>Bezeichnung</Th>
              <Th>Taxonomie-Position</Th>
            </Tr>
          </Thead>
          <Tbody>
            {MAPPING.map((row) => (
              <Tr key={row.account}>
                <Td code>{row.account}</Td>
                <Td>{row.name}</Td>
                <Td className="text-ink-muted">{row.position}</Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      </Section>

      {showRawXML && (
        <Section title="Rohdaten" context={`${xbrlContent.length.toLocaleString('de-DE')} Zeichen`}>
          {/* Technische Ausgabe, deshalb Monospace und die dunkle Fläche (§4.1). */}
          <pre className="rounded-card border border-shell-line bg-shell text-shell-text
                          font-mono text-caption leading-relaxed p-5 max-h-96 overflow-auto">
            {xbrlContent}
          </pre>
        </Section>
      )}
    </div>
  );
};
