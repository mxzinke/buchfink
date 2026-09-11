import React, { useCallback, useEffect, useState } from 'react';
import { FilePlus2, Sparkles, Trash2 } from 'lucide-react';
import { Api } from '../services/api';
import { formatBytes, formatDate } from '../utils/formatters';
import type { CompanyDocument, DocumentKindOption } from '../types';
import { DocumentAttachDialog } from '../components/DocumentAttachDialog';
import { useWriteLock } from '../components/WriteLock';
import {
  Button,
  ConfirmDialog,
  EmptyState,
  PageHeader,
  Section,
  SkeletonRows,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  cn,
  toast,
} from '../components/ui';

/**
 * Die Unterlagen des Unternehmens.
 *
 * Der Ort für Papiere, die weder Beleg noch Anlagendokument sind: der
 * Gesellschaftsvertrag, der Registerauszug, der Gewerbeschein — später der
 * Mietvertrag und der Versicherungsschein. Sie gehören zum Unternehmen und
 * gelten, solange es das Unternehmen gibt; ein Geschäftsjahr haben sie nicht,
 * und deshalb folgt die Liste auch nicht dem Jahr aus der Kopfzeile.
 *
 * Bewusst eine Liste und keine Verwaltung. Was hier fehlt — Suche, Ordner,
 * Versionen — fehlt, weil es bei zwei Dutzend Unterlagen nichts zu suchen gibt.
 * Was gebraucht wird, ist die Antwort auf „wo ist der Handelsregisterauszug",
 * und dafür genügen Art, Bezeichnung und Datum.
 */

export const DokumentePage: React.FC = () => {
  const writeLock = useWriteLock();
  const [documents, setDocuments] = useState<CompanyDocument[]>([]);
  const [kinds, setKinds] = useState<DocumentKindOption[]>([]);
  const [loading, setLoading] = useState(true);
  const [attaching, setAttaching] = useState(false);
  const [removing, setRemoving] = useState<CompanyDocument | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      setDocuments(await Api.getDocuments());
    } catch (e) {
      setDocuments([]);
      toast.error(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
    // Die Bezeichnungen der Arten. Fehlen sie, steht der Schlüssel da — das ist
    // hässlich und keine Störung.
    Api.getDocumentKinds()
      .then(setKinds)
      .catch(() => setKinds([]));
  }, [load]);

  const kindLabel = (value: string) => kinds.find((k) => k.value === value)?.label ?? value;

  async function open(doc: CompanyDocument) {
    try {
      const preview = await Api.getDocumentContent(doc.id);
      window.open(preview.dataUrl, '_blank');
    } catch (e) {
      // Die Prüfsumme stimmt nicht mehr, oder die Datei ist weg. Das Backend
      // sagt, was los ist; ein eigener Satz wäre die schlechtere Auskunft.
      toast.error(e instanceof Error ? e.message : String(e));
    }
  }

  async function remove() {
    if (!removing) return;
    try {
      await Api.removeDocument(removing.id);
      toast.success(`${removing.title || removing.fileName} ist aus den Unterlagen entfernt.`);
      setRemoving(null);
      await load();
    } catch (e) {
      toast.error(e instanceof Error ? e.message : String(e));
    }
  }

  return (
    <div className="max-w-[1200px] mx-auto px-8 py-8">
      <PageHeader helpSummary="Bewahren Sie hier Verträge, Bescheide und andere Unterlagen Ihres Unternehmens auf."
        title="Unterlagen"
        context={
          documents.length === 1 ? '1 Unterlage' : `${documents.length} Unterlagen des Unternehmens`
        }
        explain={
          <>
            Papiere, die zum Unternehmen gehören und nicht zu einer Buchung: Gesellschaftsvertrag,
            Registerauszug, Verträge, Bescheide. Jede Datei liegt unter ihrer Prüfsumme und wird
            erst herausgegeben, wenn diese noch stimmt.
          </>
        }
        action={
          <Button
            variant="primary"
            icon={<FilePlus2 className="w-4 h-4" strokeWidth={1.5} />}
            disabled={writeLock.locked}
            title={writeLock.hint}
            onClick={() => setAttaching(true)}
          >
            Unterlage ablegen
          </Button>
        }
      />

      <Section helpSummary="Bewahren Sie weiterhin benötigte Unterlagen auch nach Ablauf der Mindestfrist auf."
        title="Ablage"
        context="Das jüngste Dokument zuerst"
        className="mt-8"
        explain={
          <>
            Aufbewahrt wird zehn Jahre (§ 147 Abs. 1 Nr. 1 AO). Die Frist ist die gesetzliche
            Untergrenze und keine Aufforderung, danach zu löschen — der Gesellschaftsvertrag gilt,
            solange es die Gesellschaft gibt.
          </>
        }
      >
        {loading ? (
          <SkeletonRows rows={5} />
        ) : documents.length === 0 ? (
          <EmptyState
            title="Noch keine Unterlage abgelegt"
            description="Hier liegen die Papiere des Unternehmens: Gesellschaftsvertrag, Registerauszug, Verträge."
            action={
              <Button
                variant="primary"
                disabled={writeLock.locked}
                title={writeLock.hint}
                onClick={() => setAttaching(true)}
              >
                Unterlage ablegen
              </Button>
            }
          />
        ) : (
          <Table>
            <Thead>
              <Tr>
                <Th className="w-52">Art</Th>
                <Th>Bezeichnung</Th>
                <Th className="w-28">Datum</Th>
                <Th numeric className="w-24">
                  Größe
                </Th>
                <Th className="w-32">Aufbewahrung</Th>
                <Th className="w-32" aria-label="Aktion" />
              </Tr>
            </Thead>
            <Tbody>
              {documents.map((doc) => (
                <Tr key={doc.id}>
                  <Td className="text-ink-muted">{kindLabel(doc.kind)}</Td>
                  <Td className="max-w-[24rem]">
                    <button
                      type="button"
                      onClick={() => void open(doc)}
                      className="flex items-center gap-2 text-left truncate text-accent-text
                                 hover:text-accent transition-colors duration-120 ease-quiet"
                    >
                      <span className="truncate">{doc.title || doc.fileName}</span>
                      {/* Ein selbst erzeugtes Dokument lässt sich jederzeit neu
                          herstellen, ein hochgeladenes nicht. */}
                      {doc.generatedBy && (
                        <Sparkles
                          className="w-3.5 h-3.5 shrink-0 text-ink-faint"
                          strokeWidth={1.5}
                          aria-label="Von Buchfink erzeugt"
                        />
                      )}
                    </button>
                    {doc.note && <p className="text-caption text-ink-subtle truncate">{doc.note}</p>}
                  </Td>
                  <Td className={cn(doc.documentDate ? 'num text-ink-subtle' : 'text-ink-faint')}>
                    {doc.documentDate ? formatDate(doc.documentDate) : '—'}
                  </Td>
                  <Td numeric className="text-ink-subtle">
                    {formatBytes(doc.size)}
                  </Td>
                  <Td className="num text-ink-subtle">
                    {doc.retentionUntil ? `bis ${formatDate(doc.retentionUntil)}` : '—'}
                  </Td>
                  <Td className="text-right">
                    <Button
                      variant="quiet"
                      size="sm"
                      icon={<Trash2 className="w-4 h-4" strokeWidth={1.5} />}
                      iconOnly
                      aria-label={`${doc.title || doc.fileName} entfernen`}
                      title={writeLock.hint ?? 'Entfernen'}
                      disabled={writeLock.locked}
                      onClick={() => setRemoving(doc)}
                    />
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </Section>

      <DocumentAttachDialog
        open={attaching}
        dutyKey=""
        dutyTitle=""
        onOpenChange={setAttaching}
        onAttached={() => {
          setAttaching(false);
          void load();
        }}
      />

      <ConfirmDialog
        open={removing !== null}
        onOpenChange={(next) => !next && setRemoving(null)}
        title={`${removing?.title || removing?.fileName || 'Unterlage'} entfernen`}
        description={
          'Die Unterlage verschwindet aus der Ablage. Die Datei wird gelöscht, sobald kein ' +
          'anderes Dokument mehr auf sie zeigt. Der Vorgang steht im Änderungsprotokoll.'
        }
        confirmLabel="Entfernen"
        destructive
        onConfirm={() => void remove()}
      />
    </div>
  );
};

export default DokumentePage;
