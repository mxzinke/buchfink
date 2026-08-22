// SPDX-FileCopyrightText: 2026 Maximilian Pfennig
// SPDX-License-Identifier: EUPL-1.2

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
    link.download = `e-bilanz-export.xbrl`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  return (
    <div className="p-3 sm:p-6 md:p-8 max-w-7xl mx-auto space-y-4 sm:space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-stone-900 tracking-tight flex items-center">
            E-Bilanz
            <HelpTooltip
              title="E-Bilanz"
              content="Buchfink generiert die standardisierte E-Bilanz-Datei inklusive Kontennachweis. Diese Datei kann direkt in Mein ELSTER hochgeladen oder an Ihren Steuerberater übergeben werden."
            />
          </h2>
          <p className="text-xs text-stone-500 mt-1">
            Standardisierter Export für das Finanzamt und die elektronische Steuererklärung
          </p>
        </div>

        <div className="flex gap-2">
          <button
            onClick={() => setShowRawXML(!showRawXML)}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-stone-100 text-stone-700 text-xs font-semibold hover:bg-stone-200 transition-colors border border-stone-200"
          >
            <Code className="w-3.5 h-3.5 text-stone-500" />
            {showRawXML ? 'Vorschau ausblenden' : 'Rohdaten anzeigen'}
          </button>
          <button
            onClick={handleDownloadXBRL}
            className="flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg bg-amber-700 text-white text-xs font-semibold hover:bg-amber-800 transition-colors shadow-xs"
          >
            <Download className="w-3.5 h-3.5" />
            E-Bilanz-Datei herunterladen
          </button>
        </div>
      </div>

      {/* Scope Info Box */}
      <div className="p-4 bg-stone-100 border border-stone-200/80 rounded-xl text-xs space-y-2">
        <div className="flex items-center gap-2 font-bold text-stone-900">
          <CheckCircle2 className="w-4 h-4 text-emerald-600" />
          Elektronische Bilanz (§ 5b EStG)
        </div>
        <p className="text-stone-600 leading-relaxed text-xs">
          Buchfink erzeugt eine vollständige E-Bilanz-Datei nach amtlicher Taxonomie für bilanzierungspflichtige Unternehmen.
          Laden Sie die XBRL-Datei einfach in <strong>Mein ELSTER</strong> hoch oder übergeben Sie diese an Ihre Steuerberatung.
        </p>
      </div>

      {/* Taxonomy Mapping Overview */}
      <div className="bg-white rounded-xl border border-stone-200/80 shadow-xs p-6 space-y-4">
        <h3 className="text-sm font-bold text-stone-900">Zuordnung der Standardkonten</h3>
        
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs">
          <div className="p-3 bg-stone-50 rounded-lg border border-stone-200/60 space-y-1">
            <div className="font-mono font-bold text-amber-800">1800 Bankkonto</div>
            <div className="text-xs text-stone-500">Umlaufvermögen / Liquide Mittel</div>
          </div>
          <div className="p-3 bg-stone-50 rounded-lg border border-stone-200/60 space-y-1">
            <div className="font-mono font-bold text-amber-800">1200 Forderungen LuL</div>
            <div className="text-xs text-stone-500">Forderungen aus Lieferungen und Leistungen</div>
          </div>
          <div className="p-3 bg-stone-50 rounded-lg border border-stone-200/60 space-y-1">
            <div className="font-mono font-bold text-amber-800">4400 Erlöse 19% USt</div>
            <div className="text-xs text-stone-500">Umsatzerlöse / Betriebliche Erträge</div>
          </div>
          <div className="p-3 bg-stone-50 rounded-lg border border-stone-200/60 space-y-1">
            <div className="font-mono font-bold text-amber-800">6500 Büromiete</div>
            <div className="text-xs text-stone-500">Raumkosten / Betriebliche Aufwendungen</div>
          </div>
        </div>
      </div>

      {/* Raw XML Inspector */}
      {showRawXML && (
        <div className="bg-[#24211E] text-stone-200 p-5 rounded-xl font-mono text-xs leading-relaxed max-h-96 overflow-y-auto border border-stone-800">
          <pre>{xbrlContent}</pre>
        </div>
      )}
    </div>
  );
};
