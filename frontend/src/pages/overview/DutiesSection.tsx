import React from 'react';
import { ChevronRight } from 'lucide-react';
import type { Task } from '../../types';
import { formatCents, formatDate } from '../../utils/formatters';
import type { DutyRow, OverviewPlan } from '../../utils/overview';
import { targetLabel } from '../../utils/findings';
import { toRoute, type NavigateFn } from '../../navigation';
import {
  Button,
  EmptyState,
  Help,
  Notice,
  Section,
  SkeletonRows,
  StatusBadge,
  Table,
  Tbody,
  Td,
  Th,
  Thead,
  Tr,
  cn,
} from '../../components/ui';

/** Anzahl, Betrag und Fälligkeit einer Aufgabe, soweit sie etwas aussagen. */
function taskContext(task: Task): string {
  const parts: string[] = [];
  if (task.count > 0) parts.push(task.count === 1 ? '1 Vorgang' : `${task.count} Vorgänge`);
  if (task.amount !== 0) parts.push(formatCents(task.amount));
  if (task.dueDate) parts.push(`fällig ${formatDate(task.dueDate)}`);
  return parts.join(' · ');
}

const MARK: Record<Task['group'], string> = {
  overdue: 'bg-negative',
  open: 'bg-attention',
  upcoming: 'bg-ink-faint',
};

function dutiesContext(duties: DutyRow[]): string {
  if (duties.length === 0) return 'Keine Frist in den nächsten 30 Tagen';
  const overdue = duties.filter((d) => d.group === 'overdue').length;
  const count = duties.length === 1 ? '1 Pflicht' : `${duties.length} Pflichten`;
  return overdue > 0 ? `${count} · ${overdue} überfällig` : count;
}

export interface DutiesSectionProps {
  plan: OverviewPlan | null;
  /** Aufgaben, die die Übersicht selbst ableitet, etwa fehlende Stammdaten. */
  extra?: Task[];
  /** Pflichten oder Aufgaben ließen sich nicht laden. */
  error?: string;
  onOpen: (key: string) => void;
  onNavigate: NavigateFn;
}

/**
 * Was jetzt zu tun ist: die Pflichten, die nur der Mensch abgibt, festschreibt
 * oder bestätigt, und darunter weitere Aufgaben, die nur er erledigt, etwa
 * Stammdaten oder Sicherung. Was Claude in den Büchern erledigt, steht hier
 * nicht. Eine Pflicht öffnet ihre Einzelheiten in der Seitenleiste.
 */
export const DutiesSection: React.FC<DutiesSectionProps> = ({ plan, extra = [], error, onOpen, onNavigate }) => {
  const duties = plan?.duties ?? [];
  const work = plan ? [...extra, ...plan.human] : [];

  return (
    <Section
      title="Was jetzt zu tun ist"
      context={plan && !error ? dutiesContext(duties) : 'Fristen und Bestätigungen'}
      divider={false}
      helpSummary="Hier stehen die Fristen, die nur Sie erledigen können."
      explain={
        <>
          Buchfink rechnet die Fristen aus Ihren Stammdaten und der Buchführung. Claude bereitet
          Voranmeldungen und Erklärungen vor. Abgeben, festschreiben und bestätigen können nur Sie,
          hier in der App. Was erledigt ist, verschwindet von selbst aus der Liste.
        </>
      }
    >
      {error ? (
        <Notice tone="negative" text={error} />
      ) : !plan ? (
        <SkeletonRows rows={4} />
      ) : duties.length === 0 ? (
        <EmptyState
          title="Nichts fällig"
          description={
            plan.next
              ? `Die nächste Frist: ${plan.next.title}, fällig am ${formatDate(plan.next.dueDate)}.`
              : 'Der Pflichtenkalender enthält keine offene Pflicht.'
          }
        />
      ) : (
        <Table>
          <Thead>
            <Tr>
              <Th>Pflicht</Th>
              <Th className="w-32">Fällig</Th>
              <Th className="w-48">Stand</Th>
              <Th className="w-10" aria-label="Einzelheiten" />
            </Tr>
          </Thead>
          <Tbody>
            {duties.map((duty) => (
              <Tr key={duty.key} className="cursor-pointer" onClick={() => onOpen(duty.key)}>
                <Td className="max-w-[28rem]">
                  <button
                    type="button"
                    className="block max-w-full truncate text-left text-ink underline-offset-2 hover:underline"
                    onClick={(event) => {
                      event.stopPropagation();
                      onOpen(duty.key);
                    }}
                  >
                    {duty.title}
                  </button>
                </Td>
                <Td className="num text-ink-muted">
                  {duty.dueDate ? formatDate(duty.dueDate) : '—'}
                  {duty.obligation?.statutoryDate && duty.obligation.statutoryDate !== duty.dueDate && (
                    <span className="block text-caption text-ink-subtle">
                      {`verschoben vom ${formatDate(duty.obligation.statutoryDate).slice(0, 6)}`}
                    </span>
                  )}
                </Td>
                <Td>
                  <span className="inline-flex items-center gap-2">
                    <StatusBadge status={duty.group === 'overdue' ? 'ueberfaellig' : 'offen'} />
                    {duty.obligation?.preparation && (
                      <span className="text-caption text-ink-subtle">vorbereitet</span>
                    )}
                  </span>
                </Td>
                <Td className="pl-0 text-right">
                  <ChevronRight
                    className="inline w-3.5 h-3.5 text-ink-faint"
                    strokeWidth={1.5}
                    aria-hidden="true"
                  />
                </Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      )}

      {!error && work.length > 0 && (
        <div className="mt-8">
          <h3 className="text-label text-ink-muted">Weitere Aufgaben</h3>
          <ul className="mt-2 flex flex-col">
            {work.map((task) => {
              const route = toRoute(task.target.area, task.target.tab, task.target.id);
              return (
                <li
                  key={task.key}
                  className="flex items-center gap-3 py-2 border-t border-line first:border-t-0"
                >
                  <span className={cn('mark-diamond shrink-0', MARK[task.group])} aria-hidden="true" />
                  <span className="flex min-w-0 flex-1 items-center">
                    <span className="truncate text-body text-ink">{task.title}</span>
                    <Help summary={task.why} label={`Erklärung zu ${task.title}`}>
                      {task.reference ? `Grundlage: ${task.reference}` : undefined}
                    </Help>
                  </span>
                  <span className="text-caption text-ink-subtle num">{taskContext(task)}</span>
                  {route && (
                    <Button variant="quiet" size="sm" onClick={() => onNavigate(route)}>
                      {targetLabel(route)}
                    </Button>
                  )}
                </li>
              );
            })}
          </ul>
        </div>
      )}
    </Section>
  );
};
