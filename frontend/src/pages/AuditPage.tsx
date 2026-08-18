import React, { useEffect, useState } from 'react';
import { ShieldCheck, ShieldAlert, RefreshCw } from 'lucide-react';
import { AuditLogEntry, IntegrityCheckResult } from '../types';
import { Api } from '../services/api';
import { HelpTooltip } from '../components/HelpTooltip';

export const AuditPage: React.FC = () => {
  const [logs, setLogs] = useState<AuditLogEntry[]>([]);
  const [integrity, setIntegrity] = useState<IntegrityCheckResult | null>(null);
  const [isVerifying, setIsVerifying] = useState(false);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    try {
      const [l, res] = await Promise.all([
        Api.getAuditLogs(),
        Api.verifyIntegrity(),
      ]);
      setLogs(l);
      setIntegrity(res);
    } catch (e) {
      console.error(e);
    }
  };

  const handleReverify = async () => {
    setIsVerifying(true);
    try {
      const res = await Api.verifyIntegrity();
      setIntegrity(res);
    } finally {
      setIsVerifying(false);
    }
  };

  return (
    <div className="p-3 sm:p-6 md:p-8 max-w-7xl mx-auto space-y-4 sm:space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <h2 className="text-2xl font-bold text-stone-900 tracking-tight flex items-center">
            Sicherheit & Protokoll
            <HelpTooltip
              title="Sicherheit & Nachvollziehbarkeit"
              content="Jede Buchung und jede Änderung wird automatisch erfasst und vor nachträglicher Manipulation geschützt. So sind alle Vorgänge jederzeit lückenlos nachvollziehbar."
            />
          </h2>
          <p className="text-xs text-stone-500 mt-1">
            Lückenloses Änderungsprotokoll und Schutz vor nachträglicher Datenveränderung
          </p>
        </div>

        <button
          onClick={handleReverify}
          disabled={isVerifying}
          className="flex items-center gap-2 px-3.5 py-2 rounded-lg bg-stone-700 text-stone-100 text-xs font-semibold hover:bg-stone-800 transition-colors shadow-xs"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${isVerifying ? 'animate-spin text-amber-300' : ''}`} />
          Daten jetzt prüfen
        </button>
      </div>

      {/* Integrity Card */}
      {integrity && (
        <div
          className={`p-6 rounded-xl border flex flex-col md:flex-row items-start md:items-center justify-between gap-4 ${
            integrity.isValid
              ? 'bg-emerald-50/70 border-emerald-200 text-emerald-950'
              : 'bg-rose-50 border-rose-200 text-rose-950'
          }`}
        >
          <div className="flex items-start gap-3.5">
            <div
              className={`p-2.5 rounded-lg shrink-0 ${
                integrity.isValid ? 'bg-emerald-600 text-white' : 'bg-rose-600 text-white'
              }`}
            >
              {integrity.isValid ? (
                <ShieldCheck className="w-5 h-5" />
              ) : (
                <ShieldAlert className="w-5 h-5" />
              )}
            </div>
            <div>
              <h3 className="font-bold text-sm">
                {integrity.isValid
                  ? 'Alle Buchungen sind vollständig und unverändert'
                  : 'Achtung: Unstimmigkeit festgestellt!'}
              </h3>
              <p className="text-xs text-stone-600 mt-1">{integrity.message}</p>
              <div className="mt-2 text-xs text-stone-500">
                Letzte erfolgreiche Prüfung am {integrity.checkedAt}
              </div>
            </div>
          </div>

          <div className="text-right shrink-0">
            <span className="px-3 py-1 rounded-full text-xs font-bold bg-emerald-100 text-emerald-800 border border-emerald-300">
              {integrity.checkedEntries}/{integrity.totalEntries} Buchungen geprüft
            </span>
          </div>
        </div>
      )}

      {/* Audit Log Table */}
      <div className="bg-white rounded-xl border border-stone-200/80 shadow-xs overflow-hidden">
        <div className="p-4 border-b border-stone-200 bg-stone-50/60 flex items-center justify-between">
          <h3 className="text-xs font-bold text-stone-900 uppercase tracking-wider">
            Änderungsprotokoll
          </h3>
          <span className="text-xs text-stone-500">Chronologischer Verlauf</span>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead className="bg-stone-50 border-b border-stone-200 text-stone-500 font-medium">
              <tr>
                <th className="py-3 px-4 w-40">Zeitpunkt</th>
                <th className="py-3 px-4 w-28">Aktion</th>
                <th className="py-3 px-4 w-32">Bereich</th>
                <th className="py-3 px-4">Beschreibung</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-stone-100">
              {logs.length === 0 ? (
                <tr>
                  <td colSpan={4} className="py-8 text-center text-stone-400">
                    Keine Einträge vorhanden.
                  </td>
                </tr>
              ) : (
                logs.map((l) => (
                  <tr key={l.id} className="hover:bg-stone-50 transition-colors">
                    <td className="py-3 px-4 text-stone-600 font-mono text-xs">
                      {l.timestamp}
                    </td>
                    <td className="py-3 px-4">
                      <span className="px-2 py-0.5 rounded text-[10px] font-semibold bg-stone-100 text-stone-800">
                        {l.action}
                      </span>
                    </td>
                    <td className="py-3 px-4 text-stone-600 text-xs">
                      {l.entityType}
                    </td>
                    <td className="py-3 px-4 text-stone-900 font-medium">{l.details}</td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};
