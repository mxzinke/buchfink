import React, { useEffect, useState } from 'react';
import { Download, CheckCircle2, Code } from 'lucide-react';
import { Api } from '../services/api';
import { HelpTooltip } from '../components/HelpTooltip';

export const EBilanzPage: React.FC = () => {
  const [xbrlContent, setXbrlContent] = useState<string>('');
  const [showRawXML, setShowRawXML] = useState(false);

  useEffect(() => {
    loadXBRL();
  }, []);

  const loadXBRL = async () => {
    try {
      const xml = await Api.exportEBilanzXBRL();
      setXbrlContent(xml);
    } catch (e) {
      console.error(e);
    }
  };

  const handleDownloadXBRL = () => {
    const blob = new Blob([xbrlContent], { type: 'application/xml;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = `e-bilanz-skr04-2024.xbrl`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  return (
    <div className="p-8 max-w-7xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-stone-900 tracking-tight flex items-center">
            E-Bilanz-Export (XBRL)
            <HelpTooltip
              title="E-Bilanz & XBRL Taxonomie"
              content="Buchfink generiert eine standardkonforme XBRL-Instanzdatei nach der deutschen GAAP-Taxonomie 6.7 inklusive vollständigem SKR04-Kontennachweis. Die Datei kann direkt in Mein ELSTER hochgeladen oder über Bridges übermittelt werden."
            />
          </h2>
          <p className="text-xs text-stone-500 mt-1">
            Gültige XBRL-Instanz nach Taxonomie 6.7 &bull; Direkter Kontennachweis
          </p>
        </div>

        <div className="flex gap-2">
          <button
            onClick={() => setShowRawXML(!showRawXML)}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-stone-100 text-stone-700 text-xs font-semibold hover:bg-stone-200 transition-colors border border-stone-200"
          >
            <Code className="w-3.5 h-3.5 text-stone-500" />
            {showRawXML ? 'Vorschau ausblenden' : 'XBRL XML anzeigen'}
          </button>
          <button
            onClick={handleDownloadXBRL}
            className="flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg bg-amber-600 text-white text-xs font-semibold hover:bg-amber-700 transition-colors shadow-xs"
          >
            <Download className="w-3.5 h-3.5" />
            XBRL-Datei exportieren
          </button>
        </div>
      </div>

      {/* Scope Info Box */}
      <div className="p-4 bg-stone-100 border border-stone-200/80 rounded-xl text-xs space-y-2">
        <div className="flex items-center gap-2 font-bold text-stone-900">
          <CheckCircle2 className="w-4 h-4 text-emerald-600" />
          E-Bilanz-Export in v1: Standardisierter XBRL-Datensatz
        </div>
        <p className="text-stone-600 leading-relaxed">
          Buchfink erzeugt eine vollständige XBRL-Datei mit GCD-Stammdaten, GAAP-Bilanz-/GuV-Positionen und Kontennachweis.
          Die ERiC-Übermittlung ist bewusst out-of-scope (proprietäre C-Bibliothek des Finanzamts). Laden Sie die erzeugte XBRL-Datei einfach in <strong>Mein ELSTER</strong> hoch oder nutzen Sie freie Bridges wie eBilanz+.
        </p>
      </div>

      {/* Taxonomy Mapping Overview */}
      <div className="bg-white rounded-xl border border-stone-200/90 shadow-xs p-6 space-y-4">
        <h3 className="text-sm font-bold text-stone-900">Taxonomie-Mapping (SKR04 &rarr; XBRL GAAP)</h3>
        
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs">
          <div className="p-3 bg-stone-50 rounded-lg border border-stone-200/60 space-y-1">
            <div className="font-mono font-bold text-amber-800">1800 Bankkonto</div>
            <div className="font-mono text-[11px] text-stone-500">de-gaap-ci:bs.ass.currAss.cashEquiv.bank</div>
          </div>
          <div className="p-3 bg-stone-50 rounded-lg border border-stone-200/60 space-y-1">
            <div className="font-mono font-bold text-amber-800">1200 Forderungen LuL</div>
            <div className="font-mono text-[11px] text-stone-500">de-gaap-ci:bs.ass.currAss.receiv.trade</div>
          </div>
          <div className="p-3 bg-stone-50 rounded-lg border border-stone-200/60 space-y-1">
            <div className="font-mono font-bold text-amber-800">4400 Erlöse 19% USt</div>
            <div className="font-mono text-[11px] text-stone-500">de-gaap-ci:is.netSales.grossSales.vat19</div>
          </div>
          <div className="p-3 bg-stone-50 rounded-lg border border-stone-200/60 space-y-1">
            <div className="font-mono font-bold text-amber-800">6500 Büromiete</div>
            <div className="font-mono text-[11px] text-stone-500">de-gaap-ci:is.otherCost.rent</div>
          </div>
        </div>
      </div>

      {/* Raw XML Inspector */}
      {showRawXML && (
        <div className="bg-stone-900 text-stone-200 p-5 rounded-xl font-mono text-[11px] leading-relaxed max-h-96 overflow-y-auto border border-stone-800">
          <pre>{xbrlContent}</pre>
        </div>
      )}
    </div>
  );
};
